package botengine

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"purr-chat-server/internal/models"
	"purr-chat-server/pkg/logger"

	"github.com/google/uuid"
)

// DebugSession 调试运行时会话（内存中，不持久化）
type DebugSession struct {
	ID            string
	BotID         uuid.UUID
	BotName       string
	Config        *SpecialModeSpec
	Round         int
	ContextBuffer []ContextMessage
	EventOutputs  map[string]string
	Variables     map[string]string
	CreatedAt     time.Time
	// 逐步执行状态
	StepMode    bool
	StepQueue   []string // 待执行事件 ID 队列（按 flow 拓扑顺序）
	StepVisited map[string]bool
}

// toSpecialModeSession 将 DebugSession 转换为 SpecialModeSession 以复用执行逻辑
func (ds *DebugSession) toSpecialModeSession() *SpecialModeSession {
	return &SpecialModeSession{
		BotID:         ds.BotID,
		BotName:       ds.BotName,
		Config:        ds.Config,
		ContextBuffer: ds.ContextBuffer,
		EventOutputs:  ds.EventOutputs,
		Variables:     ds.Variables,
	}
}

// toFlowContext 将 DebugSession 转换为 ExecutionContext
func (ds *DebugSession) toFlowContext() *ExecutionContext {
	return NewExecutionContext(ds.Config.Events, ds.Config.Connections, ds.toSpecialModeSession())
}

// DebugExecute 执行调试（全量或逐步首事件）
func (e *BotEngine) DebugExecute(ctx context.Context, botID uuid.UUID, req *models.DebugBotRequest) (*models.DebugTraceResult, error) {
	var specialSpec *SpecialModeSpec

	// 确定使用哪个配置：优先使用传入的 special_mode_config（向后兼容）
	if len(req.SpecialModeConfig) > 0 {
		if err := json.Unmarshal(req.SpecialModeConfig, &specialSpec); err != nil {
			return nil, fmt.Errorf("invalid special_mode_config override: %w", err)
		}
	}

	// 如果没有传入配置，从 bot 的 mechanism_config 中查找 special_mode 机制
	if specialSpec == nil || len(specialSpec.Events) == 0 {
		bot, err := e.botRepo.FindByID(ctx, botID)
		if err == nil {
			mechConfig, parseErr := ParseMechanismConfig(bot.MechanismConfig)
			if parseErr == nil {
				specialMech := FindSpecialModeMechanism(mechConfig.Mechanisms)
				if specialMech != nil && specialMech.Reply.SpecialMode != nil {
					specialSpec = specialMech.Reply.SpecialMode
				}
			}
		}
	}

	if specialSpec == nil || len(specialSpec.Events) == 0 {
		return nil, fmt.Errorf("no special mode events defined")
	}

	// 获取 Bot 信息（用于 BotName）
	botName := "Bot"
	bot, err := e.botRepo.FindByID(ctx, botID)
	if err == nil {
		botName = bot.Name
	}

	// 获取或创建调试会话
	var session *DebugSession
	if req.SessionID != "" {
		if val, ok := e.debugSessions.Load(req.SessionID); ok {
			session = val.(*DebugSession)
			// 如果提供了新配置覆盖，更新会话配置
			if len(req.SpecialModeConfig) > 0 {
				session.Config = specialSpec
				session.StepQueue = nil // 清空步骤队列
			}
		}
	}

	if session == nil {
		sessionID := uuid.New().String()
		session = &DebugSession{
			ID:            sessionID,
			BotID:         botID,
			BotName:       botName,
			Config:        specialSpec,
			Round:         0,
			ContextBuffer: []ContextMessage{},
			EventOutputs:  map[string]string{},
			Variables:     map[string]string{},
			CreatedAt:     time.Now(),
			StepMode:      req.StepMode,
			StepVisited:   map[string]bool{},
		}
		e.debugSessions.Store(sessionID, session)
	}

	// 增加轮次
	session.Round++

	// 将用户消息加入上下文
	session.ContextBuffer = append(session.ContextBuffer, ContextMessage{
		Role:    "user",
		Content: req.Message,
	})

	// 限制上下文大小
	maxContext := 100
	if len(session.ContextBuffer) > maxContext {
		session.ContextBuffer = session.ContextBuffer[len(session.ContextBuffer)-maxContext:]
	}

	// 构建节点执行顺序（基于 flow 拓扑）
	flowCtx := session.toFlowContext()
	nodeOrder := flowCtx.collectNodeOrder()
	if len(nodeOrder) == 0 {
		return nil, fmt.Errorf("no events to execute")
	}

	smSession := session.toSpecialModeSession()

	if req.StepMode {
		// 逐步模式：仅执行第一个未访问的事件
		return e.debugStepFirst(ctx, session, smSession, nodeOrder)
	}

	// 全量执行
	return e.debugRunAll(ctx, session, smSession, nodeOrder)
}

