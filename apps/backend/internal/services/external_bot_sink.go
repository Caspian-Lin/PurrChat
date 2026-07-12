package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"purr-chat-server/internal/messaging"
	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"
	"purr-chat-server/pkg/logger"

	"github.com/google/uuid"
)

// BotEventPublisher 推送 OneBot 事件到外部 Bot WebSocket 连接
type BotEventPublisher interface {
	PublishBotEvent(botID uuid.UUID, event any) int
}

// ExternalBotSink 将消息事件推送到已安装的外部 Bot
// 仅向有 messages:read_trigger capability 的 active installation 推送
type ExternalBotSink struct {
	installationRepo repository.BotInstallationRepository
	botRepo          repository.BotRepository
	botWSManager     BotEventPublisher
}

// NewExternalBotSink 创建外部 Bot 事件 sink
func NewExternalBotSink(
	installationRepo repository.BotInstallationRepository,
	botRepo repository.BotRepository,
	botWSManager BotEventPublisher,
) *ExternalBotSink {
	return &ExternalBotSink{
		installationRepo: installationRepo,
		botRepo:          botRepo,
		botWSManager:     botWSManager,
	}
}

// OnMessageCreated 实现 messaging.MessageEventSink
// Bot 发送的消息和系统消息不推送给外部 Bot（防回复环）
func (s *ExternalBotSink) OnMessageCreated(ctx context.Context, event *messaging.MessageCreatedEvent) error {
	if !event.ShouldTriggerBots() {
		return nil
	}
	if s.botWSManager == nil {
		return nil
	}

	installations, err := s.resolveInstallations(ctx, event)
	if err != nil {
		logger.ErrorfWithCaller("[ExternalBotSink] Failed to resolve installations for conversation %s: %v",
			event.Message.ConversationID, err)
		return nil
	}

	for _, inst := range installations {
		// 仅向有 messages:read_trigger capability 的安装推送
		if !models.HasCapability(inst.GrantedCapabilities, models.CapabilityReadTrigger) {
			continue
		}

		// 加载 Bot 信息
		bot, err := s.botRepo.FindByID(ctx, inst.AppID)
		if err != nil {
			logger.ErrorfWithCaller("[ExternalBotSink] Failed to load bot %s: %v", inst.AppID, err)
			continue
		}
		if bot.Status != models.BotStatusActive {
			continue
		}

		// 构建并推送 OneBot 事件
		onebotEvent := s.buildEvent(event, bot)
		delivered := s.botWSManager.PublishBotEvent(bot.ID, onebotEvent)

		if delivered > 0 {
			logger.InfofWithCaller("[ExternalBotSink] Pushed %s event to bot %s (%d connections)",
				onebotEvent.DetailType, bot.Name, delivered)
		}
	}

	return nil
}

func (s *ExternalBotSink) resolveInstallations(ctx context.Context, event *messaging.MessageCreatedEvent) ([]*models.BotInstallation, error) {
	if event.ConversationType != models.ConversationTypeDirect {
		return s.installationRepo.FindActiveByConversation(ctx, event.Message.ConversationID)
	}

	installations := make([]*models.BotInstallation, 0, 1)
	for _, memberID := range event.MemberIDs {
		if memberID == event.Message.SenderID {
			continue
		}
		if _, err := s.botRepo.FindByID(ctx, memberID); err != nil {
			continue
		}
		inst, err := s.installationRepo.FindByAppAndTarget(
			ctx, memberID, models.InstallationTargetUser, event.Message.SenderID,
		)
		if err != nil || inst == nil || inst.Status != models.InstallationActive {
			continue
		}
		installations = append(installations, inst)
	}
	return installations, nil
}

// buildEvent 构建 OneBot 事件
func (s *ExternalBotSink) buildEvent(event *messaging.MessageCreatedEvent, bot *models.Bot) onebotEventPayload {
	detailType := "message.private"
	if event.ConversationType == models.ConversationTypeGroup {
		detailType = "message.group"
	}

	// 构建消息段
	segments := messageToSegments(event.Message)

	data, _ := json.Marshal(onebotMessageEventData{
		MessageID:      event.Message.ID.String(),
		ConversationID: event.Message.ConversationID.String(),
		UserID:         event.Message.SenderID.String(),
		SenderName:     event.SenderName,
		Source:         string(event.Source),
		Message:        segments,
	})

	return onebotEventPayload{
		Time:       event.Message.CreatedAt.Unix(),
		SelfID:     bot.ID.String(),
		PostType:   "message",
		EventID:    generateEventID(),
		DetailType: detailType,
		Data:       data,
	}
}

// messageToSegments 将 PurrChat 消息转换为 OneBot 消息段
func messageToSegments(msg *models.Message) []map[string]any {
	switch msg.MsgType {
	case models.MsgTypeText:
		return []map[string]any{
			{"type": "text", "data": map[string]any{"text": msg.Content}},
		}
	case models.MsgTypeImage:
		return []map[string]any{
			{"type": "image", "data": map[string]any{"url": msg.Content}},
		}
	case models.MsgTypeFile:
		return []map[string]any{
			{"type": "file", "data": map[string]any{"url": msg.Content}},
		}
	default:
		return []map[string]any{
			{"type": "text", "data": map[string]any{"text": msg.Content}},
		}
	}
}

// generateEventID 生成唯一事件 ID
func generateEventID() string {
	return fmt.Sprintf("evt_%s_%d", uuid.New().String()[:8], time.Now().UnixNano())
}

// onebotEventPayload OneBot 事件载荷
type onebotEventPayload struct {
	Time       int64           `json:"time"`
	SelfID     string          `json:"self_id"`
	PostType   string          `json:"post_type"`
	EventID    string          `json:"event_id"`
	DetailType string          `json:"detail_type"`
	Data       json.RawMessage `json:"data"`
}

// onebotMessageEventData 消息事件数据
type onebotMessageEventData struct {
	MessageID      string           `json:"message_id"`
	ConversationID string           `json:"conversation_id"`
	UserID         string           `json:"user_id"`
	SenderName     string           `json:"sender_name,omitempty"`
	Source         string           `json:"source"`
	Message        []map[string]any `json:"message"`
}
