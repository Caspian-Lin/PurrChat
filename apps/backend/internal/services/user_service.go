package services

import (
	"context"
	"errors"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"

	"github.com/google/uuid"
)

// UserService 用户服务
type UserService struct {
	userRepo repository.UserRepository
}

// NewUserService 创建用户服务
func NewUserService(userRepo repository.UserRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

// GetUserByID 根据ID获取用户
// viewerID 为查看者ID，查看自己返回完整信息，查看他人返回脱敏信息
func (s *UserService) GetUserByID(ctx context.Context, userID string, viewerID string) (*models.User, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 禁止查看系统占位用户信息
	if id == deletedUserID {
		return nil, errors.New("user not found")
	}

	if id.String() == viewerID {
		sanitizeUser(user)
	} else {
		sanitizePublicProfile(user)
	}

	return user, nil
}
