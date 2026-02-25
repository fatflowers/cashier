package subscription

import (
	"context"
	"fmt"
	models "github.com/fatflowers/cashier/internal/models"
	"github.com/fatflowers/cashier/pkg/logctx"
	"github.com/fatflowers/cashier/pkg/tool"
	types "github.com/fatflowers/cashier/pkg/types"
	"time"

	"errors"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// getChangeReason determines the subscription change reason from a transaction
func (s *Service) getChangeReason(ctx context.Context, item *models.Transaction) (types.SubscriptionChangeReason, error) {
	if item.RefundAt != nil {
		return types.UserSubscriptionChangeReasonRefund, nil
	}
	paymentItem := item.GetPaymentItemSnapshot()
	if paymentItem == nil {
		paymentItem = s.cfg.GetPaymentItemByID(item.PaymentItemID)
		if paymentItem == nil {
			return types.UserSubscriptionChangeReasonPurchase, fmt.Errorf("payment item not found: %s", item.PaymentItemID)
		}
	}

	// TODO: handle upgrade and downgrade scenarios.

	if paymentItem.Renewable() && !item.IsAutoRenewable() {
		return types.UserSubscriptionChangeReasonCancelRenew, nil
	}

	return types.UserSubscriptionChangeReasonPurchase, nil
}

// GetUserActiveSubscriptionItems returns all active subscription items at queryAt for a user.
func (s *Service) GetUserActiveSubscriptionItems(ctx context.Context, userID string, queryAt time.Time) ([]*UserSubscriptionItem, error) {
	items, err := s.GetAllUserTransactions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}
	return s.getAllActiveUserSubscriptionItems(ctx, items, queryAt)
}

// UpsertUserSubscriptionByItem updates user subscription state based on a transaction.
func (s *Service) UpsertUserSubscriptionByItem(ctx context.Context, item *models.Transaction) error {
	var subscriptionUpdated bool
	var subscription *models.Subscription
	var reason types.SubscriptionChangeReason

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error

		reason, err = s.getChangeReason(ctx, item)
		if err != nil {
			return fmt.Errorf("failed to get change reason: %w", err)
		}

		if err = s.upsertTransaction(ctx, tx, item, reason); err != nil {
			return fmt.Errorf("failed to upsert transaction: %w", err)
		}

		pgItems, err := s.getAllUserTransactionsWithTx(ctx, tx, item.UserID)
		if err != nil {
			return fmt.Errorf("failed to get user transactions: %w", err)
		}

		processTime := time.Now()
		if item.PurchaseAt.After(processTime) {
			processTime = item.PurchaseAt
		}

		items, err := s.getAllActiveUserSubscriptionItems(ctx, pgItems, processTime)
		if err != nil {
			return fmt.Errorf("failed to get active subscription items: %w", err)
		}

		logctx.FromCtx(ctx, s.log).Infof("upsert user subscription by item, user_id=%s, item_id=%s, reason=%s", item.UserID, item.ID, reason)

		if err := s.rebuildUserMembershipActiveItems(ctx, tx, item.UserID, items); err != nil {
			return fmt.Errorf("failed to rebuild user membership active items: %w", err)
		}

		if len(items) == 0 {
			subscription = &models.Subscription{
				UserID: item.UserID,
				Status: types.SubscriptionStatusInactive,
			}

			// business hook can be invoked after Tx commit
			if err := s.cancelMembership(ctx, tx, item.UserID, reason); err != nil {
				return err
			}
		} else {
			// set active subscription with last expireAt
			lastExpire := items[len(items)-1].ExpireAt
			subscription = &models.Subscription{
				UserID:   item.UserID,
				Status:   types.SubscriptionStatusActive,
				ExpireAt: &lastExpire,
			}
			for i := len(items) - 1; i >= 0; i-- {
				if items[i].NextAutoRenewAt != nil {
					if items[i].NextAutoRenewAt.After(processTime) {
						subscription.NextAutoRenewAt = items[i].NextAutoRenewAt
					}
					break
				}
			}

			updated, err := s.upsertSubscription(ctx, tx, subscription, reason)
			if err != nil {
				return err
			}
			subscriptionUpdated = updated
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to UpsertUserMembershipByItem: %w", err)
	}

	if subscriptionUpdated {
		go s.handleMembershipChange(ctx, subscription, reason, item)
	}

	return nil
}

