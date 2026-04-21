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
type BotEngine struct {
	deployRepo     repository.BotDeploymentRepository
	botRepo        repository.BotRepository
	messageRepo    repository.ConversationMessageRepository
	enrollmentRepo repository.EnrollmentRepository

	// 特殊模式会话：记录活跃的特殊模式运行时状态
	specialModeSessions sync.Map // map[string]*SpecialModeSession — "conversationID:botID" -> session

	// 调试会话：记录调试运行时状态
	debugSessions sync.Map // map[string]*DebugSession — sessionID -> session
}

// NewBotEngine 创建 Bot 引擎
func NewBotEngine(
	deployRepo repository.BotDeploymentRepository,
	botRepo repository.BotRepository,
	messageRepo repository.ConversationMessageRepository,
	enrollmentRepo repository.EnrollmentRepository,
) *BotEngine {
	e := &BotEngine{
		deployRepo:     deployRepo,
		botRepo:        botRepo,
		messageRepo:    messageRepo,
		enrollmentRepo: enrollmentRepo,
	}
	e.startDebugSessionCleanup()
	return e
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
					"created_at":      message.CreatedAt.Format(time.RFC3339),
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
func (e *BotEngine) processMessage(ctx context.Context, msg *BotMessage) {
	// 忽略 Bot 自身发送的消息，避免无限循环
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

	logger.InfofWithCaller("[BotEngine] Found %d bot(s) in conversation %s, processing...", len(botEnrollments), msg.ConversationID)

	// 2. 对每个 Bot 评估机制列表
	for _, enrollment := range botEnrollments {
		bot, err := e.botRepo.FindByID(ctx, enrollment.UserID)
		if err != nil {
			logger.ErrorfWithCaller("[BotEngine] Failed to load bot %s: %v", enrollment.UserID, err)
			continue
		}

		// 检查 Bot 状态
		if bot.Status != models.BotStatusActive {
			logger.InfofWithCaller("[BotEngine] Bot %s is %s, skipping", bot.Name, bot.Status)
			continue
		}

		logger.InfofWithCaller("[BotEngine] Evaluating bot %s for message: %q", bot.Name, msg.Content)

		// 解析机制配置
		mechConfig, err := ParseMechanismConfig(bot.MechanismConfig)
		if err != nil {
			logger.ErrorfWithCaller("[BotEngine] Failed to parse mechanism config for bot %s: %v", bot.ID, err)
			continue
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

			logger.InfofWithCaller("[BotEngine] Bot %s: mechanism[%d] trigger matched", bot.Name, i)

			// 触发匹配成功
			if mech.Reply.Type == "special_mode" {
				e.activateMechanismSpecialMode(ctx, msg, bot, mech.Reply.SpecialMode)
				break
			}

			// 收集上下文消息
			contextMessages := e.collectContextForMechanism(ctx, msg.ConversationID, mech)

			// 生成回复
			contextVars := map[string]string{
				"time": time.Now().Format("15:04"),
			}

			reply, err := mech.Reply.GenerateReply(msg.Content, contextVars, contextMessages, msg.SenderName)
			if err != nil {
				logger.ErrorfWithCaller("[BotEngine] Failed to generate reply for bot %s: %v", bot.ID, err)
				reply = "..."
			}

			// 发送 Bot 回复
			e.sendBotReply(ctx, bot, msg.ConversationID, reply)
			break // 首个匹配机制响应后，跳过后续机制
		}
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
					"created_at":      message.CreatedAt.Format(time.RFC3339),
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
