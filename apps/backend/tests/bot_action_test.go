package tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"purr-chat-server/internal/botaction"
	"purr-chat-server/internal/messaging"
	"purr-chat-server/internal/models"
	"purr-chat-server/internal/onebot"
	"purr-chat-server/internal/repository"
	"purr-chat-server/internal/services"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type actionTestEnv struct {
	dispatcher *botaction.Dispatcher
	bot        *models.Bot
	owner      *models.User
	otherUser  *models.User
	groupConv  *models.Conversation
	directConv *models.Conversation
	principal  models.BotPrincipal
	instRepo   repository.BotInstallationRepository
	msgRepo    repository.ConversationMessageRepository
	messageSvc *services.MessageService
}

func setupActionTestEnv(t *testing.T) *actionTestEnv {
	t.Helper()
	SetupTestDB(t)

	ctx := context.Background()
	userRepo := repository.NewUserRepository()
	convRepo := repository.NewConversationRepository()
	enrollRepo := repository.NewEnrollmentRepository()
	msgRepo := repository.NewConversationMessageRepository()
	botRepo := repository.NewBotRepository()
	instRepo := repository.NewBotInstallationRepository()

	pub := messaging.NewPublisher(5 * time.Second)
	ms := services.NewMessageService(userRepo, convRepo, enrollRepo, msgRepo, botRepo, instRepo, pub)
	dispatcher := botaction.NewDispatcher(ms, botRepo, userRepo, convRepo, enrollRepo, msgRepo, instRepo)

	owner := CreateTestUser(t, "act_owner", "act_owner@test.com", "pass")
	other := CreateTestUser(t, "act_other", "act_other@test.com", "pass")

	bot := &models.Bot{
		OwnerID:         owner.ID,
		Name:            "ActionBot",
		Status:          models.BotStatusActive,
		BotType:         models.BotTypeExternal,
		Discoverability: models.DiscoverabilityUnlisted,
		RequestedCapabilities: []string{
			models.CapabilitySend,
			models.CapabilityReadHistory,
			models.CapabilityMembersRead,
			models.CapabilityReadTrigger,
		},
	}
	require.NoError(t, botRepo.Create(ctx, bot))

	// group conversation with 3 members + bot installation
	groupConv := &models.Conversation{ConversationType: models.ConversationTypeGroup, Name: "Test Group"}
	groupConv.CreatedBy = &owner.ID
	require.NoError(t, convRepo.Create(ctx, groupConv))
	for _, uid := range []uuid.UUID{owner.ID, other.ID, bot.ID} {
		role := models.EnrollmentRoleMember
		if uid == owner.ID {
			role = models.EnrollmentRoleOwner
		}
		require.NoError(t, enrollRepo.Create(ctx, &models.Enrollment{
			ConversationID: groupConv.ID, UserID: uid, Role: role,
		}))
	}
	require.NoError(t, msgRepo.CreateMessageTable(ctx, groupConv.ID))
	require.NoError(t, instRepo.Create(ctx, &models.BotInstallation{
		AppID: bot.ID, InstalledBy: owner.ID,
		TargetType: models.InstallationTargetConversation, TargetID: groupConv.ID,
		GrantedCapabilities: []string{models.CapabilitySend, models.CapabilityReadHistory, models.CapabilityMembersRead},
		Status:              models.InstallationActive,
	}))

	// direct conversation bot↔owner
	directConv := &models.Conversation{ConversationType: models.ConversationTypeDirect}
	directConv.CreatedBy = &owner.ID
	require.NoError(t, convRepo.Create(ctx, directConv))
	require.NoError(t, enrollRepo.Create(ctx, &models.Enrollment{
		ConversationID: directConv.ID, UserID: owner.ID, Role: models.EnrollmentRoleOwner,
	}))
	require.NoError(t, enrollRepo.Create(ctx, &models.Enrollment{
		ConversationID: directConv.ID, UserID: bot.ID, Role: models.EnrollmentRoleMember,
	}))
	require.NoError(t, msgRepo.CreateMessageTable(ctx, directConv.ID))
	require.NoError(t, instRepo.Create(ctx, &models.BotInstallation{
		AppID: bot.ID, InstalledBy: owner.ID,
		TargetType: models.InstallationTargetUser, TargetID: owner.ID,
		GrantedCapabilities: []string{models.CapabilitySend},
		Status:              models.InstallationActive,
	}))

	return &actionTestEnv{
		dispatcher: dispatcher,
		bot:        bot, owner: owner, otherUser: other,
		groupConv: groupConv, directConv: directConv,
		principal: models.BotPrincipal{BotID: bot.ID, IdentityID: bot.ID},
		instRepo:  instRepo, msgRepo: msgRepo, messageSvc: ms,
	}
}