// Data access helpers.
func (s *Service) rebuildUserMembershipActiveItems(ctx context.Context, tx *gorm.DB, userID string, items []*UserSubscriptionItem) error {
	if err := tx.WithContext(ctx).Where("user_id = ?", userID).Delete(&models.UserMembershipActiveItem{}).Error; err != nil {
		return fmt.Errorf("failed to delete user membership active items: %w", err)
	}

	if len(items) == 0 {
		return nil
	}

	activeItems := make([]*models.UserMembershipActiveItem, 0, len(items))
	for _, item := range items {
		activeItem := item.ToUserMembershipActiveItem()
		if activeItem != nil {
			activeItems = append(activeItems, activeItem)
		}
	}

	if len(activeItems) == 0 {
		return nil
	}

	if err := tx.WithContext(ctx).Create(activeItems).Error; err != nil {
		return fmt.Errorf("failed to create user membership active items: %w", err)
	}

	return nil
}

func (s *Service) upsertTransaction(ctx context.Context, tx *gorm.DB, item *models.Transaction, changeReason types.SubscriptionChangeReason) error {
	var original models.Transaction
	err := tx.WithContext(ctx).
		Where("provider_id = ? AND transaction_id = ?", item.ProviderID, item.TransactionID).
		First(&original).Error

	created := false

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to load original transaction: %w", err)
	}

	if err == nil && original.ID != "" {
		// Preserve ID and important extra fields
		item.ID = original.ID
		// Safely copy extra content
		extra := item.Extra.Data()
		if extra == nil {
			extra = &models.UserSubscriptionItemExtra{}
		}
		if origExtra := original.Extra.Data(); origExtra != nil {
			extra.IsFirstPurchase = origExtra.IsFirstPurchase
			extra.PaymentItemSnapshot = original.GetPaymentItemSnapshot()
		}
		item.Extra = datatypes.NewJSONType(extra)
		item.CreatedAt = original.CreatedAt
	} else {
		// New transaction
		created = true
		if item.ID == "" {
			item.ID = tool.GenerateUUIDV7()
		}
		// Determine first purchase
		var count int64
		if err := tx.WithContext(ctx).Model(&models.Transaction{}).
			Where("user_id = ?", item.UserID).Count(&count).Error; err != nil {
			return fmt.Errorf("failed to check first purchase: %w", err)
		}
		extra := item.Extra.Data()
		if extra == nil {
			extra = &models.UserSubscriptionItemExtra{}
		}
		extra.IsFirstPurchase = count == 0
		item.Extra = datatypes.NewJSONType(extra)
	}

	if err := tx.WithContext(ctx).Save(item).Error; err != nil {
		return fmt.Errorf("failed to upsert transaction: %w", err)
	}

	// Write change log asynchronously; errors are logged but not returned
	go func(before *models.Transaction, after *models.Transaction, reason types.SubscriptionChangeReason) {
		log := &models.TransactionLog{
			ID:            tool.GenerateUUIDV7(),
			UserID:        after.UserID,
			PaymentItemID: after.PaymentItemID,
			ProviderID:    after.ProviderID,
			TransactionID: after.TransactionID,
			Reason:        reason,
			Before:        datatypes.NewJSONType(before),
			After:         datatypes.NewJSONType(after),
			Extra:         datatypes.JSONMap{},
		}
		if err := s.db.Save(log).Error; err != nil {
			logctx.FromCtx(ctx, s.log).Errorf("failed to save transaction log: %v", err)
		}
	}(func() *models.Transaction {
		if original.ID == "" {
			return nil
		}
		return &original
	}(), item, changeReason)

	if created && changeReason == types.UserSubscriptionChangeReasonRefund {
		logctx.FromCtx(ctx, s.log).Errorf("created refunded transaction not found previously: provider=%s txid=%s user=%s", item.ProviderID, item.TransactionID, item.UserID)
	}
	return nil
}

