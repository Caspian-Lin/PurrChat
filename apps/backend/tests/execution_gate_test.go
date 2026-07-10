package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"purr-chat-server/internal/botengine"
	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"
	"purr-chat-server/pkg/database"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockTSHandler struct {
	mu          sync.Mutex
	lastRequest map[string]any
	callCount   int
	replyText   string
	available   bool
}

func (m *mockTSHandler) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			m.mu.Lock()
			defer m.mu.Unlock()
			if m.available {
				w.WriteHeader(200)
				w.Write([]byte(`{"status":"ok"}`))
			} else {
				w.WriteHeader(503)
			}
			return
		}

		if r.URL.Path == "/execute" {
			m.mu.Lock()
			defer m.mu.Unlock()
			m.callCount++
			var req map[string]any
			_ = json.NewDecoder(r.Body).Decode(&req)
			m.lastRequest = req

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"reply":          m.replyText,
				"triggered":      true,
				"session_active": false,
			})
			return
		}

		w.WriteHeader(404)
	}
}

func (m *mockTSHandler) getLastRequest() map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastRequest
}

func (m *mockTSHandler) getCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

func (m *mockTSHandler) setAvailable(v bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.available = v
}

const validWorkflowDocumentJSON = `{
	"apiVersion": "purrchat.ai/v1alpha1",
	"kind": "BotWorkflow",
	"metadata": { "name": "TestBot", "revision": 1 },
	"spec": {
		"trigger": { "type": "rule", "rules": [] },
		"nodes": [
			{ "id": "n1", "type": "trigger", "name": "触发", "config": {} },
			{ "id": "n2", "type": "reply", "name": "回复", "config": { "template": "hello" } },
			{ "id": "n3", "type": "end", "name": "结束", "config": {} }
		],
		"connections": [
			{ "id": "c1", "sourceNodeId": "n1", "sourcePortId": "out_exec", "targetNodeId": "n2", "targetPortId": "in_exec" },
			{ "id": "c2", "sourceNodeId": "n2", "sourcePortId": "out_exec", "targetNodeId": "n3", "targetPortId": "in_exec" }
		],
		"endConditions": [{ "type": "max_rounds", "value": 5 }]
	}
}`

type execTestEnv struct {
	botID          uuid.UUID
	conversationID uuid.UUID
	senderID       uuid.UUID
	wfRepo         repository.WorkflowRepository
	botRepo        repository.BotRepository
	instRepo       repository.BotInstallationRepository
	enrollRepo     repository.EnrollmentRepository
	msgRepo        repository.ConversationMessageRepository
}

func setupExecutionTestEnv(t *testing.T) *execTestEnv {
	t.Helper()
	SetupTestDB(t)

	ctx := context.Background()

	botRepo := repository.NewBotRepository()
	wfRepo := repository.NewWorkflowRepository()
	instRepo := repository.NewBotInstallationRepository()
	enrollRepo := repository.NewEnrollmentRepository()
	convRepo := repository.NewConversationRepository()
	msgRepo := repository.NewConversationMessageRepository()

	owner := CreateTestUser(t, "exec_owner", "exec_owner@test.com", "pass")

	bot := &models.Bot{
		OwnerID: owner.ID,
		Name:    "ExecTestBot",
		Status:  models.BotStatusActive,
	}
	require.NoError(t, botRepo.Create(ctx, bot))

	sender := CreateTestUser(t, "exec_sender", "exec_sender@test.com", "pass")

	conv := &models.Conversation{
		ConversationType: "direct",
		Name:             "test-conv",
	}
	createdBy := sender.ID
	conv.CreatedBy = &createdBy
	require.NoError(t, convRepo.Create(ctx, conv))

	require.NoError(t, enrollRepo.Create(ctx, &models.Enrollment{
		ConversationID: conv.ID,
		UserID:         sender.ID,
		Role:           "member",
	}))
	require.NoError(t, enrollRepo.Create(ctx, &models.Enrollment{
		ConversationID: conv.ID,
		UserID:         bot.ID,
		Role:           "member",
	}))

	require.NoError(t, msgRepo.CreateMessageTable(ctx, conv.ID))

	_, err := wfRepo.Publish(ctx, bot.ID, 1, json.RawMessage(validWorkflowDocumentJSON), []string{"messages:read_trigger", "messages:send"}, owner.ID)
	require.NoError(t, err)

	pv := 1
	_, err = database.GetPool().Exec(ctx, "UPDATE bots SET published_version = $1 WHERE id = $2", pv, bot.ID)
	require.NoError(t, err)

	return &execTestEnv{
		botID:          bot.ID,
		conversationID: conv.ID,
		senderID:       sender.ID,
		wfRepo:         wfRepo,
		botRepo:        botRepo,
		instRepo:       instRepo,
		enrollRepo:     enrollRepo,
		msgRepo:        msgRepo,
	}
}

