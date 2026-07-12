package services

import (
	"context"
	"sync/atomic"
	"time"

	"purr-chat-server/internal/messaging"
	"purr-chat-server/internal/models"
	"purr-chat-server/internal/onebot"
	"purr-chat-server/internal/repository"
	"purr-chat-server/pkg/logger"

	"github.com/google/uuid"
)

type BotEventPublisher interface {
	PublishBotEvent(botID uuid.UUID, event any) int
}

type SinkEventMetrics struct {
	MessageSent    atomic.Int64
	MessageSkipped atomic.Int64
}

type ExternalBotSink struct {
	installationRepo repository.BotInstallationRepository
	botRepo          repository.BotRepository
	botWSManager     BotEventPublisher
	metrics          SinkEventMetrics
	now              func() time.Time
}

func NewExternalBotSink(
	installationRepo repository.BotInstallationRepository,
	botRepo repository.BotRepository,
	botWSManager BotEventPublisher,
) *ExternalBotSink {
	return &ExternalBotSink{
		installationRepo: installationRepo,
		botRepo:          botRepo,
		botWSManager:     botWSManager,
		now:              time.Now,
	}
}

func (s *ExternalBotSink) OnMessageCreated(ctx context.Context, event *messaging.MessageCreatedEvent) error {
	if !event.ShouldTriggerBots() {
		return nil
	}
	if s.botWSManager == nil {
		return nil
	}

	installations, err := s.resolveInstallations(ctx, event)
	if err != nil {
		s.metrics.MessageSkipped.Add(1)
		logger.ErrorfWithCaller("[ExternalBotSink] Failed to resolve installations for conversation %s: %v",
			event.Message.ConversationID, err)
		return nil
	}

	for _, inst := range installations {
		if !models.HasCapability(inst.GrantedCapabilities, models.CapabilityReadTrigger) {
			continue
		}

		bot, err := s.botRepo.FindByID(ctx, inst.AppID)
		if err != nil {
			s.metrics.MessageSkipped.Add(1)
			logger.ErrorfWithCaller("[ExternalBotSink] Failed to load bot %s: %v", inst.AppID, err)
			continue
		}
		if bot.Status != models.BotStatusActive {
			s.metrics.MessageSkipped.Add(1)
			continue
		}

		onebotEvent, err := s.buildEvent(event, bot)
		if err != nil {
			s.metrics.MessageSkipped.Add(1)
			continue
		}

		delivered := s.botWSManager.PublishBotEvent(bot.ID, onebotEvent)

		if delivered > 0 {
			s.metrics.MessageSent.Add(1)
			logger.InfofWithCaller("[ExternalBotSink] Pushed %s event to bot %s (%d connections)",
				onebotEvent.DetailType, bot.Name, delivered)
		} else {
			s.metrics.MessageSkipped.Add(1)
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

func (s *ExternalBotSink) buildEvent(event *messaging.MessageCreatedEvent, bot *models.Bot) (onebot.Event, error) {
	detailType := onebot.DetailTypePrivate
	if event.ConversationType == models.ConversationTypeGroup {
		detailType = onebot.DetailTypeGroup
	}

	segments := messageToSegments(event.Message)

	data := messageEventData{
		MessageID:      event.Message.ID.String(),
		ConversationID: event.Message.ConversationID.String(),
		UserID:         event.Message.SenderID.String(),
		SenderName:     event.SenderName,
		Source:         string(event.Source),
		Message:        segments,
	}

	return onebot.BuildMessageEvent(bot.ID.String(), detailType, event.Message.CreatedAt, data)
}

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

type messageEventData struct {
	MessageID      string           `json:"message_id"`
	ConversationID string           `json:"conversation_id"`
	UserID         string           `json:"user_id"`
	SenderName     string           `json:"sender_name,omitempty"`
	Source         string           `json:"source"`
	Message        []map[string]any `json:"message"`
}
