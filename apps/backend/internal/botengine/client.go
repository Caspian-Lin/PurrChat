package botengine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"purr-chat-server/internal/models"
	"purr-chat-server/pkg/logger"

	"github.com/google/uuid"
)

// BotEngineClient Bot 微服务 HTTP 客户端
// 用于调用 TypeScript 版 Bot 引擎（apps/bot-engine）
type BotEngineClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewBotEngineClient 创建 Bot 微服务客户端
func NewBotEngineClient(baseURL string) *BotEngineClient {
	return &BotEngineClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// ExecuteRequest 执行请求（对应 TS 服务的 POST /execute）
type ExecuteRequest struct {
	ConversationID      string           `json:"conversation_id"`
	BotID               string           `json:"bot_id"`
	BotName             string           `json:"bot_name"`
	SenderID            string           `json:"sender_id"`
	SenderName          string           `json:"sender_name"`
	Content             string           `json:"content"`
	MsgType             string           `json:"msg_type"`
	MechanismConfig     json.RawMessage  `json:"mechanism_config"`
	ContextMessages     []ContextMessage `json:"context_messages,omitempty"`
	GrantedCapabilities []string         `json:"granted_capabilities,omitempty"`
}

// ExecuteResponse 执行响应
type ExecuteResponse struct {
	Reply         string `json:"reply"`
	SessionActive bool   `json:"session_active"`
	SessionID     string `json:"session_id,omitempty"`
	Triggered     bool   `json:"triggered"`
	MechanismID   string `json:"mechanism_id,omitempty"`
	MechanismName string `json:"mechanism_name,omitempty"`
	ReplyType     string `json:"reply_type,omitempty"`
	ExecutionMs   int    `json:"execution_ms,omitempty"`
}

// Execute 调用 TS 服务执行消息处理
func (c *BotEngineClient) Execute(ctx context.Context, msg *BotMessage, botID uuid.UUID, botName string, mechanismConfig json.RawMessage, contextMessages []ContextMessage, grantedCapabilities []string) (*ExecuteResponse, error) {
	req := ExecuteRequest{
		ConversationID:      msg.ConversationID.String(),
		BotID:               botID.String(),
		BotName:             botName,
		SenderID:            msg.SenderID.String(),
		SenderName:          msg.SenderName,
		Content:             msg.Content,
		MsgType:             msg.MsgType,
		MechanismConfig:     mechanismConfig,
		ContextMessages:     contextMessages,
		GrantedCapabilities: grantedCapabilities,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal execute request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/execute", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create execute request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("execute returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var execResp ExecuteResponse
	if err := json.NewDecoder(resp.Body).Decode(&execResp); err != nil {
		return nil, fmt.Errorf("failed to decode execute response: %w", err)
	}

	return &execResp, nil
}

// DebugRequest 调试请求
type DebugRequest struct {
	Message        string          `json:"message"`
	StepMode       bool            `json:"step_mode,omitempty"`
	SessionID      string          `json:"session_id,omitempty"`
	SenderName     string          `json:"sender_name,omitempty"`
	WorkflowConfig json.RawMessage `json:"workflow_config,omitempty"`
}

// DebugResponse 调试响应（简化版，完整类型在 workflow-types 中）
type DebugResponse struct {
	SessionID      string `json:"session_id"`
	Reply          string `json:"reply"`
	WaitingForStep bool   `json:"waiting_for_step"`
	Round          int    `json:"round"`
}

// DebugExecute 调用 TS 服务的调试执行
func (c *BotEngineClient) DebugExecute(ctx context.Context, botID uuid.UUID, req *models.DebugBotRequest) (*models.DebugTraceResult, error) {
	debugReq := DebugRequest{
		Message:        req.Message,
		StepMode:       req.StepMode,
		SessionID:      req.SessionID,
		SenderName:     req.SenderName,
		WorkflowConfig: req.WorkflowConfig,
	}

	body, err := json.Marshal(debugReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal debug request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/debug", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create debug request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("debug request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("debug returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var debugResp models.DebugTraceResult
	if err := json.NewDecoder(resp.Body).Decode(&debugResp); err != nil {
		return nil, fmt.Errorf("failed to decode debug response: %w", err)
	}

	return &debugResp, nil
}

// HealthCheck 检查 TS 服务健康状态
func (c *BotEngineClient) HealthCheck(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	return nil
}

// IsAvailable 检查 TS 服务是否可用
func (c *BotEngineClient) IsAvailable() bool {
	if c.baseURL == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := c.HealthCheck(ctx)
	if err != nil {
		logger.InfofWithCaller("[BotEngineClient] TS service not available: %v", err)
		return false
	}
	return true
}
