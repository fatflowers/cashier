package models

import (
	"github.com/fatflowers/cashier/pkg/types"
	"time"

	"gorm.io/datatypes"
)

// SubscriptionDailySnapshot is a daily user subscription snapshot for analytics.
type SubscriptionDailySnapshot struct {
	ID     string                   `gorm:"column:id;type:uuid;primary_key" json:"id"`
	Status types.SubscriptionStatus `gorm:"column:status;type:varchar(64);not null" json:"status"`
	// If IsAutoRenewable is true, NextAutoRenewAt is the next auto-renewal time; otherwise it is nil.
	NextAutoRenewAt *time.Time `gorm:"column:next_auto_renew_at;default:null" json:"next_auto_renew_at"`
	// ExpireAt is the subscription end time.
	ExpireAt *time.Time `gorm:"column:expire_at;default:null" json:"expire_at"`
	// Extra stores additional JSON data (for example: price, currency, and promotion details).
	Extra datatypes.JSON `gorm:"column:extra;type:jsonb;default:'{}'" json:"extra"`
	// CreatedAt is managed by GORM and records the creation time.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is managed by GORM and records the update time.
	UpdatedAt         time.Time `json:"updated_at"`
	UserID            string    `gorm:"column:user_id;type:varchar(64);not null;uniqueIndex:idx_user_id_snapshot_date,priority:1" json:"user_id"`
	SnapshotDate      string    `gorm:"column:snapshot_date;uniqueIndex:idx_user_id_snapshot_date,priority:2" json:"snapshot_date"`
	SnapshotCreatedAt time.Time `gorm:"column:snapshot_created_at" json:"snapshot_created_at"`
}

func (SubscriptionDailySnapshot) TableName() string {
	return "subscription_daily_snapshot"
}
