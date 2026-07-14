package models

import (
	"time"

	"github.com/google/uuid"
)

// BotAppSecret BotApp 级加密密钥存储(密文)
type BotAppSecret struct {
	AppID      uuid.UUID `json:"app_id"`
	KeyName    string    `json:"key_name"`
	Ciphertext string    `json:"-"`         // 永不返回给客户端
	HasValue   bool      `json:"has_value"` // 是否已设置(列表用)
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// SetSecretRequest 设置/更新 secret 请求
type SetSecretRequest struct {
	Value string `json:"value" binding:"required,min=1,max=8192"`
}

// SecretListResponse secret 列表响应(仅 key_name,不含明文与密文)
type SecretListResponse struct {
	Secrets []*BotAppSecret `json:"secrets"`
	Total   int             `json:"total"`
}
