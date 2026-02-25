package transaction

import (
	"context"
	"strconv"
	"time"

	"github.com/awa/go-iap/appstore"
	"github.com/awa/go-iap/appstore/api"
	types "github.com/fatflowers/cashier/pkg/types"
)

type applePaymentItemLookupFn func(ctx context.Context, provider types.PaymentProvider, providerItemID string) (*types.PaymentItem, error)

func detectAppleDowngrade(ctx context.Context, parseResult *VerifiedData, txInfo *api.JWSTransaction, lookup applePaymentItemLookupFn) (string, *time.Time, bool, error) {
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

	receiptInfos := parseResult.AppleReceipt.LatestReceiptInfo
	if len(receiptInfos) == 0 {
		// Fallback to in_app when latest_receipt_info is absent.
		receiptInfos = parseResult.AppleReceipt.Receipt.InApp
	}

	var latestMS int64
	for _, info := range receiptInfos {
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

	if lookup == nil {
		return "", nil, false, nil
	}
	item, err := lookup(ctx, types.PaymentProviderApple, pending.SubscriptionAutoRenewProductID)
	if err != nil || item == nil || item.ID == "" {
		// Keep downgrade detection best-effort: lookup issues should not break verify flow.
		return "", nil, false, nil
	}

	next := time.UnixMilli(latestMS)
	return item.ID, &next, true, nil
}
