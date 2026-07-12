// Deprecated: 工作流执行已迁移至 TS 微服务 (apps/bot-engine)。
// Go 端仅保留 ActivateWorkflow/DeactivateWorkflow 用于会话状态管理，
// 实际执行由 TS WorkflowRuntime 处理。待 TS 接管后移除。
// 迁移状态：handler 仍引用 Activate/Deactivate，执行路径已切至 TS。
package botengine

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"purr-chat-server/internal/botengine/sandbox"
	"purr-chat-server/internal/messaging"
	"purr-chat-server/internal/models"
	"purr-chat-server/internal/websocket"
	"purr-chat-server/pkg/logger"

	"github.com/google/uuid"
)

// EventPort 事件端口定义
type EventPort struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	DataType  string `json:"dataType"`  // "string" | "number" | "boolean" | "trigger" | "any"
	Direction string `json:"direction"` // "input" | "output"
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
	Type     string         `json:"type"` // "llm" | "builtin" | "python" | "reply" | "trigger" | "if" | "loop" | "wait" | "end"
	Name     string         `json:"name"`
	Config   map[string]any `json:"config"`
	Ports    []EventPort    `json:"ports,omitempty"`    // 端口定义（流程引擎）
	Position *Position      `json:"position,omitempty"` // 画布位置
}

// EndCondition 工作流结束条件
type EndCondition struct {
	Type    string `json:"type"` // "message_match" | "max_rounds" | "timeout"
	Pattern string `json:"pattern,omitempty"`
	Value   int    `json:"value,omitempty"`
}

