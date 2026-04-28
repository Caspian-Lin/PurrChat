package models

import (
	"time"

	"github.com/google/uuid"
)

// MsgType 消息类型
type MsgType string

const (
	MsgTypeText   MsgType = "text"
	MsgTypeImage  MsgType = "image"
	MsgTypeFile   MsgType = "file"
	MsgTypeSystem MsgType = "system"
)

// SystemMessageContent 系统消息内容
// content 字段存储此 JSON 结构，用于前端渲染可读文本
type SystemMessageContent struct {
	Type     string `json:"type"` // special_mode_start, special_mode_end, bot_deployed, bot_undeployed
	BotID    string `json:"bot_id,omitempty"`
	BotName  string `json:"bot_name,omitempty"`
	UserID   string `json:"user_id,omitempty"`
	UserName string `json:"user_name,omitempty"`
}

// Message 消息模型
type Message struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	ConversationID uuid.UUID  `json:"conversation_id" db:"conversation_id"`
	SenderID       uuid.UUID  `json:"sender_id" db:"sender_id"`
	Content        string     `json:"content" db:"content"`
	MsgType        MsgType    `json:"msg_type" db:"msg_type"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	BotID          *uuid.UUID `json:"bot_id,omitempty" db:"bot_id"`
	BotName        *string    `json:"bot_name,omitempty" db:"bot_name"`
	Sender         *User      `json:"sender,omitempty" db:"-"` // 发送者信息
}

// FriendshipStatus 好友状态
type FriendshipStatus string

const (
	FriendshipStatusPending  FriendshipStatus = "pending"
	FriendshipStatusAccepted FriendshipStatus = "accepted"
	FriendshipStatusRejected FriendshipStatus = "rejected"
	FriendshipStatusBlocked  FriendshipStatus = "blocked"
)

// Friendship 好友关系模型
type Friendship struct {
	ID             uuid.UUID        `json:"id" db:"id"`
	UserID         uuid.UUID        `json:"user_id" db:"user_id"`
	FriendID       uuid.UUID        `json:"friend_id" db:"friend_id"`
	ConversationID uuid.UUID        `json:"conversation_id" db:"conversation_id"` // 关联的会话ID
	Status         FriendshipStatus `json:"status" db:"status"`
	CreatedAt      time.Time        `json:"created_at" db:"created_at"`
	User           *User            `json:"user,omitempty" db:"-"`   // 用户信息
	Friend         *User            `json:"friend,omitempty" db:"-"` // 好友信息
}

// FriendListResponse 好友列表响应
// Deprecated: 使用 models.APIResponse 替代
type FriendListResponse = APIResponse
