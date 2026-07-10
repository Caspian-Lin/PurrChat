package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// WorkflowVersion 已发布的工作流版本
type WorkflowVersion struct {
	ID           uuid.UUID       `json:"id"`
	BotID        uuid.UUID       `json:"bot_id"`
	Revision     int             `json:"revision"`
	Document     json.RawMessage `json:"document"`
	Capabilities []string        `json:"capabilities"`
	PublishedBy  *uuid.UUID      `json:"published_by,omitempty"`
	PublishedAt  time.Time       `json:"published_at"`
}

// UpdateWorkflowRequest PUT /api/bots/:id/workflow
type UpdateWorkflowRequest struct {
	Revision int             `json:"revision"`
	Document json.RawMessage `json:"document"`
}

// ValidateWorkflowRequest POST /api/bots/:id/workflow/validate
type ValidateWorkflowRequest struct {
	Document json.RawMessage `json:"document"`
}

// PublishWorkflowRequest POST /api/bots/:id/workflow/publish
type PublishWorkflowRequest struct {
	Revision int `json:"revision"`
}

// TestRunWorkflowRequest POST /api/bots/:id/workflow/test-runs
type TestRunWorkflowRequest struct {
	Message  string          `json:"message"`
	Document json.RawMessage `json:"document,omitempty"`
}

// WorkflowDocumentResponse GET /api/bots/:id/workflow
type WorkflowDocumentResponse struct {
	Document     json.RawMessage `json:"document"`
	Revision     int             `json:"revision"`
	ETag         string          `json:"etag"`
	PublishedRev *int            `json:"published_revision,omitempty"`
}

// ValidationResultItem 单条校验结果
type ValidationResultItem struct {
	Level        string `json:"level"`
	Code         string `json:"code"`
	Message      string `json:"message"`
	Path         string `json:"path,omitempty"`
	NodeID       string `json:"node_id,omitempty"`
	ConnectionID string `json:"connection_id,omitempty"`
}

// ValidateWorkflowResponse 校验响应
type ValidateWorkflowResponse struct {
	Valid               bool                   `json:"valid"`
	Issues              []ValidationResultItem `json:"issues"`
	DerivedCapabilities []string               `json:"derived_capabilities,omitempty"`
}
