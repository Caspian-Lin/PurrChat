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

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestGetMessages 测试获取消息列表
func TestGetMessages(t *testing.T) {
	SetupTestDB(t)
	SetupTestRouter()
	defer CleanupTestDB(t)

	// 创建测试用户
	_, passwordHash, _ := hash.HashPasswordWithSalt("password123")
	user1 := CreateTestUser(t, "user1", "user1@example.com", passwordHash)
	user2 := CreateTestUser(t, "user2", "user2@example.com", passwordHash)

	token := GetAuthToken(t, user1.ID.String())

	// 创建会话
	createReq := models.FriendRequest{
		TargetUserID: user2.ID.String(),
	}
	body, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/api/conversations", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	var createResp models.MessageResponse
	if err := json.Unmarshal(w.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	convData := createResp.Data.(map[string]interface{})
	conversationID := convData["id"].(string)

	tests := []struct {
		name            string
		conversationID  string
		limit           int
		offset          int
		token           string
		expectedStatus  int
		expectedSuccess bool
	}{
		{
			name:            "成功获取消息列表",
			conversationID:  conversationID,
			limit:           50,
			offset:          0,
			token:           token,
			expectedStatus:  http.StatusOK,
			expectedSuccess: true,
		},
		{
			name:            "缺少conversation_id",
			conversationID:  "",
			limit:           50,
			offset:          0,
			token:           token,
			expectedStatus:  http.StatusBadRequest,
			expectedSuccess: false,
		},
		{
			name:            "无效的conversation_id",
			conversationID:  "invalid-uuid",
			limit:           50,
			offset:          0,
			token:           token,
			expectedStatus:  http.StatusBadRequest,
			expectedSuccess: false,
		},
		{
			name:            "未提供token",
			conversationID:  conversationID,
			limit:           50,
			offset:          0,
			token:           "",
			expectedStatus:  http.StatusUnauthorized,
			expectedSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := url.Values{}
			if tt.conversationID != "" {
				query.Add("conversation_id", tt.conversationID)
			}
			if tt.limit > 0 {
				query.Add("limit", string(rune(tt.limit)))
			}
			if tt.offset > 0 {
				query.Add("offset", string(rune(tt.offset)))
			}

			req, _ := http.NewRequest("GET", "/api/messages?"+query.Encode(), nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			w := httptest.NewRecorder()
			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response models.MessagesResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedSuccess, response.Success)
		})
	}
}

// TestSendMessage 测试发送消息
func TestSendMessage(t *testing.T) {
	SetupTestDB(t)
	SetupTestRouter()
	defer CleanupTestDB(t)

	// 创建测试用户
	_, passwordHash, _ := hash.HashPasswordWithSalt("password123")
	user1 := CreateTestUser(t, "user1", "user1@example.com", passwordHash)
	user2 := CreateTestUser(t, "user2", "user2@example.com", passwordHash)

	token := GetAuthToken(t, user1.ID.String())

	// 创建会话
	createReq := models.FriendRequest{
		TargetUserID: user2.ID.String(),
	}
	body, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/api/conversations", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	var createResp models.MessageResponse
	if err := json.Unmarshal(w.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	convData := createResp.Data.(map[string]interface{})
	conversationID, _ := uuid.Parse(convData["id"].(string))

	tests := []struct {
		name            string
		requestBody     models.SendMessageRequest
		token           string
		expectedStatus  int
		expectedSuccess bool
	}{
		{
			name: "成功发送文本消息",
			requestBody: models.SendMessageRequest{
				ConversationID: conversationID,
				Content:        "Hello, world!",
				MsgType:        "text",
			},
			token:           token,
			expectedStatus:  http.StatusOK,
			expectedSuccess: true,
		},
		{
			name: "成功发送图片消息",
			requestBody: models.SendMessageRequest{
				ConversationID: conversationID,
				Content:        "https://example.com/image.jpg",
				MsgType:        "image",
			},
			token:           token,
			expectedStatus:  http.StatusOK,
			expectedSuccess: true,
		},
		{
			name: "会话不存在",
			requestBody: models.SendMessageRequest{
				ConversationID: uuid.New(),
				Content:        "Test message",
				MsgType:        "text",
			},
			token:           token,
			expectedStatus:  http.StatusBadRequest,
			expectedSuccess: false,
		},
		{
			name: "内容为空",
			requestBody: models.SendMessageRequest{
				ConversationID: conversationID,
				Content:        "",
				MsgType:        "text",
			},
			token:           token,
			expectedStatus:  http.StatusBadRequest,
			expectedSuccess: false,
		},
		{
			name: "未提供token",
			requestBody: models.SendMessageRequest{
				ConversationID: conversationID,
				Content:        "Test message",
				MsgType:        "text",
			},
			token:           "",
			expectedStatus:  http.StatusUnauthorized,
			expectedSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/messages", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			w := httptest.NewRecorder()
			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response models.MessageResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedSuccess, response.Success)
		})
	}
}

// TestSendFriendRequest 测试发送好友请求
func TestSendFriendRequest(t *testing.T) {
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
			name: "成功发送好友请求",
			requestBody: models.FriendRequest{
				TargetUserID: user2.ID.String(),
			},
			token:           token,
			expectedStatus:  http.StatusOK,
			expectedSuccess: true,
		},
		{
			name: "不能向自己发送好友请求",
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
			req, _ := http.NewRequest("POST", "/api/friends/request", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			w := httptest.NewRecorder()
			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response models.FriendRequestResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedSuccess, response.Success)
		})
	}
}

