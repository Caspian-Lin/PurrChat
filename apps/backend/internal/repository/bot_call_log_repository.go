package repository

import (
	"context"
	"fmt"

	"purr-chat-server/internal/models"
	"purr-chat-server/pkg/database"

	"github.com/google/uuid"
)

// BotCallLogRepository Bot 调用日志数据访问接口
type BotCallLogRepository interface {
	Create(ctx context.Context, log *models.BotCallLog) error
	FindAllByBotID(ctx context.Context, botID uuid.UUID, limit, offset int) ([]*models.BotCallLog, error)
	CountByBotID(ctx context.Context, botID uuid.UUID) (int, error)
	UpdateReplyMessageID(ctx context.Context, logID uuid.UUID, replyMessageID uuid.UUID) error
	FindByRunID(ctx context.Context, runID string) (*models.BotCallLog, error)
}

type botCallLogRepository struct{}

// NewBotCallLogRepository 创建 Bot 调用日志仓储
func NewBotCallLogRepository() BotCallLogRepository {
	return &botCallLogRepository{}
}

func (r *botCallLogRepository) Create(ctx context.Context, log *models.BotCallLog) error {
	query := `
		INSERT INTO bot_call_logs (id, bot_id, conversation_id, sender_id, sender_name,
			trigger_message, reply_content, mechanism_id, mechanism_name,
			reply_type, execution_path, success, error_message, duration_ms, created_at,
			run_id, trigger_message_id, reply_message_id, workflow_revision, run_status, error_type, trace)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21, $22)
	`
	_, err := database.GetPool().Exec(ctx, query,
		log.ID, log.BotID, log.ConversationID, log.SenderID, log.SenderName,
		log.TriggerMessage, log.ReplyContent, log.MechanismID, log.MechanismName,
		log.ReplyType, log.ExecutionPath, log.Success, log.ErrorMessage, log.DurationMs, log.CreatedAt,
		log.RunID, log.TriggerMessageID, log.ReplyMessageID, log.WorkflowRevision, log.RunStatus, log.ErrorType, log.Trace,
	)
	return err
}

func (r *botCallLogRepository) FindAllByBotID(ctx context.Context, botID uuid.UUID, limit, offset int) ([]*models.BotCallLog, error) {
	query := `
		SELECT cl.id, cl.bot_id, cl.conversation_id, cl.sender_id, cl.sender_name,
		       cl.trigger_message, cl.reply_content, cl.mechanism_id, cl.mechanism_name,
		       cl.reply_type, cl.execution_path, cl.success, cl.error_message, cl.duration_ms, cl.created_at,
		       COALESCE(c.name, '') AS conversation_name,
		       cl.run_id, cl.trigger_message_id, cl.reply_message_id, cl.workflow_revision,
		       cl.run_status, cl.error_type, cl.trace
		FROM bot_call_logs cl
		LEFT JOIN conversations c ON cl.conversation_id = c.id
		WHERE cl.bot_id = $1
		ORDER BY cl.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := database.GetPool().Query(ctx, query, botID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Keep the API collection contract stable: an empty, non-nil slice is
	// encoded as [] instead of null by encoding/json.
	logs := make([]*models.BotCallLog, 0)
	for rows.Next() {
		log := &models.BotCallLog{}
		err := rows.Scan(
			&log.ID, &log.BotID, &log.ConversationID, &log.SenderID, &log.SenderName,
			&log.TriggerMessage, &log.ReplyContent, &log.MechanismID, &log.MechanismName,
			&log.ReplyType, &log.ExecutionPath, &log.Success, &log.ErrorMessage, &log.DurationMs, &log.CreatedAt,
			&log.ConversationName,
			&log.RunID, &log.TriggerMessageID, &log.ReplyMessageID, &log.WorkflowRevision,
			&log.RunStatus, &log.ErrorType, &log.Trace,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan bot call log: %w", err)
		}
		logs = append(logs, log)
	}

	return logs, nil
}

func (r *botCallLogRepository) CountByBotID(ctx context.Context, botID uuid.UUID) (int, error) {
	var count int
	err := database.GetPool().QueryRow(ctx, "SELECT COUNT(*) FROM bot_call_logs WHERE bot_id = $1", botID).Scan(&count)
	return count, err
}

func (r *botCallLogRepository) UpdateReplyMessageID(ctx context.Context, logID uuid.UUID, replyMessageID uuid.UUID) error {
	_, err := database.GetPool().Exec(ctx,
		"UPDATE bot_call_logs SET reply_message_id = $1 WHERE id = $2",
		replyMessageID, logID,
	)
	return err
}

func (r *botCallLogRepository) FindByRunID(ctx context.Context, runID string) (*models.BotCallLog, error) {
	query := `
		SELECT cl.id, cl.bot_id, cl.conversation_id, cl.sender_id, cl.sender_name,
		       cl.trigger_message, cl.reply_content, cl.mechanism_id, cl.mechanism_name,
		       cl.reply_type, cl.execution_path, cl.success, cl.error_message, cl.duration_ms, cl.created_at,
		       COALESCE(c.name, '') AS conversation_name,
		       cl.run_id, cl.trigger_message_id, cl.reply_message_id, cl.workflow_revision,
		       cl.run_status, cl.error_type, cl.trace
		FROM bot_call_logs cl
		LEFT JOIN conversations c ON cl.conversation_id = c.id
		WHERE cl.run_id = $1
		LIMIT 1
	`
	log := &models.BotCallLog{}
	err := database.GetPool().QueryRow(ctx, query, runID).Scan(
		&log.ID, &log.BotID, &log.ConversationID, &log.SenderID, &log.SenderName,
		&log.TriggerMessage, &log.ReplyContent, &log.MechanismID, &log.MechanismName,
		&log.ReplyType, &log.ExecutionPath, &log.Success, &log.ErrorMessage, &log.DurationMs, &log.CreatedAt,
		&log.ConversationName,
		&log.RunID, &log.TriggerMessageID, &log.ReplyMessageID, &log.WorkflowRevision,
		&log.RunStatus, &log.ErrorType, &log.Trace,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to find call log by run_id: %w", err)
	}
	return log, nil
}
