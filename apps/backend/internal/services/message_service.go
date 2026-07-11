package services

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"purr-chat-server/internal/messaging"
	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"
	"purr-chat-server/internal/websocket"
	"purr-chat-server/pkg/logger"
	"purr-chat-server/pkg/utils"

	"github.com/google/uuid"
)

// MessageService 消息服务
type MessageService struct {
	userRepo                repository.UserRepository
	conversationRepo        repository.ConversationRepository
	enrollmentRepo          repository.EnrollmentRepository
	conversationMessageRepo repository.ConversationMessageRepository
	botRepo                 repository.BotRepository
	installationRepo        repository.BotInstallationRepository
	publisher               *messaging.Publisher
}

// NewMessageService 创建消息服务
func NewMessageService(
	userRepo repository.UserRepository,
	conversationRepo repository.ConversationRepository,
	enrollmentRepo repository.EnrollmentRepository,
	conversationMessageRepo repository.ConversationMessageRepository,
	botRepo repository.BotRepository,
	installationRepo repository.BotInstallationRepository,
	publisher *messaging.Publisher,
) *MessageService {
	return &MessageService{
		userRepo:                userRepo,
		conversationRepo:        conversationRepo,
		enrollmentRepo:          enrollmentRepo,
		conversationMessageRepo: conversationMessageRepo,
		botRepo:                 botRepo,
		installationRepo:        installationRepo,
		publisher:               publisher,
	}
}

// GetMessages 获取会话的消息
func (s *MessageService) GetMessages(ctx context.Context, requesterIDStr, conversationIDStr string, limit, offset int) ([]*models.Message, error) {
	requesterID, err := parseID(requesterIDStr)
	if err != nil {
		return nil, err
	}
	conversationID, err := parseID(conversationIDStr)
	if err != nil {
		return nil, err
	}
	if err := requireConversationMember(ctx, s.enrollmentRepo, conversationID, requesterID); err != nil {
		return nil, err
	}

	return s.getMessages(ctx, conversationID, limit, offset)
}

func (s *MessageService) getMessages(ctx context.Context, conversationID uuid.UUID, limit, offset int) ([]*models.Message, error) {
	messages, err := s.conversationMessageRepo.FindMessages(ctx, conversationID, limit, offset)
	if err != nil {
		return nil, err
	}
	for _, msg := range messages {
		s.fillSender(ctx, msg)
	}
	return messages, nil
}

