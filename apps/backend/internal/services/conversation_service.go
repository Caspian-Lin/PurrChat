package services

import (
	"context"
	"time"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"
	"purr-chat-server/internal/websocket"
	"purr-chat-server/pkg/database"
	"purr-chat-server/pkg/logger"

	"github.com/google/uuid"
)

// ConversationService 会话服务
type ConversationService struct {
	userRepo                repository.UserRepository
	conversationRepo        repository.ConversationRepository
	enrollmentRepo          repository.EnrollmentRepository
	conversationMessageRepo repository.ConversationMessageRepository
	friendshipRepo          repository.FriendshipRepository
	botRepo                 repository.BotRepository
}

// NewConversationService 创建会话服务
func NewConversationService(
	userRepo repository.UserRepository,
	conversationRepo repository.ConversationRepository,
	enrollmentRepo repository.EnrollmentRepository,
	conversationMessageRepo repository.ConversationMessageRepository,
	friendshipRepo repository.FriendshipRepository,
) *ConversationService {
	return &ConversationService{
		userRepo:                userRepo,
		conversationRepo:        conversationRepo,
		enrollmentRepo:          enrollmentRepo,
		conversationMessageRepo: conversationMessageRepo,
		friendshipRepo:          friendshipRepo,
	}
}

// SetBotRepo 设置 Bot 仓储（可选依赖）
func (s *ConversationService) SetBotRepo(botRepo repository.BotRepository) {
	s.botRepo = botRepo
}

// GetConversations 获取用户的所有会话
func (s *ConversationService) GetConversations(ctx context.Context, userID string) ([]*models.Conversation, error) {
	logger.InfofWithCaller("Getting conversations for user: %s", userID)

	id, err := uuid.Parse(userID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to parse user ID %s: %v", userID, err)
		return nil, err
	}

	conversations, err := s.conversationRepo.FindByUserID(ctx, id)
	if err != nil {
		logger.ErrorfWithCaller("Failed to get conversations for user %s: %v", userID, err)
		return nil, err
	}

	// 为每个会话加载成员信息和最后一条消息
	for _, conv := range conversations {
		// 加载成员信息
		members, err := s.enrollmentRepo.FindByConversationID(ctx, conv.ID)
		if err == nil {
			// 为每个成员加载用户信息
			for _, member := range members {
				user, err := s.userRepo.FindByID(ctx, member.UserID)
				if err == nil {
					// 验证返回的用户ID是否与enrollment中的user_id一致
					if user.ID == member.UserID {
						sanitizeUser(user)
						member.User = user
					} else {
						logger.ErrorfWithCaller("User ID mismatch: enrollment user_id=%s, loaded user id=%s", member.UserID, user.ID)
					}
				} else {
					logger.ErrorfWithCaller("Failed to load user for enrollment user_id=%s: %v", member.UserID, err)
				}
			}
			conv.Members = members
		}

		// 为私聊会话设置名称（如果还没有）
		var otherUserID uuid.UUID
		if conv.ConversationType == models.ConversationTypeDirect && conv.Name == "" {
			// 找到另一个用户，使用已加载的member.User信息
			for _, member := range members {
				if member.UserID != id {
					otherUserID = member.UserID
					// 使用已加载的用户信息，避免重复查询和可能的数据不一致
					if member.User != nil {
						conv.Name = member.User.Username
					} else {
						// 如果member.User为空，则查询用户信息
						user, err := s.userRepo.FindByID(ctx, member.UserID)
						if err == nil {
							conv.Name = user.Username
						}
					}
					break
				}
			}
		}

		// 为私聊会话加载好友关系状态
		if conv.ConversationType == models.ConversationTypeDirect && otherUserID != uuid.Nil {
			friendship, err := s.friendshipRepo.FindByUsers(ctx, id, otherUserID)
			if err == nil {
				conv.FriendshipStatus = &friendship.Status
			}
		}

		// 加载最后一条消息
		messages, err := s.conversationMessageRepo.FindMessages(ctx, conv.ID, 1, 0)
		if err == nil && len(messages) > 0 {
			// 加载发送者信息
			lastMsg := messages[0]
			if lastMsg.BotID != nil && lastMsg.BotName != nil && s.botRepo != nil {
				bot, botErr := s.botRepo.FindByID(ctx, *lastMsg.BotID)
				if botErr == nil {
					lastMsg.Sender = &models.User{
						ID:        bot.ID,
						Username:  bot.Name,
						AvatarURL: bot.AvatarURL,
					}
				}
			} else {
				sender, err := s.userRepo.FindByID(ctx, lastMsg.SenderID)
				if err == nil {
					sanitizeUser(sender)
					lastMsg.Sender = sender
				}
			}
			conv.LastMessage = lastMsg
		}
	}

	logger.InfofWithCaller("Retrieved %d conversations for user %s", len(conversations), userID)

	return conversations, nil
}

