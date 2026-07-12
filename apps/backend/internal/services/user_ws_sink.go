package services

import (
	"context"

	"purr-chat-server/internal/messaging"
	"purr-chat-server/internal/websocket"
	"purr-chat-server/pkg/logger"
)

// UserWebSocketSink 将消息事件广播到用户 WebSocket
type UserWebSocketSink struct{}

// NewUserWebSocketSink 创建用户 WS sink
func NewUserWebSocketSink() *UserWebSocketSink {
	return &UserWebSocketSink{}
}

// OnMessageCreated 将消息推送到会话中所有在线用户
func (s *UserWebSocketSink) OnMessageCreated(ctx context.Context, event *messaging.MessageCreatedEvent) error {
	if websocket.GlobalHub == nil {
		return nil
	}
	if event.Message == nil || len(event.MemberIDs) == 0 {
		return nil
	}

	websocket.GlobalHub.SendToConversation(
		event.Message.ConversationID,
		event.Message.SenderID,
		*event.Message,
		event.MemberIDs,
	)

	logger.InfofWithCaller("[UserWebSocketSink] Broadcasted message %s to %d members",
		event.Message.ID, len(event.MemberIDs))
	return nil
}
