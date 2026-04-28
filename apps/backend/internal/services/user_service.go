package services

import (
	"context"

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
func (s *UserService) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 清除密码相关字段
	sanitizeUser(user)

	return user, nil
}
