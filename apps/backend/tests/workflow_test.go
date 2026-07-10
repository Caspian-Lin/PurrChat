package tests

import (
	"context"
	"encoding/json"
	"testing"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"
	"purr-chat-server/internal/services"

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
		resp, err := wfService.GetWorkflow(ctx, bot.ID.String())
		require.NoError(t, err)
		assert.Equal(t, 0, resp.Revision)
		assert.Equal(t, `"0"`, resp.ETag)
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

	t.Run("publish_invalid_document_rejected", func(t *testing.T) {
		badDoc := `{"apiVersion":"purrchat.ai/v1alpha1","kind":"BotWorkflow","metadata":{"name":"x","revision":2},"spec":{"trigger":{"type":"rule"},"nodes":[],"connections":[],"endConditions":[]}}`
		_, err := wfRepo.UpdateDocument(ctx, bot.ID, json.RawMessage(badDoc), 1)
		require.NoError(t, err)

		_, err = wfService.PublishWorkflow(ctx, bot.ID.String(), owner.ID.String(), &models.PublishWorkflowRequest{Revision: 2})
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
		_, err := wfService.TestRunWorkflow(ctx, bot.ID.String(), &models.TestRunWorkflowRequest{
			Message: "hello",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bot-engine service")
	})
}
