package transaction

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDuplicateError_IsMapped(t *testing.T) {
	err := mapDuplicateErr("duplicate transaction already exists: tx-1")
	require.True(t, errors.Is(err, ErrVerifyTransactionDuplicate))
}
