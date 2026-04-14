package storage

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"purr-chat-storage/pkg/config"
	"purr-chat-storage/pkg/logger"

	miniosdk "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOProvider MinIO 存储后端实现
type MinIOProvider struct {
	client    *miniosdk.Client
	cfg       config.StorageConfig
	publicURL *url.URL
}

// NewMinIOProvider 创建 MinIO 存储提供者
func NewMinIOProvider(cfg config.StorageConfig) *MinIOProvider {
	client, err := miniosdk.New(cfg.Endpoint, &miniosdk.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil
	}

	publicURL, err := url.Parse(cfg.PublicURL)
	if err != nil {
		return nil
	}

	return &MinIOProvider{
		client:    client,
		cfg:       cfg,
		publicURL: publicURL,
	}
}

// Initialize 初始化 MinIO 后端，确保存储桶存在
func (p *MinIOProvider) Initialize(ctx context.Context) error {
	if p.client == nil {
		return fmt.Errorf("minio client not initialized")
	}

	exists, err := p.client.BucketExists(ctx, p.cfg.Bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}
	if exists {
		return nil
	}

	if err := p.client.MakeBucket(ctx, p.cfg.Bucket, miniosdk.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	logger.Info("MinIO bucket created:", p.cfg.Bucket)
	return nil
}

// GenerateObjectKey 生成对象存储 key
func (p *MinIOProvider) GenerateObjectKey(userID, category, ext string) string {
	return fmt.Sprintf("%s/%s/%s%s", category, userID, generateFileID(), ext)
}

// GeneratePresignedUploadURL 生成预签名上传 URL
func (p *MinIOProvider) GeneratePresignedUploadURL(ctx context.Context, objectKey, contentType string, fileSize int64, expires time.Duration) (string, error) {
	presignedURL, err := p.client.PresignedPutObject(ctx, p.cfg.Bucket, objectKey, expires)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned upload url: %w", err)
	}
	return presignedURL.String(), nil
}

// ConfirmUpload 验证文件存在于 MinIO 中
func (p *MinIOProvider) ConfirmUpload(ctx context.Context, objectKey string) error {
	_, err := p.client.StatObject(ctx, p.cfg.Bucket, objectKey, miniosdk.StatObjectOptions{})
	if err != nil {
		return fmt.Errorf("file not found in storage: %w", err)
	}
	return nil
}

// GetObjectInfo 获取对象信息
func (p *MinIOProvider) GetObjectInfo(ctx context.Context, objectKey string) (*ObjectInfo, error) {
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
func (p *MinIOProvider) GetPresignedDownloadURL(ctx context.Context, objectKey string, expires time.Duration) (string, error) {
	reqParams := url.Values{}
	reqParams.Set("response-content-disposition", "inline")

	u, err := p.client.PresignedGetObject(ctx, p.cfg.Bucket, objectKey, expires, reqParams)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned download url: %w", err)
	}
	return u.String(), nil
}

// DeleteObject 删除对象
func (p *MinIOProvider) DeleteObject(ctx context.Context, objectKey string) error {
	if err := p.client.RemoveObject(ctx, p.cfg.Bucket, objectKey, miniosdk.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

// GetPublicURL 获取文件的公开访问 URL
func (p *MinIOProvider) GetPublicURL(objectKey string) string {
	return fmt.Sprintf("%s/%s/%s", p.publicURL.String(), p.cfg.Bucket, objectKey)
}
