package models

import (
	"github.com/fatflowers/cashier/pkg/types"
	"time"

	"gorm.io/datatypes"
)

type UserSubscriptionItemExtra struct {
	// OperatorId 操作员ID
	OperatorId string `json:"operator_id,omitempty"`
	// PaymentItemSnapshot 支付产品快照
	PaymentItemSnapshot *types.PaymentItem `json:"payment_item_snapshot"`
	// IsFirstPurchase 是否是首次购买
	IsFirstPurchase bool `json:"is_first_purchase"`
}

// Transaction 用户订阅项目购买记录
type Transaction struct {
	ID            string                `gorm:"column:id;primary_key;type:uuid;index:idx_user_id_id,priority:2,sort:desc" json:"id"`
	UserID        string                `gorm:"column:user_id;type:varchar(64);not null;index:idx_user_id_id,priority:1" json:"user_id"`
	ProviderID    types.PaymentProvider `gorm:"column:provider_id;type:varchar(64);not null;uniqueIndex:unique_provider_id_transaction_id,priority:1" json:"provider_id"`
	PaymentItemID string                `gorm:"column:payment_item_id;type:varchar(64);not null" json:"payment_item_id"`
	TransactionID string                `gorm:"column:transaction_id;type:varchar(64);not null;uniqueIndex:unique_provider_id_transaction_id,priority:2" json:"transaction_id"`
	Currency      string                `gorm:"column:currency;type:varchar(64);not null" json:"currency"`
	Price         int64                 `gorm:"column:price;type:bigint;not null" json:"price"`
	// ParentTransactionID 父订单ID，用于自动续费
	ParentTransactionID *string `gorm:"column:parent_transaction_id;type:varchar(64);" json:"parent_transaction_id"`
	// PurchaseAt 购买时间
	PurchaseAt time.Time `gorm:"column:purchase_at;default:null" json:"purchase_at"`
	// RefundAt 退款时间
	RefundAt *time.Time `gorm:"column:refund_at;default:null" json:"refund_at"`
	// AutoRenewExpireAt 过期时间，存在于自动续费类型的订阅数据，由支付平台计算
	AutoRenewExpireAt *time.Time `gorm:"column:expire_at;default:null" json:"expire_at"`
	// 如果IsAutoRenewable为true，则NextAutoRenewAt为下次自动续费时间, 否则为null
	NextAutoRenewAt *time.Time `gorm:"column:next_auto_renew_at;default:null" json:"next_auto_renew_at"`
	// 用户升级时：原订阅的交易记录会新增revocationDate和revocationReason，标志其失效
	RevocationDate   *time.Time `gorm:"column:revocation_date;default:null" json:"revocation_date"`
	RevocationReason *string    `gorm:"column:revocation_reason;type:varchar(64);default:null" json:"revocation_reason"`

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
