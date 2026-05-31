CREATE TABLE IF NOT EXISTS bot_call_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id UUID NOT NULL,
    sender_name VARCHAR(40) NOT NULL DEFAULT '',
    trigger_message TEXT NOT NULL,
    reply_content TEXT,
    mechanism_id VARCHAR(100) NOT NULL DEFAULT '',
    mechanism_name VARCHAR(100) NOT NULL DEFAULT '',
    reply_type VARCHAR(20) NOT NULL DEFAULT '',
    execution_path VARCHAR(10) NOT NULL DEFAULT 'ts',
    success BOOLEAN NOT NULL DEFAULT TRUE,
    error_message TEXT,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_bot_call_logs_bot_id ON bot_call_logs (bot_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_bot_call_logs_bot_conv ON bot_call_logs (bot_id, conversation_id, created_at DESC);