// TestHandleFriendRequest 测试处理好友请求
func TestHandleFriendRequest(t *testing.T) {
	SetupTestDB(t)
	SetupTestRouter()
	defer CleanupTestDB(t)

	// 创建测试用户
	_, passwordHash, _ := hash.HashPasswordWithSalt("password123")
	user1 := CreateTestUser(t, "user1", "user1@example.com", passwordHash)
	user2 := CreateTestUser(t, "user2", "user2@example.com", passwordHash)

	token1 := GetAuthToken(t, user1.ID.String())
	token2 := GetAuthToken(t, user2.ID.String())

	// 发送好友请求
	sendReq := models.FriendRequest{
		TargetUserID: user2.ID.String(),
	}
	body, _ := json.Marshal(sendReq)
	req, _ := http.NewRequest("POST", "/api/friends/request", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token1)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	var sendResp models.FriendRequestResponse
	if err := json.Unmarshal(w.Body.Bytes(), &sendResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	convData := sendResp.Data.(map[string]interface{})
	conversationID, _ := uuid.Parse(convData["id"].(string))

	tests := []struct {
		name            string
		requestBody     models.HandleFriendRequestRequest
		token           string
		expectedStatus  int
		expectedSuccess bool
	}{
		{
			name: "成功接受好友请求",
			requestBody: models.HandleFriendRequestRequest{
				ConversationID: conversationID,
				Action:         "accept",
			},
			token:           token2,
			expectedStatus:  http.StatusOK,
			expectedSuccess: true,
		},
		{
			name: "成功拒绝好友请求",
			requestBody: models.HandleFriendRequestRequest{
				ConversationID: conversationID,
				Action:         "reject",
			},
			token:           token2,
			expectedStatus:  http.StatusOK,
			expectedSuccess: true,
		},
		{
			name: "无效的操作",
			requestBody: models.HandleFriendRequestRequest{
				ConversationID: conversationID,
				Action:         "invalid",
			},
			token:           token2,
			expectedStatus:  http.StatusBadRequest,
			expectedSuccess: false,
		},
		{
			name: "会话不存在",
			requestBody: models.HandleFriendRequestRequest{
				ConversationID: uuid.New(),
				Action:         "accept",
			},
			token:           token2,
			expectedStatus:  http.StatusBadRequest,
			expectedSuccess: false,
		},
		{
			name: "未提供token",
			requestBody: models.HandleFriendRequestRequest{
				ConversationID: conversationID,
				Action:         "accept",
			},
			token:           "",
			expectedStatus:  http.StatusUnauthorized,
			expectedSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/friends/handle", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			w := httptest.NewRecorder()
			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response models.HandleFriendRequestResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedSuccess, response.Success)
		})
	}
}

// TestGetFriends 测试获取好友列表
func TestGetFriends(t *testing.T) {
	SetupTestDB(t)
	SetupTestRouter()
	defer CleanupTestDB(t)

	// 创建测试用户
	_, passwordHash, _ := hash.HashPasswordWithSalt("password123")
	user1 := CreateTestUser(t, "user1", "user1@example.com", passwordHash)
	_ = CreateTestUser(t, "user2", "user2@example.com", passwordHash)

	token := GetAuthToken(t, user1.ID.String())

	tests := []struct {
		name            string
		token           string
		expectedStatus  int
		expectedSuccess bool
	}{
		{
			name:            "成功获取好友列表",
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
			req, _ := http.NewRequest("GET", "/api/friends", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			w := httptest.NewRecorder()
			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response models.FriendListResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedSuccess, response.Success)
		})
	}
}

// TestFriendWorkflow 测试完整的好友工作流
func TestFriendWorkflow(t *testing.T) {
	SetupTestDB(t)
	SetupTestRouter()
	defer CleanupTestDB(t)

	// 创建测试用户
	_, passwordHash, _ := hash.HashPasswordWithSalt("password123")
	user1 := CreateTestUser(t, "user1", "user1@example.com", passwordHash)
	user2 := CreateTestUser(t, "user2", "user2@example.com", passwordHash)

	token1 := GetAuthToken(t, user1.ID.String())
	token2 := GetAuthToken(t, user2.ID.String())

	t.Run("完整好友工作流", func(t *testing.T) {
		// 1. 发送好友请求
		sendReq := models.FriendRequest{
			TargetUserID: user2.ID.String(),
		}

		body, _ := json.Marshal(sendReq)
		req, _ := http.NewRequest("POST", "/api/friends/request", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token1)
		w := httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var sendResp models.FriendRequestResponse
		err := json.Unmarshal(w.Body.Bytes(), &sendResp)
		assert.NoError(t, err)
		assert.True(t, sendResp.Success)

		convData := sendResp.Data.(map[string]interface{})
		conversationID, _ := uuid.Parse(convData["id"].(string))

		// 2. 接受好友请求
		handleReq := models.HandleFriendRequestRequest{
			ConversationID: conversationID,
			Action:         "accept",
		}

		body, _ = json.Marshal(handleReq)
		req, _ = http.NewRequest("POST", "/api/friends/handle", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token2)
		w = httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var handleResp models.HandleFriendRequestResponse
		err = json.Unmarshal(w.Body.Bytes(), &handleResp)
		assert.NoError(t, err)
		assert.True(t, handleResp.Success)

		// 3. 获取好友列表
		req, _ = http.NewRequest("GET", "/api/friends", nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		w = httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var friendsResp models.FriendListResponse
		err = json.Unmarshal(w.Body.Bytes(), &friendsResp)
		assert.NoError(t, err)
		assert.True(t, friendsResp.Success)
		assert.GreaterOrEqual(t, len(friendsResp.Data), 1)
	})
}
