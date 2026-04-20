package botengine

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"purr-chat-server/internal/botengine/sandbox"
	"purr-chat-server/internal/models"
	"purr-chat-server/internal/websocket"
	"purr-chat-server/pkg/logger"

	"github.com/google/uuid"
)

// SpecialModeEvent 事件链中的单个事件
type SpecialModeEvent struct {
	ID     string         `json:"id"`
	Type   string         `json:"type"` // "llm" | "builtin" | "python" | "reply"
	Name   string         `json:"name"`
	Config map[string]any `json:"config"`
	Next   []string       `json:"next"`
}

// EndCondition 特殊模式结束条件
type EndCondition struct {
	Type    string `json:"type"` // "message_match" | "max_rounds" | "timeout"
	Pattern string `json:"pattern,omitempty"`
	Value   int    `json:"value,omitempty"`
}

// SpecialModeSession 特殊模式运行时会话
type SpecialModeSession struct {
	ConversationID uuid.UUID
	BotID          uuid.UUID
	BotName        string
	Config         *SpecialModeSpec
	Round          int
	StartedAt      time.Time
	ContextBuffer  []ContextMessage  // 会话级上下文
	EventOutputs   map[string]string // 事件输出缓存: eventID → output
	Variables      map[string]string // 用户变量
}

// GetSessionKey 获取特殊模式会话的存储键
func GetSessionKey(conversationID, botID uuid.UUID) string {
	return conversationID.String() + ":" + botID.String()
}

// ActivateSpecialMode 激活特殊模式（API 手动激活，从 mechanism_config 查找）
func (e *BotEngine) ActivateSpecialMode(ctx context.Context, botID, conversationID uuid.UUID) error {
	bot, err := e.botRepo.FindByID(ctx, botID)
	if err != nil {
		return fmt.Errorf("bot not found: %w", err)
	}

	// 从 mechanism_config 中查找 special_mode 类型的机制
	mechConfig, err := ParseMechanismConfig(bot.MechanismConfig)
	if err != nil {
		return fmt.Errorf("invalid mechanism config: %w", err)
	}

	specialMech := FindSpecialModeMechanism(mechConfig.Mechanisms)
	if specialMech == nil || specialMech.Reply.SpecialMode == nil {
		return fmt.Errorf("no special mode mechanism found in bot config")
	}

	return e.activateSpecialModeWithSpec(ctx, bot, conversationID, specialMech.Reply.SpecialMode)
}

