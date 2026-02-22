package handlers

import (
    "github.com/fatflowers/cashier/internal/app/service/statistics"
    "github.com/fatflowers/cashier/pkg/response"
    types "github.com/fatflowers/cashier/pkg/types"
    "time"
)

// RespOK is a generic OK envelope for endpoints returning no specific data.
type RespOK struct {
    Code    response.APIResponseCode `json:"code"`
    Message string                   `json:"message"`
    Data    interface{}              `json:"data"`
}

// RespListMembershipTransactions wraps ListMembershipTransactionsResponse in the standard envelope.
type RespListMembershipTransactions struct {
    Code    response.APIResponseCode        `json:"code"`
    Message string                          `json:"message"`
    Data    ListMembershipTransactionsResponse `json:"data"`
}

// RespMembershipStatistic wraps MembershipStatisticResponse in the standard envelope.
type RespMembershipStatistic struct {
    Code    response.APIResponseCode     `json:"code"`
    Message string                       `json:"message"`
    Data    statistics.MembershipStatisticResponse `json:"data"`
}

// RespUserListTransactions wraps a list of transactions in the standard envelope.
type RespUserListTransactions struct {
    Code    response.APIResponseCode `json:"code"`
    Message string                   `json:"message"`
    Data    []SwaggerTransaction     `json:"data"`
}

// SwaggerTransaction is a simplified view of models.Transaction for documentation purposes.
type SwaggerTransaction struct {
    ID                  string                `json:"id"`
    TransactionID       string                `json:"transaction_id"`
    UserID              string                `json:"user_id"`
    Currency            string                `json:"currency"`
    Price               int64                 `json:"price"`
    ProviderID          types.PaymentProvider `json:"provider_id"`
    PurchaseAt          time.Time             `json:"purchase_at"`
    RefundAt            *time.Time            `json:"refund_at"`
    NextAutoRenewAt     *time.Time            `json:"next_auto_renew_at"`
    AutoRenewExpireAt   *time.Time            `json:"auto_renew_expire_at"`
    ParentTransactionID *string               `json:"parent_transaction_id"`
    CreatedAt           time.Time             `json:"created_at"`
    UpdatedAt           time.Time             `json:"updated_at"`
    PaymentItemID       string                `json:"payment_item_id"`
}
