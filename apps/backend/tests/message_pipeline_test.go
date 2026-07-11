package tests

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"purr-chat-server/internal/messaging"
	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"
	"purr-chat-server/internal/services"
	"purr-chat-server/pkg/database"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupMessageServiceTestEnv 创建消息服务测试环境
func setupMessageServiceTestEnv(t *testing.T) (*services.MessageService, repository.UserRepository, repository.ConversationRepository, repository.EnrollmentRepository, repository.ConversationMessageRepository, repository.BotRepository, repository.BotInstallationRepository, *messaging.Publisher) {
	t.Helper()
	SetupTestDB(t)

	userRepo := repository.NewUserRepository()
	convRepo := repository.NewConversationRepository()
	enrollRepo := repository.NewEnrollmentRepository()
	msgRepo := repository.NewConversationMessageRepository()
	botRepo := repository.NewBotRepository()
	instRepo := repository.NewBotInstallationRepository()

	pub := messaging.NewPublisher(5 * time.Second)
	ms := services.NewMessageService(userRepo, convRepo, enrollRepo, msgRepo, botRepo, instRepo, pub)

	return ms, userRepo, convRepo, enrollRepo, msgRepo, botRepo, instRepo, pub
}

// createTestBotAndConv 创建测试 Bot 和 direct conversation
func createTestBotAndConv(t *testing.T, ctx context.Context, botRepo repository.BotRepository, convRepo repository.ConversationRepository, enrollRepo repository.EnrollmentRepository, msgRepo repository.ConversationMessageRepository, instRepo repository.BotInstallationRepository) (*models.Bot, *models.Conversation, *models.User) {
	t.Helper()

	owner := CreateTestUser(t, "msg_owner", "msg_owner@test.com", "pass")

	bot := &models.Bot{
		OwnerID: owner.ID,
		Name:    "MsgTestBot",
		Status:  models.BotStatusActive,
		BotType: models.BotTypeWorkflow,
	}
	require.NoError(t, botRepo.Create(ctx, bot))

	conv := &models.Conversation{
		ConversationType: models.ConversationTypeDirect,
	}
	createdBy := owner.ID
	conv.CreatedBy = &createdBy
	require.NoError(t, convRepo.Create(ctx, conv))

	require.NoError(t, enrollRepo.Create(ctx, &models.Enrollment{
		ConversationID: conv.ID,
		UserID:         owner.ID,
		Role:           models.EnrollmentRoleOwner,
	}))
	require.NoError(t, enrollRepo.Create(ctx, &models.Enrollment{
		ConversationID: conv.ID,
		UserID:         bot.ID,
		Role:           models.EnrollmentRoleMember,
	}))
	require.NoError(t, msgRepo.CreateMessageTable(ctx, conv.ID))

	require.NoError(t, instRepo.Create(ctx, &models.BotInstallation{
		AppID:               bot.ID,
		InstalledBy:         owner.ID,
		TargetType:          models.InstallationTargetUser,
		TargetID:            owner.ID,
		GrantedCapabilities: []string{models.CapabilitySend, models.CapabilityReadTrigger},
		Status:              models.InstallationActive,
	}))

	return bot, conv, owner
}

// TestSendMessage_UserPath 验证用户发送消息的完整流程
func TestSendMessage_UserPath(t *testing.T) {
	ms, _, convRepo, enrollRepo, msgRepo, botRepo, instRepo, _ := setupMessageServiceTestEnv(t)
	defer CleanupTestDB(t)

	ctx := context.Background()
	bot, conv, owner := createTestBotAndConv(t, ctx, botRepo, convRepo, enrollRepo, msgRepo, instRepo)
	_ = bot

	msg, err := ms.SendMessage(ctx, owner.ID.String(), &models.SendMessageRequest{
		ConversationID:  conv.ID,
		Content:         "hello world",
		MsgType:         "text",
		ClientMessageID: "client-1",
	})
	require.NoError(t, err)
	assert.Equal(t, "hello world", msg.Content)
	assert.Equal(t, models.MsgTypeText, msg.MsgType)
	assert.Equal(t, owner.ID, msg.SenderID)

	// 验证消息已持久化
	msgs, err := msgRepo.FindMessages(ctx, conv.ID, 10, 0)
	require.NoError(t, err)
	assert.Len(t, msgs, 1)
}

// TestSendMessage_Idempotent 验证 client_message_id 幂等性
func TestSendMessage_Idempotent(t *testing.T) {
	ms, _, convRepo, enrollRepo, msgRepo, botRepo, instRepo, _ := setupMessageServiceTestEnv(t)
	defer CleanupTestDB(t)

	ctx := context.Background()
	bot, conv, owner := createTestBotAndConv(t, ctx, botRepo, convRepo, enrollRepo, msgRepo, instRepo)
	_ = bot

	req := &models.SendMessageRequest{
		ConversationID:  conv.ID,
		Content:         "duplicate test",
		MsgType:         "text",
		ClientMessageID: "dup-1",
	}

	msg1, err := ms.SendMessage(ctx, owner.ID.String(), req)
	require.NoError(t, err)

	msg2, err := ms.SendMessage(ctx, owner.ID.String(), req)
	require.NoError(t, err)

	assert.Equal(t, msg1.ID, msg2.ID, "duplicate client_message_id should return same message")

	msgs, err := msgRepo.FindMessages(ctx, conv.ID, 10, 0)
	require.NoError(t, err)
	assert.Len(t, msgs, 1, "should only have one message")
}

