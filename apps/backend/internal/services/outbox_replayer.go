package services

import (
	"context"

	"purr-chat-server/internal/botws"
	"purr-chat-server/internal/repository"

	"github.com/google/uuid"
)

type OutboxReplayer struct {
	repo repository.BotEventOutboxRepository
}

func NewOutboxReplayer(repo repository.BotEventOutboxRepository) *OutboxReplayer {
	return &OutboxReplayer{repo: repo}
}

func (r *OutboxReplayer) FindUnacked(ctx context.Context, credentialID, botID uuid.UUID, afterSeq int64, limit int) ([]botws.ResumeEntry, error) {
	entries, err := r.repo.FindUnacked(ctx, credentialID, botID, afterSeq, limit)
	if err != nil {
		return nil, err
	}
	out := make([]botws.ResumeEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, botws.ResumeEntry{Seq: e.Seq, Payload: e.Payload})
	}
	return out, nil
}
