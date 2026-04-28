package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"purr-chat-server/internal/models"
	"purr-chat-server/pkg/hash"

	"github.com/stretchr/testify/assert"
)

// TestGetConversations 测试获取会话列表
func TestGetConversations(t *testing.T) {
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
		token           string
		expectedStatus  int
		expectedSuccess bool
	}{
		{
			name:            "成功获取会话列表",
			token:           token,
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
			req, _ := http.NewRequest("GET", "/api/conversations", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			w := httptest.NewRecorder()
			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response models.APIResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedSuccess, response.Success)
		})
	}
}

// TestCreateConversation 测试创建会话
func TestCreateConversation(t *testing.T) {
	SetupTestDB(t)
	SetupTestRouter()
	defer CleanupTestDB(t)

	// 创建测试用户
	_, passwordHash, _ := hash.HashPasswordWithSalt("password123")
	user1 := CreateTestUser(t, "user1", "user1@example.com", passwordHash)
	user2 := CreateTestUser(t, "user2", "user2@example.com", passwordHash)
	user3 := CreateTestUser(t, "user3", "user3@example.com", passwordHash)

	token := GetAuthToken(t, user1.ID.String())

	tests := []struct {
		name            string
		requestBody     models.FriendRequest
		token           string
		expectedStatus  int
		expectedSuccess bool
	}{
		{
			name: "成功创建会话",
			requestBody: models.FriendRequest{
				TargetUserID: user2.ID.String(),
			},
			token:           token,
			expectedStatus:  http.StatusOK,
			expectedSuccess: true,
		},
		{
			name: "创建已存在的会话",
			requestBody: models.FriendRequest{
				TargetUserID: user2.ID.String(),
			},
			token:           token,
			expectedStatus:  http.StatusOK,
			expectedSuccess: true,
		},
		{
			name: "不能与自己创建会话",
			requestBody: models.FriendRequest{
				TargetUserID: user1.ID.String(),
			},
			token:           token,
			expectedStatus:  http.StatusBadRequest,
			expectedSuccess: false,
		},
		{
			name: "目标用户不存在",
			requestBody: models.FriendRequest{
				TargetUserID: "00000000-0000-0000-0000-000000000000",
			},
			token:           token,
			expectedStatus:  http.StatusBadRequest,
			expectedSuccess: false,
		},
		{
			name: "无效的用户ID格式",
			requestBody: models.FriendRequest{
				TargetUserID: "invalid-uuid",
			},
			token:           token,
			expectedStatus:  http.StatusBadRequest,
			expectedSuccess: false,
		},
		{
			name: "未提供token",
			requestBody: models.FriendRequest{
				TargetUserID: user3.ID.String(),
			},
			token:           "",
			expectedStatus:  http.StatusUnauthorized,
			expectedSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/conversations", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			w := httptest.NewRecorder()
			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response models.APIResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedSuccess, response.Success)
		})
	}
}

// TestConversationWorkflow 测试会话工作流
func TestConversationWorkflow(t *testing.T) {
	SetupTestDB(t)
	SetupTestRouter()
	defer CleanupTestDB(t)

	// 创建测试用户
	_, passwordHash, _ := hash.HashPasswordWithSalt("password123")
	user1 := CreateTestUser(t, "user1", "user1@example.com", passwordHash)
	user2 := CreateTestUser(t, "user2", "user2@example.com", passwordHash)

	token1 := GetAuthToken(t, user1.ID.String())
	_ = GetAuthToken(t, user2.ID.String())

	t.Run("创建会话并获取", func(t *testing.T) {
		// 1. 创建会话
		createReq := models.FriendRequest{
			TargetUserID: user2.ID.String(),
		}

		body, _ := json.Marshal(createReq)
		req, _ := http.NewRequest("POST", "/api/conversations", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token1)
		w := httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var createResp models.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &createResp)
		assert.NoError(t, err)
		assert.True(t, createResp.Success)

		convData, ok := createResp.Data.(map[string]interface{})
		assert.True(t, ok)
		conversationID, ok := convData["id"].(string)
		assert.True(t, ok)
		assert.NotEmpty(t, conversationID)

		// 2. 获取会话列表
		req, _ = http.NewRequest("GET", "/api/conversations", nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		w = httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var listResp models.APIResponse
		err = json.Unmarshal(w.Body.Bytes(), &listResp)
		assert.NoError(t, err)
		assert.True(t, listResp.Success)
		assert.NotNil(t, listResp.Data)
	})
}