// activateSpecialModeWithSpec 使用给定的 SpecialModeSpec 激活特殊模式
func (e *BotEngine) activateSpecialModeWithSpec(ctx context.Context, bot *models.Bot, conversationID uuid.UUID, spec *SpecialModeSpec) error {
	// 检查是否已有活跃的特殊模式
	sessionKey := GetSessionKey(conversationID, bot.ID)
	if _, exists := e.specialModeSessions.Load(sessionKey); exists {
		return fmt.Errorf("special mode already active for this bot in this conversation")
	}

	if spec == nil || len(spec.Events) == 0 {
		return fmt.Errorf("no events defined in special mode config")
	}

	// 创建会话
	session := &SpecialModeSession{
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

	e.specialModeSessions.Store(sessionKey, session)

	// 更新数据库中的部署状态
	deployment, err := e.deployRepo.FindByBotAndConversation(ctx, bot.ID, conversationID)
	if err == nil {
		now := time.Now().UTC()
		deployment.SpecialModeActive = true
		deployment.SpecialModeStartedAt = &now
		if updateErr := e.deployRepo.Update(ctx, deployment); updateErr != nil {
			logger.ErrorfWithCaller("[BotEngine] Failed to update deployment special mode status: %v", updateErr)
		}
	}

	// WebSocket 广播
	e.broadcastSpecialModeEvent(ctx, conversationID, "bot_special_mode_started", bot.ID.String(), bot.Name)

	// 插入系统消息
	e.sendSystemMessage(ctx, conversationID, &models.SystemMessageContent{
		Type:    "special_mode_start",
		BotID:   bot.ID.String(),
		BotName: bot.Name,
	})

	logger.InfofWithCaller("[BotEngine] Special mode activated: bot=%s, conversation=%s", bot.Name, conversationID)
	return nil
}

// activateMechanismSpecialMode 从机制触发自动激活特殊模式
func (e *BotEngine) activateMechanismSpecialMode(ctx context.Context, msg *BotMessage, bot *models.Bot, spec *SpecialModeSpec) {
	sessionKey := GetSessionKey(msg.ConversationID, bot.ID)
	if _, exists := e.specialModeSessions.Load(sessionKey); exists {
		return // 已激活，不重复
	}

	if err := e.activateSpecialModeWithSpec(ctx, bot, msg.ConversationID, spec); err != nil {
		logger.ErrorfWithCaller("[BotEngine] Failed to auto-activate special mode for bot %s: %v", bot.ID, err)
	}
}

// DeactivateSpecialMode 停用特殊模式
func (e *BotEngine) DeactivateSpecialMode(ctx context.Context, botID, conversationID uuid.UUID) error {
	sessionKey := GetSessionKey(conversationID, botID)

	session, exists := e.specialModeSessions.Load(sessionKey)
	if !exists {
		return fmt.Errorf("special mode not active")
	}

	s := session.(*SpecialModeSession)
	botName := s.BotName

	e.specialModeSessions.Delete(sessionKey)

	// 更新数据库
	deployment, err := e.deployRepo.FindByBotAndConversation(ctx, botID, conversationID)
	if err == nil {
		deployment.SpecialModeActive = false
		deployment.SpecialModeStartedAt = nil
		if updateErr := e.deployRepo.Update(ctx, deployment); updateErr != nil {
			logger.ErrorfWithCaller("[BotEngine] Failed to update deployment special mode status: %v", updateErr)
		}
	}

	// WebSocket 广播
	e.broadcastSpecialModeEvent(ctx, conversationID, "bot_special_mode_ended", botID.String(), botName)

	// 插入系统消息
	e.sendSystemMessage(ctx, conversationID, &models.SystemMessageContent{
		Type:    "special_mode_end",
		BotID:   botID.String(),
		BotName: botName,
	})

	logger.InfofWithCaller("[BotEngine] Special mode deactivated: bot=%s, conversation=%s", botName, conversationID)
	return nil
}

// IsSpecialModeActive 检查特殊模式是否活跃
func (e *BotEngine) IsSpecialModeActive(conversationID, botID uuid.UUID) bool {
	sessionKey := GetSessionKey(conversationID, botID)
	_, exists := e.specialModeSessions.Load(sessionKey)
	return exists
}

// HandleSpecialMode 处理特殊模式下的消息
func (e *BotEngine) HandleSpecialMode(ctx context.Context, msg *BotMessage, bot *models.Bot, deployment *models.BotDeployment, spec *SpecialModeSpec) {
	sessionKey := GetSessionKey(msg.ConversationID, bot.ID)

	sessionVal, exists := e.specialModeSessions.Load(sessionKey)
	if !exists {
		// 数据库标记活跃但内存中不存在（服务器重启），恢复会话
		if spec == nil {
			// 尝试从 mechanism_config 查找
			mechConfig, err := ParseMechanismConfig(bot.MechanismConfig)
			if err != nil {
				return
			}
			specialMech := FindSpecialModeMechanism(mechConfig.Mechanisms)
			if specialMech == nil || specialMech.Reply.SpecialMode == nil {
				return
			}
			spec = specialMech.Reply.SpecialMode
		}
		newSession := &SpecialModeSession{
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
		e.specialModeSessions.Store(sessionKey, newSession)
		sessionVal = newSession
	}

	session := sessionVal.(*SpecialModeSession)

	// 增加轮次
	session.Round++

	// 将用户消息加入上下文
	session.ContextBuffer = append(session.ContextBuffer, ContextMessage{
		Role:    "user",
		Content: msg.Content,
	})

	// 限制上下文大小
	maxContext := 100
	if len(session.ContextBuffer) > maxContext {
		session.ContextBuffer = session.ContextBuffer[len(session.ContextBuffer)-maxContext:]
	}

	// 检查结束条件
	if e.checkEndConditions(session, msg.Content) {
		logger.InfofWithCaller("[BotEngine] End condition met: bot=%s, conversation=%s, round=%d", bot.Name, msg.ConversationID, session.Round)
		_ = e.DeactivateSpecialMode(ctx, bot.ID, msg.ConversationID)
		return
	}

	// 执行事件链
	reply, err := e.executeEventChain(ctx, session, msg.Content)
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
	e.sendBotReply(ctx, bot, msg.ConversationID, reply)
}

// EventStep 事件链遍历中的单步信息
type EventStep struct {
	ID    string
	Input string
}

// buildEventTraversal 构建 BFS 事件遍历顺序
func buildEventTraversal(events []SpecialModeEvent, initialInput string) []EventStep {
	if len(events) == 0 {
		return nil
	}

	entryEvent := events[0]
	queue := []string{entryEvent.ID}
	visited := map[string]bool{}
	eventInputs := map[string]string{entryEvent.ID: initialInput}
	var steps []EventStep

	for len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]

		if visited[currentID] {
			continue
		}
		visited[currentID] = true

		var event *SpecialModeEvent
		for i := range events {
			if events[i].ID == currentID {
				event = &events[i]
				break
			}
		}
		if event == nil {
			continue
		}

		steps = append(steps, EventStep{ID: currentID, Input: eventInputs[currentID]})

		if len(event.Next) > 0 {
			for _, nextID := range event.Next {
				if !visited[nextID] {
					// 输入在执行后由调用方设置，这里预填入当前步骤的输入作为默认
					if _, exists := eventInputs[nextID]; !exists {
						eventInputs[nextID] = eventInputs[currentID]
					}
					queue = append(queue, nextID)
				}
			}
		}
	}

	return steps
}

