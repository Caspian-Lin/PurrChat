package tests

import (
	"context"
	"testing"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"
	"purr-chat-server/internal/services"
	"purr-chat-server/pkg/database"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// migration009SQL 是 migration 009 的内容(内联以避免工作目录依赖)
const migration009SQL = `
INSERT INTO bot_installations (app_id, installed_by, target_type, target_id, granted_capabilities, diagnostics_consent, status, installed_at)
SELECT
    d.bot_id, d.deployed_by, 'conversation', d.conversation_id,
    COALESCE(b.requested_capabilities, ARRAY[]::text[]),
    CASE WHEN b.requested_capabilities @> ARRAY['network:external']::text[] THEN 'granted' ELSE 'denied' END,
    CASE WHEN d.status = 'paused' THEN 'paused' ELSE 'active' END,
    d.deployed_at
FROM bot_deployments d
JOIN bots b ON d.bot_id = b.id
ON CONFLICT (target_type, target_id, app_id) DO NOTHING;

INSERT INTO bot_installations (app_id, installed_by, target_type, target_id, granted_capabilities, diagnostics_consent, status, installed_at)
SELECT
    f.friend_id, f.user_id, 'user', f.user_id,
    COALESCE(b.requested_capabilities, ARRAY[]::text[]),
    CASE WHEN b.requested_capabilities @> ARRAY['network:external']::text[] THEN 'granted' ELSE 'denied' END,
    CASE WHEN f.status = 'accepted' THEN 'active' ELSE 'paused' END,
    f.created_at
FROM friendships f
JOIN users u ON f.friend_id = u.id AND u.is_bot = true
JOIN bots b ON f.friend_id = b.id
ON CONFLICT (target_type, target_id, app_id) DO NOTHING;
`

// TestMigrationToInstallations 验证 #36 迁移脚本: bot_deployments + Bot friendship → bot_installations,且幂等
func TestMigrationToInstallations(t *testing.T) {
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

	owner := CreateTestUser(t, "mig_owner", "mig_owner@test.com", "pass")
	regularUser := CreateTestUser(t, "mig_user", "mig_user@test.com", "pass")

	// 创建外发 Bot(含 network:external)
	bot, err := botService.CreateBot(ctx, owner.ID.String(), &models.CreateBotRequest{
		Name:            "MigrationBot",
		Description:     "test",
		Discoverability: models.DiscoverabilityListed,
	})
	require.NoError(t, err)

	bot.RequestedCapabilities = []string{models.CapabilityNetworkExternal, models.CapabilitySend}
	require.NoError(t, botRepo.Update(ctx, bot))

	// 创建纯本地 Bot(无 network:external)
	localBot, err := botService.CreateBot(ctx, owner.ID.String(), &models.CreateBotRequest{
		Name:            "LocalBot",
		Description:     "no external",
		Discoverability: models.DiscoverabilityListed,
	})
	require.NoError(t, err)
	localBot.RequestedCapabilities = []string{models.CapabilitySend}
	require.NoError(t, botRepo.Update(ctx, localBot))

	// 创建群聊会话
	groupConv := &models.Conversation{
		ConversationType: models.ConversationTypeGroup,
		CreatedBy:        &owner.ID,
	}
	require.NoError(t, conversationRepo.Create(ctx, groupConv))

	// 1. 构造 bot_deployments 记录(外发 Bot 部署到群聊)
	_, err = database.GetPool().Exec(ctx,
		`INSERT INTO bot_deployments (bot_id, conversation_id, deployed_by, status) VALUES ($1, $2, $3, 'active')`,
		bot.ID, groupConv.ID, owner.ID)
	require.NoError(t, err)

	// 构造 bot_deployments 记录(纯本地 Bot,paused 状态)
	_, err = database.GetPool().Exec(ctx,
		`INSERT INTO bot_deployments (bot_id, conversation_id, deployed_by, status) VALUES ($1, $2, $3, 'paused')`,
		localBot.ID, groupConv.ID, owner.ID)
	require.NoError(t, err)

	// 2. 构造 Bot friendship 记录(普通用户添加了外发 Bot)
	_, err = database.GetPool().Exec(ctx,
		`INSERT INTO friendships (id, user_id, friend_id, status, created_at) VALUES ($1, $2, $3, 'accepted', NOW())`,
		uuid.New(), regularUser.ID, bot.ID)
	require.NoError(t, err)

	// 3. 执行 migration 009
	_, err = database.GetPool().Exec(ctx, migration009SQL)
	require.NoError(t, err)

	// 4. 验证 conversation installation(外发 Bot)
	convInst, err := installationRepo.FindByAppAndTarget(ctx, bot.ID, models.InstallationTargetConversation, groupConv.ID)
	require.NoError(t, err)
	assert.Equal(t, models.DiagnosticsGranted, convInst.DiagnosticsConsent) // network:external → granted
	assert.Contains(t, convInst.GrantedCapabilities, models.CapabilityNetworkExternal)

	// 验证 conversation installation(纯本地 Bot,paused → paused)
	localConvInst, err := installationRepo.FindByAppAndTarget(ctx, localBot.ID, models.InstallationTargetConversation, groupConv.ID)
	require.NoError(t, err)
	assert.Equal(t, models.DiagnosticsDenied, localConvInst.DiagnosticsConsent) // 无 network:external → denied
	assert.Equal(t, models.InstallationPaused, localConvInst.Status)            // paused 状态映射

	// 验证 user installation(普通用户 → 外发 Bot,friendship 回填)
	userInst, err := installationRepo.FindByAppAndTarget(ctx, bot.ID, models.InstallationTargetUser, regularUser.ID)
	require.NoError(t, err)
	assert.Equal(t, models.DiagnosticsGranted, userInst.DiagnosticsConsent)
	assert.Contains(t, userInst.GrantedCapabilities, models.CapabilityNetworkExternal)

	// 记录当前 installation 总数
	insts, _ := installationRepo.FindByApp(ctx, bot.ID)
	countBefore := len(insts)

	// 5. 再次执行 migration(幂等验证)
	_, err = database.GetPool().Exec(ctx, migration009SQL)
	require.NoError(t, err)

	insts2, _ := installationRepo.FindByApp(ctx, bot.ID)
	assert.Len(t, insts2, countBefore, "重复执行迁移不应产生重复 installation")
}

// TestFriendServiceRejectsBot 验证移除 Bot 好友分支后,SendFriendRequest 拒绝 Bot 目标
func TestFriendServiceRejectsBot(t *testing.T) {
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
	friendService := services.NewFriendService(userRepo, friendshipRepo, enrollmentRepo, messageRepo)

	owner := CreateTestUser(t, "fs_owner", "fs_owner@test.com", "pass")
	regularUser := CreateTestUser(t, "fs_user", "fs_user@test.com", "pass")

	bot, err := botService.CreateBot(ctx, owner.ID.String(), &models.CreateBotRequest{
		Name:            "FriendBot",
		Discoverability: models.DiscoverabilityListed,
	})
	require.NoError(t, err)

	// 发送好友请求到 Bot 应被拒绝
	_, err = friendService.SendFriendRequest(ctx, regularUser.ID.String(), bot.ID.String(), func(ctx context.Context, uid, tid string) (*models.Conversation, error) {
		return nil, nil
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bots cannot be added as friends")
}

// TestCreateBotConversationCreatesInstallation 验证 CreateBotConversation 创建 installation(替代 deployment)
func TestCreateBotConversationCreatesInstallation(t *testing.T) {
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

	owner := CreateTestUser(t, "cbc_owner", "cbc_owner@test.com", "pass")
	regularUser := CreateTestUser(t, "cbc_user", "cbc_user@test.com", "pass")

	// 创建 listed Bot(任何用户可添加)
	bot, err := botService.CreateBot(ctx, owner.ID.String(), &models.CreateBotRequest{
		Name:            "ConvBot",
		Discoverability: models.DiscoverabilityListed,
	})
	require.NoError(t, err)
	bot.RequestedCapabilities = []string{models.CapabilityNetworkExternal}
	require.NoError(t, botRepo.Update(ctx, bot))

	// 普通用户创建与 Bot 的私聊
	conv, err := botService.CreateBotConversation(ctx, bot.ID.String(), regularUser.ID.String())
	require.NoError(t, err)
	require.NotNil(t, conv)

	// 验证 installation 存在(target_type=user)
	inst, err := installationRepo.FindByAppAndTarget(ctx, bot.ID, models.InstallationTargetUser, regularUser.ID)
	require.NoError(t, err)
	assert.Equal(t, models.DiagnosticsGranted, inst.DiagnosticsConsent) // network:external → granted
	assert.Contains(t, inst.GrantedCapabilities, models.CapabilityNetworkExternal)

	// 不应有 bot_deployments 记录
	var depCount int
	err = database.GetPool().QueryRow(ctx, "SELECT COUNT(*) FROM bot_deployments WHERE bot_id = $1", bot.ID).Scan(&depCount)
	require.NoError(t, err)
	assert.Equal(t, 0, depCount, "CreateBotConversation 不应创建 bot_deployments 记录")

	// 幂等:再次调用返回同一会话
	conv2, err := botService.CreateBotConversation(ctx, bot.ID.String(), regularUser.ID.String())
	require.NoError(t, err)
	assert.Equal(t, conv.ID, conv2.ID)
}

// TestCreateBotConversationRejectsUnlisted 验证 unlisted Bot 拒绝非 owner 用户
func TestCreateBotConversationRejectsUnlisted(t *testing.T) {
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

	owner := CreateTestUser(t, "unl_owner", "unl_owner@test.com", "pass")
	regularUser := CreateTestUser(t, "unl_user", "unl_user@test.com", "pass")

	// 创建 unlisted Bot(默认,只有 owner 可访问)
	bot, err := botService.CreateBot(ctx, owner.ID.String(), &models.CreateBotRequest{
		Name: "PrivateBot",
	})
	require.NoError(t, err)

	// 非 owner 添加 unlisted Bot 应被拒绝
	_, err = botService.CreateBotConversation(ctx, bot.ID.String(), regularUser.ID.String())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "private")

	// owner 可以添加
	_, err = botService.CreateBotConversation(ctx, bot.ID.String(), owner.ID.String())
	assert.NoError(t, err)
}
