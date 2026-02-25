package transaction

import (
	"testing"

	"github.com/awa/go-iap/appstore"
	"github.com/awa/go-iap/appstore/api"
	"github.com/stretchr/testify/require"
)

func TestDetectAppleUpgrade_Success(t *testing.T) {
	parseResult := &VerifiedData{AppleReceipt: &appstore.IAPResponse{
		LatestReceiptInfo: []appstore.InApp{
			{TransactionID: "tx-current"},
			{TransactionID: "tx-before", IsUpgraded: "true"},
		},
	}}

	beforeID, ok := detectAppleUpgrade(parseResult, &api.JWSTransaction{TransactionID: "tx-current"})
	require.True(t, ok)
	require.Equal(t, "tx-before", beforeID)
}

func TestDetectAppleUpgrade_FallbackToInApp(t *testing.T) {
	parseResult := &VerifiedData{AppleReceipt: &appstore.IAPResponse{
		Receipt: appstore.Receipt{
			InApp: []appstore.InApp{
				{TransactionID: "tx-current"},
				{TransactionID: "tx-before", IsUpgraded: "true"},
			},
		},
	}}

	beforeID, ok := detectAppleUpgrade(parseResult, &api.JWSTransaction{TransactionID: "tx-current"})
	require.True(t, ok)
	require.Equal(t, "tx-before", beforeID)
}

func TestDetectAppleUpgrade_NoUpgrade(t *testing.T) {
	parseResult := &VerifiedData{AppleReceipt: &appstore.IAPResponse{
		LatestReceiptInfo: []appstore.InApp{
			{TransactionID: "tx-current"},
			{TransactionID: "tx-before", IsUpgraded: "false"},
		},
	}}

	beforeID, ok := detectAppleUpgrade(parseResult, &api.JWSTransaction{TransactionID: "tx-current"})
	require.False(t, ok)
	require.Empty(t, beforeID)
}

func TestDetectAppleUpgrade_NilSafety(t *testing.T) {
	beforeID, ok := detectAppleUpgrade(nil, nil)
	require.False(t, ok)
	require.Empty(t, beforeID)
}
