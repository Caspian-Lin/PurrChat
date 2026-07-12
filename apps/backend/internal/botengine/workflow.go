// workflow.go — 工作流部署状态管理
// 实际工作流执行由 TS bot-engine 处理，Go 端仅维护部署状态（active/inactive）。
package botengine

import (
	"context"
	"fmt"
	"time"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/websocket"
	"purr-chat-server/pkg/logger"

	"github.com/google/uuid"
)

// EventPort 事件端口定义（序列化到 mechanism_config）
type EventPort struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	DataType  string `json:"dataType"`
	Direction string `json:"direction"`
}

// FlowConnection 端口化连线
type FlowConnection struct {
	ID           string `json:"id"`
	SourceNodeID string `json:"sourceNodeId"`
	SourcePortID string `json:"sourcePortId"`
	TargetNodeID string `json:"targetNodeId"`
	TargetPortID string `json:"targetPortId"`
}

// Position 节点在画布中的位置
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// WorkflowEvent 事件链中的单个事件
type WorkflowEvent struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Name     string         `json:"name"`
	Config   map[string]any `json:"config"`
	Ports    []EventPort    `json:"ports,omitempty"`
	Position *Position      `json:"position,omitempty"`
}

// EndCondition 工作流结束条件
type EndCondition struct {
	Type    string `json:"type"`
	Pattern string `json:"pattern,omitempty"`
	Value   int    `json:"value,omitempty"`
}

// WorkflowSession 工作流部署会话状态
type WorkflowSession struct {
	ConversationID uuid.UUID
	BotID          uuid.UUID
	BotName        string
	Config         *WorkflowSpec
	Round          int
	StartedAt      time.Time
	ContextBuffer  []ContextMessage
	EventOutputs   map[string]string
	Variables      map[string]string
}

// GetSessionKey 获取工作流会话的存储键
func GetSessionKey(conversationID, botID uuid.UUID) string {
	return conversationID.String() + ":" + botID.String()
}

// ActivateWorkflow 激活工作流（API 手动激活，从 mechanism_config 查找）
func (e *BotEngine) ActivateWorkflow(ctx context.Context, botID, conversationID uuid.UUID) error {
	bot, err := e.botRepo.FindByID(ctx, botID)
	if err != nil {
		return fmt.Errorf("bot not found: %w", err)
	}

	// 从 mechanism_config 中查找 workflow 类型的机制
	mechConfig, err := ParseMechanismConfig(bot.MechanismConfig)
	if err != nil {
		return fmt.Errorf("invalid mechanism config: %w", err)
	}

	workflowMech := FindWorkflowMechanism(mechConfig.Mechanisms)
	if workflowMech == nil || workflowMech.Reply.Workflow == nil {
		return fmt.Errorf("no workflow mechanism found in bot config")
	}

	return e.activateWorkflowWithSpec(ctx, bot, conversationID, workflowMech.Reply.Workflow)
}

// activateWorkflowWithSpec 使用给定的 WorkflowSpec 激活工作流
func (e *BotEngine) activateWorkflowWithSpec(ctx context.Context, bot *models.Bot, conversationID uuid.UUID, spec *WorkflowSpec) error {
	// 检查是否已有活跃的工作流
	sessionKey := GetSessionKey(conversationID, bot.ID)
	if _, exists := e.workflowSessions.Load(sessionKey); exists {
		return fmt.Errorf("workflow already active for this bot in this conversation")
	}

	if spec == nil || len(spec.Events) == 0 {
		return fmt.Errorf("no events defined in workflow config")
	}

	// 创建会话
	session := &WorkflowSession{
		ConversationID: conversationID,
		BotID:          bot.ID,
		BotName:        bot.Name,
		Config:         spec,
		Round:          0,
		StartedAt:      time.Now().UTC(),
		ContextBuffer:  []ContextMessage{},
		EventOutputs:   map[string]string{},
		Variables:      map[string]string{},
	}

	e.workflowSessions.Store(sessionKey, session)

	// 更新数据库中的部署状态
	deployment, err := e.deployRepo.FindByBotAndConversation(ctx, bot.ID, conversationID)
	if err == nil {
		now := time.Now().UTC()
		deployment.WorkflowActive = true
		deployment.WorkflowStartedAt = &now
		if updateErr := e.deployRepo.Update(ctx, deployment); updateErr != nil {
			logger.ErrorfWithCaller("[BotEngine] Failed to update deployment workflow status: %v", updateErr)
		}
	}

	// WebSocket 广播
	e.broadcastWorkflowEvent(ctx, conversationID, "bot_workflow_started", bot.ID.String(), bot.Name)

	// 插入系统消息
	e.sendSystemMessage(ctx, conversationID, &models.SystemMessageContent{
		Type:    "workflow_start",
		BotID:   bot.ID.String(),
		BotName: bot.Name,
	})

	logger.InfofWithCaller("[BotEngine] Workflow activated: bot=%s, conversation=%s", bot.Name, conversationID)
	return nil
}

// DeactivateWorkflow 停用工作流
func (e *BotEngine) DeactivateWorkflow(ctx context.Context, botID, conversationID uuid.UUID) error {
	sessionKey := GetSessionKey(conversationID, botID)

	session, exists := e.workflowSessions.Load(sessionKey)
	if !exists {
		return fmt.Errorf("workflow not active")
	}

	s := session.(*WorkflowSession)
	botName := s.BotName

	e.workflowSessions.Delete(sessionKey)

	// 更新数据库
	deployment, err := e.deployRepo.FindByBotAndConversation(ctx, botID, conversationID)
	if err == nil {
		deployment.WorkflowActive = false
		deployment.WorkflowStartedAt = nil
		if updateErr := e.deployRepo.Update(ctx, deployment); updateErr != nil {
			logger.ErrorfWithCaller("[BotEngine] Failed to update deployment workflow status: %v", updateErr)
		}
	}

	// WebSocket 广播
	e.broadcastWorkflowEvent(ctx, conversationID, "bot_workflow_ended", botID.String(), botName)

	// 插入系统消息
	e.sendSystemMessage(ctx, conversationID, &models.SystemMessageContent{
		Type:    "workflow_end",
		BotID:   botID.String(),
		BotName: botName,
	})

	logger.InfofWithCaller("[BotEngine] Workflow deactivated: bot=%s, conversation=%s", botName, conversationID)
	return nil
}

// IsWorkflowActive 检查工作流是否活跃
func (e *BotEngine) IsWorkflowActive(conversationID, botID uuid.UUID) bool {
	sessionKey := GetSessionKey(conversationID, botID)
	_, exists := e.workflowSessions.Load(sessionKey)
	return exists
}

// broadcastWorkflowEvent 广播工作流状态变更事件
func (e *BotEngine) broadcastWorkflowEvent(ctx context.Context, conversationID uuid.UUID, eventType, botID, botName string) {
	if websocket.GlobalHub == nil {
		return
	}

	members, err := e.enrollmentRepo.FindByConversationID(ctx, conversationID)
	if err != nil {
		return
	}

	for _, m := range members {
		websocket.GlobalHub.SendToUser(m.UserID, eventType, map[string]any{
			"bot_id":          botID,
			"bot_name":        botName,
			"conversation_id": conversationID.String(),
		})
	}
}
