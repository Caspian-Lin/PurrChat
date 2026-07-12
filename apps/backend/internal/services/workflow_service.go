package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"purr-chat-server/internal/botengine"
	"purr-chat-server/internal/models"
	"purr-chat-server/pkg/logger"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type WorkflowService struct {
	workflowRepo WorkflowRepo
	botRepo      BotOwnerChecker
	tsClient     TSDebugExecutor
}

type WorkflowRepo interface {
	GetDocument(ctx context.Context, botID uuid.UUID) (json.RawMessage, int, error)
	UpdateDocument(ctx context.Context, botID uuid.UUID, doc json.RawMessage, expectedRevision int) (int, error)
	FindPublishedByBotID(ctx context.Context, botID uuid.UUID) ([]*models.WorkflowVersion, error)
	FindPublishedByRevision(ctx context.Context, botID uuid.UUID, revision int) (*models.WorkflowVersion, error)
	FindLatestPublished(ctx context.Context, botID uuid.UUID) (*models.WorkflowVersion, error)
	Publish(ctx context.Context, botID uuid.UUID, revision int, doc json.RawMessage, capabilities []string, publishedBy uuid.UUID) (*models.WorkflowVersion, error)
}

type BotOwnerChecker interface {
	FindByID(ctx context.Context, id uuid.UUID) (*models.Bot, error)
}

type TSDebugExecutor interface {
	IsAvailable() bool
	TestRun(ctx context.Context, req *botengine.TestRunRequest) (*botengine.TestRunResponse, error)
	TestRunStep(ctx context.Context, sessionID string) (*botengine.TestRunResponse, error)
}

func NewWorkflowService(wfRepo WorkflowRepo, botRepo BotOwnerChecker, tsClient TSDebugExecutor) *WorkflowService {
	return &WorkflowService{workflowRepo: wfRepo, botRepo: botRepo, tsClient: tsClient}
}

func (s *WorkflowService) GetWorkflow(ctx context.Context, botID string, requesterID string) (*models.WorkflowDocumentResponse, error) {
	id, _, err := s.requireOwner(ctx, botID, requesterID, "view workflow")
	if err != nil {
		return nil, err
	}

	doc, revision, err := s.workflowRepo.GetDocument(ctx, id)
	if err != nil {
		return nil, err
	}

	resp := &models.WorkflowDocumentResponse{
		Document: doc,
		Revision: revision,
		ETag:     fmt.Sprintf(`"%d"`, revision),
	}

	pub, err := s.workflowRepo.FindLatestPublished(ctx, id)
	if err == nil && pub != nil {
		resp.PublishedRev = &pub.Revision
	}

	return resp, nil
}

func (s *WorkflowService) UpdateWorkflow(ctx context.Context, botID string, requesterID string, req *models.UpdateWorkflowRequest) (*models.WorkflowDocumentResponse, error) {
	id, err := uuid.Parse(botID)
	if err != nil {
		return nil, err
	}
	requesterUUID, err := uuid.Parse(requesterID)
	if err != nil {
		return nil, err
	}

	bot, err := s.botRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if bot.OwnerID != requesterUUID {
		return nil, errors.New("not authorized: only the bot owner can update workflow")
	}

	document, err := setWorkflowDocumentRevision(req.Document, req.Revision+1)
	if err != nil {
		return nil, err
	}
	issues := validateDocumentStructure(document)
	if hasValidationErrors(issues) {
		return nil, fmt.Errorf("invalid workflow document: %s", formatIssues(issues))
	}

	newRevision, err := s.workflowRepo.UpdateDocument(ctx, id, document, req.Revision)
	if err != nil {
		return nil, err
	}

	logger.InfofWithCaller("[WorkflowService] Bot %s workflow updated: rev %d → %d", botID, req.Revision, newRevision)

	return &models.WorkflowDocumentResponse{
		Document: document,
		Revision: newRevision,
		ETag:     fmt.Sprintf(`"%d"`, newRevision),
	}, nil
}

func (s *WorkflowService) ValidateWorkflow(ctx context.Context, req *models.ValidateWorkflowRequest) (*models.ValidateWorkflowResponse, error) {
	issues := validateDocumentStructure(req.Document)
	if issues == nil {
		issues = []models.ValidationResultItem{}
	}
	resp := &models.ValidateWorkflowResponse{
		Valid:  !hasValidationErrors(issues),
		Issues: issues,
	}
	return resp, nil
}

