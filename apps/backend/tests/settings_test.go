package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"purr-chat-server/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetSettings 测试获取用户设置
func TestGetSettings(t *testing.T) {
	SetupTestDB(t)
	SetupTestRouter()
	defer CleanupTestDB(t)

	user := CreateTestUser(t, "settings_usr", "settings@test.com", "password123")
	token := GetAuthToken(t, user.ID.String())

	tests := []struct {
		name           string
		token          string
		expectedStatus int
		expectedOk     bool
	}{
		{
			name:           "成功获取设置（新用户返回空map）",
			token:          token,
			expectedStatus: http.StatusOK,
			expectedOk:     true,
		},
		{
			name:           "未提供token",
			token:          "",
			expectedStatus: http.StatusUnauthorized,
			expectedOk:     false,
		},
		{
			name:           "无效token",
			token:          "invalid-token",
			expectedStatus: http.StatusUnauthorized,
			expectedOk:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/settings", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			w := httptest.NewRecorder()
			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response models.AuthResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedOk, response.Success)
		})
	}
}

// TestUpdateSettings 测试更新用户设置
func TestUpdateSettings(t *testing.T) {
	SetupTestDB(t)
	SetupTestRouter()
	defer CleanupTestDB(t)

	user := CreateTestUser(t, "update_usr", "update_settings@test.com", "password123")
	token := GetAuthToken(t, user.ID.String())

	tests := []struct {
		name           string
		token          string
		body           map[string]interface{}
		expectedStatus int
		expectedOk     bool
	}{
		{
			name:  "成功更新设置",
			token: token,
			body: map[string]interface{}{
				"settings": map[string]interface{}{
					"panels":        map[string]interface{}{"visiblePanels": []string{"chat", "friends"}},
					"notifications": map[string]interface{}{"soundEnabled": false},
					"general":       map[string]interface{}{"themeMode": "dark"},
				},
			},
			expectedStatus: http.StatusOK,
			expectedOk:     true,
		},
		{
			name:  "白名单过滤：非法键被剔除",
			token: token,
			body: map[string]interface{}{
				"settings": map[string]interface{}{
					"panels":        map[string]interface{}{"visiblePanels": []string{"chat"}},
					"forbidden_key": "should_be_stripped",
				},
			},
			expectedStatus: http.StatusOK,
			expectedOk:     true,
		},
		{
			name:  "空settings返回400",
			token: token,
			body: map[string]interface{}{
				"settings": nil,
			},
			expectedStatus: http.StatusBadRequest,
			expectedOk:     false,
		},
		{
			name:           "未提供token",
			token:          "",
			body:           map[string]interface{}{"settings": map[string]interface{}{"panels": map[string]interface{}{}}},
			expectedStatus: http.StatusUnauthorized,
			expectedOk:     false,
		},
		{
			name:           "无效JSON",
			token:          token,
			body:           nil,
			expectedStatus: http.StatusBadRequest,
			expectedOk:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			if tt.body != nil {
				bodyBytes, _ = json.Marshal(tt.body)
			} else {
				bodyBytes = []byte("not json")
			}

			req, _ := http.NewRequest("PUT", "/api/settings", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			w := httptest.NewRecorder()
			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response models.AuthResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedOk, response.Success)
		})
	}
}

// TestSettingsWorkflow 测试设置工作流（GET -> PUT -> GET 验证持久化）
func TestSettingsWorkflow(t *testing.T) {
	SetupTestDB(t)
	SetupTestRouter()
	defer CleanupTestDB(t)

	user := CreateTestUser(t, "workflow_usr", "workflow@test.com", "password123")
	token := GetAuthToken(t, user.ID.String())

	// Step 1: GET settings -> 新用户返回空 data
	req1, _ := http.NewRequest("GET", "/api/settings", nil)
	req1.Header.Set("Authorization", "Bearer "+token)
	w1 := httptest.NewRecorder()
	testRouter.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)
	var resp1 models.AuthResponse
	require.NoError(t, json.Unmarshal(w1.Body.Bytes(), &resp1))
	assert.True(t, resp1.Success)

	// Step 2: PUT settings -> 成功更新
	newSettings := map[string]interface{}{
		"settings": map[string]interface{}{
			"general": map[string]interface{}{
				"themeMode":  "dark",
				"themeColor": "sage",
				"language":   "zh-CN",
				"fontSize":   "large",
			},
		},
	}
	bodyBytes, _ := json.Marshal(newSettings)
	req2, _ := http.NewRequest("PUT", "/api/settings", bytes.NewBuffer(bodyBytes))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	testRouter.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)

	// Step 3: GET settings -> 验证持久化
	req3, _ := http.NewRequest("GET", "/api/settings", nil)
	req3.Header.Set("Authorization", "Bearer "+token)
	w3 := httptest.NewRecorder()
	testRouter.ServeHTTP(w3, req3)

	assert.Equal(t, http.StatusOK, w3.Code)
	var resp3 struct {
		Success bool                   `json:"success"`
		Data    map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w3.Body.Bytes(), &resp3))
	assert.True(t, resp3.Success)

	general := resp3.Data["general"].(map[string]interface{})
	assert.Equal(t, "dark", general["themeMode"])
	assert.Equal(t, "large", general["fontSize"])

	// Step 4: PUT 部分更新 -> 验证合并
	partialSettings := map[string]interface{}{
		"settings": map[string]interface{}{
			"notifications": map[string]interface{}{
				"soundEnabled": false,
			},
		},
	}
	bodyBytes2, _ := json.Marshal(partialSettings)
	req4, _ := http.NewRequest("PUT", "/api/settings", bytes.NewBuffer(bodyBytes2))
	req4.Header.Set("Content-Type", "application/json")
	req4.Header.Set("Authorization", "Bearer "+token)
	w4 := httptest.NewRecorder()
	testRouter.ServeHTTP(w4, req4)

	assert.Equal(t, http.StatusOK, w4.Code)

	// 验证 notifications 存在
	req5, _ := http.NewRequest("GET", "/api/settings", nil)
	req5.Header.Set("Authorization", "Bearer "+token)
	w5 := httptest.NewRecorder()
	testRouter.ServeHTTP(w5, req5)

	var resp5 struct {
		Success bool                   `json:"success"`
		Data    map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w5.Body.Bytes(), &resp5))
	assert.NotNil(t, resp5.Data["notifications"])
}
