package minio

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"purr-chat-server/pkg/config"
	"purr-chat-server/pkg/logger"

	miniosdk "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/set"
)

// Client MinIO 客户端封装
type Client struct {
	client    *miniosdk.Client
	cfg       config.MinIOConfig
	publicURL *url.URL
}

var globalClient *Client

// Init 初始化 MinIO 客户端
func Init(cfg config.MinIOConfig) error {
	client, err := miniosdk.New(cfg.Endpoint, &miniosdk.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return fmt.Errorf("failed to create minio client: %w", err)
	}

	// 解析公共URL
	publicURL, err := url.Parse(cfg.PublicURL)
	if err != nil {
		return fmt.Errorf("failed to parse minio public url: %w", err)
	}

	globalClient = &Client{
		client:    client,
		cfg:       cfg,
		publicURL: publicURL,
	}

	// 确保存储桶存在
	if err := globalClient.ensureBucket(context.Background()); err != nil {
		return fmt.Errorf("failed to ensure bucket: %w", err)
	}

	logger.Info("MinIO client initialized successfully, endpoint:", cfg.Endpoint, "bucket:", cfg.Bucket)
	return nil
}

// GetClient 获取全局 MinIO 客户端
func GetClient() *Client {
	return globalClient
}

// ensureBucket 确保存储桶存在
func (c *Client) ensureBucket(ctx context.Context) error {
	exists, err := c.client.BucketExists(ctx, c.cfg.Bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}
	if exists {
		return nil
	}

	if err := c.client.MakeBucket(ctx, c.cfg.Bucket, miniosdk.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	logger.Info("MinIO bucket created:", c.cfg.Bucket)
	return nil
}

// GenerateObjectKey 生成对象存储 key
// category: avatar / background / chat-image / file
func (c *Client) GenerateObjectKey(userID, category, ext string) string {
	return fmt.Sprintf("%s/%s/%s%s", category, userID, generateFileID(), ext)
}

// GeneratePresignedUploadURL 生成预签名上传 URL
// 返回上传 URL 和对象 key，客户端通过 PUT 请求上传文件
func (c *Client) GeneratePresignedUploadURL(ctx context.Context, objectKey, contentType string, fileSize int64, expires time.Duration) (uploadURL string, err error) {
	reqParams := make(map[string]string)
	reqParams["Content-Type"] = contentType

	presignedURL, err := c.client.PresignedPutObject(ctx, c.cfg.Bucket, objectKey, expires)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned upload url: %w", err)
	}

	return presignedURL.String(), nil
}

// ConfirmUpload 确认文件上传成功
// 验证对象是否存在于 MinIO 中
func (c *Client) ConfirmUpload(ctx context.Context, objectKey string) error {
	_, err := c.client.StatObject(ctx, c.cfg.Bucket, objectKey, miniosdk.StatObjectOptions{})
	if err != nil {
		return fmt.Errorf("file not found in storage: %w", err)
	}
	return nil
}

// GetPresignedDownloadURL 生成预签名下载 URL
func (c *Client) GetPresignedDownloadURL(ctx context.Context, objectKey string, expires time.Duration) (string, error) {
	reqParams := url.Values{}
	reqParams.Set("response-content-disposition", "inline")

	u, err := c.client.PresignedGetObject(ctx, c.cfg.Bucket, objectKey, expires, reqParams)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned download url: %w", err)
	}

	return u.String(), nil
}

