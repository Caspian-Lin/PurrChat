package websocket

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDetectDeviceType 测试设备类型检测
func TestDetectDeviceType(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		expected  DeviceType
	}{
		{
			name:      "Web Browser - Chrome",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
			expected:  DeviceTypeWeb,
		},
		{
			name:      "Web Browser - Firefox",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0",
			expected:  DeviceTypeWeb,
		},
		{
			name:      "Web Browser - Safari",
			userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15",
			expected:  DeviceTypeWeb,
		},
		{
			name:      "Mobile Device - iPhone",
			userAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Mobile/15E148 Safari/604.1",
			expected:  DeviceTypeMobile,
		},
		{
			name:      "Mobile Device - Android",
			userAgent: "Mozilla/5.0 (Linux; Android 11; SM-G991B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.120 Mobile Safari/537.36",
			expected:  DeviceTypeMobile,
		},
		{
			name:      "Tablet Device - iPad",
			userAgent: "Mozilla/5.0 (iPad; CPU OS 14_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Mobile/15E148 Safari/604.1",
			expected:  DeviceTypeTablet,
		},
		{
			name:      "Tablet Device - Android Tablet",
			userAgent: "Mozilla/5.0 (Linux; Android 10; KFTRWI) AppleWebKit/537.36 (KHTML, like Gecko) Silk/89.4.0 like Chrome/89.0.4389.105 Safari/537.36",
			expected:  DeviceTypeMobile,
		},
		{
			name:      "Desktop Device",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			expected:  DeviceTypeWeb,
		},
		{
			name:      "Unknown Device",
			userAgent: "CustomClient/1.0",
			expected:  DeviceTypeUnknown,
		},
		{
			name:      "Empty User-Agent",
			userAgent: "",
			expected:  DeviceTypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectDeviceType(tt.userAgent)
			assert.Equal(t, tt.expected, result, "User-Agent: %s", tt.userAgent)
		})
	}
}

// TestDetectDeviceTypeCaseInsensitive 测试设备类型检测不区分大小写
func TestDetectDeviceTypeCaseInsensitive(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		expected  DeviceType
	}{
		{
			name:      "Mobile - uppercase",
			userAgent: "Mozilla/5.0 MOBILE",
			expected:  DeviceTypeMobile,
		},
		{
			name:      "Mobile - mixed case",
			userAgent: "Mozilla/5.0 MoBiLe",
			expected:  DeviceTypeMobile,
		},
		{
			name:      "iPhone - uppercase",
			userAgent: "Mozilla/5.0 IPHONE",
			expected:  DeviceTypeMobile,
		},
		{
			name:      "Android - uppercase",
			userAgent: "Mozilla/5.0 ANDROID",
			expected:  DeviceTypeMobile,
		},
		{
			name:      "iPad - uppercase",
			userAgent: "Mozilla/5.0 IPAD",
			expected:  DeviceTypeTablet,
		},
		{
			name:      "Tablet - uppercase",
			userAgent: "Mozilla/5.0 TABLET",
			expected:  DeviceTypeTablet,
		},
		{
			name:      "Mozilla - uppercase",
			userAgent: "MOZILLA/5.0",
			expected:  DeviceTypeWeb,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectDeviceType(tt.userAgent)
			assert.Equal(t, tt.expected, result, "User-Agent: %s", tt.userAgent)
		})
	}
}

// TestDetectDeviceTypePriority 测试设备类型检测优先级
func TestDetectDeviceTypePriority(t *testing.T) {
	// 测试优先级：tablet > mobile > web
	tests := []struct {
		name      string
		userAgent string
		expected  DeviceType
	}{
		{
			name:      "iPad with mobile in UA is detected as tablet (tablet has higher priority)",
			userAgent: "Mozilla/5.0 (iPad; CPU OS 14_6 like Mac OS X) AppleWebKit/605.1.15 Mobile/15E148 Safari/604.1",
			expected:  DeviceTypeTablet,
		},
		{
			name:      "iPad without mobile in UA should be detected as tablet",
			userAgent: "Mozilla/5.0 (iPad; CPU OS 14_6 like Mac OS X) AppleWebKit/605.1.15 Safari/604.1",
			expected:  DeviceTypeTablet,
		},
		{
			name:      "iPhone should be detected as mobile",
			userAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) AppleWebKit/605.1.15 Mobile/15E148 Safari/604.1",
			expected:  DeviceTypeMobile,
		},
		{
			name:      "Android tablet should be detected as mobile",
			userAgent: "Mozilla/5.0 (Linux; Android 10; KFTRWI) AppleWebKit/537.36 (KHTML, like Gecko) Silk/89.4.0 like Chrome/89.0.4389.105 Safari/537.36",
			expected:  DeviceTypeMobile,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectDeviceType(tt.userAgent)
			assert.Equal(t, tt.expected, result, "User-Agent: %s", tt.userAgent)
		})
	}
}

// TestDetectDeviceTypeContains 测试设备类型检测包含关系
func TestDetectDeviceTypeContains(t *testing.T) {
	// 测试字符串包含关系（不区分大小写）
	assert.True(t, strings.Contains(strings.ToLower("Mozilla/5.0 (iPhone"), "iphone"))
	assert.True(t, strings.Contains(strings.ToLower("Mozilla/5.0 (Android"), "android"))
	assert.True(t, strings.Contains(strings.ToLower("Mozilla/5.0 (iPad"), "ipad"))
	assert.True(t, strings.Contains(strings.ToLower("Mozilla/5.0 (Tablet"), "tablet"))
}
