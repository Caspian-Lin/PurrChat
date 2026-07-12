package botws

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

var ErrClosed = errors.New("bot websocket manager is closed")

const MaxReplayBatch = 500

type ResumeEntry struct {
	Seq     int64
	Payload []byte
}

type EventReplayer interface {
	FindUnacked(ctx context.Context, credentialID, botID uuid.UUID, afterSeq int64, limit int) ([]ResumeEntry, error)
}

type MetricsSnapshot struct {
	Accepted        uint64 `json:"accepted"`
	Active          int64  `json:"active"`
	Rejected        uint64 `json:"rejected"`
	MessagesRead    uint64 `json:"messages_read"`
	MessagesWritten uint64 `json:"messages_written"`
	ActionStarted   uint64 `json:"action_started"`
	ActionCompleted uint64 `json:"action_completed"`
	ActionRejected  uint64 `json:"action_rejected"`
	QueueOverflows  uint64 `json:"queue_overflows"`
	ProtocolErrors  uint64 `json:"protocol_errors"`
}

type metrics struct {
	accepted, rejected, read, written, actionStarted, actionCompleted, actionRejected, queueOverflows, protocolErrors atomic.Uint64
	active                                                                                                            atomic.Int64
}

type BotStatus struct {
	Online        bool       `json:"online"`
	Connections   int        `json:"connections"`
	LastHeartbeat *time.Time `json:"last_heartbeat,omitempty"`
	LastError     string     `json:"last_error,omitempty"`
}

type botState struct {
	lastHeartbeat time.Time
	lastError     string
}

type Manager struct {
	config      Config
	dispatcher  ActionDispatcher
	replayer    EventReplayer
	mu          sync.RWMutex
	connections map[uuid.UUID]map[*connection]struct{}
	credentials map[uuid.UUID]map[*connection]struct{}
	states      map[uuid.UUID]botState
	closed      bool
	metrics     metrics
	now         func() time.Time
}

func NewManager(config Config, dispatcher ActionDispatcher) *Manager {
	defaults := DefaultConfig()
	if config.MaxConnections <= 0 {
		config.MaxConnections = defaults.MaxConnections
	}
	if config.MaxBotConnections <= 0 {
		config.MaxBotConnections = defaults.MaxBotConnections
	}
	if config.MaxConcurrentActions <= 0 {
		config.MaxConcurrentActions = defaults.MaxConcurrentActions
	}
	if config.SendQueueSize <= 0 {
		config.SendQueueSize = defaults.SendQueueSize
	}
	if config.MaxFrameBytes <= 0 {
		config.MaxFrameBytes = defaults.MaxFrameBytes
	}
	if config.MaxMessageBytes <= 0 {
		config.MaxMessageBytes = defaults.MaxMessageBytes
	}
	if config.ReadTimeout <= 0 {
		config.ReadTimeout = defaults.ReadTimeout
	}
	if config.WriteTimeout <= 0 {
		config.WriteTimeout = defaults.WriteTimeout
	}
	if config.ActionTimeout <= 0 {
		config.ActionTimeout = defaults.ActionTimeout
	}
	if config.PingInterval <= 0 {
		config.PingInterval = defaults.PingInterval
	}
	if dispatcher == nil {
		dispatcher = RegistryDispatcher{}
	}
	return &Manager{config: config, dispatcher: dispatcher, connections: make(map[uuid.UUID]map[*connection]struct{}), credentials: make(map[uuid.UUID]map[*connection]struct{}), states: make(map[uuid.UUID]botState), now: time.Now}
}

func (m *Manager) SetReplayer(replayer EventReplayer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.replayer = replayer
}

func (m *Manager) register(c *connection) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return ErrClosed
	}
	if int(m.metrics.active.Load()) >= m.config.MaxConnections || len(m.connections[c.principal.BotID]) >= m.config.MaxBotConnections {
		m.metrics.rejected.Add(1)
		return errConnectionLimit
	}
	if m.connections[c.principal.BotID] == nil {
		m.connections[c.principal.BotID] = make(map[*connection]struct{})
	}
	if m.credentials[c.principal.CredentialID] == nil {
		m.credentials[c.principal.CredentialID] = make(map[*connection]struct{})
	}
	m.connections[c.principal.BotID][c] = struct{}{}
	m.credentials[c.principal.CredentialID][c] = struct{}{}
	m.metrics.accepted.Add(1)
	m.metrics.active.Add(1)
	return nil
}

