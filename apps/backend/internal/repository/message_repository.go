package repository

import (
	"context"
	"fmt"
	"time"

	"purr-chat-server/internal/models"
	"purr-chat-server/pkg/database"
	"purr-chat-server/pkg/logger"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// MessageRepository 消息仓储接口
type MessageRepository interface {
	Create(ctx context.Context, message *models.Message) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Message, error)
	FindByConversationID(ctx context.Context, conversationID uuid.UUID, limit, offset int) ([]*models.Message, error)
	CountByConversationID(ctx context.Context, conversationID uuid.UUID) (int, error)
	CountUnreadByConversationID(ctx context.Context, conversationID, userID uuid.UUID) (int, error)
	FindLastByConversationID(ctx context.Context, conversationID uuid.UUID) (*models.Message, error)
	FindByConversationIDSince(ctx context.Context, conversationID uuid.UUID, since time.Time) ([]*models.Message, error)
}

type messageRepository struct {
}

// NewMessageRepository 创建消息仓储
func NewMessageRepository() MessageRepository {
	return &messageRepository{}
}

// Create 创建消息
func (r *messageRepository) Create(ctx context.Context, message *models.Message) error {
	logger.InfofWithCaller("Creating message in conversation %s from user %s", message.ConversationID, message.SenderID)

	message.CreatedAt = time.Now().UTC()

	// 使用数据库函数插入消息
	query := `
		SELECT insert_conversation_message($1, $2, $3, $4) as id
	`

	err := database.GetPool().QueryRow(ctx, query,
		message.ConversationID,
		message.SenderID,
		message.Content,
		message.MsgType,
	).Scan(&message.ID)

	if err != nil {
		logger.ErrorfWithCaller("Failed to create message: %v", err)
	} else {
		logger.InfofWithCaller("Message created successfully: ID=%s", message.ID)
	}

	return err
}

// FindByID 根据ID查找消息
// 注意：由于消息存储在conversation_messages schema的独立表中，此方法不再支持
// 请使用FindByConversationID方法来查找消息
func (r *messageRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Message, error) {
	// 由于消息存储在conversation_messages schema的独立表中，无法直接按ID查找
	// 返回错误提示使用其他方法
	return nil, fmt.Errorf("finding message by ID is not supported. Please use FindByConversationID method instead")
}

