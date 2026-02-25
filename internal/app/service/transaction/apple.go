package transaction

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	notificationlog "github.com/fatflowers/cashier/internal/app/service/notification_log"
	"github.com/fatflowers/cashier/internal/app/service/subscription"
	models "github.com/fatflowers/cashier/internal/models"
	"github.com/fatflowers/cashier/internal/platform/apple/apple_iap"
	"github.com/fatflowers/cashier/pkg/config"
	types "github.com/fatflowers/cashier/pkg/types"
	"strings"
	"time"

	"github.com/awa/go-iap/appstore/api"
	"github.com/samber/lo"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// AppleTransactionManager processes Apple IAP transactions
type AppleTransactionManager struct {
	iapClient *api.StoreClient
	opts      *apple_iap.GetAppleIAPClientOptions
	cfg       *config.Config
	db        *gorm.DB
	subSvc    *subscription.Service
	notifSvc  *notificationlog.Service
}

func NewAppleTransactionManager(cfg *config.Config, db *gorm.DB, sub *subscription.Service, notif *notificationlog.Service) (*AppleTransactionManager, error) {
	opts := &apple_iap.GetAppleIAPClientOptions{
		KeyID:        cfg.AppleIAP.KeyID,
		KeyContent:   cfg.AppleIAP.KeyContent,
		BundleID:     cfg.AppleIAP.BundleID,
		Issuer:       cfg.AppleIAP.Issuer,
		SharedSecret: cfg.AppleIAP.SharedSecret,
		Sandbox:      !cfg.AppleIAP.IsProd,
	}
	cli, err := apple_iap.GetAppleIAPClient(context.Background(), opts)
	if err != nil {
		return nil, fmt.Errorf("failed to init Apple IAP client: %w", err)
	}
	return &AppleTransactionManager{iapClient: cli, opts: opts, cfg: cfg, db: db, subSvc: sub, notifSvc: notif}, nil
}

// Request/response types are defined in manager.go in this package.

func (a *AppleTransactionManager) getPaymentItemByProviderItemID(providerID types.PaymentProvider, providerItemID string) *types.PaymentItem {
	for _, it := range a.cfg.PaymentItems {
		if it.ProviderID == providerID && it.ProviderItemID == providerItemID {
			return it
		}
	}
	return nil
}

func (a *AppleTransactionManager) toTransaction(ctx context.Context, ti *api.JWSTransaction) (*models.Transaction, error) {
	paymentItem := a.getPaymentItemByProviderItemID(types.PaymentProviderApple, ti.ProductID)
	if paymentItem == nil {
		return nil, fmt.Errorf("payment item not found for product: %s", ti.ProductID)
	}

	if ti.AppAccountToken == "" {
		return nil, fmt.Errorf("app account token is empty")
	}
	userID, err := apple_iap.UUIDToUserID(ti.AppAccountToken)
	if err != nil {
		return nil, fmt.Errorf("invalid app account token: %w", err)
	}

	res := &models.Transaction{
		UserID:        userID,
		ProviderID:    types.PaymentProviderApple,
		PaymentItemID: paymentItem.ID,
		TransactionID: ti.TransactionID,
		PurchaseAt:    time.UnixMilli(int64(ti.PurchaseDate)),
		Price:         ti.Price * 100,
		Currency:      ti.Currency,
		Extra: datatypes.NewJSONType(&models.UserSubscriptionItemExtra{
			PaymentItemSnapshot: paymentItem,
		}),
	}

	if ti.OriginalTransactionId != "" {
		res.ParentTransactionID = lo.ToPtr(ti.OriginalTransactionId)
	}
	if ti.RevocationDate > 0 {
		res.RefundAt = lo.ToPtr(time.UnixMilli(int64(ti.RevocationDate)))
	}

	if ti.Type == api.AutoRenewable {
		if ti.ExpiresDate > 0 {
			res.AutoRenewExpireAt = lo.ToPtr(time.UnixMilli(int64(ti.ExpiresDate)))
		} else {
			return nil, fmt.Errorf("auto renew transaction expires date is 0")
		}

		statuses, err := a.iapClient.GetALLSubscriptionStatuses(ctx, ti.TransactionID, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get subscription status: %w", err)
		}

		for _, item := range statuses.Data {
			if ti.SubscriptionGroupIdentifier != item.SubscriptionGroupIdentifier {
				continue
			}
			for _, last := range item.LastTransactions {
				value, err := a.iapClient.ParseJWSEncodeString(last.SignedRenewalInfo)
				if err != nil {
					return nil, fmt.Errorf("failed to parse signed renewal info: %w", err)
				}
				renewalInfo := value.(*api.JWSRenewalInfoDecodedPayload)
				if renewalInfo.ProductId == ti.ProductID && renewalInfo.AutoRenewStatus == api.AutoRenewStatusOn && renewalInfo.RenewalDate > 0 {
					res.NextAutoRenewAt = lo.ToPtr(time.UnixMilli(int64(renewalInfo.RenewalDate)))
					if res.ParentTransactionID == nil {
						res.ParentTransactionID = lo.ToPtr(renewalInfo.OriginalTransactionId)
					}
				}
			}
		}
	}
	return res, nil
}

