package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"purr-chat-server/internal/models"
	"purr-chat-server/pkg/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type WorkflowRepository interface {
	GetDocument(ctx context.Context, botID uuid.UUID) (json.RawMessage, int, error)
	UpdateDocument(ctx context.Context, botID uuid.UUID, doc json.RawMessage, revision int) (int, error)
	FindPublishedByBotID(ctx context.Context, botID uuid.UUID) ([]*models.WorkflowVersion, error)
	FindLatestPublished(ctx context.Context, botID uuid.UUID) (*models.WorkflowVersion, error)
	Publish(ctx context.Context, botID uuid.UUID, revision int, doc json.RawMessage, capabilities []string, publishedBy uuid.UUID) (*models.WorkflowVersion, error)
}

type workflowRepository struct{}

func NewWorkflowRepository() WorkflowRepository {
	return &workflowRepository{}
}

func (r *workflowRepository) GetDocument(ctx context.Context, botID uuid.UUID) (json.RawMessage, int, error) {
	var doc json.RawMessage
	var revision int
	err := database.GetPool().QueryRow(ctx,
		`SELECT workflow_document, workflow_revision FROM bots WHERE id = $1`,
		botID,
	).Scan(&doc, &revision)
	if err != nil {
		return nil, 0, err
	}
	return doc, revision, nil
}

func (r *workflowRepository) UpdateDocument(ctx context.Context, botID uuid.UUID, doc json.RawMessage, expectedRevision int) (int, error) {
	var newRevision int
	err := pgx.BeginTxFunc(ctx, database.GetPool(), pgx.TxOptions{}, func(tx pgx.Tx) error {
		var currentRev int
		err := tx.QueryRow(ctx,
			`SELECT workflow_revision FROM bots WHERE id = $1 FOR UPDATE`,
			botID,
		).Scan(&currentRev)
		if err != nil {
			return err
		}
		if currentRev != expectedRevision {
			return fmt.Errorf("revision mismatch: expected %d, current %d", expectedRevision, currentRev)
		}
		newRevision = currentRev + 1
		_, err = tx.Exec(ctx,
			`UPDATE bots SET workflow_document = $1, workflow_revision = $2, updated_at = NOW() WHERE id = $3`,
			doc, newRevision, botID,
		)
		return err
	})
	if err != nil {
		return 0, err
	}
	return newRevision, nil
}

func (r *workflowRepository) FindLatestPublished(ctx context.Context, botID uuid.UUID) (*models.WorkflowVersion, error) {
	v := &models.WorkflowVersion{}
	err := database.GetPool().QueryRow(ctx, `
		SELECT id, bot_id, revision, document, capabilities, published_by, published_at
		FROM workflow_versions
		WHERE bot_id = $1
		ORDER BY revision DESC
		LIMIT 1
	`, botID).Scan(
		&v.ID, &v.BotID, &v.Revision, &v.Document, &v.Capabilities, &v.PublishedBy, &v.PublishedAt,
	)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (r *workflowRepository) FindPublishedByBotID(ctx context.Context, botID uuid.UUID) ([]*models.WorkflowVersion, error) {
	rows, err := database.GetPool().Query(ctx, `
		SELECT id, bot_id, revision, document, capabilities, published_by, published_at
		FROM workflow_versions
		WHERE bot_id = $1
		ORDER BY revision DESC
	`, botID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []*models.WorkflowVersion
	for rows.Next() {
		v := &models.WorkflowVersion{}
		if err := rows.Scan(
			&v.ID, &v.BotID, &v.Revision, &v.Document, &v.Capabilities, &v.PublishedBy, &v.PublishedAt,
		); err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}
	return versions, nil
}

func (r *workflowRepository) Publish(ctx context.Context, botID uuid.UUID, revision int, doc json.RawMessage, capabilities []string, publishedBy uuid.UUID) (*models.WorkflowVersion, error) {
	if capabilities == nil {
		capabilities = []string{}
	}
	v := &models.WorkflowVersion{}
	err := database.GetPool().QueryRow(ctx, `
		INSERT INTO workflow_versions (bot_id, revision, document, capabilities, published_by)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (bot_id, revision) DO UPDATE SET document = $3, capabilities = $4
		RETURNING id, bot_id, revision, document, capabilities, published_by, published_at
	`, botID, revision, doc, capabilities, publishedBy).Scan(
		&v.ID, &v.BotID, &v.Revision, &v.Document, &v.Capabilities, &v.PublishedBy, &v.PublishedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to publish workflow: %w", err)
	}

	_, err = database.GetPool().Exec(ctx,
		`UPDATE bots SET requested_capabilities = $1, published_version = $2, updated_at = NOW() WHERE id = $3`,
		capabilities, revision, botID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update bot capabilities: %w", err)
	}

	return v, nil
}
