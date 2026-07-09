package repository

import (
	"context"
	"time"

	"purr-chat-server/internal/models"
	"purr-chat-server/pkg/database"

	"github.com/google/uuid"
)

// BotAppSecretRepository secret 存储接口(密文读写)
type BotAppSecretRepository interface {
	// Set 写入/更新 secret(幂等 upsert)
	Set(ctx context.Context, appID uuid.UUID, keyName, ciphertext string) error
	// Get 读取单个 secret 密文;不存在返回 nil, nil
	Get(ctx context.Context, appID uuid.UUID, keyName string) (*models.BotAppSecret, error)
	// GetAll 读取某 Bot 的全部 secret 密文(运行时批量解密注入用)
	GetAll(ctx context.Context, appID uuid.UUID) ([]*models.BotAppSecret, error)
	// ListKeys 返回 key 列表(不取密文,列表展示用)
	ListKeys(ctx context.Context, appID uuid.UUID) ([]*models.BotAppSecret, error)
	// Delete 删除 secret
	Delete(ctx context.Context, appID uuid.UUID, keyName string) error
	// DeleteByApp 删除某 Bot 的全部 secret(级联备用)
	DeleteByApp(ctx context.Context, appID uuid.UUID) error
}

type botAppSecretRepository struct{}

func NewBotAppSecretRepository() BotAppSecretRepository {
	return &botAppSecretRepository{}
}

func (r *botAppSecretRepository) Set(ctx context.Context, appID uuid.UUID, keyName, ciphertext string) error {
	now := time.Now().UTC()
	_, err := database.GetPool().Exec(ctx, `
		INSERT INTO bot_app_secrets (app_id, key_name, ciphertext, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4)
		ON CONFLICT (app_id, key_name) DO UPDATE
			SET ciphertext = EXCLUDED.ciphertext,
			    updated_at = EXCLUDED.updated_at
	`, appID, keyName, ciphertext, now)
	return err
}

func (r *botAppSecretRepository) Get(ctx context.Context, appID uuid.UUID, keyName string) (*models.BotAppSecret, error) {
	s := &models.BotAppSecret{}
	err := database.GetPool().QueryRow(ctx, `
		SELECT app_id, key_name, ciphertext, created_at, updated_at
		FROM bot_app_secrets
		WHERE app_id = $1 AND key_name = $2
	`, appID, keyName).Scan(&s.AppID, &s.KeyName, &s.Ciphertext, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, nil // 不存在返回 nil,nil(best-effort)
	}
	s.HasValue = true
	return s, nil
}

func (r *botAppSecretRepository) GetAll(ctx context.Context, appID uuid.UUID) ([]*models.BotAppSecret, error) {
	rows, err := database.GetPool().Query(ctx, `
		SELECT app_id, key_name, ciphertext, created_at, updated_at
		FROM bot_app_secrets
		WHERE app_id = $1
	`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*models.BotAppSecret
	for rows.Next() {
		s := &models.BotAppSecret{}
		if err := rows.Scan(&s.AppID, &s.KeyName, &s.Ciphertext, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		s.HasValue = true
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *botAppSecretRepository) ListKeys(ctx context.Context, appID uuid.UUID) ([]*models.BotAppSecret, error) {
	rows, err := database.GetPool().Query(ctx, `
		SELECT app_id, key_name, created_at, updated_at
		FROM bot_app_secrets
		WHERE app_id = $1
		ORDER BY key_name
	`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*models.BotAppSecret
	for rows.Next() {
		s := &models.BotAppSecret{HasValue: true}
		if err := rows.Scan(&s.AppID, &s.KeyName, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *botAppSecretRepository) Delete(ctx context.Context, appID uuid.UUID, keyName string) error {
	_, err := database.GetPool().Exec(ctx, `
		DELETE FROM bot_app_secrets WHERE app_id = $1 AND key_name = $2
	`, appID, keyName)
	return err
}

func (r *botAppSecretRepository) DeleteByApp(ctx context.Context, appID uuid.UUID) error {
	_, err := database.GetPool().Exec(ctx, `DELETE FROM bot_app_secrets WHERE app_id = $1`, appID)
	return err
}
