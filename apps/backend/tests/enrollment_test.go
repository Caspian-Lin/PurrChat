package tests

import (
	"context"
	"testing"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnrollmentCRUD 测试会话成员 CRUD 操作
func TestEnrollmentCRUD(t *testing.T) {
	SetupTestDB(t)
	defer CleanupTestDB(t)

	ctx := context.Background()
	enrollmentRepo := repository.NewEnrollmentRepository()
	conversationRepo := repository.NewConversationRepository()

	// 创建测试用户
	user1 := CreateTestUser(t, "enroll_u1", "enroll1@test.com", "pass123")
	user2 := CreateTestUser(t, "enroll_u2", "enroll2@test.com", "pass123")

	// 创建会话
	conversation := &models.Conversation{
		ConversationType: models.ConversationTypeDirect,
		CreatedBy:        &user1.ID,
	}
	err := conversationRepo.Create(ctx, conversation)
	require.NoError(t, err)

	t.Run("CreateEnrollment", func(t *testing.T) {
		enrollment := &models.Enrollment{
			ConversationID: conversation.ID,
			UserID:         user1.ID,
			Role:           models.EnrollmentRoleOwner,
		}

		err := enrollmentRepo.Create(ctx, enrollment)
		require.NoError(t, err)

		// 验证自动生成 ID 和 JoinedAt
		assert.NotEmpty(t, enrollment.ID)
		assert.False(t, enrollment.JoinedAt.IsZero())
		assert.Equal(t, models.EnrollmentRoleOwner, enrollment.Role)
	})

	t.Run("UpsertOnConflict", func(t *testing.T) {
		// 同一个 conversation + user 再次创建应该更新而不是报错
		enrollment := &models.Enrollment{
			ConversationID: conversation.ID,
			UserID:         user1.ID,
			Role:           models.EnrollmentRoleAdmin,
		}

		err := enrollmentRepo.Create(ctx, enrollment)
		require.NoError(t, err)
		assert.Equal(t, models.EnrollmentRoleAdmin, enrollment.Role)
	})

	t.Run("CreateSecondEnrollment", func(t *testing.T) {
		enrollment := &models.Enrollment{
			ConversationID: conversation.ID,
			UserID:         user2.ID,
			Role:           models.EnrollmentRoleMember,
		}

		err := enrollmentRepo.Create(ctx, enrollment)
		require.NoError(t, err)
	})

	t.Run("FindByConversationID", func(t *testing.T) {
		enrollments, err := enrollmentRepo.FindByConversationID(ctx, conversation.ID)
		require.NoError(t, err)
		assert.Len(t, enrollments, 2)
		// 验证按 joined_at ASC 排序
		assert.True(t, enrollments[0].JoinedAt.Before(enrollments[1].JoinedAt) ||
			enrollments[0].JoinedAt.Equal(enrollments[1].JoinedAt))
	})

	t.Run("FindByUserID", func(t *testing.T) {
		enrollments, err := enrollmentRepo.FindByUserID(ctx, user1.ID)
		require.NoError(t, err)
		assert.Len(t, enrollments, 1)
		assert.Equal(t, conversation.ID, enrollments[0].ConversationID)
	})

	t.Run("FindByConversationAndUser", func(t *testing.T) {
		enrollment, err := enrollmentRepo.FindByConversationAndUser(ctx, conversation.ID, user1.ID)
		require.NoError(t, err)
		assert.NotNil(t, enrollment)
		assert.Equal(t, models.EnrollmentRoleAdmin, enrollment.Role)

		// 不存在的组合
		_, err = enrollmentRepo.FindByConversationAndUser(ctx, conversation.ID, user2.ID)
		require.NoError(t, err)
	})

	t.Run("UpdateEnrollment", func(t *testing.T) {
		enrollment, err := enrollmentRepo.FindByConversationAndUser(ctx, conversation.ID, user2.ID)
		require.NoError(t, err)

		enrollment.Role = models.EnrollmentRoleAdmin
		err = enrollmentRepo.Update(ctx, enrollment)
		require.NoError(t, err)

		updated, err := enrollmentRepo.FindByID(ctx, enrollment.ID)
		require.NoError(t, err)
		assert.Equal(t, models.EnrollmentRoleAdmin, updated.Role)
	})

	t.Run("UpdateLastReadAt", func(t *testing.T) {
		err := enrollmentRepo.UpdateLastReadAt(ctx, conversation.ID, user2.ID)
		require.NoError(t, err)

		enrollment, err := enrollmentRepo.FindByConversationAndUser(ctx, conversation.ID, user2.ID)
		require.NoError(t, err)
		assert.NotNil(t, enrollment.LastReadAt)
	})

	t.Run("DeleteEnrollment", func(t *testing.T) {
		enrollment, err := enrollmentRepo.FindByConversationAndUser(ctx, conversation.ID, user2.ID)
		require.NoError(t, err)

		err = enrollmentRepo.Delete(ctx, enrollment.ID)
		require.NoError(t, err)

		_, err = enrollmentRepo.FindByID(ctx, enrollment.ID)
		assert.Error(t, err)
	})

	t.Run("DeleteByConversationAndUser", func(t *testing.T) {
		// 重新添加 user2
		newEnrollment := &models.Enrollment{
			ConversationID: conversation.ID,
			UserID:         user2.ID,
			Role:           models.EnrollmentRoleMember,
		}
		err := enrollmentRepo.Create(ctx, newEnrollment)
		require.NoError(t, err)

		err = enrollmentRepo.DeleteByConversationAndUser(ctx, conversation.ID, user2.ID)
		require.NoError(t, err)

		enrollments, err := enrollmentRepo.FindByConversationID(ctx, conversation.ID)
		require.NoError(t, err)
		assert.Len(t, enrollments, 1)
	})
}
