package models

import (
	"github.com/fatflowers/cashier/pkg/types"
	"time"

	"gorm.io/datatypes"
)

type UserSubscriptionItemExtra struct {
	// OperatorId operator ID
	OperatorId string `json:"operator_id,omitempty"`
	// PaymentItemSnapshot payment item snapshot
	PaymentItemSnapshot *types.PaymentItem `json:"payment_item_snapshot"`
	// IsFirstPurchase indicates whether this is the user's first purchase
	IsFirstPurchase bool `json:"is_first_purchase"`
}

// Transaction stores a user subscription purchase record.
type Transaction struct {
	ID            string                `gorm:"column:id;primary_key;type:uuid;index:idx_user_id_id,priority:2,sort:desc" json:"id"`
	UserID        string                `gorm:"column:user_id;type:varchar(64);not null;index:idx_user_id_id,priority:1" json:"user_id"`
	ProviderID    types.PaymentProvider `gorm:"column:provider_id;type:varchar(64);not null;uniqueIndex:unique_provider_id_transaction_id,priority:1;uniqueIndex:unique_provider_id_before_upgraded_transaction_id,priority:1" json:"provider_id"`
	PaymentItemID string                `gorm:"column:payment_item_id;type:varchar(64);not null" json:"payment_item_id"`
	TransactionID string                `gorm:"column:transaction_id;type:varchar(64);not null;uniqueIndex:unique_provider_id_transaction_id,priority:2" json:"transaction_id"`
	Currency      string                `gorm:"column:currency;type:varchar(64);not null" json:"currency"`
	Price         int64                 `gorm:"column:price;type:bigint;not null" json:"price"`
	// ParentTransactionID is the parent transaction ID used for auto-renewal.
	ParentTransactionID *string `gorm:"column:parent_transaction_id;type:varchar(64);" json:"parent_transaction_id"`
	// PurchaseAt is the purchase time.
	PurchaseAt time.Time `gorm:"column:purchase_at;default:null" json:"purchase_at"`
	// RefundAt is the refund time.
	RefundAt *time.Time `gorm:"column:refund_at;default:null" json:"refund_at"`
	// AutoRenewExpireAt is the expiry time for auto-renewable subscriptions, calculated by the payment provider.
	AutoRenewExpireAt *time.Time `gorm:"column:expire_at;default:null" json:"expire_at"`
	// If IsAutoRenewable is true, NextAutoRenewAt is the next auto-renewal time; otherwise it is nil.
	NextAutoRenewAt *time.Time `gorm:"column:next_auto_renew_at;default:null" json:"next_auto_renew_at"`
	// During subscription upgrades, the original transaction may carry revocationDate and revocationReason to mark invalidation.
	RevocationDate   *time.Time `gorm:"column:revocation_date;default:null" json:"revocation_date"`
	RevocationReason *string    `gorm:"column:revocation_reason;type:varchar(64);default:null" json:"revocation_reason"`
	// BeforeUpgradedTransactionID points to the transaction_id this record upgrades from.
	BeforeUpgradedTransactionID *string `gorm:"column:before_upgraded_transaction_id;type:varchar(64);uniqueIndex:unique_provider_id_before_upgraded_transaction_id,priority:2" json:"before_upgraded_transaction_id"`

	Extra     datatypes.JSONType[*UserSubscriptionItemExtra] `gorm:"column:extra;type:jsonb;default:'{}'" json:"extra"`
	CreatedAt time.Time                                      `json:"created_at"`
	UpdatedAt time.Time                                      `json:"updated_at"`
}

func (Transaction) TableName() string {
	return "transaction"
}

func (item *Transaction) IsAutoRenewable() bool {
	if item == nil {
		return false
	}

	return item.NextAutoRenewAt != nil
}

func (item *Transaction) GetPaymentItemSnapshot() *types.PaymentItem {
	if item == nil || item.Extra.Data() == nil {
		return nil
	}

	return item.Extra.Data().PaymentItemSnapshot
}