// FindByConversationID 根据会话ID查找消息
func (r *messageRepository) FindByConversationID(ctx context.Context, conversationID uuid.UUID, limit, offset int) ([]*models.Message, error) {
	if limit <= 0 {
		limit = 50 // 默认限制
	}

	// 使用数据库函数获取消息
	query := `
		SELECT * FROM get_conversation_messages($1, $2, $3)
	`

	rows, err := database.GetPool().Query(ctx, query, conversationID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*models.Message
	for rows.Next() {
		message := &models.Message{
			ConversationID: conversationID,
		}
		err := rows.Scan(
			&message.ID,
			&message.SenderID,
			&message.Content,
			&message.MsgType,
			&message.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}

	return messages, nil
}

// CountByConversationID 统计会话中的消息数量
func (r *messageRepository) CountByConversationID(ctx context.Context, conversationID uuid.UUID) (int, error) {
	// 使用数据库函数获取消息数量
	query := `
		SELECT get_conversation_message_count($1)
	`

	var count int
	err := database.GetPool().QueryRow(ctx, query, conversationID).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// CountUnreadByConversationID 统计会话中未读消息数量
func (r *messageRepository) CountUnreadByConversationID(ctx context.Context, conversationID, userID uuid.UUID) (int, error) {
	// 这里简化处理，实际应用中需要实现消息已读标记功能
	// 暂时返回总消息数
	return r.CountByConversationID(ctx, conversationID)
}

// FindLastByConversationID 查找会话中的最后一条消息
func (r *messageRepository) FindLastByConversationID(ctx context.Context, conversationID uuid.UUID) (*models.Message, error) {
	// 使用数据库函数获取最后一条消息
	query := `
		SELECT * FROM get_conversation_last_message($1)
	`

	message := &models.Message{
		ConversationID: conversationID,
	}
	err := database.GetPool().QueryRow(ctx, query, conversationID).Scan(
		&message.ID,
		&message.SenderID,
		&message.Content,
		&message.MsgType,
		&message.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return message, nil
}

// FindByConversationIDSince 增量获取会话中的消息（从指定时间之后）
func (r *messageRepository) FindByConversationIDSince(ctx context.Context, conversationID uuid.UUID, since time.Time) ([]*models.Message, error) {
	logger.InfofWithCaller("Finding messages for conversation %s since %v", conversationID, since)

	// 使用数据库函数增量获取消息
	query := `
		SELECT * FROM get_conversation_messages_incremental($1, $2)
	`

	rows, err := database.GetPool().Query(ctx, query, conversationID, since)
	if err != nil {
		logger.ErrorfWithCaller("Failed to query messages: %v", err)
		return nil, err
	}
	defer rows.Close()

	var messages []*models.Message
	for rows.Next() {
		message := &models.Message{
			ConversationID: conversationID,
		}
		err := rows.Scan(
			&message.ID,
			&message.SenderID,
			&message.Content,
			&message.MsgType,
			&message.CreatedAt,
		)
		if err != nil {
			logger.ErrorfWithCaller("Failed to scan message: %v", err)
			return nil, err
		}
		messages = append(messages, message)
	}

	logger.InfofWithCaller("Found %d new messages for conversation %s", len(messages), conversationID)
	return messages, nil
}

// FriendshipRepository 好友关系仓储接口
type FriendshipRepository interface {
	Create(ctx context.Context, friendship *models.Friendship) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Friendship, error)
	FindByUsers(ctx context.Context, userID, friendID uuid.UUID) (*models.Friendship, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Friendship, error)
	FindPendingRequests(ctx context.Context, userID uuid.UUID) ([]*models.Friendship, error)
	FindAllRequests(ctx context.Context, userID uuid.UUID) ([]*models.Friendship, error)
	Update(ctx context.Context, friendship *models.Friendship) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type friendshipRepository struct {
}

// NewFriendshipRepository 创建好友关系仓储
func NewFriendshipRepository() FriendshipRepository {
	return &friendshipRepository{}
}

// Create 创建好友关系
func (r *friendshipRepository) Create(ctx context.Context, friendship *models.Friendship) error {
	friendship.ID = uuid.New()
	friendship.CreatedAt = time.Now().UTC()

	query := `
		INSERT INTO friendships (id, user_id, friend_id, status, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`

	err := pgx.BeginTxFunc(ctx, database.GetPool(), pgx.TxOptions{}, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query,
			friendship.ID,
			friendship.UserID,
			friendship.FriendID,
			friendship.Status,
			friendship.CreatedAt,
		).Scan(&friendship.ID, &friendship.CreatedAt)
	})

	return err
}

// FindByID 根据ID查找好友关系
func (r *friendshipRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Friendship, error) {
	query := `
		SELECT id, user_id, friend_id, status, created_at
		FROM friendships
		WHERE id = $1
	`

	friendship := &models.Friendship{}
	err := database.GetPool().QueryRow(ctx, query, id).Scan(
		&friendship.ID,
		&friendship.UserID,
		&friendship.FriendID,
		&friendship.Status,
		&friendship.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return friendship, nil
}

// FindByUsers 根据两个用户ID查找好友关系
func (r *friendshipRepository) FindByUsers(ctx context.Context, userID, friendID uuid.UUID) (*models.Friendship, error) {
	query := `
		SELECT id, user_id, friend_id, status, created_at
		FROM friendships
		WHERE (user_id = $1 AND friend_id = $2) OR (user_id = $2 AND friend_id = $1)
	`

	friendship := &models.Friendship{}
	err := database.GetPool().QueryRow(ctx, query, userID, friendID).Scan(
		&friendship.ID,
		&friendship.UserID,
		&friendship.FriendID,
		&friendship.Status,
		&friendship.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return friendship, nil
}

// FindByUserID 根据用户ID查找所有好友关系
func (r *friendshipRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Friendship, error) {
	query := `
		SELECT id, user_id, friend_id, status, created_at
		FROM friendships
		WHERE user_id = $1 OR friend_id = $1
		ORDER BY created_at DESC
	`

	rows, err := database.GetPool().Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var friendships []*models.Friendship
	for rows.Next() {
		friendship := &models.Friendship{}
		err := rows.Scan(
			&friendship.ID,
			&friendship.UserID,
			&friendship.FriendID,
			&friendship.Status,
			&friendship.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		friendships = append(friendships, friendship)
	}

	return friendships, nil
}

// Update 更新好友关系
func (r *friendshipRepository) Update(ctx context.Context, friendship *models.Friendship) error {
	query := `
		UPDATE friendships
		SET status = $1
		WHERE id = $2
	`

	_, err := database.GetPool().Exec(ctx, query,
		friendship.Status,
		friendship.ID,
	)

	return err
}

// Delete 删除好友关系
func (r *friendshipRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM friendships
		WHERE id = $1
	`

	_, err := database.GetPool().Exec(ctx, query, id)
	return err
}

// FindPendingRequests 查找用户的待处理好友请求（接收方）
func (r *friendshipRepository) FindPendingRequests(ctx context.Context, userID uuid.UUID) ([]*models.Friendship, error) {
	logger.InfofWithCaller("Finding pending friend requests for user %s", userID)

	query := `
		SELECT id, user_id, friend_id, status, created_at
		FROM friendships
		WHERE friend_id = $1 AND status = 'pending'
		ORDER BY created_at DESC
	`

	rows, err := database.GetPool().Query(ctx, query, userID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to find pending friend requests: %v", err)
		return nil, err
	}
	defer rows.Close()

	var friendships []*models.Friendship
	for rows.Next() {
		friendship := &models.Friendship{}
		err := rows.Scan(
			&friendship.ID,
			&friendship.UserID,
			&friendship.FriendID,
			&friendship.Status,
			&friendship.CreatedAt,
		)
		if err != nil {
			logger.ErrorfWithCaller("Failed to scan friendship: %v", err)
			return nil, err
		}
		friendships = append(friendships, friendship)
	}

	logger.InfofWithCaller("Found %d pending friend requests for user %s", len(friendships), userID)
	return friendships, nil
}

// FindAllRequests 查找用户的所有好友申请记录（包括已发送、已接收、已接受、已拒绝）
func (r *friendshipRepository) FindAllRequests(ctx context.Context, userID uuid.UUID) ([]*models.Friendship, error) {
	logger.InfofWithCaller("Finding all friend requests for user %s", userID)

	query := `
		SELECT id, user_id, friend_id, status, created_at
		FROM friendships
		WHERE user_id = $1 OR friend_id = $1
		ORDER BY created_at DESC
	`

	rows, err := database.GetPool().Query(ctx, query, userID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to find all friend requests: %v", err)
		return nil, err
	}
	defer rows.Close()

	var friendships []*models.Friendship
	for rows.Next() {
		friendship := &models.Friendship{}
		err := rows.Scan(
			&friendship.ID,
			&friendship.UserID,
			&friendship.FriendID,
			&friendship.Status,
			&friendship.CreatedAt,
		)
		if err != nil {
			logger.ErrorfWithCaller("Failed to scan friendship: %v", err)
			return nil, err
		}
		friendships = append(friendships, friendship)
	}

	logger.InfofWithCaller("Found %d friend requests for user %s", len(friendships), userID)
	return friendships, nil
}