func dispatchAction(t *testing.T, ctx context.Context, d *botaction.Dispatcher, p models.BotPrincipal, action string, params any) (json.RawMessage, error) {
	t.Helper()
	b, err := json.Marshal(params)
	require.NoError(t, err)
	return d.Dispatch(ctx, p, onebot.ActionRequest{Action: action, Params: b})
}

func TestAction_GetLoginInfo(t *testing.T) {
	env := setupActionTestEnv(t)
	defer CleanupTestDB(t)
	data, err := dispatchAction(t, context.Background(), env.dispatcher, env.principal, "get_login_info", map[string]any{})
	require.NoError(t, err)
	var r struct {
		UserID   string `json:"user_id"`
		Nickname string `json:"nickname"`
	}
	require.NoError(t, json.Unmarshal(data, &r))
	assert.Equal(t, env.bot.ID.String(), r.UserID)
	assert.Equal(t, "ActionBot", r.Nickname)
}

func TestAction_GetStatus(t *testing.T) {
	env := setupActionTestEnv(t)
	defer CleanupTestDB(t)
	data, err := dispatchAction(t, context.Background(), env.dispatcher, env.principal, "get_status", nil)
	require.NoError(t, err)
	var r struct{ Online, Good bool }
	require.NoError(t, json.Unmarshal(data, &r))
	assert.True(t, r.Online)
	assert.True(t, r.Good)
}

func TestAction_GetVersionInfo(t *testing.T) {
	env := setupActionTestEnv(t)
	defer CleanupTestDB(t)
	data, err := dispatchAction(t, context.Background(), env.dispatcher, env.principal, "get_version_info", nil)
	require.NoError(t, err)
	var r struct{ Impl, Version string }
	require.NoError(t, json.Unmarshal(data, &r))
	assert.Equal(t, "PurrChat", r.Impl)
	assert.Equal(t, onebot.ProfileVersion, r.Version)
}

func TestAction_SendMessage_GroupSuccess(t *testing.T) {
	env := setupActionTestEnv(t)
	defer CleanupTestDB(t)
	data, err := dispatchAction(t, context.Background(), env.dispatcher, env.principal, "send_message", map[string]any{
		"conversation_id": env.groupConv.ID.String(),
		"message":         []map[string]any{{"type": "text", "data": map[string]any{"text": "hello group"}}},
	})
	require.NoError(t, err)
	var r struct {
		MessageID      string `json:"message_id"`
		ConversationID string `json:"conversation_id"`
	}
	require.NoError(t, json.Unmarshal(data, &r))
	assert.NotEmpty(t, r.MessageID)
	assert.Equal(t, env.groupConv.ID.String(), r.ConversationID)
}

func TestAction_SendMessage_Alias(t *testing.T) {
	env := setupActionTestEnv(t)
	defer CleanupTestDB(t)
	data, err := dispatchAction(t, context.Background(), env.dispatcher, env.principal, "send_msg", map[string]any{
		"conversation_id": env.groupConv.ID.String(),
		"message":         []map[string]any{{"type": "text", "data": map[string]any{"text": "via alias"}}},
	})
	require.NoError(t, err)
	assert.NotNil(t, data)
}

func TestAction_SendMessage_DirectSuccess(t *testing.T) {
	env := setupActionTestEnv(t)
	defer CleanupTestDB(t)
	data, err := dispatchAction(t, context.Background(), env.dispatcher, env.principal, "send_private_msg", map[string]any{
		"conversation_id": env.directConv.ID.String(),
		"message":         []map[string]any{{"type": "text", "data": map[string]any{"text": "hello direct"}}},
	})
	require.NoError(t, err)
	assert.NotNil(t, data)
}

