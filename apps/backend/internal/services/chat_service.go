package services

import (
	"context"
	"errors"
	"time"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"
	"purr-chat-server/internal/websocket"
	"purr-chat-server/pkg/logger"

	"github.com/google/uuid"
)

// ChatService 聊天服务
type ChatService struct {
	userRepo                repository.UserRepository
	conversationRepo        repository.ConversationRepository
	messageRepo             repository.MessageRepository
	friendshipRepo          repository.FriendshipRepository
	enrollmentRepo          repository.EnrollmentRepository
	conversationMessageRepo repository.ConversationMessageRepository
}

// NewChatService 创建聊天服务
func NewChatService(
	userRepo repository.UserRepository,
	conversationRepo repository.ConversationRepository,
	messageRepo repository.MessageRepository,
	friendshipRepo repository.FriendshipRepository,
	enrollmentRepo repository.EnrollmentRepository,
	conversationMessageRepo repository.ConversationMessageRepository,
) *ChatService {
	return &ChatService{
		userRepo:                userRepo,
		conversationRepo:        conversationRepo,
		messageRepo:             messageRepo,
		friendshipRepo:          friendshipRepo,
		enrollmentRepo:          enrollmentRepo,
		conversationMessageRepo: conversationMessageRepo,
	}
}

// GetConversations 获取用户的所有会话
func (s *ChatService) GetConversations(ctx context.Context, userID string) ([]*models.Conversation, error) {
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
						user.PasswordHash = ""
						user.Salt = ""
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
			sender, err := s.userRepo.FindByID(ctx, messages[0].SenderID)
			if err == nil {
				sender.PasswordHash = ""
				sender.Salt = ""
				messages[0].Sender = sender
			}
			conv.LastMessage = messages[0]
		}
	}

	logger.InfofWithCaller("Retrieved %d conversations for user %s", len(conversations), userID)

	return conversations, nil
}

// GetMessages 获取会话的消息
func (s *ChatService) GetMessages(ctx context.Context, conversationIDStr string, limit, offset int) ([]*models.Message, error) {
	// 解析 conversationID
	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		return nil, err
	}

	messages, err := s.conversationMessageRepo.FindMessages(ctx, conversationID, limit, offset)
	if err != nil {
		return nil, err
	}

	// 为每条消息加载发送者信息
	for _, msg := range messages {
		sender, err := s.userRepo.FindByID(ctx, msg.SenderID)
		if err == nil {
			sender.PasswordHash = ""
			sender.Salt = ""
			msg.Sender = sender
		}
	}

	return messages, nil
}

// GetAllMessages 获取会话的所有消息
func (s *ChatService) GetAllMessages(ctx context.Context, conversationIDStr string) ([]*models.Message, error) {
	// 解析 conversationID
	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		return nil, err
	}

	messages, err := s.conversationMessageRepo.FindAllMessages(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	// 为每条消息加载发送者信息
	for _, msg := range messages {
		sender, err := s.userRepo.FindByID(ctx, msg.SenderID)
		if err == nil {
			sender.PasswordHash = ""
			sender.Salt = ""
			msg.Sender = sender
		}
	}

	return messages, nil
}

// GetMessagesIncremental 增量获取会话的消息（从指定时间之后）
func (s *ChatService) GetMessagesIncremental(ctx context.Context, conversationIDStr string, sinceTimestamp int64) ([]*models.Message, error) {
	logger.InfofWithCaller("Getting incremental messages for conversation %s since %d", conversationIDStr, sinceTimestamp)

	// 解析 conversationID
	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		logger.ErrorfWithCaller("Failed to parse conversation ID %s: %v", conversationIDStr, err)
		return nil, err
	}

	// 将时间戳转换为time.Time
	since := time.UnixMilli(sinceTimestamp)

	// 获取增量消息
	messages, err := s.messageRepo.FindByConversationIDSince(ctx, conversationID, since)
	if err != nil {
		logger.ErrorfWithCaller("Failed to get incremental messages: %v", err)
		return nil, err
	}

	// 为每条消息加载发送者信息
	for _, msg := range messages {
		sender, err := s.userRepo.FindByID(ctx, msg.SenderID)
		if err == nil {
			sender.PasswordHash = ""
			sender.Salt = ""
			msg.Sender = sender
		}
	}

	logger.InfofWithCaller("Retrieved %d incremental messages for conversation %s", len(messages), conversationIDStr)
	return messages, nil
}

