package services

import (
	"context"
	"errors"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"
	"purr-chat-server/pkg/logger"

	"github.com/google/uuid"
)

// ChatService 聊天服务
type ChatService struct {
	userRepo         repository.UserRepository
	conversationRepo repository.ConversationRepository
	messageRepo      repository.MessageRepository
	friendshipRepo   repository.FriendshipRepository
}

// NewChatService 创建聊天服务
func NewChatService(
	userRepo repository.UserRepository,
	conversationRepo repository.ConversationRepository,
	messageRepo repository.MessageRepository,
	friendshipRepo repository.FriendshipRepository,
) *ChatService {
	return &ChatService{
		userRepo:         userRepo,
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		friendshipRepo:   friendshipRepo,
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

	// 为每个会话加载关联的用户信息和最后一条消息
	for _, conv := range conversations {
		// 加载用户信息
		if conv.User1ID != id {
			user, err := s.userRepo.FindByID(ctx, conv.User1ID)
			if err == nil {
				user.PasswordHash = ""
				user.Salt = ""
				conv.User1 = user
			}
		} else {
			user, err := s.userRepo.FindByID(ctx, conv.User2ID)
			if err == nil {
				user.PasswordHash = ""
				user.Salt = ""
				conv.User2 = user
			}
		}

		// 加载最后一条消息
		lastMsg, err := s.messageRepo.FindLastByConversationID(ctx, conv.ID)
		if err == nil {
			// 加载发送者信息
			sender, err := s.userRepo.FindByID(ctx, lastMsg.SenderID)
			if err == nil {
				sender.PasswordHash = ""
				sender.Salt = ""
				lastMsg.Sender = sender
			}
			conv.LastMessage = lastMsg
		}
	}

	logger.InfofWithCaller("Retrieved %d conversations for user %s", len(conversations), userID)

	return conversations, nil
}

// GetMessages 获取会话的消息
func (s *ChatService) GetMessages(ctx context.Context, conversationID uuid.UUID, limit, offset int) ([]*models.Message, error) {
	messages, err := s.messageRepo.FindByConversationID(ctx, conversationID, limit, offset)
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

// SendMessage 发送消息
func (s *ChatService) SendMessage(ctx context.Context, senderID string, req *models.SendMessageRequest) (*models.Message, error) {
	logger.InfofWithCaller("Sending message from user %s to conversation %s", senderID, req.ConversationID)

	senderUUID, err := uuid.Parse(senderID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to parse sender ID %s: %v", senderID, err)
		return nil, err
	}

	// 验证会话是否存在
	conversation, err := s.conversationRepo.FindByID(ctx, req.ConversationID)
	if err != nil {
		logger.ErrorfWithCaller("Conversation not found: %s", req.ConversationID)
		return nil, errors.New("conversation not found")
	}

	// 检查发送者是否是会话的参与者
	if conversation.User1ID != senderUUID && conversation.User2ID != senderUUID {
		logger.ErrorfWithCaller("User %s is not a participant in conversation %s", senderID, req.ConversationID)
		return nil, errors.New("not a participant in this conversation")
	}

	// 如果是陌生人会话，检查消息数量限制
	if conversation.ConversationType == models.ConversationTypeStranger {
		// 检查是否是发送者发送的消息
		count, err := s.messageRepo.CountByConversationID(ctx, req.ConversationID)
		if err != nil {
			logger.ErrorfWithCaller("Failed to count messages for conversation %s: %v", req.ConversationID, err)
			return nil, err
		}

		// 如果对方还没有回复，限制只能发送3条消息
		if conversation.RequestStatus == models.RequestStatusNone && count >= 3 {
			logger.ErrorfWithCaller("Message limit reached for stranger conversation %s", req.ConversationID)
			return nil, errors.New("message limit reached for stranger conversation")
		}
	}

	// 创建消息
	message := &models.Message{
		ConversationID: req.ConversationID,
		SenderID:       senderUUID,
		Content:        req.Content,
		MsgType:        models.MsgType(req.MsgType),
	}

	err = s.messageRepo.Create(ctx, message)
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

	// 检查会话是否已存在
	existingConv, err := s.conversationRepo.FindByUsers(ctx, userUUID, targetUUID)
	if err == nil {
		logger.InfofWithCaller("Conversation already exists: %s", existingConv.ID)
		return existingConv, nil
	}

	// 检查是否已经是好友
	friendship, err := s.friendshipRepo.FindByUsers(ctx, userUUID, targetUUID)
	isFriend := err == nil && friendship.Status == models.FriendshipStatusAccepted

	// 创建会话
	conversation := &models.Conversation{
		ConversationType:  models.ConversationTypeStranger,
		User1ID:           userUUID,
		User2ID:           targetUUID,
		HasPendingRequest: false,
		RequestStatus:     models.RequestStatusNone,
	}

	if isFriend {
		conversation.ConversationType = models.ConversationTypeFriend
	}

	err = s.conversationRepo.Create(ctx, conversation)
	if err != nil {
		logger.ErrorfWithCaller("Failed to create conversation: %v", err)
		return nil, err
	}

	logger.InfofWithCaller("Conversation created successfully: ID=%s, Type=%s", conversation.ID, conversation.ConversationType)

	return conversation, nil
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

	// 检查是否已经是好友
	friendship, err := s.friendshipRepo.FindByUsers(ctx, userUUID, targetUUID)
	if err == nil && friendship.Status == models.FriendshipStatusAccepted {
		logger.ErrorfWithCaller("Users are already friends: %s and %s", userID, targetUserID)
		return nil, errors.New("already friends")
	}

	// 获取或创建会话
	conversation, err := s.conversationRepo.FindByUsers(ctx, userUUID, targetUUID)
	if err != nil {
		// 创建新会话
		conversation = &models.Conversation{
			ConversationType:  models.ConversationTypeStranger,
			User1ID:           userUUID,
			User2ID:           targetUUID,
			HasPendingRequest: true,
			RequestStatus:     models.RequestStatusPending,
		}
		err = s.conversationRepo.Create(ctx, conversation)
		if err != nil {
			logger.ErrorfWithCaller("Failed to create conversation for friend request: %v", err)
			return nil, err
		}
	} else {
		// 标记为有待处理请求
		err = s.conversationRepo.MarkAsPendingRequest(ctx, conversation.ID)
		if err != nil {
			logger.ErrorfWithCaller("Failed to mark conversation as pending: %v", err)
			return nil, err
		}
		conversation.HasPendingRequest = true
		conversation.RequestStatus = models.RequestStatusPending
	}

	// 创建好友关系记录
	if friendship == nil {
		friendship = &models.Friendship{
			UserID:   userUUID,
			FriendID: targetUUID,
			Status:   models.FriendshipStatusPending,
		}
		err = s.friendshipRepo.Create(ctx, friendship)
		if err != nil {
			logger.ErrorfWithCaller("Failed to create friendship: %v", err)
			return nil, err
		}
	} else {
		friendship.Status = models.FriendshipStatusPending
		err = s.friendshipRepo.Update(ctx, friendship)
		if err != nil {
			logger.ErrorfWithCaller("Failed to update friendship: %v", err)
			return nil, err
		}
	}

	logger.InfofWithCaller("Friend request sent successfully: From=%s, To=%s", userID, targetUserID)

	return conversation, nil
}

// HandleFriendRequest 处理好友请求
func (s *ChatService) HandleFriendRequest(ctx context.Context, userID string, req *models.HandleFriendRequestRequest) (*models.Conversation, error) {
	logger.InfofWithCaller("Handling friend request: ConversationID=%s, Action=%s, User=%s", req.ConversationID, req.Action, userID)

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to parse user ID %s: %v", userID, err)
		return nil, err
	}

	// 获取会话
	conversation, err := s.conversationRepo.FindByID(ctx, req.ConversationID)
	if err != nil {
		logger.ErrorfWithCaller("Conversation not found: %s", req.ConversationID)
		return nil, errors.New("conversation not found")
	}

	// 检查用户是否是会话的参与者
	if conversation.User1ID != userUUID && conversation.User2ID != userUUID {
		logger.ErrorfWithCaller("User %s is not a participant in conversation %s", userID, req.ConversationID)
		return nil, errors.New("not a participant in this conversation")
	}

	// 确定对方用户ID
	var otherUserID uuid.UUID
	if conversation.User1ID == userUUID {
		otherUserID = conversation.User2ID
	} else {
		otherUserID = conversation.User1ID
	}

	// 获取好友关系
	friendship, err := s.friendshipRepo.FindByUsers(ctx, userUUID, otherUserID)
	if err != nil {
		logger.ErrorfWithCaller("Friend request not found for users %s and %s", userID, otherUserID)
		return nil, errors.New("friend request not found")
	}

	// 处理请求
	if req.Action == "accept" {
		// 接受好友请求
		friendship.Status = models.FriendshipStatusAccepted
		err = s.friendshipRepo.Update(ctx, friendship)
		if err != nil {
			logger.ErrorfWithCaller("Failed to update friendship: %v", err)
			return nil, err
		}

		// 更新会话类型
		conversation.ConversationType = models.ConversationTypeFriend
		conversation.HasPendingRequest = false
		conversation.RequestStatus = models.RequestStatusAccepted
		err = s.conversationRepo.Update(ctx, conversation)
		if err != nil {
			logger.ErrorfWithCaller("Failed to update conversation: %v", err)
			return nil, err
		}

		// 创建双向好友关系
		reverseFriendship := &models.Friendship{
			UserID:   otherUserID,
			FriendID: userUUID,
			Status:   models.FriendshipStatusAccepted,
		}
		err = s.friendshipRepo.Create(ctx, reverseFriendship)
		if err != nil {
			logger.ErrorfWithCaller("Failed to create reverse friendship: %v", err)
			return nil, err
		}

		logger.InfofWithCaller("Friend request accepted: %s and %s are now friends", userID, otherUserID)
	} else {
		// 拒绝好友请求
		friendship.Status = models.FriendshipStatusBlocked
		err = s.friendshipRepo.Update(ctx, friendship)
		if err != nil {
			logger.ErrorfWithCaller("Failed to update friendship: %v", err)
			return nil, err
		}

		// 更新会话状态
		conversation.HasPendingRequest = false
		conversation.RequestStatus = models.RequestStatusRejected
		err = s.conversationRepo.Update(ctx, conversation)
		if err != nil {
			logger.ErrorfWithCaller("Failed to update conversation: %v", err)
			return nil, err
		}

		logger.InfofWithCaller("Friend request rejected: %s rejected %s", userID, otherUserID)
	}

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
		} else {
			friendID = fs.UserID
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
