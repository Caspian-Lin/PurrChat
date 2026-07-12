package repository

import (
	"context"
	"encoding/json"
	"time"

	"purr-chat-server/internal/models"
	"purr-chat-server/pkg/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type AuthenticatedBotCredential struct {
	Credential *models.BotAPICredential
	BotType    models.BotType
	BotStatus  models.BotStatus
	IdentityID uuid.UUID
}

type BotAPICredentialRepository interface {
	Create(ctx context.Context, credential *models.BotAPICredential, tokenHash []byte, actorID uuid.UUID) error
	ListByBot(ctx context.Context, botID uuid.UUID) ([]*models.BotAPICredential, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.BotAPICredential, error)
	FindForAuthentication(ctx context.Context, tokenHash []byte) (*AuthenticatedBotCredential, error)
	Rotate(ctx context.Context, id uuid.UUID, tokenHash []byte, tokenPrefix string, actorID uuid.UUID) (*models.BotAPICredential, error)
	Revoke(ctx context.Context, id, actorID uuid.UUID) (*models.BotAPICredential, error)
	TouchLastUsed(ctx context.Context, id uuid.UUID) error
	RecordAudit(ctx context.Context, credentialID, botID uuid.UUID, eventType string, metadata map[string]any) error
}

type botAPICredentialRepository struct{}

func NewBotAPICredentialRepository() BotAPICredentialRepository { return &botAPICredentialRepository{} }

const credentialColumns = `id, bot_id, name, token_prefix, last_used_at, expires_at, revoked_at, created_at, updated_at`

func scanCredential(row pgx.Row) (*models.BotAPICredential, error) {
	c := &models.BotAPICredential{}
	err := row.Scan(&c.ID, &c.BotID, &c.Name, &c.TokenPrefix, &c.LastUsedAt, &c.ExpiresAt, &c.RevokedAt, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func (r *botAPICredentialRepository) Create(ctx context.Context, c *models.BotAPICredential, tokenHash []byte, actorID uuid.UUID) error {
	return pgx.BeginTxFunc(ctx, database.GetPool(), pgx.TxOptions{}, func(tx pgx.Tx) error {
		now := time.Now().UTC()
		c.ID, c.CreatedAt, c.UpdatedAt = uuid.New(), now, now
		if _, err := tx.Exec(ctx, `INSERT INTO bot_api_credentials
			(id, bot_id, name, token_hash, token_prefix, expires_at, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$7)`, c.ID, c.BotID, c.Name, tokenHash, c.TokenPrefix, c.ExpiresAt, now); err != nil {
			return err
		}
		return insertCredentialAudit(ctx, tx, c.ID, c.BotID, &actorID, "created", nil)
	})
}

func (r *botAPICredentialRepository) ListByBot(ctx context.Context, botID uuid.UUID) ([]*models.BotAPICredential, error) {
	rows, err := database.GetPool().Query(ctx, `SELECT `+credentialColumns+` FROM bot_api_credentials WHERE bot_id=$1 ORDER BY created_at DESC`, botID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.BotAPICredential
	for rows.Next() {
		c, err := scanCredential(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *botAPICredentialRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.BotAPICredential, error) {
	return scanCredential(database.GetPool().QueryRow(ctx, `SELECT `+credentialColumns+` FROM bot_api_credentials WHERE id=$1`, id))
}

func (r *botAPICredentialRepository) FindForAuthentication(ctx context.Context, tokenHash []byte) (*AuthenticatedBotCredential, error) {
	a := &AuthenticatedBotCredential{Credential: &models.BotAPICredential{}}
	c := a.Credential
	err := database.GetPool().QueryRow(ctx, `SELECT c.id, c.bot_id, c.name, c.token_prefix, c.last_used_at,
		c.expires_at, c.revoked_at, c.created_at, c.updated_at, b.bot_type, b.status,
		COALESCE(i.user_id, '00000000-0000-0000-0000-000000000000'::uuid)
		FROM bot_api_credentials c JOIN bots b ON b.id=c.bot_id LEFT JOIN bot_identities i ON i.app_id=b.id
		WHERE c.token_hash=$1`, tokenHash).Scan(
		&c.ID, &c.BotID, &c.Name, &c.TokenPrefix, &c.LastUsedAt, &c.ExpiresAt, &c.RevokedAt, &c.CreatedAt, &c.UpdatedAt,
		&a.BotType, &a.BotStatus, &a.IdentityID)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (r *botAPICredentialRepository) Rotate(ctx context.Context, id uuid.UUID, tokenHash []byte, tokenPrefix string, actorID uuid.UUID) (*models.BotAPICredential, error) {
	var c *models.BotAPICredential
	err := pgx.BeginTxFunc(ctx, database.GetPool(), pgx.TxOptions{}, func(tx pgx.Tx) error {
		var err error
		c, err = scanCredential(tx.QueryRow(ctx, `UPDATE bot_api_credentials SET token_hash=$2, token_prefix=$3,
			last_used_at=NULL, updated_at=$4 WHERE id=$1 AND revoked_at IS NULL RETURNING `+credentialColumns, id, tokenHash, tokenPrefix, time.Now().UTC()))
		if err != nil {
			return err
		}
		return insertCredentialAudit(ctx, tx, c.ID, c.BotID, &actorID, "rotated", nil)
	})
	return c, err
}

func (r *botAPICredentialRepository) Revoke(ctx context.Context, id, actorID uuid.UUID) (*models.BotAPICredential, error) {
	var c *models.BotAPICredential
	err := pgx.BeginTxFunc(ctx, database.GetPool(), pgx.TxOptions{}, func(tx pgx.Tx) error {
		var err error
		now := time.Now().UTC()
		c, err = scanCredential(tx.QueryRow(ctx, `UPDATE bot_api_credentials SET revoked_at=COALESCE(revoked_at,$2), updated_at=$2
			WHERE id=$1 RETURNING `+credentialColumns, id, now))
		if err != nil {
			return err
		}
		return insertCredentialAudit(ctx, tx, c.ID, c.BotID, &actorID, "revoked", nil)
	})
	return c, err
}

func (r *botAPICredentialRepository) TouchLastUsed(ctx context.Context, id uuid.UUID) error {
	_, err := database.GetPool().Exec(ctx, `UPDATE bot_api_credentials SET last_used_at=$2 WHERE id=$1`, id, time.Now().UTC())
	return err
}

func (r *botAPICredentialRepository) RecordAudit(ctx context.Context, credentialID, botID uuid.UUID, eventType string, metadata map[string]any) error {
	return insertCredentialAudit(ctx, database.GetPool(), credentialID, botID, nil, eventType, metadata)
}

type auditExecer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func insertCredentialAudit(ctx context.Context, db auditExecer, credentialID, botID uuid.UUID, actorID *uuid.UUID, eventType string, metadata map[string]any) error {
	if metadata == nil {
		metadata = map[string]any{}
	}
	raw, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `INSERT INTO bot_api_credential_audit_logs
		(id, credential_id, bot_id, actor_id, event_type, metadata) VALUES ($1,$2,$3,$4,$5,$6)`,
		uuid.New(), credentialID, botID, actorID, eventType, raw)
	return err
}
