package services

import (
	"errors"

	"purr-chat-server/internal/models"

	"github.com/google/uuid"
)

// deletedUserID 系统占位用户 ID，用于消息匿名化，禁止出现在业务查询中
var deletedUserID = uuid.MustParse("00000000-0000-0000-0000-000000000000")

// sanitizeUser 清除用户密码相关字段
func sanitizeUser(u *models.User) {
	u.PasswordHash = ""
	u.Salt = ""
}

// sanitizePublicProfile 返回脱敏后的用户信息（仅公开字段）
func sanitizePublicProfile(u *models.User) {
	u.PasswordHash = ""
	u.Salt = ""
	u.Email = ""
	u.Phone = ""
	u.EmailVerified = false
	u.PhoneVerified = false
}

// 服务层共享错误
var (
	errNotParticipant       = errors.New("not a participant in this conversation")
	errConversationNotFound = errors.New("conversation not found")
	errNotAuthorized        = errors.New("not authorized")
	errOnlyOwnerCanDelete   = errors.New("only owner can delete conversation")
	errOnlyGroupDeletable   = errors.New("can only delete group conversations")
	errCannotSelfChat       = errors.New("cannot create conversation with yourself")
	errTargetNotFound       = errors.New("target user not found")
)
