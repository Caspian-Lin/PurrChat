package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"sync"
)

// MasterKeyEnv 主密钥环境变量名
const MasterKeyEnv = "PURRCHAT_MASTER_KEY"

// ErrMasterKeyNotSet 主密钥未配置
var ErrMasterKeyNotSet = errors.New(
	MasterKeyEnv + " is not set; generate a 32-byte key (e.g. `openssl rand -base64 32`) " +
		"and export it as " + MasterKeyEnv)

// ErrInvalidMasterKey 主密钥格式错误
var ErrInvalidMasterKey = errors.New(MasterKeyEnv + " must be a valid base64- or hex-encoded 32-byte key")

// SecretCipher 应用层加密接口（AES-256-GCM）。
// 未来可替换为 KMS 后端，不破坏调用方。
type SecretCipher interface {
	// Encrypt 加密明文，返回 base64(IV||ciphertext+tag)
	Encrypt(plaintext string) (string, error)
	// Decrypt 解密 base64(IV||ciphertext+tag)，返回明文
	Decrypt(ciphertext string) (string, error)
}

// aesGCMCipher AES-256-GCM 实现
type aesGCMCipher struct {
	aead cipher.AEAD
}

var (
	defaultCipher     SecretCipher
	defaultCipherErr  error
	defaultCipherOnce sync.Once
)

// GetDefaultCipher 返回基于 PURRCHAT_MASTER_KEY 的全局单例 cipher。
// 主密钥缺失时返回错误，调用方决定是否致命（secret 相关功能不可用）。
func GetDefaultCipher() (SecretCipher, error) {
	defaultCipherOnce.Do(func() {
		raw := os.Getenv(MasterKeyEnv)
		c, err := NewCipherFromKey(raw)
		if err != nil {
			defaultCipherErr = err
			return
		}
		defaultCipher = c
	})
	return defaultCipher, defaultCipherErr
}

// NewCipherFromKey 从 base64 或 hex 编码的 32 字节密钥创建 cipher。
// 空字符串返回 ErrMasterKeyNotSet。
func NewCipherFromKey(encodedKey string) (SecretCipher, error) {
	if encodedKey == "" {
		return nil, ErrMasterKeyNotSet
	}

	key, err := decodeKey(encodedKey)
	if err != nil {
		return nil, ErrInvalidMasterKey
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("%w: decoded length is %d bytes, expected 32", ErrInvalidMasterKey, len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return &aesGCMCipher{aead: aead}, nil
}

// decodeKey 尝试 base64 然后尝试 hex 解码
func decodeKey(s string) ([]byte, error) {
	if b, err := base64.StdEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	if b, err := base64.URLEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	// hex
	b := make([]byte, len(s)/2)
	for i := range b {
		_, err := fmt.Sscanf(s[i*2:i*2+2], "%02x", &b[i])
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

func (c *aesGCMCipher) Encrypt(plaintext string) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// 密文追加到 nonce 后，Base64 整体编码
	ciphertext := c.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (c *aesGCMCipher) Decrypt(encoded string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to base64-decode ciphertext: %w", err)
	}

	ns := c.aead.NonceSize()
	if len(raw) < ns {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := raw[:ns], raw[ns:]
	plaintext, err := c.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}
	return string(plaintext), nil
}