// SendMessage 发送消息
func (s *ChatService) SendMessage(ctx context.Context, senderID string, req *models.SendMessageRequest) (*models.Message, error) {
	logger.InfofWithCaller("Sending message from user %s to conversation %s", senderID, req.ConversationID)

	senderUUID, err := uuid.Parse(senderID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to parse sender ID %s: %v", senderID, err)
		return nil, err
	}

	// 检查发送者是否是会话的参与者
	enrollment, err := s.enrollmentRepo.FindByConversationAndUser(ctx, req.ConversationID, senderUUID)
	if err != nil {
		logger.ErrorfWithCaller("User %s is not a participant in conversation %s", senderID, req.ConversationID)
		return nil, errors.New("not a participant in this conversation")
	}
	_ = enrollment // 避免未使用变量警告

	// 创建消息
	message := &models.Message{
		ID:             uuid.New(),
		ConversationID: req.ConversationID,
		SenderID:       senderUUID,
		Content:        req.Content,
		MsgType:        models.MsgType(req.MsgType),
		CreatedAt:      time.Now(),
	}

	err = s.conversationMessageRepo.InsertMessage(ctx, req.ConversationID, message)
	if err != nil {
		logger.ErrorfWithCaller("Failed to create message: %v", err)
		return nil, err
	}

	// 加载发送者信息
	sender, err := s.userRepo.FindByID(ctx, senderUUID)
	if err == nil {
		sender.PasswordHash = ""
		sender.Salt = ""
		message.Sender = sender
	}

	// 通过WebSocket推送消息给会话的其他成员
	if websocket.GlobalHub != nil {
		// 获取会话的所有成员
		members, err := s.enrollmentRepo.FindByConversationID(ctx, req.ConversationID)
		if err == nil {
			// 提取成员ID列表
			memberIDs := make([]uuid.UUID, 0, len(members))
			for _, member := range members {
				memberIDs = append(memberIDs, member.UserID)
			}
			// 推送消息给所有成员
			websocket.GlobalHub.SendToConversation(req.ConversationID, senderUUID, *message, memberIDs)
			logger.InfofWithCaller("Message broadcasted via WebSocket to %d members", len(memberIDs))
		} else {
			logger.ErrorfWithCaller("Failed to get conversation members for WebSocket broadcast: %v", err)
		}
	}

	logger.InfofWithCaller("Message sent successfully: ID=%s, ConversationID=%s, SenderID=%s", message.ID, message.ConversationID, message.SenderID)

	return message, nil
}

