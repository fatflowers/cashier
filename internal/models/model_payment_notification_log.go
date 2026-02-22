package models

import (
	"time"

	"gorm.io/datatypes"
)

type PaymentNotificationLogStatus string

const (
	PaymentNotificationLogStatusReceived     PaymentNotificationLogStatus = "received"
	PaymentNotificationLogStatusHandled      PaymentNotificationLogStatus = "handled"
	PaymentNotificationLogStatusHandleFailed PaymentNotificationLogStatus = "handle_failed"
)

type PaymentNotificationLog struct {
	ID               string                       `gorm:"column:id;type:uuid;primary_key" json:"id"`
	ProviderID       string                       `gorm:"column:provider_id;type:varchar(64);not null" json:"provider_id"`
	UserID           *string                      `gorm:"column:user_id;type:varchar(64)" json:"user_id"`
	TraceID          string                       `gorm:"column:trace_id;type:varchar(128)" json:"trace_id"`
	TransactionID    string                       `gorm:"column:transaction_id;type:varchar(128)" json:"transaction_id"`
	NotificationTime time.Time                    `gorm:"column:notification_time" json:"notification_time"`
	Data             datatypes.JSON               `gorm:"column:data;type:jsonb" json:"data"`
	Result           *datatypes.JSON              `gorm:"column:result;type:jsonb" json:"result"`
	Status           PaymentNotificationLogStatus `gorm:"column:status;type:varchar(64);not null" json:"status"`
	CreatedAt        time.Time                    `json:"created_at"`
	UpdatedAt        time.Time                    `json:"updated_at"`
}

func (PaymentNotificationLog) TableName() string { return "payment_notification_log" }
