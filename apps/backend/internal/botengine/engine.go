package botengine

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"
	"purr-chat-server/internal/websocket"
	"purr-chat-server/pkg/logger"

	"github.com/google/uuid"
)

// BotEngine Bot 处理引擎
//
// 当前职责（保留）：
//   - 消息路由入口：接收消息、查找 bot enrollment、调 TS/Go fallback
//   - TS 微服务客户端：通过 client.go 调用 TS bot-engine
//   - Bot 回复发送：sendBotReply、sendSystemMessage
//   - 调用日志记录：recordCallLog（持久化到数据库）
//
// Deprecated（待迁移）：
//   - Go fallback 路径：goFallbackProcess 中的触发评估和回复生成
//   - 工作流会话管理：workflowSessions sync.Map
//   - 调试会话管理：debugSessions sync.Map → 迁移至 TS /debug
type BotEngine struct {
	deployRepo     repository.BotDeploymentRepository
	botRepo        repository.BotRepository
	messageRepo    repository.ConversationMessageRepository
	enrollmentRepo repository.EnrollmentRepository
	callLogRepo    repository.BotCallLogRepository

	// 工作流会话：记录活跃的工作流运行时状态
	workflowSessions sync.Map // map[string]*SpecialModeSession — "conversationID:botID" -> session

	// 调试会话：记录调试运行时状态
	debugSessions sync.Map // map[string]*DebugSession — sessionID -> session

	// TS 微服务客户端（可选，用于调用 XState 版 Bot 引擎）
	tsClient *BotEngineClient
}

// NewBotEngine 创建 Bot 引擎
func NewBotEngine(
	deployRepo repository.BotDeploymentRepository,
	botRepo repository.BotRepository,
	messageRepo repository.ConversationMessageRepository,
	enrollmentRepo repository.EnrollmentRepository,
	tsServiceURL string,
) *BotEngine {
	e := &BotEngine{
		deployRepo:     deployRepo,
		botRepo:        botRepo,
		messageRepo:    messageRepo,
		enrollmentRepo: enrollmentRepo,
	}
	if tsServiceURL != "" {
		e.tsClient = NewBotEngineClient(tsServiceURL)
		logger.InfofWithCaller("[BotEngine] TS service client initialized: %s", tsServiceURL)
	}
	e.startDebugSessionCleanup()
	return e
}

// SetCallLogRepo 设置调用日志仓储（可选依赖）
func (e *BotEngine) SetCallLogRepo(repo repository.BotCallLogRepository) {
	e.callLogRepo = repo
}

// recordCallLog 记录调用日志（best-effort，失败不阻塞主流程）
func (e *BotEngine) recordCallLog(ctx context.Context, log *models.BotCallLog) {
	if e.callLogRepo == nil {
		return
	}
	log.ID = uuid.New()
	log.CreatedAt = time.Now().UTC()
	if err := e.callLogRepo.Create(ctx, log); err != nil {
		logger.ErrorfWithCaller("[BotEngine] Failed to record call log: %v", err)
	}
}

// truncateStr 截断字符串
func truncateStr(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}