// CreateConversation 创建会话
func (s *ChatService) CreateConversation(ctx context.Context, userID, targetUserID string) (*models.Conversation, error) {
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
		return nil, errors.New("cannot create conversation with yourself")
	}

	// 检查目标用户是否存在
	targetUser, err := s.userRepo.FindByID(ctx, targetUUID)
	if err != nil {
		logger.ErrorfWithCaller("Target user not found: %s", targetUserID)
		return nil, errors.New("target user not found")
	}
	_ = targetUser // 避免未使用变量警告

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

	// 为会话创建消息表
	err = s.conversationMessageRepo.CreateMessageTable(ctx, conversation.ID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to create message table: %v", err)
		return nil, err
	}

	// 创建enrollment记录
	ownerEnrollment := &models.Enrollment{
		ConversationID: conversation.ID,
		UserID:         userUUID,
		Role:           models.EnrollmentRoleOwner,
		JoinedAt:       time.Now(),
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
		JoinedAt:       time.Now(),
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
func (s *ChatService) CreateGroupConversation(ctx context.Context, userID, name string, memberIDs []string) (*models.Conversation, error) {
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

	// 为会话创建消息表
	err = s.conversationMessageRepo.CreateMessageTable(ctx, conversation.ID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to create message table: %v", err)
		return nil, err
	}

	// 创建创建者的enrollment记录
	ownerEnrollment := &models.Enrollment{
		ConversationID: conversation.ID,
		UserID:         userUUID,
		Role:           models.EnrollmentRoleOwner,
		JoinedAt:       time.Now(),
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
			JoinedAt:       time.Now(),
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

// GetFriends 获取用户的好友列表
func (s *ChatService) GetFriends(ctx context.Context, userID string) ([]*models.Friendship, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	friendships, err := s.friendshipRepo.FindByUserID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 为每个好友关系加载用户信息
	for _, fs := range friendships {
		// 确定好友ID
		var friendID uuid.UUID
		if fs.UserID == id {
			friendID = fs.FriendID
		} else if fs.FriendID == id {
			friendID = fs.UserID
		} else {
			logger.ErrorfWithCaller("Friendship does not belong to user %s, skipping", id)
			continue // 跳过不属于当前用户的好友关系
		}

		// 加载好友信息
		friend, err := s.userRepo.FindByID(ctx, friendID)
		if err == nil {
			friend.PasswordHash = ""
			friend.Salt = ""
			fs.Friend = friend
		}
	}

	return friendships, nil
}

// GetPendingFriendRequests 获取用户的待处理好友请求
func (s *ChatService) GetPendingFriendRequests(ctx context.Context, userID string) ([]*models.Friendship, error) {
	logger.InfofWithCaller("Getting pending friend requests for user: %s", userID)

	id, err := uuid.Parse(userID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to parse user ID %s: %v", userID, err)
		return nil, err
	}

	friendships, err := s.friendshipRepo.FindPendingRequests(ctx, id)
	if err != nil {
		logger.ErrorfWithCaller("Failed to get pending friend requests for user %s: %v", userID, err)
		return nil, err
	}

	// 为每个好友请求加载发送者信息
	for _, fs := range friendships {
		// 加载发送者信息（UserID是发送方）
		sender, err := s.userRepo.FindByID(ctx, fs.UserID)
		if err == nil {
			sender.PasswordHash = ""
			sender.Salt = ""
			fs.User = sender
		}
	}

	logger.InfofWithCaller("Retrieved %d pending friend requests for user %s", len(friendships), userID)
	return friendships, nil
}

// GetAllFriendRequests 获取用户的所有好友申请记录
func (s *ChatService) GetAllFriendRequests(ctx context.Context, userID string) ([]*models.Friendship, error) {
	logger.InfofWithCaller("Getting all friend requests for user: %s", userID)

	id, err := uuid.Parse(userID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to parse user ID %s: %v", userID, err)
		return nil, err
	}

	friendships, err := s.friendshipRepo.FindAllRequests(ctx, id)
	if err != nil {
		logger.ErrorfWithCaller("Failed to get all friend requests for user %s: %v", userID, err)
		return nil, err
	}

	// 为每个好友请求加载相关用户信息
	for _, fs := range friendships {
		if fs.UserID == id {
			// 当前用户是发送方，加载接收方信息
			receiver, err := s.userRepo.FindByID(ctx, fs.FriendID)
			if err == nil {
				receiver.PasswordHash = ""
				receiver.Salt = ""
				fs.Friend = receiver
			}
		} else {
			// 当前用户是接收方，加载发送方信息
			sender, err := s.userRepo.FindByID(ctx, fs.UserID)
			if err == nil {
				sender.PasswordHash = ""
				sender.Salt = ""
				fs.User = sender
			}
		}
	}

	logger.InfofWithCaller("Retrieved %d friend requests for user %s", len(friendships), userID)
	return friendships, nil
}

// GetUserByID 根据ID获取用户
func (s *ChatService) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 清除密码相关字段
	user.PasswordHash = ""
	user.Salt = ""

	return user, nil
}

// AddMemberToConversation 添加成员到会话
func (s *ChatService) AddMemberToConversation(ctx context.Context, conversationIDStr, userID, targetUserID string, role models.EnrollmentRole) error {
	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		return err
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	targetUUID, err := uuid.Parse(targetUserID)
	if err != nil {
		return err
	}

	// 检查操作者是否是会话的管理员或拥有者
	enrollment, err := s.enrollmentRepo.FindByConversationAndUser(ctx, conversationID, userUUID)
	if err != nil {
		return errors.New("not authorized")
	}

	if enrollment.Role != models.EnrollmentRoleOwner && enrollment.Role != models.EnrollmentRoleAdmin {
		return errors.New("not authorized")
	}

	// 检查目标用户是否已经在会话中
	_, err = s.enrollmentRepo.FindByConversationAndUser(ctx, conversationID, targetUUID)
	if err == nil {
		return errors.New("user already in conversation")
	}

	// 添加成员
	newEnrollment := &models.Enrollment{
		ConversationID: conversationID,
		UserID:         targetUUID,
		Role:           role,
		JoinedAt:       time.Now(),
	}

	err = s.enrollmentRepo.Create(ctx, newEnrollment)
	if err != nil {
		return err
	}

	// 通过WebSocket通知会话的所有成员有新成员加入
	if websocket.GlobalHub != nil {
		// 获取会话的所有成员
		members, err := s.enrollmentRepo.FindByConversationID(ctx, conversationID)
		if err == nil {
			// 提取成员ID列表
			memberIDs := make([]uuid.UUID, 0, len(members))
			for _, member := range members {
				memberIDs = append(memberIDs, member.UserID)
			}

			// 通知所有成员
			for _, memberID := range memberIDs {
				websocket.GlobalHub.SendToUser(memberID, "conversation_member_added", map[string]interface{}{
					"conversation_id": conversationIDStr,
					"user_id":         targetUserID,
					"role":            role,
					"added_by":        userID,
				})
			}
			logger.InfofWithCaller("Member added notification sent to %d members", len(memberIDs))
		}
	}

	return nil
}

// RemoveMemberFromConversation 从会话中移除成员
func (s *ChatService) RemoveMemberFromConversation(ctx context.Context, conversationIDStr, userID, targetUserID string) error {
	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		return err
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	targetUUID, err := uuid.Parse(targetUserID)
	if err != nil {
		return err
	}

	// 检查操作者是否是会话的管理员或拥有者
	enrollment, err := s.enrollmentRepo.FindByConversationAndUser(ctx, conversationID, userUUID)
	if err != nil {
		return errors.New("not authorized")
	}

	if enrollment.Role != models.EnrollmentRoleOwner && enrollment.Role != models.EnrollmentRoleAdmin {
		return errors.New("not authorized")
	}

	// 不能移除拥有者
	targetEnrollment, err := s.enrollmentRepo.FindByConversationAndUser(ctx, conversationID, targetUUID)
	if err != nil {
		return errors.New("user not in conversation")
	}

	if targetEnrollment.Role == models.EnrollmentRoleOwner {
		return errors.New("cannot remove owner")
	}

	// 移除成员
	err = s.enrollmentRepo.DeleteByConversationAndUser(ctx, conversationID, targetUUID)
	if err != nil {
		return err
	}

	// 通过WebSocket通知会话的所有成员有成员被移除
	if websocket.GlobalHub != nil {
		// 获取会话的所有成员
		members, err := s.enrollmentRepo.FindByConversationID(ctx, conversationID)
		if err == nil {
			// 提取成员ID列表
			memberIDs := make([]uuid.UUID, 0, len(members))
			for _, member := range members {
				memberIDs = append(memberIDs, member.UserID)
			}

			// 通知所有成员
			for _, memberID := range memberIDs {
				websocket.GlobalHub.SendToUser(memberID, "conversation_member_removed", map[string]interface{}{
					"conversation_id": conversationIDStr,
					"user_id":         targetUserID,
					"removed_by":      userID,
				})
			}
			logger.InfofWithCaller("Member removed notification sent to %d members", len(memberIDs))
		}
	}

	return nil
}

// GetConversationMembers 获取会话成员
func (s *ChatService) GetConversationMembers(ctx context.Context, conversationIDStr string) ([]*models.Enrollment, error) {
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
			user.PasswordHash = ""
			user.Salt = ""
			member.User = user
		}
	}

	return members, nil
}

