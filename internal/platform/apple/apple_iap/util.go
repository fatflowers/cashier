package apple_iap

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

const (
	uuidHexLen      = 32
	maxUserIDHexLen = 30
	padChar         = "a"
)

// UserIDToUUID converts a user ID to UUID format.
func UserIDToUUID(userID string) (string, error) {
	if userID == "" {
		return "", fmt.Errorf("user id is empty")
	}

	// Use a reversible length-prefixed hex scheme for all user IDs.
	// format: [2-hex len][hex userID][padding to 32 with 'a']
	normalized := strings.ToLower(userID)
	if !isHex(normalized) {
		return "", fmt.Errorf("string is not valid hex")
	}
	if len(normalized) > maxUserIDHexLen {
		return "", fmt.Errorf("hex string too long: max length is %d", maxUserIDHexLen)
	}

	prefix := fmt.Sprintf("%02x", len(normalized))
	uuidHex := prefix + normalized
	if len(uuidHex) < uuidHexLen {
		uuidHex += strings.Repeat(padChar, uuidHexLen-len(uuidHex))
	}

	return formatUUID(uuidHex)
}

func formatUUID(uuidHex string) (string, error) {
	if len(uuidHex) != uuidHexLen {
		return "", fmt.Errorf("invalid uuid hex length: %d", len(uuidHex))
	}
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
	cleanUUID := strings.ToLower(strings.ReplaceAll(uuid, "-", ""))
	if len(cleanUUID) != uuidHexLen || !isHex(cleanUUID) {
		return "", fmt.Errorf("invalid uuid format")
	}

	// Try to decode length-prefixed hex scheme first.
	if n, err := strconv.ParseUint(cleanUUID[:2], 16, 8); err == nil {
		size := int(n)
		if size > 0 && size <= maxUserIDHexLen {
			end := 2 + size
			payload := cleanUUID[2:end]
			padding := cleanUUID[end:]
			if isHex(payload) && strings.Trim(padding, padChar) == "" {
				return payload, nil
			}
		}
	}

	return "", fmt.Errorf("uuid is not encoded by known user id scheme")
}

func isHex(s string) bool {
	if s == "" {
		return false
	}
	for _, ch := range s {
		if !(unicode.IsDigit(ch) || ('a' <= ch && ch <= 'f') || ('A' <= ch && ch <= 'F')) {
			return false
		}
	}
	return true
}