func (a *AppleTransactionManager) existsSamePurchaseTransaction(ctx context.Context, transactionID string, providerID types.PaymentProvider, parentTransactionID string, purchaseAt time.Time) (bool, error) {
	var t models.Transaction
	err := a.db.WithContext(ctx).Where(
		"transaction_id <> ? AND provider_id = ? AND parent_transaction_id = ? AND purchase_at = ?",
		transactionID, providerID, parentTransactionID, purchaseAt,
	).First(&t).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func mapDuplicateErr(msg string) error {
	if strings.Contains(msg, "duplicate transaction already exists") {
		return fmt.Errorf("%w: %s", ErrVerifyTransactionDuplicate, msg)
	}
	return errors.New(msg)
}

func (a *AppleTransactionManager) getTransactionByProviderTransactionID(ctx context.Context, providerID types.PaymentProvider, transactionID string) (*models.Transaction, error) {
	var item models.Transaction
	if err := a.db.WithContext(ctx).
		Where("provider_id = ? AND transaction_id = ?", providerID, transactionID).
		First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (a *AppleTransactionManager) VerifyTransaction(ctx context.Context, req *TransactionVerifyRequest) (*VerifyTransactionResult, error) {
	result := &VerifyTransactionResult{}
	// Prepare and save a 'received' notification log
	var userIDPtr *string
	if v, ok := ctx.Value("user_id").(string); ok && v != "" {
		userIDPtr = &v
	}
	var traceID string
	if v, ok := ctx.Value("traceID").(string); ok {
		traceID = v
	}
	dataBytes, _ := json.Marshal(req)
	receivedLog := &models.PaymentNotificationLog{
		ProviderID:       string(types.PaymentProviderApple),
		UserID:           userIDPtr,
		TraceID:          traceID,
		NotificationTime: time.Now(),
		Data:             datatypes.JSON(dataBytes),
		Status:           models.PaymentNotificationLogStatusReceived,
	}
	a.notifSvc.Save(ctx, receivedLog)

	var mappedItem *models.Transaction
	var txInfo *api.JWSTransaction
	var retErr error
	defer func() {
		resMap := map[string]any{
			"membership_item":  mappedItem,
			"transaction_info": txInfo,
		}
		if retErr != nil {
			resMap["error"] = retErr.Error()
		}
		resBytes, _ := json.Marshal(resMap)
		status := models.PaymentNotificationLogStatusHandled
		if retErr != nil {
			status = models.PaymentNotificationLogStatusHandleFailed
		}
		a.notifSvc.Save(ctx, &models.PaymentNotificationLog{
			ProviderID: string(types.PaymentProviderApple),
			UserID:     userIDPtr,
			TraceID:    traceID,
			TransactionID: func() string {
				if txInfo != nil {
					return txInfo.TransactionID
				}
				return ""
			}(),
			NotificationTime: time.Now(),
			Data:             datatypes.JSON(dataBytes),
			Result:           func() *datatypes.JSON { j := datatypes.JSON(resBytes); return &j }(),
			Status:           status,
		})
	}()
	// Fetch and parse transaction
	infoResp, err := a.iapClient.GetTransactionInfo(ctx, req.TransactionID)
	if err != nil {
		retErr = fmt.Errorf("failed to get transaction info: %w", err)
		return nil, retErr
	}
	txInfo, err = a.iapClient.ParseSignedTransaction(infoResp.SignedTransactionInfo)
	if err != nil {
		retErr = fmt.Errorf("failed to parse signed transaction: %w", err)
		return nil, retErr
	}

	// Environment check
	if a.cfg.AppleIAP.IsProd && txInfo.Environment != api.Production {
		retErr = fmt.Errorf("transaction is not in production environment")
		return nil, retErr
	}

	if txInfo.Type != api.AutoRenewable && txInfo.Type != api.NonRenewable {
		retErr = fmt.Errorf("unsupported transaction type: %s", txInfo.Type)
		return nil, retErr
	}

	item, err := a.toTransaction(ctx, txInfo)
	if err != nil {
		retErr = fmt.Errorf("failed to map transaction: %w", err)
		return nil, retErr
	}
	mappedItem = item

	if txInfo.Type == api.AutoRenewable {
		if req.ServerVerificationData == "" {
			retErr = fmt.Errorf("server verification data is empty")
			return nil, retErr
		}

		parseResult, err := a.ParseVerificationData(ctx, &VerificationDataRequest{
			ProviderID:  string(types.PaymentProviderApple),
			ReceiptData: req.ServerVerificationData,
		})
		if err != nil {
			retErr = fmt.Errorf("failed to parse verification data: %w", err)
			return nil, retErr
		}

		downgradeVipID, downgradeAt, ok, err := detectAppleDowngrade(ctx, parseResult, txInfo, func(ctx context.Context, provider types.PaymentProvider, providerItemID string) (*types.PaymentItem, error) {
			return a.cfg.GetPaymentItemByProviderItemID(ctx, provider, providerItemID)
		})
		if err != nil {
			// Downgrade detection is best-effort and should not block verify flow.
		} else if ok {
			result.DowngradeToVipID = downgradeVipID
			result.DowngradeNextAutoRenewAt = downgradeAt
			if existing, e := a.getTransactionByProviderTransactionID(ctx, types.PaymentProviderApple, txInfo.TransactionID); e == nil {
				result.UserTransaction = existing
			}
			return result, nil
		}

		// Determine upgrade by looking at the immediate next receipt item.
		receiptInfos := parseResult.AppleReceipt.LatestReceiptInfo
		if len(receiptInfos) == 0 {
			receiptInfos = parseResult.AppleReceipt.Receipt.InApp
		}
		currentIdx := -1
		for i, info := range receiptInfos {
			if info.TransactionID == txInfo.TransactionID {
				currentIdx = i
				break
			}
		}
		if currentIdx >= 0 && currentIdx+1 < len(receiptInfos) && receiptInfos[currentIdx+1].IsUpgraded == "true" {
			result.IsUpgrade = true
		}
	}

	if txInfo.Type == api.AutoRenewable && item.ParentTransactionID != nil {
		exists, err := a.existsSamePurchaseTransaction(ctx, txInfo.TransactionID, types.PaymentProviderApple, *item.ParentTransactionID, item.PurchaseAt)
		if err != nil {
			retErr = fmt.Errorf("failed to check duplicate transaction: %w", err)
			return nil, retErr
		}
		if exists {
			retErr = mapDuplicateErr(fmt.Sprintf("duplicate transaction already exists: %s", txInfo.TransactionID))
			return nil, retErr
		}
	}

	// Persist via subscription service
	if err := a.subSvc.UpsertUserSubscriptionByItem(ctx, item); err != nil {
		retErr = fmt.Errorf("failed to upsert membership: %w", err)
		return nil, retErr
	}

	if persisted, err := a.getTransactionByProviderTransactionID(ctx, types.PaymentProviderApple, txInfo.TransactionID); err == nil {
		result.UserTransaction = persisted
	} else {
		// Non-fatal fallback to mapped transaction view.
		result.UserTransaction = item
	}
	return result, nil
}

func (a *AppleTransactionManager) ParseVerificationData(ctx context.Context, req *VerificationDataRequest) (*VerifiedData, error) {
	receipt, err := apple_iap.VerifyServerVerificationData(ctx, req.ReceiptData, a.opts)
	if err != nil {
		return nil, fmt.Errorf("failed to verify server verification data: %w", err)
	}
	return &VerifiedData{AppleReceipt: receipt}, nil
}

func (a *AppleTransactionManager) RefundTransaction(ctx context.Context, transactionId string, outRefundId string) error {
	return fmt.Errorf("apple refund transaction is not supported yet")
}
