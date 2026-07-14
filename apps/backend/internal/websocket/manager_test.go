package websocket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHub(t *testing.T) {
	hub := NewHub(HubConfig{MaxConnections: 100, MaxUserDeviceConnections: 5, SendQueueSize: 64})
	assert.NotNil(t, hub)
	assert.Equal(t, 100, hub.config.MaxConnections)
	assert.Equal(t, 5, hub.config.MaxUserDeviceConnections)
	assert.NotNil(t, hub.clients)
	assert.NotNil(t, hub.userClients)
	assert.NotNil(t, hub.metrics)
}

func TestHubConfigDefaults(t *testing.T) {
	hub := NewHub(HubConfig{})
	assert.Equal(t, 5, hub.config.MaxUserDeviceConnections)
	assert.Equal(t, 256, hub.config.SendQueueSize)
	assert.Equal(t, int64(1<<20), hub.config.ReadLimit)
	assert.Equal(t, 10*time.Second, hub.config.WriteTimeout)
	assert.Equal(t, 60*time.Second, hub.config.ReadTimeout)
	assert.Equal(t, 54*time.Second, hub.config.PingInterval)
}

func TestBroadcastMessageStruct(t *testing.T) {
	msg := BroadcastMessage{Type: "test_type", Data: "test_data", Timestamp: time.Now().Unix()}
	assert.Equal(t, "test_type", msg.Type)
	assert.Equal(t, "test_data", msg.Data)
	assert.NotZero(t, msg.Timestamp)
}

func TestPrivateMessageStruct(t *testing.T) {
	userID := uuid.New()
	msg := PrivateMessage{RecipientID: userID}
	assert.Equal(t, userID, msg.RecipientID)
}

func TestClientCloseOnce(t *testing.T) {
	client := &Client{
		ID:     uuid.New(),
		UserID: uuid.New(),
		Send:   make(chan []byte, 1),
		done:   make(chan struct{}),
	}
	// close should be idempotent
	client.close(1000, "test")
	client.close(1001, "test2") // should not panic
	assert.Equal(t, 1000, client.closeCode)
	assert.Equal(t, "test", client.closeReason)
}

func TestRegisterAndUnregister(t *testing.T) {
	hub := NewHub(HubConfig{MaxConnections: 10, MaxUserDeviceConnections: 5, SendQueueSize: 64})
	go hub.Run()
	defer hub.Shutdown()

	userID := uuid.New()
	client := &Client{
		ID:          uuid.New(),
		UserID:      userID,
		Send:        make(chan []byte, 64),
		DeviceType:  DeviceTypeWeb,
		ConnectedAt: time.Now(),
	}

	err := hub.RegisterClient(client)
	require.NoError(t, err)
	assert.Equal(t, 1, hub.GetClientCount())
	assert.Equal(t, 1, hub.GetUserConnectionCount(userID))

	hub.UnregisterClient(client)
	assert.Equal(t, 0, hub.GetClientCount())
	assert.Equal(t, 0, hub.GetUserConnectionCount(userID))
}

func TestMaxConnections(t *testing.T) {
	hub := NewHub(HubConfig{MaxConnections: 2, MaxUserDeviceConnections: 5, SendQueueSize: 64})
	go hub.Run()
	defer hub.Shutdown()

	for i := 0; i < 2; i++ {
		client := &Client{
			ID:     uuid.New(),
			UserID: uuid.New(),
			Send:   make(chan []byte, 64),
		}
		err := hub.RegisterClient(client)
		require.NoError(t, err)
	}

	client3 := &Client{
		ID:     uuid.New(),
		UserID: uuid.New(),
		Send:   make(chan []byte, 64),
	}
	err := hub.RegisterClient(client3)
	assert.Error(t, err)
}

