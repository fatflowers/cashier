package notification_handler

import (
	"context"
	"github.com/fatflowers/cashier/internal/models"
	"github.com/fatflowers/cashier/pkg/types"
	"time"
)

type NotificationParser interface {
	GetProvider(ctx context.Context) types.PaymentProvider
	GetNotificationTime(ctx context.Context) time.Time
	GetApp(ctx context.Context) string
	GetUserID(ctx context.Context) (string, error)
	GetTransactionID(ctx context.Context) string
	GetPaymentItem(ctx context.Context) (*types.PaymentItem, error)
	GetTransaction(ctx context.Context) (*models.Transaction, error)
	GetData(ctx context.Context) any
}
