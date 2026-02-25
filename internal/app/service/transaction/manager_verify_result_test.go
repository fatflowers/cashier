package transaction

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestVerifyTransactionResult_IsDowngrade(t *testing.T) {
	next := time.Unix(1735689600, 0)
	res := &VerifyTransactionResult{
		DowngradeToVipID:         "vip_low",
		DowngradeNextAutoRenewAt: &next,
	}
	require.True(t, res.IsDowngrade())
}

func TestVerifyTransactionResult_IsDowngrade_FalseWhenMissingFields(t *testing.T) {
	res := &VerifyTransactionResult{DowngradeToVipID: ""}
	require.False(t, res.IsDowngrade())
}
