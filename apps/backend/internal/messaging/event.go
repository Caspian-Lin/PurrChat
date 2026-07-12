package messaging

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"purr-chat-server/internal/models"
	"purr-chat-server/pkg/logger"

	"github.com/google/uuid"
)

// ActorType 标识消息发送者类型
type ActorType string

const (
	ActorUser   ActorType = "user"
	ActorBot    ActorType = "bot"
	ActorSystem ActorType = "system"
)

// MessageSource 标识消息来源通道
type MessageSource string

const (
	SourceUser     MessageSource = "user"
	SourceWorkflow MessageSource = "workflow"
	SourceExternal MessageSource = "external"
	SourceSystem   MessageSource = "system"
)

// MessageCreatedEvent 消息持久化后发布的事件
type MessageCreatedEvent struct {
	Message          *models.Message
	ActorType        ActorType
	Source           MessageSource
	SenderName       string
	MemberIDs        []uuid.UUID
	ConversationType models.ConversationType
	TriggerMessageID *uuid.UUID
	RunID            string
}

// IsBotSent 判断是否为 Bot 发送的消息（用于防回复环）
func (e *MessageCreatedEvent) IsBotSent() bool {
	return e.ActorType == ActorBot
}

// IsSystem 判断是否为系统消息
func (e *MessageCreatedEvent) IsSystem() bool {
	return e.ActorType == ActorSystem
}

// ShouldTriggerBots 判断该事件是否应触发 Bot 处理
// 系统消息和 Bot 发送的消息不触发其他 Bot
func (e *MessageCreatedEvent) ShouldTriggerBots() bool {
	if e.IsBotSent() || e.IsSystem() {
		return false
	}
	if e.Message == nil {
		return false
	}
	return e.Message.MsgType == models.MsgTypeText ||
		e.Message.MsgType == models.MsgTypeImage ||
		e.Message.MsgType == models.MsgTypeFile
}

// MessageEventSink 消息事件订阅者接口
type MessageEventSink interface {
	OnMessageCreated(ctx context.Context, event *MessageCreatedEvent) error
}

// BotMessageSender 统一 Bot 消息发送入口
// 由 MessageService 实现，BotEngine 通过此接口发送回复，
// 确保所有 Bot 消息走统一的校验、持久化和广播管线。
type BotMessageSender interface {
	SendBotMessage(ctx context.Context, req *BotSendRequest) (*models.Message, error)
}

// BotSendRequest Bot 发送消息请求
type BotSendRequest struct {
	BotID            uuid.UUID
	ConversationID   uuid.UUID
	Content          string
	MsgType          string
	Source           MessageSource
	RunID            string
	TriggerMessageID *uuid.UUID
}

// SinkMetrics 单个 sink 的运行指标（线程安全）
type SinkMetrics struct {
	Invoked   atomic.Int64
	Succeeded atomic.Int64
	Failed    atomic.Int64
	TimedOut  atomic.Int64
}

// Snapshot 返回当前值的快照
func (m *SinkMetrics) Snapshot() (invoked, succeeded, failed, timedOut int64) {
	return m.Invoked.Load(), m.Succeeded.Load(), m.Failed.Load(), m.TimedOut.Load()
}

// registeredSink 已注册的 sink 及其配置
type registeredSink struct {
	name    string
	sink    MessageEventSink
	metrics *SinkMetrics
}

// Publisher 消息事件发布器
// 将 MessageCreatedEvent fan-out 到所有已注册的 sink，
// 每个 sink 有独立的超时和错误隔离，单个 sink 失败不影响其他 sink 或已提交消息。
type Publisher struct {
	mu          sync.RWMutex
	sinks       []registeredSink
	sinkTimeout time.Duration
}

// NewPublisher 创建发布器
func NewPublisher(sinkTimeout time.Duration) *Publisher {
	if sinkTimeout <= 0 {
		sinkTimeout = 10 * time.Second
	}
	return &Publisher{
		sinkTimeout: sinkTimeout,
	}
}

// RegisterSink 注册一个 sink
func (p *Publisher) RegisterSink(name string, sink MessageEventSink) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sinks = append(p.sinks, registeredSink{
		name:    name,
		sink:    sink,
		metrics: &SinkMetrics{},
	})
}

// Publish 将事件 fan-out 到所有已注册的 sink
// 每个 sink 在独立 goroutine 中执行，有独立超时，
// Publisher 等待所有 sink 完成（或超时）后返回。
func (p *Publisher) Publish(ctx context.Context, event *MessageCreatedEvent) {
	p.mu.RLock()
	sinks := make([]registeredSink, len(p.sinks))
	copy(sinks, p.sinks)
	p.mu.RUnlock()

	if len(sinks) == 0 {
		return
	}

	var wg sync.WaitGroup
	for _, rs := range sinks {
		wg.Add(1)
		go func(rs registeredSink) {
			defer wg.Done()
			p.invokeSink(ctx, rs, event)
		}(rs)
	}
	wg.Wait()
}

// invokeSink 执行单个 sink，带超时和错误隔离
func (p *Publisher) invokeSink(parent context.Context, rs registeredSink, event *MessageCreatedEvent) {
	rs.metrics.Invoked.Add(1)

	ctx, cancel := context.WithTimeout(parent, p.sinkTimeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- errPanic(r)
			}
		}()
		done <- rs.sink.OnMessageCreated(ctx, event)
	}()

	select {
	case err := <-done:
		if err != nil {
			rs.metrics.Failed.Add(1)
			logger.ErrorfWithCaller("[MessagePublisher] sink=%s failed: %v", rs.name, err)
		} else {
			rs.metrics.Succeeded.Add(1)
		}
	case <-ctx.Done():
		rs.metrics.TimedOut.Add(1)
		logger.ErrorfWithCaller("[MessagePublisher] sink=%s timed out after %s", rs.name, p.sinkTimeout)
	}
}

// Metrics 返回所有 sink 的指标快照
func (p *Publisher) Metrics() map[string]SinkMetricsSnapshot {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make(map[string]SinkMetricsSnapshot, len(p.sinks))
	for _, rs := range p.sinks {
		invoked, succeeded, failed, timedOut := rs.metrics.Snapshot()
		out[rs.name] = SinkMetricsSnapshot{
			Invoked:   invoked,
			Succeeded: succeeded,
			Failed:    failed,
			TimedOut:  timedOut,
		}
	}
	return out
}

// SinkMetricsSnapshot sink 指标快照（值类型）
type SinkMetricsSnapshot struct {
	Invoked   int64
	Succeeded int64
	Failed    int64
	TimedOut  int64
}
