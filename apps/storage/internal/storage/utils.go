package storage

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

// generateFileID 生成随机文件 ID
func generateFileID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// GetFileExtension 从文件名中提取扩展名（含点号）
func GetFileExtension(filename string) string {
	idx := strings.LastIndex(filename, ".")
	if idx == -1 {
		return ""
	}
	return strings.ToLower(filename[idx:])
}

// ExtractCategoryFromKey 从对象 key 中提取文件类别
// key 格式: category/userID/fileID.ext
func ExtractCategoryFromKey(key string) string {
	parts := strings.SplitN(key, "/", 3)
	if len(parts) < 1 {
		return ""
	}
	return parts[0]
}