func formatMessageCreatedAt(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

// sendSystemMessage 发送系统消息到会话（居中显示，无头像）
func (e *BotEngine) sendSystemMessage(ctx context.Context, conversationID uuid.UUID, content *models.SystemMessageContent) {
	contentJSON, err := json.Marshal(content)
	if err != nil {
		logger.ErrorfWithCaller("[BotEngine] Failed to marshal system message content: %v", err)
		return
	}

	message := &models.Message{
		SenderID:  uuid.Nil, // 系统用户
		Content:   string(contentJSON),
		MsgType:   models.MsgTypeSystem,
		CreatedAt: time.Now().UTC(),
	}

	err = e.messageRepo.InsertMessage(ctx, conversationID, message)
	if err != nil {
		logger.ErrorfWithCaller("[BotEngine] Failed to insert system message: %v", err)
		return
	}

	// 通过 WebSocket 通知所有会话成员
	if websocket.GlobalHub != nil {
		members, err := e.enrollmentRepo.FindByConversationID(ctx, conversationID)
		if err == nil {
			for _, m := range members {
				websocket.GlobalHub.SendToUser(m.UserID, "new_message", map[string]any{
					"id":              message.ID.String(),
					"conversation_id": conversationID.String(),
					"sender_id":       uuid.Nil.String(),
					"content":         string(contentJSON),
					"msg_type":        "system",
					"created_at":      formatMessageCreatedAt(message.CreatedAt),
				})
			}
		}
	}

	logger.InfofWithCaller("[BotEngine] System message sent to conversation %s: type=%s", conversationID, content.Type)
}

// BotMessage Bot 处理的入站消息
type BotMessage struct {
	ConversationID uuid.UUID
	SenderID       uuid.UUID
	SenderName     string
	Content        string
	MsgType        string
	CreatedAt      time.Time
}

// OnMessage 处理入站消息，异步评估并触发 Bot 回复
func (e *BotEngine) OnMessage(ctx context.Context, msg *BotMessage) {
	// 异步处理，不阻塞消息广播
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.ErrorfWithCaller("[BotEngine] Panic recovered: %v", r)
			}
		}()

		// 使用独立 context，不受 HTTP 请求生命周期影响
		e.processMessage(context.Background(), msg)
	}()
}

// processMessage 实际处理消息
// 当 TS 服务可用时，直接将 mechanism_config 传给 TS，由 TS 全权处理触发评估和执行。
// 仅在 TS 不可用时回退到 Go 引擎的本地触发评估。
func (e *BotEngine) processMessage(ctx context.Context, msg *BotMessage) {
	// 忽略系统消息
	if msg.SenderID == uuid.Nil {
		return
	}

	// 检查发送者是否是 Bot（Bot 消息不触发其他 Bot 响应）
	if e.isBotUser(ctx, msg.SenderID) {
		return
	}

	// 1. 通过 enrollment 查找会话中的 Bot 成员（权威来源）
	botEnrollments, err := e.enrollmentRepo.FindBotEnrollmentsByConversationID(ctx, msg.ConversationID)
	if err != nil {
		logger.ErrorfWithCaller("[BotEngine] Failed to find bot enrollments for conversation %s: %v", msg.ConversationID, err)
		return
	}

	if len(botEnrollments) == 0 {
		return
	}

	// 2. 对每个 Bot 处理
	for _, enrollment := range botEnrollments {
		bot, err := e.botRepo.FindByID(ctx, enrollment.UserID)
		if err != nil {
			logger.ErrorfWithCaller("[BotEngine] Failed to load bot %s: %v", enrollment.UserID, err)
			continue
		}

		// 检查 Bot 状态
		if bot.Status != models.BotStatusActive {
			continue
		}

		if e.tsClient != nil && e.tsClient.IsAvailable() {
			// TS 路径：收集上下文，直接调 TS（TS 负责触发评估 + 执行）
			contextMsgs := e.collectContextMessages(ctx, msg.ConversationID)
			start := time.Now()
			execResp, tsErr := e.tsClient.Execute(ctx, msg, bot.ID, bot.Name, bot.MechanismConfig, contextMsgs)
			duration := time.Since(start)

			if tsErr == nil {
				logger.InfofWithCaller("[BotEngine] TS bot=%s triggered=%v mechanism=%s reply=%q sessionActive=%v ms=%d",
					bot.Name, execResp.Triggered, execResp.MechanismName,
					truncateStr(execResp.Reply, 50), execResp.SessionActive, int(duration.Milliseconds()))

				// 记录调用日志
				e.recordCallLog(ctx, &models.BotCallLog{
					BotID:          bot.ID,
					ConversationID: msg.ConversationID,
					SenderID:       msg.SenderID,
					SenderName:     msg.SenderName,
					TriggerMessage: msg.Content,
					ReplyContent:   truncateStr(execResp.Reply, 500),
					MechanismID:    execResp.MechanismID,
					MechanismName:  execResp.MechanismName,
					ReplyType:      execResp.ReplyType,
					ExecutionPath:  "ts",
					Success:        true,
					DurationMs:     int(duration.Milliseconds()),
				})

				if execResp.Reply != "" {
					e.sendBotReply(ctx, bot, msg.ConversationID, execResp.Reply)
				}
			} else {
				logger.ErrorfWithCaller("[BotEngine] TS failed bot=%s error=%v", bot.Name, tsErr)
				e.recordCallLog(ctx, &models.BotCallLog{
					BotID:          bot.ID,
					ConversationID: msg.ConversationID,
					SenderID:       msg.SenderID,
					SenderName:     msg.SenderName,
					TriggerMessage: msg.Content,
					ExecutionPath:  "ts",
					Success:        false,
					ErrorMessage:   tsErr.Error(),
					DurationMs:     int(duration.Milliseconds()),
				})
			}
			continue
		}

		// Go fallback（TS 不可用时）
		logger.InfofWithCaller("[BotEngine] TS unavailable, using Go fallback for bot %s", bot.Name)
		e.goFallbackProcess(ctx, msg, bot)
	}
}

