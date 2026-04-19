package models

import "time"

// UserSettings 用户设置模型
type UserSettings struct {
    UserID    string    `json:"user_id" db:"user_id"`
    Settings  string    `json:"settings" db:"settings"`
    UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// UpdateSettingsRequest 更新设置请求
type UpdateSettingsRequest struct {
    Settings map[string]any `json:"settings" binding:"required"`
}
