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
				_, _ = w.Write([]byte(`{"status":"ok"}`))
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
				"run_id":         "test-run-id-12345",
				"reply":          m.replyText,
				"triggered":      true,
				"session_active": false,
				"status":         "completed",
				"execution_ms":   42,
				"revision":       1,
				"trace": map[string]any{
					"runId":  "test-run-id-12345",
					"status": "completed",
					"nodes": []map[string]any{
						{"nodeId": "n1", "nodeType": "trigger", "status": "success"},
						{"nodeId": "n2", "nodeType": "reply", "status": "success"},
						{"nodeId": "n3", "nodeType": "end", "status": "success"},
					},
					"startedAt":   1700000000000,
					"completedAt": 1700000000042,
				},
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

	callLogRepo := repository.NewBotCallLogRepository()
	engine := botengine.NewBotEngine(nil, env.botRepo, env.msgRepo, env.enrollRepo, tsServer.URL)
	engine.SetWorkflowRepo(env.wfRepo)
	engine.SetInstallationRepo(env.instRepo)
	engine.SetCallLogRepo(callLogRepo)

	require.NoError(t, env.instRepo.Create(ctx, &models.BotInstallation{
		AppID:               env.botID,
		InstalledBy:         env.senderID,
		TargetType:          models.InstallationTargetUser,
		TargetID:            env.senderID,
		Status:              models.InstallationActive,
		GrantedCapabilities: []string{models.CapabilityReadTrigger, models.CapabilitySend},
		DiagnosticsConsent:  models.DiagnosticsGranted,
	}))

	triggerMsgID := uuid.New()
	engine.OnMessage(ctx, &botengine.BotMessage{
		ConversationID: env.conversationID,
		SenderID:       env.senderID,
		SenderName:     "sender",
		Content:        "hello",
		MsgType:        "text",
		CreatedAt:      time.Now(),
		MessageID:      triggerMsgID,
	})

	time.Sleep(500 * time.Millisecond)

	req := mock.getLastRequest()
	require.NotNil(t, req, "TS /execute should have been called")

	assert.NotNil(t, req["document"], "should send document, not mechanism_config")
	assert.Nil(t, req["mechanism_config"], "mechanism_config should not be present")
	assert.EqualValues(t, 1, req["revision"])

	msgs, err := env.msgRepo.FindMessages(ctx, env.conversationID, 10, 0)
	require.NoError(t, err)
	var replyMsgID *uuid.UUID
	found := false
	for _, m := range msgs {
		if m.BotID != nil && *m.BotID == env.botID {
			assert.Equal(t, "bot-reply", m.Content)
			found = true
			id := m.ID
			replyMsgID = &id
		}
	}
	assert.True(t, found, "bot reply should be persisted")

	logs, err := callLogRepo.FindAllByBotID(ctx, env.botID, 10, 0)
	require.NoError(t, err)
	require.Len(t, logs, 1, "should have exactly one call log")

	log := logs[0]
	assert.Equal(t, "test-run-id-12345", log.RunID, "call log should have run_id from TS response")
	require.NotNil(t, log.TriggerMessageID, "call log should have trigger_message_id")
	assert.Equal(t, triggerMsgID, *log.TriggerMessageID, "call log trigger_message_id should match")
	if replyMsgID != nil {
		require.NotNil(t, log.ReplyMessageID, "call log should have reply_message_id")
		assert.Equal(t, *replyMsgID, *log.ReplyMessageID, "call log reply_message_id should match persisted message ID")
	}
	assert.Equal(t, "completed", log.RunStatus, "call log should have completed status")
	require.NotNil(t, log.WorkflowRevision, "call log should have workflow revision")
	assert.Equal(t, 1, *log.WorkflowRevision, "call log should have workflow revision 1")
	assert.NotNil(t, log.Trace, "call log should have trace data")
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

