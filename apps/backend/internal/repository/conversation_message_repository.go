package repository

import (
	"context"
	"time"

	"purr-chat-server/internal/models"
	"purr-chat-server/pkg/database"
	"purr-chat-server/pkg/logger"

	"github.com/google/uuid"
)

// ConversationMessageRepository 会话消息表管理仓储接口
type ConversationMessageRepository interface {
	// CreateMessageTable 为会话创建消息表
	CreateMessageTable(ctx context.Context, conversationID uuid.UUID) error
	// DropMessageTable 删除会话消息表
	DropMessageTable(ctx context.Context, conversationID uuid.UUID) error
	// InsertMessage 向会话消息表插入消息
	InsertMessage(ctx context.Context, conversationID uuid.UUID, message *models.Message) error
	// FindMessages 从会话消息表获取消息
	FindMessages(ctx context.Context, conversationID uuid.UUID, limit, offset int) ([]*models.Message, error)
	// FindAllMessages 从会话消息表获取所有消息
	FindAllMessages(ctx context.Context, conversationID uuid.UUID) ([]*models.Message, error)
	// CountMessages 统计会话消息数量
	CountMessages(ctx context.Context, conversationID uuid.UUID) (int, error)
	// FindLastMessage 查找会话中的最后一条消息
	FindLastMessage(ctx context.Context, conversationID uuid.UUID) (*models.Message, error)
	// FindByConversationIDSince 增量获取会话中的消息（从指定时间之后）
	FindByConversationIDSince(ctx context.Context, conversationID uuid.UUID, since time.Time) ([]*models.Message, error)
}

type conversationMessageRepository struct {
}

// NewConversationMessageRepository 创建会话消息表管理仓储
func NewConversationMessageRepository() ConversationMessageRepository {
	return &conversationMessageRepository{}
}

// CreateMessageTable 为会话创建消息表
func (r *conversationMessageRepository) CreateMessageTable(ctx context.Context, conversationID uuid.UUID) error {
	logger.InfofWithCaller("Creating message table for conversation %s", conversationID)

	query := `
        SELECT create_conversation_message_table($1)
    `

	_, err := database.GetPool().Exec(ctx, query, conversationID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to create message table for conversation %s: %v", conversationID, err)
	} else {
		logger.InfofWithCaller("Message table created successfully for conversation %s", conversationID)
	}

	return err
}

// DropMessageTable 删除会话消息表
func (r *conversationMessageRepository) DropMessageTable(ctx context.Context, conversationID uuid.UUID) error {
	logger.InfofWithCaller("Dropping message table for conversation %s", conversationID)

	query := `
        SELECT drop_conversation_message_table($1)
    `

	_, err := database.GetPool().Exec(ctx, query, conversationID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to drop message table for conversation %s: %v", conversationID, err)
	} else {
		logger.InfofWithCaller("Message table dropped successfully for conversation %s", conversationID)
	}

	return err
}

// InsertMessage 向会话消息表插入消息
func (r *conversationMessageRepository) InsertMessage(ctx context.Context, conversationID uuid.UUID, message *models.Message) error {
	logger.InfofWithCaller("Inserting message into conversation %s", conversationID)

	message.ID = uuid.New()
	message.ConversationID = conversationID
	message.CreatedAt = time.Now().UTC()

	query := `
        SELECT insert_conversation_message($1, $2, $3, $4, $5, $6)
    `

	err := database.GetPool().QueryRow(ctx, query,
		conversationID,
		message.SenderID,
		message.Content,
		message.MsgType,
		message.BotID,
		message.BotName,
	).Scan(&message.ID)

	if err != nil {
		logger.ErrorfWithCaller("Failed to insert message into conversation %s: %v", conversationID, err)
	} else {
		logger.InfofWithCaller("Message inserted successfully: ID=%s, ConversationID=%s", message.ID, conversationID)
	}

	return err
}

