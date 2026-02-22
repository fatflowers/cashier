package notification_handler

import (
	"context"
	"fmt"
	"github.com/fatflowers/cashier/internal/models"
	"github.com/fatflowers/cashier/internal/platform/apple/apple_iap"
	"github.com/fatflowers/cashier/internal/platform/apple/apple_notification"
	"github.com/fatflowers/cashier/pkg/config"
	"github.com/fatflowers/cashier/pkg/types"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"gorm.io/datatypes"
)

type AppleNotificationParser struct {
	cfg              *config.Config
	NotificationTime time.Time
	Notification     *apple_notification.AppStoreServerNotification
}

func (p *AppleNotificationParser) GetProvider(ctx context.Context) types.PaymentProvider {
	return types.PaymentProviderApple
}

func (p *AppleNotificationParser) GetNotificationTime(ctx context.Context) time.Time {
	return p.NotificationTime
}

func (p *AppleNotificationParser) GetApp(ctx context.Context) string {
	return p.Notification.TransactionInfo.BundleId
}

func (p *AppleNotificationParser) GetUserID(ctx context.Context) (string, error) {
	if p.Notification.TransactionInfo.AppAccountToken == "" {
		return "", fmt.Errorf("app account token is empty")
	}

	return apple_iap.UUIDToUserID(p.Notification.TransactionInfo.AppAccountToken)
}

func (p *AppleNotificationParser) GetTransactionID(ctx context.Context) string {
	return p.Notification.TransactionInfo.TransactionId
}

func (p *AppleNotificationParser) GetPaymentItem(ctx context.Context) (*types.PaymentItem, error) {
	return p.cfg.GetPaymentItemByProviderItemID(ctx, p.GetProvider(ctx), p.Notification.TransactionInfo.ProductId)
}

func (p *AppleNotificationParser) GetTransaction(ctx context.Context) (*models.Transaction, error) {
	paymentItem, err := p.GetPaymentItem(ctx)
	if err != nil {
		return nil, err
	}

	if paymentItem.Type != types.PaymentItemTypeAutoRenewableSubscription && paymentItem.Type != types.PaymentItemTypeNonRenewableSubscription {
		return nil, nil
	}

	userID, err := p.GetUserID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user id: %w", err)
	}

	res := &models.Transaction{
		UserID:        userID,
		ProviderID:    p.GetProvider(ctx),
		PaymentItemID: paymentItem.ID,
		TransactionID: p.GetTransactionID(ctx),
		Currency:      p.Notification.TransactionInfo.Currency,
		Price:         p.Notification.TransactionInfo.Price * 100,
		PurchaseAt:    time.UnixMilli(int64(p.Notification.TransactionInfo.PurchaseDate)),
		Extra: datatypes.NewJSONType(&models.UserSubscriptionItemExtra{
			PaymentItemSnapshot: paymentItem,
		}),
	}

	if p.Notification.TransactionInfo.RevocationDate > 0 {
		res.RefundAt = lo.ToPtr(time.UnixMilli(int64(p.Notification.TransactionInfo.RevocationDate)))
	}

	if paymentItem.Renewable() && p.Notification.TransactionInfo.ExpiresDate > 0 {
		res.AutoRenewExpireAt = lo.ToPtr(time.UnixMilli(int64(p.Notification.TransactionInfo.ExpiresDate)))
	}

	if p.Notification.RenewalInfo != nil {
		// https://developer.apple.com/documentation/appstoreserverapi/autorenewstatus
		if p.Notification.RenewalInfo.AutoRenewStatus == 1 && p.Notification.RenewalInfo.RenewalDate > 0 {
			res.NextAutoRenewAt = lo.ToPtr(time.UnixMilli(int64(p.Notification.RenewalInfo.RenewalDate)))
		}
		res.ParentTransactionID = lo.ToPtr(p.Notification.RenewalInfo.OriginalTransactionId)
	}

	return res, nil
}

func (p *AppleNotificationParser) GetData(ctx context.Context) any {
	return p.Notification
}

func GetAppleNotificationParser(cfg *config.Config, ginCtx *gin.Context, notificationTime time.Time) (NotificationParser, error) {
	if notificationTime.IsZero() {
		notificationTime = time.Now()
	}

	var request apple_notification.AppStoreServerRequest
	err := ginCtx.BindJSON(&request)
	if err != nil {
		return nil, err
	}

	notification, err := apple_notification.New(request.SignedPayload)
	if err != nil {
		return nil, err
	}

	return &AppleNotificationParser{
		cfg:              cfg,
		NotificationTime: notificationTime,
		Notification:     notification,
	}, nil
}
