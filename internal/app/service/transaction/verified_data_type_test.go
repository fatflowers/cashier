package transaction

import (
	"testing"

	"github.com/awa/go-iap/appstore"
	"github.com/stretchr/testify/require"
)

func TestVerifiedData_UsesAppstoreIAPResponse(t *testing.T) {
	got := &VerifiedData{AppleReceipt: &appstore.IAPResponse{}}
	require.NotNil(t, got.AppleReceipt)
}