func (s *WorkflowService) PublishWorkflow(ctx context.Context, botID string, requesterID string, req *models.PublishWorkflowRequest) (*models.WorkflowVersion, error) {
	id, requesterUUID, err := s.requireOwner(ctx, botID, requesterID, "publish workflow")
	if err != nil {
		return nil, err
	}

	if req.Revision != 0 {
		existing, findErr := s.workflowRepo.FindPublishedByRevision(ctx, id, req.Revision)
		if findErr == nil {
			return existing, nil
		}
		if !errors.Is(findErr, pgx.ErrNoRows) {
			return nil, findErr
		}
	}

	doc, revision, err := s.workflowRepo.GetDocument(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Revision != 0 && req.Revision != revision {
		return nil, fmt.Errorf("revision mismatch: expected %d, request has %d", revision, req.Revision)
	}

	issues := validateDocumentStructure(doc)
	if hasValidationErrors(issues) {
		return nil, fmt.Errorf("cannot publish invalid workflow: %s", formatIssues(issues))
	}

	capabilities := deriveCapabilitiesFromDoc(doc)

	version, err := s.workflowRepo.Publish(ctx, id, revision, doc, capabilities, requesterUUID)
	if err != nil {
		return nil, err
	}

	logger.InfofWithCaller("[WorkflowService] Bot %s workflow published: rev %d, capabilities=%v", botID, revision, capabilities)

	return version, nil
}

func (s *WorkflowService) ListPublishedVersions(ctx context.Context, botID string, requesterID string) ([]*models.WorkflowVersion, error) {
	id, _, err := s.requireOwner(ctx, botID, requesterID, "view workflow versions")
	if err != nil {
		return nil, err
	}
	return s.workflowRepo.FindPublishedByBotID(ctx, id)
}

func (s *WorkflowService) RollbackWorkflow(ctx context.Context, botID string, requesterID string, revision int) (*models.WorkflowDocumentResponse, error) {
	id, _, err := s.requireOwner(ctx, botID, requesterID, "rollback workflow")
	if err != nil {
		return nil, err
	}
	version, err := s.workflowRepo.FindPublishedByRevision(ctx, id, revision)
	if err != nil {
		return nil, err
	}
	_, currentRevision, err := s.workflowRepo.GetDocument(ctx, id)
	if err != nil {
		return nil, err
	}
	document, err := setWorkflowDocumentRevision(version.Document, currentRevision+1)
	if err != nil {
		return nil, err
	}
	newRevision, err := s.workflowRepo.UpdateDocument(ctx, id, document, currentRevision)
	if err != nil {
		return nil, err
	}

	return &models.WorkflowDocumentResponse{
		Document: document,
		Revision: newRevision,
		ETag:     fmt.Sprintf(`"%d"`, newRevision),
	}, nil
}

func setWorkflowDocumentRevision(raw json.RawMessage, revision int) (json.RawMessage, error) {
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, fmt.Errorf("invalid workflow document: %w", err)
	}
	metadata, ok := document["metadata"].(map[string]any)
	if !ok {
		return nil, errors.New("invalid workflow document: metadata field is missing")
	}
	metadata["revision"] = revision
	normalized, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("failed to encode workflow document: %w", err)
	}
	return normalized, nil
}

func (s *WorkflowService) TestRunWorkflow(ctx context.Context, botID string, requesterID string, req *models.TestRunWorkflowRequest) (*botengine.TestRunResponse, error) {
	if _, _, err := s.requireOwner(ctx, botID, requesterID, "test workflow"); err != nil {
		return nil, err
	}
	if s.tsClient == nil || !s.tsClient.IsAvailable() {
		return nil, errors.New("test-run requires bot-engine service to be available")
	}

	// 如果请求未携带 document，从数据库加载最新草稿
	document := req.Document
	if len(document) == 0 {
		id, err := uuid.Parse(botID)
		if err != nil {
			return nil, fmt.Errorf("invalid bot ID: %w", err)
		}
		doc, _, err := s.workflowRepo.GetDocument(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to load workflow document: %w", err)
		}
		document = doc
	}

	tsReq := &botengine.TestRunRequest{
		Message:     req.Message,
		Document:    document,
		SideEffects: "mock",
	}

	logger.InfofWithCaller("[WorkflowService] TestRun bot=%s msgLen=%d", botID, len(req.Message))

	result, err := s.tsClient.TestRun(ctx, tsReq)
	if err != nil {
		return nil, fmt.Errorf("test-run delegation failed: %w", err)
	}

	return result, nil
}

