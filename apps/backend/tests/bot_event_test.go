package tests

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"purr-chat-server/internal/botws"
	"purr-chat-server/internal/messaging"
	"purr-chat-server/internal/models"
	"purr-chat-server/internal/onebot"
	"purr-chat-server/internal/repository"
	"purr-chat-server/internal/services"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type capturedEvent struct {
	BotID uuid.UUID
	Event onebot.Event
}

type eventCollector struct {
	mu     sync.Mutex
	events []capturedEvent
}

func newEventCollector() *eventCollector {
	return &eventCollector{}
}

func (c *eventCollector) PublishBotEvent(botID uuid.UUID, event any) int {
	data, err := json.Marshal(event)
	if err != nil {
		return 0
	}
	var ev onebot.Event
	if json.Unmarshal(data, &ev) != nil {
		return 0
	}
	c.mu.Lock()
	c.events = append(c.events, capturedEvent{BotID: botID, Event: ev})
	c.mu.Unlock()
	return 1
}

func (c *eventCollector) Events() []capturedEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]capturedEvent, len(c.events))
	copy(out, c.events)
	return out
}

func (c *eventCollector) Clear() {
	c.mu.Lock()
	c.events = nil
	c.mu.Unlock()
}

func (c *eventCollector) Filter(detailType string) []capturedEvent {
	var out []capturedEvent
	for _, e := range c.Events() {
		if e.Event.DetailType == detailType {
			out = append(out, e)
		}
	}
	return out
}

type eventTestEnv struct {
	collector       *eventCollector
	emitter         *services.BotNoticeEmitter
	memberSvc       *services.MemberService
	installSvc      *services.InstallationService
	externalSink    *services.ExternalBotSink
	botRepo         repository.BotRepository
	instRepo        repository.BotInstallationRepository
	owner           *models.User
	otherUser       *models.User
	bot             *models.Bot
	noMemberReadBot *models.Bot
	groupConv       *models.Conversation
}

func setupEventTestEnv(t *testing.T) *eventTestEnv {
	t.Helper()
	SetupTestDB(t)
	ctx := context.Background()

	userRepo := repository.NewUserRepository()
	convRepo := repository.NewConversationRepository()
	enrollRepo := repository.NewEnrollmentRepository()
	msgRepo := repository.NewConversationMessageRepository()
	botRepo := repository.NewBotRepository()
	instRepo := repository.NewBotInstallationRepository()

	collector := newEventCollector()
	emitter := services.NewBotNoticeEmitter(instRepo, botRepo, collector)
	memberSvc := services.NewMemberService(userRepo, convRepo, enrollRepo)
	memberSvc.SetBotNoticeEmitter(emitter)
	installSvc := services.NewInstallationService(instRepo, botRepo, enrollRepo, msgRepo)
	installSvc.SetBotNoticeEmitter(emitter)

	pub := messaging.NewPublisher(5 * time.Second)
	externalSink := services.NewExternalBotSink(instRepo, botRepo, collector)
	pub.RegisterSink("external_bot", externalSink)

	owner := CreateTestUser(t, "evt_owner", "evt_owner@test.com", "pass")
	other := CreateTestUser(t, "evt_other", "evt_other@test.com", "pass")

	bot := &models.Bot{
		OwnerID:         owner.ID,
		Name:            "EventBot",
		Status:          models.BotStatusActive,
		BotType:         models.BotTypeExternal,
		Discoverability: models.DiscoverabilityUnlisted,
		RequestedCapabilities: []string{
			models.CapabilityMembersRead,
			models.CapabilityReadTrigger,
			models.CapabilitySend,
		},
	}
	require.NoError(t, botRepo.Create(ctx, bot))

	noMemberReadBot := &models.Bot{
		OwnerID:         owner.ID,
		Name:            "NoMemberReadBot",
		Status:          models.BotStatusActive,
		BotType:         models.BotTypeExternal,
		Discoverability: models.DiscoverabilityUnlisted,
		RequestedCapabilities: []string{
			models.CapabilityReadTrigger,
			models.CapabilitySend,
		},
	}
	require.NoError(t, botRepo.Create(ctx, noMemberReadBot))

	groupConv := &models.Conversation{ConversationType: models.ConversationTypeGroup, Name: "Event Group"}
	groupConv.CreatedBy = &owner.ID
	require.NoError(t, convRepo.Create(ctx, groupConv))
	for _, uid := range []uuid.UUID{owner.ID, other.ID, bot.ID, noMemberReadBot.ID} {
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
		GrantedCapabilities: []string{models.CapabilityMembersRead, models.CapabilityReadTrigger, models.CapabilitySend},
		Status:              models.InstallationActive,
	}))
	require.NoError(t, instRepo.Create(ctx, &models.BotInstallation{
		AppID: noMemberReadBot.ID, InstalledBy: owner.ID,
		TargetType: models.InstallationTargetConversation, TargetID: groupConv.ID,
		GrantedCapabilities: []string{models.CapabilityReadTrigger, models.CapabilitySend},
		Status:              models.InstallationActive,
	}))

	return &eventTestEnv{
		collector:       collector,
		emitter:         emitter,
		memberSvc:       memberSvc,
		installSvc:      installSvc,
		externalSink:    externalSink,
		botRepo:         botRepo,
		instRepo:        instRepo,
		owner:           owner,
		otherUser:       other,
		bot:             bot,
		noMemberReadBot: noMemberReadBot,
		groupConv:       groupConv,
	}
}

