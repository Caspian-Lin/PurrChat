//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

const workflowDocument = `{
  "apiVersion":"purrchat.ai/v1alpha1",
  "kind":"BotWorkflow",
  "metadata":{"name":"E2E Bot","revision":0},
  "spec":{
    "trigger":{"type":"rule","rules":[]},
    "nodes":[
      {"id":"trigger","type":"trigger","name":"Trigger","config":{}},
      {"id":"reply","type":"reply","name":"Reply","config":{"template":"E2E reply: ${input.text}"}},
      {"id":"end","type":"end","name":"End","config":{}}
    ],
    "connections":[
      {"id":"c1","sourceNodeId":"trigger","sourcePortId":"out_exec","targetNodeId":"reply","targetPortId":"in_exec"},
      {"id":"c2","sourceNodeId":"reply","sourcePortId":"out_exec","targetNodeId":"end","targetPortId":"in_exec"}
    ],
    "endConditions":[{"type":"max_rounds","value":5}]
  }
}`

type testProcess struct {
	cmd     *exec.Cmd
	logFile *os.File
}

type harness struct {
	t             *testing.T
	backendDir    string
	backendBinary string
	engineEntry   string
	backendPort   int
	enginePort    int
	baseURL       string
	backend       *testProcess
	engine        *testProcess
}

type apiClient struct {
	baseURL string
	token   string
	http    *http.Client
}

type user struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type bot struct {
	ID string `json:"id"`
}

type conversation struct {
	ID string `json:"id"`
}

type message struct {
	ID             string  `json:"id"`
	ConversationID string  `json:"conversation_id"`
	SenderID       string  `json:"sender_id"`
	Content        string  `json:"content"`
	BotID          *string `json:"bot_id,omitempty"`
}

type installation struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type callLog struct {
	RunID            string          `json:"run_id"`
	TriggerMessageID *string         `json:"trigger_message_id"`
	ReplyMessageID   *string         `json:"reply_message_id"`
	WorkflowRevision *int            `json:"workflow_revision"`
	RunStatus        string          `json:"run_status"`
	ErrorType        string          `json:"error_type"`
	Trace            json.RawMessage `json:"trace"`
}

