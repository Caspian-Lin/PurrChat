-- PurrChat 数据库初始化脚本
-- 简化版数据库结构，移除了不需要的groups、group_members和messages表

-- 创建UID序列
CREATE SEQUENCE IF NOT EXISTS user_uid_seq START WITH 1;

-- 用户表
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    uid INTEGER UNIQUE NOT NULL DEFAULT nextval('user_uid_seq'),
    username VARCHAR(20) NOT NULL,
    password_hash TEXT NOT NULL,
    salt TEXT NOT NULL,
    avatar_url TEXT DEFAULT '',
    email VARCHAR(100) UNIQUE,
    email_verified BOOLEAN DEFAULT FALSE,
    phone VARCHAR(20) UNIQUE,
    phone_verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 会话表
CREATE TABLE conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_type VARCHAR(20) NOT NULL, -- 'direct' 或 'group'
    name VARCHAR(100), -- 会话名称（群聊时使用）
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT check_conversation_type CHECK (conversation_type IN ('direct', 'group'))
);

-- 好友关系表
CREATE TABLE friendships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    friend_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    conversation_id UUID REFERENCES conversations(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, accepted, rejected, blocked
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, friend_id),
    CONSTRAINT check_status CHECK (status IN ('pending', 'accepted', 'rejected', 'blocked'))
);

-- 创建conversation_messages schema用于存放会话消息表
CREATE SCHEMA IF NOT EXISTS conversation_messages;

-- 用户与会话的关联表（enrollments）
CREATE TABLE enrollments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) DEFAULT 'member', -- 'owner', 'admin', 'member'
    joined_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_read_at TIMESTAMP, -- 最后阅读时间
    UNIQUE(conversation_id, user_id),
    CONSTRAINT check_role CHECK (role IN ('owner', 'admin', 'member'))
);

-- 创建索引
CREATE INDEX idx_users_uid ON users(uid);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_phone ON users(phone);
CREATE INDEX idx_conversations_type ON conversations(conversation_type);
CREATE INDEX idx_conversations_created_by ON conversations(created_by);
CREATE INDEX idx_friendships_user_id ON friendships(user_id);
CREATE INDEX idx_friendships_friend_id ON friendships(friend_id);
CREATE INDEX idx_friendships_conversation_id ON friendships(conversation_id);
CREATE INDEX idx_friendships_status ON friendships(status);
CREATE INDEX idx_enrollments_conversation_id ON enrollments(conversation_id);
CREATE INDEX idx_enrollments_user_id ON enrollments(user_id);

-- 创建更新时间触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 为conversations表创建更新时间触发器
CREATE TRIGGER update_conversations_updated_at
    BEFORE UPDATE ON conversations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

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
            created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
            CONSTRAINT check_msg_type CHECK (msg_type IN (''text'', ''image''))
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

-- 创建函数：增量获取会话消息
CREATE OR REPLACE FUNCTION get_conversation_messages_incremental(
    conversation_uuid UUID,
    since_timestamp TIMESTAMP
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
        WHERE created_at > $1
        ORDER BY created_at ASC
    ',
    table_name)
    USING since_timestamp;
END;
$$ LANGUAGE plpgsql;

-- 创建函数：获取会话消息总数
CREATE OR REPLACE FUNCTION get_conversation_message_count(conversation_uuid UUID)
RETURNS BIGINT AS $$
DECLARE
    table_name TEXT;
    message_count BIGINT;
BEGIN
    -- 将UUID转换为表名（移除连字符）
    table_name := replace(conversation_uuid::TEXT, '-', '_');
    EXECUTE format('
        SELECT COUNT(*)
        FROM conversation_messages.%I
    ',
    table_name)
    INTO message_count;

    RETURN message_count;
END;
$$ LANGUAGE plpgsql;

-- 创建函数：获取会话最后一条消息
CREATE OR REPLACE FUNCTION get_conversation_last_message(conversation_uuid UUID)
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
        LIMIT 1
    ',
    table_name);
END;
$$ LANGUAGE plpgsql;

-- 添加注释
COMMENT ON SCHEMA conversation_messages IS '存放所有会话消息表的schema';
COMMENT ON TABLE users IS '用户表';
COMMENT ON TABLE conversations IS '会话表（私聊和群聊）';
COMMENT ON TABLE friendships IS '好友关系表';
COMMENT ON TABLE enrollments IS '用户与会话的关联表（多对多关系）';
COMMENT ON FUNCTION create_conversation_message_table IS '为指定会话创建消息表';
COMMENT ON FUNCTION drop_conversation_message_table IS '删除指定会话的消息表';
COMMENT ON FUNCTION insert_conversation_message IS '向指定会话的消息表插入消息';
COMMENT ON FUNCTION get_conversation_messages IS '从指定会话的消息表获取消息';
COMMENT ON FUNCTION get_conversation_messages_incremental IS '增量获取会话消息';
COMMENT ON FUNCTION get_conversation_message_count IS '获取会话消息总数';
COMMENT ON FUNCTION get_conversation_last_message IS '获取会话最后一条消息';