func TestMaxUserDeviceConnectionsEvictsOldestOfSameDeviceType(t *testing.T) {
	hub := NewHub(HubConfig{MaxConnections: 100, MaxUserDeviceConnections: 2, SendQueueSize: 64})
	go hub.Run()
	defer hub.Shutdown()

	userID := uuid.New()
	c1 := &Client{ID: uuid.New(), UserID: userID, Send: make(chan []byte, 64), DeviceType: DeviceTypeWeb}
	c2 := &Client{ID: uuid.New(), UserID: userID, Send: make(chan []byte, 64), DeviceType: DeviceTypeWeb}

	require.NoError(t, hub.RegisterClient(c1))
	require.NoError(t, hub.RegisterClient(c2))

	c3 := &Client{ID: uuid.New(), UserID: userID, Send: make(chan []byte, 64), DeviceType: DeviceTypeWeb}
	require.NoError(t, hub.RegisterClient(c3))

	assert.Equal(t, 2, hub.GetUserConnectionCount(userID))
	assert.Equal(t, 2, hub.GetUserDeviceConnectionCount(userID, DeviceTypeWeb))
	assert.Equal(t, CloseConnectionReplaced, c1.closeCode)
	assert.Equal(t, "connection replaced by newer session", c1.closeReason)
	select {
	case <-c1.done:
	default:
		t.Fatal("expected oldest connection to be closed")
	}
}

func TestMaxUserDeviceConnectionsCountsDeviceTypesSeparately(t *testing.T) {
	hub := NewHub(HubConfig{MaxConnections: 100, MaxUserDeviceConnections: 2, SendQueueSize: 64})
	go hub.Run()
	defer hub.Shutdown()

	userID := uuid.New()
	web1 := &Client{ID: uuid.New(), UserID: userID, Send: make(chan []byte, 64), DeviceType: DeviceTypeWeb}
	web2 := &Client{ID: uuid.New(), UserID: userID, Send: make(chan []byte, 64), DeviceType: DeviceTypeWeb}
	desktop1 := &Client{ID: uuid.New(), UserID: userID, Send: make(chan []byte, 64), DeviceType: DeviceTypeDesktop}
	desktop2 := &Client{ID: uuid.New(), UserID: userID, Send: make(chan []byte, 64), DeviceType: DeviceTypeDesktop}

	for _, client := range []*Client{web1, web2, desktop1, desktop2} {
		require.NoError(t, hub.RegisterClient(client))
	}

	assert.Equal(t, 4, hub.GetUserConnectionCount(userID))
	assert.Equal(t, 2, hub.GetUserDeviceConnectionCount(userID, DeviceTypeWeb))
	assert.Equal(t, 2, hub.GetUserDeviceConnectionCount(userID, DeviceTypeDesktop))
	assert.Equal(t, 0, web1.closeCode)
	assert.Equal(t, 0, desktop1.closeCode)
}

func TestConcurrentRegisterUnregister(t *testing.T) {
	hub := NewHub(HubConfig{MaxConnections: 1000, MaxUserDeviceConnections: 10, SendQueueSize: 64})
	go hub.Run()
	defer hub.Shutdown()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			client := &Client{
				ID:     uuid.New(),
				UserID: uuid.New(),
				Send:   make(chan []byte, 64),
			}
			_ = hub.RegisterClient(client)
			time.Sleep(time.Millisecond)
			hub.UnregisterClient(client)
		}(i)
	}
	wg.Wait()
	assert.Equal(t, 0, hub.GetClientCount())
}

func TestConcurrentBroadcast(t *testing.T) {
	hub := NewHub(HubConfig{MaxConnections: 100, MaxUserDeviceConnections: 10, SendQueueSize: 64})
	go hub.Run()
	defer hub.Shutdown()

	clients := make([]*Client, 20)
	for i := range clients {
		clients[i] = &Client{
			ID:     uuid.New(),
			UserID: uuid.New(),
			Send:   make(chan []byte, 64),
		}
		require.NoError(t, hub.RegisterClient(clients[i]))
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hub.broadcastMessage([]byte(`{"type":"test"}`))
		}()
	}
	wg.Wait()

	for _, c := range clients {
		select {
		case <-c.Send:
		default:
			t.Error("expected message in client send queue")
		}
	}
}

