package services

import (
	"context"
	"sync/atomic"
	"time"

	"purr-chat-server/internal/repository"
	"purr-chat-server/pkg/logger"
)

const (
	cleanupInterval     = 5 * time.Minute
	ackedRetentionAge   = 10 * time.Minute
	unackedMaxRetention = 24 * time.Hour
)

type CleanupMetrics struct {
	DeletedAcked atomic.Int64
	ExpiredOld   atomic.Int64
	Runs         atomic.Int64
}

type CleanupSnapshot struct {
	DeletedAcked int64 `json:"deleted_acked"`
	ExpiredOld   int64 `json:"expired_old"`
	Runs         int64 `json:"runs"`
}

type BotEventRelay struct {
	outboxRepo repository.BotEventOutboxRepository
	metrics    CleanupMetrics
	interval   time.Duration
}

func NewBotEventRelay(outboxRepo repository.BotEventOutboxRepository) *BotEventRelay {
	return &BotEventRelay{
		outboxRepo: outboxRepo,
		interval:   cleanupInterval,
	}
}

func (r *BotEventRelay) MetricsSnapshot() CleanupSnapshot {
	return CleanupSnapshot{
		DeletedAcked: r.metrics.DeletedAcked.Load(),
		ExpiredOld:   r.metrics.ExpiredOld.Load(),
		Runs:         r.metrics.Runs.Load(),
	}
}

func (r *BotEventRelay) Start(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.run(ctx)
		}
	}
}

func (r *BotEventRelay) run(ctx context.Context) {
	r.metrics.Runs.Add(1)
	deleted, err := r.outboxRepo.DeleteAcked(ctx, ackedRetentionAge)
	if err != nil {
		logger.ErrorfWithCaller("[BotEventRelay] Failed to delete acked events: %v", err)
	} else if deleted > 0 {
		r.metrics.DeletedAcked.Add(deleted)
		logger.InfofWithCaller("[BotEventRelay] Deleted %d fully acknowledged events", deleted)
	}

	expired, err := r.outboxRepo.ExpireOld(ctx, unackedMaxRetention)
	if err != nil {
		logger.ErrorfWithCaller("[BotEventRelay] Failed to expire old events: %v", err)
	} else if expired > 0 {
		r.metrics.ExpiredOld.Add(expired)
		logger.ErrorfWithCaller("[BotEventRelay] Expired %d unacked events older than %s", expired, unackedMaxRetention)
	}
}