func TestBotFullStackE2E(t *testing.T) {
	h := newHarness(t)
	t.Cleanup(h.stopAll)
	h.startEngine()
	h.startBackend()

	stamp := time.Now().UnixNano()
	suffix := fmt.Sprintf("%08x", uint32(stamp))
	ownerClient, owner := register(t, h.baseURL, "owner_"+suffix, fmt.Sprintf("owner_%d@example.com", stamp))
	memberClient, member := register(t, h.baseURL, "member_"+suffix, fmt.Sprintf("member_%d@example.com", stamp))
	_, third := register(t, h.baseURL, "third_"+suffix, fmt.Sprintf("third_%d@example.com", stamp))

	createdBot := createAndPublishBot(t, ownerClient, suffix, workflowDocument)

	t.Run("private install to reply to trace to websocket to history", func(t *testing.T) {
		var direct conversation
		memberClient.doJSON(t, http.MethodPost, "/api/bots/"+createdBot.ID+"/conversation", nil, http.StatusOK, &direct)
		inst := findInstallation(t, memberClient, "user", member.ID, createdBot.ID)
		memberClient.doJSON(t, http.MethodPatch, "/api/installations/"+inst.ID, map[string]any{
			"diagnostics_consent": "granted",
		}, http.StatusOK, nil)

		ws := dialUserWebSocket(t, h.baseURL, memberClient.token)
		defer ws.Close()

		trigger := sendMessage(t, memberClient, direct.ID, "private-one")
		reply := waitForBotReply(t, ws, createdBot.ID, "E2E reply: private-one")
		log := waitForCallLog(t, ownerClient, createdBot.ID, trigger.ID, "completed")
		assertCorrelatedRun(t, log, trigger.ID, reply.ID)
		assertHistoryContains(t, memberClient, direct.ID, reply.ID)

		memberClient.doJSON(t, http.MethodPatch, "/api/installations/"+inst.ID, map[string]any{"status": "paused"}, http.StatusOK, nil)
		pausedTrigger := sendMessage(t, memberClient, direct.ID, "paused")
		assertNoCallLog(t, ownerClient, createdBot.ID, pausedTrigger.ID, 800*time.Millisecond)

		memberClient.doJSON(t, http.MethodPatch, "/api/installations/"+inst.ID, map[string]any{"status": "active"}, http.StatusOK, nil)
		resumedTrigger := sendMessage(t, memberClient, direct.ID, "resumed")
		resumedReply := waitForBotReply(t, ws, createdBot.ID, "E2E reply: resumed")
		resumedLog := waitForCallLog(t, ownerClient, createdBot.ID, resumedTrigger.ID, "completed")
		assertCorrelatedRun(t, resumedLog, resumedTrigger.ID, resumedReply.ID)

		h.stopBackend()
		h.startBackend()
		assertHistoryContains(t, memberClient, direct.ID, resumedReply.ID)

		wsAfterRestart := dialUserWebSocket(t, h.baseURL, memberClient.token)
		defer wsAfterRestart.Close()
		restartTrigger := sendMessage(t, memberClient, direct.ID, "backend-restarted")
		restartReply := waitForBotReply(t, wsAfterRestart, createdBot.ID, "E2E reply: backend-restarted")
		restartLog := waitForCallLog(t, ownerClient, createdBot.ID, restartTrigger.ID, "completed")
		assertCorrelatedRun(t, restartLog, restartTrigger.ID, restartReply.ID)

		h.stopEngine()
		unavailableTrigger := sendMessage(t, memberClient, direct.ID, "engine-down")
		unavailableLog := waitForCallLog(t, ownerClient, createdBot.ID, unavailableTrigger.ID, "error")
		require.Equal(t, "ts_unavailable", unavailableLog.ErrorType)
		h.startEngine()

		recoveredTrigger := sendMessage(t, memberClient, direct.ID, "engine-recovered")
		recoveredReply := waitForBotReply(t, wsAfterRestart, createdBot.ID, "E2E reply: engine-recovered")
		recoveredLog := waitForCallLog(t, ownerClient, createdBot.ID, recoveredTrigger.ID, "completed")
		assertCorrelatedRun(t, recoveredLog, recoveredTrigger.ID, recoveredReply.ID)

		memberClient.doJSON(t, http.MethodDelete, "/api/installations/"+inst.ID, nil, http.StatusOK, nil)
		uninstalledTrigger := sendMessage(t, memberClient, direct.ID, "uninstalled")
		assertNoCallLog(t, ownerClient, createdBot.ID, uninstalledTrigger.ID, 800*time.Millisecond)
	})

	t.Run("group install and disabled gate", func(t *testing.T) {
		var group conversation
		memberClient.doJSON(t, http.MethodPost, "/api/conversations/group", map[string]any{
			"name": "Bot E2E group", "members": []string{owner.ID, third.ID},
		}, http.StatusOK, &group)

		var installEnvelope struct {
			Installation installation `json:"installation"`
		}
		memberClient.doJSON(t, http.MethodPost, "/api/bots/"+createdBot.ID+"/installations", map[string]any{
			"target_type": "conversation", "target_id": group.ID, "diagnostics_consent": "granted",
		}, http.StatusOK, &installEnvelope)

		ws := dialUserWebSocket(t, h.baseURL, memberClient.token)
		defer ws.Close()
		trigger := sendMessage(t, memberClient, group.ID, "group-one")
		reply := waitForBotReply(t, ws, createdBot.ID, "E2E reply: group-one")
		log := waitForCallLog(t, ownerClient, createdBot.ID, trigger.ID, "completed")
		assertCorrelatedRun(t, log, trigger.ID, reply.ID)

		ownerClient.doJSON(t, http.MethodPut, "/api/bots/"+createdBot.ID, map[string]any{"status": "disabled"}, http.StatusOK, nil)
		disabledTrigger := sendMessage(t, memberClient, group.ID, "disabled")
		assertNoCallLog(t, ownerClient, createdBot.ID, disabledTrigger.ID, 800*time.Millisecond)
		ownerClient.doJSON(t, http.MethodPut, "/api/bots/"+createdBot.ID, map[string]any{"status": "active"}, http.StatusOK, nil)
	})

	t.Run("external tool uses local fake provider", func(t *testing.T) {
		var calls atomic.Int32
		provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			w.Header().Set("Content-Type", "text/plain")
			_, _ = io.WriteString(w, "fake-provider-ok")
		}))
		defer provider.Close()

		toolBot := createAndPublishBot(t, ownerClient, suffix+"tool", toolWorkflowDocument(provider.URL))
		var direct conversation
		memberClient.doJSON(t, http.MethodPost, "/api/bots/"+toolBot.ID+"/conversation", nil, http.StatusOK, &direct)

		ws := dialUserWebSocket(t, h.baseURL, memberClient.token)
		defer ws.Close()
		trigger := sendMessage(t, memberClient, direct.ID, "call-local-provider")
		reply := waitForBotReply(t, ws, toolBot.ID, "fake-provider-ok")
		log := waitForCallLog(t, ownerClient, toolBot.ID, trigger.ID, "completed")
		assertCorrelatedRun(t, log, trigger.ID, reply.ID)
		require.Equal(t, int32(1), calls.Load())
	})
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	backendBinary := os.Getenv("E2E_BACKEND_BIN")
	engineEntry := os.Getenv("E2E_BOT_ENGINE_ENTRY")
	require.NotEmpty(t, backendBinary, "E2E_BACKEND_BIN must point to the built Go backend")
	require.NotEmpty(t, engineEntry, "E2E_BOT_ENGINE_ENTRY must point to bot-engine dist/index.js")

	backendDir, err := filepath.Abs("..")
	require.NoError(t, err)
	backendPort, enginePort := freePort(t), freePort(t)
	return &harness{
		t:             t,
		backendDir:    backendDir,
		backendBinary: backendBinary,
		engineEntry:   engineEntry,
		backendPort:   backendPort,
		enginePort:    enginePort,
		baseURL:       fmt.Sprintf("http://127.0.0.1:%d", backendPort),
	}
}