// FindMessages 从会话消息表获取消息
func (r *conversationMessageRepository) FindMessages(ctx context.Context, conversationID uuid.UUID, limit, offset int) ([]*models.Message, error) {
	query := `
        SELECT * FROM get_conversation_messages($1, $2, $3)
    `

	if limit <= 0 {
		limit = 50 // 默认限制
	}

	rows, err := database.GetPool().Query(ctx, query, conversationID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*models.Message
	for rows.Next() {
		message := &models.Message{}
		err := rows.Scan(
			&message.ID,
			&message.SenderID,
			&message.Content,
			&message.MsgType,
			&message.CreatedAt,
			&message.BotID,
			&message.BotName,
		)
		if err != nil {
			return nil, err
		}
		// 设置conversation_id
		message.ConversationID = conversationID
		messages = append(messages, message)
	}

	return messages, nil
}

// FindAllMessages 从会话消息表获取所有消息
func (r *conversationMessageRepository) FindAllMessages(ctx context.Context, conversationID uuid.UUID) ([]*models.Message, error) {
	// 使用PostgreSQL函数获取消息，避免SQL注入问题
	query := `
        SELECT id, sender_id, content, msg_type, created_at, bot_id, bot_name
        FROM get_conversation_messages($1, 100000, 0)
        ORDER BY created_at ASC
    `

	rows, err := database.GetPool().Query(ctx, query, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*models.Message
	for rows.Next() {
		message := &models.Message{}
		err := rows.Scan(
			&message.ID,
			&message.SenderID,
			&message.Content,
			&message.MsgType,
			&message.CreatedAt,
			&message.BotID,
			&message.BotName,
		)
		if err != nil {
			return nil, err
		}
		// 设置conversation_id
		message.ConversationID = conversationID
		messages = append(messages, message)
	}

	return messages, nil
}

// CountMessages 统计会话消息数量
func (r *conversationMessageRepository) CountMessages(ctx context.Context, conversationID uuid.UUID) (int, error) {
	// 使用PostgreSQL函数获取消息数量
	query := `
        SELECT COUNT(*)
        FROM get_conversation_messages($1, 100000, 0)
    `

	var count int
	err := database.GetPool().QueryRow(ctx, query, conversationID).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// FindLastMessage 查找会话中的最后一条消息
func (r *conversationMessageRepository) FindLastMessage(ctx context.Context, conversationID uuid.UUID) (*models.Message, error) {
	// 使用PostgreSQL函数获取最后一条消息
	query := `
        SELECT id, sender_id, content, msg_type, created_at, bot_id, bot_name
        FROM get_conversation_messages($1, 1, 0)
        ORDER BY created_at DESC
        LIMIT 1
    `

	message := &models.Message{}
	err := database.GetPool().QueryRow(ctx, query, conversationID).Scan(
		&message.ID,
		&message.SenderID,
		&message.Content,
		&message.MsgType,
		&message.CreatedAt,
		&message.BotID,
		&message.BotName,
	)

	if err != nil {
		return nil, err
	}

	message.ConversationID = conversationID
	return message, nil
}

// FindByConversationIDSince 增量获取会话中的消息（从指定时间之后）
func (r *conversationMessageRepository) FindByConversationIDSince(ctx context.Context, conversationID uuid.UUID, since time.Time) ([]*models.Message, error) {
	logger.InfofWithCaller("Finding messages for conversation %s since %v", conversationID, since)

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
		message := &models.Message{}
		err := rows.Scan(
			&message.ID,
			&message.SenderID,
			&message.Content,
			&message.MsgType,
			&message.CreatedAt,
			&message.BotID,
			&message.BotName,
		)
		if err != nil {
			logger.ErrorfWithCaller("Failed to scan message: %v", err)
			return nil, err
		}
		message.ConversationID = conversationID
		messages = append(messages, message)
	}

	logger.InfofWithCaller("Found %d new messages for conversation %s", len(messages), conversationID)
	return messages, nil
}
