package models

import (
	"time"

	"github.com/google/uuid"
)

// MsgType 消息类型
type MsgType string

const (
	MsgTypeText  MsgType = "text"
	MsgTypeImage MsgType = "image"
)

// Message 消息模型
type Message struct {
	ID             uuid.UUID `json:"id" db:"id"`
	ConversationID uuid.UUID `json:"conversation_id" db:"conversation_id"`
	SenderID       uuid.UUID `json:"sender_id" db:"sender_id"`
	Content        string    `json:"content" db:"content"`
	MsgType        MsgType   `json:"msg_type" db:"msg_type"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	Sender         *User     `json:"sender,omitempty" db:"-"` // 发送者信息
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
	ID        uuid.UUID        `json:"id" db:"id"`
	UserID    uuid.UUID        `json:"user_id" db:"user_id"`
	FriendID  uuid.UUID        `json:"friend_id" db:"friend_id"`
	Status    FriendshipStatus `json:"status" db:"status"`
	CreatedAt time.Time        `json:"created_at" db:"created_at"`
	User      *User            `json:"user,omitempty" db:"-"`   // 用户信息
	Friend    *User            `json:"friend,omitempty" db:"-"` // 好友信息
}

// FriendListResponse 好友列表响应
type FriendListResponse struct {
	Success bool         `json:"success"`
	Message string       `json:"message,omitempty"`
	Data    []Friendship `json:"data,omitempty"`
}
