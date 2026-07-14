package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"purr-chat-server/internal/models"
	"purr-chat-server/pkg/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type WorkflowRepository interface {
	GetDocument(ctx context.Context, botID uuid.UUID, mechanismID string) (json.RawMessage, int, error)
	UpdateDocument(ctx context.Context, botID uuid.UUID, mechanismID string, doc json.RawMessage, expectedRevision int) (int, error)
	FindPublishedByBotAndMechanism(ctx context.Context, botID uuid.UUID, mechanismID string) ([]*models.WorkflowVersion, error)
	FindPublishedByRevision(ctx context.Context, botID uuid.UUID, mechanismID string, revision int) (*models.WorkflowVersion, error)
	FindLatestPublished(ctx context.Context, botID uuid.UUID, mechanismID string) (*models.WorkflowVersion, error)
	// FindLatestPublishedByBotID 返回该 Bot 每个 mechanism 的最新发布版本，供运行时执行遍历。
	FindLatestPublishedByBotID(ctx context.Context, botID uuid.UUID) ([]*models.WorkflowVersion, error)
	Publish(ctx context.Context, botID uuid.UUID, mechanismID string, revision int, doc json.RawMessage, capabilities []string, publishedBy uuid.UUID) (*models.WorkflowVersion, error)
}

type workflowRepository struct{}

func NewWorkflowRepository() WorkflowRepository {
	return &workflowRepository{}
}

func (r *workflowRepository) GetDocument(ctx context.Context, botID uuid.UUID, mechanismID string) (json.RawMessage, int, error) {
	var doc json.RawMessage
	var revision int
	err := database.GetPool().QueryRow(ctx,
		`SELECT document, revision FROM bot_workflow_documents WHERE bot_id = $1 AND mechanism_id = $2`,
		botID, mechanismID,
	).Scan(&doc, &revision)
	if err == pgx.ErrNoRows {
		// 首次访问该 mechanism 草稿：返回空文档 + revision 0
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}
	return doc, revision, nil
}

func (r *workflowRepository) UpdateDocument(ctx context.Context, botID uuid.UUID, mechanismID string, doc json.RawMessage, expectedRevision int) (int, error) {
	var newRevision int
	err := pgx.BeginTxFunc(ctx, database.GetPool(), pgx.TxOptions{}, func(tx pgx.Tx) error {
		var currentRev int
		err := tx.QueryRow(ctx,
			`SELECT revision FROM bot_workflow_documents WHERE bot_id = $1 AND mechanism_id = $2 FOR UPDATE`,
			botID, mechanismID,
		).Scan(&currentRev)
		if err == pgx.ErrNoRows {
			if expectedRevision != 0 {
				return fmt.Errorf("revision mismatch: expected %d, current 0", expectedRevision)
			}
			newRevision = 1
			_, err = tx.Exec(ctx,
				`INSERT INTO bot_workflow_documents (bot_id, mechanism_id, document, revision, updated_at)
				 VALUES ($1, $2, $3, $4, NOW())
				 ON CONFLICT (bot_id, mechanism_id) DO UPDATE SET document = EXCLUDED.document, revision = EXCLUDED.revision, updated_at = NOW()`,
				botID, mechanismID, doc, newRevision,
			)
			return err
		}
		if err != nil {
			return err
		}
		if currentRev != expectedRevision {
			return fmt.Errorf("revision mismatch: expected %d, current %d", expectedRevision, currentRev)
		}
		newRevision = currentRev + 1
		_, err = tx.Exec(ctx,
			`UPDATE bot_workflow_documents SET document = $1, revision = $2, updated_at = NOW() WHERE bot_id = $3 AND mechanism_id = $4`,
			doc, newRevision, botID, mechanismID,
		)
		return err
	})
	if err != nil {
		return 0, err
	}
	return newRevision, nil
}

const workflowVersionColumns = `id, bot_id, mechanism_id, revision, document, capabilities, published_by, published_at`