func (h *harness) startEngine() {
	h.t.Helper()
	if h.engine != nil {
		return
	}
	h.engine = h.startProcess("bot-engine", "node", []string{h.engineEntry}, map[string]string{
		"BOT_ENGINE_PORT":          fmt.Sprint(h.enginePort),
		"BOT_ENGINE_SHARED_SECRET": "bot-e2e-secret",
	})
	waitForHealth(h.t, fmt.Sprintf("http://127.0.0.1:%d/health", h.enginePort))
}

func (h *harness) startBackend() {
	h.t.Helper()
	if h.backend != nil {
		return
	}
	h.backend = h.startProcess("backend", h.backendBinary, nil, map[string]string{
		"PORT":                       fmt.Sprint(h.backendPort),
		"GIN_MODE":                   "release",
		"JWT_SECRET":                 "bot-e2e-jwt-secret",
		"TURNSTILE_ENABLED":          "false",
		"BOT_ENGINE_URL":             fmt.Sprintf("http://127.0.0.1:%d", h.enginePort),
		"BOT_ENGINE_SHARED_SECRET":   "bot-e2e-secret",
		"LOG_DIR":                    h.t.TempDir(),
		"RATE_LIMIT_GLOBAL_RPS":      "1000",
		"RATE_LIMIT_GLOBAL_BURST":    "1000",
		"RATE_LIMIT_AUTH_RPS":        "1000",
		"RATE_LIMIT_AUTH_BURST":      "1000",
		"RATE_LIMIT_USER_RPS":        "1000",
		"RATE_LIMIT_USER_BURST":      "1000",
		"RATE_LIMIT_SENSITIVE_RPS":   "1000",
		"RATE_LIMIT_SENSITIVE_BURST": "1000",
	})
	waitForHealth(h.t, h.baseURL+"/health")
}

