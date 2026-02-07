package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"purr-chat-server/internal/models"
	"purr-chat-server/pkg/hash"

	"github.com/stretchr/testify/assert"
)

// TestSearchUsers 测试搜索用户
func TestSearchUsers(t *testing.T) {
	SetupTestDB(t)
	SetupTestRouter()
	defer CleanupTestDB(t)

	// 创建测试用户
	_, passwordHash, _ := hash.HashPasswordWithSalt("password123")
	user1 := CreateTestUser(t, "user1", "user1@example.com", passwordHash)
	_ = CreateTestUser(t, "user2", "user2@example.com", passwordHash)
	_ = CreateTestUser(t, "user3", "user3@example.com", passwordHash)

	token := GetAuthToken(t, user1.ID.String())

	tests := []struct {
		name            string
		query           string
		token           string
		expectedStatus  int
		expectedSuccess bool
	}{
		{
			name:            "成功搜索用户",
			query:           "user2",
			token:           token,
			expectedStatus:  http.StatusOK,
			expectedSuccess: true,
		},
		{
			name:            "通过邮箱搜索",
			query:           "user2@example.com",
			token:           token,
			expectedStatus:  http.StatusOK,
			expectedSuccess: true,
		},
		{
			name:            "搜索结果为空",
			query:           "nonexistent",
			token:           token,
			expectedStatus:  http.StatusOK,
			expectedSuccess: true,
		},
		{
			name:            "缺少查询参数",
			query:           "",
			token:           token,
			expectedStatus:  http.StatusBadRequest,
			expectedSuccess: false,
		},
		{
			name:            "未提供token",
			query:           "user2",
			token:           "",
			expectedStatus:  http.StatusUnauthorized,
			expectedSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/users/search?query="+url.QueryEscape(tt.query), nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			w := httptest.NewRecorder()
			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response models.AuthResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedSuccess, response.Success)
		})
	}
}

// TestGetUserByID 测试根据ID获取用户信息
func TestGetUserByID(t *testing.T) {
	SetupTestDB(t)
	SetupTestRouter()
	defer CleanupTestDB(t)

	// 创建测试用户
	_, passwordHash, _ := hash.HashPasswordWithSalt("password123")
	user1 := CreateTestUser(t, "user1", "user1@example.com", passwordHash)
	user2 := CreateTestUser(t, "user2", "user2@example.com", passwordHash)

	token := GetAuthToken(t, user1.ID.String())

	tests := []struct {
		name            string
		userID          string
		token           string
		expectedStatus  int
		expectedSuccess bool
	}{
		{
			name:            "成功获取用户信息",
			userID:          user2.ID.String(),
			token:           token,
			expectedStatus:  http.StatusOK,
			expectedSuccess: true,
		},
		{
			name:            "用户不存在",
			userID:          "00000000-0000-0000-0000-000000000000",
			token:           token,
			expectedStatus:  http.StatusNotFound,
			expectedSuccess: false,
		},
		{
			name:            "无效的用户ID格式",
			userID:          "invalid-uuid",
			token:           token,
			expectedStatus:  http.StatusBadRequest,
			expectedSuccess: false,
		},
		{
			name:            "未提供token",
			userID:          user2.ID.String(),
			token:           "",
			expectedStatus:  http.StatusUnauthorized,
			expectedSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/users/"+tt.userID, nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			w := httptest.NewRecorder()
			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response models.AuthResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedSuccess, response.Success)

			if tt.expectedSuccess {
				userData, ok := response.Data.(map[string]interface{})
				assert.True(t, ok)
				assert.Equal(t, tt.userID, userData["id"])
			}
		})
	}
}

// TestUserAuthentication 测试用户认证流程
func TestUserAuthentication(t *testing.T) {
	SetupTestDB(t)
	SetupTestRouter()
	defer CleanupTestDB(t)

	// 测试完整的认证流程
	t.Run("完整认证流程", func(t *testing.T) {
		// 1. 注册用户
		registerReq := models.RegisterRequest{
			Username: "newuser",
			Password: "password123",
			Email:    "newuser@example.com",
		}

		// 注册
		body, _ := json.Marshal(registerReq)
		req, _ := http.NewRequest("POST", "/api/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var registerResp models.AuthResponse
		err := json.Unmarshal(w.Body.Bytes(), &registerResp)
		assert.NoError(t, err)
		assert.True(t, registerResp.Success)

		// 2. 登录
		loginReq := models.LoginRequest{
			Email:    "newuser@example.com",
			Password: "password123",
		}

		body, _ = json.Marshal(loginReq)
		req, _ = http.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var loginResp models.AuthResponse
		err = json.Unmarshal(w.Body.Bytes(), &loginResp)
		assert.NoError(t, err)
		assert.True(t, loginResp.Success)

		loginData, ok := loginResp.Data.(map[string]interface{})
		assert.True(t, ok)
		token, ok := loginData["token"].(string)
		assert.True(t, ok)
		assert.NotEmpty(t, token)

		// 3. 获取用户信息
		req, _ = http.NewRequest("GET", "/api/me", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w = httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var meResp models.AuthResponse
		err = json.Unmarshal(w.Body.Bytes(), &meResp)
		assert.NoError(t, err)
		assert.True(t, meResp.Success)

		meData, ok := meResp.Data.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "newuser", meData["username"])
	})
}