// SendFriendRequest 发送好友请求
func (s *ChatService) SendFriendRequest(ctx context.Context, userID, targetUserID string) (*models.Conversation, error) {
	logger.InfofWithCaller("Sending friend request from %s to %s", userID, targetUserID)

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
		logger.ErrorfWithCaller("Attempt to send friend request to yourself: %s", userID)
		return nil, errors.New("cannot send friend request to yourself")
	}

	// 检查目标用户是否存在
	targetUser, err := s.userRepo.FindByID(ctx, targetUUID)
	if err != nil {
		logger.ErrorfWithCaller("Target user not found: %s", targetUserID)
		return nil, errors.New("target user not found")
	}
	_ = targetUser // 避免未使用变量警告

	// 检查是否已经存在好友关系
	existingFriendship, err := s.friendshipRepo.FindByUsers(ctx, userUUID, targetUUID)
	if err == nil {
		// 如果已存在好友关系，检查状态
		if existingFriendship.Status == models.FriendshipStatusPending {
			logger.InfofWithCaller("Friend request already pending between %s and %s", userID, targetUserID)
			return nil, errors.New("friend request already pending")
		} else if existingFriendship.Status == models.FriendshipStatusAccepted {
			logger.InfofWithCaller("Already friends with user %s", targetUserID)
			return nil, errors.New("already friends with this user")
		}
		// 如果是 blocked 状态，允许重新发送请求
	}

	// 创建会话
	conversation, err := s.CreateConversation(ctx, userID, targetUserID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to create conversation: %v", err)
		return nil, err
	}

	// 创建好友关系记录（状态为 pending）
	friendship := &models.Friendship{
		UserID:         userUUID,
		FriendID:       targetUUID,
		ConversationID: conversation.ID,
		Status:         models.FriendshipStatusPending,
	}
	err = s.friendshipRepo.Create(ctx, friendship)
	if err != nil {
		logger.ErrorfWithCaller("Failed to create friendship: %v", err)
		return nil, err
	}

	logger.InfofWithCaller("Friendship created successfully: ID=%s, Status=%s", friendship.ID, friendship.Status)

	// 通过WebSocket通知接收者有新的好友请求
	if websocket.GlobalHub != nil {
		websocket.GlobalHub.SendToUser(targetUUID, "new_friend_request", map[string]interface{}{
			"conversation_id": conversation.ID.String(),
			"sender_id":       userID,
			"status":          "pending",
		})
		logger.InfofWithCaller("New friend request notification sent to user %s", targetUUID)
	}

	logger.InfofWithCaller("Friend request sent successfully from %s to %s", userID, targetUserID)

	return conversation, nil
}

