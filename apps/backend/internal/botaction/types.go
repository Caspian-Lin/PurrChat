package botaction

import "purr-chat-server/internal/onebot"

// ── Request params ──

type sendMessageParams struct {
	ConversationID string                  `json:"conversation_id"`
	Message        []onebot.MessageSegment `json:"message"`
}

type conversationIDParams struct {
	ConversationID string `json:"conversation_id"`
}

type memberInfoParams struct {
	ConversationID string `json:"conversation_id"`
	UserID         string `json:"user_id"`
}

type messageHistoryParams struct {
	ConversationID string `json:"conversation_id"`
	Limit          int    `json:"limit,omitempty"`
	Offset         int    `json:"offset,omitempty"`
}

// ── Response types ──

type sendMessageResponse struct {
	MessageID      string `json:"message_id"`
	ConversationID string `json:"conversation_id"`
	Time           int64  `json:"time"`
}

type loginInfoResponse struct {
	UserID   string `json:"user_id"`
	Nickname string `json:"nickname"`
}

type statusResponse struct {
	Online bool `json:"online"`
	Good   bool `json:"good"`
}

type versionInfoResponse struct {
	Impl          string `json:"impl"`
	Version       string `json:"version"`
	OnebotVersion string `json:"onebot_version"`
}

type conversationInfoResponse struct {
	ConversationID   string `json:"conversation_id"`
	ConversationType string `json:"conversation_type"`
	Name             string `json:"name,omitempty"`
	AvatarURL        string `json:"avatar_url,omitempty"`
}

type memberInfoResponse struct {
	UserID   string `json:"user_id"`
	Nickname string `json:"nickname,omitempty"`
	Role     string `json:"role"`
	IsBot    bool   `json:"is_bot"`
}

type messageDetail struct {
	MessageID      string           `json:"message_id"`
	ConversationID string           `json:"conversation_id"`
	UserID         string           `json:"user_id"`
	Time           int64            `json:"time"`
	Message        []map[string]any `json:"message"`
}
