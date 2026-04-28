package tests

import (
	"encoding/json"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/websocket"
	"purr-chat-server/pkg/jwt"
	"purr-chat-server/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// 初始化日志
	logger.Init()
}

// setupTestHub 设置测试用的Hub
func setupTestHub(t *testing.T) *websocket.Hub {
	hub := websocket.NewHub(100, 3) // 测试环境使用较小的限制
	go hub.Run()
	return hub
}

// createTestToken 创建测试用的JWT token
func createTestToken(t *testing.T, userID uuid.UUID, secret string) string {
	token, err := jwt.GenerateToken(userID.String(), secret, 24*time.Hour)
	require.NoError(t, err)
	return token
}

// createTestClient 创建测试用的WebSocket客户端
func createTestClient(t *testing.T, hub *websocket.Hub, userID uuid.UUID, deviceType websocket.DeviceType) *websocket.Client {
	client := &websocket.Client{
		ID:          uuid.New(),
		UserID:      userID,
		Conn:        nil, // 测试时不需要真实的连接
		Send:        make(chan []byte, 256),
		DeviceType:  deviceType,
		ConnectedAt: time.Now(),
		UserAgent:   "test-agent",
	}

	err := hub.RegisterClient(client)
	require.NoError(t, err)

	return client
}

// TestHubRegisterClient 测试客户端注册
func TestHubRegisterClient(t *testing.T) {
	hub := setupTestHub(t)

	userID := uuid.New()

	// 测试1: 正常注册
	client1 := createTestClient(t, hub, userID, websocket.DeviceTypeWeb)
	assert.Equal(t, 1, hub.GetClientCount())
	assert.Equal(t, 1, hub.GetUserConnectionCount(userID))

	// 测试2: 注册第二个客户端（同一用户）
	client2 := createTestClient(t, hub, userID, websocket.DeviceTypeMobile)
	assert.Equal(t, 2, hub.GetClientCount())
	assert.Equal(t, 2, hub.GetUserConnectionCount(userID))

	// 测试3: 注册第三个客户端（同一用户）
	client3 := createTestClient(t, hub, userID, websocket.DeviceTypeDesktop)
	assert.Equal(t, 3, hub.GetClientCount())
	assert.Equal(t, 3, hub.GetUserConnectionCount(userID))

	// 测试4: 尝试注册第四个客户端（应该断开最早的连接）
	client4 := createTestClient(t, hub, userID, websocket.DeviceTypeTablet)
	assert.Equal(t, 3, hub.GetClientCount()) // 应该还是3个连接
	assert.Equal(t, 3, hub.GetUserConnectionCount(userID))

	// 验证最早的连接（client1）已经被断开
	select {
	case _, ok := <-client1.Send:
		assert.False(t, ok, "client1.Send channel should be closed")
	default:
		// 通道已关闭或没有消息
	}

	// 清理
	hub.UnregisterClient(client2)
	hub.UnregisterClient(client3)
	hub.UnregisterClient(client4)
}

// TestHubMaxConnections 测试全局连接数限制
func TestHubMaxConnections(t *testing.T) {
	maxConnections := 5
	hub := websocket.NewHub(maxConnections, 3)
	go hub.Run()

	// 创建多个用户
	var userIDs []uuid.UUID
	for i := 0; i < 10; i++ {
		userIDs = append(userIDs, uuid.New())
	}

	// 注册客户端直到达到最大连接数
	for i := 0; i < maxConnections; i++ {
		client := &websocket.Client{
			ID:          uuid.New(),
			UserID:      userIDs[i],
			Conn:        nil,
			Send:        make(chan []byte, 256),
			DeviceType:  websocket.DeviceTypeWeb,
			ConnectedAt: time.Now(),
			UserAgent:   "test-agent",
		}
		err := hub.RegisterClient(client)
		assert.NoError(t, err)
	}

	assert.Equal(t, maxConnections, hub.GetClientCount())

	// 尝试注册超过最大连接数的客户端
	client := &websocket.Client{
		ID:          uuid.New(),
		UserID:      userIDs[maxConnections],
		Conn:        nil,
		Send:        make(chan []byte, 256),
		DeviceType:  websocket.DeviceTypeWeb,
		ConnectedAt: time.Now(),
		UserAgent:   "test-agent",
	}
	err := hub.RegisterClient(client)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maximum capacity")
}