func TestEvent_MemberJoined(t *testing.T) {
	env := setupEventTestEnv(t)
	defer CleanupTestDB(t)
	ctx := context.Background()

	newUser := CreateTestUser(t, "evt_joiner", "evt_joiner@test.com", "pass")
	err := env.memberSvc.AddMemberToConversation(ctx, env.groupConv.ID.String(), env.owner.ID.String(), newUser.ID.String(), models.EnrollmentRoleMember)
	require.NoError(t, err)

	notices := env.collector.Filter(onebot.NoticeGroupMemberIncrease)
	require.Len(t, notices, 1, "bot with members:read should receive increase notice")
	assert.Equal(t, env.bot.ID, notices[0].BotID)
	assert.Equal(t, onebot.PostTypeNotice, notices[0].Event.PostType)

	var data map[string]any
	require.NoError(t, json.Unmarshal(notices[0].Event.Data, &data))
	assert.Equal(t, env.groupConv.ID.String(), data["conversation_id"])
	assert.Equal(t, newUser.ID.String(), data["user_id"])
}

func TestEvent_MemberJoined_MissingCapability(t *testing.T) {
	env := setupEventTestEnv(t)
	defer CleanupTestDB(t)
	ctx := context.Background()

	newUser := CreateTestUser(t, "evt_joiner2", "evt_joiner2@test.com", "pass")
	err := env.memberSvc.AddMemberToConversation(ctx, env.groupConv.ID.String(), env.owner.ID.String(), newUser.ID.String(), models.EnrollmentRoleMember)
	require.NoError(t, err)

	for _, e := range env.collector.Events() {
		assert.NotEqual(t, env.noMemberReadBot.ID, e.BotID,
			"bot without members:read should not receive notices")
	}
}

func TestEvent_MemberLeft(t *testing.T) {
	env := setupEventTestEnv(t)
	defer CleanupTestDB(t)
	ctx := context.Background()

	leaver := CreateTestUser(t, "evt_leaver", "evt_leaver@test.com", "pass")
	require.NoError(t, env.memberSvc.AddMemberToConversation(ctx, env.groupConv.ID.String(), env.owner.ID.String(), leaver.ID.String(), models.EnrollmentRoleMember))
	env.collector.Events()

	err := env.memberSvc.RemoveMemberFromConversation(ctx, env.groupConv.ID.String(), env.owner.ID.String(), leaver.ID.String())
	require.NoError(t, err)

	notices := env.collector.Filter(onebot.NoticeGroupMemberDecrease)
	require.Len(t, notices, 1)
	assert.Equal(t, env.bot.ID, notices[0].BotID)

	var data map[string]any
	require.NoError(t, json.Unmarshal(notices[0].Event.Data, &data))
	assert.Equal(t, leaver.ID.String(), data["user_id"])
}