// goFallbackProcess Go 引擎 fallback 路径：本地评估触发条件并执行
// Deprecated: 仅在 TS 微服务不可用时使用，后续将完全迁移至 TS。
func (e *BotEngine) goFallbackProcess(ctx context.Context, msg *BotMessage, bot *models.Bot) {
	// 解析机制配置
	mechConfig, err := ParseMechanismConfig(bot.MechanismConfig)
	if err != nil {
		logger.ErrorfWithCaller("[BotEngine] Failed to parse mechanism config for bot %s: %v", bot.ID, err)
		return
	}

	// 遍历机制列表（从上到下，首个匹配即响应）
	for i := range mechConfig.Mechanisms {
		mech := &mechConfig.Mechanisms[i]
		if !mech.Enabled {
			continue
		}

		// 评估触发条件
		matched := mech.Trigger.Evaluate(msg.Content)
		if !matched {
			continue
		}

		logger.InfofWithCaller("[BotEngine] Bot %s: mechanism[%d] trigger matched (Go fallback)", bot.Name, i)

		// 触发匹配成功（Go 引擎路径）
		switch mech.Reply.Type {
		case "workflow":
			// Deprecated: workflow handled by TS microservice
			e.sendBotReply(ctx, bot, msg.ConversationID, "...")

		case "predefined", "llm":
			// 编译为简单工作流执行（统一执行路径）
			compiled := CompileSimpleMechanism(mech)
			if compiled != nil {
				contextMsgs := e.collectContextForMechanism(ctx, msg.ConversationID, mech)
				reply, err := e.ExecuteSimpleFlow(ctx, compiled, msg, bot, contextMsgs)
				if err != nil {
					logger.ErrorfWithCaller("[BotEngine] Simple flow execution failed for bot %s: %v", bot.ID, err)
					reply = "..."
				}
				e.sendBotReply(ctx, bot, msg.ConversationID, reply)
			} else {
				// 编译失败，回退到原始路径
				contextMessages := e.collectContextForMechanism(ctx, msg.ConversationID, mech)
				contextVars := map[string]string{"time": time.Now().Format("15:04")}
				reply, err := mech.Reply.GenerateReply(msg.Content, contextVars, contextMessages, msg.SenderName)
				if err != nil {
					reply = "..."
				}
				e.sendBotReply(ctx, bot, msg.ConversationID, reply)
			}

		default:
			// 未知类型，使用原始回复路径
			contextMessages := e.collectContextForMechanism(ctx, msg.ConversationID, mech)
			contextVars := map[string]string{"time": time.Now().Format("15:04")}
			reply, err := mech.Reply.GenerateReply(msg.Content, contextVars, contextMessages, msg.SenderName)
			if err != nil {
				reply = "..."
			}
			e.sendBotReply(ctx, bot, msg.ConversationID, reply)
		}
		break // 首个匹配机制响应后，跳过后续机制
	}
}

// isBotUser 检查用户是否是 Bot（通过 bots 表判断）
func (e *BotEngine) isBotUser(ctx context.Context, userID uuid.UUID) bool {
	_, err := e.botRepo.FindByID(ctx, userID)
	return err == nil
}

