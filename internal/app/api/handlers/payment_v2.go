package handlers

import (
	"errors"
	"net/http"
	"time"

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
