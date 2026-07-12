package services

import (
	"context"
	"testing"
	"time"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/onebot"
	"purr-chat-server/pkg/logger"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type backlogOutboxRepository struct{}

func (backlogOutboxRepository) Append(context.Context, uuid.UUID, string, []byte) (int64, error) {
	panic("Append must not be called after reaching the backlog limit")
}
func (backlogOutboxRepository) FindUnacked(context.Context, uuid.UUID, uuid.UUID, int64, int) ([]*models.BotEventOutbox, error) {
	return nil, nil
}
func (backlogOutboxRepository) FindByEventID(context.Context, uuid.UUID, string) (*models.BotEventOutbox, error) {
	return nil, nil
}
func (backlogOutboxRepository) CountUnacked(context.Context, uuid.UUID) (int64, error) {
	return maxBacklogPerBot, nil
}
func (backlogOutboxRepository) AckUpTo(context.Context, uuid.UUID, uuid.UUID, int64) (int64, error) {
	return 0, nil
}
func (backlogOutboxRepository) GetAckState(context.Context, uuid.UUID, uuid.UUID) (int64, error) {
	return 0, nil
}
func (backlogOutboxRepository) DeleteAcked(context.Context, time.Duration) (int64, error) {
	return 0, nil
}
func (backlogOutboxRepository) ExpireOld(context.Context, time.Duration) (int64, error) {
	return 0, nil
}

type countingEventPublisher struct {
	calls int
}

func (p *countingEventPublisher) PublishBotEvent(uuid.UUID, any) int {
	p.calls++
	return 1
}

func TestReliableEventPublisherDropsAtBacklogLimit(t *testing.T) {
	logger.Init()
	delegate := &countingEventPublisher{}
	publisher := NewReliableEventPublisher(backlogOutboxRepository{}, delegate)

	delivered := publisher.PublishBotEvent(uuid.New(), onebot.Event{EventID: onebot.GenerateEventID()})
	require.Zero(t, delivered)
	require.Zero(t, delegate.calls)
	require.Equal(t, int64(1), publisher.MetricsSnapshot().BacklogHit)
}
