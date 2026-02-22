package notification_handler

import (
	"encoding/json"
	"fmt"
	notificationlog "github.com/fatflowers/cashier/internal/app/service/notification_log"
	subscription "github.com/fatflowers/cashier/internal/app/service/subscription"
	models "github.com/fatflowers/cashier/internal/models"
	"github.com/fatflowers/cashier/pkg/config"
	"github.com/fatflowers/cashier/pkg/types"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"gorm.io/datatypes"
)

type NotificationHandler struct {
	cfg      *config.Config
	notifSvc *notificationlog.Service
	subSvc   *subscription.Service
	Logger   *zap.SugaredLogger
}

func NewNotificationHandler(cfg *config.Config, notif *notificationlog.Service, sub *subscription.Service, log *zap.SugaredLogger) *NotificationHandler {
	return &NotificationHandler{cfg: cfg, notifSvc: notif, subSvc: sub, Logger: log}
}

func (h *NotificationHandler) HandleNotification(c *gin.Context, provider types.PaymentProvider) (resErr error) {
	// Build provider-specific parser
	var parser NotificationParser
	var err error
	switch provider {
	case types.PaymentProviderApple:
		parser, err = GetAppleNotificationParser(h.cfg, c, time.Now())
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}

	// Prepare initial log fields
	var userID string
	if v, e := parser.GetUserID(c.Request.Context()); e == nil {
		userID = v
	}
	var traceID string
	if v, ok := c.Get("traceID"); ok {
		if s, ok2 := v.(string); ok2 {
			traceID = s
		}
	}
	dataBytes, _ := json.Marshal(parser.GetData(c.Request.Context()))

	// Save 'received' log
	h.notifSvc.Save(c.Request.Context(), &models.PaymentNotificationLog{
		ProviderID: string(provider),
		UserID: func() *string {
			if userID == "" {
				return nil
			}
			return lo.ToPtr(userID)
		}(),
		TraceID:          traceID,
		TransactionID:    parser.GetTransactionID(c.Request.Context()),
		NotificationTime: parser.GetNotificationTime(c.Request.Context()),
		Data:             datatypes.JSON(dataBytes),
		Status:           models.PaymentNotificationLogStatusReceived,
	})

	// Process notification → transaction → subscription
	var txn *models.Transaction
	defer func() {
		// Build result payload
		resMap := map[string]any{
			"transaction": txn,
		}
		if resErr != nil {
			resMap["error"] = resErr.Error()
		}
		resBytes, _ := json.Marshal(resMap)
		status := models.PaymentNotificationLogStatusHandled
		if resErr != nil {
			status = models.PaymentNotificationLogStatusHandleFailed
		}
		h.notifSvc.Save(c.Request.Context(), &models.PaymentNotificationLog{
			ProviderID: string(provider),
			UserID: func() *string {
				if userID == "" {
					return nil
				}
				return lo.ToPtr(userID)
			}(),
			TraceID:          traceID,
			TransactionID:    parser.GetTransactionID(c.Request.Context()),
			NotificationTime: time.Now(),
			Data:             datatypes.JSON(dataBytes),
			Result:           func() *datatypes.JSON { j := datatypes.JSON(resBytes); return &j }(),
			Status:           status,
		})
	}()

	txn, resErr = parser.GetTransaction(c.Request.Context())
	if resErr != nil {
		h.Logger.Errorw("failed to get transaction", "error", resErr.Error())
		resErr = fmt.Errorf("failed to get transaction: %w", resErr)
		return resErr
	}

	h.Logger.Infow("got transaction", "transaction", txn)

	if txn != nil {
		return h.subSvc.UpsertUserSubscriptionByItem(c.Request.Context(), txn)
	}

	resErr = fmt.Errorf("unsupported notification type")
	return resErr
}