func TestEvent_MemberRoleChanged(t *testing.T) {
	env := setupEventTestEnv(t)
	defer CleanupTestDB(t)
	ctx := context.Background()

	member := CreateTestUser(t, "evt_member", "evt_member@test.com", "pass")
	require.NoError(t, env.memberSvc.AddMemberToConversation(ctx, env.groupConv.ID.String(), env.owner.ID.String(), member.ID.String(), models.EnrollmentRoleMember))
	env.collector.Events()

	err := env.memberSvc.UpdateMemberRole(ctx, env.groupConv.ID.String(), env.owner.ID.String(), &models.UpdateMemberRoleRequest{
		UserID: member.ID,
		Role:   string(models.EnrollmentRoleAdmin),
	})
	require.NoError(t, err)

	notices := env.collector.Filter(onebot.NoticeGroupMemberRoleChanged)
	require.Len(t, notices, 1)
	assert.Equal(t, env.bot.ID, notices[0].BotID)

	var data map[string]any
	require.NoError(t, json.Unmarshal(notices[0].Event.Data, &data))
	assert.Equal(t, member.ID.String(), data["user_id"])
	assert.Equal(t, "admin", data["new_role"])
}

func TestEvent_Installation_Installed(t *testing.T) {
	env := setupEventTestEnv(t)
	defer CleanupTestDB(t)
	ctx := context.Background()

	newBot := &models.Bot{
		OwnerID:               env.owner.ID,
		Name:                  "InstallBot",
		Status:                models.BotStatusActive,
		BotType:               models.BotTypeExternal,
		Discoverability:       models.DiscoverabilityListed,
		RequestedCapabilities: []string{models.CapabilitySend},
	}
	require.NoError(t, env.botRepo.Create(ctx, newBot))

	inst, err := env.installSvc.CreateInstallation(ctx, env.owner.ID.String(), newBot.ID.String(), &models.CreateInstallationRequest{
		TargetType: models.InstallationTargetUser,
		TargetID:   env.owner.ID,
	})
	require.NoError(t, err)

	notices := env.collector.Filter(onebot.NoticeInstallationChanged)
	require.Len(t, notices, 1)
	assert.Equal(t, newBot.ID, notices[0].BotID)
	assert.Equal(t, onebot.SubTypeInstalled, notices[0].Event.SubType)

	var data map[string]any
	require.NoError(t, json.Unmarshal(notices[0].Event.Data, &data))
	assert.Equal(t, inst.ID.String(), data["installation_id"])
	assert.Equal(t, onebot.SubTypeInstalled, data["change_type"])
}

func TestEvent_Installation_Suspended(t *testing.T) {
	env := setupEventTestEnv(t)
	defer CleanupTestDB(t)
	ctx := context.Background()

	newBot := &models.Bot{
		OwnerID:               env.owner.ID,
		Name:                  "SuspendBot",
		Status:                models.BotStatusActive,
		BotType:               models.BotTypeExternal,
		Discoverability:       models.DiscoverabilityListed,
		RequestedCapabilities: []string{models.CapabilitySend},
	}
	require.NoError(t, env.botRepo.Create(ctx, newBot))

	inst, err := env.installSvc.CreateInstallation(ctx, env.owner.ID.String(), newBot.ID.String(), &models.CreateInstallationRequest{
		TargetType: models.InstallationTargetUser, TargetID: env.owner.ID,
	})
	require.NoError(t, err)
	env.collector.Clear()

	_, err = env.installSvc.UpdateInstallation(ctx, env.owner.ID.String(), inst.ID.String(), &models.UpdateInstallationRequest{
		Status: models.InstallationPaused,
	})
	require.NoError(t, err)

	notices := env.collector.Filter(onebot.NoticeInstallationChanged)
	require.Len(t, notices, 1)
	assert.Equal(t, onebot.SubTypeSuspended, notices[0].Event.SubType)
}

