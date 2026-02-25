package transaction

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrVerifyTransactionDuplicate_IsWrapFriendly(t *testing.T) {
	err := fmt.Errorf("wrapped: %w", ErrVerifyTransactionDuplicate)
	require.True(t, errors.Is(err, ErrVerifyTransactionDuplicate))
}
