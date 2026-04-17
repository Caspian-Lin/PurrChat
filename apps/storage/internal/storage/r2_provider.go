package storage

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"purr-chat-storage/pkg/config"
	"purr-chat-storage/pkg/logger"

	miniosdk "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// R2Provider Cloudflare R2 存储后端实现
// R2 兼容 S3 API，使用 minio-go SDK 连接
type R2Provider struct {
	client    *miniosdk.Client
	cfg       config.StorageConfig
	publicURL *url.URL
}

// NewR2Provider 创建 R2 存储提供者
func NewR2Provider(cfg config.StorageConfig) *R2Provider {
	client, err := miniosdk.New(cfg.Endpoint, &miniosdk.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: true, // R2 始终使用 SSL
		Region: cfg.Region,
	})
	if err != nil {
		return nil
	}

	publicURL, err := url.Parse(cfg.PublicURL)
	if err != nil {
		return nil
	}

	return &R2Provider{
		client:    client,
		cfg:       cfg,
		publicURL: publicURL,
	}
}

// Initialize 初始化 R2 后端，检查存储桶是否存在
// 注意：R2 存储桶需要在 Cloudflare 控制台手动创建
func (p *R2Provider) Initialize(ctx context.Context) error {
	if p.client == nil {
		return fmt.Errorf("r2 client not initialized")
	}

	exists, err := p.client.BucketExists(ctx, p.cfg.Bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}
	if exists {
		return nil
	}

	// R2 不支持通过 API 创建存储桶，提示用户手动创建
	return fmt.Errorf("R2 bucket '%s' does not exist. Please create it in the Cloudflare dashboard", p.cfg.Bucket)
}

// GenerateObjectKey 生成对象存储 key
func (p *R2Provider) GenerateObjectKey(userID, category, ext string) string {
	return fmt.Sprintf("%s/%s/%s%s", category, userID, generateFileID(), ext)
}

// GeneratePresignedUploadURL 生成预签名上传 URL
func (p *R2Provider) GeneratePresignedUploadURL(ctx context.Context, objectKey, contentType string, fileSize int64, expires time.Duration) (string, error) {
	presignedURL, err := p.client.PresignedPutObject(ctx, p.cfg.Bucket, objectKey, expires)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned upload url: %w", err)
	}
	return presignedURL.String(), nil
}

// ConfirmUpload 验证文件存在于 R2 中
func (p *R2Provider) ConfirmUpload(ctx context.Context, objectKey string) error {
	_, err := p.client.StatObject(ctx, p.cfg.Bucket, objectKey, miniosdk.StatObjectOptions{})
	if err != nil {
		return fmt.Errorf("file not found in storage: %w", err)
	}
	return nil
}

// GetObjectInfo 获取对象信息
func (p *R2Provider) GetObjectInfo(ctx context.Context, objectKey string) (*ObjectInfo, error) {
	stat, err := p.client.StatObject(ctx, p.cfg.Bucket, objectKey, miniosdk.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object info: %w", err)
	}
	return &ObjectInfo{
		Key:          stat.Key,
		Size:         stat.Size,
		ContentType:  stat.ContentType,
		ETag:         stat.ETag,
		LastModified: stat.LastModified,
	}, nil
}

// GetPresignedDownloadURL 生成预签名下载 URL
func (p *R2Provider) GetPresignedDownloadURL(ctx context.Context, objectKey string, expires time.Duration) (string, error) {
	reqParams := url.Values{}
	reqParams.Set("response-content-disposition", "inline")

	u, err := p.client.PresignedGetObject(ctx, p.cfg.Bucket, objectKey, expires, reqParams)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned download url: %w", err)
	}
	return u.String(), nil
}

// DeleteObject 删除对象
func (p *R2Provider) DeleteObject(ctx context.Context, objectKey string) error {
	if err := p.client.RemoveObject(ctx, p.cfg.Bucket, objectKey, miniosdk.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

// GetPublicURL 获取文件的公开访问 URL
func (p *R2Provider) GetPublicURL(objectKey string) string {
	if p.publicURL.String() == "" {
		logger.Error("R2 public URL is not configured, returning empty string")
		return ""
	}
	// 清理末尾斜杠，防止生成双斜杠 URL（如 https://pub-xxx.r2.dev//avatar/...）
	base := strings.TrimRight(p.publicURL.String(), "/")
	url := fmt.Sprintf("%s/%s", base, objectKey)
	logger.InfofWithCaller("R2 GetPublicURL: base=%s objectKey=%s result=%s", base, objectKey, url)
	return url
}
