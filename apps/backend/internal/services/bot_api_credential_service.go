package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrCredentialInvalid   = errors.New("invalid bot credential")
	ErrCredentialExpired   = errors.New("bot credential expired")
	ErrCredentialRevoked   = errors.New("bot credential revoked")
	ErrCredentialForbidden = errors.New("bot credential access forbidden")
	ErrBotNotExternal      = errors.New("bot is not external")
	ErrBotDisabled         = errors.New("bot is disabled")
	ErrBotIdentityMissing  = errors.New("bot identity missing")
)

type CredentialConnectionCloser interface {
	DisconnectCredential(ctx context.Context, credentialID uuid.UUID) error
}

type NoopCredentialConnectionCloser struct{}

func (NoopCredentialConnectionCloser) DisconnectCredential(context.Context, uuid.UUID) error {
	return nil
}

type BotAPICredentialService struct {
	repo       repository.BotAPICredentialRepository
	botRepo    repository.BotRepository
	disconnect CredentialConnectionCloser
}

func NewBotAPICredentialService(repo repository.BotAPICredentialRepository, botRepo repository.BotRepository, disconnect CredentialConnectionCloser) *BotAPICredentialService {
	if disconnect == nil {
		disconnect = NoopCredentialConnectionCloser{}
	}
	return &BotAPICredentialService{repo: repo, botRepo: botRepo, disconnect: disconnect}
}

func (s *BotAPICredentialService) Create(ctx context.Context, ownerID, botID uuid.UUID, name string, expiresAt *time.Time) (*models.BotAPICredentialSecret, error) {
	if err := s.assertExternalOwner(ctx, ownerID, botID); err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 64 {
		return nil, ErrCredentialInvalid
	}
	if expiresAt != nil {
		expiresUTC := expiresAt.UTC()
		if !expiresUTC.After(time.Now().UTC()) {
			return nil, ErrCredentialExpired
		}
		expiresAt = &expiresUTC
	}
	token, hash, prefix, err := generateBotToken()
	if err != nil {
		return nil, err
	}
	c := &models.BotAPICredential{BotID: botID, Name: name, TokenPrefix: prefix, ExpiresAt: expiresAt}
	if err := s.repo.Create(ctx, c, hash, ownerID); err != nil {
		return nil, err
	}
	return &models.BotAPICredentialSecret{Credential: c, Token: token}, nil
}

func (s *BotAPICredentialService) List(ctx context.Context, ownerID, botID uuid.UUID) ([]*models.BotAPICredential, error) {
	if err := s.assertExternalOwner(ctx, ownerID, botID); err != nil {
		return nil, err
	}
	return s.repo.ListByBot(ctx, botID)
}

func (s *BotAPICredentialService) Rotate(ctx context.Context, ownerID, botID, credentialID uuid.UUID) (*models.BotAPICredentialSecret, error) {
	if err := s.assertExternalOwner(ctx, ownerID, botID); err != nil {
		return nil, err
	}
	c, err := s.repo.FindByID(ctx, credentialID)
	if err != nil || c.BotID != botID {
		return nil, ErrCredentialForbidden
	}
	if c.RevokedAt != nil {
		return nil, ErrCredentialRevoked
	}
	token, hash, prefix, err := generateBotToken()
	if err != nil {
		return nil, err
	}
	c, err = s.repo.Rotate(ctx, credentialID, hash, prefix, ownerID)
	if err != nil {
		return nil, mapCredentialNotFound(err)
	}
	// Rotation is already committed and the plaintext token cannot be recovered.
	// Connection cleanup is therefore best-effort and must not turn success into
	// an error that encourages the owner to retry and lose another token.
	_ = s.disconnect.DisconnectCredential(ctx, credentialID)
	return &models.BotAPICredentialSecret{Credential: c, Token: token}, nil
}

func (s *BotAPICredentialService) Revoke(ctx context.Context, ownerID, botID, credentialID uuid.UUID) (*models.BotAPICredential, error) {
	if err := s.assertExternalOwner(ctx, ownerID, botID); err != nil {
		return nil, err
	}
	c, err := s.repo.FindByID(ctx, credentialID)
	if err != nil || c.BotID != botID {
		return nil, ErrCredentialForbidden
	}
	c, err = s.repo.Revoke(ctx, credentialID, ownerID)
	if err != nil {
		return nil, mapCredentialNotFound(err)
	}
	// Revocation is durable before active transports are notified. The
	// authenticator remains the authority even if connection cleanup is delayed.
	_ = s.disconnect.DisconnectCredential(ctx, credentialID)
	return c, nil
}

// Authenticate accepts exactly one Authorization Bearer value. Cookies and query parameters are intentionally ignored.
func (s *BotAPICredentialService) Authenticate(ctx context.Context, authorization string) (*models.BotPrincipal, error) {
	if !strings.HasPrefix(authorization, "Bearer ") || strings.Count(authorization, " ") != 1 {
		return nil, ErrCredentialInvalid
	}
	token := strings.TrimPrefix(authorization, "Bearer ")
	if token == "" {
		return nil, ErrCredentialInvalid
	}
	hash := sha256.Sum256([]byte(token))
	auth, err := s.repo.FindForAuthentication(ctx, hash[:])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCredentialInvalid
		}
		return nil, err
	}
	if auth.Credential.RevokedAt != nil {
		return nil, ErrCredentialRevoked
	}
	if auth.Credential.ExpiresAt != nil && !auth.Credential.ExpiresAt.After(time.Now().UTC()) {
		return nil, ErrCredentialExpired
	}
	if auth.BotType != models.BotTypeExternal {
		return nil, ErrBotNotExternal
	}
	if auth.BotStatus != models.BotStatusActive {
		return nil, ErrBotDisabled
	}
	if auth.IdentityID == uuid.Nil {
		return nil, ErrBotIdentityMissing
	}
	if err := s.repo.TouchLastUsed(ctx, auth.Credential.ID); err != nil {
		return nil, err
	}
	return &models.BotPrincipal{BotID: auth.Credential.BotID, IdentityID: auth.IdentityID, CredentialID: auth.Credential.ID}, nil
}

func (s *BotAPICredentialService) RecordConnected(ctx context.Context, principal *models.BotPrincipal, remoteAddr string) error {
	return s.repo.RecordAudit(ctx, principal.CredentialID, principal.BotID, "connected", map[string]any{"remote_addr": remoteAddr})
}

func (s *BotAPICredentialService) RecordInvoked(ctx context.Context, principal *models.BotPrincipal, action string) error {
	return s.repo.RecordAudit(ctx, principal.CredentialID, principal.BotID, "invoked", map[string]any{"action": action})
}

func (s *BotAPICredentialService) assertExternalOwner(ctx context.Context, ownerID, botID uuid.UUID) error {
	bot, err := s.botRepo.FindByID(ctx, botID)
	if err != nil || bot == nil {
		return ErrResourceNotFound
	}
	if bot.OwnerID != ownerID {
		return ErrCredentialForbidden
	}
	if bot.BotType != models.BotTypeExternal {
		return ErrBotNotExternal
	}
	return nil
}

func generateBotToken() (string, []byte, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, "", err
	}
	token := "purr_bot_" + base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))
	return token, hash[:], token[:17], nil
}

func mapCredentialNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrCredentialForbidden
	}
	return err
}
