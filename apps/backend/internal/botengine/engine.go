package botengine

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"purr-chat-server/internal/messaging"
	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"
	"purr-chat-server/pkg/logger"

	"github.com/google/uuid"
)

// SecretResolver 运行时解密 secret 的接口(解耦 botengine 与 services 包)
type SecretResolver interface {
	ResolveSecrets(ctx context.Context, appID uuid.UUID) (map[string]string, error)
}

// BotEngine Bot 处理引擎
//
// 当前职责：
//   - 消息事件订阅者：接收 MessageCreatedEvent，查找 bot enrollment/installation，调 TS
//   - TS 微服务客户端：通过 client.go 调用 TS bot-engine
//   - Bot 回复发送：通过 messaging.BotMessageSender 统一发送管线
//   - Installation / Capability / Secret 运行时校验
//   - 调用日志记录：recordCallLog（持久化 Trace 与消息关联 ID）
type BotEngine struct {
	deployRepo       repository.BotDeploymentRepository
	botRepo          repository.BotRepository
	messageRepo      repository.ConversationMessageRepository
	enrollmentRepo   repository.EnrollmentRepository
	callLogRepo      repository.BotCallLogRepository
	installationRepo repository.BotInstallationRepository
	workflowRepo     repository.WorkflowRepository
	secretResolver   SecretResolver
	messageSender    messaging.BotMessageSender

	// 工作流会话：记录活跃的工作流部署状态
	workflowSessions sync.Map // map[string]*WorkflowSession — "conversationID:botID" -> session

	// TS 微服务客户端（用于调用 TS bot-engine）
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
	return e
}

func (e *BotEngine) SetCallLogRepo(repo repository.BotCallLogRepository) {
	e.callLogRepo = repo
}

func (e *BotEngine) GetTSClient() *BotEngineClient {
	return e.tsClient
}

func (e *BotEngine) SetInstallationRepo(repo repository.BotInstallationRepository) {
	e.installationRepo = repo
}

func (e *BotEngine) SetSecretResolver(resolver SecretResolver) {
	e.secretResolver = resolver
}

func (e *BotEngine) SetWorkflowRepo(repo repository.WorkflowRepository) {
	e.workflowRepo = repo
}

// SetMessageSender 设置统一消息发送器（由 MessageService 实现）
func (e *BotEngine) SetMessageSender(sender messaging.BotMessageSender) {
	e.messageSender = sender
}

// OnMessageCreated 实现 messaging.MessageEventSink。
// 异步边界由 MessageService 的 publisher 调度负责，这里同步执行以便超时和指标覆盖真实处理。
func (e *BotEngine) OnMessageCreated(ctx context.Context, event *messaging.MessageCreatedEvent) error {
	// Bot 发送的消息和系统消息不触发其他 Bot（防回复环）
	if !event.ShouldTriggerBots() {
		return nil
	}

	e.processMessage(ctx, event)
	return nil
}

// recordCallLog 记录调用日志（best-effort）
func (e *BotEngine) recordCallLog(ctx context.Context, log *models.BotCallLog) {
	if e.callLogRepo == nil {
		return
	}
	log.ID = uuid.New()
	log.CreatedAt = time.Now().UTC()

	if e.installationRepo != nil {
		consent := e.resolveDiagnosticsConsent(ctx, log.BotID, log.ConversationID, log.SenderID)
		if consent != models.DiagnosticsGranted {
			log.TriggerMessage = ""
			log.Trace = nil
		}
	}

	if err := e.callLogRepo.Create(ctx, log); err != nil {
		logger.ErrorfWithCaller("[BotEngine] Failed to record call log: %v", err)
	}
}

func (e *BotEngine) resolveInstallation(ctx context.Context, botID, conversationID, senderID uuid.UUID) *models.BotInstallation {
	if e.installationRepo == nil {
		return nil
	}
	if inst, _ := e.installationRepo.FindByAppAndTarget(ctx, botID, models.InstallationTargetConversation, conversationID); inst != nil {
		return inst
	}
	if inst, _ := e.installationRepo.FindByAppAndTarget(ctx, botID, models.InstallationTargetUser, senderID); inst != nil {
		return inst
	}
	return nil
}

func (e *BotEngine) resolveDiagnosticsConsent(ctx context.Context, botID, conversationID, senderID uuid.UUID) models.DiagnosticsConsent {
	if inst := e.resolveInstallation(ctx, botID, conversationID, senderID); inst != nil {
		return inst.DiagnosticsConsent
	}
	return models.DiagnosticsDenied
}

func truncateStr(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}