// executeEventChain 执行事件链（BFS 遍历 DAG）
func (e *BotEngine) executeEventChain(ctx context.Context, session *SpecialModeSession, input string) (string, error) {
	steps := buildEventTraversal(session.Config.Events, input)
	if len(steps) == 0 {
		return "", fmt.Errorf("no events defined")
	}

	var finalOutput string
	lastOutput := input

	for _, step := range steps {
		// 更新步骤输入为上一个事件的输出
		if step.ID != steps[0].ID {
			step.Input = lastOutput
		}

		var event *SpecialModeEvent
		for i := range session.Config.Events {
			if session.Config.Events[i].ID == step.ID {
				event = &session.Config.Events[i]
				break
			}
		}
		if event == nil {
			continue
		}

		output, err := e.executeEvent(ctx, session, event, step.Input)
		if err != nil {
			logger.ErrorfWithCaller("[BotEngine] Event %s (%s) failed: %v", event.ID, event.Name, err)
			output = ""
		}

		session.EventOutputs[event.ID] = output
		lastOutput = output

		if event.Type == "reply" {
			finalOutput = output
		}
	}

	return finalOutput, nil
}

// executeEvent 执行单个事件
func (e *BotEngine) executeEvent(ctx context.Context, session *SpecialModeSession, event *SpecialModeEvent, input string) (string, error) {
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
func (e *BotEngine) executeLLMEvent(ctx context.Context, session *SpecialModeSession, event *SpecialModeEvent, input string) (string, error) {
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
func (e *BotEngine) executeReplyEvent(_ context.Context, session *SpecialModeSession, event *SpecialModeEvent, input string) (string, error) {
	template := getStringField(event.Config, "template")
	if template == "" {
		return input, nil
	}

	// 替换事件输出变量 $evt_ID.output
	result := template
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
func (e *BotEngine) executeBuiltinEvent(_ context.Context, session *SpecialModeSession, event *SpecialModeEvent, input string) (string, error) {
	return executeBuiltinHandler(event.Config, input, session.Variables)
}

// executePythonEvent 执行 Python 事件
func (e *BotEngine) executePythonEvent(ctx context.Context, _ *SpecialModeSession, event *SpecialModeEvent, input string) (string, error) {
	return sandbox.ExecutePythonEvent(ctx, event.Config, input)
}

// collectEventContext 根据 context_scope 收集事件所需的上下文
func (e *BotEngine) collectEventContext(_ context.Context, session *SpecialModeSession, event *SpecialModeEvent) []ContextMessage {
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

// checkEndConditions 检查特殊模式是否应该结束
func (e *BotEngine) checkEndConditions(session *SpecialModeSession, content string) bool {
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

// broadcastSpecialModeEvent 广播特殊模式状态变更事件
func (e *BotEngine) broadcastSpecialModeEvent(ctx context.Context, conversationID uuid.UUID, eventType, botID, botName string) {
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

// GetActiveSpecialModeSession 获取活跃的特殊模式会话（供调试面板使用）
func (e *BotEngine) GetActiveSpecialModeSession(conversationID, botID uuid.UUID) *SpecialModeSession {
	sessionKey := GetSessionKey(conversationID, botID)
	val, exists := e.specialModeSessions.Load(sessionKey)
	if !exists {
		return nil
	}
	return val.(*SpecialModeSession)
}

// RestoreSpecialModeFromDB 从数据库恢复活跃的特殊模式会话
func (e *BotEngine) RestoreSpecialModeFromDB(ctx context.Context) {
	deployments, err := e.deployRepo.FindActiveByConversation(ctx, uuid.Nil)
	if err != nil {
		// uuid.Nil 不会匹配到任何数据，我们需要另一种方式
		// 跳过恢复
		return
	}
	_ = deployments // 服务器重启后的恢复逻辑（MVP 阶段暂时跳过）
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