// collectContextForMechanism 收集机制所需的上下文消息
func (e *BotEngine) collectContextForMechanism(ctx context.Context, conversationID uuid.UUID, mech *Mechanism) []ContextMessage {
	windowSize := 20
	if mech.Reply.Type == "llm" && mech.Reply.LLM != nil && mech.Reply.LLM.ContextWindow > 0 {
		windowSize = mech.Reply.LLM.ContextWindow
	}

	// 获取最近的消息
	messages, err := e.messageRepo.FindMessages(ctx, conversationID, windowSize, 0)
	if err != nil {
		return nil
	}

	var contextMessages []ContextMessage
	for _, msg := range messages {
		// 只包含文本消息
		if msg.MsgType == models.MsgTypeText {
			contextMessages = append(contextMessages, ContextMessage{
				Role:    "user",
				Content: msg.Content,
			})
		}
	}

	// 按 CreatedAt 正序排列（FindMessages 是 DESC）
	for i, j := 0, len(contextMessages)-1; i < j; i, j = i+1, j-1 {
		contextMessages[i], contextMessages[j] = contextMessages[j], contextMessages[i]
	}

	return contextMessages
}

// collectContextMessages 收集会话的上下文消息（TS 路径使用，不需要 mechanism 参数）
func (e *BotEngine) collectContextMessages(ctx context.Context, conversationID uuid.UUID) []ContextMessage {
	messages, err := e.messageRepo.FindMessages(ctx, conversationID, 20, 0)
	if err != nil {
		return nil
	}
	var contextMessages []ContextMessage
	for _, msg := range messages {
		if msg.MsgType == models.MsgTypeText {
			contextMessages = append(contextMessages, ContextMessage{
				Role:    "user",
				Content: msg.Content,
			})
		}
	}
	// FindMessages 是 DESC，需要反转为正序
	for i, j := 0, len(contextMessages)-1; i < j; i, j = i+1, j-1 {
		contextMessages[i], contextMessages[j] = contextMessages[j], contextMessages[i]
	}
	return contextMessages
}

// sendBotReply 发送 Bot 回复到会话
func (e *BotEngine) sendBotReply(ctx context.Context, bot *models.Bot, conversationID uuid.UUID, content string) {
	// Bot 现在是真实用户，sender_id 使用 bot.ID（等于 users 表中的 id）
	botID := bot.ID
	botName := bot.Name
	message := &models.Message{
		ID:             uuid.New(),
		ConversationID: conversationID,
		SenderID:       bot.ID, // Bot 的 user_id
		Content:        content,
		MsgType:        models.MsgTypeText,
		CreatedAt:      time.Now().UTC(),
		BotID:          &botID,
		BotName:        &botName,
	}

	// 插入消息
	err := e.messageRepo.InsertMessage(ctx, conversationID, message)
	if err != nil {
		logger.ErrorfWithCaller("[BotEngine] Failed to insert bot message: %v", err)
		return
	}

	// 通过 WebSocket 通知所有会话成员
	if websocket.GlobalHub != nil {
		members, err := e.enrollmentRepo.FindByConversationID(ctx, conversationID)
		if err == nil {
			for _, m := range members {
				websocket.GlobalHub.SendToUser(m.UserID, "new_message", map[string]any{
					"id":              message.ID.String(),
					"conversation_id": conversationID.String(),
					"sender_id":       message.SenderID.String(),
					"content":         content,
					"msg_type":        "text",
					"created_at":      formatMessageCreatedAt(message.CreatedAt),
					"sender": map[string]any{
						"id":         bot.ID.String(),
						"username":   bot.Name,
						"avatar_url": bot.AvatarURL,
						"is_bot":     true,
					},
					"bot_id":   bot.ID.String(),
					"bot_name": bot.Name,
				})
			}
		}

		previewContent := content
		if len(previewContent) > 50 {
			previewContent = previewContent[:50] + "..."
		}
		logger.InfofWithCaller("[BotEngine] Bot %s replied to conversation %s: %s", bot.Name, conversationID, previewContent)
	}
}
