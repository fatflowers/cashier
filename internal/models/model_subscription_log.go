package models

import (
	"github.com/fatflowers/cashier/pkg/types"
	"time"

	"gorm.io/datatypes"
)

// SubscriptionLog 用户订阅变更日志
// 使用场景：用于问题排查
type SubscriptionLog struct {
	ID     string `gorm:"column:id;type:uuid;primary_key" json:"id"`
	UserID string `gorm:"column:user_id;type:varchar(64);index:idx_user_id_id,priority:1;not null"`
	// Reason 变更原因
	Reason types.SubscriptionChangeReason `gorm:"column:reason;type:varchar(64);not null"`
	// Before 变更前的订阅信息，JSON格式存储
	Before datatypes.JSONType[*Subscription] `gorm:"column:before;type:jsonb;default:'null'"`
	// After 变更后的订阅信息，JSON格式存储
	After datatypes.JSONType[*Subscription] `gorm:"column:after;type:jsonb;default:'null'"`
	// Extra 额外的上下文信息，如变更原因、触发来源等
	Extra     datatypes.JSONMap `gorm:"column:extra;type:jsonb;default:'{}'"`
	CreatedAt time.Time
}

func (SubscriptionLog) TableName() string {
	return "subscription_log"
}