func (h *harness) startProcess(name, binary string, args []string, extraEnv map[string]string) *testProcess {
	h.t.Helper()
	logFile, err := os.CreateTemp(h.t.TempDir(), name+"-*.log")
	require.NoError(h.t, err)
	cmd := exec.Command(binary, args...)
	cmd.Dir = h.backendDir
	cmd.Stdout, cmd.Stderr = logFile, logFile
	cmd.Env = mergedEnv(extraEnv)
	require.NoError(h.t, cmd.Start(), "start %s", name)
	return &testProcess{cmd: cmd, logFile: logFile}
}

func mergedEnv(overrides map[string]string) []string {
	result := make([]string, 0, len(os.Environ())+len(overrides))
	for _, item := range os.Environ() {
		key, _, _ := strings.Cut(item, "=")
		if _, replaced := overrides[key]; !replaced {
			result = append(result, item)
		}
	}
	for key, value := range overrides {
		result = append(result, key+"="+value)
	}
	return result
}

func (h *harness) stopBackend() { h.stopProcess("backend", &h.backend) }
func (h *harness) stopEngine()  { h.stopProcess("bot-engine", &h.engine) }
func (h *harness) stopAll()     { h.stopBackend(); h.stopEngine() }

func (h *harness) stopProcess(name string, process **testProcess) {
	p := *process
	if p == nil {
		return
	}
	_ = p.cmd.Process.Signal(syscall.SIGTERM)
	done := make(chan error, 1)
	go func() { done <- p.cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		_ = p.cmd.Process.Kill()
		<-done
	}
	_ = p.logFile.Close()
	if h.t.Failed() {
		content, _ := os.ReadFile(p.logFile.Name())
		h.t.Logf("%s log:\n%s", name, content)
	}
	*process = nil
}

func freePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func waitForHealth(t *testing.T, endpoint string) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	client := &http.Client{Timeout: 500 * time.Millisecond}
	var lastErr error
	for time.Now().Before(deadline) {
		response, err := client.Get(endpoint) // #nosec G107 -- fixed local E2E endpoint
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return
			}
			lastErr = fmt.Errorf("health returned %d", response.StatusCode)
		} else {
			lastErr = err
		}
		time.Sleep(50 * time.Millisecond)
	}
	require.NoError(t, lastErr, "service failed to become healthy: %s", endpoint)
}

func register(t *testing.T, baseURL, username, email string) (*apiClient, user) {
	t.Helper()
	client := &apiClient{baseURL: baseURL, http: &http.Client{Timeout: 5 * time.Second}}
	var auth struct {
		Token string `json:"token"`
		User  user   `json:"user"`
	}
	client.doJSON(t, http.MethodPost, "/api/register", map[string]any{
		"username": username, "email": email, "phone": fmt.Sprint(time.Now().UnixNano()), "password": "e2e-password",
	}, http.StatusOK, &auth)
	client.token = auth.Token
	require.NotEmpty(t, client.token)
	require.NotEmpty(t, auth.User.ID)
	return client, auth.User
}