func TestEvent_Installation_Resumed(t *testing.T) {
	env := setupEventTestEnv(t)
	defer CleanupTestDB(t)
	ctx := context.Background()

	newBot := &models.Bot{
		OwnerID:               env.owner.ID,
		Name:                  "ResumeBot",
		Status:                models.BotStatusActive,
		BotType:               models.BotTypeExternal,
		Discoverability:       models.DiscoverabilityListed,
		RequestedCapabilities: []string{models.CapabilitySend},
	}
	require.NoError(t, env.botRepo.Create(ctx, newBot))

	inst, err := env.installSvc.CreateInstallation(ctx, env.owner.ID.String(), newBot.ID.String(), &models.CreateInstallationRequest{
		TargetType: models.InstallationTargetUser, TargetID: env.owner.ID,
	})
	require.NoError(t, err)

	_, err = env.installSvc.UpdateInstallation(ctx, env.owner.ID.String(), inst.ID.String(), &models.UpdateInstallationRequest{
		Status: models.InstallationPaused,
	})
	require.NoError(t, err)
	env.collector.Clear()

	_, err = env.installSvc.UpdateInstallation(ctx, env.owner.ID.String(), inst.ID.String(), &models.UpdateInstallationRequest{
		Status: models.InstallationActive,
	})
	require.NoError(t, err)

	notices := env.collector.Filter(onebot.NoticeInstallationChanged)
	require.Len(t, notices, 1)
	assert.Equal(t, onebot.SubTypeResumed, notices[0].Event.SubType)
}

func TestEvent_Installation_Uninstalled(t *testing.T) {
	env := setupEventTestEnv(t)
	defer CleanupTestDB(t)
	ctx := context.Background()

	newBot := &models.Bot{
		OwnerID:               env.owner.ID,
		Name:                  "UninstallBot",
		Status:                models.BotStatusActive,
		BotType:               models.BotTypeExternal,
		Discoverability:       models.DiscoverabilityListed,
		RequestedCapabilities: []string{models.CapabilitySend},
	}
	require.NoError(t, env.botRepo.Create(ctx, newBot))

	inst, err := env.installSvc.CreateInstallation(ctx, env.owner.ID.String(), newBot.ID.String(), &models.CreateInstallationRequest{
		TargetType: models.InstallationTargetUser, TargetID: env.owner.ID,
	})
	require.NoError(t, err)
	env.collector.Clear()

	err = env.installSvc.UninstallInstallation(ctx, env.owner.ID.String(), inst.ID.String())
	require.NoError(t, err)

	notices := env.collector.Filter(onebot.NoticeInstallationChanged)
	require.Len(t, notices, 1)
	assert.Equal(t, onebot.SubTypeUninstalled, notices[0].Event.SubType)
}

func TestEvent_Installation_CapabilityChanged(t *testing.T) {
	env := setupEventTestEnv(t)
	defer CleanupTestDB(t)
	ctx := context.Background()

	newBot := &models.Bot{
		OwnerID:               env.owner.ID,
		Name:                  "CapBot",
		Status:                models.BotStatusActive,
		BotType:               models.BotTypeExternal,
		Discoverability:       models.DiscoverabilityListed,
		RequestedCapabilities: []string{models.CapabilitySend, models.CapabilityMembersRead},
	}
	require.NoError(t, env.botRepo.Create(ctx, newBot))

	inst, err := env.installSvc.CreateInstallation(ctx, env.owner.ID.String(), newBot.ID.String(), &models.CreateInstallationRequest{
		TargetType: models.InstallationTargetUser, TargetID: env.owner.ID,
		GrantedCapabilities: []string{models.CapabilitySend, models.CapabilityMembersRead},
	})
	require.NoError(t, err)
	env.collector.Clear()

	_, err = env.installSvc.UpdateInstallation(ctx, env.owner.ID.String(), inst.ID.String(), &models.UpdateInstallationRequest{
		GrantedCapabilities: []string{models.CapabilitySend},
	})
	require.NoError(t, err)

	notices := env.collector.Filter(onebot.NoticeInstallationChanged)
	require.Len(t, notices, 1)
	assert.Equal(t, onebot.SubTypeCapabilityChanged, notices[0].Event.SubType)
}