// CreateConversation 创建会话
func (s *ConversationService) CreateConversation(ctx context.Context, userID, targetUserID string) (*models.Conversation, error) {
	logger.InfofWithCaller("Creating conversation between %s and %s", userID, targetUserID)

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to parse user ID %s: %v", userID, err)
		return nil, err
	}

	targetUUID, err := uuid.Parse(targetUserID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to parse target user ID %s: %v", targetUserID, err)
		return nil, err
	}

	// 检查是否是同一个用户
	if userUUID == targetUUID {
		logger.ErrorfWithCaller("Attempt to create conversation with yourself: %s", userID)
		return nil, errCannotSelfChat
	}

	// 检查目标用户是否存在
	_, err = s.userRepo.FindByID(ctx, targetUUID)
	if err != nil {
		logger.ErrorfWithCaller("Target user not found: %s", targetUserID)
		return nil, errTargetNotFound
	}

	// 检查会话是否已存在
	existingConv, err := s.conversationRepo.FindByUsers(ctx, userUUID, targetUUID)
	if err == nil {
		logger.InfofWithCaller("Conversation already exists: %s", existingConv.ID)
		return existingConv, nil
	}

	// 创建会话
	conversation := &models.Conversation{
		ConversationType: models.ConversationTypeDirect,
		Name:             "", // 私聊会话名称将在加载时动态生成
		CreatedBy:        &userUUID,
	}

	err = s.conversationRepo.Create(ctx, conversation)
	if err != nil {
		logger.ErrorfWithCaller("Failed to create conversation: %v", err)
		return nil, err
	}

	// 创建enrollment记录（使用UTC时间）
	ownerEnrollment := &models.Enrollment{
		ConversationID: conversation.ID,
		UserID:         userUUID,
		Role:           models.EnrollmentRoleOwner,
		JoinedAt:       time.Now().UTC(),
	}
	err = s.enrollmentRepo.Create(ctx, ownerEnrollment)
	if err != nil {
		logger.ErrorfWithCaller("Failed to create owner enrollment: %v", err)
		return nil, err
	}

	memberEnrollment := &models.Enrollment{
		ConversationID: conversation.ID,
		UserID:         targetUUID,
		Role:           models.EnrollmentRoleMember,
		JoinedAt:       time.Now().UTC(),
	}
	err = s.enrollmentRepo.Create(ctx, memberEnrollment)
	if err != nil {
		logger.ErrorfWithCaller("Failed to create member enrollment: %v", err)
		return nil, err
	}

	logger.InfofWithCaller("Conversation created successfully: ID=%s, Type=%s", conversation.ID, conversation.ConversationType)

	return conversation, nil
}

// CreateGroupConversation 创建群聊会话
func (s *ConversationService) CreateGroupConversation(ctx context.Context, userID, name string, memberIDs []string) (*models.Conversation, error) {
	logger.InfofWithCaller("Creating group conversation: %s", name)

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to parse user ID %s: %v", userID, err)
		return nil, err
	}

	// 创建会话
	conversation := &models.Conversation{
		ConversationType: models.ConversationTypeGroup,
		Name:             name,
		CreatedBy:        &userUUID,
	}

	err = s.conversationRepo.Create(ctx, conversation)
	if err != nil {
		logger.ErrorfWithCaller("Failed to create conversation: %v", err)
		return nil, err
	}

	// 创建创建者的enrollment记录（使用UTC时间）
	ownerEnrollment := &models.Enrollment{
		ConversationID: conversation.ID,
		UserID:         userUUID,
		Role:           models.EnrollmentRoleOwner,
		JoinedAt:       time.Now().UTC(),
	}
	err = s.enrollmentRepo.Create(ctx, ownerEnrollment)
	if err != nil {
		logger.ErrorfWithCaller("Failed to create owner enrollment: %v", err)
		return nil, err
	}

	// 为其他成员创建enrollment记录
	memberUUIDs := []uuid.UUID{userUUID} // 包含创建者
	for _, memberIDStr := range memberIDs {
		memberUUID, err := uuid.Parse(memberIDStr)
		if err != nil {
			logger.ErrorfWithCaller("Failed to parse member ID %s: %v", memberIDStr, err)
			continue
		}

		memberEnrollment := &models.Enrollment{
			ConversationID: conversation.ID,
			UserID:         memberUUID,
			Role:           models.EnrollmentRoleMember,
			JoinedAt:       time.Now().UTC(),
		}
		err = s.enrollmentRepo.Create(ctx, memberEnrollment)
		if err != nil {
			logger.ErrorfWithCaller("Failed to create member enrollment: %v", err)
			continue
		}

		memberUUIDs = append(memberUUIDs, memberUUID)
	}

	// 通过WebSocket通知所有成员群聊创建成功
	if websocket.GlobalHub != nil {
		for _, memberUUID := range memberUUIDs {
			websocket.GlobalHub.SendToUser(memberUUID, "new_group_conversation", map[string]interface{}{
				"conversation_id": conversation.ID.String(),
				"name":            conversation.Name,
				"created_by":      userID,
				"member_count":    len(memberUUIDs),
			})
		}
		logger.InfofWithCaller("New group conversation notification sent to %d members", len(memberUUIDs))
	}

	logger.InfofWithCaller("Group conversation created successfully: ID=%s, Name=%s", conversation.ID, conversation.Name)

	return conversation, nil
}