func createAndPublishBot(t *testing.T, owner *apiClient, suffix, document string) bot {
	t.Helper()
	var created bot
	owner.doJSON(t, http.MethodPost, "/api/bots", map[string]any{
		"name": "E2E Bot " + suffix, "description": "black-box test bot", "discoverability": "listed",
	}, http.StatusOK, &created)

	var draft struct {
		Revision int `json:"revision"`
	}
	owner.doJSON(t, http.MethodPut, "/api/bots/"+created.ID+"/workflow", map[string]any{
		"revision": 0, "document": json.RawMessage(document),
	}, http.StatusOK, &draft)
	require.Equal(t, 1, draft.Revision)
	owner.doJSON(t, http.MethodPost, "/api/bots/"+created.ID+"/workflow/publish", map[string]any{
		"revision": draft.Revision,
	}, http.StatusOK, nil)
	return created
}

func toolWorkflowDocument(providerURL string) string {
	encodedURL, _ := json.Marshal(providerURL)
	return fmt.Sprintf(`{
  "apiVersion":"purrchat.ai/v1alpha1",
  "kind":"BotWorkflow",
  "metadata":{"name":"E2E Tool Bot","revision":0},
  "spec":{
    "trigger":{"type":"rule","rules":[]},
    "nodes":[
      {"id":"trigger","type":"trigger","name":"Trigger","config":{}},
      {"id":"tool","type":"tool","name":"Tool","config":{"method":"GET","url":%s,"headers":{},"timeout":2000}},
      {"id":"reply","type":"reply","name":"Reply","config":{"template":"$tool:out_output"}},
      {"id":"end","type":"end","name":"End","config":{}}
    ],
    "connections":[
      {"id":"c1","sourceNodeId":"trigger","sourcePortId":"out_exec","targetNodeId":"tool","targetPortId":"in_exec"},
      {"id":"c2","sourceNodeId":"tool","sourcePortId":"out_exec","targetNodeId":"reply","targetPortId":"in_exec"},
      {"id":"c3","sourceNodeId":"reply","sourcePortId":"out_exec","targetNodeId":"end","targetPortId":"in_exec"}
    ],
    "endConditions":[{"type":"max_rounds","value":5}]
  }
}`, encodedURL)
}

func findInstallation(t *testing.T, client *apiClient, targetType, targetID, botID string) installation {
	t.Helper()
	var data struct {
		Installations []struct {
			installation
			AppID string `json:"app_id"`
		} `json:"installations"`
	}
	client.doJSON(t, http.MethodGet, "/api/installations?target_type="+url.QueryEscape(targetType)+"&target_id="+url.QueryEscape(targetID), nil, http.StatusOK, &data)
	for _, item := range data.Installations {
		if item.AppID == botID {
			return item.installation
		}
	}
	t.Fatalf("installation for bot %s not found", botID)
	return installation{}
}

func sendMessage(t *testing.T, client *apiClient, conversationID, content string) message {
	t.Helper()
	var sent message
	client.doJSON(t, http.MethodPost, "/api/messages", map[string]any{
		"conversation_id":   conversationID,
		"content":           content,
		"msg_type":          "text",
		"client_message_id": uuid.NewString(),
	}, http.StatusOK, &sent)
	return sent
}

func dialUserWebSocket(t *testing.T, baseURL, token string) *websocket.Conn {
	t.Helper()
	endpoint := "ws" + strings.TrimPrefix(baseURL, "http") + "/api/ws?token=" + url.QueryEscape(token)
	conn, response, err := websocket.DefaultDialer.Dial(endpoint, nil)
	if response != nil {
		defer response.Body.Close()
	}
	require.NoError(t, err)
	return conn
}

func waitForBotReply(t *testing.T, conn *websocket.Conn, botID, content string) message {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		require.NoError(t, conn.SetReadDeadline(deadline))
		_, raw, err := conn.ReadMessage()
		require.NoError(t, err)
		var envelope struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}
		if json.Unmarshal(raw, &envelope) != nil || envelope.Type != "new_message" {
			continue
		}
		var candidate message
		if json.Unmarshal(envelope.Data, &candidate) != nil {
			continue
		}
		if candidate.BotID != nil && *candidate.BotID == botID && candidate.Content == content {
			return candidate
		}
	}
	t.Fatalf("timed out waiting for bot %s reply %q", botID, content)
	return message{}
}

