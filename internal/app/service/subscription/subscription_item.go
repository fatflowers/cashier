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

// Subscription对应的Transaction的包装
type UserSubscriptionItem struct {
	models.Transaction
	// RemainingDurationSeconds 剩余有效期时长，单位：秒
	// 退款导致的数据变动，会更新这个值
	RemainingDurationSeconds int64 `json:"remaining_duration_seconds"`
	// ActivatedAt 开始生效时间
	ActivatedAt time.Time `json:"activated_at"`
	// ExpireTime 过期时间
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
		// 非自动续费订阅，追加到列表尾部，如果购买时间小于最后一个元素的过期时间，
		// 则将生效时间设置为最后一个元素的过期时间，过期时间设置为购买时间加上剩余时长
		if item.PurchaseAt.Before(result[len(result)-1].ExpireAt) {
			item.ActivatedAt = result[len(result)-1].ExpireAt
			item.ExpireAt = item.ActivatedAt.Add(time.Duration(item.RemainingDurationSeconds) * time.Second)
		}
	}

	// 如果退款时间不为空，并且过期时间大于查询时间，则跳过不处理
	if item.RefundAt != nil && item.ExpireAt.After(queryAt) {
		return result, nil
	}

	return append(result, item), nil
}

func (s *Service) processAutoRenewableSubscription(result []*UserSubscriptionItem, item *UserSubscriptionItem, queryAt time.Time) ([]*UserSubscriptionItem, error) {
	// 如果退款时间不为空，并且过期时间大于查询时间，则跳过不处理
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
		// 自动续费订阅，需要优先使用
		// 当前已有订阅中，是否存在与当前自动续费购买时间有重叠的订阅
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
			// 如果与已有的订阅有重叠
			for index := insertIndex; index < len(result); index += 1 {
				if index == insertIndex {
					// 中断已有订阅，并将当前自动订阅插入到已有订阅的前面
					// 重新计算有重叠的存量项目的剩余有效期时长（RemainingDurationSeconds），为当前购买时间与已有订阅的过期时间差
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

// selectLastActivePeriods 过滤并返回活跃的订阅周期
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

// getAllActiveUserSubscriptionItems 获取指定时间点的所有有效会员订阅项
// pgItems: 数据库中的会员订阅项
// queryAt: 查询时间点
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
		// 如果购买时间大于查询时间
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
		// 提前判断支付项类型
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
