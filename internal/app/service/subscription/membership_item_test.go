package subscription

import (
	"context"
	"testing"
	"time"

	models "github.com/fatflowers/cashier/internal/models"
	"github.com/fatflowers/cashier/pkg/config"
	types "github.com/fatflowers/cashier/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockPaymentItem is a payment item test fixture.
type mockPaymentItem struct {
	id           string
	itemType     types.PaymentItemType
	durationHour *int64
}

func (m *mockPaymentItem) toType() *types.PaymentItem {
	return &types.PaymentItem{ID: m.id, Type: m.itemType, DurationHour: m.durationHour}
}

func TestGetAllActiveUserMembershipItems_AllCases(t *testing.T) {
	now := time.Now()
	oneMonth := 30 * 24 * time.Hour
	twoMonths := 60 * 24 * time.Hour
	threeMonths := 90 * 24 * time.Hour
	oneHundredDays := 100 * 24 * time.Hour
	// hours payloads for PaymentItem.DurationHour
	oneMonthHours := int64(30 * 24)

	type wantItem struct {
		id   string
		act  time.Time
		exp  time.Time
		remS int64
	}

	tests := []struct {
		name        string
		txs         []*models.Transaction
		queryAt     time.Time
		paymentStub []*mockPaymentItem
		wantLen     int
		wantErr     bool
		want        []wantItem
	}{
		{name: "empty input", txs: nil, queryAt: now, wantLen: 0},
		{name: "invalid query time", txs: []*models.Transaction{}, queryAt: time.Time{}, wantErr: true},
		{
			name:        "single non-renewable subscription",
			txs:         []*models.Transaction{{ID: "1", PaymentItemID: "payment1", PurchaseAt: now}},
			queryAt:     now.Add(15 * 24 * time.Hour),
			paymentStub: []*mockPaymentItem{{id: "payment1", itemType: types.PaymentItemTypeNonRenewableSubscription, durationHour: &oneMonthHours}},
			wantLen:     1,
			want:        []wantItem{{id: "1", act: now, exp: now.Add(oneMonth), remS: int64(oneMonth.Seconds())}},
		},
		{
			name:        "single auto-renewable subscription",
			txs:         []*models.Transaction{{ID: "1", PaymentItemID: "payment1", PurchaseAt: now, AutoRenewExpireAt: &[]time.Time{now.Add(oneMonth)}[0]}},
			queryAt:     now.Add(15 * 24 * time.Hour),
			paymentStub: []*mockPaymentItem{{id: "payment1", itemType: types.PaymentItemTypeAutoRenewableSubscription}},
			wantLen:     1,
			want:        []wantItem{{id: "1", act: now, exp: now.Add(oneMonth), remS: int64(oneMonth.Seconds())}},
		},
		{
			name: "overlapping subscriptions",
			txs: []*models.Transaction{
				{ID: "1", PaymentItemID: "payment1", PurchaseAt: now},
				{ID: "2", PaymentItemID: "payment2", PurchaseAt: now.Add(15 * 24 * time.Hour), AutoRenewExpireAt: &[]time.Time{now.Add(45 * 24 * time.Hour)}[0]},
			},
			queryAt:     now.Add(20 * 24 * time.Hour),
			paymentStub: []*mockPaymentItem{{id: "payment1", itemType: types.PaymentItemTypeNonRenewableSubscription, durationHour: &oneMonthHours}, {id: "payment2", itemType: types.PaymentItemTypeAutoRenewableSubscription}},
			wantLen:     2,
			want: []wantItem{
				{id: "2", act: now.Add(15 * 24 * time.Hour), exp: now.Add(45 * 24 * time.Hour), remS: int64((30 * 24 * time.Hour).Seconds())},
				{id: "1", act: now.Add(45 * 24 * time.Hour), exp: now.Add(60 * 24 * time.Hour), remS: int64((15 * 24 * time.Hour).Seconds())},
			},
		},
		{
			name:        "refunded subscription",
			txs:         []*models.Transaction{{ID: "1", PaymentItemID: "payment1", PurchaseAt: now, RefundAt: &[]time.Time{now.Add(5 * 24 * time.Hour)}[0]}},
			queryAt:     now.Add(15 * 24 * time.Hour),
			paymentStub: []*mockPaymentItem{{id: "payment1", itemType: types.PaymentItemTypeNonRenewableSubscription, durationHour: &oneMonthHours}},
			wantLen:     0,
		},
		{
			name: "refunded subscription 2",
			txs: []*models.Transaction{
				{ID: "1", PaymentItemID: "payment1", PurchaseAt: now, RefundAt: &[]time.Time{now.Add(32 * 24 * time.Hour)}[0]},
				{ID: "2", PaymentItemID: "payment1", PurchaseAt: now.Add(4 * time.Hour)},
			},
			queryAt:     now.Add(35 * 24 * time.Hour),
			paymentStub: []*mockPaymentItem{{id: "payment1", itemType: types.PaymentItemTypeNonRenewableSubscription, durationHour: &oneMonthHours}},
			wantLen:     2,
			want: []wantItem{
				{id: "1", act: now, exp: now.Add(oneMonth), remS: int64(oneMonth.Seconds())},
				{id: "2", act: now.Add(oneMonth), exp: now.Add(twoMonths), remS: int64(oneMonth.Seconds())},
			},
		},
		{
			name: "refunded subscription 3",
			txs: []*models.Transaction{
				{ID: "1", PaymentItemID: "payment1", PurchaseAt: now},
				{ID: "2", PaymentItemID: "payment1", PurchaseAt: now.Add(4 * time.Hour), RefundAt: &[]time.Time{now.Add(5 * time.Hour)}[0]},
			},
			queryAt:     now.Add(35 * 24 * time.Hour),
			paymentStub: []*mockPaymentItem{{id: "payment1", itemType: types.PaymentItemTypeNonRenewableSubscription, durationHour: &oneMonthHours}},
			wantLen:     1,
			want:        []wantItem{{id: "1", act: now, exp: now.Add(oneMonth), remS: int64(oneMonth.Seconds())}},
		},
		{
			name:        "refunded subscription 4",
			txs:         []*models.Transaction{{ID: "1", PaymentItemID: "payment1", PurchaseAt: now, AutoRenewExpireAt: &[]time.Time{now.Add(oneMonth)}[0], RefundAt: &[]time.Time{now.Add(5 * time.Hour)}[0]}},
			queryAt:     now.Add(7 * time.Hour),
			paymentStub: []*mockPaymentItem{{id: "payment1", itemType: types.PaymentItemTypeAutoRenewableSubscription}},
			wantLen:     0,
		},
		{
			name:        "refunded subscription 5",
			txs:         []*models.Transaction{{ID: "1", PaymentItemID: "payment1", PurchaseAt: now, AutoRenewExpireAt: &[]time.Time{now.Add(oneMonth)}[0], RefundAt: &[]time.Time{now.Add(5 * time.Hour)}[0]}},
			queryAt:     now.Add(twoMonths),
			paymentStub: []*mockPaymentItem{{id: "payment1", itemType: types.PaymentItemTypeAutoRenewableSubscription}},
			wantLen:     0,
		},
		{
			name: "refunded subscription 6",
			txs: []*models.Transaction{
				{ID: "1", PaymentItemID: "payment1", PurchaseAt: now.Add(-oneMonth), AutoRenewExpireAt: &[]time.Time{now}[0]},
				{ID: "2", PaymentItemID: "payment1", PurchaseAt: now, AutoRenewExpireAt: &[]time.Time{now.Add(oneMonth)}[0], RefundAt: &[]time.Time{now.Add(5 * time.Hour)}[0]},
			},
			queryAt:     now.Add(twoMonths),
			paymentStub: []*mockPaymentItem{{id: "payment1", itemType: types.PaymentItemTypeAutoRenewableSubscription}},
			wantLen:     1,
			want:        []wantItem{{id: "1", act: now.Add(-oneMonth), exp: now, remS: int64(oneMonth.Seconds())}},
		},
		{
			name: "consecutive non-renewable subscriptions",
			txs: []*models.Transaction{
				{ID: "1", PaymentItemID: "payment1", PurchaseAt: now},
				{ID: "2", PaymentItemID: "payment1", PurchaseAt: now.Add(15 * 24 * time.Hour)},
			},
			queryAt:     now.Add(40 * 24 * time.Hour),
			paymentStub: []*mockPaymentItem{{id: "payment1", itemType: types.PaymentItemTypeNonRenewableSubscription, durationHour: &oneMonthHours}},
			wantLen:     2,
			want: []wantItem{
				{id: "1", act: now, exp: now.Add(oneMonth), remS: int64(oneMonth.Seconds())},
				{id: "2", act: now.Add(oneMonth), exp: now.Add(twoMonths), remS: int64(oneMonth.Seconds())},
			},
		},
		{
			name: "consecutive non-renewable subscriptions 2",
			txs: []*models.Transaction{
				{ID: "1", PaymentItemID: "payment1", PurchaseAt: now},
				{ID: "2", PaymentItemID: "payment1", PurchaseAt: now.Add(twoMonths)},
			},
			queryAt:     now.Add(70 * 24 * time.Hour),
			paymentStub: []*mockPaymentItem{{id: "payment1", itemType: types.PaymentItemTypeNonRenewableSubscription, durationHour: &oneMonthHours}},
			wantLen:     1,
			want:        []wantItem{{id: "2", act: now.Add(twoMonths), exp: now.Add(threeMonths), remS: int64(oneMonth.Seconds())}},
		},
		{
			name: "consecutive renewable subscriptions",
			txs: []*models.Transaction{
				{ID: "1", PaymentItemID: "payment1", PurchaseAt: now, AutoRenewExpireAt: &[]time.Time{now.Add(oneMonth)}[0]},
				{ID: "2", PaymentItemID: "payment1", PurchaseAt: now.Add(oneMonth), AutoRenewExpireAt: &[]time.Time{now.Add(twoMonths)}[0]},
			},
			queryAt:     now.Add(20 * 24 * time.Hour),
			paymentStub: []*mockPaymentItem{{id: "payment1", itemType: types.PaymentItemTypeAutoRenewableSubscription}},
			wantLen:     1,
			want:        []wantItem{{id: "1", act: now, exp: now.Add(oneMonth), remS: int64(oneMonth.Seconds())}},
		},
		{
			name: "consecutive renewable subscriptions query at 35 days",
			txs: []*models.Transaction{
				{ID: "1", PaymentItemID: "payment1", PurchaseAt: now, AutoRenewExpireAt: &[]time.Time{now.Add(oneMonth)}[0]},
				{ID: "2", PaymentItemID: "payment1", PurchaseAt: now.Add(oneMonth), AutoRenewExpireAt: &[]time.Time{now.Add(twoMonths)}[0]},
			},
			queryAt:     now.Add(35 * 24 * time.Hour),
			paymentStub: []*mockPaymentItem{{id: "payment1", itemType: types.PaymentItemTypeAutoRenewableSubscription}},
			wantLen:     2,
			want: []wantItem{
				{id: "1", act: now, exp: now.Add(oneMonth), remS: int64(oneMonth.Seconds())},
				{id: "2", act: now.Add(oneMonth), exp: now.Add(twoMonths), remS: int64(oneMonth.Seconds())},
			},
		},
		{
			name: "multiple overlapping subscriptions",
			txs: []*models.Transaction{
				{ID: "1", PaymentItemID: "payment1", PurchaseAt: now},
				{ID: "2", PaymentItemID: "payment2", PurchaseAt: now.Add(10 * 24 * time.Hour), AutoRenewExpireAt: &[]time.Time{now.Add(40 * 24 * time.Hour)}[0]},
				{ID: "3", PaymentItemID: "payment3", PurchaseAt: now.Add(20 * 24 * time.Hour)},
			},
			queryAt:     now.Add(25 * 24 * time.Hour),
			paymentStub: []*mockPaymentItem{{id: "payment1", itemType: types.PaymentItemTypeNonRenewableSubscription, durationHour: &oneMonthHours}, {id: "payment2", itemType: types.PaymentItemTypeAutoRenewableSubscription}, {id: "payment3", itemType: types.PaymentItemTypeNonRenewableSubscription, durationHour: &oneMonthHours}},
			wantLen:     3,
			want: []wantItem{
				{id: "2", act: now.Add(10 * 24 * time.Hour), exp: now.Add(40 * 24 * time.Hour), remS: int64((30 * 24 * time.Hour).Seconds())},
				{id: "1", act: now.Add(40 * 24 * time.Hour), exp: now.Add(60 * 24 * time.Hour), remS: int64((20 * 24 * time.Hour).Seconds())},
				{id: "3", act: now.Add(60 * 24 * time.Hour), exp: now.Add(90 * 24 * time.Hour), remS: int64((30 * 24 * time.Hour).Seconds())},
			},
		},
		{
			name: "multiple overlapping subscriptions period",
			txs: []*models.Transaction{
				{ID: "1", PaymentItemID: "payment1", PurchaseAt: now},
				{ID: "2", PaymentItemID: "payment2", PurchaseAt: now.Add(10 * 24 * time.Hour), AutoRenewExpireAt: &[]time.Time{now.Add(40 * 24 * time.Hour)}[0]},
				{ID: "3", PaymentItemID: "payment3", PurchaseAt: now.Add(20 * 24 * time.Hour)},
				{ID: "4", PaymentItemID: "payment1", PurchaseAt: now.Add(oneHundredDays)},
				{ID: "5", PaymentItemID: "payment2", PurchaseAt: now.Add(oneHundredDays).Add(10 * 24 * time.Hour), AutoRenewExpireAt: &[]time.Time{now.Add(oneHundredDays).Add(40 * 24 * time.Hour)}[0]},
				{ID: "6", PaymentItemID: "payment3", PurchaseAt: now.Add(oneHundredDays).Add(20 * 24 * time.Hour)},
			},
			queryAt:     now.Add(oneHundredDays).Add(25 * 24 * time.Hour),
			paymentStub: []*mockPaymentItem{{id: "payment1", itemType: types.PaymentItemTypeNonRenewableSubscription, durationHour: &oneMonthHours}, {id: "payment2", itemType: types.PaymentItemTypeAutoRenewableSubscription}, {id: "payment3", itemType: types.PaymentItemTypeNonRenewableSubscription, durationHour: &oneMonthHours}},
			wantLen:     3,
			want: []wantItem{
				{id: "5", act: now.Add(oneHundredDays).Add(10 * 24 * time.Hour), exp: now.Add(oneHundredDays).Add(40 * 24 * time.Hour), remS: int64((30 * 24 * time.Hour).Seconds())},
				{id: "4", act: now.Add(oneHundredDays).Add(40 * 24 * time.Hour), exp: now.Add(oneHundredDays).Add(60 * 24 * time.Hour), remS: int64((20 * 24 * time.Hour).Seconds())},
				{id: "6", act: now.Add(oneHundredDays).Add(60 * 24 * time.Hour), exp: now.Add(oneHundredDays).Add(90 * 24 * time.Hour), remS: int64((30 * 24 * time.Hour).Seconds())},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{}
			for _, p := range tc.paymentStub {
				cfg.PaymentItems = append(cfg.PaymentItems, p.toType())
			}
			svc := NewService(cfg, nil, zap.NewNop().Sugar())

			got, err := svc.getAllActiveUserSubscriptionItems(context.Background(), tc.txs, tc.queryAt)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.wantLen, len(got))
			if len(tc.want) > 0 {
				for i := range tc.want {
					assert.Equal(t, tc.want[i].id, got[i].ID)
					assert.True(t, tc.want[i].act.Equal(got[i].ActivatedAt), "activatedAt mismatch at %d", i)
					assert.True(t, tc.want[i].exp.Equal(got[i].ExpireAt), "expireAt mismatch at %d", i)
					assert.Equal(t, tc.want[i].remS, got[i].RemainingDurationSeconds, "remaining seconds mismatch at %d", i)
				}
			}
		})
	}
}

