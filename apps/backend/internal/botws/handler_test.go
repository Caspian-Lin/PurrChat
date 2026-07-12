package botws

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/onebot"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

type testDispatcher struct {
	active  atomic.Int64
	maximum atomic.Int64
}

type testReplayer struct {
	entries []ResumeEntry
}

func (r testReplayer) FindUnacked(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ int64, _ int) ([]ResumeEntry, error) {
	return r.entries, nil
}

func (d *testDispatcher) Dispatch(ctx context.Context, _ models.BotPrincipal, request onebot.ActionRequest) (json.RawMessage, error) {
	active := d.active.Add(1)
	defer d.active.Add(-1)
	for current := d.maximum.Load(); active > current && !d.maximum.CompareAndSwap(current, active); current = d.maximum.Load() {
	}
	var params struct {
		Delay int `json:"delay"`
	}
	_ = json.Unmarshal(request.Params, &params)
	select {
	case <-time.After(time.Duration(params.Delay) * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return json.RawMessage(`{"action":"` + request.Action + `"}`), nil
}

func testConfig() Config {
	cfg := DefaultConfig()
	cfg.MaxConnections = 4
	cfg.MaxBotConnections = 2
	cfg.MaxConcurrentActions = 2
	cfg.SendQueueSize = 4
	cfg.MaxMessageBytes = 256
	cfg.PingInterval = time.Hour
	cfg.HeartbeatInterval = 0
	cfg.ReadTimeout = time.Hour
	return cfg
}

func startTestServer(t *testing.T, manager *Manager, principal *models.BotPrincipal) (*httptest.Server, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewHandler(manager, nil, nil)
	router.GET("/ws", func(c *gin.Context) {
		if principal != nil {
			c.Set("bot_principal", principal)
		}
		c.Next()
	}, handler.Connect)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return server, "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
}

func dialWS(t *testing.T, endpoint string) *websocket.Conn {
	t.Helper()
	conn, response, err := websocket.DefaultDialer.Dial(endpoint, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusSwitchingProtocols, response.StatusCode)
	_, _, err = conn.ReadMessage()
	require.NoError(t, err) // lifecycle/connect
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func readResponse(t *testing.T, conn *websocket.Conn) onebot.ActionResponse {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, payload, err := conn.ReadMessage()
	require.NoError(t, err)
	var response onebot.ActionResponse
	require.NoError(t, json.Unmarshal(payload, &response))
	return response
}

func TestUniversalWebSocketConcurrentActionsEchoAndLimit(t *testing.T) {
	dispatcher := &testDispatcher{}
	manager := NewManager(testConfig(), dispatcher)
	principal := &models.BotPrincipal{BotID: uuid.New(), IdentityID: uuid.New(), CredentialID: uuid.New()}
	_, endpoint := startTestServer(t, manager, principal)
	conn := dialWS(t, endpoint)
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(`{"action":"slow","params":{"delay":100},"echo":{"id":1}}`)))
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(`{"action":"fast","params":{"delay":5},"echo":"two"}`)))
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(`{"action":"extra","params":{"delay":5},"echo":3}`)))
	first, second, third := readResponse(t, conn), readResponse(t, conn), readResponse(t, conn)
	require.Equal(t, onebot.RetCodeRateLimited, first.RetCode)
	require.JSONEq(t, `3`, string(first.Echo))
	require.JSONEq(t, `"two"`, string(second.Echo))
	require.JSONEq(t, `{"id":1}`, string(third.Echo))
	require.Equal(t, int64(2), dispatcher.maximum.Load())
}

