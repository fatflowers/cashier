package models

import "time"

type UserMembershipActiveItem struct {
	ID                       string     `gorm:"column:id;type:uuid;primaryKey"`
	UserTransactionID        string     `gorm:"column:user_transaction_id;type:uuid;not null;index"`
	PaymentItemID            string     `gorm:"column:payment_item_id;type:varchar(64);not null"`
	UserID                   string     `gorm:"column:user_id;type:varchar(64);not null;index:idx_user_active_time,priority:1"`
	RemainingDurationSeconds int64      `gorm:"column:remaining_duration_seconds;type:bigint;not null"`
	ActivatedAt              time.Time  `gorm:"column:activated_at;not null;index:idx_user_active_time,priority:2"`
	ExpireAt                 time.Time  `gorm:"column:expire_at;not null;index:idx_user_active_time,priority:3"`
	NextAutoRenewAt          *time.Time `gorm:"column:next_auto_renew_at"`
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

func (UserMembershipActiveItem) TableName() string {
	return "user_membership_active_item"
}