func (s *WorkflowService) TestRunStep(ctx context.Context, botID string, requesterID string, sessionID string) (*botengine.TestRunResponse, error) {
	if _, _, err := s.requireOwner(ctx, botID, requesterID, "test workflow"); err != nil {
		return nil, err
	}
	if s.tsClient == nil || !s.tsClient.IsAvailable() {
		return nil, errors.New("test-run requires bot-engine service to be available")
	}

	return s.tsClient.TestRunStep(ctx, sessionID)
}

func (s *WorkflowService) requireOwner(ctx context.Context, botID string, requesterID string, action string) (uuid.UUID, uuid.UUID, error) {
	id, err := uuid.Parse(botID)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	requesterUUID, err := uuid.Parse(requesterID)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	bot, err := s.botRepo.FindByID(ctx, id)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	if bot.OwnerID != requesterUUID {
		return uuid.Nil, uuid.Nil, fmt.Errorf("not authorized: only the bot owner can %s", action)
	}
	return id, requesterUUID, nil
}

// ─── Go 端基础结构校验 ─────────────────────────────────────────

func validateDocumentStructure(raw json.RawMessage) []models.ValidationResultItem {
	var issues []models.ValidationResultItem

	if len(raw) == 0 {
		issues = append(issues, models.ValidationResultItem{
			Level:   "error",
			Code:    "empty_document",
			Message: "文档为空",
		})
		return issues
	}

	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		issues = append(issues, models.ValidationResultItem{
			Level:   "error",
			Code:    "invalid_json",
			Message: fmt.Sprintf("JSON 解析失败: %v", err),
		})
		return issues
	}

	if doc["apiVersion"] != "purrchat.ai/v1alpha1" {
		issues = append(issues, models.ValidationResultItem{
			Level:   "error",
			Code:    "unknown_api_version",
			Message: fmt.Sprintf("apiVersion 必须是 purrchat.ai/v1alpha1, 实际: %v", doc["apiVersion"]),
			Path:    "apiVersion",
		})
	}

	if doc["kind"] != "BotWorkflow" {
		issues = append(issues, models.ValidationResultItem{
			Level:   "error",
			Code:    "unknown_kind",
			Message: fmt.Sprintf("kind 必须是 BotWorkflow, 实际: %v", doc["kind"]),
			Path:    "kind",
		})
	}

	metadata, ok := doc["metadata"].(map[string]any)
	if !ok {
		issues = append(issues, models.ValidationResultItem{
			Level:   "error",
			Code:    "missing_metadata",
			Message: "metadata 字段缺失或类型错误",
			Path:    "metadata",
		})
	} else {
		if name, ok := metadata["name"].(string); !ok || name == "" {
			issues = append(issues, models.ValidationResultItem{
				Level:   "error",
				Code:    "missing_name",
				Message: "metadata.name 必须存在",
				Path:    "metadata.name",
			})
		}
	}

	spec, ok := doc["spec"].(map[string]any)
	if !ok {
		issues = append(issues, models.ValidationResultItem{
			Level:   "error",
			Code:    "missing_spec",
			Message: "spec 字段缺失",
			Path:    "spec",
		})
		return issues
	}

	if _, ok := spec["trigger"].(map[string]any); !ok {
		issues = append(issues, models.ValidationResultItem{
			Level:   "error",
			Code:    "invalid_trigger",
			Message: "spec.trigger 缺失",
			Path:    "spec.trigger",
		})
	}

	nodes, ok := spec["nodes"].([]any)
	if !ok {
		issues = append(issues, models.ValidationResultItem{
			Level:   "error",
			Code:    "invalid_nodes",
			Message: "spec.nodes 必须是数组",
			Path:    "spec.nodes",
		})
	} else {
		triggerCount := 0
		for i, n := range nodes {
			node, ok := n.(map[string]any)
			if !ok {
				issues = append(issues, models.ValidationResultItem{
					Level:   "error",
					Code:    "invalid_node",
					Message: fmt.Sprintf("节点 #%d 不是对象", i),
					Path:    fmt.Sprintf("spec.nodes[%d]", i),
				})
				continue
			}
			if node["type"] == "trigger" {
				triggerCount++
			}
			nodeType, _ := node["type"].(string)
			if !productionWorkflowNodeTypes[nodeType] {
				issues = append(issues, models.ValidationResultItem{
					Level:   "error",
					Code:    "node_not_production_ready",
					Message: fmt.Sprintf("节点类型尚未通过生产验证: %s", nodeType),
					Path:    fmt.Sprintf("spec.nodes[%d].type", i),
				})
			}
			if id, ok := node["id"].(string); !ok || id == "" {
				issues = append(issues, models.ValidationResultItem{
					Level:   "error",
					Code:    "missing_node_id",
					Message: fmt.Sprintf("节点 #%d 缺少 id", i),
					Path:    fmt.Sprintf("spec.nodes[%d].id", i),
				})
			}
		}
		if triggerCount == 0 {
			issues = append(issues, models.ValidationResultItem{
				Level:   "error",
				Code:    "no_trigger",
				Message: "工作流必须包含一个 trigger 节点",
			})
		} else if triggerCount > 1 {
			issues = append(issues, models.ValidationResultItem{
				Level:   "error",
				Code:    "multiple_trigger",
				Message: fmt.Sprintf("工作流只能有一个 trigger 节点, 实际: %d", triggerCount),
			})
		}
	}

	if _, ok := spec["connections"].([]any); !ok {
		issues = append(issues, models.ValidationResultItem{
			Level:   "error",
			Code:    "invalid_connections",
			Message: "spec.connections 必须是数组",
			Path:    "spec.connections",
		})
	}
	if _, ok := spec["endConditions"].([]any); !ok {
		issues = append(issues, models.ValidationResultItem{
			Level:   "warning",
			Code:    "missing_end_conditions",
			Message: "spec.endConditions 缺失, 将使用默认结束条件",
			Path:    "spec.endConditions",
		})
	}

	return issues
}

