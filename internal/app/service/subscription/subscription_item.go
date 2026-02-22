package subscription

import (
	"context"
	"fmt"
	models "github.com/fatflowers/cashier/internal/models"
	"github.com/fatflowers/cashier/pkg/config"
	types "github.com/fatflowers/cashier/pkg/types"
	"slices"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Service struct {
	cfg *config.Config
	db  *gorm.DB
	log *zap.SugaredLogger
}

func NewService(cfg *config.Config, db *gorm.DB, log *zap.SugaredLogger) *Service {
	return &Service{cfg: cfg, db: db, log: log}
}

// UserSubscriptionItem wraps a Transaction with computed subscription-period data.
type UserSubscriptionItem struct {
	models.Transaction
	// RemainingDurationSeconds is the remaining active duration in seconds.
	// It is updated when refund-related adjustments occur.
	RemainingDurationSeconds int64 `json:"remaining_duration_seconds"`
	// ActivatedAt is the effective start time.
	ActivatedAt time.Time `json:"activated_at"`
	// ExpireAt is the expiration time.
	ExpireAt time.Time `json:"expire_at"`
}

func (s *Service) compareUserMembershipItemByPurchaseAt(a, b *models.Transaction) int {
	return a.PurchaseAt.Compare(b.PurchaseAt)
}

func (s *Service) processNonRenewableSubscription(result []*UserSubscriptionItem, paymentItem *types.PaymentItem, item *UserSubscriptionItem, queryAt time.Time) ([]*UserSubscriptionItem, error) {
	item.ActivatedAt = item.PurchaseAt
	if paymentItem.DurationHour != nil {
		item.RemainingDurationSeconds = int64(*paymentItem.DurationHour * 60 * 60)
	} else {
		return nil, fmt.Errorf("duration is nil for non renewable subscription")
	}
	item.ExpireAt = item.ActivatedAt.Add(time.Duration(item.RemainingDurationSeconds) * time.Second)

	if len(result) > 0 {
		// For non-renewable subscriptions, append to the tail.
		// If purchase time is earlier than the last item's expiration,
		// move activation to that expiration and recalculate expiration from remaining duration.
		if item.PurchaseAt.Before(result[len(result)-1].ExpireAt) {
			item.ActivatedAt = result[len(result)-1].ExpireAt
			item.ExpireAt = item.ActivatedAt.Add(time.Duration(item.RemainingDurationSeconds) * time.Second)
		}
	}

	// Skip refunded items when the computed expiration is still after queryAt.
	if item.RefundAt != nil && item.ExpireAt.After(queryAt) {
		return result, nil
	}

	return append(result, item), nil
}

func (s *Service) processAutoRenewableSubscription(result []*UserSubscriptionItem, item *UserSubscriptionItem, queryAt time.Time) ([]*UserSubscriptionItem, error) {
	// Skip refunded items when the computed expiration is still after queryAt.
	if item.RefundAt != nil && item.AutoRenewExpireAt.After(queryAt) {
		return result, nil
	}

	item.ActivatedAt = item.PurchaseAt
	if item.AutoRenewExpireAt != nil {
		item.ExpireAt = *item.AutoRenewExpireAt
	} else {
		return nil, fmt.Errorf("auto renew expire at is nil for auto renewable subscription")
	}

	item.RemainingDurationSeconds = int64(item.ExpireAt.Sub(item.PurchaseAt).Seconds())

	if len(result) == 0 {
		result = append(result, item)
	} else {
		// Auto-renewable subscriptions should take precedence.
		// Find whether an existing period overlaps with this purchase time.
		insertIndex := -1
		for index := range result {
			if result[index].ExpireAt.After(item.PurchaseAt) {
				insertIndex = index
				break
			}
		}

		if insertIndex == -1 {
			result = append(result, item)
		} else {
			// Overlap exists with existing subscription periods.
			for index := insertIndex; index < len(result); index += 1 {
				if index == insertIndex {
					// Split the existing period and insert current auto-renew item before it.
					// Recalculate remaining duration for the overlapped existing item.
					remainingDuration := result[index].ExpireAt.Sub(item.PurchaseAt)
					result[index].RemainingDurationSeconds = int64(remainingDuration.Seconds())
					result[index].ActivatedAt = item.ExpireAt
					result[index].ExpireAt = result[index].ActivatedAt.Add(remainingDuration)
				} else {
					result[index].ActivatedAt = result[index-1].ExpireAt
					result[index].ExpireAt = result[index].ActivatedAt.Add(time.Duration(result[index].RemainingDurationSeconds) * time.Second)
				}
			}

			result = slices.Insert(result, insertIndex, item)
		}
	}

	return result, nil
}

// selectLastActivePeriods filters and returns the last contiguous active periods.
func (s *Service) selectLastActivePeriods(items []*UserSubscriptionItem) ([]*UserSubscriptionItem, error) {
	if len(items) == 0 {
		return items, nil
	}

	selectedPeriodStartIndex := 0
	for index := 1; index < len(items); index++ {
		if items[index].ActivatedAt != items[index-1].ExpireAt {
			selectedPeriodStartIndex = index
		}
	}

	result := items[selectedPeriodStartIndex:]

	if len(result) == 0 {
		return result, nil
	}

	lastIndex := len(result) - 1
	for lastIndex >= 0 {
		if result[lastIndex].RefundAt != nil {
			lastIndex -= 1
		} else {
			break
		}
	}

	return result[:lastIndex+1], nil
}

// getAllActiveUserSubscriptionItems returns all active subscription items at queryAt.
// pgItems: subscription items loaded from database.
// queryAt: point-in-time used for evaluation.
func (s *Service) getAllActiveUserSubscriptionItems(ctx context.Context, pgItems []*models.Transaction, queryAt time.Time) ([]*UserSubscriptionItem, error) {
	if queryAt.IsZero() {
		return nil, fmt.Errorf("invalid queryAt: zero value")
	}

	if len(pgItems) == 0 {
		return nil, nil
	}

	slices.SortStableFunc(pgItems, s.compareUserMembershipItemByPurchaseAt)

	var result []*UserSubscriptionItem

	for _, pgItem := range pgItems {
		// Stop when purchase time is after queryAt.
		if pgItem.PurchaseAt.After(queryAt) {
			break
		}

		item := &UserSubscriptionItem{
			Transaction: *pgItem,
		}

		paymentItem := pgItem.GetPaymentItemSnapshot()
		if paymentItem == nil {
			paymentItem = s.cfg.GetPaymentItemByID(pgItem.PaymentItemID)
			if paymentItem == nil {
				return nil, fmt.Errorf("failed to get payment item by id: %s", pgItem.PaymentItemID)
			}
		}

		var err error
		// Branch early by payment item type.
		switch paymentItem.Type {
		case types.PaymentItemTypeNonRenewableSubscription:
			result, err = s.processNonRenewableSubscription(result, paymentItem, item, queryAt)
		case types.PaymentItemTypeAutoRenewableSubscription:
			result, err = s.processAutoRenewableSubscription(result, item, queryAt)
		default:
			return nil, fmt.Errorf("unsupported payment item type: %s", paymentItem.Type)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to process subscription: %w", err)
		}
	}

	return s.selectLastActivePeriods(result)
}
