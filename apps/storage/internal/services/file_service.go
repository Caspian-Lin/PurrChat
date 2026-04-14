package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"purr-chat-storage/internal/models"
	"purr-chat-storage/internal/repository"
	"purr-chat-storage/internal/storage"

	"github.com/google/uuid"
)

const uploadURLExpires = 15 * time.Minute
const downloadURLExpires = 1 * time.Hour

// FileService 文件服务
type FileService struct {
	fileRepo repository.FileRepository
	storage  storage.StorageProvider
}

// NewFileService 创建文件服务
func NewFileService(fileRepo repository.FileRepository, storage storage.StorageProvider) *FileService {
	return &FileService{fileRepo: fileRepo, storage: storage}
}

// RequestUpload 申请上传
func (s *FileService) RequestUpload(ctx context.Context, userID string, req *models.UploadRequest) (*models.UploadResponse, error) {
	if s.storage == nil {
		return nil, errors.New("storage service is not available")
	}

	if err := storage.ValidateCategory(req.Category); err != nil {
		return nil, err
	}

	if err := storage.ValidateContentType(req.Category, req.ContentType); err != nil {
		return nil, err
	}

	if err := storage.ValidateFileSize(req.Category, req.FileSize); err != nil {
		return nil, err
	}

	uploaderUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	// 提取文件扩展名
	ext := storage.GetFileExtension(req.FileName)
	if ext == "" {
		ext = contentTypeToExt(req.ContentType)
	}

	// 生成对象 key
	objectKey := s.storage.GenerateObjectKey(userID, req.Category, ext)

	// 生成预签名上传 URL
	uploadURL, err := s.storage.GeneratePresignedUploadURL(ctx, objectKey, req.ContentType, req.FileSize, uploadURLExpires)
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
func (s *FileService) ConfirmUpload(ctx context.Context, userID string, req *models.ConfirmUploadRequest) (*models.ConfirmUploadResponse, error) {
	if s.storage == nil {
		return nil, errors.New("storage service is not available")
	}

	uploadID, err := uuid.Parse(req.UploadID)
	if err != nil {
		return nil, fmt.Errorf("invalid upload id: %w", err)
	}

	meta, err := s.fileRepo.GetByID(ctx, uploadID)
	if err != nil {
		return nil, fmt.Errorf("failed to query file record: %w", err)
	}
	if meta == nil {
		return nil, errors.New("upload record not found")
	}

	if meta.UploaderID.String() != userID {
		return nil, errors.New("permission denied: not the uploader")
	}

	if meta.ObjectKey != req.ObjectKey {
		return nil, errors.New("object key mismatch")
	}

	if meta.Confirmed {
		return nil, errors.New("upload already confirmed")
	}

	// 验证文件在存储中存在
	if err := s.storage.ConfirmUpload(ctx, meta.ObjectKey); err != nil {
		return nil, fmt.Errorf("file verification failed: %w", err)
	}

	// 获取对象信息
	info, err := s.storage.GetObjectInfo(ctx, meta.ObjectKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get object info: %w", err)
	}

	// 生成公开 URL
	publicURL := s.storage.GetPublicURL(meta.ObjectKey)

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
	if s.storage == nil {
		return nil, errors.New("storage service is not available")
	}

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

	downloadURL, err := s.storage.GetPresignedDownloadURL(ctx, meta.ObjectKey, downloadURLExpires)
	if err != nil {
		return nil, fmt.Errorf("failed to generate download url: %w", err)
	}

	return &models.DownloadURLResponse{
		DownloadURL: downloadURL,
		ExpiresIn:   int(downloadURLExpires.Seconds()),
	}, nil
}

// DeleteFile 删除文件
func (s *FileService) DeleteFile(ctx context.Context, userID string, fileID string) error {
	if s.storage == nil {
		return errors.New("storage service is not available")
	}

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

	if meta.UploaderID.String() != userID {
		return errors.New("permission denied: not the uploader")
	}

	if err := s.storage.DeleteObject(ctx, meta.ObjectKey); err != nil {
		_ = err
	}

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
