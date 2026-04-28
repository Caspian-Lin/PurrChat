package services

import (
	"errors"

	"purr-chat-server/internal/models"
)

// sanitizeUser 清除用户密码相关字段
func sanitizeUser(u *models.User) {
	u.PasswordHash = ""
	u.Salt = ""
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
