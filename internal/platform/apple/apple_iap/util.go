package apple_iap

import (
	"fmt"
	"strings"
	"unicode"
)

// UserIDToUUID converts a user ID to UUID format.
func UserIDToUUID(userID string) (string, error) {
	if len(userID) > 32 {
		return "", fmt.Errorf("string too long")
	}

	// Validate that the input contains digits only.
	for _, ch := range userID {
		if !unicode.IsDigit(ch) {
			return "", fmt.Errorf("string is not pure number")
		}
	}

	// Build the base UUID hex string.
	uuidHex := userID

	// Left-pad with 'a' if the hex string is shorter than 32 chars.
	if len(uuidHex) < 32 {
		uuidHex = strings.Repeat("a", 32-len(uuidHex)) + uuidHex
	}

	// Format as RFC 4122 UUID.
	var formattedUUID strings.Builder
	formattedUUID.WriteString(uuidHex[:8])
	formattedUUID.WriteString("-")
	formattedUUID.WriteString(uuidHex[8:12])
	formattedUUID.WriteString("-")
	formattedUUID.WriteString(uuidHex[12:16])
	formattedUUID.WriteString("-")
	formattedUUID.WriteString(uuidHex[16:20])
	formattedUUID.WriteString("-")
	formattedUUID.WriteString(uuidHex[20:32])

	return formattedUUID.String(), nil
}

// UUIDToUserID converts UUID format back to user ID.
func UUIDToUserID(uuid string) (string, error) {
	// Remove UUID hyphens.
	cleanUUID := strings.ReplaceAll(uuid, "-", "")

	return strings.TrimLeft(cleanUUID, "a"), nil
}
