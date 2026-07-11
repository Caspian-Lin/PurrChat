package websocket

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectDeviceType(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		expected  DeviceType
	}{
		{"Web Chrome", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/91.0", DeviceTypeWeb},
		{"Web Firefox", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0", DeviceTypeWeb},
		{"Web Safari", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 Safari/605.1.15", DeviceTypeWeb},
		{"Mobile iPhone", "Mozilla/5.0 (iPhone; CPU iPhone OS 14_6) AppleWebKit/605.1.15 Mobile/15E148", DeviceTypeMobile},
		{"Mobile Android", "Mozilla/5.0 (Linux; Android 11; SM-G991B) AppleWebKit/537.36 Mobile Safari/537.36", DeviceTypeMobile},
		{"Tablet iPad", "Mozilla/5.0 (iPad; CPU OS 14_6) AppleWebKit/605.1.15 Mobile/15E148", DeviceTypeTablet},
		{"Unknown", "CustomClient/1.0", DeviceTypeUnknown},
		{"Empty", "", DeviceTypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, detectDeviceType(tt.userAgent))
		})
	}
}

func TestDetectDeviceTypePriority(t *testing.T) {
	assert.Equal(t, DeviceTypeTablet, detectDeviceType("Mozilla/5.0 (iPad; CPU OS 14_6) Mobile/15E148"))
	assert.Equal(t, DeviceTypeMobile, detectDeviceType("Mozilla/5.0 (iPhone; CPU iPhone OS 14_6) Mobile/15E148"))
}

func TestDetectDeviceTypeCaseInsensitive(t *testing.T) {
	assert.Equal(t, DeviceTypeMobile, detectDeviceType("MOBILE"))
	assert.Equal(t, DeviceTypeTablet, detectDeviceType("IPAD"))
	assert.Equal(t, DeviceTypeWeb, detectDeviceType("MOZILLA"))
}

func makeRequest(t *testing.T, method, url string) *http.Request {
	t.Helper()
	r := httptest.NewRequest(method, url, nil)
	return r
}

func TestExtractTokenFromCookie(t *testing.T) {
	r := makeRequest(t, "GET", "/api/ws")
	r.AddCookie(&http.Cookie{Name: "purrchat_token", Value: "cookie_token"})

	token, source := extractToken(r, false)
	assert.Equal(t, "cookie_token", token)
	assert.Equal(t, "cookie", source)
}

func TestExtractTokenFromSubprotocol(t *testing.T) {
	r := makeRequest(t, "GET", "/api/ws")
	r.Header.Set("Sec-WebSocket-Protocol", "bearer,subproto_token")

	token, source := extractToken(r, false)
	assert.Equal(t, "subproto_token", token)
	assert.Equal(t, "subprotocol", source)
}

func TestExtractTokenFromQueryDisabled(t *testing.T) {
	r := makeRequest(t, "GET", "/api/ws?token=query_token")

	token, _ := extractToken(r, false)
	assert.Empty(t, token)
}

func TestExtractTokenFromQueryEnabled(t *testing.T) {
	r := makeRequest(t, "GET", "/api/ws?token=query_token")

	token, source := extractToken(r, true)
	assert.Equal(t, "query_token", token)
	assert.Equal(t, "query", source)
}

func TestExtractTokenCookiePreferredOverSubprotocol(t *testing.T) {
	r := makeRequest(t, "GET", "/api/ws")
	r.AddCookie(&http.Cookie{Name: "purrchat_token", Value: "cookie_token"})
	r.Header.Set("Sec-WebSocket-Protocol", "bearer,subproto_token")

	token, source := extractToken(r, false)
	assert.Equal(t, "cookie_token", token)
	assert.Equal(t, "cookie", source)
}

func TestExtractTokenCookiePreferredOverQuery(t *testing.T) {
	r := makeRequest(t, "GET", "/api/ws?token=query_token")
	r.AddCookie(&http.Cookie{Name: "purrchat_token", Value: "cookie_token"})

	token, source := extractToken(r, true)
	assert.Equal(t, "cookie_token", token)
	assert.Equal(t, "cookie", source)
}

func TestExtractTokenSubprotocolPreferredOverQuery(t *testing.T) {
	r := makeRequest(t, "GET", "/api/ws?token=query_token")
	r.Header.Set("Sec-WebSocket-Protocol", "bearer,subproto_token")

	token, source := extractToken(r, true)
	assert.Equal(t, "subproto_token", token)
	assert.Equal(t, "subprotocol", source)
}

func TestExtractTokenEmpty(t *testing.T) {
	r := makeRequest(t, "GET", "/api/ws")
	token, _ := extractToken(r, false)
	assert.Empty(t, token)
}

func TestCheckOriginNoAllowlist(t *testing.T) {
	hub := NewHub(HubConfig{AllowedOrigins: nil})
	r := makeRequest(t, "GET", "/api/ws")
	r.Header.Set("Origin", "https://evil.com")
	assert.True(t, hub.checkOrigin(r))
}

func TestCheckOriginEmptyOriginAllowed(t *testing.T) {
	hub := NewHub(HubConfig{AllowedOrigins: []string{"https://app.purrchat.com"}})
	r := makeRequest(t, "GET", "/api/ws")
	assert.True(t, hub.checkOrigin(r))
}

func TestCheckOriginAllowed(t *testing.T) {
	hub := NewHub(HubConfig{AllowedOrigins: []string{"https://app.purrchat.com", "https://staging.purrchat.com"}})
	r := makeRequest(t, "GET", "/api/ws")
	r.Header.Set("Origin", "https://app.purrchat.com")
	assert.True(t, hub.checkOrigin(r))
}

func TestCheckOriginRejected(t *testing.T) {
	hub := NewHub(HubConfig{AllowedOrigins: []string{"https://app.purrchat.com"}})
	r := makeRequest(t, "GET", "/api/ws")
	r.Header.Set("Origin", "https://evil.com")
	assert.False(t, hub.checkOrigin(r))
}
