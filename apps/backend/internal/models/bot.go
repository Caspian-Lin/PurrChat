package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// BotStatus Bot 状态
type BotStatus string

const (
	BotStatusActive   BotStatus = "active"
	BotStatusDisabled BotStatus = "disabled"
)

// BotVisibility Bot 可见性
type BotVisibility string

const (
	BotVisibilityPrivate BotVisibility = "private" // 仅创建者可用
	BotVisibilityPublic  BotVisibility = "public"  // 所有人可搜索添加
	BotVisibilityGlobal  BotVisibility = "global"  // 系统级 Bot
)

// Bot Bot 模型(演进为 BotApp 等价物,见 docs/bot-engine/BOT_APP_MODEL.md)
type Bot struct {
	ID                    uuid.UUID       `json:"id"`
	OwnerID               uuid.UUID       `json:"owner_id"`
	Name                  string          `json:"name"`
	AvatarURL             string          `json:"avatar_url"`
	Description           string          `json:"description"`
	Status                BotStatus       `json:"status"`
	Visibility            BotVisibility   `json:"visibility"` // deprecated, #36 迁移后移除
	MechanismConfig       json.RawMessage `json:"mechanism_config"`
	BotType               BotType         `json:"bot_type"`
	Discoverability       Discoverability `json:"discoverability"`
	IsSystem              bool            `json:"is_system"`
	PublishedVersion      *int            `json:"published_version,omitempty"`
	RequestedCapabilities []string        `json:"requested_capabilities"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
}

// CreateBotRequest 创建 Bot 请求
type CreateBotRequest struct {
	Name            string          `json:"name" binding:"required,min=1,max=40"`
	AvatarURL       string          `json:"avatar_url" binding:"omitempty,uri"`
	Description     string          `json:"description" binding:"omitempty,max=500"`
	Discoverability Discoverability `json:"discoverability" binding:"omitempty,oneof=unlisted listed featured"`
	// Visibility deprecated,兼容旧客户端;若传则映射到 discoverability
	Visibility BotVisibility `json:"visibility" binding:"omitempty,oneof=private public global"`
}

// UpdateBotRequest 更新 Bot 请求
type UpdateBotRequest struct {
	Name                  string          `json:"name" binding:"omitempty,min=1,max=40"`
	AvatarURL             string          `json:"avatar_url" binding:"omitempty,uri"`
	Description           string          `json:"description" binding:"omitempty,max=500"`
	Status                BotStatus       `json:"status" binding:"omitempty,oneof=active disabled"`
	Visibility            BotVisibility   `json:"visibility" binding:"omitempty,oneof=private public global"`
	MechanismConfig       json.RawMessage `json:"mechanism_config"`
	RequestedCapabilities []string        `json:"requested_capabilities" binding:"omitempty"`
}

// DeployBotRequest 部署 Bot 请求
type DeployBotRequest struct {
	ConversationID uuid.UUID `json:"conversation_id" binding:"required,uuid"`
}

// UpdateDeploymentStatusRequest 更新部署状态请求
type UpdateDeploymentStatusRequest struct {
	ConversationID uuid.UUID `json:"conversation_id" binding:"required,uuid"`
	Status         string    `json:"status" binding:"required,oneof=active paused"`
}

// ActivateWorkflowRequest 激活/停用工作流请求
type ActivateWorkflowRequest struct {
	ConversationID uuid.UUID `json:"conversation_id" binding:"required,uuid"`
}

// PublicBotDetail 公开 Bot 详情（含统计数据）
type PublicBotDetail struct {
	Bot
	DeploymentCount int    `json:"deployment_count"`
	OwnerName       string `json:"owner_name"`
	TriggerSummary  string `json:"trigger_summary"`
	ReplyType       string `json:"reply_type"`
}

// PaginatedSearchResult 分页搜索结果
type PaginatedSearchResult struct {
	Bots   []*PublicBotDetail `json:"bots"`
	Total  int                `json:"total"`
	Limit  int                `json:"limit"`
	Offset int                `json:"offset"`
}

// DeployableConversation 可部署的会话
type DeployableConversation struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	MemberCount int       `json:"member_count"`
}

// BotDeploymentStatus 部署状态
type BotDeploymentStatus string

const (
	BotDeploymentActive BotDeploymentStatus = "active"
	BotDeploymentPaused BotDeploymentStatus = "paused"
)

// BotDeployment Bot 部署模型
type BotDeployment struct {
	ID                uuid.UUID           `json:"id"`
	BotID             uuid.UUID           `json:"bot_id"`
	ConversationID    uuid.UUID           `json:"conversation_id"`
	DeployedBy        uuid.UUID           `json:"deployed_by"`
	Status            BotDeploymentStatus `json:"status"`
	WorkflowActive    bool                `json:"workflow_active"`
	WorkflowStartedAt *time.Time          `json:"workflow_started_at,omitempty"`
	DeployedAt        time.Time           `json:"deployed_at"`
	// 关联数据（查询时填充）
	Bot          *Bot          `json:"bot,omitempty"`
	Conversation *Conversation `json:"conversation,omitempty"`
}

// BotCallLog Bot 调用日志
type BotCallLog struct {
	ID               uuid.UUID `json:"id"`
	BotID            uuid.UUID `json:"bot_id"`
	ConversationID   uuid.UUID `json:"conversation_id"`
	SenderID         uuid.UUID `json:"sender_id"`
	SenderName       string    `json:"sender_name"`
	TriggerMessage   string    `json:"trigger_message"`
	ReplyContent     string    `json:"reply_content"`
	MechanismID      string    `json:"mechanism_id"`
	MechanismName    string    `json:"mechanism_name"`
	ReplyType        string    `json:"reply_type"`
	ExecutionPath    string    `json:"execution_path"`
	Success          bool      `json:"success"`
	ErrorMessage     string    `json:"error_message,omitempty"`
	DurationMs       int       `json:"duration_ms"`
	CreatedAt        time.Time `json:"created_at"`
	ConversationName string    `json:"conversation_name,omitempty"`
}

// BotCallLogListResponse 调用日志列表响应
type BotCallLogListResponse struct {
	Logs   []*BotCallLog `json:"logs"`
	Total  int           `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}
