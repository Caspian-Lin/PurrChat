package services

import (
	"context"
	"errors"
	"time"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"
	"purr-chat-server/pkg/hash"
	"purr-chat-server/pkg/jwt"
	"purr-chat-server/pkg/logger"

	"github.com/google/uuid"
)

// AuthService 认证服务
type AuthService struct {
	userRepo repository.UserRepository
	jwtKey   string
}

// NewAuthService 创建认证服务
func NewAuthService(userRepo repository.UserRepository, jwtKey string) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		jwtKey:   jwtKey,
	}
}

// Register 用户注册
func (s *AuthService) Register(ctx context.Context, req *models.RegisterRequest) (*models.LoginResponse, error) {
	logger.InfofWithCaller("Registering user: %s", req.Username)

	// 检查用户名是否已存在
	_, err := s.userRepo.FindByUsername(ctx, req.Username)
	if err == nil {
		logger.ErrorfWithCaller("Username already exists: %s", req.Username)
		return nil, errors.New("username already exists")
	}

	// 检查邮箱是否已存在
	if req.Email != "" {
		_, err := s.userRepo.FindByEmail(ctx, req.Email)
		if err == nil {
			logger.ErrorfWithCaller("Email already exists: %s", req.Email)
			return nil, errors.New("email already exists")
		}
	}

	// 检查手机号是否已存在
	if req.Phone != "" {
		_, err := s.userRepo.FindByPhone(ctx, req.Phone)
		if err == nil {
			logger.ErrorfWithCaller("Phone already exists: %s", req.Phone)
			return nil, errors.New("phone already exists")
		}
	}

	// 哈希密码
	salt, passwordHash, err := hash.HashPasswordWithSalt(req.Password)
	if err != nil {
		logger.ErrorfWithCaller("Failed to hash password for user %s: %v", req.Username, err)
		return nil, err
	}

	// 创建用户
	user := &models.User{
		Username:      req.Username,
		PasswordHash:  passwordHash,
		Salt:          salt,
		AvatarURL:     "",
		Email:         req.Email,
		EmailVerified: false,
		Phone:         req.Phone,
		PhoneVerified: false,
	}

	err = s.userRepo.Create(ctx, user)
	if err != nil {
		logger.ErrorfWithCaller("Failed to create user %s: %v", req.Username, err)
		return nil, err
	}

	// 生成 JWT token
	token, err := jwt.GenerateToken(user.ID.String(), s.jwtKey, 24*time.Hour)
	if err != nil {
		logger.ErrorfWithCaller("Failed to generate token for user %s: %v", user.Username, err)
		return nil, err
	}

	logger.InfofWithCaller("User registered successfully: %s (ID: %s)", user.Username, user.ID)

	// 清除密码相关字段
	user.PasswordHash = ""
	user.Salt = ""

	return &models.LoginResponse{
		Token: token,
		User:  user,
	}, nil
}

// Login 用户登录
func (s *AuthService) Login(ctx context.Context, req *models.LoginRequest) (*models.LoginResponse, error) {
	logger.InfofWithCaller("Login attempt for email: %s", req.Email)

	// 查找用户
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		logger.ErrorfWithCaller("User not found for email: %s", req.Email)
		return nil, errors.New("invalid email or password")
	}

	// Bot 账号不能登录
	if user.IsBot {
		logger.InfofWithCaller("Bot account login attempt rejected: %s", user.Username)
		return nil, errors.New("bot accounts cannot login")
	}

	// 验证密码
	valid, err := hash.VerifyPassword(req.Password, user.PasswordHash, user.Salt)
	if err != nil || !valid {
		logger.ErrorfWithCaller("Invalid password for user: %s", user.Username)
		return nil, errors.New("invalid email or password")
	}

	// 生成 JWT token
	token, err := jwt.GenerateToken(user.ID.String(), s.jwtKey, 24*time.Hour)
	if err != nil {
		logger.ErrorfWithCaller("Failed to generate token for user %s: %v", user.Username, err)
		return nil, err
	}

	// 清除密码相关字段
	user.PasswordHash = ""
	user.Salt = ""

	logger.InfofWithCaller("User logged in successfully: %s (ID: %s)", user.Username, user.ID)

	return &models.LoginResponse{
		Token: token,
		User:  user,
	}, nil
}

