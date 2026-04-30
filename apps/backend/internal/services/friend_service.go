package services

import (
	"context"
	"errors"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"
	"purr-chat-server/internal/websocket"
	"purr-chat-server/pkg/logger"

	"github.com/google/uuid"
)

// FriendService 好友服务
type FriendService struct {
	userRepo                repository.UserRepository
	friendshipRepo          repository.FriendshipRepository
	enrollmentRepo          repository.EnrollmentRepository
	conversationMessageRepo repository.ConversationMessageRepository
	botRepo                 repository.BotRepository
}

// NewFriendService 创建好友服务
func NewFriendService(
	userRepo repository.UserRepository,
	friendshipRepo repository.FriendshipRepository,
	enrollmentRepo repository.EnrollmentRepository,
	conversationMessageRepo repository.ConversationMessageRepository,
) *FriendService {
	return &FriendService{
		userRepo:                userRepo,
		friendshipRepo:          friendshipRepo,
		enrollmentRepo:          enrollmentRepo,
		conversationMessageRepo: conversationMessageRepo,
	}
}

// SetBotRepo 设置 Bot 仓储（可选依赖）
func (s *FriendService) SetBotRepo(botRepo repository.BotRepository) {
	s.botRepo = botRepo
}

// GetFriends 获取用户的好友列表
func (s *FriendService) GetFriends(ctx context.Context, userID string) ([]*models.Friendship, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	friendships, err := s.friendshipRepo.FindByUserID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 按 friend ID 去重（Bot 创建时会建立双向好友关系，
	// FindByUserID 的 OR 查询会导致同一好友返回两条记录）
	seen := make(map[uuid.UUID]bool)
	var deduped []*models.Friendship
	for _, fs := range friendships {
		var friendID uuid.UUID
		if fs.UserID == id {
			friendID = fs.FriendID
		} else if fs.FriendID == id {
			friendID = fs.UserID
		} else {
			continue
		}
		if !seen[friendID] {
			seen[friendID] = true
			deduped = append(deduped, fs)
		}
	}
	friendships = deduped

	// 为每个好友关系加载用户信息
	for _, fs := range friendships {
		var friendID uuid.UUID
		if fs.UserID == id {
			friendID = fs.FriendID
		} else {
			friendID = fs.UserID
		}

		friend, err := s.userRepo.FindByID(ctx, friendID)
		if err == nil {
			sanitizeUser(friend)
			fs.Friend = friend
		}
	}

	return friendships, nil
}

// GetPendingFriendRequests 获取用户的待处理好友请求
func (s *FriendService) GetPendingFriendRequests(ctx context.Context, userID string) ([]*models.Friendship, error) {
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
			sanitizeUser(sender)
			fs.User = sender
		}
	}

	logger.InfofWithCaller("Retrieved %d pending friend requests for user %s", len(friendships), userID)
	return friendships, nil
}

// GetAllFriendRequests 获取用户的所有好友申请记录
func (s *FriendService) GetAllFriendRequests(ctx context.Context, userID string) ([]*models.Friendship, error) {
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

	// 为每个好友请求加载对方用户信息，始终填充到 User 字段供前端使用
	for _, fs := range friendships {
		var otherUserID uuid.UUID
		if fs.UserID == id {
			otherUserID = fs.FriendID
		} else {
			otherUserID = fs.UserID
		}

		otherUser, err := s.userRepo.FindByID(ctx, otherUserID)
		if err == nil {
			sanitizeUser(otherUser)
			fs.User = otherUser
		}
	}

	logger.InfofWithCaller("Retrieved %d friend requests for user %s", len(friendships), userID)
	return friendships, nil
}

// SendFriendRequest 发送好友请求
func (s *FriendService) SendFriendRequest(ctx context.Context, userID, targetUserID string, createConversationFn func(ctx context.Context, uid, tid string) (*models.Conversation, error)) (*models.Conversation, error) {
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

	// 检查目标是否是 Bot，走不同的好友添加逻辑
	if targetUser.IsBot && s.botRepo != nil {
		bot, botErr := s.botRepo.FindByID(ctx, targetUUID)
		if botErr != nil {
			return nil, errors.New("bot not found")
		}

		// private Bot 不允许被添加为好友
		if bot.Visibility == models.BotVisibilityPrivate {
			return nil, errors.New("this bot is private")
		}

		// 创建会话（双方 enrollment）
		conversation, err := createConversationFn(ctx, userID, targetUserID)
		if err != nil {
			logger.ErrorfWithCaller("Failed to create conversation for bot friend request: %v", err)
			return nil, err
		}

		// 创建好友关系
		autoAccept := bot.Visibility == models.BotVisibilityGlobal
		friendshipStatus := models.FriendshipStatusAccepted
		if !autoAccept {
			friendshipStatus = models.FriendshipStatusPending
		}

		friendship := &models.Friendship{
			UserID:         userUUID,
			FriendID:       targetUUID,
			ConversationID: conversation.ID,
			Status:         friendshipStatus,
		}
		err = s.friendshipRepo.Create(ctx, friendship)
		if err != nil {
			return nil, err
		}

		// WebSocket 通知
		if websocket.GlobalHub != nil {
			if autoAccept {
				websocket.GlobalHub.SendToUser(userUUID, "friend_request_update", map[string]interface{}{
					"conversation_id": conversation.ID.String(),
					"status":          "accepted",
					"action":          "auto_accept",
				})
			} else {
				websocket.GlobalHub.SendToUser(bot.OwnerID, "new_friend_request", map[string]interface{}{
					"conversation_id": conversation.ID.String(),
					"sender_id":       userID,
					"status":          "pending",
					"bot_id":          bot.ID.String(),
				})
				websocket.GlobalHub.SendToUser(userUUID, "friend_request_update", map[string]interface{}{
					"conversation_id": conversation.ID.String(),
					"status":          "pending",
					"action":          "sent",
				})
			}
		}

		logger.InfofWithCaller("Bot friend request processed: user=%s, bot=%s, visibility=%s, autoAccept=%v", userID, targetUserID, bot.Visibility, autoAccept)
		return conversation, nil
	}

	// 创建会话
	conversation, err := createConversationFn(ctx, userID, targetUserID)
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
func (s *FriendService) HandleFriendRequest(ctx context.Context, userID, conversationIDStr string, action string) error {
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

	// 检查是否是 Bot 好友请求：验证当前处理者是 Bot 的 owner
	senderUser, senderErr := s.userRepo.FindByID(ctx, senderUUID)
	if senderErr == nil && senderUser.IsBot && s.botRepo != nil {
		bot, botErr := s.botRepo.FindByID(ctx, senderUUID)
		if botErr == nil && bot.OwnerID != userUUID {
			return errors.New("only bot owner can approve bot friend requests")
		}
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
		// 这里需要 conversationRepo，但 friendService 没有它
		// 使用 friendship 中已有的 ConversationID
		conversationUUID = friendship.ConversationID
		friendship.ConversationID = conversationUUID
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
