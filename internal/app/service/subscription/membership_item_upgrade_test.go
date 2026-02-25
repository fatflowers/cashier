package subscription

import (
	"context"
	"testing"
	"time"

	models "github.com/fatflowers/cashier/internal/models"
	"github.com/fatflowers/cashier/pkg/config"
	types "github.com/fatflowers/cashier/pkg/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGetAllActiveUserSubscriptionItems_SkipsBeforeUpgradedTransaction(t *testing.T) {
	now := time.Unix(1735689600, 0) // 2025-01-01 UTC
	cfg := &config.Config{PaymentItems: []*types.PaymentItem{
		{ID: "payment1", Type: types.PaymentItemTypeAutoRenewableSubscription},
	}}
	svc := NewService(cfg, nil, zap.NewNop().Sugar())

	beforeID := "tx-old"
	txs := []*models.Transaction{
		{
			ID:                "old",
			UserID:            "u1",
			ProviderID:        types.PaymentProviderApple,
			PaymentItemID:     "payment1",
			TransactionID:     "tx-old",
			PurchaseAt:        now,
			AutoRenewExpireAt: ptrTime(now.Add(30 * 24 * time.Hour)),
		},
		{
			ID:                          "new",
			UserID:                      "u1",
			ProviderID:                  types.PaymentProviderApple,
			PaymentItemID:               "payment1",
			TransactionID:               "tx-new",
			PurchaseAt:                  now.Add(10 * 24 * time.Hour),
			AutoRenewExpireAt:           ptrTime(now.Add(40 * 24 * time.Hour)),
			BeforeUpgradedTransactionID: &beforeID,
		},
	}

	items, err := svc.getAllActiveUserSubscriptionItems(context.Background(), txs, now.Add(20*24*time.Hour))
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "new", items[0].ID)
}

func TestGetAllActiveUserSubscriptionItems_UpgradeDoesNotAffectHistoricalQuery(t *testing.T) {
	now := time.Unix(1735689600, 0) // 2025-01-01 UTC
	cfg := &config.Config{PaymentItems: []*types.PaymentItem{
		{ID: "payment1", Type: types.PaymentItemTypeAutoRenewableSubscription},
	}}
	svc := NewService(cfg, nil, zap.NewNop().Sugar())

	beforeID := "tx-old"
	txs := []*models.Transaction{
		{
			ID:                "old",
			UserID:            "u1",
			ProviderID:        types.PaymentProviderApple,
			PaymentItemID:     "payment1",
			TransactionID:     "tx-old",
			PurchaseAt:        now,
			AutoRenewExpireAt: ptrTime(now.Add(30 * 24 * time.Hour)),
		},
		{
			ID:                          "new",
			UserID:                      "u1",
			ProviderID:                  types.PaymentProviderApple,
			PaymentItemID:               "payment1",
			TransactionID:               "tx-new",
			PurchaseAt:                  now.Add(10 * 24 * time.Hour),
			AutoRenewExpireAt:           ptrTime(now.Add(40 * 24 * time.Hour)),
			BeforeUpgradedTransactionID: &beforeID,
		},
	}

	items, err := svc.getAllActiveUserSubscriptionItems(context.Background(), txs, now.Add(5*24*time.Hour))
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "old", items[0].ID)
}

func ptrTime(v time.Time) *time.Time { return &v }