func TestExecutionGate_ActiveInstallation_ExecutesWithPublishedDocument(t *testing.T) {
	env := setupExecutionTestEnv(t)
	defer CleanupTestDB(t)

	ctx := context.Background()

	mock := &mockTSHandler{replyText: "bot-reply", available: true}
	tsServer := httptest.NewServer(http.HandlerFunc(mock.handler()))
	defer tsServer.Close()

	engine := botengine.NewBotEngine(nil, env.botRepo, env.msgRepo, env.enrollRepo, tsServer.URL)
	engine.SetWorkflowRepo(env.wfRepo)
	engine.SetInstallationRepo(env.instRepo)

	require.NoError(t, env.instRepo.Create(ctx, &models.BotInstallation{
		AppID:               env.botID,
		InstalledBy:         env.senderID,
		TargetType:          models.InstallationTargetUser,
		TargetID:            env.senderID,
		Status:              models.InstallationActive,
		GrantedCapabilities: []string{models.CapabilityReadTrigger, models.CapabilitySend},
	}))

	engine.OnMessage(ctx, &botengine.BotMessage{
		ConversationID: env.conversationID,
		SenderID:       env.senderID,
		SenderName:     "sender",
		Content:        "hello",
		MsgType:        "text",
		CreatedAt:      time.Now(),
	})

	time.Sleep(500 * time.Millisecond)

	req := mock.getLastRequest()
	require.NotNil(t, req, "TS /execute should have been called")

	assert.NotNil(t, req["document"], "should send document, not mechanism_config")
	assert.Nil(t, req["mechanism_config"], "mechanism_config should not be present")
	assert.EqualValues(t, 1, req["revision"])

	msgs, err := env.msgRepo.FindMessages(ctx, env.conversationID, 10, 0)
	require.NoError(t, err)
	found := false
	for _, m := range msgs {
		if m.BotID != nil && *m.BotID == env.botID {
			assert.Equal(t, "bot-reply", m.Content)
			found = true
		}
	}
	assert.True(t, found, "bot reply should be persisted")
}

func TestExecutionGate_PausedInstallation_DoesNotExecute(t *testing.T) {
	env := setupExecutionTestEnv(t)
	defer CleanupTestDB(t)

	ctx := context.Background()

	mock := &mockTSHandler{replyText: "bot-reply", available: true}
	tsServer := httptest.NewServer(http.HandlerFunc(mock.handler()))
	defer tsServer.Close()

	engine := botengine.NewBotEngine(nil, env.botRepo, env.msgRepo, env.enrollRepo, tsServer.URL)
	engine.SetWorkflowRepo(env.wfRepo)
	engine.SetInstallationRepo(env.instRepo)

	require.NoError(t, env.instRepo.Create(ctx, &models.BotInstallation{
		AppID:               env.botID,
		InstalledBy:         env.senderID,
		TargetType:          models.InstallationTargetUser,
		TargetID:            env.senderID,
		Status:              models.InstallationPaused,
		GrantedCapabilities: []string{models.CapabilityReadTrigger, models.CapabilitySend},
	}))

	engine.OnMessage(ctx, &botengine.BotMessage{
		ConversationID: env.conversationID,
		SenderID:       env.senderID,
		Content:        "hello",
		MsgType:        "text",
		CreatedAt:      time.Now(),
	})

	time.Sleep(500 * time.Millisecond)

	assert.Equal(t, 0, mock.getCallCount(), "TS should not be called for paused installation")
}

func TestExecutionGate_NoInstallation_DoesNotExecute(t *testing.T) {
	env := setupExecutionTestEnv(t)
	defer CleanupTestDB(t)

	ctx := context.Background()
	mock := &mockTSHandler{replyText: "bot-reply", available: true}
	tsServer := httptest.NewServer(http.HandlerFunc(mock.handler()))
	defer tsServer.Close()

	engine := botengine.NewBotEngine(nil, env.botRepo, env.msgRepo, env.enrollRepo, tsServer.URL)
	engine.SetWorkflowRepo(env.wfRepo)
	engine.SetInstallationRepo(repository.NewBotInstallationRepository())

	engine.OnMessage(ctx, &botengine.BotMessage{
		ConversationID: env.conversationID,
		SenderID:       env.senderID,
		Content:        "hello",
		MsgType:        "text",
		CreatedAt:      time.Now(),
	})

	time.Sleep(500 * time.Millisecond)

	assert.Equal(t, 0, mock.getCallCount(), "TS should not be called without installation")
}

func TestExecutionGate_NoPublishedVersion_DoesNotExecute(t *testing.T) {
	env := setupExecutionTestEnv(t)
	defer CleanupTestDB(t)

	ctx := context.Background()

	_, err := database.GetPool().Exec(ctx, "UPDATE bots SET published_version = NULL WHERE id = $1", env.botID)
	require.NoError(t, err)

	mock := &mockTSHandler{replyText: "bot-reply", available: true}
	tsServer := httptest.NewServer(http.HandlerFunc(mock.handler()))
	defer tsServer.Close()

	engine := botengine.NewBotEngine(nil, env.botRepo, env.msgRepo, env.enrollRepo, tsServer.URL)
	engine.SetWorkflowRepo(env.wfRepo)
	engine.SetInstallationRepo(env.instRepo)

	require.NoError(t, env.instRepo.Create(ctx, &models.BotInstallation{
		AppID:               env.botID,
		InstalledBy:         env.senderID,
		TargetType:          models.InstallationTargetUser,
		TargetID:            env.senderID,
		Status:              models.InstallationActive,
		GrantedCapabilities: []string{models.CapabilityReadTrigger, models.CapabilitySend},
	}))

	engine.OnMessage(ctx, &botengine.BotMessage{
		ConversationID: env.conversationID,
		SenderID:       env.senderID,
		Content:        "hello",
		MsgType:        "text",
		CreatedAt:      time.Now(),
	})

	time.Sleep(500 * time.Millisecond)

	assert.Equal(t, 0, mock.getCallCount(), "TS should not be called without published version")
}

