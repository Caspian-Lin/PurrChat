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

// EnrollmentRepository 会话成员仓储接口
type EnrollmentRepository interface {
	Create(ctx context.Context, enrollment *models.Enrollment) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Enrollment, error)
	FindByConversationID(ctx context.Context, conversationID uuid.UUID) ([]*models.Enrollment, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Enrollment, error)
	FindByConversationAndUser(ctx context.Context, conversationID, userID uuid.UUID) (*models.Enrollment, error)
	Update(ctx context.Context, enrollment *models.Enrollment) error
	UpdateLastReadAt(ctx context.Context, conversationID, userID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByConversationAndUser(ctx context.Context, conversationID, userID uuid.UUID) error
}

type enrollmentRepository struct {
}

// NewEnrollmentRepository 创建会话成员仓储
func NewEnrollmentRepository() EnrollmentRepository {
	return &enrollmentRepository{}
}

// Create 创建会话成员
func (r *enrollmentRepository) Create(ctx context.Context, enrollment *models.Enrollment) error {
	logger.InfofWithCaller("Creating enrollment for user %s in conversation %s", enrollment.UserID, enrollment.ConversationID)

	enrollment.ID = uuid.New()
	enrollment.JoinedAt = time.Now()

	query := `
		INSERT INTO enrollments (id, conversation_id, user_id, role, joined_at, last_read_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (conversation_id, user_id) DO UPDATE
		SET role = EXCLUDED.role, joined_at = EXCLUDED.joined_at
		RETURNING id, joined_at
	`

	err := pgx.BeginTxFunc(ctx, database.GetPool(), pgx.TxOptions{}, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query,
			enrollment.ID,
			enrollment.ConversationID,
			enrollment.UserID,
			enrollment.Role,
			enrollment.JoinedAt,
			enrollment.LastReadAt,
		).Scan(&enrollment.ID, &enrollment.JoinedAt)
	})

	if err != nil {
		logger.ErrorfWithCaller("Failed to create enrollment: %v", err)
	} else {
		logger.InfofWithCaller("Enrollment created successfully: ID=%s", enrollment.ID)
	}

	return err
}

// FindByID 根据ID查找会话成员
func (r *enrollmentRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Enrollment, error) {
	query := `
		SELECT id, conversation_id, user_id, role, joined_at, last_read_at
		FROM enrollments
		WHERE id = $1
	`

	enrollment := &models.Enrollment{}
	err := database.GetPool().QueryRow(ctx, query, id).Scan(
		&enrollment.ID,
		&enrollment.ConversationID,
		&enrollment.UserID,
		&enrollment.Role,
		&enrollment.JoinedAt,
		&enrollment.LastReadAt,
	)

	if err != nil {
		return nil, err
	}

	return enrollment, nil
}

// FindByConversationID 根据会话ID查找所有成员
func (r *enrollmentRepository) FindByConversationID(ctx context.Context, conversationID uuid.UUID) ([]*models.Enrollment, error) {
	query := `
		SELECT id, conversation_id, user_id, role, joined_at, last_read_at
		FROM enrollments
		WHERE conversation_id = $1
		ORDER BY joined_at ASC
	`

	rows, err := database.GetPool().Query(ctx, query, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var enrollments []*models.Enrollment
	for rows.Next() {
		enrollment := &models.Enrollment{}
		err := rows.Scan(
			&enrollment.ID,
			&enrollment.ConversationID,
			&enrollment.UserID,
			&enrollment.Role,
			&enrollment.JoinedAt,
			&enrollment.LastReadAt,
		)
		if err != nil {
			return nil, err
		}
		enrollments = append(enrollments, enrollment)
	}

	return enrollments, nil
}

// FindByUserID 根据用户ID查找所有会话
func (r *enrollmentRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Enrollment, error) {
	query := `
		SELECT id, conversation_id, user_id, role, joined_at, last_read_at
		FROM enrollments
		WHERE user_id = $1
		ORDER BY joined_at DESC
	`

	rows, err := database.GetPool().Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var enrollments []*models.Enrollment
	for rows.Next() {
		enrollment := &models.Enrollment{}
		err := rows.Scan(
			&enrollment.ID,
			&enrollment.ConversationID,
			&enrollment.UserID,
			&enrollment.Role,
			&enrollment.JoinedAt,
			&enrollment.LastReadAt,
		)
		if err != nil {
			return nil, err
		}
		enrollments = append(enrollments, enrollment)
	}

	return enrollments, nil
}

// FindByConversationAndUser 根据会话ID和用户ID查找成员
func (r *enrollmentRepository) FindByConversationAndUser(ctx context.Context, conversationID, userID uuid.UUID) (*models.Enrollment, error) {
	query := `
		SELECT id, conversation_id, user_id, role, joined_at, last_read_at
		FROM enrollments
		WHERE conversation_id = $1 AND user_id = $2
	`

	enrollment := &models.Enrollment{}
	err := database.GetPool().QueryRow(ctx, query, conversationID, userID).Scan(
		&enrollment.ID,
		&enrollment.ConversationID,
		&enrollment.UserID,
		&enrollment.Role,
		&enrollment.JoinedAt,
		&enrollment.LastReadAt,
	)

	if err != nil {
		return nil, err
	}

	return enrollment, nil
}

// Update 更新会话成员
func (r *enrollmentRepository) Update(ctx context.Context, enrollment *models.Enrollment) error {
	query := `
		UPDATE enrollments
		SET role = $1, last_read_at = $2
		WHERE id = $3
	`

	_, err := database.GetPool().Exec(ctx, query,
		enrollment.Role,
		enrollment.LastReadAt,
		enrollment.ID,
	)

	return err
}

// UpdateLastReadAt 更新最后阅读时间
func (r *enrollmentRepository) UpdateLastReadAt(ctx context.Context, conversationID, userID uuid.UUID) error {
	now := time.Now()
	query := `
		UPDATE enrollments
		SET last_read_at = $1
		WHERE conversation_id = $2 AND user_id = $3
	`

	_, err := database.GetPool().Exec(ctx, query, now, conversationID, userID)
	return err
}

// Delete 删除会话成员
func (r *enrollmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM enrollments
		WHERE id = $1
	`

	_, err := database.GetPool().Exec(ctx, query, id)
	return err
}

// DeleteByConversationAndUser 根据会话ID和用户ID删除成员
func (r *enrollmentRepository) DeleteByConversationAndUser(ctx context.Context, conversationID, userID uuid.UUID) error {
	query := `
		DELETE FROM enrollments
		WHERE conversation_id = $1 AND user_id = $2
	`

	_, err := database.GetPool().Exec(ctx, query, conversationID, userID)
	return err
}
