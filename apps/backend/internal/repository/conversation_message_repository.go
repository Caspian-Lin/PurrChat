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

// GetConversationParticipants 获取会话参与者ID列表
func (r *conversationMessageRepository) GetConversationParticipants(ctx context.Context, conversationID uuid.UUID) ([]uuid.UUID, error) {
	query := `
        SELECT user_id
        FROM enrollments
        WHERE conversation_id = $1
    `

	rows, err := database.GetPool().Query(ctx, query, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []uuid.UUID
	for rows.Next() {
		var userID uuid.UUID
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		participants = append(participants, userID)
	}

	return participants, nil
}

// BroadcastMessage 广播消息给会话中的所有参与者（不包括发送者）
// 这个函数返回应该接收消息的用户ID列表
func (r *conversationMessageRepository) BroadcastMessage(ctx context.Context, conversationID, senderID uuid.UUID) ([]uuid.UUID, error) {
	query := `
        SELECT user_id
        FROM enrollments
        WHERE conversation_id = $1 AND user_id != $2
    `

	rows, err := database.GetPool().Query(ctx, query, conversationID, senderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recipients []uuid.UUID
	for rows.Next() {
		var userID uuid.UUID
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		recipients = append(recipients, userID)
	}

	return recipients, nil
}
