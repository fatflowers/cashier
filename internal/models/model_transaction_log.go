package models

import (
	"github.com/fatflowers/cashier/pkg/types"
	"time"

	"gorm.io/datatypes"
)

// TransactionLog 用户订阅项目变更日志
// 使用场景：用户订阅项目变更记录，用于问题排查
type TransactionLog struct {
	ID            string                `gorm:"column:id;primary_key;type:uuid;index:idx_user_id_id,priority:2,sort:desc"`
	UserID        string                `gorm:"column:user_id;type:varchar(64);index:idx_user_id_id,priority:1;not null"`
	PaymentItemID string                `gorm:"column:payment_item_id;type:varchar(64);not null"`
	ProviderID    types.PaymentProvider `gorm:"column:provider_id;type:varchar(64);not null"`
	TransactionID string                `gorm:"column:transaction_id;type:varchar(64);not null"`
	// Reason 变更原因
	Reason types.SubscriptionChangeReason `gorm:"column:reason;type:varchar(64);not null"`
	// Before 变更前的订阅信息，JSON格式存储
	Before datatypes.JSONType[*Transaction] `gorm:"column:before;type:jsonb;default:'null'"`
	// After 变更后的订阅信息，JSON格式存储
	After datatypes.JSONType[*Transaction] `gorm:"column:after;type:jsonb;default:'null'"`
	// Extra 额外的上下文信息，如变更原因、触发来源等
	Extra     datatypes.JSONMap `gorm:"column:extra;type:jsonb;default:'{}'"`
	CreatedAt time.Time         `json:"created_at"`
}

func (TransactionLog) TableName() string {
	return "transaction_log"
}