// HandleFriendRequest 处理好友请求
func (s *ChatService) HandleFriendRequest(ctx context.Context, userID, conversationIDStr string, action string) error {
	logger.InfofWithCaller("Handling friend request: action=%s, user=%s, conversation=%s", action, userID, conversationIDStr)

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to parse user ID %s: %v", userID, err)
		return err
	}

	// 查找当前用户的所有好友请求，找到待处理的请求
	friendships, err := s.friendshipRepo.FindByUserID(ctx, userUUID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to get friendships for user %s: %v", userID, err)
		return errors.New("failed to get friendships")
	}

	// 找到待处理的好友请求（当前用户是接收方）
	var friendship *models.Friendship
	var senderUUID uuid.UUID
	var conversationUUID uuid.UUID

	for _, fs := range friendships {
		// 检查是否是待处理的请求
		if fs.Status == models.FriendshipStatusPending {
			// 如果当前用户是发送方，则不允许处理自己的好友请求
			if fs.UserID == userUUID {
				logger.ErrorfWithCaller("Sender %s is not authorized to handle their own friend request", userID)
				return errors.New("not authorized to handle this friend request")
			}
			// 当前用户是接收方
			if fs.FriendID == userUUID {
				friendship = fs
				senderUUID = fs.UserID
				break
			}
		}
	}

	if friendship == nil {
		logger.ErrorfWithCaller("No pending friend request found for user %s", userID)
		return errors.New("no pending friend request found")
	}

	// 如果提供了 conversation_id，使用它；否则查找对应的会话
	if conversationIDStr != "" {
		conversationUUID, err = uuid.Parse(conversationIDStr)
		if err != nil {
			logger.ErrorfWithCaller("Failed to parse conversation ID %s: %v", conversationIDStr, err)
			return err
		}
	} else {
		// 通过 user_id 和 friend_id 查找对应的会话
		conversation, err := s.conversationRepo.FindByUsers(ctx, userUUID, senderUUID)
		if err != nil {
			logger.ErrorfWithCaller("Failed to find conversation between %s and %s: %v", userID, senderUUID, err)
			return errors.New("conversation not found")
		}
		conversationUUID = conversation.ID

		// 更新好友关系的 conversation_id
		friendship.ConversationID = conversationUUID
		err = s.friendshipRepo.Update(ctx, friendship)
		if err != nil {
			logger.ErrorfWithCaller("Failed to update friendship conversation_id: %v", err)
			// 继续处理，不因为更新失败而中断
		}
	}

	// 根据操作更新状态
	if action == "accept" {
		friendship.Status = models.FriendshipStatusAccepted
		logger.InfofWithCaller("Friend request accepted between %s and %s", userID, senderUUID)

		// 更新好友关系状态
		err = s.friendshipRepo.Update(ctx, friendship)
		if err != nil {
			logger.ErrorfWithCaller("Failed to update friendship: %v", err)
			return err
		}

		// 通过WebSocket通知双方好友请求状态已更新
		if websocket.GlobalHub != nil {
			// 通知接收者（当前用户）
			websocket.GlobalHub.SendToUser(userUUID, "friend_request_update", map[string]interface{}{
				"conversation_id": conversationUUID.String(),
				"status":          "accepted",
				"action":          action,
			})
			// 通知发送者（另一个用户）
			websocket.GlobalHub.SendToUser(senderUUID, "friend_request_update", map[string]interface{}{
				"conversation_id": conversationUUID.String(),
				"status":          "accepted",
				"action":          action,
			})
			logger.InfofWithCaller("Friend request acceptance notification sent to both users %s and %s", userUUID, senderUUID)
		}
	} else if action == "reject" {
		// 将好友关系状态设置为 rejected
		friendship.Status = models.FriendshipStatusRejected
		logger.InfofWithCaller("Friend request rejected between %s and %s", userID, senderUUID)

		// 更新好友关系状态
		err = s.friendshipRepo.Update(ctx, friendship)
		if err != nil {
			logger.ErrorfWithCaller("Failed to update friendship: %v", err)
			return err
		}

		// 通过WebSocket通知发送者好友请求被拒绝
		if websocket.GlobalHub != nil {
			websocket.GlobalHub.SendToUser(senderUUID, "friend_request_update", map[string]interface{}{
				"conversation_id": conversationUUID.String(),
				"status":          "rejected",
				"action":          action,
			})
			logger.InfofWithCaller("Friend request rejection notification sent to user %s", senderUUID)
		}
		return nil
	} else {
		logger.ErrorfWithCaller("Invalid action: %s", action)
		return errors.New("invalid action")
	}

	return nil
}
