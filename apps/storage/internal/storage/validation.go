package storage

import (
	"errors"
	"fmt"
)

// ValidateFileSize 校验文件大小是否符合类别限制
const (
	MaxAvatarSize      = 2 * 1024 * 1024  // 2MB
	MaxBackgroundSize  = 10 * 1024 * 1024 // 10MB
	MaxChatImageSize   = 20 * 1024 * 1024 // 20MB
	MaxFileSize        = 100 * 1024 * 1024 // 100MB
)

// ValidateFileSize 校验文件大小
func ValidateFileSize(category string, size int64) error {
	switch category {
	case "avatar":
		if size > MaxAvatarSize {
			return errors.New("avatar file size must not exceed 2MB")
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

// allowedCategories 允许的文件类别
var allowedCategories = map[string]bool{
	"avatar": true, "background": true, "chat-image": true, "file": true,
}

// ValidateCategory 校验文件类别
func ValidateCategory(category string) error {
	if !allowedCategories[category] {
		return fmt.Errorf("invalid file category: %s, allowed: avatar, background, chat-image, file", category)
	}
	return nil
}

// allowedImageTypes 允许的图片类型
var allowedImageTypes = map[string]bool{
	"image/jpeg": true, "image/png": true, "image/gif": true,
	"image/webp": true, "image/bmp": true, "image/svg+xml": true,
}

// allowedFileTypes 允许的所有文件类型
var allowedFileTypes = map[string]bool{
	"image/jpeg": true, "image/png": true, "image/gif": true,
	"image/webp": true, "image/bmp": true, "image/svg+xml": true,
	"application/pdf": true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"application/vnd.ms-excel": true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
	"application/vnd.ms-powerpoint": true,
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
	"text/plain": true,
	"application/zip": true,
	"application/x-rar-compressed": true,
	"video/mp4": true, "video/quicktime": true,
	"audio/mpeg": true, "audio/wav": true, "audio/ogg": true,
}

// ValidateContentType 校验内容类型是否允许
func ValidateContentType(category, contentType string) error {
	switch category {
	case "avatar", "background", "chat-image":
		if !allowedImageTypes[contentType] {
			return fmt.Errorf("content type %s is not allowed for %s", contentType, category)
		}
	default:
		if !allowedFileTypes[contentType] {
			return fmt.Errorf("content type %s is not allowed", contentType)
		}
	}
	return nil
}
