package tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"
	"purr-chat-server/internal/services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConversationBotBroadcast 验证安装到群聊后:
//   - bot_deployed 系统消息插入会话
//   - Bot 出现在 GetActiveBotsForConversation 结果中
//   - 卸载后 bot_undeployed 系统消息插入, Bot 从列表消失
func TestConversationBotBroadcast(t *testing.T) {
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

	owner := CreateTestUser(t, "convbot_owner", "convbot_owner@test.com", "pass")

	bot, err := botService.CreateBot(ctx, owner.ID.String(), &models.CreateBotRequest{
		Name:            "ConvBot",
		Discoverability: models.DiscoverabilityListed,
	})
	require.NoError(t, err)

	_, err = botService.UpdateBot(ctx, bot.ID.String(), owner.ID.String(), &models.UpdateBotRequest{
		RequestedCapabilities: []string{models.CapabilityReadTrigger, models.CapabilitySend},
	})
	require.NoError(t, err)

	conv := &models.Conversation{ConversationType: models.ConversationTypeGroup, CreatedBy: &owner.ID}
	require.NoError(t, conversationRepo.Create(ctx, conv))

	now := time.Now().UTC()
	require.NoError(t, enrollmentRepo.Create(ctx, &models.Enrollment{ConversationID: conv.ID, UserID: owner.ID, Role: models.EnrollmentRoleOwner, JoinedAt: now}))

	_, err = installationService.CreateInstallation(ctx, owner.ID.String(), bot.ID.String(), &models.CreateInstallationRequest{
		TargetType: models.InstallationTargetConversation,
		TargetID:   conv.ID,
	})
	require.NoError(t, err)

	bots, err := botService.GetActiveBotsForConversation(ctx, owner.ID.String(), conv.ID.String())
	require.NoError(t, err)
	require.Len(t, bots, 1)
	assert.Equal(t, bot.ID, bots[0].AppID)
	assert.Equal(t, "ConvBot", bots[0].App.Name)

	msgs, err := messageRepo.FindMessages(ctx, conv.ID, 50, 0)
	require.NoError(t, err)
	require.True(t, len(msgs) >= 1)
	var foundDeployed bool
	for _, m := range msgs {
		if m.MsgType == models.MsgTypeSystem {
			var sys models.SystemMessageContent
			if json.Unmarshal([]byte(m.Content), &sys) == nil && sys.Type == "bot_deployed" {
				foundDeployed = true
				assert.Equal(t, "ConvBot", sys.BotName)
			}
		}
	}
	assert.True(t, foundDeployed, "bot_deployed system message should exist")

	inst, err := installationRepo.FindByAppAndTarget(ctx, bot.ID, models.InstallationTargetConversation, conv.ID)
	require.NoError(t, err)
	require.NotNil(t, inst)

	err = installationService.UninstallInstallation(ctx, owner.ID.String(), inst.ID.String())
	require.NoError(t, err)

	msgsAfter, err := messageRepo.FindMessages(ctx, conv.ID, 50, 0)
	require.NoError(t, err)
	var foundUndeployed bool
	for _, m := range msgsAfter {
		if m.MsgType == models.MsgTypeSystem {
			var sys models.SystemMessageContent
			if json.Unmarshal([]byte(m.Content), &sys) == nil && sys.Type == "bot_undeployed" {
				foundUndeployed = true
			}
		}
	}
	assert.True(t, foundUndeployed, "bot_undeployed system message should exist")

	botsAfter, err := botService.GetActiveBotsForConversation(ctx, owner.ID.String(), conv.ID.String())
	require.NoError(t, err)
	assert.Empty(t, botsAfter)
}
