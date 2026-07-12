package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"purr-chat-server/internal/models"
	"purr-chat-server/pkg/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// BotInstallationRepository Bot 安装数据访问接口
type BotInstallationRepository interface {
	Create(ctx context.Context, inst *models.BotInstallation) error
	CreateTx(ctx context.Context, tx pgx.Tx, inst *models.BotInstallation) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.BotInstallation, error)
	FindByIDWithApp(ctx context.Context, id uuid.UUID) (*models.BotInstallation, error)
	FindByApp(ctx context.Context, appID uuid.UUID) ([]*models.BotInstallation, error)
	FindByTarget(ctx context.Context, targetType models.InstallationTargetType, targetID uuid.UUID) ([]*models.BotInstallation, error)
	FindByAppAndTarget(ctx context.Context, appID uuid.UUID, targetType models.InstallationTargetType, targetID uuid.UUID) (*models.BotInstallation, error)
	FindByInstaller(ctx context.Context, installerID uuid.UUID) ([]*models.BotInstallation, error)
	FindActiveByConversation(ctx context.Context, conversationID uuid.UUID) ([]*models.BotInstallation, error)
	Update(ctx context.Context, inst *models.BotInstallation) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByAppAndTarget(ctx context.Context, appID uuid.UUID, targetType models.InstallationTargetType, targetID uuid.UUID) error
}

type botInstallationRepository struct{}

type installationQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// NewBotInstallationRepository 创建 Bot 安装仓储
func NewBotInstallationRepository() BotInstallationRepository {
	return &botInstallationRepository{}
}

func (r *botInstallationRepository) Create(ctx context.Context, inst *models.BotInstallation) error {
	prepareInstallation(inst)
	return createInstallation(ctx, database.GetPool(), inst)
}

// CreateTx 在给定事务中创建安装记录
func (r *botInstallationRepository) CreateTx(ctx context.Context, tx pgx.Tx, inst *models.BotInstallation) error {
	prepareInstallation(inst)
	return createInstallation(ctx, tx, inst)
}

func prepareInstallation(inst *models.BotInstallation) {
	inst.ID = uuid.New()
	now := time.Now().UTC()
	inst.InstalledAt = now
	inst.UpdatedAt = now

	if inst.GrantedCapabilities == nil {
		inst.GrantedCapabilities = []string{}
	}
	if inst.Config == nil {
		inst.Config = json.RawMessage(`{}`)
	}
	if inst.DiagnosticsConsent == "" {
		inst.DiagnosticsConsent = models.DiagnosticsDenied
	}
	if inst.Status == "" {
		inst.Status = models.InstallationActive
	}

}

func createInstallation(ctx context.Context, db installationQuerier, inst *models.BotInstallation) error {
	query := `
        INSERT INTO bot_installations (id, app_id, installed_by, target_type, target_id,
                                       granted_capabilities, diagnostics_consent, status, config,
                                       installed_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
        ON CONFLICT (target_type, target_id, app_id) DO UPDATE
        SET updated_at = bot_installations.updated_at
        RETURNING id, granted_capabilities, diagnostics_consent, status, config, installed_at, updated_at
    `
	return db.QueryRow(ctx, query,
		inst.ID, inst.AppID, inst.InstalledBy, inst.TargetType, inst.TargetID,
		inst.GrantedCapabilities, inst.DiagnosticsConsent, inst.Status, inst.Config,
		inst.InstalledAt, inst.UpdatedAt,
	).Scan(&inst.ID, &inst.GrantedCapabilities, &inst.DiagnosticsConsent, &inst.Status,
		&inst.Config, &inst.InstalledAt, &inst.UpdatedAt)
}

const installationColumns = `id, app_id, installed_by, target_type, target_id,
       granted_capabilities, diagnostics_consent, status, config, installed_at, updated_at`

