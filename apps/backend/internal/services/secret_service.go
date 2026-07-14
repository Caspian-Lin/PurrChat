package services

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"
	"purr-chat-server/internal/security"

	"github.com/google/uuid"
)

// 错误
var (
	ErrSecretCipherUnavailable = errors.New("secret encryption is not configured (PURRCHAT_MASTER_KEY missing)")
)

// keyName 合法字符:[a-zA-Z0-9_],1-64 位
var keyNameRe = regexp.MustCompile(`^[a-zA-Z0-9_]{1,64}$`)

// SecretService secret 管理(加密存储 + 解密读取)
type SecretService struct {
	repo    repository.BotAppSecretRepository
	botRepo repository.BotRepository
}

func NewSecretService(repo repository.BotAppSecretRepository, botRepo repository.BotRepository) *SecretService {
	return &SecretService{repo: repo, botRepo: botRepo}
}

// SetSecret 加密并存储 secret(仅 owner 可调用)
func (s *SecretService) SetSecret(ctx context.Context, ownerID, appID uuid.UUID, keyName, plaintext string) error {
	if !keyNameRe.MatchString(keyName) {
		return fmt.Errorf("invalid key_name: must match [a-zA-Z0-9_]{1,64}")
	}
	if err := s.assertOwner(ctx, ownerID, appID); err != nil {
		return err
	}

	cipher, err := security.GetDefaultCipher()
	if err != nil {
		return ErrSecretCipherUnavailable
	}

	ciphertext, err := cipher.Encrypt(plaintext)
	if err != nil {
		return fmt.Errorf("failed to encrypt secret: %w", err)
	}

	return s.repo.Set(ctx, appID, keyName, ciphertext)
}

// ListSecrets 返回 key 列表(不含明文)
func (s *SecretService) ListSecrets(ctx context.Context, ownerID, appID uuid.UUID) ([]*models.BotAppSecret, error) {
	if err := s.assertOwner(ctx, ownerID, appID); err != nil {
		return nil, err
	}
	return s.repo.ListKeys(ctx, appID)
}

// DeleteSecret 删除 secret(仅 owner)
func (s *SecretService) DeleteSecret(ctx context.Context, ownerID, appID uuid.UUID, keyName string) error {
	if err := s.assertOwner(ctx, ownerID, appID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, appID, keyName)
}

// ResolveSecrets 运行时批量解密 secret,返回 key->明文 映射。
// 仅在 secrets:use 已授予时由 engine 调用。失败不阻塞整体(跳过该 key)。
func (s *SecretService) ResolveSecrets(ctx context.Context, appID uuid.UUID) (map[string]string, error) {
	cipher, err := security.GetDefaultCipher()
	if err != nil {
		return nil, ErrSecretCipherUnavailable
	}

	all, err := s.repo.GetAll(ctx, appID)
	if err != nil {
		return nil, err
	}

	out := make(map[string]string, len(all))
	for _, sc := range all {
		pt, err := cipher.Decrypt(sc.Ciphertext)
		if err != nil {
			// 跳过无法解密的,不泄露错误细节
			continue
		}
		out[sc.KeyName] = pt
	}
	return out, nil
}

// assertOwner 校验 ownerID 是 appID 的所有者
func (s *SecretService) assertOwner(ctx context.Context, ownerID, appID uuid.UUID) error {
	bot, err := s.botRepo.FindByID(ctx, appID)
	if err != nil || bot == nil {
		return errors.New("bot not found")
	}
	if bot.OwnerID != ownerID {
		return errors.New("forbidden: not the bot owner")
	}
	return nil
}
