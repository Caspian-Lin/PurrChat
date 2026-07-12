package botaction

import (
	"context"
	"encoding/json"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/onebot"
)

func (d *Dispatcher) handleGetMessageHistory(ctx context.Context, principal models.BotPrincipal, request onebot.ActionRequest) (json.RawMessage, error) {
	params, err := onebot.DecodeParams[messageHistoryParams](request)
	if err != nil {
		return nil, err
	}

	conversationID, err := parseConversationID(params.ConversationID)
	if err != nil {
		return nil, err
	}

	if err := d.authorize(ctx, principal, conversationID, models.CapabilityReadHistory); err != nil {
		return nil, err
	}

	limit := params.Limit
	offset := params.Offset
	messages, err := d.messageRepo.FindMessages(ctx, conversationID, limit, offset)
	if err != nil {
		return nil, onebot.NewError(onebot.RetCodeInternal, "failed to fetch messages", err)
	}

	details := make([]messageDetail, 0, len(messages))
	for _, msg := range messages {
		details = append(details, messageDetail{
			MessageID:      msg.ID.String(),
			ConversationID: msg.ConversationID.String(),
			UserID:         msg.SenderID.String(),
			Time:           msg.CreatedAt.Unix(),
			Message:        messageToSegments(msg),
		})
	}

	return marshalData(details)
}

// messageToSegments 将 PurrChat 消息转换为 OneBot 消息段。
func messageToSegments(msg *models.Message) []map[string]any {
	switch msg.MsgType {
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
