package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"purr-chat-server/internal/handlers"
	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"
	"purr-chat-server/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPublishDoesNotExpandThirdPartyInstallation 验证发布工作流后:
//   - owner 的私聊安装自动同步新权限
//   - 第三方用户的安装不被自动扩权
func TestPublishDoesNotExpandThirdPartyInstallation(t *testing.T) {
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
	installationService := services.NewInstallationService(installationRepo, botRepo, enrollmentRepo, messageRepo)

	owner := CreateTestUser(t, "reauth_owner", "reauth_owner@test.com", "pass")
	thirdParty := CreateTestUser(t, "reauth_third", "reauth_third@test.com", "pass")

	bot, err := botService.CreateBot(ctx, owner.ID.String(), &models.CreateBotRequest{
		Name:            "ReauthBot",
		Discoverability: models.DiscoverabilityListed,
	})
	require.NoError(t, err)

	// owner 先手动声明旧权限 [read_trigger, send]
	_, err = botService.UpdateBot(ctx, bot.ID.String(), owner.ID.String(), &models.UpdateBotRequest{
		RequestedCapabilities: []string{models.CapabilityReadTrigger, models.CapabilitySend},
	})
	require.NoError(t, err)

	// 第三方用户安装 Bot(仅授予 read_trigger)
	thirdPartyInst, err := installationService.CreateInstallation(ctx, thirdParty.ID.String(), bot.ID.String(), &models.CreateInstallationRequest{
		TargetType:          models.InstallationTargetUser,
		TargetID:            thirdParty.ID,
		GrantedCapabilities: []string{models.CapabilityReadTrigger},
	})
	require.NoError(t, err)

	validDocument := `{
		"apiVersion": "purrchat.ai/v1alpha1",
		"kind": "BotWorkflow",
		"metadata": {"name":"ReauthBot","revision":0},
		"spec":{
			"trigger":{"type":"rule","rules":[]},
			"nodes":[
				{"id":"n1","type":"trigger","name":"t","config":{}},
				{"id":"n2","type":"reply","name":"r","config":{"template":"hi"}},
				{"id":"n3","type":"end","name":"e","config":{}}
			],
			"connections":[
				{"id":"c1","sourceNodeId":"n1","sourcePortId":"out_exec","targetNodeId":"n2","targetPortId":"in_exec"},
				{"id":"c2","sourceNodeId":"n2","sourcePortId":"out_exec","targetNodeId":"n3","targetPortId":"in_exec"}
			],
			"endConditions":[{"type":"max_rounds","value":5}]
		}
	}`

	_, err = wfService.UpdateWorkflow(ctx, bot.ID.String(), owner.ID.String(), &models.UpdateWorkflowRequest{
		Revision: 0,
		Document: json.RawMessage(validDocument),
	})
	require.NoError(t, err)

	// 发布工作流(推导出 [read_trigger, send])
	version, err := wfService.PublishWorkflow(ctx, bot.ID.String(), owner.ID.String(), &models.PublishWorkflowRequest{Revision: 1})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{models.CapabilityReadTrigger, models.CapabilitySend}, version.Capabilities)

	// owner 的私聊安装应自动同步到全部 requested
	ownerInst, err := installationRepo.FindByAppAndTarget(ctx, bot.ID, models.InstallationTargetUser, owner.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, version.Capabilities, ownerInst.GrantedCapabilities)

	// 第三方用户的安装不应被扩权,保持用户授权时的值
	thirdPartyInstAfter, err := installationRepo.FindByID(ctx, thirdPartyInst.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{models.CapabilityReadTrigger}, thirdPartyInstAfter.GrantedCapabilities,
		"第三方安装的权限应保持用户授权时的值,不被发布自动扩权")
}

// TestUpdateInstallationErrorCodes 验证 PATCH installation API 返回结构化错误码和正确 HTTP 状态码
func TestUpdateInstallationErrorCodes(t *testing.T) {
	SetupTestDB(t)
	defer CleanupTestDB(t)

	ctx := context.Background()

	botRepo := repository.NewBotRepository()
	installationRepo := repository.NewBotInstallationRepository()
	userRepo := repository.NewUserRepository()
	friendshipRepo := repository.NewFriendshipRepository()
	conversationRepo := repository.NewConversationRepository()
	enrollmentRepo := repository.NewEnrollmentRepository()
	messageRepo := repository.NewConversationMessageRepository()
	callLogRepo := repository.NewBotCallLogRepository()

	botService := services.NewBotService(botRepo, installationRepo, userRepo, friendshipRepo, conversationRepo, enrollmentRepo, messageRepo, callLogRepo)
	installationService := services.NewInstallationService(installationRepo, botRepo, enrollmentRepo, messageRepo)
	installationHandler := handlers.NewInstallationHandler(installationService)

	owner := CreateTestUser(t, "err_owner", "err_owner@test.com", "pass")
	stranger := CreateTestUser(t, "err_stranger", "err_stranger@test.com", "pass")

	bot, err := botService.CreateBot(ctx, owner.ID.String(), &models.CreateBotRequest{
		Name:            "ErrBot",
		Discoverability: models.DiscoverabilityListed,
	})
	require.NoError(t, err)
	_, err = botService.UpdateBot(ctx, bot.ID.String(), owner.ID.String(), &models.UpdateBotRequest{
		RequestedCapabilities: []string{models.CapabilityReadTrigger, models.CapabilitySend},
	})
	require.NoError(t, err)

	// CreateBot 已自动创建 owner 的 user installation,直接使用
	inst, err := installationRepo.FindByAppAndTarget(ctx, bot.ID, models.InstallationTargetUser, owner.ID)
	require.NoError(t, err)
	// 收缩权限到仅 read_trigger,便于后续测试 granted_exceeds_requested
	inst.GrantedCapabilities = []string{models.CapabilityReadTrigger}
	require.NoError(t, installationRepo.Update(ctx, inst))

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", c.GetHeader("X-Test-User"))
		c.Next()
	})
	router.PATCH("/api/installations/:iid", installationHandler.UpdateInstallation)

	doPatch := func(t *testing.T, iid string, body string, userID string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPatch, "/api/installations/"+iid, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User", userID)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	t.Run("forbidden_when_stranger_patches", func(t *testing.T) {
		w := doPatch(t, inst.ID.String(), `{"status":"paused"}`, stranger.ID.String())
		assert.Equal(t, http.StatusForbidden, w.Code)
		var resp models.APIResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.False(t, resp.Success)
		assert.Equal(t, "forbidden", resp.Code)
	})

	t.Run("not_found_when_installation_missing", func(t *testing.T) {
		w := doPatch(t, uuid.New().String(), `{"status":"paused"}`, owner.ID.String())
		assert.Equal(t, http.StatusNotFound, w.Code)
		var resp models.APIResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.False(t, resp.Success)
		assert.Equal(t, "installation_not_found", resp.Code)
	})

	t.Run("granted_exceeds_requested", func(t *testing.T) {
		w := doPatch(t, inst.ID.String(), `{"granted_capabilities":["messages:read_trigger","secrets:use"]}`, owner.ID.String())
		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp models.APIResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.False(t, resp.Success)
		assert.Equal(t, "granted_exceeds_requested", resp.Code)
	})

	t.Run("success_returns_installation", func(t *testing.T) {
		w := doPatch(t, inst.ID.String(), `{"granted_capabilities":["messages:read_trigger","messages:send"]}`, owner.ID.String())
		assert.Equal(t, http.StatusOK, w.Code)
		var resp models.APIResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.True(t, resp.Success)
	})
}
