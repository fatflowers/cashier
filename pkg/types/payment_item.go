package types

type PaymentProvider string

const (
	PaymentProviderApple  PaymentProvider = "apple"
	PaymentProviderGoogle PaymentProvider = "google"
	PaymentProviderInner  PaymentProvider = "inner"
)

type PaymentItemType string

const (
	PaymentItemTypeAutoRenewableSubscription PaymentItemType = "auto_renewable_subscription"
	PaymentItemTypeNonRenewableSubscription  PaymentItemType = "non_renewable_subscription"
)

type PaymentItem struct {
	ID             string          `json:"id" mapstructure:"id"`
	ProviderID     PaymentProvider `json:"provider_id" mapstructure:"provider_id"`
	ProviderItemID string          `json:"provider_item_id" mapstructure:"provider_item_id"`
	Type           PaymentItemType `json:"type" mapstructure:"type"`
	// 时长类商品对应的时长，如果非时长类商品，则DurationHour为nil
	DurationHour *int64 `json:"duration_hour" mapstructure:"duration_hour"`
}

func (item *PaymentItem) IsSubscription() bool {
	return item.Type == PaymentItemTypeAutoRenewableSubscription || item.Type == PaymentItemTypeNonRenewableSubscription
}

func (item *PaymentItem) Renewable() bool {
	return item.Type == PaymentItemTypeAutoRenewableSubscription
}
