package models

import (
	"time"

	"github.com/google/uuid"
)

type BotAPICredential struct {
	ID          uuid.UUID  `json:"id"`
	BotID       uuid.UUID  `json:"bot_id"`
	Name        string     `json:"name"`
	TokenPrefix string     `json:"token_prefix"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type CreateBotAPICredentialRequest struct {
	Name      string     `json:"name" binding:"required,min=1,max=64"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type BotAPICredentialSecret struct {
	Credential *BotAPICredential `json:"credential"`
	Token      string            `json:"token"`
}

// BotPrincipal is the immutable identity established by a Bot credential.
type BotPrincipal struct {
	BotID        uuid.UUID `json:"bot_id"`
	IdentityID   uuid.UUID `json:"identity_id"`
	CredentialID uuid.UUID `json:"credential_id"`
}