// debugRunAll 全量执行所有事件（按 flow 拓扑顺序）
func (e *BotEngine) debugRunAll(ctx context.Context, session *DebugSession, smSession *SpecialModeSession, nodeOrder []string) (*models.DebugTraceResult, error) {
	var traces []models.EventTrace
	var finalOutput string

	// 将最近用户消息写入 trigger 节点的输出端口
	if len(session.ContextBuffer) > 0 {
		userMsg := session.ContextBuffer[len(session.ContextBuffer)-1].Content
		smSession.EventOutputs["__user_input__"] = userMsg
	}

	for _, nodeID := range nodeOrder {
		event := findEventInList(session.Config.Events, nodeID)
		if event == nil {
			continue
		}

		// trigger 和 end 节点跳过（不产生输出）
		if event.Type == "trigger" || event.Type == "end" {
			continue
		}

		// 对于 if/loop/wait/history 节点，flow engine 已处理逻辑，
		// 在全量调试模式下直接走完整的 flow engine
		// 但为调试目的，我们对处理节点逐个执行并记录 trace
		input := ""
		if event.Type == "llm" || event.Type == "builtin" || event.Type == "python" || event.Type == "reply" {
			// 从 session.EventOutputs 获取前一个节点的输出作为输入
			input = smSession.EventOutputs["__last_output__"]
		}

		if event.Type == "llm" || event.Type == "builtin" || event.Type == "python" || event.Type == "reply" {
			trace := e.executeEventWithTrace(ctx, smSession, event, input)
			traces = append(traces, trace)

			session.EventOutputs[event.ID] = trace.Output
			smSession.EventOutputs[event.ID] = trace.Output
			smSession.EventOutputs["__last_output__"] = trace.Output

			if event.Type == "reply" {
				finalOutput = trace.Output
			}
		}
	}

	// 将 Bot 回复加入上下文
	if finalOutput != "" {
		session.ContextBuffer = append(session.ContextBuffer, ContextMessage{
			Role:    "assistant",
			Content: finalOutput,
		})
	}

	return &models.DebugTraceResult{
		SessionID:       session.ID,
		Reply:           finalOutput,
		ContextMessages: toDebugContextMessages(session.ContextBuffer),
		EventTraces:     traces,
		WaitingForStep:  false,
		Round:           session.Round,
	}, nil
}

// debugStepFirst 逐步模式：执行第一个事件
func (e *BotEngine) debugStepFirst(ctx context.Context, session *DebugSession, smSession *SpecialModeSession, nodeOrder []string) (*models.DebugTraceResult, error) {
	// 构建步骤队列（跳过 trigger/end 节点）
	session.StepQueue = nil
	for _, nodeID := range nodeOrder {
		event := findEventInList(session.Config.Events, nodeID)
		if event == nil {
			continue
		}
		if event.Type == "trigger" || event.Type == "end" {
			continue
		}
		if !session.StepVisited[nodeID] {
			session.StepQueue = append(session.StepQueue, nodeID)
		}
	}

	if len(session.StepQueue) == 0 {
		return nil, fmt.Errorf("no more events to execute")
	}

	// 将最近用户消息写入
	if len(session.ContextBuffer) > 0 {
		userMsg := session.ContextBuffer[len(session.ContextBuffer)-1].Content
		smSession.EventOutputs["__user_input__"] = userMsg
		smSession.EventOutputs["__last_output__"] = userMsg
	}

	return e.debugExecuteNext(ctx, session, smSession)
}

