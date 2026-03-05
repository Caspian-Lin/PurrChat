-- PurrChat 新的会话结构迁移

-- 创建conversation_messages schema用于存放会话消息表
CREATE SCHEMA IF NOT EXISTS conversation_messages;

-- 创建enrollments表（用户与会话的多对多关系）
CREATE TABLE enrollments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) DEFAULT 'member', -- 'owner', 'admin', 'member'
    joined_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_read_at TIMESTAMP, -- 最后阅读时间
    UNIQUE(conversation_id, user_id)
);

-- 创建索引
CREATE INDEX idx_enrollments_conversation_id ON enrollments(conversation_id);
CREATE INDEX idx_enrollments_user_id ON enrollments(user_id);

-- 修改conversations表，完全移除user1_id和user2_id，改为支持群聊
-- 注意：由于现有数据依赖user1_id和user2_id，我们需要先迁移数据

-- 1. 添加新字段
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS name VARCHAR(100);
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS conversation_type_new VARCHAR(20);
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS created_by UUID REFERENCES users(id) ON DELETE SET NULL;

-- 2. 迁移现有数据到enrollments表
INSERT INTO enrollments (conversation_id, user_id, role, joined_at)
SELECT
    id,
    user1_id,
    'owner' as role,
    created_at
FROM conversations
ON CONFLICT (conversation_id, user_id) DO NOTHING;

INSERT INTO enrollments (conversation_id, user_id, role, joined_at)
SELECT
    id,
    user2_id,
    'member' as role,
    created_at
FROM conversations
ON CONFLICT (conversation_id, user_id) DO NOTHING;

-- 3. 更新conversations表的created_by字段
UPDATE conversations SET created_by = user1_id;

-- 4. 迁移conversation_type到新字段
UPDATE conversations SET conversation_type_new = conversation_type::VARCHAR(20);

-- 5. 为私聊会话设置名称（使用对方用户名）
UPDATE conversations c
SET name = (
    SELECT CASE
        WHEN c.user1_id = u1.id THEN u2.username
        ELSE u1.username
    END
    FROM users u1
    JOIN users u2 ON u1.id != u2.id
    WHERE (u1.id = c.user1_id AND u2.id = c.user2_id)
       OR (u1.id = c.user2_id AND u2.id = c.user1_id)
    LIMIT 1
)
WHERE conversation_type_new = 'friend' OR conversation_type_new = 'stranger';

-- 6. 删除旧字段
ALTER TABLE conversations DROP COLUMN IF EXISTS user1_id;
ALTER TABLE conversations DROP COLUMN IF EXISTS user2_id;
ALTER TABLE conversations DROP COLUMN IF EXISTS conversation_type;

-- 7. 重命名新字段
ALTER TABLE conversations RENAME COLUMN conversation_type_new TO conversation_type;

-- 8. 删除旧的has_pending_request和request_status字段（这些逻辑现在通过enrollment管理）
ALTER TABLE conversations DROP COLUMN IF EXISTS has_pending_request;
ALTER TABLE conversations DROP COLUMN IF EXISTS request_status;

-- 创建函数：为会话创建消息表
CREATE OR REPLACE FUNCTION create_conversation_message_table(conversation_uuid UUID)
RETURNS VOID AS $$
DECLARE
    table_name TEXT;
    idx_sender_name TEXT;
    idx_created_at_name TEXT;
BEGIN
    -- 将UUID转换为表名（移除连字符）
    table_name := replace(conversation_uuid::TEXT, '-', '_');

    -- 创建消息表
    EXECUTE format('
        CREATE TABLE IF NOT EXISTS conversation_messages.%I (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
            content TEXT NOT NULL,
            msg_type VARCHAR(20) NOT NULL DEFAULT ''text'',
            created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
        )',
    table_name
    );

    -- 创建索引名称
    idx_sender_name := 'idx_' || table_name || '_sender_id';
    idx_created_at_name := 'idx_' || table_name || '_created_at';

    -- 创建索引
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

-- 创建函数：删除会话消息表
CREATE OR REPLACE FUNCTION drop_conversation_message_table(conversation_uuid UUID)
RETURNS VOID AS $$
DECLARE
    table_name TEXT;
BEGIN
    -- 将UUID转换为表名（移除连字符）
    table_name := replace(conversation_uuid::TEXT, '-', '_');
    EXECUTE format('DROP TABLE IF EXISTS conversation_messages.%I CASCADE', table_name);
END;
$$ LANGUAGE plpgsql;

-- 创建函数：向会话消息表插入消息
CREATE OR REPLACE FUNCTION insert_conversation_message(
    conversation_uuid UUID,
    sender_uuid UUID,
    msg_content TEXT,
    msg_type VARCHAR(20)
)
RETURNS UUID AS $$
DECLARE
    new_message_id UUID;
    table_name TEXT;
BEGIN
    -- 将UUID转换为表名（移除连字符）
    table_name := replace(conversation_uuid::TEXT, '-', '_');
    EXECUTE format('
        INSERT INTO conversation_messages.%I (sender_id, content, msg_type)
        VALUES ($1, $2, $3)
        RETURNING id
    ',
    table_name)
    INTO new_message_id
    USING sender_uuid, msg_content, msg_type;

    RETURN new_message_id;
END;
$$ LANGUAGE plpgsql;

-- 创建函数：从会话消息表获取消息
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
    created_at TIMESTAMP
) AS $$
DECLARE
    table_name TEXT;
BEGIN
    -- 将UUID转换为表名（移除连字符）
    table_name := replace(conversation_uuid::TEXT, '-', '_');
    RETURN QUERY EXECUTE format('
        SELECT id, sender_id, content, msg_type, created_at
        FROM conversation_messages.%I
        ORDER BY created_at DESC
        LIMIT $1 OFFSET $2
    ',
    table_name)
    USING msg_limit, msg_offset;
END;
$$ LANGUAGE plpgsql;

-- 为现有会话创建消息表
DO $$
DECLARE
    conv_record RECORD;
    messages_exists BOOLEAN;
    conv_table_name TEXT;
BEGIN
    -- 检查messages表是否存在
    SELECT EXISTS (
        SELECT FROM information_schema.tables
        WHERE table_schema = 'public'
        AND table_name = 'messages'
    ) INTO messages_exists;

    FOR conv_record IN SELECT id FROM conversations LOOP
        PERFORM create_conversation_message_table(conv_record.id);

        -- 如果messages表存在，迁移现有消息到新的消息表
        IF messages_exists THEN
            -- 将UUID转换为表名（移除连字符）
            conv_table_name := replace(conv_record.id::TEXT, '-', '_');
            EXECUTE format('INSERT INTO conversation_messages.%I (id, sender_id, content, msg_type, created_at) SELECT id, sender_id, content, msg_type, created_at FROM messages WHERE conversation_id = $1 ON CONFLICT (id) DO NOTHING', conv_table_name)
            USING conv_record.id;
        END IF;
    END LOOP;
END $$;

-- 添加注释
COMMENT ON SCHEMA conversation_messages IS '存放所有会话消息表的schema';
COMMENT ON TABLE enrollments IS '用户与会话的关联表（多对多关系）';
COMMENT ON FUNCTION create_conversation_message_table IS '为指定会话创建消息表';
COMMENT ON FUNCTION drop_conversation_message_table IS '删除指定会话的消息表';
COMMENT ON FUNCTION insert_conversation_message IS '向指定会话的消息表插入消息';
COMMENT ON FUNCTION get_conversation_messages IS '从指定会话的消息表获取消息';
