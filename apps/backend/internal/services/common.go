package services

import (
	"context"
	"errors"
	"fmt"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"

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
	ErrInvalidID            = errors.New("invalid id")
	ErrResourceNotFound     = errors.New("resource not found")
	errNotParticipant       = errors.New("not a participant in this conversation")
	errConversationNotFound = errors.New("conversation not found")
	errNotAuthorized        = errors.New("not authorized")
	errOnlyOwnerCanDelete   = errors.New("only owner can delete conversation")
	errOnlyGroupDeletable   = errors.New("can only delete group conversations")
	errCannotSelfChat       = errors.New("cannot create conversation with yourself")
	errTargetNotFound       = errors.New("target user not found")
)

func parseID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: %v", ErrInvalidID, err)
	}
	return id, nil
}

func requireConversationMember(ctx context.Context, enrollmentRepo repository.EnrollmentRepository, conversationID, requesterID uuid.UUID) error {
	enrollment, err := enrollmentRepo.FindByConversationAndUser(ctx, conversationID, requesterID)
	if err != nil || enrollment == nil {
		return ErrResourceNotFound
	}
	return nil
}
