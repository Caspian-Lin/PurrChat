package models

import (
	"time"

	"github.com/google/uuid"
)

// ConversationType 会话类型
type ConversationType string

const (
	ConversationTypeFriend   ConversationType = "friend"
	ConversationTypeStranger ConversationType = "stranger"
)

// RequestStatus 请求状态
type RequestStatus string

const (
	RequestStatusNone     RequestStatus = "none"
	RequestStatusPending  RequestStatus = "pending"
	RequestStatusAccepted RequestStatus = "accepted"
	RequestStatusRejected RequestStatus = "rejected"
)

// Conversation 会话模型
type Conversation struct {
	ID                uuid.UUID        `json:"id" db:"id"`
	ConversationType  ConversationType `json:"conversation_type" db:"conversation_type"`
	User1ID           uuid.UUID        `json:"user1_id" db:"user1_id"`
	User2ID           uuid.UUID        `json:"user2_id" db:"user2_id"`
	HasPendingRequest bool             `json:"has_pending_request" db:"has_pending_request"`
	RequestStatus     RequestStatus    `json:"request_status" db:"request_status"`
	CreatedAt         time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at" db:"updated_at"`
	User1             *User            `json:"user1,omitempty" db:"-"`        // 关联的用户信息
	User2             *User            `json:"user2,omitempty" db:"-"`        // 关联的用户信息
	LastMessage       *Message         `json:"last_message,omitempty" db:"-"` // 最后一条消息
	UnreadCount       int              `json:"unread_count,omitempty" db:"-"` // 未读消息数
}

// ConversationListResponse 会话列表响应
type ConversationListResponse struct {
	Success bool           `json:"success"`
	Message string         `json:"message,omitempty"`
	Data    []Conversation `json:"data,omitempty"`
}

// SendMessageRequest 发送消息请求
type SendMessageRequest struct {
	ConversationID uuid.UUID `json:"conversation_id" binding:"required,uuid"`
	Content        string    `json:"content" binding:"required,max=5000"`
	MsgType        string    `json:"msg_type" binding:"required,oneof=text image"`
}

// MessageResponse 消息响应
type MessageResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

// GetMessagesRequest 获取消息请求
type GetMessagesRequest struct {
	ConversationID uuid.UUID `form:"conversation_id" binding:"required,uuid"`
	Limit          int       `form:"limit" binding:"omitempty,min=1,max=100"`
	Offset         int       `form:"offset" binding:"omitempty,min=0"`
}

// MessagesResponse 消息列表响应
type MessagesResponse struct {
	Success bool      `json:"success"`
	Message string    `json:"message,omitempty"`
	Data    []Message `json:"data,omitempty"`
}

// HandleFriendRequestRequest 处理好友请求
type HandleFriendRequestRequest struct {
	ConversationID uuid.UUID `json:"conversation_id" binding:"required,uuid"`
	Action         string    `json:"action" binding:"required,oneof=accept reject"` // accept 或 reject
}

// HandleFriendRequestResponse 处理好友请求响应
type HandleFriendRequestResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}