func TestUniversalWebSocketProtocolErrors(t *testing.T) {
	tests := []struct {
		name         string
		messageType  int
		payload      []byte
		closeCode    int
		responseCode onebot.RetCode
	}{
		{name: "malformed JSON", messageType: websocket.TextMessage, payload: []byte(`{"action":`), responseCode: onebot.RetCodeBadRequest},
		{name: "binary frame", messageType: websocket.BinaryMessage, payload: []byte(`{}`), closeCode: CloseInvalidMessage},
		{name: "oversized message", messageType: websocket.TextMessage, payload: []byte(strings.Repeat("x", 300)), closeCode: websocket.CloseMessageTooBig},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager(testConfig(), nil)
			principal := &models.BotPrincipal{BotID: uuid.New(), IdentityID: uuid.New(), CredentialID: uuid.New()}
			_, endpoint := startTestServer(t, manager, principal)
			conn := dialWS(t, endpoint)
			require.NoError(t, conn.WriteMessage(tt.messageType, tt.payload))
			if tt.responseCode != 0 {
				require.Equal(t, tt.responseCode, readResponse(t, conn).RetCode)
				return
			}
			_ = conn.SetReadDeadline(time.Now().Add(time.Second))
			_, _, err := conn.ReadMessage()
			var closeErr *websocket.CloseError
			require.ErrorAs(t, err, &closeErr)
			require.Equal(t, tt.closeCode, closeErr.Code)
		})
	}
}

func TestUniversalWebSocketAuthenticationQueryAndConnectionLimits(t *testing.T) {
	cfg := testConfig()
	cfg.MaxBotConnections = 1
	manager := NewManager(cfg, nil)
	principal := &models.BotPrincipal{BotID: uuid.New(), IdentityID: uuid.New(), CredentialID: uuid.New()}
	_, endpoint := startTestServer(t, manager, principal)
	first := dialWS(t, endpoint)
	second, response, err := websocket.DefaultDialer.Dial(endpoint, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusSwitchingProtocols, response.StatusCode)
	_, _, err = second.ReadMessage()
	var closeErr *websocket.CloseError
	require.ErrorAs(t, err, &closeErr)
	require.Equal(t, CloseConnectionLimit, closeErr.Code)
	_ = second.Close()
	_, response, err = websocket.DefaultDialer.Dial(endpoint+"?access_token=secret", nil)
	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	manager2 := NewManager(cfg, nil)
	_, unauthenticated := startTestServer(t, manager2, nil)
	_, response, err = websocket.DefaultDialer.Dial(unauthenticated, nil)
	require.Error(t, err)
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
	_ = first.Close()
}

func TestManagerBroadcastDisconnectOverflowMetricsAndShutdown(t *testing.T) {
	cfg := testConfig()
	cfg.SendQueueSize = 1
	manager := NewManager(cfg, nil)
	botID, credentialID := uuid.New(), uuid.New()
	c := &connection{manager: manager, principal: models.BotPrincipal{BotID: botID, CredentialID: credentialID}, send: make(chan outbound, 1), done: make(chan struct{}), actions: make(chan struct{}, 1)}
	require.NoError(t, manager.register(c))
	require.Equal(t, 1, manager.PublishBotEvent(botID, map[string]string{"type": "one"}))
	require.Equal(t, 0, manager.PublishBotEvent(botID, map[string]string{"type": "two"}))
	metrics := manager.Metrics()
	require.Equal(t, uint64(2), metrics.EventsPublished)
	require.Equal(t, uint64(1), metrics.EventsDelivered)
	require.Equal(t, uint64(1), metrics.EventsDropped)
	require.Equal(t, int64(1), metrics.QueueDepth)
	require.Equal(t, uint64(1), metrics.QueueOverflows)
	require.Equal(t, "send queue overflow", manager.Status(botID).LastError)
	manager.unregister(c)

	manager = NewManager(testConfig(), nil)
	_, endpoint := startTestServer(t, manager, &models.BotPrincipal{BotID: botID, IdentityID: uuid.New(), CredentialID: credentialID})
	conn := dialWS(t, endpoint)
	require.Equal(t, 1, manager.PublishBotEvent(botID, map[string]string{"post_type": "notice"}))
	_, payload, err := conn.ReadMessage()
	require.NoError(t, err)
	require.Contains(t, string(payload), "notice")
	require.NoError(t, manager.DisconnectCredential(context.Background(), credentialID))
	_, _, err = conn.ReadMessage()
	var closeErr *websocket.CloseError
	require.ErrorAs(t, err, &closeErr)
	require.Equal(t, CloseCredentialInvalid, closeErr.Code)

	manager2 := NewManager(testConfig(), nil)
	_, endpoint2 := startTestServer(t, manager2, &models.BotPrincipal{BotID: botID, IdentityID: uuid.New(), CredentialID: uuid.New()})
	conn2 := dialWS(t, endpoint2)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, manager2.Shutdown(ctx))
	_, _, err = conn2.ReadMessage()
	require.ErrorAs(t, err, &closeErr)
	require.Equal(t, CloseServerShutdown, closeErr.Code)
	require.Equal(t, int64(0), manager2.Metrics().Active)
}