// TestSendBotMessage_WorkflowReply 验证 Bot 通过 workflow 回复
func TestSendBotMessage_WorkflowReply(t *testing.T) {
	ms, _, convRepo, enrollRepo, msgRepo, botRepo, instRepo, _ := setupMessageServiceTestEnv(t)
	defer CleanupTestDB(t)

	ctx := context.Background()
	bot, conv, _ := createTestBotAndConv(t, ctx, botRepo, convRepo, enrollRepo, msgRepo, instRepo)

	msg, err := ms.SendBotMessage(ctx, &messaging.BotSendRequest{
		BotID:          bot.ID,
		ConversationID: conv.ID,
		Content:        "bot reply",
		MsgType:        "text",
		Source:         messaging.SourceWorkflow,
		RunID:          "run-123",
	})
	require.NoError(t, err)
	assert.Equal(t, "bot reply", msg.Content)
	assert.Equal(t, bot.ID, msg.SenderID)
	require.NotNil(t, msg.BotID)
	assert.Equal(t, bot.ID, *msg.BotID)

	// 验证消息已持久化
	msgs, err := msgRepo.FindMessages(ctx, conv.ID, 10, 0)
	require.NoError(t, err)
	assert.Len(t, msgs, 1)
	assert.Equal(t, "bot reply", msgs[0].Content)
}

// TestSendBotMessage_InactiveBot 验证非 active Bot 不能发送
func TestSendBotMessage_InactiveBot(t *testing.T) {
	ms, _, convRepo, enrollRepo, msgRepo, botRepo, instRepo, _ := setupMessageServiceTestEnv(t)
	defer CleanupTestDB(t)

	ctx := context.Background()
	bot, conv, _ := createTestBotAndConv(t, ctx, botRepo, convRepo, enrollRepo, msgRepo, instRepo)

	// 禁用 Bot
	_, err := database.GetPool().Exec(ctx, "UPDATE bots SET status = 'disabled' WHERE id = $1", bot.ID)
	require.NoError(t, err)

	_, err = ms.SendBotMessage(ctx, &messaging.BotSendRequest{
		BotID:          bot.ID,
		ConversationID: conv.ID,
		Content:        "should fail",
		MsgType:        "text",
		Source:         messaging.SourceExternal,
	})
	assert.Error(t, err)
}

// TestSendBotMessage_NoSendCapability 验证无 messages:send capability 不能发送
func TestSendBotMessage_NoSendCapability(t *testing.T) {
	ms, _, convRepo, enrollRepo, msgRepo, botRepo, instRepo, _ := setupMessageServiceTestEnv(t)
	defer CleanupTestDB(t)

	ctx := context.Background()
	bot, conv, owner := createTestBotAndConv(t, ctx, botRepo, convRepo, enrollRepo, msgRepo, instRepo)

	// 删除有 send capability 的 installation，创建没有 send 的
	instRepo2 := repository.NewBotInstallationRepository()
	_ = instRepo2
	// 直接更新 installation 去掉 messages:send
	_, err := database.GetPool().Exec(ctx,
		"UPDATE bot_installations SET granted_capabilities = $1 WHERE app_id = $2",
		[]string{models.CapabilityReadTrigger}, bot.ID)
	require.NoError(t, err)

	_, err = ms.SendBotMessage(ctx, &messaging.BotSendRequest{
		BotID:          bot.ID,
		ConversationID: conv.ID,
		Content:        "should fail",
		MsgType:        "text",
		Source:         messaging.SourceExternal,
	})
	assert.Error(t, err)
	_ = owner
}

// TestAntiLoop_BotMessageDoesNotTriggerBots 验证 Bot 发送的消息不触发其他 Bot
func TestAntiLoop_BotMessageDoesNotTriggerBots(t *testing.T) {
	pub := messaging.NewPublisher(5 * time.Second)

	// 创建一个 fake workflow sink 来跟踪是否被调用
	workflowSink := &fakeWorkflowSink{}
	pub.RegisterSink("workflow", workflowSink)

	// 用户消息事件 → 应该触发 workflow
	userEvent := &messaging.MessageCreatedEvent{
		Message:   &models.Message{MsgType: models.MsgTypeText, Content: "hello"},
		ActorType: messaging.ActorUser,
		Source:    messaging.SourceUser,
	}
	pub.Publish(context.Background(), userEvent)
	assert.Equal(t, int64(1), workflowSink.invoked.Load(), "user message should trigger workflow sink")

	// Bot 消息事件 → 不应该触发 workflow
	botEvent := &messaging.MessageCreatedEvent{
		Message:   &models.Message{MsgType: models.MsgTypeText, Content: "bot reply"},
		ActorType: messaging.ActorBot,
		Source:    messaging.SourceWorkflow,
	}
	pub.Publish(context.Background(), botEvent)
	assert.Equal(t, int64(1), workflowSink.invoked.Load(), "bot message should NOT trigger workflow sink")
}