// GetUserByID 根据ID获取用户
func (s *AuthService) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 清除密码相关字段
	user.PasswordHash = ""
	user.Salt = ""

	return user, nil
}

// SearchUsers 搜索用户（通过UID、手机号或邮箱）
func (s *AuthService) SearchUsers(ctx context.Context, query string) ([]*models.User, error) {
	users, err := s.userRepo.Search(ctx, query)
	if err != nil {
		return nil, err
	}

	// 清除密码相关字段
	for _, user := range users {
		user.PasswordHash = ""
		user.Salt = ""
	}

	return users, nil
}

// UpdateProfile 更新用户个人资料
func (s *AuthService) UpdateProfile(ctx context.Context, userID string, req *models.UpdateProfileRequest) (*models.User, error) {
	logger.InfofWithCaller("Updating profile for user: %s", userID)

	id, err := uuid.Parse(userID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to parse user ID %s: %v", userID, err)
		return nil, err
	}

	// 获取现有用户
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		logger.ErrorfWithCaller("Failed to find user %s: %v", userID, err)
		return nil, err
	}

	// 更新用户名
	if req.Username != "" && req.Username != user.Username {
		existingUser, err := s.userRepo.FindByUsername(ctx, req.Username)
		if err == nil && existingUser.ID != id {
			logger.ErrorfWithCaller("Username already exists: %s", req.Username)
			return nil, errors.New("username already exists")
		}
		user.Username = req.Username
	}

	// 更新字段
	if req.AvatarURL != "" {
		user.AvatarURL = req.AvatarURL
	}
	if req.Email != "" && req.Email != user.Email {
		// 检查邮箱是否已被其他用户使用
		existingUser, err := s.userRepo.FindByEmail(ctx, req.Email)
		if err == nil && existingUser.ID != id {
			logger.ErrorfWithCaller("Email already exists: %s", req.Email)
			return nil, errors.New("email already exists")
		}
		user.Email = req.Email
		user.EmailVerified = false // 更新邮箱后需要重新验证
	}
	if req.Phone != "" && req.Phone != user.Phone {
		// 检查手机号是否已被其他用户使用
		existingUser, err := s.userRepo.FindByPhone(ctx, req.Phone)
		if err == nil && existingUser.ID != id {
			logger.ErrorfWithCaller("Phone already exists: %s", req.Phone)
			return nil, errors.New("phone already exists")
		}
		user.Phone = req.Phone
		user.PhoneVerified = false // 更新手机号后需要重新验证
	}

	// 更新到数据库
	err = s.userRepo.Update(ctx, user)
	if err != nil {
		logger.ErrorfWithCaller("Failed to update profile for user %s: %v", userID, err)
		return nil, err
	}

	logger.InfofWithCaller("Profile updated successfully for user: %s", userID)

	// 清除密码相关字段
	user.PasswordHash = ""
	user.Salt = ""

	return user, nil
}

// ChangePassword 修改用户密码
func (s *AuthService) ChangePassword(ctx context.Context, userID string, req *models.ChangePasswordRequest) error {
	logger.InfofWithCaller("Changing password for user: %s", userID)

	id, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	// 获取用户（含密码哈希）
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return errors.New("user not found")
	}

	// 验证旧密码
	valid, err := hash.VerifyPassword(req.OldPassword, user.PasswordHash, user.Salt)
	if err != nil || !valid {
		return errors.New("invalid current password")
	}

	// 检查新旧密码不能相同
	same, _ := hash.VerifyPassword(req.NewPassword, user.PasswordHash, user.Salt)
	if same {
		return errors.New("new password must be different from current password")
	}

	// 哈希新密码
	newSalt, newHash, err := hash.HashPasswordWithSalt(req.NewPassword)
	if err != nil {
		return err
	}

	// 更新密码
	err = s.userRepo.UpdatePassword(ctx, id, newHash, newSalt)
	if err != nil {
		logger.ErrorfWithCaller("Failed to update password for user %s: %v", userID, err)
		return err
	}

	logger.InfofWithCaller("Password changed successfully for user: %s", userID)
	return nil
}