// UpdateConversation 更新会话信息（群名称、群头像）
func (s *ConversationService) UpdateConversation(ctx context.Context, conversationIDStr, userID string, req *models.UpdateConversationRequest) (*models.Conversation, error) {
	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		return nil, err
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	// 验证用户是会话成员
	_, err = s.enrollmentRepo.FindByConversationAndUser(ctx, conversationID, userUUID)
	if err != nil {
		return nil, errNotParticipant
	}

	// 查找现有会话
	conversation, err := s.conversationRepo.FindByID(ctx, conversationID)
	if err != nil {
		return nil, errConversationNotFound
	}

	// 更新字段
	if req.Name != "" {
		conversation.Name = req.Name
	}
	if req.AvatarURL != "" {
		conversation.AvatarURL = req.AvatarURL
	}

	err = s.conversationRepo.Update(ctx, conversation)
	if err != nil {
		return nil, err
	}

	logger.InfofWithCaller("Conversation %s updated by user %s", conversationID, userID)
	return conversation, nil
}

// DeleteConversation 解散会话（仅群聊 owner 可操作）
func (s *ConversationService) DeleteConversation(ctx context.Context, conversationIDStr, userID string) error {
	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		return err
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	// 验证操作者是 owner
	enrollment, err := s.enrollmentRepo.FindByConversationAndUser(ctx, conversationID, userUUID)
	if err != nil {
		return errNotAuthorized
	}

	if enrollment.Role != models.EnrollmentRoleOwner {
		return errOnlyOwnerCanDelete
	}

	// 验证是群聊
	conversation, err := s.conversationRepo.FindByID(ctx, conversationID)
	if err != nil {
		return errConversationNotFound
	}

	if conversation.ConversationType != models.ConversationTypeGroup {
		return errOnlyGroupDeletable
	}

	// 通知所有成员群聊即将解散
	if websocket.GlobalHub != nil {
		members, err := s.enrollmentRepo.FindByConversationID(ctx, conversationID)
		if err == nil {
			memberIDs := make([]uuid.UUID, 0, len(members))
			for _, m := range members {
				memberIDs = append(memberIDs, m.UserID)
			}
			for _, mID := range memberIDs {
				websocket.GlobalHub.SendToUser(mID, "conversation_deleted", map[string]interface{}{
					"conversation_id": conversationIDStr,
					"deleted_by":      userID,
				})
			}
		}
	}

	// 删除会话（由于 ON DELETE CASCADE，enrollments 和 friendships 会被自动删除）
	// 消息表也需要手动删除
	err = s.conversationMessageRepo.DropMessageTable(ctx, conversationID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to drop message table for conversation %s: %v", conversationIDStr, err)
		// 继续执行，不因为消息表删除失败而中断
	}

	// 使用 conversation_repo 的方法删除
	_, err = database.GetPool().Exec(ctx, "DELETE FROM conversations WHERE id = $1", conversationID)
	if err != nil {
		return err
	}

	logger.InfofWithCaller("Conversation %s deleted by user %s", conversationIDStr, userID)
	return nil
}

// GetConversationMembers 获取会话成员
func (s *ConversationService) GetConversationMembers(ctx context.Context, conversationIDStr string) ([]*models.Enrollment, error) {
	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		return nil, err
	}

	members, err := s.enrollmentRepo.FindByConversationID(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	// 加载用户信息
	for _, member := range members {
		user, err := s.userRepo.FindByID(ctx, member.UserID)
		if err == nil {
			sanitizeUser(user)
			member.User = user
		}
	}

	return members, nil
}
