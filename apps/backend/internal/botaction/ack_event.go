package botaction

import (
	"context"
	"encoding/json"
	"errors"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/onebot"
	"purr-chat-server/internal/repository"

	"github.com/jackc/pgx/v5"
)

type ackEventParams struct {
	Seq     int64  `json:"seq"`
	EventID string `json:"event_id"`
}

func (d *Dispatcher) handleAckEvent(ctx context.Context, principal models.BotPrincipal, request onebot.ActionRequest) (json.RawMessage, error) {
	if d.outboxRepo == nil {
		return nil, onebot.NewError(onebot.RetCodeUnsupportedAction, "ack is not available", nil)
	}

	params, err := onebot.DecodeParams[ackEventParams](request)
	if err != nil {
		return nil, err
	}

	var ackSeq int64
	if params.Seq > 0 {
		ackSeq = params.Seq
	} else if params.EventID != "" {
		entry, err := d.outboxRepo.FindByEventID(ctx, principal.BotID, params.EventID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, onebot.NewError(onebot.RetCodeResourceNotFound, "event not found or expired", nil)
			}
			return nil, onebot.NewError(onebot.RetCodeInternal, "failed to look up event", err)
		}
		ackSeq = entry.Seq
	} else {
		return nil, onebot.NewError(onebot.RetCodeInvalidParams, "seq or event_id is required", nil)
	}

	marked, err := d.outboxRepo.AckUpTo(ctx, principal.CredentialID, principal.BotID, ackSeq)
	if err != nil {
		if errors.Is(err, repository.ErrAckSequenceAhead) {
			return nil, onebot.NewError(onebot.RetCodeInvalidParams, "seq has not been allocated", nil)
		}
		return nil, onebot.NewError(onebot.RetCodeInternal, "ack failed", err)
	}

	return marshalData(map[string]any{
		"ack_seq":    ackSeq,
		"marked_num": marked,
	})
}