func TestExecutionGate_TSUnavailable_LogsErrorType(t *testing.T) {
	env := setupExecutionTestEnv(t)
	defer CleanupTestDB(t)

	ctx := context.Background()

	mock := &mockTSHandler{replyText: "bot-reply", available: false}
	tsServer := httptest.NewServer(http.HandlerFunc(mock.handler()))
	defer tsServer.Close()

	callLogRepo := repository.NewBotCallLogRepository()
	engine := botengine.NewBotEngine(nil, env.botRepo, env.msgRepo, env.enrollRepo, tsServer.URL)
	engine.SetWorkflowRepo(env.wfRepo)
	engine.SetInstallationRepo(env.instRepo)
	engine.SetCallLogRepo(callLogRepo)

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

	logs, err := callLogRepo.FindAllByBotID(ctx, env.botID, 10, 0)
	require.NoError(t, err)
	require.Len(t, logs, 1)
	assert.Equal(t, models.RunStatusError, logs[0].RunStatus)
	assert.Equal(t, "ts_unavailable", logs[0].ErrorType)
	assert.False(t, logs[0].Success)
}

func TestExecutionGate_NoPublishedVersion_LogsErrorType(t *testing.T) {
	env := setupExecutionTestEnv(t)
	defer CleanupTestDB(t)

	ctx := context.Background()

	_, err := database.GetPool().Exec(ctx, "UPDATE bots SET published_version = NULL WHERE id = $1", env.botID)
	require.NoError(t, err)

	mock := &mockTSHandler{replyText: "bot-reply", available: true}
	tsServer := httptest.NewServer(http.HandlerFunc(mock.handler()))
	defer tsServer.Close()

	callLogRepo := repository.NewBotCallLogRepository()
	engine := botengine.NewBotEngine(nil, env.botRepo, env.msgRepo, env.enrollRepo, tsServer.URL)
	engine.SetWorkflowRepo(env.wfRepo)
	engine.SetInstallationRepo(env.instRepo)
	engine.SetCallLogRepo(callLogRepo)

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

	logs, err := callLogRepo.FindAllByBotID(ctx, env.botID, 10, 0)
	require.NoError(t, err)
	require.Len(t, logs, 1)
	assert.Equal(t, models.RunStatusError, logs[0].RunStatus)
	assert.Equal(t, "no_published_version", logs[0].ErrorType)
}

func TestExecutionGate_DiagnosticsDenied_ClearsTraceAndTrigger(t *testing.T) {
	env := setupExecutionTestEnv(t)
	defer CleanupTestDB(t)

	ctx := context.Background()

	mock := &mockTSHandler{replyText: "bot-reply", available: true}
	tsServer := httptest.NewServer(http.HandlerFunc(mock.handler()))
	defer tsServer.Close()

	callLogRepo := repository.NewBotCallLogRepository()
	engine := botengine.NewBotEngine(nil, env.botRepo, env.msgRepo, env.enrollRepo, tsServer.URL)
	engine.SetWorkflowRepo(env.wfRepo)
	engine.SetInstallationRepo(env.instRepo)
	engine.SetCallLogRepo(callLogRepo)

	require.NoError(t, env.instRepo.Create(ctx, &models.BotInstallation{
		AppID:               env.botID,
		InstalledBy:         env.senderID,
		TargetType:          models.InstallationTargetUser,
		TargetID:            env.senderID,
		Status:              models.InstallationActive,
		GrantedCapabilities: []string{models.CapabilityReadTrigger, models.CapabilitySend},
		DiagnosticsConsent:  models.DiagnosticsDenied,
	}))

	engine.OnMessage(ctx, &botengine.BotMessage{
		ConversationID: env.conversationID,
		SenderID:       env.senderID,
		Content:        "sensitive content",
		MsgType:        "text",
		CreatedAt:      time.Now(),
	})

	time.Sleep(500 * time.Millisecond)

	logs, err := callLogRepo.FindAllByBotID(ctx, env.botID, 10, 0)
	require.NoError(t, err)
	require.Len(t, logs, 1)
	assert.Empty(t, logs[0].TriggerMessage, "trigger_message should be empty when consent denied")
	assert.Nil(t, logs[0].Trace, "trace should be nil when consent denied")
}