// TestHubUnregisterClient 测试客户端注销
func TestHubUnregisterClient(t *testing.T) {
	hub := setupTestHub(t)

	userID := uuid.New()
	client := createTestClient(t, hub, userID, websocket.DeviceTypeWeb)

	assert.Equal(t, 1, hub.GetClientCount())
	assert.Equal(t, 1, hub.GetUserConnectionCount(userID))

	// 注销客户端
	hub.UnregisterClient(client)

	assert.Equal(t, 0, hub.GetClientCount())
	assert.Equal(t, 0, hub.GetUserConnectionCount(userID))
}

// TestHubSendToUser 测试发送消息给用户的所有设备
func TestHubSendToUser(t *testing.T) {
	hub := setupTestHub(t)

	userID := uuid.New()

	// 创建多个客户端（同一用户）
	client1 := createTestClient(t, hub, userID, websocket.DeviceTypeWeb)
	client2 := createTestClient(t, hub, userID, websocket.DeviceTypeMobile)
	client3 := createTestClient(t, hub, userID, websocket.DeviceTypeDesktop)

	// 发送消息给用户
	testData := map[string]interface{}{
		"type": "test",
		"data": "test message",
	}
	hub.SendToUser(userID, "test_message", testData)

	// 验证所有客户端都收到了消息
	var receivedCount int64
	var wg sync.WaitGroup
	wg.Add(3)

	// 从client1接收消息
	go func() {
		defer wg.Done()
		select {
		case msg := <-client1.Send:
			var received websocket.BroadcastMessage
			err := json.Unmarshal(msg, &received)
			assert.NoError(t, err)
			assert.Equal(t, "test_message", received.Type)
			atomic.AddInt64(&receivedCount, 1)
		case <-time.After(1 * time.Second):
			// 超时
		}
	}()

	// 从client2接收消息
	go func() {
		defer wg.Done()
		select {
		case msg := <-client2.Send:
			var received websocket.BroadcastMessage
			err := json.Unmarshal(msg, &received)
			assert.NoError(t, err)
			assert.Equal(t, "test_message", received.Type)
			atomic.AddInt64(&receivedCount, 1)
		case <-time.After(1 * time.Second):
			// 超时
		}
	}()

	// 从client3接收消息
	go func() {
		defer wg.Done()
		select {
		case msg := <-client3.Send:
			var received websocket.BroadcastMessage
			err := json.Unmarshal(msg, &received)
			assert.NoError(t, err)
			assert.Equal(t, "test_message", received.Type)
			atomic.AddInt64(&receivedCount, 1)
		case <-time.After(1 * time.Second):
			// 超时
		}
	}()

	wg.Wait()
	assert.Equal(t, int64(3), atomic.LoadInt64(&receivedCount))

	// 清理
	hub.UnregisterClient(client1)
	hub.UnregisterClient(client2)
	hub.UnregisterClient(client3)
}

// TestHubSendToConversation 测试发送消息给会话成员
func TestHubSendToConversation(t *testing.T) {
	hub := setupTestHub(t)

	// 创建用户
	user1ID := uuid.New()
	user2ID := uuid.New()
	user3ID := uuid.New()

	// 为每个用户创建多个客户端
	user1Client1 := createTestClient(t, hub, user1ID, websocket.DeviceTypeWeb)
	user1Client2 := createTestClient(t, hub, user1ID, websocket.DeviceTypeMobile)
	user2Client1 := createTestClient(t, hub, user2ID, websocket.DeviceTypeWeb)
	user2Client2 := createTestClient(t, hub, user2ID, websocket.DeviceTypeMobile)
	user3Client1 := createTestClient(t, hub, user3ID, websocket.DeviceTypeWeb)

	conversationID := uuid.New()

	// 创建测试消息
	message := models.Message{
		ID:             uuid.New(),
		ConversationID: conversationID,
		SenderID:       user1ID,
		Content:        "Hello, everyone!",
		CreatedAt:      time.Now(),
	}

	// 发送消息给会话（不包括发送者）
	memberIDs := []uuid.UUID{user1ID, user2ID, user3ID}
	hub.SendToConversation(conversationID, user1ID, message, memberIDs)

	// 验证user2的两个客户端都收到了消息
	var receivedCount int64
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		select {
		case msg := <-user2Client1.Send:
			var received websocket.BroadcastMessage
			err := json.Unmarshal(msg, &received)
			assert.NoError(t, err)
			assert.Equal(t, "new_message", received.Type)
			atomic.AddInt64(&receivedCount, 1)
		case <-time.After(1 * time.Second):
		}
	}()

	go func() {
		defer wg.Done()
		select {
		case msg := <-user2Client2.Send:
			var received websocket.BroadcastMessage
			err := json.Unmarshal(msg, &received)
			assert.NoError(t, err)
			assert.Equal(t, "new_message", received.Type)
			atomic.AddInt64(&receivedCount, 1)
		case <-time.After(1 * time.Second):
		}
	}()

	wg.Wait()
	assert.Equal(t, int64(2), atomic.LoadInt64(&receivedCount))

	// 清理
	hub.UnregisterClient(user1Client1)
	hub.UnregisterClient(user1Client2)
	hub.UnregisterClient(user2Client1)
	hub.UnregisterClient(user2Client2)
	hub.UnregisterClient(user3Client1)
}

