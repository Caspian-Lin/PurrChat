-- 003: 添加 'file' 消息类型支持
-- 更新 create_conversation_message_table 函数和已有消息表的 CHECK 约束

-- 1. 更新函数定义（影响新创建的会话消息表）
CREATE OR REPLACE FUNCTION create_conversation_message_table(conversation_uuid UUID)
RETURNS VOID AS $$
DECLARE
    table_name TEXT;
    idx_sender_name TEXT;
    idx_created_at_name TEXT;
BEGIN
    table_name := replace(conversation_uuid::TEXT, '-', '_');

    EXECUTE format('
        CREATE TABLE IF NOT EXISTS conversation_messages.%I (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
            content TEXT NOT NULL,
            msg_type VARCHAR(20) NOT NULL DEFAULT ''text'',
            created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
            CONSTRAINT check_msg_type CHECK (msg_type IN (''text'', ''image'', ''file''))
        )',
    table_name
    );

    idx_sender_name := 'idx_' || table_name || '_sender_id';
    idx_created_at_name := 'idx_' || table_name || '_created_at';

    EXECUTE format('
        CREATE INDEX IF NOT EXISTS %I ON conversation_messages.%I(sender_id)',
    idx_sender_name, table_name
    );

    EXECUTE format('
        CREATE INDEX IF NOT EXISTS %I ON conversation_messages.%I(created_at DESC)',
    idx_created_at_name, table_name
    );
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION create_conversation_message_table IS '为指定会话创建消息表';

-- 2. 更新已有消息表的 CHECK 约束（影响已存在的会话消息表）
DO $$
DECLARE
    tbl RECORD;
BEGIN
    FOR tbl IN
        SELECT table_name, table_schema
        FROM information_schema.tables
        WHERE table_schema = 'conversation_messages' AND table_type = 'BASE TABLE'
    LOOP
        EXECUTE format('
            ALTER TABLE %I.%I DROP CONSTRAINT IF EXISTS check_msg_type;
            ALTER TABLE %I.%I ADD CONSTRAINT check_msg_type
                CHECK (msg_type IN (''text'', ''image'', ''file''));
        ', tbl.table_schema, tbl.table_name, tbl.table_schema, tbl.table_name);
    END LOOP;
END;
$$;
