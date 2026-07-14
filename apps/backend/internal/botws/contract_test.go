package botws

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"purr-chat-server/internal/botws/testkit"
	"purr-chat-server/internal/models"
	"purr-chat-server/internal/onebot"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type contractDispatcher struct{}

func (contractDispatcher) Dispatch(_ context.Context, _ models.BotPrincipal, request onebot.ActionRequest) (json.RawMessage, error) {
	return json.RawMessage(`{"action":"` + request.Action + `"}`), nil
}

func TestFakeClientActionEchoEventAndReconnect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := NewManager(testConfig(), contractDispatcher{})
	principal := &models.BotPrincipal{BotID: uuid.New(), IdentityID: uuid.New(), CredentialID: uuid.New()}
	router := gin.New()
	router.GET("/ws", func(c *gin.Context) {
		c.Set("bot_principal", principal)
	}, NewHandler(manager, nil, nil).Connect)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	endpoint := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	client, err := testkit.Dial(context.Background(), endpoint, "")
	require.NoError(t, err)
	lifecycle, err := client.ReadEvent(time.Second)
	require.NoError(t, err)
	assert.Equal(t, "lifecycle", lifecycle.DetailType)
	require.NoError(t, client.SendAction("get_status", map[string]any{}, map[string]any{"request": 1}))
	response, err := client.ReadActionResponse(time.Second)
	require.NoError(t, err)
	assert.Equal(t, onebot.StatusOK, response.Status)
	assert.JSONEq(t, `{"request":1}`, string(response.Echo))
	require.NoError(t, client.Close())

	client, err = testkit.Dial(context.Background(), endpoint, "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	lifecycle, err = client.ReadEvent(time.Second)
	require.NoError(t, err)
	assert.Equal(t, "connect", lifecycle.SubType)
}

func TestStableRegistryEntriesHaveWireContracts(t *testing.T) {
	for _, action := range onebot.Actions() {
		if action.Status != onebot.StatusStable {
			continue
		}
		request, err := onebot.DecodeActionRequest(action.RequestExample)
		require.NoError(t, err, action.Name)
		assert.Equal(t, action.Name, request.Action, action.Name)
		var response onebot.ActionResponse
		require.NoError(t, json.Unmarshal(action.ResponseExample, &response), action.Name)
		assert.NotEmpty(t, response.Status, action.Name)
	}
	for _, event := range onebot.Events() {
		if event.Status != onebot.StatusStable {
			continue
		}
		var example onebot.Event
		require.NoError(t, json.Unmarshal(event.EventExample, &example), event.DetailType)
		assert.Equal(t, event.PostType, example.PostType, event.DetailType)
		assert.Equal(t, event.DetailType, example.DetailType, event.DetailType)
	}
}

func TestHealthReturnsAggregateMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := NewManager(testConfig(), nil)
	router := gin.New()
	router.GET("/api/bot/v1/health", NewHandler(manager, nil, nil).Health)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/bot/v1/health", nil))
	require.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"status":"ok","metrics":{"accepted":0,"active":0,"rejected":0,"messages_read":0,"messages_written":0,"events_published":0,"events_delivered":0,"events_dropped":0,"action_started":0,"action_completed":0,"action_failed":0,"action_rejected":0,"action_latency_nanoseconds":0,"queue_depth":0,"queue_overflows":0,"protocol_errors":0}}`, response.Body.String())
}
