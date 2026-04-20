package services

import (
	"context"

	"purr-chat-server/internal/repository"
	"purr-chat-server/pkg/logger"

	"github.com/google/uuid"
)

// 允许的顶层设置键（白名单）
var allowedSettingKeys = map[string]bool{
	"panels":        true,
	"notifications": true,
	"general":       true,
}

// SettingsService 设置服务
type SettingsService struct {
	settingsRepo repository.SettingsRepository
}

// NewSettingsService 创建设置服务
func NewSettingsService(settingsRepo repository.SettingsRepository) *SettingsService {
	return &SettingsService{settingsRepo: settingsRepo}
}

// GetSettings 获取用户设置
func (s *SettingsService) GetSettings(ctx context.Context, userID string) (map[string]any, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		logger.ErrorfWithCaller("Invalid user ID format: %s", userID)
		return nil, err
	}

	settings, err := s.settingsRepo.GetByUserID(ctx, userUUID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to get settings for user %s: %v", userID, err)
		return nil, err
	}

	return settings, nil
}

// UpdateSettings 更新用户设置（白名单过滤）
func (s *SettingsService) UpdateSettings(ctx context.Context, userID string, settings map[string]any) (map[string]any, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		logger.ErrorfWithCaller("Invalid user ID format: %s", userID)
		return nil, err
	}

	// 白名单过滤：只保留允许的顶层键
	filtered := make(map[string]any)
	for key, value := range settings {
		if allowedSettingKeys[key] {
			filtered[key] = value
		}
	}

	if err := s.settingsRepo.Upsert(ctx, userUUID, filtered); err != nil {
		logger.ErrorfWithCaller("Failed to update settings for user %s: %v", userID, err)
		return nil, err
	}

	logger.InfofWithCaller("Settings updated for user %s", userID)
	return filtered, nil
}
