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

// ConversationRepository 会话仓储接口
type ConversationRepository interface {
	Create(ctx context.Context, conversation *models.Conversation) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Conversation, error)
	FindByUsers(ctx context.Context, user1ID, user2ID uuid.UUID) (*models.Conversation, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Conversation, error)
	Update(ctx context.Context, conversation *models.Conversation) error
	UpdateRequestStatus(ctx context.Context, conversationID uuid.UUID, status models.RequestStatus) error
	MarkAsPendingRequest(ctx context.Context, conversationID uuid.UUID) error
}

type conversationRepository struct {
}

// NewConversationRepository 创建会话仓储
func NewConversationRepository() ConversationRepository {
	return &conversationRepository{}
}

// Create 创建会话
func (r *conversationRepository) Create(ctx context.Context, conversation *models.Conversation) error {
	logger.InfofWithCaller("Creating conversation between %s and %s", conversation.User1ID, conversation.User2ID)

	conversation.ID = uuid.New()
	conversation.CreatedAt = time.Now()
	conversation.UpdatedAt = time.Now()

	query := `
		INSERT INTO conversations (id, conversation_type, user1_id, user2_id, has_pending_request, request_status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`

	err := pgx.BeginTxFunc(ctx, database.GetPool(), pgx.TxOptions{}, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query,
			conversation.ID,
			conversation.ConversationType,
			conversation.User1ID,
			conversation.User2ID,
			conversation.HasPendingRequest,
			conversation.RequestStatus,
			conversation.CreatedAt,
			conversation.UpdatedAt,
		).Scan(&conversation.ID, &conversation.CreatedAt, &conversation.UpdatedAt)
	})

	if err != nil {
		logger.ErrorfWithCaller("Failed to create conversation: %v", err)
	} else {
		logger.InfofWithCaller("Conversation created successfully: ID=%s", conversation.ID)
	}

	return err
}

// FindByID 根据ID查找会话
func (r *conversationRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Conversation, error) {
	query := `
		SELECT id, conversation_type, user1_id, user2_id, has_pending_request, request_status, created_at, updated_at
		FROM conversations
		WHERE id = $1
	`

	conversation := &models.Conversation{}
	err := database.GetPool().QueryRow(ctx, query, id).Scan(
		&conversation.ID,
		&conversation.ConversationType,
		&conversation.User1ID,
		&conversation.User2ID,
		&conversation.HasPendingRequest,
		&conversation.RequestStatus,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return conversation, nil
}

// FindByUsers 根据两个用户ID查找会话
func (r *conversationRepository) FindByUsers(ctx context.Context, user1ID, user2ID uuid.UUID) (*models.Conversation, error) {
	query := `
		SELECT id, conversation_type, user1_id, user2_id, has_pending_request, request_status, created_at, updated_at
		FROM conversations
		WHERE (user1_id = $1 AND user2_id = $2) OR (user1_id = $2 AND user2_id = $1)
	`

	conversation := &models.Conversation{}
	err := database.GetPool().QueryRow(ctx, query, user1ID, user2ID).Scan(
		&conversation.ID,
		&conversation.ConversationType,
		&conversation.User1ID,
		&conversation.User2ID,
		&conversation.HasPendingRequest,
		&conversation.RequestStatus,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return conversation, nil
}

// FindByUserID 根据用户ID查找所有会话
func (r *conversationRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Conversation, error) {
	query := `
		SELECT id, conversation_type, user1_id, user2_id, has_pending_request, request_status, created_at, updated_at
		FROM conversations
		WHERE user1_id = $1 OR user2_id = $1
		ORDER BY
			has_pending_request DESC,
			updated_at DESC
	`

	rows, err := database.GetPool().Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conversations []*models.Conversation
	for rows.Next() {
		conversation := &models.Conversation{}
		err := rows.Scan(
			&conversation.ID,
			&conversation.ConversationType,
			&conversation.User1ID,
			&conversation.User2ID,
			&conversation.HasPendingRequest,
			&conversation.RequestStatus,
			&conversation.CreatedAt,
			&conversation.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		conversations = append(conversations, conversation)
	}

	return conversations, nil
}

// Update 更新会话
func (r *conversationRepository) Update(ctx context.Context, conversation *models.Conversation) error {
	query := `
		UPDATE conversations
		SET conversation_type = $1, has_pending_request = $2, request_status = $3, updated_at = $4
		WHERE id = $5
	`

	_, err := database.GetPool().Exec(ctx, query,
		conversation.ConversationType,
		conversation.HasPendingRequest,
		conversation.RequestStatus,
		time.Now(),
		conversation.ID,
	)

	return err
}

// UpdateRequestStatus 更新请求状态
func (r *conversationRepository) UpdateRequestStatus(ctx context.Context, conversationID uuid.UUID, status models.RequestStatus) error {
	query := `
		UPDATE conversations
		SET request_status = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := database.GetPool().Exec(ctx, query,
		status,
		time.Now(),
		conversationID,
	)

	return err
}

// MarkAsPendingRequest 标记为有待处理请求
func (r *conversationRepository) MarkAsPendingRequest(ctx context.Context, conversationID uuid.UUID) error {
	query := `
		UPDATE conversations
		SET has_pending_request = true, request_status = 'pending', updated_at = $1
		WHERE id = $2
	`

	_, err := database.GetPool().Exec(ctx, query,
		time.Now(),
		conversationID,
	)

	return err
}
