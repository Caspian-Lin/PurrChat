package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"
	minioclient "purr-chat-server/pkg/minio"

	"github.com/google/uuid"
)

// 文件上传 URL 有效期
const uploadURLExpires = 15 * time.Minute
const downloadURLExpires = 1 * time.Hour

// FileService 文件服务
type FileService struct {
	fileRepo repository.FileRepository
}

// NewFileService 创建文件服务
func NewFileService(fileRepo repository.FileRepository) *FileService {
	return &FileService{fileRepo: fileRepo}
}

// RequestUpload 申请上传
// 1. 校验请求参数（文件大小、类别、类型）
// 2. 生成对象 key
// 3. 写入数据库（confirmed=false）
// 4. 生成预签名上传 URL
func (s *FileService) RequestUpload(ctx context.Context, userID string, req *models.UploadRequest) (*models.UploadResponse, error) {
	// 校验文件类别
	if err := minioclient.ValidateCategory(req.Category); err != nil {
		return nil, err
	}

	// 校验内容类型
	if err := minioclient.ValidateContentType(req.Category, req.ContentType); err != nil {
		return nil, err
	}

	// 校验文件大小
	if err := minioclient.ValidateFileSize(req.Category, req.FileSize); err != nil {
		return nil, err
	}

	uploaderUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	client := minioclient.GetClient()
	if client == nil {
		return nil, errors.New("file storage service is not available")
	}

	// 提取文件扩展名
	ext := minioclient.GetFileExtension(req.FileName)
	if ext == "" {
		ext = contentTypeToExt(req.ContentType)
	}

	// 生成对象 key
	objectKey := client.GenerateObjectKey(userID, req.Category, ext)

	// 生成预签名上传 URL
	uploadURL, err := client.GeneratePresignedUploadURL(ctx, objectKey, req.ContentType, req.FileSize, uploadURLExpires)
	if err != nil {
		return nil, fmt.Errorf("failed to generate upload url: %w", err)
	}

	// 写入数据库（预创建记录，confirmed=false）
	meta := &models.FileMetadata{
		ID:          uuid.New(),
		ObjectKey:   objectKey,
		FileName:    req.FileName,
		FileSize:    req.FileSize,
		ContentType: req.ContentType,
		Category:    models.FileCategory(req.Category),
		Usage:       models.FileUsage(req.Usage),
		UploaderID:  uploaderUUID,
		Confirmed:   false,
	}

	if err := s.fileRepo.Create(ctx, meta); err != nil {
		return nil, fmt.Errorf("failed to create file record: %w", err)
	}

	return &models.UploadResponse{
		UploadID:  meta.ID.String(),
		ObjectKey: objectKey,
		UploadURL: uploadURL,
		Method:    "PUT",
		ExpiresIn: int(uploadURLExpires.Seconds()),
	}, nil
}

// ConfirmUpload 确认上传
// 1. 根据 uploadID 查询数据库记录
// 2. 向 MinIO 验证文件是否存在
// 3. 获取对象信息（ETag、大小等）
// 4. 更新数据库（confirmed=true，写入 public_url）
func (s *FileService) ConfirmUpload(ctx context.Context, userID string, req *models.ConfirmUploadRequest) (*models.ConfirmUploadResponse, error) {
	uploadID, err := uuid.Parse(req.UploadID)
	if err != nil {
		return nil, fmt.Errorf("invalid upload id: %w", err)
	}

	// 查询文件记录
	meta, err := s.fileRepo.GetByID(ctx, uploadID)
	if err != nil {
		return nil, fmt.Errorf("failed to query file record: %w", err)
	}
	if meta == nil {
		return nil, errors.New("upload record not found")
	}

	// 验证上传者身份
	if meta.UploaderID.String() != userID {
		return nil, errors.New("permission denied: not the uploader")
	}

	// 验证对象 key 一致
	if meta.ObjectKey != req.ObjectKey {
		return nil, errors.New("object key mismatch")
	}

	// 已确认的不能重复确认
	if meta.Confirmed {
		return nil, errors.New("upload already confirmed")
	}

	client := minioclient.GetClient()
	if client == nil {
		return nil, errors.New("file storage service is not available")
	}

	// 验证文件在 MinIO 中存在
	if err := client.ConfirmUpload(ctx, meta.ObjectKey); err != nil {
		return nil, fmt.Errorf("file verification failed: %w", err)
	}

	// 获取对象信息
	info, err := client.GetObjectInfo(ctx, meta.ObjectKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get object info: %w", err)
	}

	// 生成公开 URL
	publicURL := client.GetPublicURL(meta.ObjectKey)

	// 更新数据库
	if err := s.fileRepo.ConfirmUpload(ctx, uploadID, info.ETag, publicURL); err != nil {
		return nil, fmt.Errorf("failed to confirm upload: %w", err)
	}

	return &models.ConfirmUploadResponse{
		FileID:    uploadID,
		ObjectKey: meta.ObjectKey,
		PublicURL: publicURL,
	}, nil
}

// GetDownloadURL 获取预签名下载 URL
func (s *FileService) GetDownloadURL(ctx context.Context, userID string, fileID string) (*models.DownloadURLResponse, error) {
	id, err := uuid.Parse(fileID)
	if err != nil {
		return nil, fmt.Errorf("invalid file id: %w", err)
	}

	meta, err := s.fileRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to query file record: %w", err)
	}
	if meta == nil {
		return nil, errors.New("file not found")
	}

	if !meta.Confirmed {
		return nil, errors.New("file upload not confirmed yet")
	}

	client := minioclient.GetClient()
	if client == nil {
		return nil, errors.New("file storage service is not available")
	}

	// 生成预签名下载 URL
	downloadURL, err := client.GetPresignedDownloadURL(ctx, meta.ObjectKey, downloadURLExpires)
	if err != nil {
		return nil, fmt.Errorf("failed to generate download url: %w", err)
	}

	return &models.DownloadURLResponse{
		DownloadURL: downloadURL,
		ExpiresIn:   int(downloadURLExpires.Seconds()),
	}, nil
}

// DeleteFile 删除文件（从 MinIO 和数据库中同时删除）
func (s *FileService) DeleteFile(ctx context.Context, userID string, fileID string) error {
	id, err := uuid.Parse(fileID)
	if err != nil {
		return fmt.Errorf("invalid file id: %w", err)
	}

	meta, err := s.fileRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to query file record: %w", err)
	}
	if meta == nil {
		return errors.New("file not found")
	}

	// 只有上传者可以删除
	if meta.UploaderID.String() != userID {
		return errors.New("permission denied: not the uploader")
	}

	client := minioclient.GetClient()

	// 从 MinIO 删除
	if client != nil {
		if err := client.DeleteObject(ctx, meta.ObjectKey); err != nil {
			// 记录日志但不阻断流程（可能 MinIO 中已不存在）
			// logger.Warnf("failed to delete object from minio: %v", err)
			_ = err
		}
	}

	// 从数据库删除
	return s.fileRepo.DeleteByID(ctx, id)
}

// contentTypeToExt 根据 MIME 类型推断文件扩展名
func contentTypeToExt(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/bmp":
		return ".bmp"
	case "image/svg+xml":
		return ".svg"
	case "application/pdf":
		return ".pdf"
	case "text/plain":
		return ".txt"
	case "video/mp4":
		return ".mp4"
	case "audio/mpeg":
		return ".mp3"
	case "audio/wav":
		return ".wav"
	case "application/zip":
		return ".zip"
	default:
		return ""
	}
}