var productionWorkflowNodeTypes = map[string]bool{
	"trigger":  true,
	"end":      true,
	"wait":     true,
	"if":       true,
	"loop":     true,
	"switch":   true,
	"merge":    true,
	"builtin":  true,
	"template": true,
	"reply":    true,
	"history":  true,
	"tool":     true,
	"dify":     true,
	"n8n":      true,
	"llm":      true,
}

func deriveCapabilitiesFromDoc(raw json.RawMessage) []string {
	var doc struct {
		Spec struct {
			Nodes []struct {
				Type   string         `json:"type"`
				Config map[string]any `json:"config"`
			} `json:"nodes"`
		} `json:"spec"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return []string{}
	}

	caps := map[string]bool{}
	for _, node := range doc.Spec.Nodes {
		for _, c := range models.GetNodeCapabilities(node.Type) {
			caps[c] = true
		}
		if hasSecretRef(node.Config) {
			caps[models.CapabilitySecretsUse] = true
		}
	}

	result := make([]string, 0, len(caps))
	for cap := range caps {
		result = append(result, cap)
	}
	return result
}

func hasSecretRef(config map[string]any) bool {
	for _, v := range config {
		if s, ok := v.(string); ok && strings.HasPrefix(strings.TrimSpace(s), "secrets.") {
			return true
		}
		if m, ok := v.(map[string]any); ok {
			if hasSecretRef(m) {
				return true
			}
		}
	}
	return false
}

func formatIssues(issues []models.ValidationResultItem) string {
	parts := make([]string, len(issues))
	for i, iss := range issues {
		parts[i] = fmt.Sprintf("[%s] %s: %s", iss.Level, iss.Code, iss.Message)
	}
	return strings.Join(parts, "; ")
}

func hasValidationErrors(issues []models.ValidationResultItem) bool {
	for _, issue := range issues {
		if issue.Level == "error" {
			return true
		}
	}
	return false
}
