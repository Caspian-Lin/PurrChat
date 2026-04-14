package storage

import (
	"context"
	"time"
)

// StorageProvider 对象存储后端接口
// MinIO 和 R2 均实现此接口
type StorageProvider interface {
	// Initialize 初始化存储后端（创建存储桶等）
	Initialize(ctx context.Context) error

	// GenerateObjectKey 生成对象存储 key
	GenerateObjectKey(userID, category, ext string) string

	// GeneratePresignedUploadURL 生成预签名上传 URL
	GeneratePresignedUploadURL(ctx context.Context, objectKey, contentType string, fileSize int64, expires time.Duration) (string, error)

	// ConfirmUpload 确认文件上传成功
	ConfirmUpload(ctx context.Context, objectKey string) error

	// GetObjectInfo 获取对象信息
	GetObjectInfo(ctx context.Context, objectKey string) (*ObjectInfo, error)

	// GetPresignedDownloadURL 生成预签名下载 URL
	GetPresignedDownloadURL(ctx context.Context, objectKey string, expires time.Duration) (string, error)

	// DeleteObject 删除对象
	DeleteObject(ctx context.Context, objectKey string) error

	// GetPublicURL 获取文件的公开访问 URL
	GetPublicURL(objectKey string) string
}

// ObjectInfo 对象信息
type ObjectInfo struct {
	Key          string
	Size         int64
	ContentType  string
	ETag         string
	LastModified time.Time
}
