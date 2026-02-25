package transaction

import (
	"context"
	"fmt"
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
	matchedReceipt := false
	missingExpires := false
	invalidExpires := false
	for _, info := range receiptInfos {
		if string(info.OriginalTransactionID) != pending.OriginalTransactionID || info.ProductID != pending.ProductID {
			continue
		}
		matchedReceipt = true
		if info.ExpiresDateMS == "" {
			missingExpires = true
			continue
		}
		ms, err := strconv.ParseInt(info.ExpiresDateMS, 10, 64)
		if err != nil {
			invalidExpires = true
			continue
		}
		if ms > latestMS {
			latestMS = ms
		}
	}
	if latestMS == 0 {
		if matchedReceipt && (missingExpires || invalidExpires) {
			if invalidExpires {
				return "", nil, false, fmt.Errorf("invalid expires_date_ms in receipt for original_transaction_id=%s", pending.OriginalTransactionID)
			}
			return "", nil, false, fmt.Errorf("missing expires_date_ms in receipt for original_transaction_id=%s", pending.OriginalTransactionID)
		}
		return "", nil, false, nil
	}

	if lookup == nil {
		return "", nil, false, fmt.Errorf("apple payment item lookup is nil")
	}
	item, err := lookup(ctx, types.PaymentProviderApple, pending.SubscriptionAutoRenewProductID)
	if err != nil {
		return "", nil, false, fmt.Errorf("failed to lookup payment item by provider item id=%s: %w", pending.SubscriptionAutoRenewProductID, err)
	}
	if item == nil || item.ID == "" {
		return "", nil, false, fmt.Errorf("payment item not found for provider item id=%s", pending.SubscriptionAutoRenewProductID)
	}

	next := time.UnixMilli(latestMS)
	return item.ID, &next, true, nil
}