// WorkflowSession 工作流运行时会话
type WorkflowSession struct {
	ConversationID uuid.UUID
	BotID          uuid.UUID
	BotName        string
	Config         *WorkflowSpec
	Round          int
	StartedAt      time.Time
	ContextBuffer  []ContextMessage  // 会话级上下文
	EventOutputs   map[string]string // 事件输出缓存: eventID → output
	Variables      map[string]string // 用户变量
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

// HandleWorkflow 处理工作流下的消息
func (e *BotEngine) HandleWorkflow(ctx context.Context, msg *BotMessage, bot *models.Bot, deployment *models.BotDeployment, spec *WorkflowSpec) {
	sessionKey := GetSessionKey(msg.ConversationID, bot.ID)

	sessionVal, exists := e.workflowSessions.Load(sessionKey)
	if !exists {
		// 数据库标记活跃但内存中不存在（服务器重启），恢复会话
		if spec == nil {
			// 尝试从 mechanism_config 查找
			mechConfig, err := ParseMechanismConfig(bot.MechanismConfig)
			if err != nil {
				return
			}
			workflowMech := FindWorkflowMechanism(mechConfig.Mechanisms)
			if workflowMech == nil || workflowMech.Reply.Workflow == nil {
				return
			}
			spec = workflowMech.Reply.Workflow
		}
		newSession := &WorkflowSession{
			ConversationID: msg.ConversationID,
			BotID:          bot.ID,
			BotName:        bot.Name,
			Config:         spec,
			Round:          0,
			StartedAt:      time.Now().UTC(),
			ContextBuffer:  []ContextMessage{},
			EventOutputs:   map[string]string{},
			Variables:      map[string]string{},
		}
		e.workflowSessions.Store(sessionKey, newSession)
		sessionVal = newSession
	}

	session := sessionVal.(*WorkflowSession)

	// 增加轮次
	session.Round++

	// 将用户消息加入上下文
	session.ContextBuffer = append(session.ContextBuffer, ContextMessage{
		Role:    "user",
		Content: msg.Content,
		MsgType: msg.MsgType,
	})

	// 限制上下文大小
	maxContext := 100
	if len(session.ContextBuffer) > maxContext {
		session.ContextBuffer = session.ContextBuffer[len(session.ContextBuffer)-maxContext:]
	}

	// 检查结束条件
	if e.checkEndConditions(session, msg.Content) {
		logger.InfofWithCaller("[BotEngine] End condition met: bot=%s, conversation=%s, round=%d", bot.Name, msg.ConversationID, session.Round)
		_ = e.DeactivateWorkflow(ctx, bot.ID, msg.ConversationID)
		return
	}

	// 注入消息元数据到会话变量（供 trigger 节点端口使用）
	session.Variables["username"] = msg.SenderName
	session.Variables["time"] = msg.CreatedAt.Format("15:04")
	session.Variables["sender_id"] = msg.SenderID.String()

	// 使用端口化流程引擎执行事件链
	flowCtx := NewExecutionContext(spec.Events, spec.Connections, session)
	reply, err := flowCtx.ExecuteFlow(ctx, e, msg.Content)
	if err != nil {
		logger.ErrorfWithCaller("[BotEngine] Event chain execution failed: bot=%s, error=%v", bot.Name, err)
		reply = "..."
	}

	if reply == "" {
		reply = "..."
	}

	// 将 Bot 回复加入上下文
	session.ContextBuffer = append(session.ContextBuffer, ContextMessage{
		Role:    "assistant",
		Content: reply,
	})

	// 发送回复
	if e.messageSender != nil {
		if _, err := e.messageSender.SendBotMessage(ctx, &messaging.BotSendRequest{
			BotID:          bot.ID,
			ConversationID: msg.ConversationID,
			Content:        reply,
			MsgType:        "text",
			Source:         messaging.SourceWorkflow,
		}); err != nil {
			logger.ErrorfWithCaller("[BotEngine] Failed to send workflow reply: %v", err)
		}
	}
}

// executeEvent 执行单个事件
func (e *BotEngine) executeEvent(ctx context.Context, session *WorkflowSession, event *WorkflowEvent, input string) (string, error) {
	switch event.Type {
	case "llm":
		return e.executeLLMEvent(ctx, session, event, input)
	case "builtin":
		return e.executeBuiltinEvent(ctx, session, event, input)
	case "python":
		return e.executePythonEvent(ctx, session, event, input)
	case "reply":
		return e.executeReplyEvent(ctx, session, event, input)
	default:
		return "", fmt.Errorf("unknown event type: %s", event.Type)
	}
}

// executeLLMEvent 执行 LLM 事件
func (e *BotEngine) executeLLMEvent(ctx context.Context, session *WorkflowSession, event *WorkflowEvent, input string) (string, error) {
	config := event.Config

	// 构建 LLM 配置
	llmConfig := &LLMConfig{
		APIURL:       getStringField(config, "api_url"),
		APIKey:       getStringField(config, "api_key"),
		Model:        getStringField(config, "model"),
		SystemPrompt: getStringField(config, "system_prompt"),
	}

	if temp, ok := config["temperature"].(float64); ok {
		llmConfig.Temperature = temp
	}
	if maxTokens, ok := config["max_tokens"].(float64); ok {
		llmConfig.MaxTokens = int(maxTokens)
	}
	if contextWindow, ok := config["context_window"].(float64); ok {
		llmConfig.ContextWindow = int(contextWindow)
	}

	// 收集上下文
	contextMessages := e.collectEventContext(ctx, session, event)

	return CallLLM(ctx, llmConfig, input, contextMessages)
}

// executeReplyEvent 执行回复事件
func (e *BotEngine) executeReplyEvent(_ context.Context, session *WorkflowSession, event *WorkflowEvent, input string) (string, error) {
	template := getStringField(event.Config, "template")
	if template == "" {
		return input, nil
	}

	result := template

	// 替换 {nodeName.portName} 格式（人类可读）
	if session.Config != nil {
		for _, evt := range session.Config.Events {
			for _, port := range evt.Ports {
				if port.Direction == "output" {
					ref := "{" + evt.Name + "." + port.Name + "}"
					if output, ok := session.EventOutputs[evt.ID]; ok && port.ID == "out_output" {
						result = strings.ReplaceAll(result, ref, output)
					}
				}
			}
		}
	}

	// 替换事件输出变量 $evt_ID.output（向后兼容）
	for evtID, output := range session.EventOutputs {
		result = strings.ReplaceAll(result, "$"+evtID+".output", output)
	}

	// 替换会话变量
	for key, value := range session.Variables {
		result = strings.ReplaceAll(result, "$"+key, value)
	}

	// 替换 args 变量（{args}, {args:N}）
	result = ReplaceArgsVars(result, input)

	return result, nil
}

// executeBuiltinEvent 执行内置事件
func (e *BotEngine) executeBuiltinEvent(_ context.Context, session *WorkflowSession, event *WorkflowEvent, input string) (string, error) {
	return executeBuiltinHandler(event.Config, input, session.Variables)
}

// executePythonEvent 执行 Python 事件
func (e *BotEngine) executePythonEvent(ctx context.Context, _ *WorkflowSession, event *WorkflowEvent, input string) (string, error) {
	return sandbox.ExecutePythonEvent(ctx, event.Config, input)
}

// collectEventContext 根据 context_scope 收集事件所需的上下文
func (e *BotEngine) collectEventContext(_ context.Context, session *WorkflowSession, event *WorkflowEvent) []ContextMessage {
	scope := getStringField(event.Config, "context_scope")
	if scope == "" || scope == "session" {
		// 默认：整个会话上下文
		return session.ContextBuffer
	}

	if nStr, ok := strings.CutPrefix(scope, "last:"); ok {
		// 最近 N 条消息
		n, err := strconv.Atoi(nStr)
		if err != nil || n <= 0 {
			return session.ContextBuffer
		}
		if len(session.ContextBuffer) > n {
			return session.ContextBuffer[len(session.ContextBuffer)-n:]
		}
		return session.ContextBuffer
	}

	if _, ok := strings.CutPrefix(scope, "from_event:"); ok {
		// 从指定事件开始的消息（暂不实现，返回全部上下文）
		return session.ContextBuffer
	}

	return session.ContextBuffer
}

// checkEndConditions 检查工作流是否应该结束
func (e *BotEngine) checkEndConditions(session *WorkflowSession, content string) bool {
	for _, cond := range session.Config.EndConditions {
		switch cond.Type {
		case "message_match":
			if cond.Pattern != "" && strings.Contains(content, cond.Pattern) {
				return true
			}
		case "max_rounds":
			if cond.Value > 0 && session.Round >= cond.Value {
				return true
			}
		case "timeout":
			elapsed := time.Since(session.StartedAt)
			if cond.Value > 0 && int(elapsed.Minutes()) >= cond.Value {
				return true
			}
		}
	}
	return false
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

// ExecuteSimpleFlow 无状态执行编译后的简单工作流（不需要持久化 WorkflowSession）
// 用于 predefined/llm 机制的底层统一执行
func (e *BotEngine) ExecuteSimpleFlow(ctx context.Context, spec *WorkflowSpec, msg *BotMessage, bot *models.Bot, contextMessages []ContextMessage) (string, error) {
	// 创建临时的无状态 session
	tempSession := &WorkflowSession{
		BotID:         bot.ID,
		BotName:       bot.Name,
		Config:        spec,
		ContextBuffer: contextMessages,
		EventOutputs:  map[string]string{},
		Variables: map[string]string{
			"username": msg.SenderName,
			"time":     msg.CreatedAt.Format("15:04"),
		},
	}

	// 预设 trigger 节点的输出端口值（绕过当前 flow engine 不注入 trigger 端口值的问题）
	tempSession.EventOutputs["compiled_trigger"] = msg.Content
	tempSession.EventOutputs["compiled_trigger:out_output"] = msg.Content

	flowCtx := NewExecutionContext(spec.Events, spec.Connections, tempSession)
	return flowCtx.ExecuteFlow(ctx, e, msg.Content)
}

// GetActiveWorkflowSession 获取活跃的工作流会话（供调试面板使用）
func (e *BotEngine) GetActiveWorkflowSession(conversationID, botID uuid.UUID) *WorkflowSession {
	sessionKey := GetSessionKey(conversationID, botID)
	val, exists := e.workflowSessions.Load(sessionKey)
	if !exists {
		return nil
	}
	return val.(*WorkflowSession)
}

// getStringField 从 map[string]any 中安全获取字符串字段
func getStringField(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	val, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := val.(string)
	if !ok {
		return ""
	}
	return s
}
