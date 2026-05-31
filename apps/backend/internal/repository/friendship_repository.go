package repository

import (
	"context"
	"time"

	"purr-chat-server/internal/models"
	"purr-chat-server/pkg/database"
	"purr-chat-server/pkg/logger"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// FriendshipRepository 好友关系仓储接口
type FriendshipRepository interface {
	Create(ctx context.Context, friendship *models.Friendship) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Friendship, error)
	FindByUsers(ctx context.Context, userID, friendID uuid.UUID) (*models.Friendship, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Friendship, error)
	FindPendingRequests(ctx context.Context, userID uuid.UUID) ([]*models.Friendship, error)
	FindAllRequests(ctx context.Context, userID uuid.UUID) ([]*models.Friendship, error)
	CountSentSince(ctx context.Context, userID uuid.UUID, since time.Time) (int, error)
	Update(ctx context.Context, friendship *models.Friendship) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type friendshipRepository struct{}

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

// CountSentSince 统计用户在指定时间后发送的好友请求数量
func (r *friendshipRepository) CountSentSince(ctx context.Context, userID uuid.UUID, since time.Time) (int, error) {
	var count int
	err := database.GetPool().QueryRow(ctx,
		"SELECT COUNT(*) FROM friendships WHERE user_id = $1 AND created_at > $2",
		userID, since,
	).Scan(&count)
	return count, err
}