func scanWorkflowVersion(scanner interface{ Scan(...any) error }) (*models.WorkflowVersion, error) {
	v := &models.WorkflowVersion{}
	err := scanner.Scan(&v.ID, &v.BotID, &v.MechanismID, &v.Revision, &v.Document, &v.Capabilities, &v.PublishedBy, &v.PublishedAt)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (r *workflowRepository) FindLatestPublished(ctx context.Context, botID uuid.UUID, mechanismID string) (*models.WorkflowVersion, error) {
	return scanWorkflowVersion(database.GetPool().QueryRow(ctx, `
		SELECT `+workflowVersionColumns+` FROM workflow_versions
		WHERE bot_id = $1 AND mechanism_id = $2
		ORDER BY revision DESC LIMIT 1
	`, botID, mechanismID))
}

func (r *workflowRepository) FindPublishedByBotAndMechanism(ctx context.Context, botID uuid.UUID, mechanismID string) ([]*models.WorkflowVersion, error) {
	rows, err := database.GetPool().Query(ctx, `
		SELECT `+workflowVersionColumns+` FROM workflow_versions
		WHERE bot_id = $1 AND mechanism_id = $2
		ORDER BY revision DESC
	`, botID, mechanismID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	versions := make([]*models.WorkflowVersion, 0)
	for rows.Next() {
		v, err := scanWorkflowVersion(rows)
		if err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}
	return versions, nil
}

func (r *workflowRepository) FindPublishedByRevision(ctx context.Context, botID uuid.UUID, mechanismID string, revision int) (*models.WorkflowVersion, error) {
	return scanWorkflowVersion(database.GetPool().QueryRow(ctx, `
		SELECT `+workflowVersionColumns+` FROM workflow_versions
		WHERE bot_id = $1 AND mechanism_id = $2 AND revision = $3
	`, botID, mechanismID, revision))
}

// FindLatestPublishedByBotID 返回该 Bot 每个 mechanism 的最新发布版本（运行时执行遍历用）。
func (r *workflowRepository) FindLatestPublishedByBotID(ctx context.Context, botID uuid.UUID) ([]*models.WorkflowVersion, error) {
	rows, err := database.GetPool().Query(ctx, `
		SELECT DISTINCT ON (mechanism_id) `+workflowVersionColumns+` FROM workflow_versions
		WHERE bot_id = $1
		ORDER BY mechanism_id, revision DESC
	`, botID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	versions := make([]*models.WorkflowVersion, 0)
	for rows.Next() {
		v, err := scanWorkflowVersion(rows)
		if err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}
	return versions, nil
}

func (r *workflowRepository) Publish(ctx context.Context, botID uuid.UUID, mechanismID string, revision int, doc json.RawMessage, capabilities []string, publishedBy uuid.UUID) (*models.WorkflowVersion, error) {
	if capabilities == nil {
		capabilities = []string{}
	}
	var version *models.WorkflowVersion
	err := pgx.BeginTxFunc(ctx, database.GetPool(), pgx.TxOptions{}, func(tx pgx.Tx) error {
		v, err := scanWorkflowVersion(tx.QueryRow(ctx, `
			INSERT INTO workflow_versions (bot_id, mechanism_id, revision, document, capabilities, published_by)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (bot_id, mechanism_id, revision) DO NOTHING
			RETURNING `+workflowVersionColumns+`
		`, botID, mechanismID, revision, doc, capabilities, publishedBy))
		if err == pgx.ErrNoRows {
			// 同 (bot, mechanism, revision) 已存在，校验不可变性
			existing, findErr := scanWorkflowVersion(tx.QueryRow(ctx, `
				SELECT `+workflowVersionColumns+` FROM workflow_versions
				WHERE bot_id = $1 AND mechanism_id = $2 AND revision = $3
			`, botID, mechanismID, revision))
			if findErr != nil {
				return findErr
			}
			if !jsonDocumentsEqual(existing.Document, doc) || !stringSetsEqual(existing.Capabilities, capabilities) {
				return fmt.Errorf("published revision conflict: revision %d is immutable", revision)
			}
			version = existing
			return nil
		}
		if err != nil {
			return err
		}

		// 重新汇总 Bot 整体 requested_capabilities：所有已发布 mechanism 的能力并集
		var aggregated []string
		aggErr := tx.QueryRow(ctx, `
			SELECT COALESCE(array_agg(DISTINCT cap), '{}'::text[]) FROM workflow_versions, unnest(capabilities) AS cap WHERE bot_id = $1
		`, botID).Scan(&aggregated)
		if aggErr != nil {
			return aggErr
		}
		if _, err = tx.Exec(ctx,
			`UPDATE bots SET requested_capabilities = $1, updated_at = NOW() WHERE id = $2`,
			aggregated, botID,
		); err != nil {
			return err
		}

		// Bot 创建者的安装可能早于首次工作流发布，此时 granted_capabilities 为空。
		// 发布时同步创建者本人安装的授权，确保新声明立即可用于真实消息；
		// 其他用户的安装不自动扩权，仍需由安装者重新授权。
		if _, err = tx.Exec(ctx, `
			UPDATE bot_installations
			SET granted_capabilities = $1,
				diagnostics_consent = CASE
					WHEN $1::text[] @> ARRAY['network:external']::text[] THEN 'granted'
					ELSE diagnostics_consent
				END,
				updated_at = NOW()
			WHERE app_id = $2 AND installed_by = $3
		`, aggregated, botID, publishedBy); err != nil {
			return err
		}
		version = v
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to publish workflow: %w", err)
	}
	return version, nil
}

func jsonDocumentsEqual(a, b json.RawMessage) bool {
	var av, bv any
	if json.Unmarshal(a, &av) != nil || json.Unmarshal(b, &bv) != nil {
		return false
	}
	return reflect.DeepEqual(av, bv)
}

func stringSetsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	counts := make(map[string]int, len(a))
	for _, value := range a {
		counts[value]++
	}
	for _, value := range b {
		counts[value]--
		if counts[value] < 0 {
			return false
		}
	}
	return true
}