func TestRegistryDispatcherReturnsStableUnsupported(t *testing.T) {
	dispatcher := RegistryDispatcher{}
	_, err := dispatcher.Dispatch(context.Background(), models.BotPrincipal{}, onebot.ActionRequest{Action: "get_login_info", Params: json.RawMessage(`{}`)})
	require.Equal(t, onebot.RetCodeUnsupportedAction, onebot.AsError(err).Code)
	require.Equal(t, "action is not implemented: get_login_info", err.Error())
}

func TestHeartbeatEventDoesNotMasqueradeAsClientPong(t *testing.T) {
	cfg := testConfig()
	cfg.HeartbeatInterval = time.Millisecond
	manager := NewManager(cfg, nil)
	principal := &models.BotPrincipal{BotID: uuid.New(), IdentityID: uuid.New(), CredentialID: uuid.New()}
	_, endpoint := startTestServer(t, manager, principal)
	conn := dialWS(t, endpoint)
	_, _, err := conn.ReadMessage()
	require.NoError(t, err)
	require.Nil(t, manager.Status(principal.BotID).LastHeartbeat)
}

func TestUniversalWebSocketActionTimeoutKeepsConnectionUsable(t *testing.T) {
	cfg := testConfig()
	cfg.ActionTimeout = 10 * time.Millisecond
	manager := NewManager(cfg, &testDispatcher{})
	principal := &models.BotPrincipal{BotID: uuid.New(), IdentityID: uuid.New(), CredentialID: uuid.New()}
	_, endpoint := startTestServer(t, manager, principal)
	conn := dialWS(t, endpoint)
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(`{"action":"slow","params":{"delay":100},"echo":1}`)))
	response := readResponse(t, conn)
	require.Equal(t, onebot.RetCodeInternal, response.RetCode)
	require.Equal(t, "action timeout", response.Message)
	require.Eventually(t, func() bool {
		metrics := manager.Metrics()
		return metrics.ActionFailed == 1 && metrics.ActionCompleted == 1 && metrics.ActionLatencyNanoseconds > 0
	}, time.Second, time.Millisecond)
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(`{"action":"next","params":{"delay":0},"echo":2}`)))
	require.Equal(t, onebot.RetCodeOK, readResponse(t, conn).RetCode)
}

func TestReplayTargetsOnlyReconnectingConnection(t *testing.T) {
	manager := NewManager(testConfig(), nil)
	botID, credentialID := uuid.New(), uuid.New()
	manager.SetReplayer(testReplayer{entries: []ResumeEntry{{Seq: 7, Payload: []byte(`{"event_id":"evt_test"}`)}}})

	principal := models.BotPrincipal{BotID: botID, CredentialID: credentialID}
	reconnecting := &connection{manager: manager, principal: principal, send: make(chan outbound, 2), done: make(chan struct{}), actions: make(chan struct{}, 1)}
	other := &connection{manager: manager, principal: principal, send: make(chan outbound, 2), done: make(chan struct{}), actions: make(chan struct{}, 1)}

	require.Equal(t, 1, manager.ReplayConnection(context.Background(), reconnecting, 0))

	item := <-reconnecting.send
	var event onebot.Event
	require.NoError(t, json.Unmarshal(item.payload, &event))
	require.Equal(t, int64(7), event.Seq)

	select {
	case <-other.send:
		t.Fatal("replay must not broadcast to other connections")
	default:
	}
}
