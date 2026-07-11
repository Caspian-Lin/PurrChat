-- 011: 为 bot_call_logs 增加 run_id、消息关联、运行状态、错误类型和 trace 列
-- 回滚: ALTER TABLE bot_call_logs
--   DROP COLUMN IF EXISTS trace, error_type, run_status, workflow_revision,
--   reply_message_id, trigger_message_id, run_id;

ALTER TABLE bot_call_logs
    ADD COLUMN IF NOT EXISTS run_id VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS trigger_message_id UUID,
    ADD COLUMN IF NOT EXISTS reply_message_id UUID,
    ADD COLUMN IF NOT EXISTS workflow_revision INTEGER,
    ADD COLUMN IF NOT EXISTS run_status VARCHAR(20) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS error_type VARCHAR(60) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS trace JSONB;

CREATE INDEX IF NOT EXISTS idx_bot_call_logs_run_id ON bot_call_logs (run_id) WHERE run_id <> '';
CREATE INDEX IF NOT EXISTS idx_bot_call_logs_reply_msg ON bot_call_logs (reply_message_id) WHERE reply_message_id IS NOT NULL;
