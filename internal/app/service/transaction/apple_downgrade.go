package transaction

import (
	"context"
	"strconv"
	"time"

	"github.com/awa/go-iap/appstore"
	"github.com/awa/go-iap/appstore/api"
	types "github.com/fatflowers/cashier/pkg/types"
)

var applePaymentItemLookup = func(ctx context.Context, provider types.PaymentProvider, providerItemID string) (*types.PaymentItem, error) {
	return nil, nil
}

func detectAppleDowngrade(ctx context.Context, parseResult *VerifiedData, txInfo *api.JWSTransaction) (string, *time.Time, bool, error) {
	if parseResult == nil || parseResult.AppleReceipt == nil || txInfo == nil {
		return "", nil, false, nil
	}

	var pending *appstore.PendingRenewalInfo
	for i := range parseResult.AppleReceipt.PendingRenewalInfo {
		v := &parseResult.AppleReceipt.PendingRenewalInfo[i]
		if v.OriginalTransactionID == txInfo.OriginalTransactionId {
			pending = v
			break
		}
	}
	if pending == nil || pending.SubscriptionAutoRenewStatus != "1" || pending.ProductID == pending.SubscriptionAutoRenewProductID {
		return "", nil, false, nil
	}

	var latestMS int64
	for _, info := range parseResult.AppleReceipt.LatestReceiptInfo {
		if string(info.OriginalTransactionID) != pending.OriginalTransactionID || info.ProductID != pending.ProductID || info.ExpiresDateMS == "" {
			continue
		}
		ms, err := strconv.ParseInt(info.ExpiresDateMS, 10, 64)
		if err == nil && ms > latestMS {
			latestMS = ms
		}
	}
	if latestMS == 0 {
		return "", nil, false, nil
	}

	item, err := applePaymentItemLookup(ctx, types.PaymentProviderApple, pending.SubscriptionAutoRenewProductID)
	if err != nil || item == nil || item.ID == "" {
		return "", nil, false, err
	}

	next := time.UnixMilli(latestMS)
	return item.ID, &next, true, nil
}
