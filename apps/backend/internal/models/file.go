package models

import (
	"time"

	"github.com/google/uuid"
)

// FileCategory 文件类别
type FileCategory string

const (
	FileCategoryAvatar     FileCategory = "avatar"      // 用户头像
	FileCategoryBackground FileCategory = "background"  // 用户主页背景
	FileCategoryChatImage  FileCategory = "chat-image"  // 聊天图片
	FileCategoryFile       FileCategory = "file"        // 通用文件
)

// FileUsage 文件用途
type FileUsage string

const (
	FileUsageAvatar    FileUsage = "avatar"
	FileUsageBackground FileUsage = "background"
	FileUsageMessage   FileUsage = "message"
	FileUsageTemp      FileUsage = "temp" // 临时上传，待确认
)

// FileMetadata 文件元数据模型
type FileMetadata struct {
	ID          uuid.UUID    `json:"id" db:"id"`
	ObjectKey   string       `json:"object_key" db:"object_key"`       // MinIO 中的对象 key
	FileName    string       `json:"file_name" db:"file_name"`         // 原始文件名
	FileSize    int64        `json:"file_size" db:"file_size"`         // 文件大小（字节）
	ContentType string       `json:"content_type" db:"content_type"`   // MIME 类型
	Category    FileCategory `json:"category" db:"category"`           // 文件类别
	Usage       FileUsage    `json:"usage" db:"usage"`                 // 文件用途
	UploaderID  uuid.UUID    `json:"uploader_id" db:"uploader_id"`     // 上传者 ID
	PublicURL   string       `json:"public_url" db:"public_url"`       // 公开访问 URL
	ETag        string       `json:"etag" db:"etag"`                   // 文件哈希
	Confirmed   bool         `json:"confirmed" db:"confirmed"`         // 上传是否已确认
	CreatedAt   time.Time    `json:"created_at" db:"created_at"`
	ConfirmedAt *time.Time   `json:"confirmed_at,omitempty" db:"confirmed_at"`
}

// UploadRequest 申请上传请求
type UploadRequest struct {
	FileName    string `json:"file_name" binding:"required,min=1,max=255"`
	FileSize    int64  `json:"file_size" binding:"required,gt=0"`
	ContentType string `json:"content_type" binding:"required"`
	Category    string `json:"category" binding:"required"`
	Usage       string `json:"usage" binding:"required"`
}

// UploadResponse 上传申请响应
type UploadResponse struct {
	UploadID  string `json:"upload_id"`
	ObjectKey string `json:"object_key"`
	UploadURL string `json:"upload_url"` // 预签名上传 URL
	Method    string `json:"method"`     // HTTP 方法 PUT
	ExpiresIn int    `json:"expires_in"` // URL 有效期（秒）
}

// ConfirmUploadRequest 确认上传请求
type ConfirmUploadRequest struct {
	UploadID  string `json:"upload_id" binding:"required,uuid"`
	ObjectKey string `json:"object_key" binding:"required"`
}

// ConfirmUploadResponse 确认上传响应
type ConfirmUploadResponse struct {
	FileID    uuid.UUID `json:"file_id"`
	ObjectKey string    `json:"object_key"`
	PublicURL string    `json:"public_url"`
}

// DownloadURLRequest 获取下载链接请求
type DownloadURLRequest struct {
	FileID string `json:"file_id" binding:"required,uuid"`
}

// DownloadURLResponse 下载链接响应
type DownloadURLResponse struct {
	DownloadURL string `json:"download_url"`
	ExpiresIn   int    `json:"expires_in"`
}

// FileResponse 文件信息响应
type FileResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}
