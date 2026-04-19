-- 007: 为消息表添加 bot_id 和 bot_name 字段
-- 用于标识 Bot 发送的消息，避免通过 sender_id 查询 System 用户来反推

-- 0. 先删除需要改变返回类型/签名的函数
DROP FUNCTION IF EXISTS insert_conversation_message(UUID, UUID, TEXT, VARCHAR(20));
DROP FUNCTION IF EXISTS get_conversation_messages(UUID, INT, INT);
DROP FUNCTION IF EXISTS get_conversation_messages_incremental(UUID, TIMESTAMP);
DROP FUNCTION IF EXISTS get_conversation_last_message(UUID);

-- 1. 为所有已有的会话消息表添加列
DO $$
DECLARE
    tbl RECORD;
BEGIN
    FOR tbl IN
        SELECT table_name
        FROM information_schema.tables
        WHERE table_schema = 'conversation_messages'
          AND table_type = 'BASE TABLE'
    LOOP
        BEGIN
            EXECUTE format('ALTER TABLE conversation_messages.%I ADD COLUMN IF NOT EXISTS bot_id UUID', tbl.table_name);
            EXECUTE format('ALTER TABLE conversation_messages.%I ADD COLUMN IF NOT EXISTS bot_name VARCHAR(100)', tbl.table_name);
        EXCEPTION WHEN OTHERS THEN
            RAISE NOTICE 'Skipped %: %', tbl.table_name, SQLERRM;
        END;
    END LOOP;
END;
$$;

-- 2. 更新 create_conversation_message_table 函数（新会话）
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
            bot_id UUID,
            bot_name VARCHAR(100),
            CONSTRAINT check_msg_type CHECK (msg_type IN (''text'', ''image'', ''file'', ''system''))
        )',
    table_name
    );

    idx_sender_name := 'idx_' || table_name || '_sender_id';
    idx_created_at_name := 'idx_' || table_name || '_created_at';

    EXECUTE format('CREATE INDEX IF NOT EXISTS %I ON conversation_messages.%I(sender_id)', idx_sender_name, table_name);
    EXECUTE format('CREATE INDEX IF NOT EXISTS %I ON conversation_messages.%I(created_at DESC)', idx_created_at_name, table_name);
END;
$$ LANGUAGE plpgsql;

-- 3. 更新 insert_conversation_message 函数
CREATE OR REPLACE FUNCTION insert_conversation_message(
    conversation_uuid UUID,
    sender_uuid UUID,
    msg_content TEXT,
    msg_type VARCHAR(20),
    bot_id UUID DEFAULT NULL,
    bot_name VARCHAR(100) DEFAULT NULL
)
RETURNS UUID AS $$
DECLARE
    new_message_id UUID;
    table_name TEXT;
BEGIN
    table_name := replace(conversation_uuid::TEXT, '-', '_');
    EXECUTE format('
        INSERT INTO conversation_messages.%I (sender_id, content, msg_type, bot_id, bot_name)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    ', table_name)
    INTO new_message_id
    USING sender_uuid, msg_content, msg_type, bot_id, bot_name;

    RETURN new_message_id;
END;
$$ LANGUAGE plpgsql;

-- 4. 更新 get_conversation_messages 函数
CREATE OR REPLACE FUNCTION get_conversation_messages(
    conversation_uuid UUID,
    msg_limit INT DEFAULT 50,
    msg_offset INT DEFAULT 0
)
RETURNS TABLE (
    id UUID,
    sender_id UUID,
    content TEXT,
    msg_type VARCHAR(20),
    created_at TIMESTAMP,
    bot_id UUID,
    bot_name VARCHAR(100)
) AS $$
DECLARE
    table_name TEXT;
BEGIN
    table_name := replace(conversation_uuid::TEXT, '-', '_');
    RETURN QUERY EXECUTE format('
        SELECT id, sender_id, content, msg_type, created_at, bot_id, bot_name
        FROM conversation_messages.%I
        ORDER BY created_at DESC
        LIMIT $1 OFFSET $2
    ', table_name)
    USING msg_limit, msg_offset;
END;
$$ LANGUAGE plpgsql;

-- 5. 更新 get_conversation_messages_incremental 函数
CREATE OR REPLACE FUNCTION get_conversation_messages_incremental(
    conversation_uuid UUID,
    since_timestamp TIMESTAMP
)
RETURNS TABLE (
    id UUID,
    sender_id UUID,
    content TEXT,
    msg_type VARCHAR(20),
    created_at TIMESTAMP,
    bot_id UUID,
    bot_name VARCHAR(100)
) AS $$
DECLARE
    table_name TEXT;
BEGIN
    table_name := replace(conversation_uuid::TEXT, '-', '_');
    RETURN QUERY EXECUTE format('
        SELECT id, sender_id, content, msg_type, created_at, bot_id, bot_name
        FROM conversation_messages.%I
        WHERE created_at > $1
        ORDER BY created_at ASC
    ', table_name)
    USING since_timestamp;
END;
$$ LANGUAGE plpgsql;

-- 6. 更新 get_conversation_last_message 函数
CREATE OR REPLACE FUNCTION get_conversation_last_message(conversation_uuid UUID)
RETURNS TABLE (
    id UUID,
    sender_id UUID,
    content TEXT,
    msg_type VARCHAR(20),
    created_at TIMESTAMP,
    bot_id UUID,
    bot_name VARCHAR(100)
) AS $$
DECLARE
    table_name TEXT;
BEGIN
    table_name := replace(conversation_uuid::TEXT, '-', '_');
    RETURN QUERY EXECUTE format('
        SELECT id, sender_id, content, msg_type, created_at, bot_id, bot_name
        FROM conversation_messages.%I
        ORDER BY created_at DESC
        LIMIT 1
    ', table_name);
END;
$$ LANGUAGE plpgsql;