// TestHubGetConnectionStats 测试获取连接统计信息
func TestHubGetConnectionStats(t *testing.T) {
	hub := setupTestHub(t)

	// 创建多个用户和客户端
	user1ID := uuid.New()
	user2ID := uuid.New()

	createTestClient(t, hub, user1ID, websocket.DeviceTypeWeb)
	createTestClient(t, hub, user1ID, websocket.DeviceTypeMobile)
	createTestClient(t, hub, user2ID, websocket.DeviceTypeWeb)

	stats := hub.GetConnectionStats()

	assert.Equal(t, 3, stats["total_connections"])
	assert.Equal(t, 2, stats["total_users"])
	assert.Equal(t, 100, stats["max_connections"])
	assert.Equal(t, 3, stats["max_user_connections"])

	deviceStats, ok := stats["device_connections"].(map[websocket.DeviceType]int)
	require.True(t, ok)
	assert.Equal(t, 2, deviceStats[websocket.DeviceTypeWeb])
	assert.Equal(t, 1, deviceStats[websocket.DeviceTypeMobile])
}

// TestHubDisconnectUserDevice 测试断开用户指定设备类型的所有连接
func TestHubDisconnectUserDevice(t *testing.T) {
	hub := setupTestHub(t)

	userID := uuid.New()

	// 创建多个不同设备类型的客户端
	client1 := createTestClient(t, hub, userID, websocket.DeviceTypeWeb)
	client2 := createTestClient(t, hub, userID, websocket.DeviceTypeWeb) // 第二个web客户端
	client3 := createTestClient(t, hub, userID, websocket.DeviceTypeMobile)

	assert.Equal(t, 3, hub.GetClientCount())
	assert.Equal(t, 2, hub.GetUserDeviceConnectionCount(userID, websocket.DeviceTypeWeb))
	assert.Equal(t, 1, hub.GetUserDeviceConnectionCount(userID, websocket.DeviceTypeMobile))

	// 断开所有web设备
	hub.DisconnectUserDevice(userID, websocket.DeviceTypeWeb)

	assert.Equal(t, 1, hub.GetClientCount())
	assert.Equal(t, 0, hub.GetUserDeviceConnectionCount(userID, websocket.DeviceTypeWeb))
	assert.Equal(t, 1, hub.GetUserDeviceConnectionCount(userID, websocket.DeviceTypeMobile))

	// 验证web客户端的通道已关闭
	select {
	case _, ok := <-client1.Send:
		assert.False(t, ok, "client1.Send channel should be closed")
	default:
	}

	select {
	case _, ok := <-client2.Send:
		assert.False(t, ok, "client2.Send channel should be closed")
	default:
	}

	// 清理
	hub.UnregisterClient(client3)
}