func waitForCallLog(t *testing.T, owner *apiClient, botID, triggerID, status string) callLog {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		logs := listCallLogs(t, owner, botID)
		for _, log := range logs {
			if log.TriggerMessageID != nil && *log.TriggerMessageID == triggerID {
				require.Equal(t, status, log.RunStatus)
				return log
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for call log trigger_message_id=%s", triggerID)
	return callLog{}
}

func assertNoCallLog(t *testing.T, owner *apiClient, botID, triggerID string, duration time.Duration) {
	t.Helper()
	deadline := time.Now().Add(duration)
	for time.Now().Before(deadline) {
		for _, log := range listCallLogs(t, owner, botID) {
			if log.TriggerMessageID != nil && *log.TriggerMessageID == triggerID {
				t.Fatalf("unexpected execution for gated trigger %s: %+v", triggerID, log)
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func listCallLogs(t *testing.T, owner *apiClient, botID string) []callLog {
	t.Helper()
	var data struct {
		Logs []callLog `json:"logs"`
	}
	owner.doJSON(t, http.MethodGet, "/api/bots/"+botID+"/call-logs?limit=100", nil, http.StatusOK, &data)
	return data.Logs
}

func assertCorrelatedRun(t *testing.T, log callLog, triggerID, replyID string) {
	t.Helper()
	require.NotEmpty(t, log.RunID)
	require.NotNil(t, log.TriggerMessageID)
	require.Equal(t, triggerID, *log.TriggerMessageID)
	require.NotNil(t, log.ReplyMessageID)
	require.Equal(t, replyID, *log.ReplyMessageID)
	require.NotNil(t, log.WorkflowRevision)
	require.Equal(t, 1, *log.WorkflowRevision)
	require.NotEmpty(t, log.Trace)
	var trace struct {
		Status string `json:"status"`
	}
	require.NoError(t, json.Unmarshal(log.Trace, &trace))
	require.Equal(t, "completed", trace.Status)
}

func assertHistoryContains(t *testing.T, client *apiClient, conversationID, messageID string) {
	t.Helper()
	var messages []message
	client.doJSON(t, http.MethodGet, "/api/messages?conversation_id="+url.QueryEscape(conversationID)+"&limit=100", nil, http.StatusOK, &messages)
	for _, item := range messages {
		if item.ID == messageID {
			return
		}
	}
	t.Fatalf("message %s not found in conversation history", messageID)
}

func (c *apiClient) doJSON(t *testing.T, method, path string, body any, expectedStatus int, result any) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(raw)
	}
	request, err := http.NewRequestWithContext(context.Background(), method, c.baseURL+path, reader)
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		request.Header.Set("Authorization", "Bearer "+c.token)
	}
	response, err := c.http.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	raw, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	if response.StatusCode != expectedStatus {
		t.Fatalf("%s %s returned %d, expected %d: %s", method, path, response.StatusCode, expectedStatus, raw)
	}
	if result == nil {
		return
	}
	var envelope struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err == nil && envelope.Data != nil {
		require.True(t, envelope.Success, "API envelope reported failure: %s", raw)
		require.NoError(t, json.Unmarshal(envelope.Data, result), "decode API data: %s", raw)
		return
	}
	require.NoError(t, json.Unmarshal(raw, result), "decode raw response: %s", raw)
}

func TestMain(m *testing.M) {
	if os.Getenv("E2E_BACKEND_BIN") == "" || os.Getenv("E2E_BOT_ENGINE_ENTRY") == "" {
		fmt.Fprintln(os.Stderr, "bot E2E requires E2E_BACKEND_BIN and E2E_BOT_ENGINE_ENTRY")
		os.Exit(2)
	}
	os.Exit(m.Run())
}
