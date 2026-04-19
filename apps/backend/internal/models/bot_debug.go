package models

import "encoding/json"

// DebugBotRequest 调试执行请求
type DebugBotRequest struct {
	Message           string          `json:"message" binding:"required"`
	StepMode          bool            `json:"step_mode"`
	SessionID         string          `json:"session_id"`
	SenderName        string          `json:"sender_name"`
	SpecialModeConfig json.RawMessage `json:"special_mode_config,omitempty"` // 覆盖未保存的配置
}

// DebugStepRequest 逐步执行请求
type DebugStepRequest struct {
	SessionID string `json:"session_id" binding:"required"`
}

// DebugResetRequest 重置调试会话请求
type DebugResetRequest struct {
	SessionID string `json:"session_id" binding:"required"`
}

// DebugTraceResult 调试执行结果
type DebugTraceResult struct {
	SessionID       string                `json:"session_id"`
	Reply           string                `json:"reply"`
	ContextMessages []DebugContextMessage `json:"context_messages"`
	EventTraces     []EventTrace          `json:"event_traces"`
	WaitingForStep  bool                  `json:"waiting_for_step"`
	NextEventID     string                `json:"next_event_id,omitempty"`
	Round           int                   `json:"round"`
}

// EventTrace 单个事件的执行轨迹
type EventTrace struct {
	EventID         string                `json:"event_id"`
	EventType       string                `json:"event_type"`
	EventName       string                `json:"event_name"`
	Status          string                `json:"status"` // pending|running|success|error
	Input           string                `json:"input"`
	Output          string                `json:"output"`
	Error           string                `json:"error,omitempty"`
	DurationMs      int64                 `json:"duration_ms"`
	ContextMessages []DebugContextMessage `json:"context_messages,omitempty"`
}

// DebugContextMessage 调试上下文消息
type DebugContextMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
