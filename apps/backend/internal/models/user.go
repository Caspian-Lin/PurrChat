package models

import (
	"time"

	"github.com/google/uuid"
)

// User 用户模型
type User struct {
	ID            uuid.UUID `json:"id" db:"id"`
	UID           int       `json:"uid" db:"uid"`
	Username      string    `json:"username" db:"username"`
	PasswordHash  string    `json:"-" db:"password_hash"`
	Salt          string    `json:"-" db:"salt"`
	AvatarURL     string    `json:"avatar_url" db:"avatar_url"`
	Email         string    `json:"email,omitempty" db:"email"`
	EmailVerified bool      `json:"email_verified" db:"email_verified"`
	Phone         string    `json:"phone,omitempty" db:"phone"`
	PhoneVerified bool      `json:"phone_verified" db:"phone_verified"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=20"`
	Password string `json:"password" binding:"required,min=6"`
	Email    string `json:"email" binding:"omitempty,email"`
	Phone    string `json:"phone" binding:"omitempty,max=20"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token string `json:"token"`
	User  *User  `json:"user"`
}

// AuthResponse 认证响应
type AuthResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

// SearchUserRequest 搜索用户请求
type SearchUserRequest struct {
	Query string `json:"query" binding:"required,min=1,max=100"` // 可以是uid、手机号或邮箱
}

// FriendRequest 好友请求
type FriendRequest struct {
	TargetUserID string `json:"target_user_id" binding:"required,uuid"`
}

// FriendRequestResponse 好友请求响应
type FriendRequestResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

// UpdateProfileRequest 更新个人资料请求
type UpdateProfileRequest struct {
	Nickname  string `json:"nickname,omitempty" binding:"omitempty,min=1,max=20"`
	AvatarURL string `json:"avatar_url,omitempty" binding:"omitempty,url"`
	Email     string `json:"email,omitempty" binding:"omitempty,email"`
	Phone     string `json:"phone,omitempty" binding:"omitempty,max=20"`
}
