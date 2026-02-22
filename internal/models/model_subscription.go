package models

import (
	"github.com/fatflowers/cashier/pkg/types"
	"time"

	"gorm.io/datatypes"
)

// Subscription stores user subscription information.
// Use Valid() to determine whether the subscription is currently valid.
type Subscription struct {
	ID     string                   `gorm:"column:id;type:uuid;primary_key" json:"id"`
	UserID string                   `gorm:"column:user_id;type:varchar(64);not null;uniqueIndex" json:"user_id"`
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
	UpdatedAt time.Time `json:"updated_at"`
}

func (Subscription) TableName() string {
	return "subscription"
}

func (userSubscription *Subscription) Valid() bool {
	return userSubscription != nil &&
		userSubscription.Status == types.SubscriptionStatusActive &&
		userSubscription.ExpireAt != nil &&
		userSubscription.ExpireAt.After(time.Now())
}