func TestEvent_MessageEventPushed(t *testing.T) {
	env := setupEventTestEnv(t)
	defer CleanupTestDB(t)
	ctx := context.Background()

	userRepo := repository.NewUserRepository()
	convRepo := repository.NewConversationRepository()
	enrollRepo := repository.NewEnrollmentRepository()
	msgRepo := repository.NewConversationMessageRepository()
	pub := messaging.NewPublisher(5 * time.Second)
	pub.RegisterSink("external_bot", env.externalSink)
	ms := services.NewMessageService(userRepo, convRepo, enrollRepo, msgRepo, env.botRepo, env.instRepo, pub)

	_, err := ms.SendMessage(ctx, env.owner.ID.String(), &models.SendMessageRequest{
		ConversationID: env.groupConv.ID,
		Content:        "hello event push",
		MsgType:        string(models.MsgTypeText),
	})
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	var msgEvents []capturedEvent
	for _, e := range env.collector.Events() {
		if e.Event.PostType == onebot.PostTypeMessage {
			msgEvents = append(msgEvents, e)
		}
	}
	assert.GreaterOrEqual(t, len(msgEvents), 1, "bots with read_trigger should receive message events")

	botIDs := make(map[uuid.UUID]bool)
	for _, e := range msgEvents {
		botIDs[e.BotID] = true
		var data map[string]any
		require.NoError(t, json.Unmarshal(e.Event.Data, &data))
		assert.NotEmpty(t, data["message_id"])
		assert.NotEmpty(t, data["conversation_id"])
		assert.Equal(t, "user", data["source"])
	}
	assert.True(t, botIDs[env.bot.ID], "bot with members:read+read_trigger should receive event")
}

func TestEvent_MessageEvent_BotSentNotPushed(t *testing.T) {
	SetupTestDB(t)
	defer CleanupTestDB(t)
	ctx := context.Background()

	userRepo := repository.NewUserRepository()
	convRepo := repository.NewConversationRepository()
	enrollRepo := repository.NewEnrollmentRepository()
	msgRepo := repository.NewConversationMessageRepository()
	botRepo := repository.NewBotRepository()
	instRepo := repository.NewBotInstallationRepository()
	collector := newEventCollector()

	pub := messaging.NewPublisher(5 * time.Second)
	sink := services.NewExternalBotSink(instRepo, botRepo, collector)
	pub.RegisterSink("external_bot", sink)
	ms := services.NewMessageService(userRepo, convRepo, enrollRepo, msgRepo, botRepo, instRepo, pub)

	owner := CreateTestUser(t, "evt_bs_owner", "evt_bs_owner@test.com", "pass")

	bot1 := &models.Bot{
		OwnerID: owner.ID, Name: "Bot1", Status: models.BotStatusActive,
		BotType: models.BotTypeExternal, Discoverability: models.DiscoverabilityUnlisted,
		RequestedCapabilities: []string{models.CapabilityReadTrigger, models.CapabilitySend},
	}
	require.NoError(t, botRepo.Create(ctx, bot1))

	bot2 := &models.Bot{
		OwnerID: owner.ID, Name: "Bot2", Status: models.BotStatusActive,
		BotType: models.BotTypeExternal, Discoverability: models.DiscoverabilityUnlisted,
		RequestedCapabilities: []string{models.CapabilityReadTrigger, models.CapabilitySend},
	}
	require.NoError(t, botRepo.Create(ctx, bot2))

	groupConv := &models.Conversation{ConversationType: models.ConversationTypeGroup, Name: "Bot Msg Group"}
	groupConv.CreatedBy = &owner.ID
	require.NoError(t, convRepo.Create(ctx, groupConv))
	for _, uid := range []uuid.UUID{owner.ID, bot1.ID, bot2.ID} {
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
		AppID: bot1.ID, InstalledBy: owner.ID,
		TargetType: models.InstallationTargetConversation, TargetID: groupConv.ID,
		GrantedCapabilities: []string{models.CapabilityReadTrigger, models.CapabilitySend},
		Status:              models.InstallationActive,
	}))
	require.NoError(t, instRepo.Create(ctx, &models.BotInstallation{
		AppID: bot2.ID, InstalledBy: owner.ID,
		TargetType: models.InstallationTargetConversation, TargetID: groupConv.ID,
		GrantedCapabilities: []string{models.CapabilityReadTrigger, models.CapabilitySend},
		Status:              models.InstallationActive,
	}))

	_, err := ms.SendBotMessage(ctx, &messaging.BotSendRequest{
		BotID:          bot1.ID,
		ConversationID: groupConv.ID,
		Content:        "bot1 speaking",
		MsgType:        string(models.MsgTypeText),
		Source:         messaging.SourceExternal,
	})
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	for _, e := range collector.Events() {
		assert.NotEqual(t, bot2.ID, e.BotID,
			"bot2 should NOT receive events from bot1's message (anti-loop)")
	}
}

