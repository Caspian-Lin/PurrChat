package models

import (
    "time"

    "github.com/google/uuid"
)

// ConversationType 会话类型
type ConversationType string

const (
    ConversationTypeDirect ConversationType = "direct" // 私聊（一对一）
    ConversationTypeGroup  ConversationType = "group"  // 群聊
)

// Conversation 会话模型
type Conversation struct {
    ID               uuid.UUID         `json:"id" db:"id"`
    ConversationType ConversationType  `json:"conversation_type" db:"conversation_type"`
    Name             string            `json:"name,omitempty" db:"name"`             // 会话名称（群聊时使用）
    CreatedBy        *uuid.UUID        `json:"created_by,omitempty" db:"created_by"` // 创建者ID
    CreatedAt        time.Time         `json:"created_at" db:"created_at"`
    UpdatedAt        time.Time         `json:"updated_at" db:"updated_at"`
    Members          []*Enrollment     `json:"members,omitempty" db:"-"`           // 会话成员列表
    LastMessage      *Message          `json:"last_message,omitempty" db:"-"`      // 最后一条消息
    UnreadCount      int               `json:"unread_count,omitempty" db:"-"`      // 未读消息数
    FriendshipStatus *FriendshipStatus `json:"friendship_status,omitempty" db:"-"` // 好友关系状态（仅私聊会话）
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
    Content        string    `json:"content" binding:"required,max=10000"`
    MsgType        string    `json:"msg_type" binding:"required,oneof=text image file system"`
}

// MessageResponse 消息响应
type MessageResponse struct {
    Success bool   `json:"success"`
    Message string `json:"message,omitempty"`
    Data    any    `json:"data,omitempty"`
}

// GetMessagesRequest 获取消息请求
type GetMessagesRequest struct {
    ConversationID string `form:"conversation_id" binding:"required,uuid"`
    Limit          int    `form:"limit" binding:"omitempty,min=1,max=100"`
    Offset         int    `form:"offset" binding:"omitempty,min=0"`
}

// GetMessagesIncrementalRequest 增量获取消息请求
type GetMessagesIncrementalRequest struct {
    ConversationID string `form:"conversation_id" binding:"required,uuid"`
    SinceTimestamp int64  `form:"since_timestamp" binding:"required"` // Unix时间戳（毫秒）
}

// MessagesResponse 消息列表响应
type MessagesResponse struct {
    Success bool      `json:"success"`
    Message string    `json:"message,omitempty"`
    Data    []Message `json:"data,omitempty"`
}

// CreateGroupRequest 创建群聊请求
type CreateGroupRequest struct {
    Name    string   `json:"name" binding:"required,min=1,max=100"`
    Members []string `json:"members" binding:"required,min=2"` // 成员用户ID列表
}

// UpdateConversationRequest 更新会话请求
type UpdateConversationRequest struct {
    Name      string `json:"name" binding:"omitempty,min=1,max=100"`
    AvatarURL string `json:"avatar_url" binding:"omitempty,uri"`
}

// DeleteConversationRequest 删除会话请求
type DeleteConversationRequest struct {
    ConversationID uuid.UUID `json:"conversation_id" binding:"required,uuid"`
}

// ConversationMemberResponse 会话成员响应
type ConversationMemberResponse struct {
    Success bool         `json:"success"`
    Message string       `json:"message,omitempty"`
    Data    []Enrollment `json:"data,omitempty"`
}

// HandleFriendRequestRequest 处理好友请求请求
type HandleFriendRequestRequest struct {
    ConversationID uuid.UUID `json:"conversation_id" binding:"omitempty,uuid"`
    Action         string    `json:"action" binding:"required,oneof=accept reject"`
}

// HandleFriendRequestResponse 处理好友请求响应
type HandleFriendRequestResponse struct {
    Success bool   `json:"success"`
    Message string `json:"message,omitempty"`
    Data    any    `json:"data,omitempty"`
}
