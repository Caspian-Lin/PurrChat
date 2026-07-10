package tests

import (
	"context"
	"testing"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"
	"purr-chat-server/internal/services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInstallationLifecycle 覆盖 #12 AC7:安装后暂停、禁用、重新授权与权限校验
func TestInstallationLifecycle(t *testing.T) {
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

	owner := CreateTestUser(t, "lc_owner", "lc_owner@test.com", "pass")
	otherUser := CreateTestUser(t, "lc_user", "lc_user@test.com", "pass")

	// helper:创建 listed Bot 并设置 requested_capabilities
	makeBot := func(name string, caps []string) *models.Bot {
		b, err := botService.CreateBot(ctx, owner.ID.String(), &models.CreateBotRequest{
			Name:            name,
			Discoverability: models.DiscoverabilityListed,
		})
		require.NoError(t, err)
		if len(caps) > 0 {
			_, err = botService.UpdateBot(ctx, b.ID.String(), owner.ID.String(), &models.UpdateBotRequest{
				RequestedCapabilities: caps,
			})
			require.NoError(t, err)
		}
		return b
	}

	extCaps := []string{models.CapabilityReadTrigger, models.CapabilitySend, models.CapabilityNetworkExternal}
	localCaps := []string{models.CapabilityReadTrigger, models.CapabilitySend}

	t.Run("pause_and_resume", func(t *testing.T) {
		bot := makeBot("PauseBot", localCaps)
		inst, err := installationService.CreateInstallation(ctx, otherUser.ID.String(), bot.ID.String(), &models.CreateInstallationRequest{
			TargetType: models.InstallationTargetUser,
			TargetID:   otherUser.ID,
		})
		require.NoError(t, err)

		_, err = installationService.UpdateInstallation(ctx, otherUser.ID.String(), inst.ID.String(), &models.UpdateInstallationRequest{
			Status: models.InstallationPaused,
		})
		require.NoError(t, err)
		updated, _ := installationRepo.FindByID(ctx, inst.ID)
		assert.Equal(t, models.InstallationPaused, updated.Status)

		_, err = installationService.UpdateInstallation(ctx, otherUser.ID.String(), inst.ID.String(), &models.UpdateInstallationRequest{
			Status: models.InstallationActive,
		})
		require.NoError(t, err)
		updated, _ = installationRepo.FindByID(ctx, inst.ID)
		assert.Equal(t, models.InstallationActive, updated.Status)
	})

	t.Run("disable", func(t *testing.T) {
		bot := makeBot("DisableBot", localCaps)
		inst, err := installationService.CreateInstallation(ctx, otherUser.ID.String(), bot.ID.String(), &models.CreateInstallationRequest{
			TargetType: models.InstallationTargetUser,
			TargetID:   otherUser.ID,
		})
		require.NoError(t, err)

		_, err = installationService.UpdateInstallation(ctx, otherUser.ID.String(), inst.ID.String(), &models.UpdateInstallationRequest{
			Status: models.InstallationDisabled,
		})
		require.NoError(t, err)
		updated, _ := installationRepo.FindByID(ctx, inst.ID)
		assert.Equal(t, models.InstallationDisabled, updated.Status)
	})

	t.Run("reauthorize_capabilities", func(t *testing.T) {
		bot := makeBot("ReauthBot", extCaps)
		inst, err := installationService.CreateInstallation(ctx, otherUser.ID.String(), bot.ID.String(), &models.CreateInstallationRequest{
			TargetType: models.InstallationTargetUser,
			TargetID:   otherUser.ID,
		})
		require.NoError(t, err)

		_, err = installationService.UpdateInstallation(ctx, otherUser.ID.String(), inst.ID.String(), &models.UpdateInstallationRequest{
			GrantedCapabilities: []string{models.CapabilityReadTrigger},
		})
		require.NoError(t, err)
		updated, _ := installationRepo.FindByID(ctx, inst.ID)
		assert.Contains(t, updated.GrantedCapabilities, models.CapabilityReadTrigger)
		assert.NotContains(t, updated.GrantedCapabilities, models.CapabilityNetworkExternal)
	})

	t.Run("external_bot_diagnostics_cannot_downgrade", func(t *testing.T) {
		bot := makeBot("DiagBot", extCaps)
		inst, err := installationService.CreateInstallation(ctx, otherUser.ID.String(), bot.ID.String(), &models.CreateInstallationRequest{
			TargetType: models.InstallationTargetUser,
			TargetID:   otherUser.ID,
		})
		require.NoError(t, err)
		require.Equal(t, models.DiagnosticsGranted, inst.DiagnosticsConsent)

		// 尝试降级为 denied → 系统静默纠正为 granted(不报错,但值不变)
		updated, err := installationService.UpdateInstallation(ctx, otherUser.ID.String(), inst.ID.String(), &models.UpdateInstallationRequest{
			DiagnosticsConsent: models.DiagnosticsDenied,
		})
		require.NoError(t, err)
		assert.Equal(t, models.DiagnosticsGranted, updated.DiagnosticsConsent)
	})

	t.Run("unauthorized_user_cannot_update", func(t *testing.T) {
		bot := makeBot("PermBot", localCaps)
		inst, err := installationService.CreateInstallation(ctx, otherUser.ID.String(), bot.ID.String(), &models.CreateInstallationRequest{
			TargetType: models.InstallationTargetUser,
			TargetID:   otherUser.ID,
		})
		require.NoError(t, err)

		stranger := CreateTestUser(t, "stranger", "stranger@test.com", "pass")
		_, err = installationService.UpdateInstallation(ctx, stranger.ID.String(), inst.ID.String(), &models.UpdateInstallationRequest{
			Status: models.InstallationPaused,
		})
		assert.Error(t, err)
	})
}
