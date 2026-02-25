package subscription

import (
	"context"
	"testing"

	models "github.com/fatflowers/cashier/internal/models"
	"github.com/fatflowers/cashier/pkg/config"
	types "github.com/fatflowers/cashier/pkg/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/datatypes"
)

func TestGetChangeReason_Upgrade(t *testing.T) {
	beforeID := "tx-before"
	item := &models.Transaction{
		PaymentItemID:               "payment1",
		BeforeUpgradedTransactionID: &beforeID,
		Extra: datatypes.NewJSONType(&models.UserSubscriptionItemExtra{
			PaymentItemSnapshot: &types.PaymentItem{
				ID:   "payment1",
				Type: types.PaymentItemTypeAutoRenewableSubscription,
			},
		}),
	}

	svc := NewService(&config.Config{}, nil, zap.NewNop().Sugar())
	reason, err := svc.getChangeReason(context.Background(), item)
	require.NoError(t, err)
	require.Equal(t, types.UserSubscriptionChangeReasonUpgrade, reason)
}