func TestBroadcastQueueOverflowDisconnects(t *testing.T) {
	hub := NewHub(HubConfig{MaxConnections: 100, MaxUserDeviceConnections: 10, SendQueueSize: 2})
	go hub.Run()
	defer hub.Shutdown()

	client := &Client{
		ID:     uuid.New(),
		UserID: uuid.New(),
		Send:   make(chan []byte, 2),
	}
	require.NoError(t, hub.RegisterClient(client))

	// Fill the queue
	client.Send <- []byte(`{"type":"msg1"}`)
	client.Send <- []byte(`{"type":"msg2"}`)

	// This should overflow and disconnect the client
	hub.broadcastMessage([]byte(`{"type":"msg3"}`))

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, hub.GetClientCount())
	assert.True(t, hub.metrics.QueueOverflows.Load() > 0)
}

func TestSendToUser(t *testing.T) {
	hub := NewHub(HubConfig{MaxConnections: 100, MaxUserDeviceConnections: 10, SendQueueSize: 64})
	go hub.Run()
	defer hub.Shutdown()

	userID := uuid.New()
	client := &Client{
		ID:     uuid.New(),
		UserID: userID,
		Send:   make(chan []byte, 64),
	}
	require.NoError(t, hub.RegisterClient(client))

	hub.SendToUser(userID, "test_event", map[string]string{"hello": "world"})

	select {
	case msg := <-client.Send:
		var bm BroadcastMessage
		require.NoError(t, json.Unmarshal(msg, &bm))
		assert.Equal(t, "test_event", bm.Type)
	default:
		t.Error("expected message in send queue")
	}
}

func TestShutdown(t *testing.T) {
	hub := NewHub(HubConfig{MaxConnections: 100, MaxUserDeviceConnections: 10, SendQueueSize: 64})
	go hub.Run()

	for i := 0; i < 5; i++ {
		client := &Client{
			ID:     uuid.New(),
			UserID: uuid.New(),
			Send:   make(chan []byte, 64),
		}
		require.NoError(t, hub.RegisterClient(client))
	}

	assert.Equal(t, 5, hub.GetClientCount())
	hub.Shutdown()
	assert.Equal(t, 0, hub.GetClientCount())

	// Register after shutdown should fail
	client := &Client{ID: uuid.New(), UserID: uuid.New(), Send: make(chan []byte, 64)}
	err := hub.RegisterClient(client)
	assert.Error(t, err)
}

func TestGetConnectionStats(t *testing.T) {
	hub := NewHub(HubConfig{MaxConnections: 100, MaxUserDeviceConnections: 10, SendQueueSize: 64})

	userID := uuid.New()
	client := &Client{
		ID:         uuid.New(),
		UserID:     userID,
		Send:       make(chan []byte, 64),
		DeviceType: DeviceTypeWeb,
	}
	require.NoError(t, hub.RegisterClient(client))

	stats := hub.GetConnectionStats()
	assert.Equal(t, 1, stats["total_connections"])
	assert.Equal(t, 1, stats["total_users"])

	metrics, ok := stats["metrics"].(map[string]int64)
	require.True(t, ok)
	assert.True(t, metrics["total_connections"] > 0)
}

// ===== Integration tests with real WebSocket connections =====

func setupTestHub(t *testing.T, cfg HubConfig) (*Hub, *gin.Engine, *httptest.Server) {
	t.Helper()
	if cfg.SendQueueSize == 0 {
		cfg.SendQueueSize = 64
	}
	if cfg.PingInterval == 0 {
		cfg.PingInterval = 30 * time.Second
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = 60 * time.Second
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = 10 * time.Second
	}
	if cfg.ReadLimit == 0 {
		cfg.ReadLimit = 1 << 20
	}
	hub := NewHub(cfg)
	go hub.Run()

	GlobalHub = hub
	hubConfig = cfg
	jwtSecret = "test_secret"

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/ws", HandleWebSocket)
	server := httptest.NewServer(r)
	t.Cleanup(func() {
		server.Close()
		hub.Shutdown()
	})

	return hub, r, server
}

