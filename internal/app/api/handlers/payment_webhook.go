package handlers

import (
	nh "github.com/fatflowers/cashier/internal/app/service/notification_handler"
	"github.com/fatflowers/cashier/pkg/logctx"
	"github.com/fatflowers/cashier/pkg/response"
	"github.com/fatflowers/cashier/pkg/types"
	"net/http"

	"github.com/gin-gonic/gin"
)

// @Summary      Apple Webhook
// @Description  Handles App Store Server Notifications V2. The request body should be a Signed JWS payload.
// @Tags         Webhook
// @Accept       json
// @Produce      json
// @Param        payload body string true "App Store Server Notification V2 JWS payload"
// @Success      200  {object}  handlers.RespOK
// @Router       /api/v2/payment/webhook/apple [post]
// ApiAppleWebhook handles App Store Server Notifications
func ApiAppleWebhook(h *nh.NotificationHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		logctx.FromCtx(c, h.Logger).Infow("webhook_apple_received")

		if err := h.HandleNotification(c, types.PaymentProviderApple); err != nil {
			logctx.FromCtx(c, h.Logger).Errorw("webhook_apple_handle_error", "error", err.Error())
			c.JSON(http.StatusOK, response.ErrorT[any](response.APIResponseCodeError, err.Error()))
			return
		}
		logctx.FromCtx(c, h.Logger).Infow("webhook_apple_handled")
		c.JSON(http.StatusOK, response.OKT[any](nil))
	}
}

func RegisterPaymentWebhookRoutes(r gin.IRouter, h *nh.NotificationHandler) {
	// Mount under provided group, expected at "/api"
	r.POST("/apple", ApiAppleWebhook(h))
}
