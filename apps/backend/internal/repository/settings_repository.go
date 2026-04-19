package repository

import (
    "context"
    "encoding/json"

    "purr-chat-server/pkg/database"

    "github.com/google/uuid"
)

// SettingsRepository 设置仓储接口
type SettingsRepository interface {
    GetByUserID(ctx context.Context, userID uuid.UUID) (map[string]any, error)
    Upsert(ctx context.Context, userID uuid.UUID, settings map[string]any) error
}

type settingsRepository struct{}

// NewSettingsRepository 创建设置仓储
func NewSettingsRepository() SettingsRepository {
    return &settingsRepository{}
}

// GetByUserID 获取用户设置，未找到时返回空 map（新用户）
func (r *settingsRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (map[string]any, error) {
    query := `SELECT settings FROM user_settings WHERE user_id = $1`

    var settingsJSON []byte
    err := database.GetPool().QueryRow(ctx, query, userID).Scan(&settingsJSON)
    if err != nil {
        // pgx 返回 "no rows" 表示新用户，返回空 map
        return make(map[string]any), nil
    }

    var settings map[string]any
    if err := json.Unmarshal(settingsJSON, &settings); err != nil {
        return nil, err
    }

    return settings, nil
}

// Upsert 创建或更新用户设置
func (r *settingsRepository) Upsert(ctx context.Context, userID uuid.UUID, settings map[string]any) error {
    settingsJSON, err := json.Marshal(settings)
    if err != nil {
        return err
    }

    query := `
        INSERT INTO user_settings (user_id, settings, updated_at)
        VALUES ($1, $2, CURRENT_TIMESTAMP)
        ON CONFLICT (user_id)
        DO UPDATE SET settings = $2, updated_at = CURRENT_TIMESTAMP
    `

    _, err = database.GetPool().Exec(ctx, query, userID, settingsJSON)
    return err
}
