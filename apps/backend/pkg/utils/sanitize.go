package utils

import (
	"html"
	"strings"
)

// MaskEmail 对邮箱进行脱敏处理
// 示例: "user@example.com" → "u***@example.com"
func MaskEmail(email string) string {
	if email == "" {
		return ""
	}

	at := strings.LastIndex(email, "@")
	if at < 0 {
		return "***"
	}

	local := email[:at]
	domain := email[at+1:]

	runes := []rune(local)
	switch len(runes) {
	case 0:
		return "***@" + domain
	case 1:
		return string(runes[0]) + "***@" + domain
	default:
		masked := string(runes[0]) + strings.Repeat("*", min(len(runes)-1, 3))
		return masked + "@" + domain
	}
}

// MaskPhone 对手机号进行脱敏处理
// 示例: "13812345678" → "138****5678"
func MaskPhone(phone string) string {
	if phone == "" {
		return ""
	}

	runes := []rune(phone)
	length := len(runes)

	switch {
	case length <= 3:
		return "***"
	case length <= 7:
		return string(runes[:3]) + "***"
	default:
		return string(runes[:3]) + strings.Repeat("*", length-7) + string(runes[length-4:])
	}
}

// EscapeHTML 对文本进行 HTML 实体转义（防御存储型 XSS）
// 仅用于 text 类型消息内容，image/file/system 类型不应调用
func EscapeHTML(text string) string {
	return html.EscapeString(text)
}
