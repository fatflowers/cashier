package transaction

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/awa/go-iap/appstore"
	"github.com/awa/go-iap/appstore/api"
	types "github.com/fatflowers/cashier/pkg/types"
	"github.com/stretchr/testify/require"
)

func TestDetectAppleDowngrade_Success(t *testing.T) {
	ctx := context.Background()

	lookup := func(_ context.Context, _ types.PaymentProvider, providerItemID string) (*types.PaymentItem, error) {
		require.Equal(t, "vip.low.month", providerItemID)
		return &types.PaymentItem{ID: "vip_low"}, nil
	}

	parseResult := &VerifiedData{AppleReceipt: &appstore.IAPResponse{
		PendingRenewalInfo: []appstore.PendingRenewalInfo{{
			OriginalTransactionID:          "orig-1",
			ProductID:                      "vip.high.month",
			SubscriptionAutoRenewProductID: "vip.low.month",
			SubscriptionAutoRenewStatus:    "1",
		}},
		LatestReceiptInfo: []appstore.InApp{{
			OriginalTransactionID: "orig-1",
			ProductID:             "vip.high.month",
			ExpiresDate:           appstore.ExpiresDate{ExpiresDateMS: "1770724800000"},
		}},
	}}

	vipID, nextAt, ok, err := detectAppleDowngrade(ctx, parseResult, &api.JWSTransaction{OriginalTransactionId: "orig-1"}, lookup)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "vip_low", vipID)
	require.Equal(t, time.UnixMilli(1770724800000), *nextAt)
}

func TestDetectAppleDowngrade_FallbackToReceiptInApp(t *testing.T) {
	ctx := context.Background()

	lookup := func(_ context.Context, _ types.PaymentProvider, providerItemID string) (*types.PaymentItem, error) {
		require.Equal(t, "vip.low.month", providerItemID)
		return &types.PaymentItem{ID: "vip_low"}, nil
	}

	parseResult := &VerifiedData{AppleReceipt: &appstore.IAPResponse{
		PendingRenewalInfo: []appstore.PendingRenewalInfo{{
			OriginalTransactionID:          "orig-1",
			ProductID:                      "vip.high.month",
			SubscriptionAutoRenewProductID: "vip.low.month",
			SubscriptionAutoRenewStatus:    "1",
		}},
		LatestReceiptInfo: nil,
		Receipt: appstore.Receipt{
			InApp: []appstore.InApp{{
				OriginalTransactionID: "orig-1",
				ProductID:             "vip.high.month",
				ExpiresDate:           appstore.ExpiresDate{ExpiresDateMS: "1770724800000"},
			}},
		},
	}}

	vipID, nextAt, ok, err := detectAppleDowngrade(ctx, parseResult, &api.JWSTransaction{OriginalTransactionId: "orig-1"}, lookup)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "vip_low", vipID)
	require.Equal(t, time.UnixMilli(1770724800000), *nextAt)
}

func TestDetectAppleDowngrade_LookupError_NonBlocking(t *testing.T) {
	ctx := context.Background()

	lookup := func(_ context.Context, _ types.PaymentProvider, _ string) (*types.PaymentItem, error) {
		return nil, fmt.Errorf("lookup failed")
	}

	parseResult := &VerifiedData{AppleReceipt: &appstore.IAPResponse{
		PendingRenewalInfo: []appstore.PendingRenewalInfo{{
			OriginalTransactionID:          "orig-1",
			ProductID:                      "vip.high.month",
			SubscriptionAutoRenewProductID: "vip.low.month",
			SubscriptionAutoRenewStatus:    "1",
		}},
		LatestReceiptInfo: []appstore.InApp{{
			OriginalTransactionID: "orig-1",
			ProductID:             "vip.high.month",
			ExpiresDate:           appstore.ExpiresDate{ExpiresDateMS: "1770724800000"},
		}},
	}}

	vipID, nextAt, ok, err := detectAppleDowngrade(ctx, parseResult, &api.JWSTransaction{OriginalTransactionId: "orig-1"}, lookup)
	require.NoError(t, err)
	require.False(t, ok)
	require.Empty(t, vipID)
	require.Nil(t, nextAt)
}