// debugExecuteNext 执行逐步模式中的下一个事件
func (e *BotEngine) debugExecuteNext(ctx context.Context, session *DebugSession, smSession *SpecialModeSession) (*models.DebugTraceResult, error) {
	if len(session.StepQueue) == 0 {
		return nil, fmt.Errorf("no more events to execute")
	}

	// 取出队列头部
	nextID := session.StepQueue[0]
	session.StepQueue = session.StepQueue[1:]
	session.StepVisited[nextID] = true

	event := findEventInList(session.Config.Events, nextID)
	if event == nil {
		return nil, fmt.Errorf("event %s not found", nextID)
	}

	input := smSession.EventOutputs["__last_output__"]
	trace := e.executeEventWithTrace(ctx, smSession, event, input)

	// 更新会话状态
	session.EventOutputs[event.ID] = trace.Output
	smSession.EventOutputs[event.ID] = trace.Output
	smSession.EventOutputs["__last_output__"] = trace.Output

	// 构建已执行的 traces（包括之前已执行的）
	allTraces := session.buildAllTraces()

	// 如果没有更多事件且最后一个不是 reply，收集最终输出
	reply := ""
	if len(session.StepQueue) == 0 {
		// 找到所有 reply 类型事件的输出
		for _, evt := range session.Config.Events {
			if evt.Type == "reply" && session.StepVisited[evt.ID] {
				if output, ok := session.EventOutputs[evt.ID]; ok {
					reply = output
				}
			}
		}
		if reply != "" {
			session.ContextBuffer = append(session.ContextBuffer, ContextMessage{
				Role:    "assistant",
				Content: reply,
			})
		}
	}

	return &models.DebugTraceResult{
		SessionID:       session.ID,
		Reply:           reply,
		ContextMessages: toDebugContextMessages(session.ContextBuffer),
		EventTraces:     allTraces,
		WaitingForStep:  len(session.StepQueue) > 0,
		NextEventID:     "",
		Round:           session.Round,
	}, nil
}

// DebugStep 执行逐步模式下的下一个事件
func (e *BotEngine) DebugStep(ctx context.Context, botID uuid.UUID, sessionID string) (*models.DebugTraceResult, error) {
	val, ok := e.debugSessions.Load(sessionID)
	if !ok {
		return nil, fmt.Errorf("debug session not found: %s", sessionID)
	}

	session := val.(*DebugSession)
	if len(session.StepQueue) == 0 {
		return nil, fmt.Errorf("no more events to execute")
	}

	smSession := session.toSpecialModeSession()
	return e.debugExecuteNext(ctx, session, smSession)
}

// DebugReset 清除调试会话
func (e *BotEngine) DebugReset(sessionID string) {
	e.debugSessions.Delete(sessionID)
}

// executeEventWithTrace 带轨迹记录的事件执行
func (e *BotEngine) executeEventWithTrace(ctx context.Context, session *SpecialModeSession, event *SpecialModeEvent, input string) models.EventTrace {
	start := time.Now()

	output, err := e.executeEvent(ctx, session, event, input)
	duration := time.Since(start).Milliseconds()

	trace := models.EventTrace{
		EventID:    event.ID,
		EventType:  event.Type,
		EventName:  event.Name,
		Input:      input,
		Output:     output,
		DurationMs: duration,
	}

	if err != nil {
		trace.Status = "error"
		trace.Error = err.Error()
		trace.Output = ""
		logger.ErrorfWithCaller("[BotEngine][Debug] Event %s (%s) failed: %v", event.ID, event.Name, err)
	} else {
		trace.Status = "success"
	}

	// 对于 LLM 事件，记录它看到的上下文
	if event.Type == "llm" {
		contextMsgs := e.collectEventContext(ctx, session, event)
		trace.ContextMessages = toDebugContextMessages(contextMsgs)
	}

	return trace
}

// buildAllTraces 构建包含所有已执行事件的完整轨迹列表
func (ds *DebugSession) buildAllTraces() []models.EventTrace {
	var traces []models.EventTrace

	for _, evt := range ds.Config.Events {
		if !ds.StepVisited[evt.ID] {
			// 未执行的事件显示为 pending
			traces = append(traces, models.EventTrace{
				EventID:   evt.ID,
				EventType: evt.Type,
				EventName: evt.Name,
				Status:    "pending",
			})
		}
	}

	// 按遍历顺序返回已执行的（从 StepVisited 的插入顺序无法保证，所以依赖 events 定义顺序）
	return traces
}

// startDebugSessionCleanup 启动调试会话自动清理
func (e *BotEngine) startDebugSessionCleanup() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			e.debugSessions.Range(func(key, value any) bool {
				session := value.(*DebugSession)
				if time.Since(session.CreatedAt) > 30*time.Minute {
					e.debugSessions.Delete(key)
					logger.InfofWithCaller("[BotEngine][Debug] Cleaned up expired session: %s", key)
				}
				return true
			})
		}
	}()
}

// findEventInList 在事件列表中查找指定 ID 的事件
func findEventInList(events []SpecialModeEvent, id string) *SpecialModeEvent {
	for i := range events {
		if events[i].ID == id {
			return &events[i]
		}
	}
	return nil
}

// toDebugContextMessages 将 ContextMessage 转换为 DebugContextMessage
func toDebugContextMessages(msgs []ContextMessage) []models.DebugContextMessage {
	result := make([]models.DebugContextMessage, len(msgs))
	for i, msg := range msgs {
		result[i] = models.DebugContextMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return result
}