func scanInstallation(row pgx.Row) (*models.BotInstallation, error) {
	inst := &models.BotInstallation{}
	err := row.Scan(
		&inst.ID, &inst.AppID, &inst.InstalledBy, &inst.TargetType, &inst.TargetID,
		&inst.GrantedCapabilities, &inst.DiagnosticsConsent, &inst.Status, &inst.Config,
		&inst.InstalledAt, &inst.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return inst, nil
}

func (r *botInstallationRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.BotInstallation, error) {
	query := fmt.Sprintf(`SELECT %s FROM bot_installations WHERE id = $1`, installationColumns)
	return scanInstallation(database.GetPool().QueryRow(ctx, query, id))
}

func (r *botInstallationRepository) FindByIDWithApp(ctx context.Context, id uuid.UUID) (*models.BotInstallation, error) {
	query := `
        SELECT i.id, i.app_id, i.installed_by, i.target_type, i.target_id,
               i.granted_capabilities, i.diagnostics_consent, i.status, i.config, i.installed_at, i.updated_at,
               b.id, b.owner_id, b.name, b.avatar_url, b.description, b.status, b.visibility, b.mechanism_config,
               b.bot_type, b.discoverability, b.is_system, b.published_version, b.requested_capabilities,
               b.created_at, b.updated_at
        FROM bot_installations i
        JOIN bots b ON i.app_id = b.id
        WHERE i.id = $1
    `
	inst := &models.BotInstallation{}
	bot := &models.Bot{}
	err := database.GetPool().QueryRow(ctx, query, id).Scan(
		&inst.ID, &inst.AppID, &inst.InstalledBy, &inst.TargetType, &inst.TargetID,
		&inst.GrantedCapabilities, &inst.DiagnosticsConsent, &inst.Status, &inst.Config, &inst.InstalledAt, &inst.UpdatedAt,
		&bot.ID, &bot.OwnerID, &bot.Name, &bot.AvatarURL, &bot.Description, &bot.Status, &bot.Visibility, &bot.MechanismConfig,
		&bot.BotType, &bot.Discoverability, &bot.IsSystem, &bot.PublishedVersion, &bot.RequestedCapabilities,
		&bot.CreatedAt, &bot.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	inst.App = bot
	return inst, nil
}

func (r *botInstallationRepository) FindByApp(ctx context.Context, appID uuid.UUID) ([]*models.BotInstallation, error) {
	query := fmt.Sprintf(`SELECT %s FROM bot_installations WHERE app_id = $1 ORDER BY installed_at DESC`, installationColumns)
	rows, err := database.GetPool().Query(ctx, query, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInstallationsFromRows(rows)
}

func (r *botInstallationRepository) FindByTarget(ctx context.Context, targetType models.InstallationTargetType, targetID uuid.UUID) ([]*models.BotInstallation, error) {
	query := fmt.Sprintf(`SELECT %s FROM bot_installations WHERE target_type = $1 AND target_id = $2 ORDER BY installed_at DESC`, installationColumns)
	rows, err := database.GetPool().Query(ctx, query, targetType, targetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInstallationsFromRows(rows)
}

func (r *botInstallationRepository) FindByAppAndTarget(ctx context.Context, appID uuid.UUID, targetType models.InstallationTargetType, targetID uuid.UUID) (*models.BotInstallation, error) {
	query := fmt.Sprintf(`SELECT %s FROM bot_installations WHERE app_id = $1 AND target_type = $2 AND target_id = $3`, installationColumns)
	return scanInstallation(database.GetPool().QueryRow(ctx, query, appID, targetType, targetID))
}

func (r *botInstallationRepository) FindByInstaller(ctx context.Context, installerID uuid.UUID) ([]*models.BotInstallation, error) {
	query := fmt.Sprintf(`SELECT %s FROM bot_installations WHERE installed_by = $1 ORDER BY installed_at DESC`, installationColumns)
	rows, err := database.GetPool().Query(ctx, query, installerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInstallationsFromRows(rows)
}

func (r *botInstallationRepository) FindActiveByConversation(ctx context.Context, conversationID uuid.UUID) ([]*models.BotInstallation, error) {
	query := fmt.Sprintf(`SELECT %s FROM bot_installations WHERE target_type = 'conversation' AND target_id = $1 AND status = 'active' ORDER BY installed_at ASC`, installationColumns)
	rows, err := database.GetPool().Query(ctx, query, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInstallationsFromRows(rows)
}

func (r *botInstallationRepository) Update(ctx context.Context, inst *models.BotInstallation) error {
	inst.UpdatedAt = time.Now().UTC()
	query := `
        UPDATE bot_installations SET status = $1, granted_capabilities = $2, diagnostics_consent = $3, config = $4, updated_at = $5
        WHERE id = $6
    `
	_, err := database.GetPool().Exec(ctx, query,
		inst.Status, inst.GrantedCapabilities, inst.DiagnosticsConsent, inst.Config, inst.UpdatedAt, inst.ID,
	)
	return err
}

func (r *botInstallationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := database.GetPool().Exec(ctx, "DELETE FROM bot_installations WHERE id = $1", id)
	return err
}

func (r *botInstallationRepository) DeleteByAppAndTarget(ctx context.Context, appID uuid.UUID, targetType models.InstallationTargetType, targetID uuid.UUID) error {
	_, err := database.GetPool().Exec(ctx, "DELETE FROM bot_installations WHERE app_id = $1 AND target_type = $2 AND target_id = $3", appID, targetType, targetID)
	return err
}

func scanInstallationsFromRows(rows pgx.Rows) ([]*models.BotInstallation, error) {
	var installations []*models.BotInstallation
	for rows.Next() {
		inst, err := scanInstallation(rows)
		if err != nil {
			return nil, err
		}
		installations = append(installations, inst)
	}
	return installations, nil
}
