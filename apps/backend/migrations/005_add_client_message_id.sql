-- 005: 为消息表添加 client_message_id 幂等性列
-- 用户发送消息时携带 client_message_id，服务端通过此字段实现幂等去重。
-- 当网络异常导致客户端重发时，服务端返回已有消息而非创建重复消息。

-- 1. 修改建表函数：新创建的分表自动包含 client_message_id 列
CREATE OR REPLACE FUNCTION create_conversation_message_table(conversation_uuid UUID)
RETURNS VOID AS $$
DECLARE
    table_name TEXT;
    idx_sender_name TEXT;
    idx_created_at_name TEXT;
    idx_client_msg_id_name TEXT;
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
            client_message_id VARCHAR(255),
            CONSTRAINT check_msg_type CHECK (msg_type IN (''text'', ''image'', ''file'', ''system''))
        )',
    table_name
    );

    idx_sender_name := 'idx_' || table_name || '_sender_id';
    idx_created_at_name := 'idx_' || table_name || '_created_at';
    idx_client_msg_id_name := 'idx_' || table_name || '_client_message_id';

    EXECUTE format('CREATE INDEX IF NOT EXISTS %I ON conversation_messages.%I(sender_id)', idx_sender_name, table_name);
    EXECUTE format('CREATE INDEX IF NOT EXISTS %I ON conversation_messages.%I(created_at DESC)', idx_created_at_name, table_name);
    EXECUTE format('CREATE UNIQUE INDEX IF NOT EXISTS %I ON conversation_messages.%I(client_message_id) WHERE client_message_id IS NOT NULL', idx_client_msg_id_name, table_name);
END;
$$ LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, conversation_messages, pg_temp;

-- 2. 修改插入函数：支持 client_message_id 参数，幂等写入
CREATE OR REPLACE FUNCTION insert_conversation_message(
    conversation_uuid UUID,
    sender_uuid UUID,
    msg_content TEXT,
    msg_type VARCHAR(20),
    bot_id UUID DEFAULT NULL,
    bot_name VARCHAR(100) DEFAULT NULL,
    client_message_id VARCHAR(255) DEFAULT NULL,
    message_created_at TIMESTAMP DEFAULT NULL
)
RETURNS UUID AS $$
DECLARE
    new_message_id UUID;
    table_name TEXT;
    created_at_value TIMESTAMP;
BEGIN
    table_name := replace(conversation_uuid::TEXT, '-', '_');
    created_at_value := COALESCE(message_created_at, CURRENT_TIMESTAMP);

    -- 幂等检查：如果 client_message_id 已存在，直接返回已有消息 ID
    IF client_message_id IS NOT NULL THEN
        EXECUTE format('
            SELECT id FROM conversation_messages.%I
            WHERE client_message_id = $1
        ', table_name)
        INTO new_message_id
        USING client_message_id;

        IF new_message_id IS NOT NULL THEN
            RETURN new_message_id;
        END IF;
    END IF;

    EXECUTE format('
        INSERT INTO conversation_messages.%I (sender_id, content, msg_type, bot_id, bot_name, client_message_id, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id
    ', table_name)
    INTO new_message_id
    USING sender_uuid, msg_content, msg_type, bot_id, bot_name, client_message_id, created_at_value;

    RETURN new_message_id;
END;
$$ LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, conversation_messages, pg_temp;

-- 3. 新增查询函数：通过 client_message_id 查找消息
CREATE OR REPLACE FUNCTION get_conversation_message_by_client_id(
    conversation_uuid UUID,
    p_client_message_id VARCHAR(255)
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
        WHERE client_message_id = $1
    ', table_name)
    USING p_client_message_id;
END;
$$ LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, conversation_messages, pg_temp;

-- 4. 辅助函数：为存量分表添加 client_message_id 列（幂等执行）
CREATE OR REPLACE FUNCTION add_client_message_id_to_existing_tables()
RETURNS VOID AS $$
DECLARE
    tbl RECORD;
    col_exists BOOLEAN;
BEGIN
    FOR tbl IN
        SELECT table_name FROM information_schema.tables
        WHERE table_schema = 'conversation_messages'
          AND table_type = 'BASE TABLE'
    LOOP
        SELECT EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_schema = 'conversation_messages'
              AND table_name = tbl.table_name
              AND column_name = 'client_message_id'
        ) INTO col_exists;

        IF NOT col_exists THEN
            EXECUTE format('
                ALTER TABLE conversation_messages.%I
                ADD COLUMN client_message_id VARCHAR(255)
            ', tbl.table_name);

            EXECUTE format('
                CREATE UNIQUE INDEX IF NOT EXISTS idx_%s_client_message_id
                ON conversation_messages.%I(client_message_id) WHERE client_message_id IS NOT NULL
            ', tbl.table_name, tbl.table_name);
        END IF;
    END LOOP;
END;
$$ LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, conversation_messages, pg_temp;

-- 执行存量表迁移
SELECT add_client_message_id_to_existing_tables();