// GetObjectInfo 获取对象信息
func (c *Client) GetObjectInfo(ctx context.Context, objectKey string) (*ObjectInfo, error) {
	stat, err := c.client.StatObject(ctx, c.cfg.Bucket, objectKey, miniosdk.StatObjectOptions{})
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

// DeleteObject 删除对象
func (c *Client) DeleteObject(ctx context.Context, objectKey string) error {
	if err := c.client.RemoveObject(ctx, c.cfg.Bucket, objectKey, miniosdk.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

// GeneratePresignedPostPolicy 生成 POST 表单上传策略
// 适用于浏览器直传场景
func (c *Client) GeneratePresignedPostPolicy(ctx context.Context, objectKey string, fileSize int64, expires time.Duration) (*PostPolicyResult, error) {
	policy := miniosdk.NewPostPolicy()
	if err := policy.SetBucket(c.cfg.Bucket); err != nil {
		return nil, err
	}
	if err := policy.SetKey(objectKey); err != nil {
		return nil, err
	}
	if err := policy.SetExpires(time.Now().Add(expires)); err != nil {
		return nil, err
	}
	if err := policy.SetContentLengthRange(1, fileSize); err != nil {
		return nil, err
	}

	u, formData, err := c.client.PresignedPostPolicy(ctx, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to generate post policy: %w", err)
	}

	return &PostPolicyResult{
		URL:      u.String(),
		FormData: formData,
		Key:      objectKey,
	}, nil
}

// SetBucketPublicPolicy 设置存储桶为公开读取
// 仅用于开发环境
func (c *Client) SetBucketPublicPolicy(ctx context.Context) error {
	// 设置存储桶策略为公开读取
	policy := fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {"AWS": ["*"]},
				"Action": ["s3:GetObject"],
				"Resource": ["arn:aws:s3:::%s/*"]
			}
		]
	}`, c.cfg.Bucket)

	// MinIO 使用 SetBucketPolicy 设置策略
	if err := c.client.SetBucketPolicy(ctx, c.cfg.Bucket, policy); err != nil {
		return fmt.Errorf("failed to set bucket policy: %w", err)
	}

	return nil
}

// GetPublicURL 获取文件的公开访问 URL（需要存储桶策略支持）
func (c *Client) GetPublicURL(objectKey string) string {
	return fmt.Sprintf("%s/%s/%s", c.publicURL.String(), c.cfg.Bucket, objectKey)
}

// ObjectInfo 对象信息
type ObjectInfo struct {
	Key          string
	Size         int64
	ContentType  string
	ETag         string
	LastModified time.Time
}

// PostPolicyResult POST 表单上传策略结果
type PostPolicyResult struct {
	URL      string
	FormData map[string]string
	Key      string
}

// ValidateFileSize 校验文件大小
const (
	MaxAvatarSize    = 5 * 1024 * 1024       // 5MB
	MaxBackgroundSize = 10 * 1024 * 1024      // 10MB
	MaxChatImageSize = 20 * 1024 * 1024       // 20MB
	MaxFileSize       = 100 * 1024 * 1024     // 100MB
)

// ValidateFileSize 校验文件大小是否符合类别限制
func ValidateFileSize(category string, size int64) error {
	switch category {
	case "avatar":
		if size > MaxAvatarSize {
			return errors.New("avatar file size must not exceed 5MB")
		}
	case "background":
		if size > MaxBackgroundSize {
			return errors.New("background file size must not exceed 10MB")
		}
	case "chat-image":
		if size > MaxChatImageSize {
			return errors.New("chat image size must not exceed 20MB")
		}
	case "file":
		if size > MaxFileSize {
			return errors.New("file size must not exceed 100MB")
		}
	default:
		if size > MaxFileSize {
			return errors.New("file size must not exceed 100MB")
		}
	}
	return nil
}

// ValidateCategory 校验文件类别
var allowedCategories = set.CreateStringSet("avatar", "background", "chat-image", "file")

func ValidateCategory(category string) error {
	if !allowedCategories.Contains(category) {
		return fmt.Errorf("invalid file category: %s, allowed: avatar, background, chat-image, file", category)
	}
	return nil
}

// ValidateContentType 校验内容类型是否允许
var allowedImageTypes = set.CreateStringSet(
	"image/jpeg", "image/png", "image/gif", "image/webp", "image/bmp", "image/svg+xml",
)

var allowedFileTypes = set.CreateStringSet(
	"image/jpeg", "image/png", "image/gif", "image/webp", "image/bmp", "image/svg+xml",
	"application/pdf",
	"application/msword",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"application/vnd.ms-excel",
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"application/vnd.ms-powerpoint",
	"application/vnd.openxmlformats-officedocument.presentationml.presentation",
	"text/plain",
	"application/zip",
	"application/x-rar-compressed",
	"video/mp4", "video/quicktime",
	"audio/mpeg", "audio/wav", "audio/ogg",
)

func ValidateContentType(category, contentType string) error {
	switch category {
	case "avatar", "background", "chat-image":
		if !allowedImageTypes.Contains(contentType) {
			return fmt.Errorf("content type %s is not allowed for %s", contentType, category)
		}
	default:
		if !allowedFileTypes.Contains(contentType) {
			return fmt.Errorf("content type %s is not allowed", contentType)
		}
	}
	return nil
}
