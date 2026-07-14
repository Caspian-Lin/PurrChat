package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// BotType Bot 应用类型
type BotType string

const (
	BotTypeBuiltin  BotType = "builtin"
	BotTypeWorkflow BotType = "workflow"
	BotTypeExternal BotType = "external"
)

// Discoverability Bot 可发现性(与安装授权解耦)
type Discoverability string

const (
	DiscoverabilityUnlisted Discoverability = "unlisted" // 默认,不可被搜索
	DiscoverabilityListed   Discoverability = "listed"   // 可被搜索
	DiscoverabilityFeatured Discoverability = "featured" // 官方推荐
)

// InstallationTargetType 安装目标类型
type InstallationTargetType string

const (
	InstallationTargetUser         InstallationTargetType = "user"
	InstallationTargetConversation InstallationTargetType = "conversation"
)

// DiagnosticsConsent 诊断数据共享授权
type DiagnosticsConsent string

const (
	DiagnosticsDenied  DiagnosticsConsent = "denied"
	DiagnosticsGranted DiagnosticsConsent = "granted"
)

// InstallationStatus 安装状态
type InstallationStatus string

const (
	InstallationActive   InstallationStatus = "active"
	InstallationPaused   InstallationStatus = "paused"
	InstallationDisabled InstallationStatus = "disabled"
)

// ─── Capability 常量(与 workflow-types capabilities.ts 保持一致) ────

const (
	CapabilityReadTrigger     = "messages:read_trigger"
	CapabilityReadHistory     = "messages:read_history"
	CapabilitySend            = "messages:send"
	CapabilityMembersRead     = "members:read"
	CapabilityNetworkExternal = "network:external"
	CapabilitySecretsUse      = "secrets:use"
)

// AllCapabilities 全部已知 capability(校验与文档用)
var AllCapabilities = []string{
	CapabilityReadTrigger,
	CapabilityReadHistory,
	CapabilitySend,
	CapabilityMembersRead,
	CapabilityNetworkExternal,
	CapabilitySecretsUse,
}

// HasCapability 判断 capability 集合是否包含指定 capability
func HasCapability(caps []string, cap string) bool {
	for _, c := range caps {
		if c == cap {
			return true
		}
	}
	return false
}

// IsGrantedSubsetOfRequested 校验 granted 是否为 requested 的子集
// 返回 granted 中超出 requested 的违规 capability 列表(空表示合法)
func IsGrantedSubsetOfRequested(granted, requested []string) []string {
	reqSet := make(map[string]bool, len(requested))
	for _, r := range requested {
		reqSet[r] = true
	}
	var violations []string
	for _, g := range granted {
		if !reqSet[g] {
			violations = append(violations, g)
		}
	}
	return violations
}

// ─── 节点类型 → Capability 映射(与 workflow-types capabilities.ts 同步) ────

var nodeCapabilities = map[string][]string{
	"trigger":  {CapabilityReadTrigger},
	"llm":      {CapabilityNetworkExternal, CapabilityReadHistory},
	"tool":     {CapabilityNetworkExternal},
	"dify":     {CapabilityNetworkExternal},
	"n8n":      {CapabilityNetworkExternal},
	"history":  {CapabilityReadHistory},
	"reply":    {CapabilitySend},
	"template": {CapabilitySend},
}

func GetNodeCapabilities(nodeType string) []string {
	return nodeCapabilities[nodeType]
}

// BotIdentity 系统身份投影(不可登录、不可好友;仅用于 message.sender_id)
type BotIdentity struct {
	AppID       uuid.UUID `json:"app_id"`
	UserID      uuid.UUID `json:"user_id"`
	DisplayName string    `json:"display_name"`
	AvatarURL   string    `json:"avatar_url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// BotInstallation Bot 安装记录(统一替代 friendship + bot_deployments 的安装语义)
type BotInstallation struct {
	ID                  uuid.UUID              `json:"id"`
	AppID               uuid.UUID              `json:"app_id"`
	InstalledBy         uuid.UUID              `json:"installed_by"`
	TargetType          InstallationTargetType `json:"target_type"`
	TargetID            uuid.UUID              `json:"target_id"`
	GrantedCapabilities []string               `json:"granted_capabilities"`
	DiagnosticsConsent  DiagnosticsConsent     `json:"diagnostics_consent"`
	Status              InstallationStatus     `json:"status"`
	Config              json.RawMessage        `json:"config,omitempty"`
	InstalledAt         time.Time              `json:"installed_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
	// 关联数据(查询时填充)
	App *Bot `json:"app,omitempty"`
	// 展示用：目标会话名称与类型（service 层填充）
	TargetName     string `json:"target_name,omitempty"`
	TargetConvType string `json:"target_conversation_type,omitempty"`
}

// CreateInstallationRequest 安装 Bot 请求
type CreateInstallationRequest struct {
	TargetType InstallationTargetType `json:"target_type" binding:"required,oneof=user conversation"`
	TargetID   uuid.UUID              `json:"target_id" binding:"required,uuid"`
	// GrantedCapabilities 安装者授予的能力(可选;为空则授予 Bot 声明的全部 requested)
	GrantedCapabilities []string `json:"granted_capabilities" binding:"omitempty"`
	// DiagnosticsConsent 是否允许 Bot 所有者查看诊断数据(默认 denied)
	DiagnosticsConsent DiagnosticsConsent `json:"diagnostics_consent" binding:"omitempty,oneof=denied granted"`
}

// UpdateInstallationRequest 更新安装(暂停/恢复/重新授权)
type UpdateInstallationRequest struct {
	Status              InstallationStatus `json:"status" binding:"omitempty,oneof=active paused disabled"`
	GrantedCapabilities []string           `json:"granted_capabilities" binding:"omitempty"`
	DiagnosticsConsent  DiagnosticsConsent `json:"diagnostics_consent" binding:"omitempty,oneof=denied granted"`
}

// InstallationListResponse 安装列表响应
type InstallationListResponse struct {
	Installations []*BotInstallation `json:"installations"`
	Total         int                `json:"total"`
}
