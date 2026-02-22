package types

import "time"

type SubscriptionStatus string

const (
	SubscriptionStatusActive   SubscriptionStatus = "active"
	SubscriptionStatusInactive SubscriptionStatus = "inactive"
)

type SubscriptionChangeReason string

const (
	UserSubscriptionChangeReasonPurchase    SubscriptionChangeReason = "purchase"
	UserSubscriptionChangeReasonRefund      SubscriptionChangeReason = "refund"
	UserSubscriptionChangeReasonCancelRenew SubscriptionChangeReason = "cancelRenew"
	UserSubscriptionChangeReasonGift        SubscriptionChangeReason = "gift"
)

type UserSubsctiptionInfo struct {
	Status          string     `json:"status"`
	NextAutoRenewAt *time.Time `json:"next_auto_renew_at"`
	ExpireAt        time.Time  `json:"expire_at"`
}
