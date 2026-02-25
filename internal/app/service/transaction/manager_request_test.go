package transaction

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransactionVerifyRequest_JSONIncludesServerVerificationData(t *testing.T) {
	req := TransactionVerifyRequest{
		ProviderID:             "apple",
		TransactionID:          "tx-1",
		ServerVerificationData: "receipt-data",
	}

	body, err := json.Marshal(req)
	require.NoError(t, err)
	require.Contains(t, string(body), "\"server_verification_data\":\"receipt-data\"")
}
