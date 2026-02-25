package notification_handler

import (
	"context"
	"testing"

	"github.com/fatflowers/cashier/internal/platform/apple/apple_notification"
	"github.com/stretchr/testify/require"
)

func TestAppleNotificationParser_GetUserID_EmptyToken(t *testing.T) {
	p := &AppleNotificationParser{
		Notification: &apple_notification.AppStoreServerNotification{
			TransactionInfo: &apple_notification.TransactionInfo{
				AppAccountToken: "",
			},
		},
	}

	_, err := p.GetUserID(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "app account token is empty")
}
