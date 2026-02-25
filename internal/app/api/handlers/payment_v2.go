package handlers

import (
	"errors"
	"net/http"
	"time"

	nh "github.com/fatflowers/cashier/internal/app/service/notification_handler"
	"github.com/fatflowers/cashier/internal/app/service/transaction"
	"github.com/fatflowers/cashier/pkg/response"
	"github.com/gin-gonic/gin"
)

type downGradeAutoRenewInfo struct {
	VipID           string    `json:"vip_id"`
	NextAutoRenewAt time.Time `json:"next_auto_renew_at"`
}

type verifyTransactionV2Resp struct {
	DownGradeAutoRenewInfo *downGradeAutoRenewInfo `json:"down_grade_auto_renew_info,omitempty"`
}

// @Summary      Verify Transaction V2
// @Description  Verifies a payment transaction and returns downgrade auto-renew information when needed.
// @Tags         Payment
// @Accept       json
// @Produce      json
// @Param        request body transaction.TransactionVerifyRequest true "Transaction verification request"
// @Success      200  {object}  handlers.RespOK
// @Router       /api/v2/payment/verify_transaction [post]
func ApiVerifyTransactionV2(mgr transaction.TransactionManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req transaction.TransactionVerifyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusOK, response.ErrorT[any](response.APIResponseCodeBadRequest, err.Error()))
			return
		}

		res, err := mgr.VerifyTransaction(c.Request.Context(), &req)
		if err != nil {
			if errors.Is(err, transaction.ErrVerifyTransactionDuplicate) {
				c.JSON(http.StatusOK, response.ErrorT[any](response.APIResponseCodeBadRequest, err.Error()))
				return
			}
			c.JSON(http.StatusOK, response.ErrorT[any](response.APIResponseCodeError, err.Error()))
			return
		}

		out := verifyTransactionV2Resp{}
		if res != nil && res.IsDowngrade() {
			out.DownGradeAutoRenewInfo = &downGradeAutoRenewInfo{VipID: res.DowngradeToVipID, NextAutoRenewAt: *res.DowngradeNextAutoRenewAt}
		}
		c.JSON(http.StatusOK, response.OKT(out))
	}
}

func RegisterPaymentV2Routes(r gin.IRouter, mgr transaction.TransactionManager, notifHandler *nh.NotificationHandler) {
	r.POST("/verify_transaction", ApiVerifyTransactionV2(mgr))
	r.POST("/webhook/apple", ApiAppleWebhook(notifHandler))
}