func (m *Manager) unregister(c *connection) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.connections[c.principal.BotID][c]; !ok {
		return
	}
	delete(m.connections[c.principal.BotID], c)
	delete(m.credentials[c.principal.CredentialID], c)
	if len(m.connections[c.principal.BotID]) == 0 {
		delete(m.connections, c.principal.BotID)
	}
	if len(m.credentials[c.principal.CredentialID]) == 0 {
		delete(m.credentials, c.principal.CredentialID)
	}
	m.metrics.active.Add(-1)
}

func (m *Manager) PublishBotEvent(botID uuid.UUID, event any) int {
	payload, err := json.Marshal(event)
	if err != nil {
		return 0
	}
	m.mu.RLock()
	items := make([]*connection, 0, len(m.connections[botID]))
	for c := range m.connections[botID] {
		items = append(items, c)
	}
	m.mu.RUnlock()
	delivered := 0
	for _, c := range items {
		if c.enqueue(payload) {
			delivered++
		}
	}
	return delivered
}

func (m *Manager) ReplayConnection(ctx context.Context, c *connection, afterSeq int64) int {
	m.mu.RLock()
	replayer := m.replayer
	m.mu.RUnlock()
	if replayer == nil {
		return 0
	}
	limit := min(MaxReplayBatch, max(1, m.config.SendQueueSize-1))
	entries, err := replayer.FindUnacked(ctx, c.principal.CredentialID, c.principal.BotID, afterSeq, limit)
	if err != nil || len(entries) == 0 {
		return 0
	}
	delivered := 0
	for _, entry := range entries {
		payload := injectSeq(entry.Payload, entry.Seq)
		if c.enqueue(payload) {
			delivered++
		} else {
			break
		}
	}
	return delivered
}

func (m *Manager) DisconnectCredential(_ context.Context, id uuid.UUID) error {
	m.disconnect(m.byCredential(id), CloseCredentialInvalid, "credential rotated or revoked")
	return nil
}

func (m *Manager) DisconnectBot(_ context.Context, id uuid.UUID) error {
	m.disconnect(m.byBot(id), CloseBotUnavailable, "bot disabled or deleted")
	return nil
}

func (m *Manager) byCredential(id uuid.UUID) []*connection {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return cloneConnections(m.credentials[id])
}
func (m *Manager) byBot(id uuid.UUID) []*connection {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return cloneConnections(m.connections[id])
}
func cloneConnections(source map[*connection]struct{}) []*connection {
	out := make([]*connection, 0, len(source))
	for c := range source {
		out = append(out, c)
	}
	return out
}
func (m *Manager) disconnect(items []*connection, code int, reason string) {
	for _, c := range items {
		c.close(code, reason)
	}
}

func (m *Manager) Status(botID uuid.UUID) BotStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	state := m.states[botID]
	status := BotStatus{Connections: len(m.connections[botID]), LastError: state.lastError}
	status.Online = status.Connections > 0
	if !state.lastHeartbeat.IsZero() {
		value := state.lastHeartbeat
		status.LastHeartbeat = &value
	}
	return status
}

func (m *Manager) heartbeat(botID uuid.UUID) {
	m.mu.Lock()
	state := m.states[botID]
	state.lastHeartbeat = m.now().UTC()
	m.states[botID] = state
	m.mu.Unlock()
}
func (m *Manager) recordError(botID uuid.UUID, message string) {
	m.mu.Lock()
	state := m.states[botID]
	state.lastError = message
	m.states[botID] = state
	m.mu.Unlock()
}

func (m *Manager) Metrics() MetricsSnapshot {
	return MetricsSnapshot{m.metrics.accepted.Load(), m.metrics.active.Load(), m.metrics.rejected.Load(), m.metrics.read.Load(), m.metrics.written.Load(), m.metrics.actionStarted.Load(), m.metrics.actionCompleted.Load(), m.metrics.actionRejected.Load(), m.metrics.queueOverflows.Load(), m.metrics.protocolErrors.Load()}
}

func (m *Manager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	m.closed = true
	var items []*connection
	for _, set := range m.connections {
		items = append(items, cloneConnections(set)...)
	}
	m.mu.Unlock()
	m.disconnect(items, CloseServerShutdown, "server shutdown")
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for m.metrics.active.Load() > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
	return nil
}

func injectSeq(payload []byte, seq int64) []byte {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		return payload
	}
	seqBytes, err := json.Marshal(seq)
	if err != nil {
		return payload
	}
	raw["seq"] = seqBytes
	out, err := json.Marshal(raw)
	if err != nil {
		return payload
	}
	return out
}
