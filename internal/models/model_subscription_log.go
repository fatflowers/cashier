package models

import (
	"github.com/fatflowers/cashier/pkg/types"
	"time"

	"gorm.io/datatypes"
)

// SubscriptionLog records changes to user subscriptions.
// Use case: troubleshooting.
type SubscriptionLog struct {
	ID     string `gorm:"column:id;type:uuid;primary_key" json:"id"`
	UserID string `gorm:"column:user_id;type:varchar(64);index:idx_user_id_id,priority:1;not null"`
	// Reason is the change reason.
	Reason types.SubscriptionChangeReason `gorm:"column:reason;type:varchar(64);not null"`
	// Before stores subscription data before the change in JSON format.
	Before datatypes.JSONType[*Subscription] `gorm:"column:before;type:jsonb;default:'null'"`
	// After stores subscription data after the change in JSON format.
	After datatypes.JSONType[*Subscription] `gorm:"column:after;type:jsonb;default:'null'"`
	// Extra stores additional context such as reason details and trigger source.
	Extra     datatypes.JSONMap `gorm:"column:extra;type:jsonb;default:'{}'"`
	CreatedAt time.Time
}

func (SubscriptionLog) TableName() string {
	return "subscription_log"
}