func (s *MessageService) getAllMessages(ctx context.Context, conversationID uuid.UUID) ([]*models.Message, error) {
	messages, err := s.conversationMessageRepo.FindAllMessages(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	for _, msg := range messages {
		s.fillSender(ctx, msg)
	}

	return messages, nil
}

// GetMessagesIncremental 增量获取会话的消息（从指定时间之后）
func (s *MessageService) GetMessagesIncremental(ctx context.Context, requesterIDStr, conversationIDStr string, sinceTimestamp int64) ([]*models.Message, error) {
	logger.InfofWithCaller("Getting incremental messages for conversation %s since %d", conversationIDStr, sinceTimestamp)

	requesterID, err := parseID(requesterIDStr)
	if err != nil {
		return nil, err
	}
	conversationID, err := parseID(conversationIDStr)
	if err != nil {
		logger.ErrorfWithCaller("Failed to parse conversation ID %s: %v", conversationIDStr, err)
		return nil, err
	}
	if err := requireConversationMember(ctx, s.enrollmentRepo, conversationID, requesterID); err != nil {
		return nil, err
	}

	since := time.UnixMilli(sinceTimestamp).In(time.Local)

	messages, err := s.conversationMessageRepo.FindByConversationIDSince(ctx, conversationID, since)
	if err != nil {
		logger.ErrorfWithCaller("Failed to get incremental messages: %v", err)
		return nil, err
	}

	for _, msg := range messages {
		s.fillSender(ctx, msg)
	}

	logger.InfofWithCaller("Retrieved %d incremental messages for conversation %s", len(messages), conversationIDStr)
	return messages, nil
}

// SendMessage 发送消息（用户入口）
func (s *MessageService) SendMessage(ctx context.Context, senderID string, req *models.SendMessageRequest) (*models.Message, error) {
	logger.InfofWithCaller("Sending message from user %s to conversation %s", senderID, req.ConversationID)

	senderUUID, err := uuid.Parse(senderID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to parse sender ID %s: %v", senderID, err)
		return nil, err
	}

	// 检查发送者是否是会话的参与者
	_, err = s.enrollmentRepo.FindByConversationAndUser(ctx, req.ConversationID, senderUUID)
	if err != nil {
		logger.ErrorfWithCaller("User %s is not a participant in conversation %s", senderID, req.ConversationID)
		return nil, errors.New("not a participant in this conversation")
	}

	// 禁止系统占位用户发送消息
	if senderUUID == deletedUserID {
		logger.ErrorfWithCaller("Deleted user placeholder attempted to send message")
		return nil, errors.New("user not found")
	}

	// 幂等性检查
	if req.ClientMessageID != "" {
		existing, err := s.conversationMessageRepo.FindByClientMessageID(ctx, req.ConversationID, req.ClientMessageID)
		if err == nil && existing != nil {
			logger.InfofWithCaller("Idempotent hit: client_message_id=%s, existing_message_id=%s", req.ClientMessageID, existing.ID)
			sender, _ := s.userRepo.FindByID(ctx, senderUUID)
			if sender != nil {
				sanitizeUser(sender)
				existing.Sender = sender
			}
			return existing, nil
		}
	}

	// 内容处理
	content := req.Content
	if req.MsgType == "text" {
		content = utils.EscapeHTML(content)
	}

	message := &models.Message{
		ID:             uuid.New(),
		ConversationID: req.ConversationID,
		SenderID:       senderUUID,
		Content:        content,
		MsgType:        models.MsgType(req.MsgType),
		CreatedAt:      time.Now().UTC(),
	}
	if req.ClientMessageID != "" {
		message.ClientMessageID = &req.ClientMessageID
	}

	if err := s.conversationMessageRepo.InsertMessage(ctx, req.ConversationID, message); err != nil {
		logger.ErrorfWithCaller("Failed to create message: %v", err)
		return nil, err
	}

	// 加载发送者信息
	sender, err := s.userRepo.FindByID(ctx, senderUUID)
	if err == nil {
		sanitizeUser(sender)
		message.Sender = sender
	}

	senderName := ""
	if sender != nil {
		senderName = sender.Username
	}

	s.publishMessageCreated(ctx, message, messaging.ActorUser, messaging.SourceUser, senderName, "", nil)

	logger.InfofWithCaller("Message sent successfully: ID=%s, ConversationID=%s, SenderID=%s", message.ID, message.ConversationID, message.SenderID)
	return message, nil
}

// SendBotMessage 统一 Bot 消息发送入口（实现 messaging.BotMessageSender）
// 校验 Bot active、enrollment、active installation 与 messages:send capability，
// 复用内容校验、HTML 处理、持久化和用户 WS 广播。
func (s *MessageService) SendBotMessage(ctx context.Context, req *messaging.BotSendRequest) (*models.Message, error) {
	// 1. 校验 Bot 存在且 active
	bot, err := s.botRepo.FindByID(ctx, req.BotID)
	if err != nil {
		return nil, errors.New("bot not found")
	}
	if bot.Status != models.BotStatusActive {
		return nil, errors.New("bot is not active")
	}

	// 2. 校验 Bot 是会话成员
	_, err = s.enrollmentRepo.FindByConversationAndUser(ctx, req.ConversationID, req.BotID)
	if err != nil {
		return nil, errors.New("bot is not a participant in this conversation")
	}

	// 3. 根据会话类型精确校验 installation，群聊不能借用用户安装。
	conv, err := s.conversationRepo.FindByID(ctx, req.ConversationID)
	if err != nil {
		return nil, errors.New("conversation not found")
	}
	members, err := s.enrollmentRepo.FindByConversationID(ctx, req.ConversationID)
	if err != nil {
		return nil, errors.New("failed to verify conversation members")
	}

	var inst *models.BotInstallation
	if conv.ConversationType == models.ConversationTypeDirect {
		if len(members) != 2 {
			return nil, errors.New("direct conversation must have exactly 2 members")
		}
		var targetUserID uuid.UUID
		for _, member := range members {
			if member.UserID != req.BotID {
				targetUserID = member.UserID
			}
		}
		if targetUserID == uuid.Nil {
			return nil, errors.New("direct conversation must be between bot and user")
		}
		inst, err = s.installationRepo.FindByAppAndTarget(ctx, req.BotID, models.InstallationTargetUser, targetUserID)
	} else {
		inst, err = s.installationRepo.FindByAppAndTarget(ctx, req.BotID, models.InstallationTargetConversation, req.ConversationID)
	}
	if err != nil || inst == nil {
		return nil, errors.New("no active installation found for this bot")
	}
	if inst.Status != models.InstallationActive {
		return nil, errors.New("bot installation is not active")
	}

	// 4. 校验 messages:send capability
	if !models.HasCapability(inst.GrantedCapabilities, models.CapabilitySend) {
		return nil, errors.New("bot does not have messages:send capability")
	}

	// 5. 内容处理
	msgType := req.MsgType
	if msgType == "" {
		msgType = "text"
	}
	content := req.Content
	if msgType == "text" {
		content = utils.EscapeHTML(content)
	}

	// 6. 持久化
	botID := bot.ID
	botName := bot.Name
	message := &models.Message{
		ID:             uuid.New(),
		ConversationID: req.ConversationID,
		SenderID:       bot.ID,
		Content:        content,
		MsgType:        models.MsgType(msgType),
		CreatedAt:      time.Now().UTC(),
		BotID:          &botID,
		BotName:        &botName,
	}

	if err := s.conversationMessageRepo.InsertMessage(ctx, req.ConversationID, message); err != nil {
		logger.ErrorfWithCaller("[MessageService] Failed to insert bot message: %v", err)
		return nil, err
	}

	// 填充 sender
	message.Sender = &models.User{
		ID:        bot.ID,
		Username:  bot.Name,
		AvatarURL: bot.AvatarURL,
		IsBot:     true,
	}

	// 7. 发布事件（ActorType=bot，workflow/external sink 会跳过，防回复环）
	s.publishMessageCreated(ctx, message, messaging.ActorBot, req.Source, bot.Name, req.RunID, req.TriggerMessageID)

	logger.InfofWithCaller("[MessageService] Bot %s sent message to conversation %s", bot.Name, req.ConversationID)
	return message, nil
}

// SendPokeMessage 发送拍一拍消息
func (s *MessageService) SendPokeMessage(ctx context.Context, senderID string, conversationID uuid.UUID, targetUserID uuid.UUID) (*models.Message, error) {
	logger.InfofWithCaller("Sending poke from user %s to user %s in conversation %s", senderID, targetUserID, conversationID)

	senderUUID, err := uuid.Parse(senderID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to parse sender ID %s: %v", senderID, err)
		return nil, err
	}

	_, err = s.enrollmentRepo.FindByConversationAndUser(ctx, conversationID, senderUUID)
	if err != nil {
		logger.ErrorfWithCaller("User %s is not a participant in conversation %s", senderID, conversationID)
		return nil, errors.New("not a participant in this conversation")
	}

	targetUser, err := s.userRepo.FindByID(ctx, targetUserID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to find target user %s: %v", targetUserID, err)
		return nil, errors.New("target user not found")
	}

	sysContent := models.SystemMessageContent{
		Type:     "poke",
		UserID:   targetUserID.String(),
		UserName: targetUser.Username,
	}
	contentBytes, err := json.Marshal(sysContent)
	if err != nil {
		logger.ErrorfWithCaller("Failed to marshal system message content: %v", err)
		return nil, err
	}

	message := &models.Message{
		ID:             uuid.New(),
		ConversationID: conversationID,
		SenderID:       senderUUID,
		Content:        string(contentBytes),
		MsgType:        models.MsgTypeSystem,
		CreatedAt:      time.Now().UTC(),
	}

	if err := s.conversationMessageRepo.InsertMessage(ctx, conversationID, message); err != nil {
		logger.ErrorfWithCaller("Failed to insert poke message: %v", err)
		return nil, err
	}

	sender, err := s.userRepo.FindByID(ctx, senderUUID)
	if err == nil {
		sanitizeUser(sender)
		message.Sender = sender
	}

	senderName := ""
	if sender != nil {
		senderName = sender.Username
	}
	s.publishMessageCreated(ctx, message, messaging.ActorUser, messaging.SourceUser, senderName, "", nil)

	logger.InfofWithCaller("Poke message sent successfully: ID=%s, ConversationID=%s, SenderID=%s, TargetUserID=%s", message.ID, message.ConversationID, message.SenderID, targetUserID)
	return message, nil
}

// ExportMessages 导出会话的所有消息
func (s *MessageService) ExportMessages(ctx context.Context, requesterIDStr, conversationIDStr string) ([]*models.Message, error) {
	requesterID, err := parseID(requesterIDStr)
	if err != nil {
		return nil, err
	}
	conversationID, err := parseID(conversationIDStr)
	if err != nil {
		return nil, err
	}
	if err := requireConversationMember(ctx, s.enrollmentRepo, conversationID, requesterID); err != nil {
		return nil, err
	}
	return s.getAllMessages(ctx, conversationID)
}

// publishMessageCreated 构建事件并 fan-out 到所有 sink
func (s *MessageService) publishMessageCreated(ctx context.Context, message *models.Message, actor messaging.ActorType, source messaging.MessageSource, senderName, runID string, triggerMsgID *uuid.UUID) {
	if s.publisher == nil {
		// 兼容未配置 publisher 的场景（如单元测试）
		if websocket.GlobalHub != nil {
			s.broadcastToUserWS(ctx, message)
		}
		return
	}

	// 获取会话成员列表
	var memberIDs []uuid.UUID
	var convType models.ConversationType
	members, err := s.enrollmentRepo.FindByConversationID(ctx, message.ConversationID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to get conversation members for event: %v", err)
	} else {
		memberIDs = make([]uuid.UUID, 0, len(members))
		for _, m := range members {
			memberIDs = append(memberIDs, m.UserID)
		}
	}

	conv, err := s.conversationRepo.FindByID(ctx, message.ConversationID)
	if err == nil {
		convType = conv.ConversationType
	}

	event := &messaging.MessageCreatedEvent{
		Message:          message,
		ActorType:        actor,
		Source:           source,
		SenderName:       senderName,
		MemberIDs:        memberIDs,
		ConversationType: convType,
		RunID:            runID,
		TriggerMessageID: triggerMsgID,
	}

	// 消息已经提交；subscriber 在请求生命周期外执行，失败不会回滚消息或阻塞响应。
	go s.publisher.Publish(context.Background(), event)
}

// broadcastToUserWS 直接广播到用户 WS（备用，当 publisher 未配置时使用）
func (s *MessageService) broadcastToUserWS(ctx context.Context, message *models.Message) {
	if websocket.GlobalHub == nil {
		return
	}
	members, err := s.enrollmentRepo.FindByConversationID(ctx, message.ConversationID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to get conversation members for WebSocket broadcast: %v", err)
		return
	}
	memberIDs := make([]uuid.UUID, 0, len(members))
	for _, member := range members {
		memberIDs = append(memberIDs, member.UserID)
	}
	websocket.GlobalHub.SendToConversation(message.ConversationID, message.SenderID, *message, memberIDs)
	logger.InfofWithCaller("Message broadcasted via WebSocket to %d members", len(memberIDs))
}

// fillSender 为消息填充发送者信息
func (s *MessageService) fillSender(ctx context.Context, msg *models.Message) {
	if msg.BotID != nil && msg.BotName != nil {
		if s.botRepo != nil {
			bot, err := s.botRepo.FindByID(ctx, *msg.BotID)
			if err == nil {
				msg.Sender = &models.User{
					ID:        bot.ID,
					Username:  bot.Name,
					AvatarURL: bot.AvatarURL,
					IsBot:     true,
				}
			}
		}
	} else {
		sender, err := s.userRepo.FindByID(ctx, msg.SenderID)
		if err == nil {
			sanitizeUser(sender)
			msg.Sender = sender
		}
	}
}
