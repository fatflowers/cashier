package transaction

import (
	"context"
	models "github.com/fatflowers/cashier/internal/models"
	"github.com/fatflowers/cashier/internal/platform/apple/apple_iap"
	types "github.com/fatflowers/cashier/pkg/types"
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
	AppleReceipt *apple_iap.Receipt
}

type RefundRequest struct {
	TransactionID string `json:"transaction_id"`
	ProviderID    string `json:"provider_id"`
	OutRefundID   string `json:"out_refund_id"`
}

type SendFreeGiftRequest struct {
	UserID        string `json:"user_id"`
	PaymentItemID string `json:"payment_item_id"`
	OperatorId    string `json:"operator_id"` // 操作员ID
}

// 校验解析交易信息，并发放权益
type TransactionManager interface {
	// 校验交易信息
	VerifyTransaction(ctx context.Context, req *TransactionVerifyRequest) error
	// 解析交易信息
	ParseVerificationData(ctx context.Context, req *VerificationDataRequest) (*VerifiedData, error)
	// 退款
	RefundTransaction(ctx context.Context, transactionId string, outRefundId string) error
	// 扫描交易（用于后台管理列表）
	ScanTransactions(ctx context.Context, req *ScanTransactionsRequest) (*ScanTransactionsResponse, error)
}

// 扫描交易请求/响应
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