func wsURL(server *httptest.Server) string {
	return "ws" + strings.TrimPrefix(server.URL, "http") + "/api/ws"
}

func TestWSConnectionRequiresToken(t *testing.T) {
	_, _, server := setupTestHub(t, HubConfig{MaxConnections: 10, MaxUserDeviceConnections: 5})

	_, resp, err := websocket.DefaultDialer.Dial(wsURL(server), nil)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestWSConnectionWithCookie(t *testing.T) {
	_, _, server := setupTestHub(t, HubConfig{MaxConnections: 10, MaxUserDeviceConnections: 5})

	token := generateTestToken(t)
	dialer := websocket.Dialer{}
	header := http.Header{}
	header.Add("Cookie", "purrchat_token="+token)

	conn, resp, err := dialer.Dial(wsURL(server), header)
	require.NoError(t, err)
	assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	defer conn.Close()
}

func TestWSNewSameDeviceConnectionReplacesOldestWithoutRetryCode(t *testing.T) {
	hub, _, server := setupTestHub(t, HubConfig{
		MaxConnections:           10,
		MaxUserDeviceConnections: 1,
		SendQueueSize:            64,
	})

	token := generateTestToken(t)
	header := http.Header{}
	header.Add("Cookie", "purrchat_token="+token)
	header.Set("User-Agent", "Mozilla/5.0 Chrome/149.0")

	oldest, _, err := websocket.DefaultDialer.Dial(wsURL(server), header)
	require.NoError(t, err)
	defer oldest.Close()

	newest, _, err := websocket.DefaultDialer.Dial(wsURL(server), header)
	require.NoError(t, err)
	defer newest.Close()

	_, _, err = oldest.ReadMessage()
	require.Error(t, err)
	var closeErr *websocket.CloseError
	require.ErrorAs(t, err, &closeErr)
	assert.Equal(t, CloseConnectionReplaced, closeErr.Code)
	assert.Equal(t, "connection replaced by newer session", closeErr.Text)
	assert.Eventually(t, func() bool {
		return hub.GetClientCount() == 1
	}, time.Second, 10*time.Millisecond)
}

func TestWSConnectionWithSubprotocol(t *testing.T) {
	_, _, server := setupTestHub(t, HubConfig{MaxConnections: 10, MaxUserDeviceConnections: 5})

	token := generateTestToken(t)
	header := http.Header{}
	header.Set("Sec-WebSocket-Protocol", "bearer,"+token)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server), header)
	require.NoError(t, err)
	defer conn.Close()
}