func TestGetAllActiveUserMembershipItems_AutoRenewOverlap_UsesActivatedAtForRemainingDuration(t *testing.T) {
	now := time.Now()
	oneMonthHours := int64(30 * 24)

	cfg := &config.Config{PaymentItems: []*types.PaymentItem{
		{ID: "p1", Type: types.PaymentItemTypeNonRenewableSubscription, DurationHour: &oneMonthHours},
		{ID: "p2", Type: types.PaymentItemTypeAutoRenewableSubscription},
	}}
	svc := NewService(cfg, nil, zap.NewNop().Sugar())

	txs := []*models.Transaction{
		{ID: "1", PaymentItemID: "p1", PurchaseAt: now},
		{ID: "2", PaymentItemID: "p2", PurchaseAt: now.Add(15 * 24 * time.Hour), AutoRenewExpireAt: &[]time.Time{now.Add(45 * 24 * time.Hour)}[0]},
	}

	items, err := svc.getAllActiveUserSubscriptionItems(context.Background(), txs, now.Add(20*24*time.Hour))
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, int64((15 * 24 * time.Hour).Seconds()), items[1].RemainingDurationSeconds)
}

func TestUserSubscriptionItem_ToUserMembershipActiveItem(t *testing.T) {
	now := time.Now()
	nextRenew := now.Add(24 * time.Hour)

	item := &UserSubscriptionItem{
		Transaction: models.Transaction{
			ID:              "tx-1",
			UserID:          "user-1",
			PaymentItemID:   "vip-monthly",
			NextAutoRenewAt: &nextRenew,
		},
		RemainingDurationSeconds: int64((30 * 24 * time.Hour).Seconds()),
		ActivatedAt:              now,
		ExpireAt:                 now.Add(30 * 24 * time.Hour),
	}

	activeItem := item.ToUserMembershipActiveItem()
	require.NotNil(t, activeItem)
	require.Equal(t, "tx-1", activeItem.ID)
	require.Equal(t, "tx-1", activeItem.UserTransactionID)
	require.Equal(t, "vip-monthly", activeItem.PaymentItemID)
	require.Equal(t, "user-1", activeItem.UserID)
	require.Equal(t, item.RemainingDurationSeconds, activeItem.RemainingDurationSeconds)
	require.True(t, item.ActivatedAt.Equal(activeItem.ActivatedAt))
	require.True(t, item.ExpireAt.Equal(activeItem.ExpireAt))
	require.Equal(t, item.NextAutoRenewAt, activeItem.NextAutoRenewAt)
}
