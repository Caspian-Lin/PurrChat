package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"purr-chat-server/internal/handlers"
	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"
	"purr-chat-server/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowAPI(t *testing.T) {
	SetupTestDB(t)
	defer CleanupTestDB(t)

	ctx := context.Background()

	botRepo := repository.NewBotRepository()
	wfRepo := repository.NewWorkflowRepository()
	installationRepo := repository.NewBotInstallationRepository()
	userRepo := repository.NewUserRepository()
	friendshipRepo := repository.NewFriendshipRepository()
	conversationRepo := repository.NewConversationRepository()
	enrollmentRepo := repository.NewEnrollmentRepository()
	messageRepo := repository.NewConversationMessageRepository()
	callLogRepo := repository.NewBotCallLogRepository()

	botService := services.NewBotService(botRepo, installationRepo, userRepo, friendshipRepo, conversationRepo, enrollmentRepo, messageRepo, callLogRepo)
	wfService := services.NewWorkflowService(wfRepo, botRepo, nil)

	owner := CreateTestUser(t, "wf_owner", "wf_owner@test.com", "pass")

	bot, err := botService.CreateBot(ctx, owner.ID.String(), &models.CreateBotRequest{
		Name:            "WFTestBot",
		Discoverability: models.DiscoverabilityUnlisted,
	})
	require.NoError(t, err)

	validDocument := `{
		"apiVersion": "purrchat.ai/v1alpha1",
		"kind": "BotWorkflow",
		"metadata": { "name": "WFTestBot", "revision": 0 },
		"spec": {
			"trigger": { "type": "rule", "rules": [] },
			"nodes": [
				{ "id": "n1", "type": "trigger", "name": "触发", "config": {} },
				{ "id": "n2", "type": "reply", "name": "回复", "config": { "template": "hello" } },
				{ "id": "n3", "type": "end", "name": "结束", "config": {} }
			],
			"connections": [
				{ "id": "c1", "sourceNodeId": "n1", "sourcePortId": "out_exec", "targetNodeId": "n2", "targetPortId": "in_exec" },
				{ "id": "c2", "sourceNodeId": "n2", "sourcePortId": "out_exec", "targetNodeId": "n3", "targetPortId": "in_exec" }
			],
			"endConditions": [{ "type": "max_rounds", "value": 5 }]
		}
	}`

	t.Run("get_workflow_empty", func(t *testing.T) {
		resp, err := wfService.GetWorkflow(ctx, bot.ID.String(), owner.ID.String())
		require.NoError(t, err)
		assert.Equal(t, 0, resp.Revision)
		assert.Equal(t, `"0"`, resp.ETag)
	})

	t.Run("get_workflow_unauthorized", func(t *testing.T) {
		stranger := CreateTestUser(t, "wf_reader", "wf_reader@test.com", "pass")
		_, err := wfService.GetWorkflow(ctx, bot.ID.String(), stranger.ID.String())
		assert.ErrorContains(t, err, "not authorized")
	})

	t.Run("update_workflow", func(t *testing.T) {
		doc := json.RawMessage(validDocument)
		resp, err := wfService.UpdateWorkflow(ctx, bot.ID.String(), owner.ID.String(), &models.UpdateWorkflowRequest{
			Revision: 0,
			Document: doc,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, resp.Revision)
		assert.Equal(t, `"1"`, resp.ETag)
	})

	t.Run("update_revision_mismatch", func(t *testing.T) {
		doc := json.RawMessage(validDocument)
		_, err := wfService.UpdateWorkflow(ctx, bot.ID.String(), owner.ID.String(), &models.UpdateWorkflowRequest{
			Revision: 0,
			Document: doc,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "revision mismatch")
	})

	t.Run("update_unauthorized", func(t *testing.T) {
		stranger := CreateTestUser(t, "wf_stranger", "wf_stranger@test.com", "pass")
		doc := json.RawMessage(validDocument)
		_, err := wfService.UpdateWorkflow(ctx, bot.ID.String(), stranger.ID.String(), &models.UpdateWorkflowRequest{
			Revision: 1,
			Document: doc,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not authorized")
	})

	t.Run("validate_valid_document", func(t *testing.T) {
		doc := json.RawMessage(validDocument)
		resp, err := wfService.ValidateWorkflow(ctx, &models.ValidateWorkflowRequest{Document: doc})
		require.NoError(t, err)
		assert.True(t, resp.Valid)
		assert.Len(t, resp.Issues, 0)
	})

	t.Run("validate_invalid_document", func(t *testing.T) {
		invalidDoc := `{
			"apiVersion": "wrong",
			"kind": "wrong",
			"spec": { "nodes": "not_an_array" }
		}`
		resp, err := wfService.ValidateWorkflow(ctx, &models.ValidateWorkflowRequest{
			Document: json.RawMessage(invalidDoc),
		})
		require.NoError(t, err)
		assert.False(t, resp.Valid)
		assert.True(t, len(resp.Issues) > 0)
	})

	t.Run("validate_rejects_nodes_not_ready_for_production", func(t *testing.T) {
		unsupported := strings.Replace(validDocument, `"type": "reply"`, `"type": "loop"`, 1)
		resp, err := wfService.ValidateWorkflow(ctx, &models.ValidateWorkflowRequest{
			Document: json.RawMessage(unsupported),
		})
		require.NoError(t, err)
		assert.False(t, resp.Valid)
		assert.Contains(t, resp.Issues, models.ValidationResultItem{
			Level:   "error",
			Code:    "node_not_production_ready",
			Message: "节点类型尚未通过生产验证: loop",
			Path:    "spec.nodes[1].type",
		})
	})

	t.Run("publish_workflow", func(t *testing.T) {
		version, err := wfService.PublishWorkflow(ctx, bot.ID.String(), owner.ID.String(), &models.PublishWorkflowRequest{
			Revision: 1,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, version.Revision)
		assert.Contains(t, version.Capabilities, "messages:read_trigger")
		assert.Contains(t, version.Capabilities, "messages:send")

		updatedBot, _ := botRepo.FindByID(ctx, bot.ID)
		assert.Contains(t, updatedBot.RequestedCapabilities, "messages:read_trigger")
		assert.NotNil(t, updatedBot.PublishedVersion)
		assert.Equal(t, 1, *updatedBot.PublishedVersion)
	})

	t.Run("publish_same_revision_is_idempotent_and_immutable", func(t *testing.T) {
		first, err := wfRepo.FindPublishedByRevision(ctx, bot.ID, 1)
		require.NoError(t, err)
		second, err := wfService.PublishWorkflow(ctx, bot.ID.String(), owner.ID.String(), &models.PublishWorkflowRequest{Revision: 1})
		require.NoError(t, err)
		assert.Equal(t, first.ID, second.ID)
		assert.JSONEq(t, string(first.Document), string(second.Document))

		_, err = wfRepo.Publish(ctx, bot.ID, 1, json.RawMessage(`{"different":true}`), first.Capabilities, owner.ID)
		assert.ErrorContains(t, err, "immutable")
		unchanged, err := wfRepo.FindPublishedByRevision(ctx, bot.ID, 1)
		require.NoError(t, err)
		assert.JSONEq(t, string(first.Document), string(unchanged.Document))
	})

	t.Run("list_versions_owner_only", func(t *testing.T) {
		versions, err := wfService.ListPublishedVersions(ctx, bot.ID.String(), owner.ID.String())
		require.NoError(t, err)
		require.Len(t, versions, 1)
		assert.Equal(t, 1, versions[0].Revision)

		stranger := CreateTestUser(t, "wf_versions", "wf_versions@test.com", "pass")
		_, err = wfService.ListPublishedVersions(ctx, bot.ID.String(), stranger.ID.String())
		assert.ErrorContains(t, err, "not authorized")
	})

	t.Run("warnings_do_not_block_save_or_publish", func(t *testing.T) {
		warningDocument := strings.Replace(validDocument, `"endConditions": [{ "type": "max_rounds", "value": 5 }]`, `"unused": true`, 1)
		validation, err := wfService.ValidateWorkflow(ctx, &models.ValidateWorkflowRequest{Document: json.RawMessage(warningDocument)})
		require.NoError(t, err)
		assert.True(t, validation.Valid)
		require.Len(t, validation.Issues, 1)
		assert.Equal(t, "warning", validation.Issues[0].Level)

		draft, err := wfService.UpdateWorkflow(ctx, bot.ID.String(), owner.ID.String(), &models.UpdateWorkflowRequest{
			Revision: 1,
			Document: json.RawMessage(warningDocument),
		})
		require.NoError(t, err)
		_, err = wfService.PublishWorkflow(ctx, bot.ID.String(), owner.ID.String(), &models.PublishWorkflowRequest{Revision: draft.Revision})
		require.NoError(t, err)
	})

	t.Run("rollback_copies_version_to_new_draft_revision", func(t *testing.T) {
		changedDocument := strings.Replace(validDocument, "hello", "changed", 1)
		_, err := wfRepo.UpdateDocument(ctx, bot.ID, json.RawMessage(changedDocument), 2)
		require.NoError(t, err)

		resp, err := wfService.RollbackWorkflow(ctx, bot.ID.String(), owner.ID.String(), 1)
		require.NoError(t, err)
		assert.Equal(t, 4, resp.Revision)
		assert.JSONEq(t, workflowDocumentAtRevision(t, validDocument, 4), string(resp.Document))

		version, err := wfRepo.FindPublishedByRevision(ctx, bot.ID, 1)
		require.NoError(t, err)
		assert.JSONEq(t, workflowDocumentAtRevision(t, validDocument, 1), string(version.Document))
		versions, err := wfService.ListPublishedVersions(ctx, bot.ID.String(), owner.ID.String())
		require.NoError(t, err)
		assert.Len(t, versions, 2)
	})

	t.Run("rollback_unauthorized", func(t *testing.T) {
		stranger := CreateTestUser(t, "wf_rollback", "wf_rollback@test.com", "pass")
		_, err := wfService.RollbackWorkflow(ctx, bot.ID.String(), stranger.ID.String(), 1)
		assert.ErrorContains(t, err, "not authorized")
	})

	t.Run("publish_invalid_document_rejected", func(t *testing.T) {
		badDoc := `{"apiVersion":"purrchat.ai/v1alpha1","kind":"BotWorkflow","metadata":{"name":"x","revision":2},"spec":{"trigger":{"type":"rule"},"nodes":[],"connections":[],"endConditions":[]}}`
		_, err := wfRepo.UpdateDocument(ctx, bot.ID, json.RawMessage(badDoc), 4)
		require.NoError(t, err)

		_, err = wfService.PublishWorkflow(ctx, bot.ID.String(), owner.ID.String(), &models.PublishWorkflowRequest{Revision: 5})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no_trigger")
	})

	t.Run("publish_unauthorized", func(t *testing.T) {
		stranger := CreateTestUser(t, "wf_stranger2", "wf_stranger2@test.com", "pass")
		_, err := wfService.PublishWorkflow(ctx, bot.ID.String(), stranger.ID.String(), &models.PublishWorkflowRequest{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not authorized")
	})

	t.Run("test_run_without_ts_unavailable", func(t *testing.T) {
		_, err := wfService.TestRunWorkflow(ctx, bot.ID.String(), owner.ID.String(), &models.TestRunWorkflowRequest{
			Message: "hello",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bot-engine service")
	})

	t.Run("test_run_unauthorized", func(t *testing.T) {
		stranger := CreateTestUser(t, "wf_runner", "wf_runner@test.com", "pass")
		_, err := wfService.TestRunWorkflow(ctx, bot.ID.String(), stranger.ID.String(), &models.TestRunWorkflowRequest{Message: "hello"})
		assert.ErrorContains(t, err, "not authorized")
	})

	t.Run("handler_uses_user_id_and_checks_if_match", func(t *testing.T) {
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("user_id", c.GetHeader("X-Test-User"))
			c.Next()
		})
		workflowHandler := handlers.NewWorkflowHandler(wfService)
		router.GET("/api/bots/:id/workflow", workflowHandler.GetWorkflow)
		router.PUT("/api/bots/:id/workflow", workflowHandler.UpdateWorkflow)

		ownerRequest := httptest.NewRequest(http.MethodGet, "/api/bots/"+bot.ID.String()+"/workflow", nil)
		ownerRequest.Header.Set("X-Test-User", owner.ID.String())
		ownerResponse := httptest.NewRecorder()
		router.ServeHTTP(ownerResponse, ownerRequest)
		assert.Equal(t, http.StatusOK, ownerResponse.Code)

		stranger := CreateTestUser(t, "wf_handler", "wf_handler@test.com", "pass")
		strangerRequest := httptest.NewRequest(http.MethodGet, "/api/bots/"+bot.ID.String()+"/workflow", nil)
		strangerRequest.Header.Set("X-Test-User", stranger.ID.String())
		strangerResponse := httptest.NewRecorder()
		router.ServeHTTP(strangerResponse, strangerRequest)
		assert.Equal(t, http.StatusForbidden, strangerResponse.Code)

		body := `{"revision":5,"document":` + validDocument + `}`
		updateRequest := httptest.NewRequest(http.MethodPut, "/api/bots/"+bot.ID.String()+"/workflow", bytes.NewBufferString(body))
		updateRequest.Header.Set("Content-Type", "application/json")
		updateRequest.Header.Set("If-Match", `"4"`)
		updateRequest.Header.Set("X-Test-User", owner.ID.String())
		updateResponse := httptest.NewRecorder()
		router.ServeHTTP(updateResponse, updateRequest)
		assert.Equal(t, http.StatusConflict, updateResponse.Code)
		assert.Contains(t, updateResponse.Body.String(), "revision_mismatch")
	})
}

func workflowDocumentAtRevision(t *testing.T, raw string, revision int) string {
	t.Helper()
	var document map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &document))
	metadata := document["metadata"].(map[string]any)
	metadata["revision"] = revision
	encoded, err := json.Marshal(document)
	require.NoError(t, err)
	return string(encoded)
}
