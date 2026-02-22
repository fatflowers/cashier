package models

import (
	"github.com/fatflowers/cashier/pkg/types"
	"time"

	"gorm.io/datatypes"
)

// TransactionLog records changes to user subscription transactions.
// Use case: troubleshooting subscription transaction changes.
type TransactionLog struct {
	ID            string                `gorm:"column:id;primary_key;type:uuid;index:idx_user_id_id,priority:2,sort:desc"`
	UserID        string                `gorm:"column:user_id;type:varchar(64);index:idx_user_id_id,priority:1;not null"`
	PaymentItemID string                `gorm:"column:payment_item_id;type:varchar(64);not null"`
	ProviderID    types.PaymentProvider `gorm:"column:provider_id;type:varchar(64);not null"`
	TransactionID string                `gorm:"column:transaction_id;type:varchar(64);not null"`
	// Reason is the change reason.
	Reason types.SubscriptionChangeReason `gorm:"column:reason;type:varchar(64);not null"`
	// Before stores subscription data before the change in JSON format.
	Before datatypes.JSONType[*Transaction] `gorm:"column:before;type:jsonb;default:'null'"`
	// After stores subscription data after the change in JSON format.
	After datatypes.JSONType[*Transaction] `gorm:"column:after;type:jsonb;default:'null'"`
	// Extra stores additional context such as reason details and trigger source.
	Extra     datatypes.JSONMap `gorm:"column:extra;type:jsonb;default:'{}'"`
	CreatedAt time.Time         `json:"created_at"`
}

func (TransactionLog) TableName() string {
	return "transaction_log"
}
