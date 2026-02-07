package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"
	"purr-chat-server/pkg/hash"

	"github.com/stretchr/testify/assert"
)

// TestRegister 测试用户注册
func TestRegister(t *testing.T) {
	SetupTestDB(t)
	SetupTestRouter()
	defer CleanupTestDB(t)

	tests := []struct {
		name            string
		requestBody     models.RegisterRequest
		expectedStatus  int
		expectedSuccess bool
	}{
		{
			name: "成功注册用户",
			requestBody: models.RegisterRequest{
				Username: "testuser",
				Password: "password123",
				Email:    "test@example.com",
				Phone:    "13800138000",
			},
			expectedStatus:  http.StatusOK,
			expectedSuccess: true,
		},
		{
			name: "用户名已存在",
			requestBody: models.RegisterRequest{
				Username: "testuser",
				Password: "password456",
				Email:    "test2@example.com",
			},
			expectedStatus:  http.StatusBadRequest,
			expectedSuccess: false,
		},
		{
			name: "邮箱已存在",
			requestBody: models.RegisterRequest{
				Username: "testuser2",
				Password: "password123",
				Email:    "test@example.com",
			},
			expectedStatus:  http.StatusBadRequest,
			expectedSuccess: false,
		},
		{
			name: "用户名太短",
			requestBody: models.RegisterRequest{
				Username: "ab",
				Password: "password123",
			},
			expectedStatus:  http.StatusBadRequest,
			expectedSuccess: false,
		},
		{
			name: "密码太短",
			requestBody: models.RegisterRequest{
				Username: "testuser3",
				Password: "12345",
			},
			expectedStatus:  http.StatusBadRequest,
			expectedSuccess: false,
		},
		{
			name: "邮箱格式错误",
			requestBody: models.RegisterRequest{
				Username: "testuser4",
				Password: "password123",
				Email:    "invalid-email",
			},
			expectedStatus:  http.StatusBadRequest,
			expectedSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

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

// TestLogin 测试用户登录
func TestLogin(t *testing.T) {
	SetupTestDB(t)
	SetupTestRouter()
	defer CleanupTestDB(t)

	// 创建测试用户
	salt, passwordHash, _ := hash.HashPasswordWithSalt("password123")
	user := &models.User{
		Username:     "testuser",
		PasswordHash: passwordHash,
		Salt:         salt,
		Email:        "test@example.com",
	}
	ctx := context.Background()
	userRepo := repository.NewUserRepository()
	err := userRepo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	tests := []struct {
		name            string
		requestBody     models.LoginRequest
		expectedStatus  int
		expectedSuccess bool
	}{
		{
			name: "成功登录",
			requestBody: models.LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			expectedStatus:  http.StatusOK,
			expectedSuccess: true,
		},
		{
			name: "用户名不存在",
			requestBody: models.LoginRequest{
				Email:    "nonexistent@example.com",
				Password: "password123",
			},
			expectedStatus:  http.StatusUnauthorized,
			expectedSuccess: false,
		},
		{
			name: "密码错误",
			requestBody: models.LoginRequest{
				Email:    "test@example.com",
				Password: "wrongpassword",
			},
			expectedStatus:  http.StatusUnauthorized,
			expectedSuccess: false,
		},
		{
			name: "缺少邮箱",
			requestBody: models.LoginRequest{
				Password: "password123",
			},
			expectedStatus:  http.StatusBadRequest,
			expectedSuccess: false,
		},
		{
			name: "缺少密码",
			requestBody: models.LoginRequest{
				Email: "test@example.com",
			},
			expectedStatus:  http.StatusBadRequest,
			expectedSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response models.AuthResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedSuccess, response.Success)

			if tt.expectedSuccess {
				loginResp, ok := response.Data.(map[string]interface{})
				assert.True(t, ok)
				assert.NotEmpty(t, loginResp["token"])
			}
		})
	}
}

// TestMe 测试获取当前用户信息
func TestMe(t *testing.T) {
	SetupTestDB(t)
	SetupTestRouter()
	defer CleanupTestDB(t)

	// 创建测试用户
	salt, passwordHash, _ := hash.HashPasswordWithSalt("password123")
	user := &models.User{
		Username:     "testuser",
		PasswordHash: passwordHash,
		Salt:         salt,
		Email:        "test@example.com",
	}
	createdUser := CreateTestUser(t, user.Username, user.Email, user.PasswordHash)

	tests := []struct {
		name            string
		token           string
		expectedStatus  int
		expectedSuccess bool
	}{
		{
			name:            "成功获取用户信息",
			token:           GetAuthToken(t, createdUser.ID.String()),
			expectedStatus:  http.StatusOK,
			expectedSuccess: true,
		},
		{
			name:            "未提供token",
			token:           "",
			expectedStatus:  http.StatusUnauthorized,
			expectedSuccess: false,
		},
		{
			name:            "无效的token",
			token:           "invalid_token",
			expectedStatus:  http.StatusUnauthorized,
			expectedSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/me", nil)
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
				assert.Equal(t, createdUser.ID.String(), userData["id"])
				assert.Equal(t, createdUser.Username, userData["username"])
			}
		})
	}
}

// TestUpdateProfile 测试更新个人资料
func TestUpdateProfile(t *testing.T) {
	SetupTestDB(t)
	SetupTestRouter()
	defer CleanupTestDB(t)

	// 创建测试用户
	salt, passwordHash, _ := hash.HashPasswordWithSalt("password123")
	user := &models.User{
		Username:     "testuser",
		PasswordHash: passwordHash,
		Salt:         salt,
		Email:        "test@example.com",
	}
	createdUser := CreateTestUser(t, user.Username, user.Email, user.PasswordHash)
	token := GetAuthToken(t, createdUser.ID.String())

	tests := []struct {
		name            string
		requestBody     models.UpdateProfileRequest
		token           string
		expectedStatus  int
		expectedSuccess bool
	}{
		{
			name: "成功更新邮箱",
			requestBody: models.UpdateProfileRequest{
				Email: "newemail@example.com",
			},
			token:           token,
			expectedStatus:  http.StatusOK,
			expectedSuccess: true,
		},
		{
			name: "成功更新手机号",
			requestBody: models.UpdateProfileRequest{
				Phone: "13900139000",
			},
			token:           token,
			expectedStatus:  http.StatusOK,
			expectedSuccess: true,
		},
		{
			name: "邮箱格式错误",
			requestBody: models.UpdateProfileRequest{
				Email: "invalid-email",
			},
			token:           token,
			expectedStatus:  http.StatusBadRequest,
			expectedSuccess: false,
		},
		{
			name: "未提供token",
			requestBody: models.UpdateProfileRequest{
				Email: "newemail@example.com",
			},
			token:           "",
			expectedStatus:  http.StatusUnauthorized,
			expectedSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("PUT", "/api/profile", bytes.NewBuffer(body))
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
			assert.Equal(t, tt.expectedSuccess, response.Success)
		})
	}
}