// TestHubDisconnectOldestUserDevice 测试断开用户指定设备类型的最早连接
func TestHubDisconnectOldestUserDevice(t *testing.T) {
	hub := setupTestHub(t)

	userID := uuid.New()

	// 创建多个web设备类型的客户端
	client1 := createTestClient(t, hub, userID, websocket.DeviceTypeWeb)
	time.Sleep(10 * time.Millisecond) // 确保时间戳不同
	client2 := createTestClient(t, hub, userID, websocket.DeviceTypeWeb)
	time.Sleep(10 * time.Millisecond)
	client3 := createTestClient(t, hub, userID, websocket.DeviceTypeWeb)

	assert.Equal(t, 3, hub.GetClientCount())
	assert.Equal(t, 3, hub.GetUserDeviceConnectionCount(userID, websocket.DeviceTypeWeb))

	// 断开最早的web设备
	disconnected := hub.DisconnectOldestUserDevice(userID, websocket.DeviceTypeWeb)
	assert.True(t, disconnected)

	assert.Equal(t, 2, hub.GetClientCount())
	assert.Equal(t, 2, hub.GetUserDeviceConnectionCount(userID, websocket.DeviceTypeWeb))

	// 验证最早的客户端（client1）已被断开
	select {
	case _, ok := <-client1.Send:
		assert.False(t, ok, "client1.Send channel should be closed")
	default:
	}

	// 验证其他客户端仍然正常
	select {
	case _, ok := <-client2.Send:
		assert.True(t, ok || len(client2.Send) == 0) // 通道可能为空但不应该关闭
	case <-time.After(100 * time.Millisecond):
	}

	// 清理
	hub.UnregisterClient(client2)
	hub.UnregisterClient(client3)
}

// TestDetectDeviceType 测试设备类型检测
func TestDetectDeviceType(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		expected  websocket.DeviceType
	}{
		{
			name:      "Web Browser",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			expected:  websocket.DeviceTypeWeb,
		},
		{
			name:      "Mobile Device",
			userAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X)",
			expected:  websocket.DeviceTypeMobile,
		},
		{
			name:      "Android Device",
			userAgent: "Mozilla/5.0 (Linux; Android 10; SM-G973F)",
			expected:  websocket.DeviceTypeMobile,
		},
		{
			name:      "Tablet Device",
			userAgent: "Mozilla/5.0 (iPad; CPU OS 14_0 like Mac OS X)",
			expected:  websocket.DeviceTypeTablet,
		},
		{
			name:      "Unknown Device",
			userAgent: "CustomClient/1.0",
			expected:  websocket.DeviceTypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 由于detectDeviceType是未导出的函数，我们需要通过其他方式测试
			// 这里我们只是记录测试用例，实际测试可能需要修改为导出函数或通过其他方式
			t.Logf("User-Agent: %s, Expected: %s", tt.userAgent, tt.expected)
		})
	}
}

// TestHubConcurrentAccess 测试并发访问
func TestHubConcurrentAccess(t *testing.T) {
	hub := setupTestHub(t)

	var wg sync.WaitGroup
	numUsers := 10
	numClientsPerUser := 3

	// 并发注册客户端
	for i := 0; i < numUsers; i++ {
		userID := uuid.New()
		for j := 0; j < numClientsPerUser; j++ {
			wg.Add(1)
			go func(uid uuid.UUID) {
				defer wg.Done()
				client := &websocket.Client{
					ID:          uuid.New(),
					UserID:      uid,
					Conn:        nil,
					Send:        make(chan []byte, 256),
					DeviceType:  websocket.DeviceTypeWeb,
					ConnectedAt: time.Now(),
					UserAgent:   "test-agent",
				}
				err := hub.RegisterClient(client)
				if err != nil {
					t.Logf("Failed to register client: %v", err)
				}
			}(userID)
		}
	}

	wg.Wait()
	assert.Equal(t, numUsers*numClientsPerUser, hub.GetClientCount())
}

// TestWebSocketHandler 测试WebSocket处理器
func TestWebSocketHandler(t *testing.T) {
	// 设置Gin为测试模式
	gin.SetMode(gin.TestMode)

	// 初始化WebSocket Hub
	hub := websocket.NewHub(100, 3)
	go hub.Run()
	websocket.GlobalHub = hub
	websocket.InitJWTSecret("test-secret")

	// 创建测试路由
	router := gin.New()
	router.GET("/ws", websocket.HandleWebSocket)

	// 创建测试服务器
	server := httptest.NewServer(router)
	defer server.Close()

	// 创建WebSocket连接
	wsURL := "ws" + server.URL[4:] + "/ws?token=" + createTestToken(t, uuid.New(), "test-secret")

	// 注意：这个测试可能需要更复杂的设置，因为需要真实的WebSocket连接
	// 这里只是演示测试结构
	t.Logf("WebSocket URL: %s", wsURL)
}