func TestWSQueryTokenRejectedByDefault(t *testing.T) {
	_, _, server := setupTestHub(t, HubConfig{MaxConnections: 10, MaxUserDeviceConnections: 5})

	token := generateTestToken(t)
	_, resp, err := websocket.DefaultDialer.Dial(wsURL(server)+"?token="+token, nil)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestWSOriginRejected(t *testing.T) {
	_, _, server := setupTestHub(t, HubConfig{
		MaxConnections:           10,
		MaxUserDeviceConnections: 5,
		AllowedOrigins:           []string{"https://allowed.com"},
	})

	token := generateTestToken(t)
	header := http.Header{}
	header.Add("Cookie", "purrchat_token="+token)
	header.Set("Origin", "https://evil.com")

	dialer := websocket.Dialer{}
	_, resp, err := dialer.Dial(wsURL(server), header)
	require.Error(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestWSOriginAllowed(t *testing.T) {
	_, _, server := setupTestHub(t, HubConfig{
		MaxConnections:           10,
		MaxUserDeviceConnections: 5,
		AllowedOrigins:           []string{"https://allowed.com"},
	})

	token := generateTestToken(t)
	header := http.Header{}
	header.Add("Cookie", "purrchat_token="+token)
	header.Set("Origin", "https://allowed.com")

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(server), header)
	require.NoError(t, err)
	defer conn.Close()
}

func TestWSEachMessageSeparateFrame(t *testing.T) {
	hub, _, server := setupTestHub(t, HubConfig{MaxConnections: 10, MaxUserDeviceConnections: 5, SendQueueSize: 64})

	token := generateTestToken(t)
	header := http.Header{}
	header.Add("Cookie", "purrchat_token="+token)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server), header)
	require.NoError(t, err)
	defer conn.Close()

	userID := getTestUserID(t)

	require.Eventually(t, func() bool {
		return hub.GetClientCount() == 1
	}, 2*time.Second, 20*time.Millisecond, "client should be registered before sending")

	for i := 0; i < 5; i++ {
		hub.SendToUser(userID, "test_event", map[string]int{"i": i})
	}

	// Read each message — each must be a complete JSON object in its own frame
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(3*time.Second)))
	for i := 0; i < 5; i++ {
		msgType, data, err := conn.ReadMessage()
		require.NoError(t, err)
		assert.Equal(t, websocket.TextMessage, msgType)

		var msg BroadcastMessage
		require.NoError(t, json.Unmarshal(data, &msg), "frame %d should be valid JSON", i)
		assert.Equal(t, "test_event", msg.Type)
	}
}

func TestWSMessageTooBig(t *testing.T) {
	_, _, server := setupTestHub(t, HubConfig{
		MaxConnections:           10,
		MaxUserDeviceConnections: 5,
		ReadLimit:                64,
		SendQueueSize:            64,
	})

	token := generateTestToken(t)
	header := http.Header{}
	header.Add("Cookie", "purrchat_token="+token)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server), header)
	require.NoError(t, err)
	defer conn.Close()

	// Send a message larger than the read limit
	bigMsg := strings.Repeat("x", 200)
	err = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"ping","data":"`+bigMsg+`"}`))
	require.NoError(t, err)

	// Server should close the connection
	_, _, err = conn.ReadMessage()
	require.Error(t, err)
}

func TestWSGracefulShutdown(t *testing.T) {
	hub, _, server := setupTestHub(t, HubConfig{MaxConnections: 10, MaxUserDeviceConnections: 5})

	token := generateTestToken(t)
	header := http.Header{}
	header.Add("Cookie", "purrchat_token="+token)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server), header)
	require.NoError(t, err)
	defer conn.Close()

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, hub.GetClientCount())

	hub.Shutdown()
	time.Sleep(100 * time.Millisecond)

	// Connection should be closed
	_, _, err = conn.ReadMessage()
	require.Error(t, err)
}

func TestWSPingPong(t *testing.T) {
	cfg := HubConfig{
		MaxConnections:           10,
		MaxUserDeviceConnections: 5,
		PingInterval:             100 * time.Millisecond,
		ReadTimeout:              500 * time.Millisecond,
		WriteTimeout:             500 * time.Millisecond,
		SendQueueSize:            64,
	}
	_, _, server := setupTestHub(t, cfg)

	token := generateTestToken(t)
	header := http.Header{}
	header.Add("Cookie", "purrchat_token="+token)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server), header)
	require.NoError(t, err)
	defer conn.Close()

	// Set pong handler
	conn.SetPongHandler(func(string) error {
		return nil
	})

	// Wait for ping
	time.Sleep(200 * time.Millisecond)

	// Connection should still be alive
	assert.Equal(t, 1, GlobalHub.GetClientCount())
}

// ===== Helpers =====

var testUserID = uuid.New()

func generateTestToken(t *testing.T) string {
	t.Helper()
	token, err := createTestToken(testUserID)
	require.NoError(t, err)
	return token
}

func getTestUserID(t *testing.T) uuid.UUID {
	t.Helper()
	return testUserID
}
