package hash

import (
	"crypto/rand"
	"encoding/base64"

	"golang.org/x/crypto/argon2"
)

// 使用 Argon2id 进行密码哈希
const (
	time    = 1
	memory  = 64 * 1024
	threads = 4
	keyLen  = 32
	saltLen = 16
)

// GenerateSalt 生成随机盐值
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, saltLen)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, err
	}
	return salt, nil
}

// HashPassword 使用 Argon2id 哈希密码
func HashPassword(password string, salt []byte) string {
	hash := argon2.IDKey([]byte(password), salt, uint32(time), uint32(memory), uint8(threads), uint32(keyLen))
	return base64.StdEncoding.EncodeToString(hash)
}

// HashPasswordWithSalt 生成盐值并哈希密码，返回 base64 编码的盐值和哈希
func HashPasswordWithSalt(password string) (string, string, error) {
	salt, err := GenerateSalt()
	if err != nil {
		return "", "", err
	}

	hash := HashPassword(password, salt)
	return base64.StdEncoding.EncodeToString(salt), hash, nil
}

// VerifyPassword 验证密码
func VerifyPassword(password, hashedPassword, salt string) (bool, error) {
	saltBytes, err := base64.StdEncoding.DecodeString(salt)
	if err != nil {
		return false, err
	}

	hash := HashPassword(password, saltBytes)
	return hash == hashedPassword, nil
}
