package botaction

import (
	"context"
	"encoding/json"
	"strings"

	"purr-chat-server/internal/messaging"
	"purr-chat-server/internal/models"
	"purr-chat-server/internal/onebot"
)

func (d *Dispatcher) handleSendMessage(ctx context.Context, principal models.BotPrincipal, request onebot.ActionRequest) (json.RawMessage, error) {
	params, err := onebot.DecodeParams[sendMessageParams](request)
	if err != nil {
		return nil, err
	}

	conversationID, err := parseConversationID(params.ConversationID)
	if err != nil {
		return nil, err
	}

	segments := params.Message
	if segments == nil {
		return nil, onebot.NewError(onebot.RetCodeInvalidParams, "message is required", nil)
	}
	if err := onebot.RequireStableSegments(segments); err != nil {
		return nil, err
	}

	content := extractText(segments)
	if content == "" {
		return nil, onebot.NewError(onebot.RetCodeInvalidParams, "message must contain non-empty text", nil)
	}

	msg, err := d.messageSender.SendBotMessage(ctx, &messaging.BotSendRequest{
		BotID:          principal.BotID,
		ConversationID: conversationID,
		Content:        content,
		MsgType:        "text",
		Source:         messaging.SourceExternal,
	})
	if err != nil {
		return nil, mapSendError(err)
	}

	return marshalData(sendMessageResponse{
		MessageID:      msg.ID.String(),
		ConversationID: msg.ConversationID.String(),
		Time:           msg.CreatedAt.Unix(),
	})
}

// extractText 从纯文本消息段中拼接出完整内容。
func extractText(segments []onebot.MessageSegment) string {
	var b strings.Builder
	for _, seg := range segments {
		if seg.Type != "text" {
			continue
		}
		var td onebot.TextData
		if err := json.Unmarshal(seg.Data, &td); err == nil {
			b.WriteString(td.Text)
		}
	}
	return b.String()
}

// mapSendError 将 SendBotMessage 的业务错误映射为 OneBot RetCode。
func mapSendError(err error) error {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "not active"):
		return onebot.NewError(onebot.RetCodePermissionDenied, "bot is not active", nil)
	case strings.Contains(msg, "not a participant"):
		return onebot.NewError(onebot.RetCodePermissionDenied, "bot is not a member of this conversation", nil)
	case strings.Contains(msg, "no active installation"):
		return onebot.NewError(onebot.RetCodeInstallationInactive, "no active installation found", nil)
	case strings.Contains(msg, "installation is not active"):
		return onebot.NewError(onebot.RetCodeInstallationInactive, "bot installation is not active", nil)
	case strings.Contains(msg, "does not have"):
		return onebot.NewError(onebot.RetCodeCapabilityRequired, "messages:send capability is required", nil)
	default:
		return onebot.NewError(onebot.RetCodeInternal, "failed to send message", err)
	}
}
