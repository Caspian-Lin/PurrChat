package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"purr-chat-server/internal/models"
	"purr-chat-server/pkg/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrAckSequenceAhead = errors.New("ack sequence has not been allocated")

type BotEventOutboxRepository interface {
	Append(ctx context.Context, botID uuid.UUID, eventID string, payload []byte) (int64, error)
	FindUnacked(ctx context.Context, credentialID, botID uuid.UUID, afterSeq int64, limit int) ([]*models.BotEventOutbox, error)
	FindByEventID(ctx context.Context, botID uuid.UUID, eventID string) (*models.BotEventOutbox, error)
	CountUnacked(ctx context.Context, botID uuid.UUID) (int64, error)
	AckUpTo(ctx context.Context, credentialID, botID uuid.UUID, seq int64) (int64, error)
	GetAckState(ctx context.Context, credentialID, botID uuid.UUID) (int64, error)
	DeleteAcked(ctx context.Context, minAge time.Duration) (int64, error)
	ExpireOld(ctx context.Context, maxAge time.Duration) (int64, error)
}

type botEventOutboxRepository struct{}

func NewBotEventOutboxRepository() BotEventOutboxRepository {
	return &botEventOutboxRepository{}
}

const outboxColumns = `id, bot_id, event_id, seq, payload, created_at, acked_at`

func scanOutboxEntry(row pgx.Row) (*models.BotEventOutbox, error) {
	e := &models.BotEventOutbox{}
	err := row.Scan(&e.ID, &e.BotID, &e.EventID, &e.Seq, &e.Payload, &e.CreatedAt, &e.ACKedAt)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (r *botEventOutboxRepository) Append(ctx context.Context, botID uuid.UUID, eventID string, payload []byte) (int64, error) {
	var seq int64
	err := database.GetPool().QueryRow(ctx, `
		WITH s AS (
			INSERT INTO bot_event_seq_counter (bot_id, next_seq) VALUES ($1, 1)
			ON CONFLICT (bot_id) DO UPDATE SET next_seq = bot_event_seq_counter.next_seq + 1
			RETURNING next_seq
		)
		INSERT INTO bot_event_outbox (id, bot_id, event_id, seq, payload)
		SELECT $2, $1, $3, s.next_seq, $4 FROM s
		RETURNING seq
	`, botID, uuid.New(), eventID, payload).Scan(&seq)
	if err != nil {
		return 0, fmt.Errorf("append outbox event: %w", err)
	}
	return seq, nil
}

func (r *botEventOutboxRepository) FindUnacked(ctx context.Context, credentialID, botID uuid.UUID, afterSeq int64, limit int) ([]*models.BotEventOutbox, error) {
	if limit <= 0 || limit > 500 {
		limit = 500
	}
	rows, err := database.GetPool().Query(ctx, `
		SELECT `+outboxColumns+` FROM bot_event_outbox
		WHERE bot_id = $2 AND seq > GREATEST(
			$3,
			COALESCE((SELECT last_acked_seq FROM bot_event_ack_state WHERE credential_id = $1 AND bot_id = $2), 0)
		)
		ORDER BY seq ASC LIMIT $4
	`, credentialID, botID, afterSeq, limit)
	if err != nil {
		return nil, fmt.Errorf("find unacked events: %w", err)
	}
	defer rows.Close()
	var out []*models.BotEventOutbox
	for rows.Next() {
		e, err := scanOutboxEntry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *botEventOutboxRepository) FindByEventID(ctx context.Context, botID uuid.UUID, eventID string) (*models.BotEventOutbox, error) {
	query := fmt.Sprintf(`SELECT %s FROM bot_event_outbox WHERE bot_id = $1 AND event_id = $2`, outboxColumns)
	return scanOutboxEntry(database.GetPool().QueryRow(ctx, query, botID, eventID))
}

func (r *botEventOutboxRepository) CountUnacked(ctx context.Context, botID uuid.UUID) (int64, error) {
	var count int64
	err := database.GetPool().QueryRow(ctx,
		`SELECT COUNT(*) FROM bot_event_outbox WHERE bot_id = $1`, botID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count unacked events: %w", err)
	}
	return count, nil
}

func (r *botEventOutboxRepository) AckUpTo(ctx context.Context, credentialID, botID uuid.UUID, seq int64) (int64, error) {
	var acknowledgedCount int64
	err := pgx.BeginTxFunc(ctx, database.GetPool(), pgx.TxOptions{}, func(tx pgx.Tx) error {
		var lastAllocated int64
		err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(next_seq), 0) FROM bot_event_seq_counter WHERE bot_id = $1`, botID).Scan(&lastAllocated)
		if err != nil {
			return err
		}
		if seq > lastAllocated {
			return ErrAckSequenceAhead
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO bot_event_ack_state (credential_id, bot_id, last_acked_seq, updated_at)
			VALUES ($1, $2, $3, now())
			ON CONFLICT (credential_id, bot_id) DO UPDATE
			SET last_acked_seq = GREATEST(bot_event_ack_state.last_acked_seq, EXCLUDED.last_acked_seq),
			    updated_at = now()
		`, credentialID, botID, seq); err != nil {
			return err
		}

		return tx.QueryRow(ctx, `
			WITH acknowledged AS (
				UPDATE bot_event_outbox e
				SET acked_at = now()
				WHERE e.bot_id = $1
				AND e.acked_at IS NULL
				AND NOT EXISTS (
					SELECT 1
					FROM bot_api_credentials c
					LEFT JOIN bot_event_ack_state s ON s.credential_id = c.id AND s.bot_id = e.bot_id
					WHERE c.bot_id = e.bot_id AND c.revoked_at IS NULL AND COALESCE(s.last_acked_seq, 0) < e.seq
				)
				RETURNING 1
			)
			SELECT COUNT(*) FROM acknowledged
		`, botID).Scan(&acknowledgedCount)
	})
	if err != nil {
		return 0, fmt.Errorf("ack events: %w", err)
	}
	return acknowledgedCount, nil
}

func (r *botEventOutboxRepository) GetAckState(ctx context.Context, credentialID, botID uuid.UUID) (int64, error) {
	var lastAcked int64
	err := database.GetPool().QueryRow(ctx,
		`SELECT last_acked_seq FROM bot_event_ack_state WHERE credential_id = $1 AND bot_id = $2`,
		credentialID, botID).Scan(&lastAcked)
	if err != nil {
		return 0, fmt.Errorf("get ack state: %w", err)
	}
	return lastAcked, nil
}

func (r *botEventOutboxRepository) DeleteAcked(ctx context.Context, minAge time.Duration) (int64, error) {
	ct, err := database.GetPool().Exec(ctx,
		`DELETE FROM bot_event_outbox WHERE acked_at IS NOT NULL AND acked_at < now() - $1 * interval '1 second'`,
		int64(minAge.Seconds()))
	if err != nil {
		return 0, fmt.Errorf("delete acked events: %w", err)
	}
	return ct.RowsAffected(), nil
}

func (r *botEventOutboxRepository) ExpireOld(ctx context.Context, maxAge time.Duration) (int64, error) {
	ct, err := database.GetPool().Exec(ctx,
		`DELETE FROM bot_event_outbox WHERE created_at < now() - $1 * interval '1 second'`,
		int64(maxAge.Seconds()))
	if err != nil {
		return 0, fmt.Errorf("expire old events: %w", err)
	}
	return ct.RowsAffected(), nil
}
