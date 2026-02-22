package apple_iap

import (
	"fmt"
	"strings"
	"unicode"
)

// 将用户ID转换为UUID格式
func UserIDToUUID(userID string) (string, error) {
	if len(userID) > 32 {
		return "", fmt.Errorf("string too long")
	}

	// 检查str是否是纯数字
	for _, ch := range userID {
		if !unicode.IsDigit(ch) {
			return "", fmt.Errorf("string is not pure number")
		}
	}

	// 生成UUID基础字符串
	uuidHex := userID

	// 如果十六进制字符串不够32位，用a填充
	if len(uuidHex) < 32 {
		uuidHex = strings.Repeat("a", 32-len(uuidHex)) + uuidHex
	}

	// 格式化为RFC 4122标准的UUID格式
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

// 将UUID格式转换为用户ID
func UUIDToUserID(uuid string) (string, error) {
	// 移除 UUID 中的连字符
	cleanUUID := strings.ReplaceAll(uuid, "-", "")

	return strings.TrimLeft(cleanUUID, "a"), nil
}