func TestAction_SendMessage_RejectsNonTextSegment(t *testing.T) {
	env := setupActionTestEnv(t)
	defer CleanupTestDB(t)
	_, err := dispatchAction(t, context.Background(), env.dispatcher, env.principal, "send_message", map[string]any{
		"conversation_id": env.groupConv.ID.String(),
		"message":         []map[string]any{{"type": "image", "data": map[string]any{"file_id": "f"}}},
	})
	require.Error(t, err)
	assert.Equal(t, onebot.RetCodeUnsupportedSegment, onebot.AsError(err).Code)
}

func TestAction_SendMessage_MissingCapability(t *testing.T) {
	env := setupActionTestEnv(t)
	defer CleanupTestDB(t)
	ctx := context.Background()

	noSendBot := &models.Bot{
		OwnerID: env.owner.ID, Name: "NoSendBot", Status: models.BotStatusActive,
		BotType: models.BotTypeExternal, Discoverability: models.DiscoverabilityUnlisted,
		RequestedCapabilities: []string{models.CapabilityReadHistory},
	}
	require.NoError(t, repository.NewBotRepository().Create(ctx, noSendBot))
	require.NoError(t, repository.NewEnrollmentRepository().Create(ctx, &models.Enrollment{
		ConversationID: env.groupConv.ID, UserID: noSendBot.ID, Role: models.EnrollmentRoleMember,
	}))
	require.NoError(t, env.instRepo.Create(ctx, &models.BotInstallation{
		AppID: noSendBot.ID, InstalledBy: env.owner.ID,
		TargetType: models.InstallationTargetConversation, TargetID: env.groupConv.ID,
		GrantedCapabilities: []string{models.CapabilityReadHistory}, Status: models.InstallationActive,
	}))

	_, err := dispatchAction(t, ctx, env.dispatcher,
		models.BotPrincipal{BotID: noSendBot.ID, IdentityID: noSendBot.ID}, "send_message",
		map[string]any{
			"conversation_id": env.groupConv.ID.String(),
			"message":         []map[string]any{{"type": "text", "data": map[string]any{"text": "x"}}},
		})
	require.Error(t, err)
	assert.Equal(t, onebot.RetCodeCapabilityRequired, onebot.AsError(err).Code)
}

func TestAction_GetConversationInfo(t *testing.T) {
	env := setupActionTestEnv(t)
	defer CleanupTestDB(t)
	data, err := dispatchAction(t, context.Background(), env.dispatcher, env.principal, "get_conversation_info", map[string]any{
		"conversation_id": env.groupConv.ID.String(),
	})
	require.NoError(t, err)
	var r struct {
		ConversationID   string `json:"conversation_id"`
		ConversationType string `json:"conversation_type"`
		Name             string `json:"name"`
	}
	require.NoError(t, json.Unmarshal(data, &r))
	assert.Equal(t, env.groupConv.ID.String(), r.ConversationID)
	assert.Equal(t, "group", r.ConversationType)
	assert.Equal(t, "Test Group", r.Name)
}

func TestAction_GetConversationList(t *testing.T) {
	env := setupActionTestEnv(t)
	defer CleanupTestDB(t)
	data, err := dispatchAction(t, context.Background(), env.dispatcher, env.principal, "get_group_list", nil)
	require.NoError(t, err)
	var list []struct{ ConversationID, ConversationType string }
	require.NoError(t, json.Unmarshal(data, &list))
	assert.GreaterOrEqual(t, len(list), 2)
}

func TestAction_GetMemberList(t *testing.T) {
	env := setupActionTestEnv(t)
	defer CleanupTestDB(t)
	data, err := dispatchAction(t, context.Background(), env.dispatcher, env.principal, "get_conversation_member_list", map[string]any{
		"conversation_id": env.groupConv.ID.String(),
	})
	require.NoError(t, err)
	var members []struct{ UserID, Role string }
	require.NoError(t, json.Unmarshal(data, &members))
	assert.GreaterOrEqual(t, len(members), 3)
}

func TestAction_GetMemberList_MissingCapability(t *testing.T) {
	env := setupActionTestEnv(t)
	defer CleanupTestDB(t)
	ctx := context.Background()
	inst, _ := env.instRepo.FindByAppAndTarget(ctx, env.bot.ID, models.InstallationTargetConversation, env.groupConv.ID)
	inst.GrantedCapabilities = []string{models.CapabilitySend}
	require.NoError(t, env.instRepo.Update(ctx, inst))

	_, err := dispatchAction(t, ctx, env.dispatcher, env.principal, "get_conversation_member_list", map[string]any{
		"conversation_id": env.groupConv.ID.String(),
	})
	require.Error(t, err)
	assert.Equal(t, onebot.RetCodeCapabilityRequired, onebot.AsError(err).Code)
}