func (s *Service) GetAllUserTransactions(ctx context.Context, userID string) ([]*models.Transaction, error) {
	var items []*models.Transaction
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("purchase_at desc").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Service) getAllUserTransactionsWithTx(ctx context.Context, tx *gorm.DB, userID string) ([]*models.Transaction, error) {
	var items []*models.Transaction
	if err := tx.WithContext(ctx).Where("user_id = ?", userID).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Service) upsertSubscription(ctx context.Context, tx *gorm.DB, m *models.Subscription, reason types.SubscriptionChangeReason) (bool, error) {
	// Load existing subscription by user_id
	var original models.Subscription
	if err := tx.WithContext(ctx).Where("user_id = ?", m.UserID).First(&original).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return false, fmt.Errorf("failed to get original subscription: %w", err)
		}
	}

	if original.ID != "" {
		m.ID = original.ID
		m.CreatedAt = original.CreatedAt
	} else {
		if m.ID == "" {
			m.ID = tool.GenerateUUIDV7()
		}
	}

	before := func() *models.Subscription {
		if original.ID == "" {
			return nil
		}
		// make a copy value to snapshot
		cp := original
		return &cp
	}()

	if err := tx.WithContext(ctx).Save(m).Error; err != nil {
		return false, fmt.Errorf("failed to upsert subscription: %w", err)
	}

	updated := (before != nil && before.Valid() != m.Valid()) || (before == nil && m.Valid())

	// async log
	go func(b *models.Subscription, a *models.Subscription) {
		log := &models.SubscriptionLog{
			ID:     tool.GenerateUUIDV7(),
			UserID: a.UserID,
			Reason: reason,
			Before: datatypes.NewJSONType(b),
			After:  datatypes.NewJSONType(a),
			Extra:  datatypes.JSONMap{},
		}
		if err := s.db.Save(log).Error; err != nil {
			logctx.FromCtx(ctx, s.log).Errorf("failed to save subscription log: %v", err)
		}
	}(before, m)

	return updated, nil
}

func (s *Service) cancelMembership(ctx context.Context, tx *gorm.DB, userID string, reason types.SubscriptionChangeReason) error {
	var subscription models.Subscription
	if err := tx.WithContext(ctx).Where("user_id = ?", userID).First(&subscription).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	subscription.Status = types.SubscriptionStatusInactive
	subscription.NextAutoRenewAt = nil
	subscription.ExpireAt = nil

	if _, err := s.upsertSubscription(ctx, tx, &subscription, reason); err != nil {
		return fmt.Errorf("failed to upsert subscription: %w", err)
	}
	return nil
}

// handleMembershipChange is a placeholder for post-subscription-state-change actions.
func (s *Service) handleMembershipChange(ctx context.Context, subscription *models.Subscription, reason types.SubscriptionChangeReason, item *models.Transaction) {
	// no-op: integrate notification hooks here if needed
}

// SendFreeGift grants an internal gift (for example, a free membership card).
func (s *Service) SendFreeGift(ctx context.Context, userID, paymentItemID, operatorID string) error {
	if userID == "" || paymentItemID == "" {
		return fmt.Errorf("invalid params: userID and paymentItemID required")
	}
	paymentItem := s.cfg.GetPaymentItemByID(paymentItemID)
	if paymentItem == nil {
		return fmt.Errorf("payment item not found: %s", paymentItemID)
	}

	txn := &models.Transaction{
		UserID:        userID,
		ProviderID:    types.PaymentProviderInner,
		PaymentItemID: paymentItemID,
		TransactionID: tool.GenerateUUIDV7(),
		PurchaseAt:    time.Now(),
		Extra: datatypes.NewJSONType(&models.UserSubscriptionItemExtra{
			PaymentItemSnapshot: paymentItem,
			OperatorId:          operatorID,
		}),
	}

	return s.UpsertUserSubscriptionByItem(ctx, txn)
}