func formatMessageCreatedAt(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

// processMessage 处理消息事件，触发匹配的 Bot
func (e *BotEngine) processMessage(ctx context.Context, event *messaging.MessageCreatedEvent) {
	msg := event.Message
	if msg == nil {
		return
	}

	var triggerMsgID *uuid.UUID
	if msg.ID != uuid.Nil {
		id := msg.ID
		triggerMsgID = &id
	}

	// 1. 通过 enrollment 查找会话中的 Bot 成员
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

		if bot.Status != models.BotStatusActive {
			continue
		}

		inst := e.resolveInstallation(ctx, bot.ID, msg.ConversationID, msg.SenderID)
		if inst == nil {
			logger.InfofWithCaller("[BotEngine] Skip bot=%s: no installation found for conversation=%s sender=%s",
				bot.Name, msg.ConversationID, msg.SenderID)
			continue
		}
		if inst.Status != models.InstallationActive {
			logger.InfofWithCaller("[BotEngine] Skip bot=%s: installation status=%s (not active)", bot.Name, inst.Status)
			continue
		}
		if !models.HasCapability(inst.GrantedCapabilities, models.CapabilityReadTrigger) {
			logger.InfofWithCaller("[BotEngine] Skip bot=%s: messages:read_trigger capability not granted", bot.Name)
			continue
		}

		if bot.PublishedVersion == nil || *bot.PublishedVersion == 0 {
			logger.InfofWithCaller("[BotEngine] Skip bot=%s: no published workflow version", bot.Name)
			e.recordCallLog(ctx, &models.BotCallLog{
				BotID:            bot.ID,
				ConversationID:   msg.ConversationID,
				SenderID:         msg.SenderID,
				SenderName:       event.SenderName,
				TriggerMessage:   msg.Content,
				ExecutionPath:    "ts",
				Success:          false,
				ErrorMessage:     "no published workflow version",
				RunStatus:        models.RunStatusError,
				ErrorType:        "no_published_version",
				TriggerMessageID: triggerMsgID,
			})
			continue
		}

		if e.workflowRepo == nil {
			logger.ErrorfWithCaller("[BotEngine] workflowRepo not injected; cannot load published document for bot=%s", bot.Name)
			continue
		}

		version, err := e.workflowRepo.FindPublishedByRevision(ctx, bot.ID, *bot.PublishedVersion)
		if err != nil {
			logger.ErrorfWithCaller("[BotEngine] Failed to load published workflow bot=%s revision=%d: %v", bot.Name, *bot.PublishedVersion, err)
			e.recordCallLog(ctx, &models.BotCallLog{
				BotID:            bot.ID,
				ConversationID:   msg.ConversationID,
				SenderID:         msg.SenderID,
				SenderName:       event.SenderName,
				TriggerMessage:   msg.Content,
				ExecutionPath:    "ts",
				Success:          false,
				ErrorMessage:     fmt.Sprintf("failed to load published revision %d: %v", *bot.PublishedVersion, err),
				RunStatus:        models.RunStatusError,
				ErrorType:        "version_load_failed",
				TriggerMessageID: triggerMsgID,
			})
			continue
		}

		if e.tsClient == nil || !e.tsClient.IsAvailable() {
			logger.ErrorfWithCaller("[BotEngine] TS service unavailable; bot=%s will not execute (no Go fallback)", bot.Name)
			e.recordCallLog(ctx, &models.BotCallLog{
				BotID:            bot.ID,
				ConversationID:   msg.ConversationID,
				SenderID:         msg.SenderID,
				SenderName:       event.SenderName,
				TriggerMessage:   msg.Content,
				ExecutionPath:    "ts",
				Success:          false,
				ErrorMessage:     "bot-engine service unavailable",
				RunStatus:        models.RunStatusError,
				ErrorType:        "ts_unavailable",
				TriggerMessageID: triggerMsgID,
			})
			continue
		}

		// 构建 BotMessage 供 TS 客户端使用
		botMsg := &BotMessage{
			ConversationID: msg.ConversationID,
			SenderID:       msg.SenderID,
			SenderName:     event.SenderName,
			Content:        msg.Content,
			MsgType:        string(msg.MsgType),
			CreatedAt:      msg.CreatedAt,
			MessageID:      msg.ID,
		}

		contextMsgs := e.collectContextMessages(ctx, msg.ConversationID)
		grantedCaps := inst.GrantedCapabilities
		var secrets map[string]string
		if e.secretResolver != nil && models.HasCapability(grantedCaps, models.CapabilitySecretsUse) {
			if dec, err := e.secretResolver.ResolveSecrets(ctx, bot.ID); err == nil {
				secrets = dec
			} else {
				logger.ErrorfWithCaller("[BotEngine] Failed to resolve secrets for bot %s: %v", bot.Name, err)
			}
		}

		start := time.Now()
		execResp, tsErr := e.tsClient.Execute(ctx, botMsg, bot.ID, bot.Name, version.Document, version.Revision, contextMsgs, grantedCaps, secrets)
		duration := time.Since(start)

		if tsErr == nil {
			logger.InfofWithCaller("[BotEngine] TS bot=%s runId=%s triggered=%v revision=%d reply=%q sessionActive=%v ms=%d",
				bot.Name, execResp.RunID, execResp.Triggered, version.Revision,
				truncateStr(execResp.Reply, 50), execResp.SessionActive, int(duration.Milliseconds()))

			runStatus := models.RunStatusCompleted
			if execResp.Status == "error" {
				runStatus = models.RunStatusError
			}

			// 通过统一发送管线持久化并广播 Bot 回复
			var replyMsgID *uuid.UUID
			if execResp.Reply != "" && e.messageSender != nil {
				replyMsg, err := e.messageSender.SendBotMessage(ctx, &messaging.BotSendRequest{
					BotID:            bot.ID,
					ConversationID:   msg.ConversationID,
					Content:          execResp.Reply,
					MsgType:          string(models.MsgTypeText),
					Source:           messaging.SourceWorkflow,
					RunID:            execResp.RunID,
					TriggerMessageID: triggerMsgID,
				})
				if err != nil {
					logger.ErrorfWithCaller("[BotEngine] Failed to send bot reply via message sender: %v", err)
				} else if replyMsg != nil {
					id := replyMsg.ID
					replyMsgID = &id
				}
			}

			e.recordCallLog(ctx, &models.BotCallLog{
				BotID:            bot.ID,
				ConversationID:   msg.ConversationID,
				SenderID:         msg.SenderID,
				SenderName:       event.SenderName,
				TriggerMessage:   msg.Content,
				ReplyContent:     truncateStr(execResp.Reply, 500),
				ExecutionPath:    "ts",
				Success:          execResp.Status != "error",
				DurationMs:       int(duration.Milliseconds()),
				RunID:            execResp.RunID,
				TriggerMessageID: triggerMsgID,
				ReplyMessageID:   replyMsgID,
				WorkflowRevision: &version.Revision,
				RunStatus:        runStatus,
				Trace:            execResp.Trace,
			})
		} else {
			logger.ErrorfWithCaller("[BotEngine] TS failed bot=%s error=%v", bot.Name, tsErr)
			e.recordCallLog(ctx, &models.BotCallLog{
				BotID:            bot.ID,
				ConversationID:   msg.ConversationID,
				SenderID:         msg.SenderID,
				SenderName:       event.SenderName,
				TriggerMessage:   msg.Content,
				ExecutionPath:    "ts",
				Success:          false,
				ErrorMessage:     tsErr.Error(),
				DurationMs:       int(duration.Milliseconds()),
				TriggerMessageID: triggerMsgID,
				WorkflowRevision: &version.Revision,
				RunStatus:        models.RunStatusError,
				ErrorType:        "ts_execution_error",
			})
		}
	}
}

