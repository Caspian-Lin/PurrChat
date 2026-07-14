package tests

import (
	"context"
	"testing"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"
	"purr-chat-server/internal/security"
	"purr-chat-server/internal/services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBotAppSecretStore secret 加密存储 CRUD + 运行时解密注入
func TestBotAppSecretStore(t *testing.T) {
	// 设置主密钥(测试用固定 32 字节密钥)
	t.Setenv("PURRCHAT_MASTER_KEY", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")

	ctx := context.Background()
	SetupTestDB(t)

	// 准备 owner + bot
	ownerRepo := repository.NewUserRepository()
	owner, err := createTestUser(ctx, ownerRepo, "secret-owner")
	require.NoError(t, err)
	ownerID := owner.ID
	botRepo := repository.NewBotRepository()
	secretRepo := repository.NewBotAppSecretRepository()
	secretService := services.NewSecretService(secretRepo, botRepo)

	bot := &models.Bot{
		OwnerID:               ownerID,
		Name:                  "Secret Bot",
		Status:                models.BotStatusActive,
		BotType:               models.BotTypeWorkflow,
		Discoverability:       models.DiscoverabilityUnlisted,
		RequestedCapabilities: []string{models.CapabilitySecretsUse},
	}
	require.NoError(t, botRepo.Create(ctx, bot))

	// --- SetSecret ---
	require.NoError(t, secretService.SetSecret(ctx, ownerID, bot.ID, "openai_key", "sk-test-1234567890"))
	require.NoError(t, secretService.SetSecret(ctx, ownerID, bot.ID, "webhook_url", "https://hooks.example.com/abc"))

	// --- ListSecrets 不返回明文 ---
	keys, err := secretService.ListSecrets(ctx, ownerID, bot.ID)
	require.NoError(t, err)
	require.Len(t, keys, 2)
	for _, k := range keys {
		assert.NotEmpty(t, k.KeyName)
		assert.True(t, k.HasValue)
		assert.Empty(t, k.Ciphertext) // 密文不外泄
	}

	// --- 密文确实存在(DB 层) ---
	raw, err := secretRepo.Get(ctx, bot.ID, "openai_key")
	require.NoError(t, err)
	assert.NotEqual(t, "sk-test-1234567890", raw.Ciphertext, "ciphertext must differ from plaintext")
	assert.Contains(t, raw.Ciphertext, "=") // base64

	// --- ResolveSecrets 运行时解密 ---
	dec, err := secretService.ResolveSecrets(ctx, bot.ID)
	require.NoError(t, err)
	assert.Equal(t, "sk-test-1234567890", dec["openai_key"])
	assert.Equal(t, "https://hooks.example.com/abc", dec["webhook_url"])

	// --- 非 owner 无权操作 ---
	other, err := createTestUser(ctx, ownerRepo, "secret-other")
	require.NoError(t, err)
	err = secretService.SetSecret(ctx, other.ID, bot.ID, "hacked", "value")
	assert.Error(t, err)

	// --- DeleteSecret ---
	require.NoError(t, secretService.DeleteSecret(ctx, ownerID, bot.ID, "openai_key"))
	keys, err = secretService.ListSecrets(ctx, ownerID, bot.ID)
	require.NoError(t, err)
	assert.Len(t, keys, 1)

	// --- Update(upsert)同一个 key 覆盖 ---
	require.NoError(t, secretService.SetSecret(ctx, ownerID, bot.ID, "webhook_url", "https://new.url/x"))
	dec, err = secretService.ResolveSecrets(ctx, bot.ID)
	require.NoError(t, err)
	assert.Equal(t, "https://new.url/x", dec["webhook_url"])
	assert.Len(t, dec, 1)
}

// TestSecretCipherUnavailable 主密钥未配置时 SetSecret 报错
func TestSecretCipherUnavailable(t *testing.T) {
	// 清除主密钥(确保 GetDefaultCipher 失败)
	// 注意: GetDefaultCipher 用 sync.Once,如果其他测试已设置过会缓存。
	// 这里独立验证 NewCipherFromKey 的空 key 行为。
	_, err := security.NewCipherFromKey("")
	assert.ErrorIs(t, err, security.ErrMasterKeyNotSet)
}
