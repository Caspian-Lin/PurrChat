package tests

import (
	"context"
	"testing"
	"time"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"
	"purr-chat-server/internal/services"
	"purr-chat-server/pkg/database"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBotAppInstallation 覆盖 issue #33 的数据模型与安装 API 验收点
func TestBotAppInstallation(t *testing.T) {
	SetupTestDB(t)
	defer CleanupTestDB(t)

	ctx := context.Background()

	botRepo := repository.NewBotRepository()
	botDeployRepo := repository.NewBotDeploymentRepository()
	installationRepo := repository.NewBotInstallationRepository()
	userRepo := repository.NewUserRepository()
	friendshipRepo := repository.NewFriendshipRepository()
	conversationRepo := repository.NewConversationRepository()
	enrollmentRepo := repository.NewEnrollmentRepository()
	messageRepo := repository.NewConversationMessageRepository()
	callLogRepo := repository.NewBotCallLogRepository()

	botService := services.NewBotService(botRepo, botDeployRepo, installationRepo, userRepo, friendshipRepo, conversationRepo, enrollmentRepo, messageRepo, callLogRepo)
	installationService := services.NewInstallationService(installationRepo, botRepo, enrollmentRepo)

	owner := CreateTestUser(t, "botowner", "owner@test.com", "pass")

	// 创建 Bot 应同时建立 bot_identity、owner user installation,且不再创建 friendship
	t.Run("CreateBot_creates_identity_and_owner_installation_no_friendship", func(t *testing.T) {
		bot, err := botService.CreateBot(ctx, owner.ID.String(), &models.CreateBotRequest{
			Name:        "TestBot",
			Description: "test",
		})
		require.NoError(t, err)
		require.NotNil(t, bot)

		// bot_identity 投影存在
		var identityCount int
		err = database.GetPool().QueryRow(ctx, "SELECT COUNT(*) FROM bot_identities WHERE app_id = $1", bot.ID).Scan(&identityCount)
		require.NoError(t, err)
		assert.Equal(t, 1, identityCount, "bot_identity should exist")

		// owner user installation 存在且 active、diagnostics granted
		inst, err := installationRepo.FindByAppAndTarget(ctx, bot.ID, models.InstallationTargetUser, owner.ID)
		require.NoError(t, err)
		assert.Equal(t, models.InstallationActive, inst.Status)
		assert.Equal(t, models.DiagnosticsGranted, inst.DiagnosticsConsent)

		// 新 Bot 不再创建任何 friendship
		var friendCount int
		err = database.GetPool().QueryRow(ctx, "SELECT COUNT(*) FROM friendships WHERE user_id = $1 OR friend_id = $1", bot.ID).Scan(&friendCount)
		require.NoError(t, err)
		assert.Equal(t, 0, friendCount, "no friendship should be created for new bot")
	})

	// 可见性解耦:旧 visibility 值映射到 discoverability + is_system
	t.Run("visibility_mapping", func(t *testing.T) {
		bot1, err := botService.CreateBot(ctx, owner.ID.String(), &models.CreateBotRequest{Name: "PrivateBot", Visibility: models.BotVisibilityPrivate})
		require.NoError(t, err)
		assert.Equal(t, models.DiscoverabilityUnlisted, bot1.Discoverability)
		assert.False(t, bot1.IsSystem)

		bot2, err := botService.CreateBot(ctx, owner.ID.String(), &models.CreateBotRequest{Name: "PublicBot", Visibility: models.BotVisibilityPublic})
		require.NoError(t, err)
		assert.Equal(t, models.DiscoverabilityListed, bot2.Discoverability)

		bot3, err := botService.CreateBot(ctx, owner.ID.String(), &models.CreateBotRequest{Name: "GlobalBot", Visibility: models.BotVisibilityGlobal})
		require.NoError(t, err)
		assert.Equal(t, models.DiscoverabilityListed, bot3.Discoverability)
		assert.True(t, bot3.IsSystem)
	})

	// 群聊安装:仅 owner/admin 可授权,普通 member 被拒
	t.Run("conversation_install_permissions", func(t *testing.T) {
		conv := &models.Conversation{ConversationType: models.ConversationTypeGroup, CreatedBy: &owner.ID}
		require.NoError(t, conversationRepo.Create(ctx, conv))

		admin := CreateTestUser(t, "adminuser", "admin@test.com", "pass")
		member := CreateTestUser(t, "plainmember", "member@test.com", "pass")

		now := time.Now().UTC()
		require.NoError(t, enrollmentRepo.Create(ctx, &models.Enrollment{ConversationID: conv.ID, UserID: owner.ID, Role: models.EnrollmentRoleOwner, JoinedAt: now}))
		require.NoError(t, enrollmentRepo.Create(ctx, &models.Enrollment{ConversationID: conv.ID, UserID: admin.ID, Role: models.EnrollmentRoleAdmin, JoinedAt: now}))
		require.NoError(t, enrollmentRepo.Create(ctx, &models.Enrollment{ConversationID: conv.ID, UserID: member.ID, Role: models.EnrollmentRoleMember, JoinedAt: now}))

		// owner 安装:成功
		bot, err := botService.CreateBot(ctx, owner.ID.String(), &models.CreateBotRequest{Name: "GroupBot", Visibility: models.BotVisibilityPublic})
		require.NoError(t, err)
		_, err = installationService.CreateInstallation(ctx, owner.ID.String(), bot.ID.String(), &models.CreateInstallationRequest{
			TargetType: models.InstallationTargetConversation,
			TargetID:   conv.ID,
		})
		assert.NoError(t, err)

		// admin 安装另一个 bot:成功
		bot2, err := botService.CreateBot(ctx, owner.ID.String(), &models.CreateBotRequest{Name: "GroupBot2", Visibility: models.BotVisibilityPublic})
		require.NoError(t, err)
		_, err = installationService.CreateInstallation(ctx, admin.ID.String(), bot2.ID.String(), &models.CreateInstallationRequest{
			TargetType: models.InstallationTargetConversation,
			TargetID:   conv.ID,
		})
		assert.NoError(t, err)

		// 普通 member 安装:被拒
		bot3, err := botService.CreateBot(ctx, owner.ID.String(), &models.CreateBotRequest{Name: "GroupBot3", Visibility: models.BotVisibilityPublic})
		require.NoError(t, err)
		_, err = installationService.CreateInstallation(ctx, member.ID.String(), bot3.ID.String(), &models.CreateInstallationRequest{
			TargetType: models.InstallationTargetConversation,
			TargetID:   conv.ID,
		})
		assert.Error(t, err)
	})

	// user 安装:非 owner 只能安装可发现的 Bot(listed/featured)
	t.Run("user_install_discoverability", func(t *testing.T) {
		// owner 的 unlisted bot
		privateBot, err := botService.CreateBot(ctx, owner.ID.String(), &models.CreateBotRequest{Name: "UnlistedBot"})
		require.NoError(t, err)

		otherUser := CreateTestUser(t, "otheruser", "other@test.com", "pass")

		// 非 owner 安装 unlisted bot:被拒
		_, err = installationService.CreateInstallation(ctx, otherUser.ID.String(), privateBot.ID.String(), &models.CreateInstallationRequest{
			TargetType: models.InstallationTargetUser,
			TargetID:   otherUser.ID,
		})
		assert.Error(t, err)

		// owner 已通过 CreateBot 自动获得 user installation
		existing, err := installationRepo.FindByAppAndTarget(ctx, privateBot.ID, models.InstallationTargetUser, owner.ID)
		require.NoError(t, err)
		assert.NotNil(t, existing)

		// listed bot 可被非 owner 安装
		listedBot, err := botService.CreateBot(ctx, owner.ID.String(), &models.CreateBotRequest{Name: "ListedBot", Visibility: models.BotVisibilityPublic})
		require.NoError(t, err)
		_, err = installationService.CreateInstallation(ctx, otherUser.ID.String(), listedBot.ID.String(), &models.CreateInstallationRequest{
			TargetType: models.InstallationTargetUser,
			TargetID:   otherUser.ID,
		})
		assert.NoError(t, err)
	})

	// 卸载:群聊安装卸载时同步移除 enrollment
	t.Run("uninstall_removes_enrollment", func(t *testing.T) {
		conv := &models.Conversation{ConversationType: models.ConversationTypeGroup, CreatedBy: &owner.ID}
		require.NoError(t, conversationRepo.Create(ctx, conv))
		require.NoError(t, enrollmentRepo.Create(ctx, &models.Enrollment{ConversationID: conv.ID, UserID: owner.ID, Role: models.EnrollmentRoleOwner, JoinedAt: time.Now().UTC()}))

		bot, err := botService.CreateBot(ctx, owner.ID.String(), &models.CreateBotRequest{Name: "UninstallBot", Visibility: models.BotVisibilityPublic})
		require.NoError(t, err)

		inst, err := installationService.CreateInstallation(ctx, owner.ID.String(), bot.ID.String(), &models.CreateInstallationRequest{
			TargetType: models.InstallationTargetConversation,
			TargetID:   conv.ID,
		})
		require.NoError(t, err)

		// Bot 已作为 member 加入会话
		botEnroll, err := enrollmentRepo.FindByConversationAndUser(ctx, conv.ID, bot.ID)
		require.NoError(t, err)
		require.NotNil(t, botEnroll)

		// 卸载
		err = installationService.UninstallInstallation(ctx, owner.ID.String(), inst.ID.String())
		require.NoError(t, err)

		// Bot enrollment 已移除
		botEnroll2, err := enrollmentRepo.FindByConversationAndUser(ctx, conv.ID, bot.ID)
		assert.Error(t, err)
		assert.Nil(t, botEnroll2)
	})
}
