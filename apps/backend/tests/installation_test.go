package tests

import (
	"context"
	"encoding/json"
	"strings"
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
	installationRepo := repository.NewBotInstallationRepository()
	userRepo := repository.NewUserRepository()
	friendshipRepo := repository.NewFriendshipRepository()
	conversationRepo := repository.NewConversationRepository()
	enrollmentRepo := repository.NewEnrollmentRepository()
	messageRepo := repository.NewConversationMessageRepository()
	callLogRepo := repository.NewBotCallLogRepository()

	botService := services.NewBotService(botRepo, installationRepo, userRepo, friendshipRepo, conversationRepo, enrollmentRepo, messageRepo, callLogRepo)
	installationService := services.NewInstallationService(installationRepo, botRepo, enrollmentRepo, messageRepo)

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

	// capability:granted 必须是 requested 的子集
	t.Run("mechanism_config_derives_capabilities_and_syncs_owner_installation", func(t *testing.T) {
		bot, err := botService.CreateBot(ctx, owner.ID.String(), &models.CreateBotRequest{Name: "FixedReplyBot"})
		require.NoError(t, err)

		_, err = botService.UpdateBot(ctx, bot.ID.String(), owner.ID.String(), &models.UpdateBotRequest{
			MechanismConfig: json.RawMessage(`{
				"mechanisms": [{
					"id": "fixed",
					"name": "固定回复",
					"enabled": true,
					"trigger": {"type": "rule", "rules": []},
					"reply": {"type": "predefined", "predefined": {"mode": "fixed", "replies": ["你好"]}}
				}]
			}`),
		})
		require.NoError(t, err)

		updatedBot, err := botService.GetBot(ctx, bot.ID.String())
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{
			models.CapabilityReadTrigger,
			models.CapabilitySend,
		}, updatedBot.RequestedCapabilities)

		installation, err := installationRepo.FindByAppAndTarget(ctx, bot.ID, models.InstallationTargetUser, owner.ID)
		require.NoError(t, err)
		assert.ElementsMatch(t, updatedBot.RequestedCapabilities, installation.GrantedCapabilities)
	})

	// capability:granted 必须是 requested 的子集
	t.Run("capability_granted_subset_requested", func(t *testing.T) {
		otherUser := CreateTestUser(t, "capuser", "cap@test.com", "pass")

		// Bot A 声明 [read_trigger, send, network:external]
		botA, err := botService.CreateBot(ctx, owner.ID.String(), &models.CreateBotRequest{Name: "CapBotA", Visibility: models.BotVisibilityPublic})
		require.NoError(t, err)
		_, err = botService.UpdateBot(ctx, botA.ID.String(), owner.ID.String(), &models.UpdateBotRequest{
			RequestedCapabilities: []string{models.CapabilityReadTrigger, models.CapabilitySend, models.CapabilityNetworkExternal},
		})
		require.NoError(t, err)

		// 合法缩减:只授予 read_trigger + send(去掉 network:external)
		inst, err := installationService.CreateInstallation(ctx, otherUser.ID.String(), botA.ID.String(), &models.CreateInstallationRequest{
			TargetType:          models.InstallationTargetUser,
			TargetID:            otherUser.ID,
			GrantedCapabilities: []string{models.CapabilityReadTrigger, models.CapabilitySend},
		})
		require.NoError(t, err)
		assert.Len(t, inst.GrantedCapabilities, 2)

		// 未授予 network:external 时不会共享外部诊断数据
		assert.Equal(t, models.DiagnosticsDenied, inst.DiagnosticsConsent)

		// 重新授权仍必须满足 granted ⊆ requested
		_, err = installationService.UpdateInstallation(ctx, otherUser.ID.String(), inst.ID.String(), &models.UpdateInstallationRequest{
			GrantedCapabilities: []string{models.CapabilityReadTrigger, models.CapabilitySecretsUse},
		})
		assert.Error(t, err)

		updated, err := installationService.UpdateInstallation(ctx, otherUser.ID.String(), inst.ID.String(), &models.UpdateInstallationRequest{
			GrantedCapabilities: []string{models.CapabilityReadTrigger, models.CapabilitySend, models.CapabilityNetworkExternal},
			DiagnosticsConsent:  models.DiagnosticsDenied,
		})
		require.NoError(t, err)
		assert.Equal(t, models.DiagnosticsGranted, updated.DiagnosticsConsent)

		// 安装列表必须携带完整 Bot 权限声明，供前端再次授权时展示。
		deployments, err := botService.GetBotDeployments(ctx, otherUser.ID.String())
		require.NoError(t, err)
		var installedBot *models.Bot
		for _, deployment := range deployments {
			if deployment.AppID == botA.ID {
				installedBot = deployment.App
				break
			}
		}
		require.NotNil(t, installedBot)
		assert.ElementsMatch(t, []string{
			models.CapabilityReadTrigger,
			models.CapabilitySend,
			models.CapabilityNetworkExternal,
		}, installedBot.RequestedCapabilities)

		// Bot B 只声明 [read_trigger, send](不含 network:external)
		botB, err := botService.CreateBot(ctx, owner.ID.String(), &models.CreateBotRequest{Name: "CapBotB", Visibility: models.BotVisibilityPublic})
		require.NoError(t, err)
		_, err = botService.UpdateBot(ctx, botB.ID.String(), owner.ID.String(), &models.UpdateBotRequest{
			RequestedCapabilities: []string{models.CapabilityReadTrigger, models.CapabilitySend},
		})
		require.NoError(t, err)

		// 非法:granted 包含 requested 未声明的 capability(secrets:use)
		_, err = installationService.CreateInstallation(ctx, otherUser.ID.String(), botB.ID.String(), &models.CreateInstallationRequest{
			TargetType:          models.InstallationTargetUser,
			TargetID:            otherUser.ID,
			GrantedCapabilities: []string{models.CapabilityReadTrigger, models.CapabilitySecretsUse},
		})
		assert.Error(t, err)

		// 纯本地 Bot(无 network:external)→ diagnostics 默认 denied
		instB, err := installationService.CreateInstallation(ctx, otherUser.ID.String(), botB.ID.String(), &models.CreateInstallationRequest{
			TargetType: models.InstallationTargetUser,
			TargetID:   otherUser.ID,
		})
		require.NoError(t, err)
		assert.Equal(t, models.DiagnosticsDenied, instB.DiagnosticsConsent)
	})

	// 外发 Bot 安装到群聊后强制发系统消息告知成员
	t.Run("external_bot_group_warning", func(t *testing.T) {
		conv := &models.Conversation{ConversationType: models.ConversationTypeGroup, CreatedBy: &owner.ID}
		require.NoError(t, conversationRepo.Create(ctx, conv))
		require.NoError(t, enrollmentRepo.Create(ctx, &models.Enrollment{ConversationID: conv.ID, UserID: owner.ID, Role: models.EnrollmentRoleOwner, JoinedAt: time.Now().UTC()}))

		// 声明 network:external 的 Bot
		extBot, err := botService.CreateBot(ctx, owner.ID.String(), &models.CreateBotRequest{Name: "ExtBot", Visibility: models.BotVisibilityPublic})
		require.NoError(t, err)
		_, err = botService.UpdateBot(ctx, extBot.ID.String(), owner.ID.String(), &models.UpdateBotRequest{
			RequestedCapabilities: []string{models.CapabilityReadTrigger, models.CapabilityNetworkExternal, models.CapabilitySend},
		})
		require.NoError(t, err)

		// 安装到群聊
		inst, err := installationService.CreateInstallation(ctx, owner.ID.String(), extBot.ID.String(), &models.CreateInstallationRequest{
			TargetType: models.InstallationTargetConversation,
			TargetID:   conv.ID,
		})
		require.NoError(t, err)

		// diagnostics 强制 granted(外发 Bot)
		assert.Equal(t, models.DiagnosticsGranted, inst.DiagnosticsConsent)

		// 系统消息 bot_external_warning 已插入(用 messageRepo 查,因消息走分表函数)
		msgs, err := messageRepo.FindAllMessages(ctx, conv.ID)
		require.NoError(t, err)
		found := false
		for _, m := range msgs {
			if strings.Contains(m.Content, "bot_external_warning") {
				found = true
				break
			}
		}
		assert.True(t, found, "external warning system message should be inserted")

		// 纯本地 Bot 安装不产生外发警告
		localBot, err := botService.CreateBot(ctx, owner.ID.String(), &models.CreateBotRequest{Name: "LocalBot", Visibility: models.BotVisibilityPublic})
		require.NoError(t, err)
		conv2 := &models.Conversation{ConversationType: models.ConversationTypeGroup, CreatedBy: &owner.ID}
		require.NoError(t, conversationRepo.Create(ctx, conv2))
		require.NoError(t, enrollmentRepo.Create(ctx, &models.Enrollment{ConversationID: conv2.ID, UserID: owner.ID, Role: models.EnrollmentRoleOwner, JoinedAt: time.Now().UTC()}))
		_, err = installationService.CreateInstallation(ctx, owner.ID.String(), localBot.ID.String(), &models.CreateInstallationRequest{
			TargetType: models.InstallationTargetConversation,
			TargetID:   conv2.ID,
		})
		require.NoError(t, err)

		msgs2, err := messageRepo.FindAllMessages(ctx, conv2.ID)
		require.NoError(t, err)
		found2 := false
		for _, m := range msgs2 {
			if strings.Contains(m.Content, "bot_external_warning") {
				found2 = true
				break
			}
		}
		assert.False(t, found2, "local bot should not produce external warning")
	})
}
