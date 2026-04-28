package services

import (
	"context"
	"errors"
	"time"

	"purr-chat-server/internal/botengine"
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
	botEngine               *botengine.BotEngine
}

// NewMessageService 创建消息服务
func NewMessageService(
	userRepo repository.UserRepository,
	conversationRepo repository.ConversationRepository,
	enrollmentRepo repository.EnrollmentRepository,
	conversationMessageRepo repository.ConversationMessageRepository,
	botRepo repository.BotRepository,
	botEngine *botengine.BotEngine,
) *MessageService {
	return &MessageService{
		userRepo:                userRepo,
		conversationRepo:        conversationRepo,
		enrollmentRepo:          enrollmentRepo,
		conversationMessageRepo: conversationMessageRepo,
		botRepo:                 botRepo,
		botEngine:               botEngine,
	}
}

// GetMessages 获取会话的消息
func (s *MessageService) GetMessages(ctx context.Context, conversationIDStr string, limit, offset int) ([]*models.Message, error) {
	// 解析 conversationID
	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		return nil, err
	}

	messages, err := s.conversationMessageRepo.FindMessages(ctx, conversationID, limit, offset)
	if err != nil {
		return nil, err
	}

	// 为每条消息加载发送者信息
	for _, msg := range messages {
		s.fillSender(ctx, msg)
	}

	return messages, nil
}

// GetAllMessages 获取会话的所有消息
func (s *MessageService) GetAllMessages(ctx context.Context, conversationIDStr string) ([]*models.Message, error) {
	// 解析 conversationID
	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		return nil, err
	}

	messages, err := s.conversationMessageRepo.FindAllMessages(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	// 为每条消息加载发送者信息
	for _, msg := range messages {
		s.fillSender(ctx, msg)
	}

	return messages, nil
}

// GetMessagesIncremental 增量获取会话的消息（从指定时间之后）
func (s *MessageService) GetMessagesIncremental(ctx context.Context, conversationIDStr string, sinceTimestamp int64) ([]*models.Message, error) {
	logger.InfofWithCaller("Getting incremental messages for conversation %s since %d", conversationIDStr, sinceTimestamp)

	// 解析 conversationID
	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		logger.ErrorfWithCaller("Failed to parse conversation ID %s: %v", conversationIDStr, err)
		return nil, err
	}

	// 将时间戳转换为time.Time（使用本地时区，即中国标准时间）
	since := time.UnixMilli(sinceTimestamp).In(time.Local)

	// 获取增量消息
	messages, err := s.conversationMessageRepo.FindByConversationIDSince(ctx, conversationID, since)
	if err != nil {
		logger.ErrorfWithCaller("Failed to get incremental messages: %v", err)
		return nil, err
	}

	// 为每条消息加载发送者信息
	for _, msg := range messages {
		s.fillSender(ctx, msg)
	}

	logger.InfofWithCaller("Retrieved %d incremental messages for conversation %s", len(messages), conversationIDStr)
	return messages, nil
}

// SendMessage 发送消息
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

	// 创建消息（使用UTC时间）
	// 对 text 类型消息内容进行 HTML 转义，防御存储型 XSS
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

	err = s.conversationMessageRepo.InsertMessage(ctx, req.ConversationID, message)
	if err != nil {
		logger.ErrorfWithCaller("Failed to create message: %v", err)
		return nil, err
	}

	// 加载发送者信息
	sender, err := s.userRepo.FindByID(ctx, senderUUID)
	if err == nil {
		sanitizeUser(sender)
		message.Sender = sender
	}

	// 通过WebSocket推送消息给会话的其他成员
	if websocket.GlobalHub != nil {
		// 获取会话的所有成员
		members, err := s.enrollmentRepo.FindByConversationID(ctx, req.ConversationID)
		if err == nil {
			// 提取成员ID列表
			memberIDs := make([]uuid.UUID, 0, len(members))
			for _, member := range members {
				memberIDs = append(memberIDs, member.UserID)
			}
			// 推送消息给所有成员
			websocket.GlobalHub.SendToConversation(req.ConversationID, senderUUID, *message, memberIDs)
			logger.InfofWithCaller("Message broadcasted via WebSocket to %d members", len(memberIDs))
		} else {
			logger.ErrorfWithCaller("Failed to get conversation members for WebSocket broadcast: %v", err)
		}
	}

	logger.InfofWithCaller("Message sent successfully: ID=%s, ConversationID=%s, SenderID=%s", message.ID, message.ConversationID, message.SenderID)

	// 异步触发 Bot 处理
	if s.botEngine != nil {
		s.botEngine.OnMessage(ctx, &botengine.BotMessage{
			ConversationID: req.ConversationID,
			SenderID:       senderUUID,
			SenderName:     "",
			Content:        req.Content,
			MsgType:        req.MsgType,
			CreatedAt:      message.CreatedAt,
		})
	}

	return message, nil
}

// ExportMessages 导出会话的所有消息（别名，内部调用 GetAllMessages）
func (s *MessageService) ExportMessages(ctx context.Context, conversationID string) ([]*models.Message, error) {
	return s.GetAllMessages(ctx, conversationID)
}

// fillSender 为消息填充发送者信息
func (s *MessageService) fillSender(ctx context.Context, msg *models.Message) {
	if msg.BotID != nil && msg.BotName != nil {
		// Bot 消息：用 Bot 信息填充 sender
		if s.botRepo != nil {
			bot, err := s.botRepo.FindByID(ctx, *msg.BotID)
			if err == nil {
				msg.Sender = &models.User{
					ID:        bot.ID,
					Username:  bot.Name,
					AvatarURL: bot.AvatarURL,
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
