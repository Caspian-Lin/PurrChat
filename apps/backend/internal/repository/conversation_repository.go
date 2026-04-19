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
}

type conversationRepository struct {
}

// NewConversationRepository 创建会话仓储
func NewConversationRepository() ConversationRepository {
    return &conversationRepository{}
}

// Create 创建会话
func (r *conversationRepository) Create(ctx context.Context, conversation *models.Conversation) error {
    logger.InfofWithCaller("Creating conversation: %s", conversation.Name)

    conversation.ID = uuid.New()
    conversation.CreatedAt = time.Now().UTC()
    conversation.UpdatedAt = time.Now().UTC()

    err := pgx.BeginTxFunc(ctx, database.GetPool(), pgx.TxOptions{}, func(tx pgx.Tx) error {
        // 插入会话
        query := `
            INSERT INTO conversations (id, conversation_type, name, created_by, created_at, updated_at)
            VALUES ($1, $2, $3, $4, $5, $6)
            RETURNING id, created_at, updated_at
        `

        err := tx.QueryRow(ctx, query,
            conversation.ID,
            conversation.ConversationType,
            conversation.Name,
            conversation.CreatedBy,
            conversation.CreatedAt,
            conversation.UpdatedAt,
        ).Scan(&conversation.ID, &conversation.CreatedAt, &conversation.UpdatedAt)

        if err != nil {
            return err
        }

        // 创建会话的消息表
        _, err = tx.Exec(ctx, "SELECT create_conversation_message_table($1)", conversation.ID)
        if err != nil {
            logger.ErrorfWithCaller("Failed to create conversation message table: %v", err)
            return err
        }

        return nil
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
        SELECT id, conversation_type, name, created_by, created_at, updated_at
        FROM conversations
        WHERE id = $1
    `

    conversation := &models.Conversation{}
    err := database.GetPool().QueryRow(ctx, query, id).Scan(
        &conversation.ID,
        &conversation.ConversationType,
        &conversation.Name,
        &conversation.CreatedBy,
        &conversation.CreatedAt,
        &conversation.UpdatedAt,
    )

    if err != nil {
        return nil, err
    }

    return conversation, nil
}

// FindByUsers 根据两个用户ID查找会话（通过enrollment表）
func (r *conversationRepository) FindByUsers(ctx context.Context, user1ID, user2ID uuid.UUID) (*models.Conversation, error) {
    query := `
        SELECT DISTINCT c.id, c.conversation_type, c.name, c.created_by, c.created_at, c.updated_at
        FROM conversations c
        INNER JOIN enrollments e1 ON c.id = e1.conversation_id AND e1.user_id = $1
        INNER JOIN enrollments e2 ON c.id = e2.conversation_id AND e2.user_id = $2
        WHERE c.conversation_type = 'direct'
        LIMIT 1
    `

    conversation := &models.Conversation{}
    err := database.GetPool().QueryRow(ctx, query, user1ID, user2ID).Scan(
        &conversation.ID,
        &conversation.ConversationType,
        &conversation.Name,
        &conversation.CreatedBy,
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
        SELECT DISTINCT c.id, c.conversation_type, c.name, c.created_by, c.created_at, c.updated_at
        FROM conversations c
        INNER JOIN enrollments e ON c.id = e.conversation_id
        WHERE e.user_id = $1
        ORDER BY c.updated_at DESC
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
            &conversation.Name,
            &conversation.CreatedBy,
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
        SET conversation_type = $1, name = $2, updated_at = $3
        WHERE id = $4
    `

    _, err := database.GetPool().Exec(ctx, query,
        conversation.ConversationType,
        conversation.Name,
        time.Now().UTC(),
        conversation.ID,
    )

    return err
}