func TestEvent_MultiConnectionBroadcast(t *testing.T) {
	SetupTestDB(t)
	defer CleanupTestDB(t)
	ctx := context.Background()

	convRepo := repository.NewConversationRepository()
	enrollRepo := repository.NewEnrollmentRepository()
	msgRepo := repository.NewConversationMessageRepository()
	botRepo := repository.NewBotRepository()
	instRepo := repository.NewBotInstallationRepository()

	dispatcher := botws.RegistryDispatcher{}
	manager := botws.NewManager(botws.DefaultConfig(), dispatcher)

	emitter := services.NewBotNoticeEmitter(instRepo, botRepo, manager)

	owner := CreateTestUser(t, "evt_mc_owner", "evt_mc_owner@test.com", "pass")

	bot := &models.Bot{
		OwnerID: owner.ID, Name: "MultiConnBot", Status: models.BotStatusActive,
		BotType: models.BotTypeExternal, Discoverability: models.DiscoverabilityUnlisted,
		RequestedCapabilities: []string{models.CapabilityMembersRead, models.CapabilityReadTrigger},
	}
	require.NoError(t, botRepo.Create(ctx, bot))

	groupConv := &models.Conversation{ConversationType: models.ConversationTypeGroup, Name: "MC Group"}
	groupConv.CreatedBy = &owner.ID
	require.NoError(t, convRepo.Create(ctx, groupConv))
	require.NoError(t, enrollRepo.Create(ctx, &models.Enrollment{
		ConversationID: groupConv.ID, UserID: owner.ID, Role: models.EnrollmentRoleOwner,
	}))
	require.NoError(t, enrollRepo.Create(ctx, &models.Enrollment{
		ConversationID: groupConv.ID, UserID: bot.ID, Role: models.EnrollmentRoleMember,
	}))
	require.NoError(t, msgRepo.CreateMessageTable(ctx, groupConv.ID))
	require.NoError(t, instRepo.Create(ctx, &models.BotInstallation{
		AppID: bot.ID, InstalledBy: owner.ID,
		TargetType: models.InstallationTargetConversation, TargetID: groupConv.ID,
		GrantedCapabilities: []string{models.CapabilityMembersRead, models.CapabilityReadTrigger},
		Status:              models.InstallationActive,
	}))

	assert.Equal(t, int64(0), manager.Metrics().Active)

	emitter.NotifyMemberJoined(ctx, groupConv.ID, owner.ID, "member")

	assert.Equal(t, uint64(0), manager.Metrics().MessagesWritten,
		"no connections = no delivery")
}