func TestAction_GetMemberInfo(t *testing.T) {
	env := setupActionTestEnv(t)
	defer CleanupTestDB(t)
	data, err := dispatchAction(t, context.Background(), env.dispatcher, env.principal, "get_conversation_member_info", map[string]any{
		"conversation_id": env.groupConv.ID.String(),
		"user_id":         env.owner.ID.String(),
	})
	require.NoError(t, err)
	var r struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}
	require.NoError(t, json.Unmarshal(data, &r))
	assert.Equal(t, env.owner.ID.String(), r.UserID)
	assert.Equal(t, "owner", r.Role)
}

func TestAction_GetMessageHistory(t *testing.T) {
	env := setupActionTestEnv(t)
	defer CleanupTestDB(t)
	ctx := context.Background()

	_, err := env.messageSvc.SendBotMessage(ctx, &messaging.BotSendRequest{
		BotID: env.bot.ID, ConversationID: env.groupConv.ID,
		Content: "history test", MsgType: "text", Source: messaging.SourceExternal,
	})
	require.NoError(t, err)

	data, err := dispatchAction(t, ctx, env.dispatcher, env.principal, "get_group_msg_history", map[string]any{
		"conversation_id": env.groupConv.ID.String(),
		"limit":           10,
	})
	require.NoError(t, err)
	var msgs []struct {
		MessageID string           `json:"message_id"`
		Message   []map[string]any `json:"message"`
	}
	require.NoError(t, json.Unmarshal(data, &msgs))
	require.GreaterOrEqual(t, len(msgs), 1)
}

func TestAction_GetMessageHistory_MissingCapability(t *testing.T) {
	env := setupActionTestEnv(t)
	defer CleanupTestDB(t)
	ctx := context.Background()
	inst, _ := env.instRepo.FindByAppAndTarget(ctx, env.bot.ID, models.InstallationTargetConversation, env.groupConv.ID)
	inst.GrantedCapabilities = []string{models.CapabilitySend}
	require.NoError(t, env.instRepo.Update(ctx, inst))

	_, err := dispatchAction(t, ctx, env.dispatcher, env.principal, "get_message_history", map[string]any{
		"conversation_id": env.groupConv.ID.String(),
	})
	require.Error(t, err)
	assert.Equal(t, onebot.RetCodeCapabilityRequired, onebot.AsError(err).Code)
}

func TestAction_UnknownAction(t *testing.T) {
	env := setupActionTestEnv(t)
	defer CleanupTestDB(t)
	_, err := dispatchAction(t, context.Background(), env.dispatcher, env.principal, "does_not_exist", nil)
	require.Error(t, err)
	assert.Equal(t, onebot.RetCodeUnknownAction, onebot.AsError(err).Code)
}

func TestAction_RejectedAction(t *testing.T) {
	env := setupActionTestEnv(t)
	defer CleanupTestDB(t)
	_, err := dispatchAction(t, context.Background(), env.dispatcher, env.principal, "get_cookies", nil)
	require.Error(t, err)
	assert.Equal(t, onebot.RetCodePermissionDenied, onebot.AsError(err).Code)
}

func TestAction_SendMessage_NonMember(t *testing.T) {
	env := setupActionTestEnv(t)
	defer CleanupTestDB(t)
	ctx := context.Background()

	strangerConv := &models.Conversation{ConversationType: models.ConversationTypeGroup, Name: "NoBot"}
	strangerConv.CreatedBy = &env.owner.ID
	require.NoError(t, repository.NewConversationRepository().Create(ctx, strangerConv))

	_, err := dispatchAction(t, ctx, env.dispatcher, env.principal, "send_message", map[string]any{
		"conversation_id": strangerConv.ID.String(),
		"message":         []map[string]any{{"type": "text", "data": map[string]any{"text": "intruder"}}},
	})
	require.Error(t, err)
	assert.Equal(t, onebot.RetCodePermissionDenied, onebot.AsError(err).Code)
}
