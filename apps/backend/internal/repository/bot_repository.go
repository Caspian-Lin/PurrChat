package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"purr-chat-server/internal/models"
	"purr-chat-server/pkg/database"
	"purr-chat-server/pkg/logger"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// BotRepository Bot 数据访问接口
type BotRepository interface {
	Create(ctx context.Context, bot *models.Bot) error
	CreateTx(ctx context.Context, tx pgx.Tx, bot *models.Bot) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Bot, error)
	FindByOwner(ctx context.Context, ownerID uuid.UUID) ([]*models.Bot, error)
	FindPublic(ctx context.Context, query string, limit, offset int) ([]*models.Bot, error)
	CountPublic(ctx context.Context, q string) (int, error)
	FindPublicWithDetails(ctx context.Context, q string, limit, offset int) ([]*models.PublicBotDetail, error)
	Update(ctx context.Context, bot *models.Bot) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// BotDeploymentRepository Bot 部署数据访问接口
type BotDeploymentRepository interface {
	Create(ctx context.Context, deployment *models.BotDeployment) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.BotDeployment, error)
	FindByBotID(ctx context.Context, botID uuid.UUID) ([]*models.BotDeployment, error)
	FindByConversationID(ctx context.Context, conversationID uuid.UUID) ([]*models.BotDeployment, error)
	FindByBotAndConversation(ctx context.Context, botID, conversationID uuid.UUID) (*models.BotDeployment, error)
	FindByUser(ctx context.Context, userID uuid.UUID) ([]*models.BotDeployment, error)
	Update(ctx context.Context, deployment *models.BotDeployment) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByBotAndConversation(ctx context.Context, botID, conversationID uuid.UUID) error
	FindActiveByConversation(ctx context.Context, conversationID uuid.UUID) ([]*models.BotDeployment, error)
}

type botRepository struct{}

// NewBotRepository 创建 Bot 仓储
func NewBotRepository() BotRepository {
	return &botRepository{}
}

func (r *botRepository) Create(ctx context.Context, bot *models.Bot) error {
	prepareBot(bot)

	err := pgx.BeginTxFunc(ctx, database.GetPool(), pgx.TxOptions{}, func(tx pgx.Tx) error {
		return createBotTx(ctx, tx, bot)
	})

	if err != nil {
		logger.ErrorfWithCaller("Failed to create bot: %v", err)
	} else {
		logger.InfofWithCaller("Bot created successfully: ID=%s, Name=%s", bot.ID, bot.Name)
	}

	return err
}

// CreateTx 在给定事务中创建 Bot 主记录及其身份投影。
func (r *botRepository) CreateTx(ctx context.Context, tx pgx.Tx, bot *models.Bot) error {
	prepareBot(bot)
	return createBotTx(ctx, tx, bot)
}

func prepareBot(bot *models.Bot) {
	bot.ID = uuid.New()
	bot.CreatedAt = time.Now().UTC()
	bot.UpdatedAt = time.Now().UTC()

	if bot.MechanismConfig == nil {
		bot.MechanismConfig = json.RawMessage(`[]`)
	}
	if bot.BotType == "" {
		bot.BotType = models.BotTypeWorkflow
	}
	if bot.Discoverability == "" {
		bot.Discoverability = models.DiscoverabilityUnlisted
	}
	if bot.Visibility == "" {
		bot.Visibility = models.BotVisibilityPrivate
	}
	if bot.RequestedCapabilities == nil {
		bot.RequestedCapabilities = []string{}
	}
	if bot.AllowedEndpoints == nil {
		bot.AllowedEndpoints = []string{}
	}

}