// fakeWorkflowSink 用于测试防回复环
type fakeWorkflowSink struct {
	invoked atomic.Int64
}

func (f *fakeWorkflowSink) OnMessageCreated(ctx context.Context, event *messaging.MessageCreatedEvent) error {
	if !event.ShouldTriggerBots() {
		return nil
	}
	f.invoked.Add(1)
	return nil
}

type fakeBotEventPublisher struct {
	delivered atomic.Int64
}

func (f *fakeBotEventPublisher) PublishBotEvent(_ uuid.UUID, _ any) int {
	f.delivered.Add(1)
	return 1
}

func TestExternalBotSink_DirectConversationUsesUserInstallation(t *testing.T) {
	ms, _, convRepo, enrollRepo, msgRepo, botRepo, instRepo, pub := setupMessageServiceTestEnv(t)
	defer CleanupTestDB(t)

	ctx := context.Background()
	bot, conv, owner := createTestBotAndConv(t, ctx, botRepo, convRepo, enrollRepo, msgRepo, instRepo)
	botEvents := &fakeBotEventPublisher{}
	pub.RegisterSink("external_bot", services.NewExternalBotSink(instRepo, botRepo, botEvents))

	_, err := ms.SendMessage(ctx, owner.ID.String(), &models.SendMessageRequest{
		ConversationID: conv.ID,
		Content:        "trigger external bot",
		MsgType:        "text",
	})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		return botEvents.delivered.Load() == 1
	}, time.Second, 10*time.Millisecond)
	_ = bot
}

// TestPublisher_Metrics 验证 publisher 指标
func TestPublisher_Metrics(t *testing.T) {
	pub := messaging.NewPublisher(5 * time.Second)
	sink := &fakeWorkflowSink{}
	pub.RegisterSink("test", sink)

	pub.Publish(context.Background(), &messaging.MessageCreatedEvent{
		Message:   &models.Message{MsgType: models.MsgTypeText},
		ActorType: messaging.ActorUser,
	})

	m := pub.Metrics()
	assert.Equal(t, int64(1), m["test"].Invoked)
	assert.Equal(t, int64(1), m["test"].Succeeded)
}

// TestSendBotMessage_DirectConvValidation 验证 direct conversation 成员校验
func TestSendBotMessage_DirectConvValidation(t *testing.T) {
	ms, _, convRepo, enrollRepo, msgRepo, botRepo, instRepo, _ := setupMessageServiceTestEnv(t)
	defer CleanupTestDB(t)

	ctx := context.Background()
	bot, conv, owner := createTestBotAndConv(t, ctx, botRepo, convRepo, enrollRepo, msgRepo, instRepo)

	// 正常情况：2 成员（Bot + 用户）
	_, err := ms.SendBotMessage(ctx, &messaging.BotSendRequest{
		BotID:          bot.ID,
		ConversationID: conv.ID,
		Content:        "ok",
		MsgType:        "text",
		Source:         messaging.SourceExternal,
	})
	assert.NoError(t, err)

	// 异常情况：添加第三个成员
	extraUser := CreateTestUser(t, "extra_user", "extra@test.com", "pass")
	require.NoError(t, enrollRepo.Create(ctx, &models.Enrollment{
		ConversationID: conv.ID,
		UserID:         extraUser.ID,
		Role:           models.EnrollmentRoleMember,
	}))

	_, err = ms.SendBotMessage(ctx, &messaging.BotSendRequest{
		BotID:          bot.ID,
		ConversationID: conv.ID,
		Content:        "should fail",
		MsgType:        "text",
		Source:         messaging.SourceExternal,
	})
	assert.Error(t, err, "direct conversation with 3 members should fail")
	_ = owner
}

// TestSendBotMessage_ExternalSource 验证 external Bot 发送消息
func TestSendBotMessage_ExternalSource(t *testing.T) {
	ms, _, convRepo, enrollRepo, msgRepo, botRepo, instRepo, _ := setupMessageServiceTestEnv(t)
	defer CleanupTestDB(t)

	ctx := context.Background()
	bot, conv, _ := createTestBotAndConv(t, ctx, botRepo, convRepo, enrollRepo, msgRepo, instRepo)

	msg, err := ms.SendBotMessage(ctx, &messaging.BotSendRequest{
		BotID:          bot.ID,
		ConversationID: conv.ID,
		Content:        "<script>alert(1)</script>",
		MsgType:        "text",
		Source:         messaging.SourceExternal,
	})
	require.NoError(t, err)
	// 验证 HTML 转义
	assert.Contains(t, msg.Content, "&lt;script&gt;")
	assert.NotContains(t, msg.Content, "<script>")
}
