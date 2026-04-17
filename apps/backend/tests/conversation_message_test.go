package tests

import (
	"context"
	"testing"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConversationMessageRepository 测试会话消息仓储
func TestConversationMessageRepository(t *testing.T) {
	SetupTestDB(t)
	defer CleanupTestDB(t)

	ctx := context.Background()
	conversationRepo := repository.NewConversationRepository()
	enrollmentRepo := repository.NewEnrollmentRepository()
	messageRepo := repository.NewConversationMessageRepository()

	// 创建测试用户
	user1 := CreateTestUser(t, "msg_user1", "msg1@test.com", "pass123")
	user2 := CreateTestUser(t, "msg_user2", "msg2@test.com", "pass123")
	user3 := CreateTestUser(t, "msg_user3", "msg3@test.com", "pass123")

	// 创建会话
	conversation := &models.Conversation{
		ConversationType: models.ConversationTypeGroup,
		Name:             "Test Group",
		CreatedBy:        &user1.ID,
	}
	err := conversationRepo.Create(ctx, conversation)
	require.NoError(t, err)

	// 添加成员
	err = enrollmentRepo.Create(ctx, &models.Enrollment{
		ConversationID: conversation.ID,
		UserID:         user1.ID,
		Role:           models.EnrollmentRoleOwner,
	})
	require.NoError(t, err)
	err = enrollmentRepo.Create(ctx, &models.Enrollment{
		ConversationID: conversation.ID,
		UserID:         user2.ID,
		Role:           models.EnrollmentRoleMember,
	})
	require.NoError(t, err)
	err = enrollmentRepo.Create(ctx, &models.Enrollment{
		ConversationID: conversation.ID,
		UserID:         user3.ID,
		Role:           models.EnrollmentRoleMember,
	})
	require.NoError(t, err)

	// 创建消息表
	err = messageRepo.CreateMessageTable(ctx, conversation.ID)
	require.NoError(t, err)

	// 重复创建不应报错
	err = messageRepo.CreateMessageTable(ctx, conversation.ID)
	assert.NoError(t, err)

	t.Run("InsertMessage", func(t *testing.T) {
		msg := &models.Message{
			SenderID: user1.ID,
			Content:  "Hello World",
			MsgType:  models.MsgTypeText,
		}

		err := messageRepo.InsertMessage(ctx, conversation.ID, msg)
		require.NoError(t, err)

		assert.NotEmpty(t, msg.ID)
		assert.Equal(t, conversation.ID, msg.ConversationID)
		assert.False(t, msg.CreatedAt.IsZero())
	})

	// 插入更多消息
	for i := 0; i < 5; i++ {
		msg := &models.Message{
			SenderID: user2.ID,
			Content:  "Message " + string(rune('A'+i)),
			MsgType:  models.MsgTypeText,
		}
		err := messageRepo.InsertMessage(ctx, conversation.ID, msg)
		require.NoError(t, err)
	}

	t.Run("FindMessages DESC order", func(t *testing.T) {
		messages, err := messageRepo.FindMessages(ctx, conversation.ID, 50, 0)
		require.NoError(t, err)
		// 包括最初的 Hello World
		assert.Len(t, messages, 6)

		// 验证 DESC 排序（最新的在前）
		assert.True(t, messages[0].CreatedAt.After(messages[1].CreatedAt) ||
			messages[0].CreatedAt.Equal(messages[1].CreatedAt))
	})

	t.Run("FindMessages pagination", func(t *testing.T) {
		// 第一页
		page1, err := messageRepo.FindMessages(ctx, conversation.ID, 2, 0)
		require.NoError(t, err)
		assert.Len(t, page1, 2)

		// 第二页
		page2, err := messageRepo.FindMessages(ctx, conversation.ID, 2, 2)
		require.NoError(t, err)
		assert.Len(t, page2, 2)

		// 第三页
		page3, err := messageRepo.FindMessages(ctx, conversation.ID, 2, 4)
		require.NoError(t, err)
		assert.Len(t, page3, 2)

		// 不应有重叠（通过 ID）
		p1Ids := map[string]bool{}
		for _, m := range page1 {
			p1Ids[m.ID.String()] = true
		}
		for _, m := range page2 {
			assert.False(t, p1Ids[m.ID.String()], "Page 2 should not overlap with page 1")
		}
	})

	t.Run("FindMessages default limit", func(t *testing.T) {
		// limit <= 0 应该使用默认值 50
		messages, err := messageRepo.FindMessages(ctx, conversation.ID, 0, 0)
		require.NoError(t, err)
		assert.Len(t, messages, 6)
	})

	t.Run("FindAllMessages ASC order", func(t *testing.T) {
		messages, err := messageRepo.FindAllMessages(ctx, conversation.ID)
		require.NoError(t, err)
		assert.Len(t, messages, 6)

		// 验证 ASC 排序（最早的在前）
		assert.True(t, messages[0].CreatedAt.Before(messages[len(messages)-1].CreatedAt) ||
			messages[0].CreatedAt.Equal(messages[len(messages)-1].CreatedAt))
	})

	t.Run("CountMessages", func(t *testing.T) {
		count, err := messageRepo.CountMessages(ctx, conversation.ID)
		require.NoError(t, err)
		assert.Equal(t, 6, count)
	})

	t.Run("FindLastMessage", func(t *testing.T) {
		lastMsg, err := messageRepo.FindLastMessage(ctx, conversation.ID)
		require.NoError(t, err)
		assert.NotNil(t, lastMsg)
		assert.Equal(t, conversation.ID, lastMsg.ConversationID)
		// 最后一条消息应该是 Message E
		assert.Equal(t, "Message E", lastMsg.Content)
	})

	t.Run("GetConversationParticipants", func(t *testing.T) {
		enrollments, err := enrollmentRepo.FindByConversationID(ctx, conversation.ID)
		require.NoError(t, err)
		assert.Len(t, enrollments, 3)

		participantIds := map[string]bool{}
		for _, e := range enrollments {
			participantIds[e.UserID.String()] = true
		}
		assert.True(t, participantIds[user1.ID.String()])
		assert.True(t, participantIds[user2.ID.String()])
		assert.True(t, participantIds[user3.ID.String()])
	})

	t.Run("BroadcastMessage - exclude sender", func(t *testing.T) {
		// 通过 enrollmentRepo 模拟广播逻辑：查找会话中非发送者的成员
		enrollments, err := enrollmentRepo.FindByConversationID(ctx, conversation.ID)
		require.NoError(t, err)

		var recipients []string
		for _, e := range enrollments {
			if e.UserID != user1.ID {
				recipients = append(recipients, e.UserID.String())
			}
		}
		assert.Len(t, recipients, 2) // 排除发送者
	})

	t.Run("DropMessageTable", func(t *testing.T) {
		err := messageRepo.DropMessageTable(ctx, conversation.ID)
		require.NoError(t, err)

		// 删除后查询应返回空
		messages, err := messageRepo.FindAllMessages(ctx, conversation.ID)
		require.NoError(t, err)
		assert.Len(t, messages, 0)
	})
}
