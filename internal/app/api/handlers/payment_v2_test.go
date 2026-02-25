package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/fatflowers/cashier/internal/app/service/transaction"
)

type stubTxMgr struct{}

func (s *stubTxMgr) VerifyTransaction(_ context.Context, _ *transaction.TransactionVerifyRequest) (*transaction.VerifyTransactionResult, error) {
	next := time.Unix(1735689600, 0)
	return &transaction.VerifyTransactionResult{
		DowngradeToVipID:         "vip_low",
		DowngradeNextAutoRenewAt: &next,
	}, nil
}

func (s *stubTxMgr) ParseVerificationData(_ context.Context, _ *transaction.VerificationDataRequest) (*transaction.VerifiedData, error) {
	panic("not used")
}

func (s *stubTxMgr) RefundTransaction(_ context.Context, _ string, _ string) error {
	panic("not used")
}

func (s *stubTxMgr) ScanTransactions(_ context.Context, _ *transaction.ScanTransactionsRequest) (*transaction.ScanTransactionsResponse, error) {
	panic("not used")
}

func TestApiVerifyTransactionV2_ReturnsDowngradeInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v2/payment/verify_transaction", ApiVerifyTransactionV2(&stubTxMgr{}))

	body, _ := json.Marshal(map[string]any{"provider_id": "apple", "transaction_id": "tx-1", "server_verification_data": "abc"})
	req := httptest.NewRequest(http.MethodPost, "/api/v2/payment/verify_transaction", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "down_grade_auto_renew_info")
}
