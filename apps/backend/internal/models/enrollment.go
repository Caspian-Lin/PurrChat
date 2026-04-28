package models

import (
	"time"

	"github.com/google/uuid"
)

// EnrollmentRole 用户在会话中的角色
type EnrollmentRole string

const (
	EnrollmentRoleOwner  EnrollmentRole = "owner"
	EnrollmentRoleAdmin  EnrollmentRole = "admin"
	EnrollmentRoleMember EnrollmentRole = "member"
)

// Enrollment 用户与会话的关联模型
type Enrollment struct {
	ID             uuid.UUID      `json:"id" db:"id"`
	ConversationID uuid.UUID      `json:"conversation_id" db:"conversation_id"`
	UserID         uuid.UUID      `json:"user_id" db:"user_id"`
	Role           EnrollmentRole `json:"role" db:"role"`
	JoinedAt       time.Time      `json:"joined_at" db:"joined_at"`
	LastReadAt     *time.Time     `json:"last_read_at,omitempty" db:"last_read_at"`
	User           *User          `json:"user,omitempty" db:"-"` // 关联的用户信息
}

// AddMemberRequest 添加成员请求
type AddMemberRequest struct {
	ConversationID uuid.UUID `json:"conversation_id" binding:"required,uuid"`
	UserID         uuid.UUID `json:"user_id" binding:"required,uuid"`
	Role           string    `json:"role" binding:"required,oneof=owner admin member"`
}

// RemoveMemberRequest 移除成员请求
type RemoveMemberRequest struct {
	ConversationID uuid.UUID `json:"conversation_id" binding:"required,uuid"`
	UserID         uuid.UUID `json:"user_id" binding:"required,uuid"`
}

// UpdateMemberRoleRequest 更新成员角色请求
type UpdateMemberRoleRequest struct {
	ConversationID uuid.UUID `json:"conversation_id" binding:"required,uuid"`
	UserID         uuid.UUID `json:"user_id" binding:"required,uuid"`
	Role           string    `json:"role" binding:"required,oneof=owner admin member"`
}
