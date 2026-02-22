package models

import (
	"github.com/fatflowers/cashier/pkg/types"
	"time"

	"gorm.io/datatypes"
)

// SubscriptionDailySnapshot 用户订阅每日快照，用于统计
type SubscriptionDailySnapshot struct {
	ID     string                   `gorm:"column:id;type:uuid;primary_key" json:"id"`
	Status types.SubscriptionStatus `gorm:"column:status;type:varchar(64);not null" json:"status"`
	// 如果IsAutoRenewable为true，则NextAutoRenewAt为下次自动续费时间, 否则为null
	NextAutoRenewAt *time.Time `gorm:"column:next_auto_renew_at;default:null" json:"next_auto_renew_at"`
	// ExpireAt 订阅结束时间
	ExpireAt *time.Time `gorm:"column:expire_at;default:null" json:"expire_at"`
	// Extra 存储额外的JSON格式数据（如：价格、货币、优惠信息等）
	Extra datatypes.JSON `gorm:"column:extra;type:jsonb;default:'{}'" json:"extra"`
	// CreatedAt 记录创建时间，由 GORM 自动管理
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt 记录更新时间，由 GORM 自动管理
	UpdatedAt         time.Time `json:"updated_at"`
	UserID            string    `gorm:"column:user_id;type:varchar(64);not null;uniqueIndex:idx_user_id_snapshot_date,priority:1" json:"user_id"`
	SnapshotDate      string    `gorm:"column:snapshot_date;uniqueIndex:idx_user_id_snapshot_date,priority:2" json:"snapshot_date"`
	SnapshotCreatedAt time.Time `gorm:"column:snapshot_created_at" json:"snapshot_created_at"`
}

func (SubscriptionDailySnapshot) TableName() string {
	return "subscription_daily_snapshot"
}