// isBotUser 检查用户是否是 Bot（通过 bots 表判断）
func (e *BotEngine) isBotUser(ctx context.Context, userID uuid.UUID) bool {
	_, err := e.botRepo.FindByID(ctx, userID)
	return err == nil
}

// BotMessage Bot 处理的入站消息（供 TS 客户端使用）
type BotMessage struct {
	ConversationID uuid.UUID
	SenderID       uuid.UUID
	SenderName     string
	Content        string
	MsgType        string
	CreatedAt      time.Time
	MessageID      uuid.UUID
}

// collectContextMessages 收集会话的上下文消息
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
	for i, j := 0, len(contextMessages)-1; i < j; i, j = i+1, j-1 {
		contextMessages[i], contextMessages[j] = contextMessages[j], contextMessages[i]
	}
	return contextMessages
}

// sendSystemMessage 发送系统消息到会话（直接持久化，用于 BotEngine 内部系统消息）
func (e *BotEngine) sendSystemMessage(ctx context.Context, conversationID uuid.UUID, content *models.SystemMessageContent) {
	contentJSON, err := json.Marshal(content)
	if err != nil {
		logger.ErrorfWithCaller("[BotEngine] Failed to marshal system message content: %v", err)
		return
	}

	message := &models.Message{
		SenderID:  uuid.Nil,
		Content:   string(contentJSON),
		MsgType:   models.MsgTypeSystem,
		CreatedAt: time.Now().UTC(),
	}

	err = e.messageRepo.InsertMessage(ctx, conversationID, message)
	if err != nil {
		logger.ErrorfWithCaller("[BotEngine] Failed to insert system message: %v", err)
		return
	}

	logger.InfofWithCaller("[BotEngine] System message sent to conversation %s: type=%s", conversationID, content.Type)
}
