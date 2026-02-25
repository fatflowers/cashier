package transaction

import (
	"context"
	"github.com/awa/go-iap/appstore"
	models "github.com/fatflowers/cashier/internal/models"
	types "github.com/fatflowers/cashier/pkg/types"
	"time"
)

type TransactionVerifyRequest struct {
	ProviderID    string `json:"provider_id"`
	TransactionID string `json:"transaction_id"`
}

type VerificationDataRequest struct {
	ProviderID  string `json:"provider_id"`
	ReceiptData string `json:"receipt_data"`
}

type VerifiedData struct {
	AppleReceipt *appstore.IAPResponse
}

type VerifyTransactionResult struct {
	DowngradeToVipID         string              `json:"downgrade_to_vip_id,omitempty"`
	DowngradeNextAutoRenewAt *time.Time          `json:"downgrade_next_auto_renew_at,omitempty"`
	IsUpgrade                bool                `json:"is_upgrade,omitempty"`
	UserTransaction          *models.Transaction `json:"user_transaction,omitempty"`
}

func (r *VerifyTransactionResult) IsDowngrade() bool {
	return r != nil &&
		r.DowngradeToVipID != "" &&
		r.DowngradeNextAutoRenewAt != nil &&
		!r.DowngradeNextAutoRenewAt.IsZero()
}

type RefundRequest struct {
	TransactionID string `json:"transaction_id"`
	ProviderID    string `json:"provider_id"`
	OutRefundID   string `json:"out_refund_id"`
}

type SendFreeGiftRequest struct {
	UserID        string `json:"user_id"`
	PaymentItemID string `json:"payment_item_id"`
	OperatorId    string `json:"operator_id"` // operator ID
}

// TransactionManager verifies/parses transaction data and grants entitlements.
type TransactionManager interface {
	// Verify transaction info.
	VerifyTransaction(ctx context.Context, req *TransactionVerifyRequest) (*VerifyTransactionResult, error)
	// Parse verification data.
	ParseVerificationData(ctx context.Context, req *VerificationDataRequest) (*VerifiedData, error)
	// Process refund.
	RefundTransaction(ctx context.Context, transactionId string, outRefundId string) error
	// Scan transactions (used by admin list pages).
	ScanTransactions(ctx context.Context, req *ScanTransactionsRequest) (*ScanTransactionsResponse, error)
}

// Scan transaction request/response.
type ScanTransactionsRequest struct {
	Filters   []*types.CommonFilter `json:"filters"`
	From      int                   `json:"from"`
	Size      int                   `json:"size"`
	SortBy    string                `json:"sort_by"`
	SortOrder string                `json:"sort_order"`
}

type ScanTransactionsResponse struct {
	Items []*models.Transaction `json:"items"`
	Total int64                 `json:"total"`
}