func TestExecutionGate_TSUnavailable_DoesNotFallbackToGo(t *testing.T) {
	env := setupExecutionTestEnv(t)
	defer CleanupTestDB(t)

	ctx := context.Background()

	mock := &mockTSHandler{replyText: "bot-reply", available: false}
	tsServer := httptest.NewServer(http.HandlerFunc(mock.handler()))
	defer tsServer.Close()

	engine := botengine.NewBotEngine(nil, env.botRepo, env.msgRepo, env.enrollRepo, tsServer.URL)
	engine.SetWorkflowRepo(env.wfRepo)
	engine.SetInstallationRepo(env.instRepo)

	require.NoError(t, env.instRepo.Create(ctx, &models.BotInstallation{
		AppID:               env.botID,
		InstalledBy:         env.senderID,
		TargetType:          models.InstallationTargetUser,
		TargetID:            env.senderID,
		Status:              models.InstallationActive,
		GrantedCapabilities: []string{models.CapabilityReadTrigger, models.CapabilitySend},
	}))

	engine.OnMessage(ctx, &botengine.BotMessage{
		ConversationID: env.conversationID,
		SenderID:       env.senderID,
		Content:        "hello",
		MsgType:        "text",
		CreatedAt:      time.Now(),
	})

	time.Sleep(500 * time.Millisecond)

	assert.Equal(t, 0, mock.getCallCount(), "TS /execute should not be called when service unavailable")

	msgs, err := env.msgRepo.FindMessages(ctx, env.conversationID, 10, 0)
	require.NoError(t, err)
	for _, m := range msgs {
		if m.BotID != nil {
			t.Fatal("Go fallback should not produce a bot reply")
		}
	}
}

func TestExecutionGate_DraftModificationDoesNotAffectExecution(t *testing.T) {
	env := setupExecutionTestEnv(t)
	defer CleanupTestDB(t)

	ctx := context.Background()

	draftDoc := `{
		"apiVersion": "purrchat.ai/v1alpha1",
		"kind": "BotWorkflow",
		"metadata": { "name": "TestBot", "revision": 99 },
		"spec": {
			"trigger": { "type": "rule", "rules": [] },
			"nodes": [
				{ "id": "n1", "type": "trigger", "name": "触发", "config": {} },
				{ "id": "n2", "type": "reply", "name": "回复", "config": { "template": "draft-reply" } },
				{ "id": "n3", "type": "end", "name": "结束", "config": {} }
			],
			"connections": [
				{ "id": "c1", "sourceNodeId": "n1", "sourcePortId": "out_exec", "targetNodeId": "n2", "targetPortId": "in_exec" },
				{ "id": "c2", "sourceNodeId": "n2", "sourcePortId": "out_exec", "targetNodeId": "n3", "targetPortId": "in_exec" }
			],
			"endConditions": [{ "type": "max_rounds", "value": 5 }]
		}
	}`
	_, err := env.wfRepo.UpdateDocument(ctx, env.botID, json.RawMessage(draftDoc), 0)
	require.NoError(t, err)

	mock := &mockTSHandler{replyText: "prod-reply", available: true}
	tsServer := httptest.NewServer(http.HandlerFunc(mock.handler()))
	defer tsServer.Close()

	engine := botengine.NewBotEngine(nil, env.botRepo, env.msgRepo, env.enrollRepo, tsServer.URL)
	engine.SetWorkflowRepo(env.wfRepo)
	engine.SetInstallationRepo(env.instRepo)

	require.NoError(t, env.instRepo.Create(ctx, &models.BotInstallation{
		AppID:               env.botID,
		InstalledBy:         env.senderID,
		TargetType:          models.InstallationTargetUser,
		TargetID:            env.senderID,
		Status:              models.InstallationActive,
		GrantedCapabilities: []string{models.CapabilityReadTrigger, models.CapabilitySend},
	}))

	engine.OnMessage(ctx, &botengine.BotMessage{
		ConversationID: env.conversationID,
		SenderID:       env.senderID,
		Content:        "hello",
		MsgType:        "text",
		CreatedAt:      time.Now(),
	})

	time.Sleep(500 * time.Millisecond)

	req := mock.getLastRequest()
	require.NotNil(t, req)

	doc, ok := req["document"].(map[string]any)
	require.True(t, ok, "document should be present")
	metadata := doc["metadata"].(map[string]any)
	assert.EqualValues(t, 1, metadata["revision"], "should use published revision 1, not draft revision 99")
	assert.EqualValues(t, 1, req["revision"])
}