func createBotTx(ctx context.Context, tx pgx.Tx, bot *models.Bot) error {
	// 1. 创建 Bot 对应的 user 记录(系统身份投影行)
	_, err := tx.Exec(ctx, `
			INSERT INTO users (id, username, password_hash, salt, avatar_url, is_bot, created_at)
			VALUES ($1, $2, '', '', $3, TRUE, $4)
		`, bot.ID, bot.Name, bot.AvatarURL, bot.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create bot user record: %w", err)
	}

	// 2. 创建 bot 记录(含 App 化字段)
	_, err = tx.Exec(ctx, `
			INSERT INTO bots (id, owner_id, name, avatar_url, description, status, visibility, mechanism_config,
			                  bot_type, discoverability, is_system, published_version, requested_capabilities, allowed_endpoints,
			                  created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		`,
		bot.ID, bot.OwnerID, bot.Name, bot.AvatarURL, bot.Description,
		bot.Status, bot.Visibility, bot.MechanismConfig,
		bot.BotType, bot.Discoverability, bot.IsSystem, bot.PublishedVersion, bot.RequestedCapabilities, bot.AllowedEndpoints,
		bot.CreatedAt, bot.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create bot record: %w", err)
	}

	// 3. 创建 bot_identity 投影(不可登录、不可好友的系统身份)
	_, err = tx.Exec(ctx, `
			INSERT INTO bot_identities (app_id, user_id, display_name, avatar_url, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, bot.ID, bot.ID, bot.Name, bot.AvatarURL, bot.CreatedAt, bot.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create bot_identity: %w", err)
	}
	return nil
}

func (r *botRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Bot, error) {
	query := `
        SELECT id, owner_id, name, avatar_url, description, status, visibility, mechanism_config,
               bot_type, discoverability, is_system, published_version, requested_capabilities, allowed_endpoints,
               created_at, updated_at
        FROM bots WHERE id = $1
    `

	bot := &models.Bot{}
	err := database.GetPool().QueryRow(ctx, query, id).Scan(
		&bot.ID, &bot.OwnerID, &bot.Name, &bot.AvatarURL, &bot.Description,
		&bot.Status, &bot.Visibility, &bot.MechanismConfig,
		&bot.BotType, &bot.Discoverability, &bot.IsSystem, &bot.PublishedVersion, &bot.RequestedCapabilities, &bot.AllowedEndpoints,
		&bot.CreatedAt, &bot.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return bot, nil
}

func (r *botRepository) FindByOwner(ctx context.Context, ownerID uuid.UUID) ([]*models.Bot, error) {
	query := `
        SELECT id, owner_id, name, avatar_url, description, status, visibility, mechanism_config,
               bot_type, discoverability, is_system, published_version, requested_capabilities, allowed_endpoints,
               created_at, updated_at
        FROM bots WHERE owner_id = $1
        ORDER BY created_at DESC
    `

	rows, err := database.GetPool().Query(ctx, query, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanBotsFromRows(rows)
}

func (r *botRepository) FindPublic(ctx context.Context, q string, limit, offset int) ([]*models.Bot, error) {
	sql := `
        SELECT id, owner_id, name, avatar_url, description, status, visibility, mechanism_config,
               bot_type, discoverability, is_system, published_version, requested_capabilities, allowed_endpoints,
               created_at, updated_at
        FROM bots
        WHERE visibility IN ('public', 'global') AND status = 'active'
    `
	args := []any{}
	argIdx := 1

	if q != "" {
		sql += fmt.Sprintf(" AND (name ILIKE $%d OR description ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+q+"%")
		argIdx++
	}

	sql += " ORDER BY created_at DESC"

	if limit > 0 {
		sql += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, limit)
		argIdx++
	}
	if offset > 0 {
		sql += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, offset)
	}

	rows, err := database.GetPool().Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanBotsFromRows(rows)
}

func (r *botRepository) CountPublic(ctx context.Context, q string) (int, error) {
	sql := `SELECT COUNT(*) FROM bots WHERE visibility IN ('public', 'global') AND status = 'active'`
	args := []any{}

	if q != "" {
		sql += " AND (name ILIKE $1 OR description ILIKE $1)"
		args = append(args, "%"+q+"%")
	}

	var count int
	err := database.GetPool().QueryRow(ctx, sql, args...).Scan(&count)
	return count, err
}

func (r *botRepository) FindPublicWithDetails(ctx context.Context, q string, limit, offset int) ([]*models.PublicBotDetail, error) {
	sql := `
        SELECT b.id, b.owner_id, b.name, b.avatar_url, b.description, b.status, b.visibility,
               b.mechanism_config, b.bot_type, b.discoverability, b.is_system, b.published_version, b.requested_capabilities, b.allowed_endpoints,
               b.created_at, b.updated_at,
               COALESCE(d.cnt, 0) AS deployment_count,
               COALESCE(u.username, '') AS owner_name
        FROM bots b
        LEFT JOIN users u ON b.owner_id = u.id
               LEFT JOIN (SELECT app_id, COUNT(*) AS cnt FROM bot_installations GROUP BY app_id) d ON b.id = d.app_id
        WHERE b.visibility IN ('public', 'global') AND b.status = 'active'
    `
	args := []any{}
	argIdx := 1

	if q != "" {
		sql += fmt.Sprintf(" AND (b.name ILIKE $%d OR b.description ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+q+"%")
		argIdx++
	}

	sql += " ORDER BY b.created_at DESC"

	if limit > 0 {
		sql += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, limit)
		argIdx++
	}
	if offset > 0 {
		sql += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, offset)
	}

	rows, err := database.GetPool().Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.PublicBotDetail
	for rows.Next() {
		d := &models.PublicBotDetail{}
		err := rows.Scan(
			&d.ID, &d.OwnerID, &d.Name, &d.AvatarURL, &d.Description,
			&d.Status, &d.Visibility, &d.MechanismConfig,
			&d.BotType, &d.Discoverability, &d.IsSystem, &d.PublishedVersion, &d.RequestedCapabilities, &d.AllowedEndpoints,
			&d.CreatedAt, &d.UpdatedAt,
			&d.DeploymentCount, &d.OwnerName,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, d)
	}

	return results, nil
}

func (r *botRepository) Update(ctx context.Context, bot *models.Bot) error {
	bot.UpdatedAt = time.Now().UTC()

	// 在事务中同时更新 users 表和 bots 表
	err := pgx.BeginTxFunc(ctx, database.GetPool(), pgx.TxOptions{}, func(tx pgx.Tx) error {
		// 1. 同步更新 users 表的 username 和 avatar_url
		_, err := tx.Exec(ctx, `
			UPDATE users SET username = $1, avatar_url = $2
			WHERE id = $3 AND is_bot = TRUE
		`, bot.Name, bot.AvatarURL, bot.ID)
		if err != nil {
			return err
		}

		// 2. 更新 bots 表(含 App 化字段)
		_, err = tx.Exec(ctx, `
			UPDATE bots SET name = $1, avatar_url = $2, description = $3, status = $4, visibility = $5,
				mechanism_config = $6, bot_type = $7, discoverability = $8, is_system = $9,
				published_version = $10, requested_capabilities = $11, allowed_endpoints = $12, updated_at = $13
			WHERE id = $14
		`,
			bot.Name, bot.AvatarURL, bot.Description, bot.Status, bot.Visibility,
			bot.MechanismConfig, bot.BotType, bot.Discoverability, bot.IsSystem,
			bot.PublishedVersion, bot.RequestedCapabilities, bot.AllowedEndpoints, bot.UpdatedAt, bot.ID,
		)
		if err != nil {
			return err
		}

		// 3. 同步 bot_identity 的 display_name/avatar_url
		_, err = tx.Exec(ctx, `
			UPDATE bot_identities SET display_name = $1, avatar_url = $2, updated_at = $3
			WHERE app_id = $4
		`, bot.Name, bot.AvatarURL, bot.UpdatedAt, bot.ID)
		return err
	})

	if err != nil {
		logger.ErrorfWithCaller("Failed to update bot %s: %v", bot.ID, err)
	}

	return err
}

func (r *botRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// 在事务中清理所有关联数据
	err := pgx.BeginTxFunc(ctx, database.GetPool(), pgx.TxOptions{}, func(tx pgx.Tx) error {
		// 1. 删除 enrollments（Bot 在所有会话中的成员身份）
		if _, err := tx.Exec(ctx, "DELETE FROM enrollments WHERE user_id = $1", id); err != nil {
			return err
		}
		// 2. 删除 friendships
		if _, err := tx.Exec(ctx, "DELETE FROM friendships WHERE user_id = $1 OR friend_id = $1", id); err != nil {
			return err
		}
		// 3. 删除 bot_deployments
		if _, err := tx.Exec(ctx, "DELETE FROM bot_deployments WHERE bot_id = $1", id); err != nil {
			return err
		}
		// 4. 删除 bots 记录
		if _, err := tx.Exec(ctx, "DELETE FROM bots WHERE id = $1", id); err != nil {
			return err
		}
		// 5. 删除 users 记录
		if _, err := tx.Exec(ctx, "DELETE FROM users WHERE id = $1 AND is_bot = TRUE", id); err != nil {
			return err
		}
		return nil
	})

	return err
}

// --- BotDeployment ---

type botDeploymentRepository struct{}

// NewBotDeploymentRepository 创建 Bot 部署仓储
func NewBotDeploymentRepository() BotDeploymentRepository {
	return &botDeploymentRepository{}
}

func (r *botDeploymentRepository) Create(ctx context.Context, d *models.BotDeployment) error {
	d.ID = uuid.New()
	d.DeployedAt = time.Now().UTC()

	query := `
        INSERT INTO bot_deployments (id, bot_id, conversation_id, deployed_by, status, workflow_active, workflow_started_at, deployed_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        ON CONFLICT (bot_id, conversation_id) DO NOTHING
    `

	_, err := database.GetPool().Exec(ctx, query,
		d.ID, d.BotID, d.ConversationID, d.DeployedBy,
		d.Status, d.WorkflowActive, d.WorkflowStartedAt, d.DeployedAt,
	)

	return err
}

func (r *botDeploymentRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.BotDeployment, error) {
	query := `
        SELECT id, bot_id, conversation_id, deployed_by, status, workflow_active, workflow_started_at, deployed_at
        FROM bot_deployments WHERE id = $1
    `

	d := &models.BotDeployment{}
	err := database.GetPool().QueryRow(ctx, query, id).Scan(
		&d.ID, &d.BotID, &d.ConversationID, &d.DeployedBy,
		&d.Status, &d.WorkflowActive, &d.WorkflowStartedAt, &d.DeployedAt,
	)

	if err != nil {
		return nil, err
	}

	return d, nil
}

func (r *botDeploymentRepository) FindByBotID(ctx context.Context, botID uuid.UUID) ([]*models.BotDeployment, error) {
	query := `
        SELECT id, bot_id, conversation_id, deployed_by, status, workflow_active, workflow_started_at, deployed_at
        FROM bot_deployments WHERE bot_id = $1
        ORDER BY deployed_at DESC
    `

	rows, err := database.GetPool().Query(ctx, query, botID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanBotDeploymentsFromRows(rows)
}

func (r *botDeploymentRepository) FindByConversationID(ctx context.Context, conversationID uuid.UUID) ([]*models.BotDeployment, error) {
	query := `
        SELECT id, bot_id, conversation_id, deployed_by, status, workflow_active, workflow_started_at, deployed_at
        FROM bot_deployments WHERE conversation_id = $1
        ORDER BY deployed_at DESC
    `

	rows, err := database.GetPool().Query(ctx, query, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanBotDeploymentsFromRows(rows)
}

func (r *botDeploymentRepository) FindByBotAndConversation(ctx context.Context, botID, conversationID uuid.UUID) (*models.BotDeployment, error) {
	query := `
        SELECT id, bot_id, conversation_id, deployed_by, status, workflow_active, workflow_started_at, deployed_at
        FROM bot_deployments WHERE bot_id = $1 AND conversation_id = $2
    `

	d := &models.BotDeployment{}
	err := database.GetPool().QueryRow(ctx, query, botID, conversationID).Scan(
		&d.ID, &d.BotID, &d.ConversationID, &d.DeployedBy,
		&d.Status, &d.WorkflowActive, &d.WorkflowStartedAt, &d.DeployedAt,
	)

	if err != nil {
		return nil, err
	}

	return d, nil
}

func (r *botDeploymentRepository) FindByUser(ctx context.Context, userID uuid.UUID) ([]*models.BotDeployment, error) {
	query := `
        SELECT bd.id, bd.bot_id, bd.conversation_id, bd.deployed_by, bd.status, bd.workflow_active, bd.workflow_started_at, bd.deployed_at
        FROM bot_deployments bd
        WHERE bd.deployed_by = $1 OR bd.bot_id IN (SELECT id FROM bots WHERE owner_id = $1)
        ORDER BY bd.deployed_at DESC
    `

	rows, err := database.GetPool().Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanBotDeploymentsFromRows(rows)
}

func (r *botDeploymentRepository) FindActiveByConversation(ctx context.Context, conversationID uuid.UUID) ([]*models.BotDeployment, error) {
	query := `
        SELECT id, bot_id, conversation_id, deployed_by, status, workflow_active, workflow_started_at, deployed_at
        FROM bot_deployments
        WHERE conversation_id = $1 AND status = 'active'
        ORDER BY deployed_at ASC
    `

	rows, err := database.GetPool().Query(ctx, query, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanBotDeploymentsFromRows(rows)
}

func (r *botDeploymentRepository) Update(ctx context.Context, d *models.BotDeployment) error {
	query := `
        UPDATE bot_deployments SET status = $1, workflow_active = $2, workflow_started_at = $3
        WHERE id = $4
    `

	_, err := database.GetPool().Exec(ctx, query, d.Status, d.WorkflowActive, d.WorkflowStartedAt, d.ID)
	return err
}

func (r *botDeploymentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := database.GetPool().Exec(ctx, "DELETE FROM bot_deployments WHERE id = $1", id)
	return err
}

func (r *botDeploymentRepository) DeleteByBotAndConversation(ctx context.Context, botID, conversationID uuid.UUID) error {
	_, err := database.GetPool().Exec(ctx, "DELETE FROM bot_deployments WHERE bot_id = $1 AND conversation_id = $2", botID, conversationID)
	return err
}

// scanBotsFromRows 从 pgx.Rows 中扫描 Bot 列表
func scanBotsFromRows(rows pgx.Rows) ([]*models.Bot, error) {
	var bots []*models.Bot

	for rows.Next() {
		bot := &models.Bot{}
		err := rows.Scan(
			&bot.ID, &bot.OwnerID, &bot.Name, &bot.AvatarURL, &bot.Description,
			&bot.Status, &bot.Visibility, &bot.MechanismConfig,
			&bot.BotType, &bot.Discoverability, &bot.IsSystem, &bot.PublishedVersion, &bot.RequestedCapabilities, &bot.AllowedEndpoints,
			&bot.CreatedAt, &bot.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		bots = append(bots, bot)
	}

	return bots, nil
}

// scanBotDeploymentsFromRows 从 pgx.Rows 中扫描 BotDeployment 列表
func scanBotDeploymentsFromRows(rows pgx.Rows) ([]*models.BotDeployment, error) {
	var deployments []*models.BotDeployment

	for rows.Next() {
		d := &models.BotDeployment{}
		err := rows.Scan(
			&d.ID, &d.BotID, &d.ConversationID, &d.DeployedBy,
			&d.Status, &d.WorkflowActive, &d.WorkflowStartedAt, &d.DeployedAt,
		)
		if err != nil {
			return nil, err
		}
		deployments = append(deployments, d)
	}

	return deployments, nil
}
