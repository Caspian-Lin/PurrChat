package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"
	"purr-chat-server/internal/services"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProtectedConversationReads(t *testing.T) {
	SetupTestDB(t)
	SetupTestRouter()
	defer CleanupTestDB(t)

	ctx := context.Background()
	conversationRepo := repository.NewConversationRepository()
	enrollmentRepo := repository.NewEnrollmentRepository()
	messageRepo := repository.NewConversationMessageRepository()

	member := CreateTestUser(t, "readmember", "readmember@test.com", "pass")
	member.Phone = "13800000000"
	stranger := CreateTestUser(t, "readstranger", "readstranger@test.com", "pass")
	conversation := &models.Conversation{ConversationType: models.ConversationTypeGroup, Name: "private group", CreatedBy: &member.ID}
	require.NoError(t, conversationRepo.Create(ctx, conversation))
	require.NoError(t, enrollmentRepo.Create(ctx, &models.Enrollment{
		ConversationID: conversation.ID,
		UserID:         member.ID,
		Role:           models.EnrollmentRoleOwner,
	}))
	require.NoError(t, messageRepo.InsertMessage(ctx, conversation.ID, &models.Message{
		SenderID: member.ID,
		Content:  "secret",
		MsgType:  models.MsgTypeText,
	}))

	memberToken := GetAuthToken(t, member.ID.String())
	strangerToken := GetAuthToken(t, stranger.ID.String())

	paths := []string{
		"/api/messages?conversation_id=" + conversation.ID.String(),
		"/api/messages/export?conversation_id=" + conversation.ID.String(),
		"/api/messages/incremental?conversation_id=" + conversation.ID.String() + "&since_timestamp=1",
		"/api/conversations/members?conversation_id=" + conversation.ID.String(),
	}
	for _, path := range paths {
		t.Run(path+"_member_allowed", func(t *testing.T) {
			response := performAuthorizedGet(t, path, memberToken)
			assert.Equal(t, http.StatusOK, response.Code)
		})
		t.Run(path+"_stranger_hidden", func(t *testing.T) {
			response := performAuthorizedGet(t, path, strangerToken)
			assert.Equal(t, http.StatusNotFound, response.Code)
			assert.JSONEq(t, `{"success":false,"message":"Resource not found"}`, response.Body.String())
		})
	}

	t.Run("removed member loses access immediately", func(t *testing.T) {
		require.NoError(t, enrollmentRepo.DeleteByConversationAndUser(ctx, conversation.ID, member.ID))
		response := performAuthorizedGet(t, paths[0], memberToken)
		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("invalid ID is bad request", func(t *testing.T) {
		response := performAuthorizedGet(t, "/api/messages?conversation_id=invalid", memberToken)
		assert.Equal(t, http.StatusBadRequest, response.Code)
	})
}

func TestConversationMemberProfilesArePublicOnly(t *testing.T) {
	SetupTestDB(t)
	SetupTestRouter()
	defer CleanupTestDB(t)

	ctx := context.Background()
	conversationRepo := repository.NewConversationRepository()
	enrollmentRepo := repository.NewEnrollmentRepository()
	member := CreateTestUser(t, "profilemember", "private@test.com", "pass")
	err := repository.NewUserRepository().Update(ctx, &models.User{
		ID:       member.ID,
		Username: member.Username,
		Email:    "private@test.com",
		Phone:    "13800000000",
	})
	require.NoError(t, err)

	conversation := &models.Conversation{ConversationType: models.ConversationTypeGroup, CreatedBy: &member.ID}
	require.NoError(t, conversationRepo.Create(ctx, conversation))
	require.NoError(t, enrollmentRepo.Create(ctx, &models.Enrollment{ConversationID: conversation.ID, UserID: member.ID, Role: models.EnrollmentRoleOwner}))

	response := performAuthorizedGet(t, "/api/conversations/members?conversation_id="+conversation.ID.String(), GetAuthToken(t, member.ID.String()))
	require.Equal(t, http.StatusOK, response.Code)
	assert.NotContains(t, response.Body.String(), "private@test.com")
	assert.NotContains(t, response.Body.String(), "13800000000")
}

func TestBotResourceAuthorization(t *testing.T) {
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
	botService := services.NewBotService(botRepo, installationRepo, userRepo, friendshipRepo, conversationRepo, enrollmentRepo, messageRepo, repository.NewBotCallLogRepository())
	installationService := services.NewInstallationService(installationRepo, botRepo, enrollmentRepo, messageRepo)

	owner := CreateTestUser(t, "resourceowner", "resourceowner@test.com", "pass")
	admin := CreateTestUser(t, "resourceadmin", "resourceadmin@test.com", "pass")
	member := CreateTestUser(t, "resourcemember", "resourcemember@test.com", "pass")
	stranger := CreateTestUser(t, "resourcestranger", "resourcestranger@test.com", "pass")
	conversation := &models.Conversation{ConversationType: models.ConversationTypeGroup, CreatedBy: &owner.ID}
	require.NoError(t, conversationRepo.Create(ctx, conversation))
	for userID, role := range map[uuid.UUID]models.EnrollmentRole{
		owner.ID: models.EnrollmentRoleOwner, admin.ID: models.EnrollmentRoleAdmin, member.ID: models.EnrollmentRoleMember,
	} {
		require.NoError(t, enrollmentRepo.Create(ctx, &models.Enrollment{ConversationID: conversation.ID, UserID: userID, Role: role}))
	}

	bot, err := botService.CreateBot(ctx, owner.ID.String(), &models.CreateBotRequest{Name: "ProtectedBot", Visibility: models.BotVisibilityPublic})
	require.NoError(t, err)
	bot, err = botService.UpdateBot(ctx, bot.ID.String(), owner.ID.String(), &models.UpdateBotRequest{
		RequestedCapabilities: []string{models.CapabilityReadHistory, models.CapabilityMembersRead},
	})
	require.NoError(t, err)
	installation, err := installationService.CreateInstallation(ctx, owner.ID.String(), bot.ID.String(), &models.CreateInstallationRequest{
		TargetType: models.InstallationTargetConversation,
		TargetID:   conversation.ID,
	})
	require.NoError(t, err)

	_, err = installationService.GetInstallation(ctx, owner.ID.String(), installation.ID.String())
	assert.NoError(t, err)
	_, err = installationService.GetInstallation(ctx, stranger.ID.String(), installation.ID.String())
	assert.ErrorIs(t, err, services.ErrResourceNotFound)
	_, err = installationService.GetInstallation(ctx, stranger.ID.String(), uuid.NewString())
	assert.ErrorIs(t, err, services.ErrResourceNotFound)

	_, err = botService.GetActiveBotsForConversation(ctx, member.ID.String(), conversation.ID.String())
	assert.NoError(t, err)
	_, err = botService.GetActiveBotsForConversation(ctx, stranger.ID.String(), conversation.ID.String())
	assert.ErrorIs(t, err, services.ErrResourceNotFound)

	update := &models.UpdateDeploymentStatusRequest{ConversationID: conversation.ID, Status: string(models.InstallationPaused)}
	assert.ErrorIs(t, botService.UpdateDeploymentStatus(ctx, bot.ID.String(), member.ID.String(), update), services.ErrResourceNotFound)
	require.NoError(t, botService.UpdateDeploymentStatus(ctx, bot.ID.String(), admin.ID.String(), update))
	updated, err := installationRepo.FindByAppAndTarget(ctx, bot.ID, models.InstallationTargetConversation, conversation.ID)
	require.NoError(t, err)
	assert.Equal(t, models.InstallationPaused, updated.Status)

	t.Run("bot capability follows live installation state", func(t *testing.T) {
		updated.Status = models.InstallationActive
		updated.GrantedCapabilities = []string{models.CapabilityReadHistory}
		require.NoError(t, installationRepo.Update(ctx, updated))
		assert.NoError(t, installationService.AuthorizeBotConversationRead(ctx, bot.ID, conversation.ID, models.CapabilityReadHistory))
		assert.ErrorIs(t, installationService.AuthorizeBotConversationRead(ctx, bot.ID, conversation.ID, models.CapabilityMembersRead), services.ErrResourceNotFound)

		updated.Status = models.InstallationPaused
		require.NoError(t, installationRepo.Update(ctx, updated))
		assert.ErrorIs(t, installationService.AuthorizeBotConversationRead(ctx, bot.ID, conversation.ID, models.CapabilityReadHistory), services.ErrResourceNotFound)

		updated.Status = models.InstallationActive
		require.NoError(t, installationRepo.Update(ctx, updated))
		bot.Status = models.BotStatusDisabled
		require.NoError(t, botRepo.Update(ctx, bot))
		assert.ErrorIs(t, installationService.AuthorizeBotConversationRead(ctx, bot.ID, conversation.ID, models.CapabilityReadHistory), services.ErrResourceNotFound)
	})
}

func performAuthorizedGet(t *testing.T, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	testRouter.ServeHTTP(response, req)
	return response
}
