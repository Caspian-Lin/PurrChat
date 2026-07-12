package services

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"time"

	"purr-chat-server/internal/onebot"
	"purr-chat-server/internal/repository"
	"purr-chat-server/pkg/logger"

	"github.com/google/uuid"
)

const maxBacklogPerBot = 10000

type ReliablePublishMetrics struct {
	Persisted  atomic.Int64
	PushFailed atomic.Int64
	BacklogHit atomic.Int64
}

type ReliablePublishSnapshot struct {
	Persisted  int64 `json:"persisted"`
	PushFailed int64 `json:"push_failed"`
	BacklogHit int64 `json:"backlog_hit"`
}

type ReliableEventPublisher struct {
	outboxRepo repository.BotEventOutboxRepository
	delegate   BotEventPublisher
	metrics    ReliablePublishMetrics
	now        func() time.Time
}

func NewReliableEventPublisher(
	outboxRepo repository.BotEventOutboxRepository,
	delegate BotEventPublisher,
) *ReliableEventPublisher {
	return &ReliableEventPublisher{
		outboxRepo: outboxRepo,
		delegate:   delegate,
		now:        time.Now,
	}
}

func (p *ReliableEventPublisher) MetricsSnapshot() ReliablePublishSnapshot {
	return ReliablePublishSnapshot{
		Persisted:  p.metrics.Persisted.Load(),
		PushFailed: p.metrics.PushFailed.Load(),
		BacklogHit: p.metrics.BacklogHit.Load(),
	}
}

func (p *ReliableEventPublisher) PublishBotEvent(botID uuid.UUID, event any) int {
	onebotEvent, ok := event.(onebot.Event)
	if !ok {
		if p.delegate != nil {
			return p.delegate.PublishBotEvent(botID, event)
		}
		return 0
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := p.outboxRepo.CountUnacked(ctx, botID)
	if err != nil {
		logger.ErrorfWithCaller("[ReliablePublisher] Failed to check backlog for bot %s: %v", botID, err)
	} else if count >= maxBacklogPerBot {
		p.metrics.BacklogHit.Add(1)
		logger.ErrorfWithCaller("[ReliablePublisher] Bot %s backlog reached %d, dropping event %s", botID, count, onebotEvent.EventID)
		return 0
	}

	seq, err := p.outboxRepo.Append(ctx, botID, onebotEvent.EventID, mustMarshalEvent(onebotEvent))
	if err != nil {
		logger.ErrorfWithCaller("[ReliablePublisher] Failed to persist event %s for bot %s: %v", onebotEvent.EventID, botID, err)
		p.metrics.PushFailed.Add(1)
		return 0
	}

	onebotEvent.Seq = seq
	p.metrics.Persisted.Add(1)

	if p.delegate == nil {
		return 0
	}
	return p.delegate.PublishBotEvent(botID, onebotEvent)
}

func mustMarshalEvent(event onebot.Event) []byte {
	payload, err := json.Marshal(event)
	if err != nil {
		return []byte(`{}`)
	}
	return payload
}
