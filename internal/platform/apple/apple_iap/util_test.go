package apple_iap

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserIDCodec_RoundTripNumeric(t *testing.T) {
	userID := "1234567890"

	uuid, err := UserIDToUUID(userID)
	require.NoError(t, err)

	decoded, err := UUIDToUserID(uuid)
	require.NoError(t, err)
	require.Equal(t, userID, decoded)
}

func TestUserIDCodec_RoundTripHexLeadingA(t *testing.T) {
	userID := "a1bcdef234"

	uuid, err := UserIDToUUID(userID)
	require.NoError(t, err)

	decoded, err := UUIDToUserID(uuid)
	require.NoError(t, err)
	require.Equal(t, userID, decoded)
}

func TestUUIDToUserID_RejectsUnknownScheme(t *testing.T) {
	// Random UUID-like value not produced by either known encoding scheme.
	_, err := UUIDToUserID("4b825dc6-5f3b-4f8e-b9d6-4f4f2d8c1122")
	require.Error(t, err)
}

func TestUUIDToUserID_RejectsLegacyFormat(t *testing.T) {
	// Old scheme example: left-padded with 'a' and no length prefix.
	_, err := UUIDToUserID("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa1234")
	require.Error(t, err)
}
