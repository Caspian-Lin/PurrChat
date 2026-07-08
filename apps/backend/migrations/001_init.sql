-- PurrChat 数据库初始化脚本
-- 统一迁移：合并 001-010 所有变更为最终 schema

-- ============================================================
-- 序列
-- ============================================================
CREATE SEQUENCE IF NOT EXISTS user_uid_seq START WITH 1;

-- ============================================================
-- 用户表
-- ============================================================
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    uid INTEGER UNIQUE NOT NULL DEFAULT nextval('user_uid_seq'),
    username VARCHAR(40) NOT NULL,
    password_hash TEXT NOT NULL,
    salt TEXT NOT NULL,
    avatar_url TEXT DEFAULT '',
    email VARCHAR(100) UNIQUE,
    email_verified BOOLEAN DEFAULT FALSE,
    phone VARCHAR(20) UNIQUE,
    phone_verified BOOLEAN DEFAULT FALSE,
    is_bot BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 部分唯一索引：普通用户和 Bot 的用户名各自唯一
CREATE UNIQUE INDEX idx_users_username_unique ON users(username) WHERE is_bot = FALSE;
CREATE UNIQUE INDEX idx_users_bot_username_unique ON users(username) WHERE is_bot = TRUE;

-- ============================================================
-- 会话表
-- ============================================================
CREATE TABLE conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_type VARCHAR(20) NOT NULL,
    name VARCHAR(100),
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT check_conversation_type CHECK (conversation_type IN ('direct', 'group'))
);

-- ============================================================
-- 好友关系表
-- ============================================================
CREATE TABLE friendships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    friend_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    conversation_id UUID REFERENCES conversations(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, friend_id),
    CONSTRAINT check_status CHECK (status IN ('pending', 'accepted', 'rejected', 'blocked'))
);

-- ============================================================
-- 消息表 Schema（按会话分表存储）
-- ============================================================
CREATE SCHEMA IF NOT EXISTS conversation_messages;

-- ============================================================
-- 用户与会话关联表
-- ============================================================
CREATE TABLE enrollments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) DEFAULT 'member',
    joined_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_read_at TIMESTAMP,
    UNIQUE(conversation_id, user_id),
    CONSTRAINT check_role CHECK (role IN ('owner', 'admin', 'member'))
);

-- ============================================================
-- 用户设置表
-- ============================================================
CREATE TABLE user_settings (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    settings JSONB NOT NULL DEFAULT '{}'::jsonb,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Bot 系统
-- ============================================================
CREATE TABLE bots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(40) NOT NULL,
    avatar_url TEXT DEFAULT '',
    description TEXT DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    visibility VARCHAR(20) NOT NULL DEFAULT 'private',
    mechanism_config JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT check_bot_status CHECK (status IN ('active', 'disabled')),
    CONSTRAINT check_bot_visibility CHECK (visibility IN ('private', 'public', 'global'))
);

CREATE TABLE bot_deployments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    deployed_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    special_mode_active BOOLEAN DEFAULT FALSE,
    special_mode_started_at TIMESTAMP,
    deployed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(bot_id, conversation_id),
    CONSTRAINT check_deployment_status CHECK (status IN ('active', 'paused'))
);

-- ============================================================
-- 索引
-- ============================================================
CREATE INDEX idx_users_uid ON users(uid);
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
CREATE INDEX idx_bots_owner_id ON bots(owner_id);
CREATE INDEX idx_bots_status ON bots(status);
CREATE INDEX idx_bots_visibility ON bots(visibility);
CREATE INDEX idx_bot_deployments_bot_id ON bot_deployments(bot_id);
CREATE INDEX idx_bot_deployments_conversation_id ON bot_deployments(conversation_id);
CREATE INDEX idx_bot_deployments_status ON bot_deployments(status);

-- ============================================================
-- 触发器函数
-- ============================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_conversations_updated_at
    BEFORE UPDATE ON conversations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_bots_updated_at
    BEFORE UPDATE ON bots
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 消息表操作函数
-- ============================================================

-- 为会话创建消息表
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
$$ LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, conversation_messages, pg_temp;

-- 删除会话消息表
CREATE OR REPLACE FUNCTION drop_conversation_message_table(conversation_uuid UUID)
RETURNS VOID AS $$
DECLARE
    table_name TEXT;
BEGIN
    table_name := replace(conversation_uuid::TEXT, '-', '_');
    EXECUTE format('DROP TABLE IF EXISTS conversation_messages.%I CASCADE', table_name);
END;
$$ LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, conversation_messages, pg_temp;

-- 向会话消息表插入消息
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
$$ LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, conversation_messages, pg_temp;

-- 从会话消息表获取消息（分页，按时间倒序）
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
        ORDER BY created_at DESC, bot_id NULLS LAST
        LIMIT $1 OFFSET $2
    ', table_name)
    USING msg_limit, msg_offset;
END;
$$ LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, conversation_messages, pg_temp;

-- 增量获取会话消息（按时间正序）
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
        ORDER BY created_at ASC, bot_id NULLS FIRST
    ', table_name)
    USING since_timestamp;
END;
$$ LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, conversation_messages, pg_temp;

-- 获取会话消息总数
CREATE OR REPLACE FUNCTION get_conversation_message_count(conversation_uuid UUID)
RETURNS BIGINT AS $$
DECLARE
    table_name TEXT;
    message_count BIGINT;
BEGIN
    table_name := replace(conversation_uuid::TEXT, '-', '_');
    EXECUTE format('
        SELECT COUNT(*)
        FROM conversation_messages.%I
    ', table_name)
    INTO message_count;

    RETURN message_count;
END;
$$ LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, conversation_messages, pg_temp;

-- 获取会话最后一条消息
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
        ORDER BY created_at DESC, bot_id NULLS LAST
        LIMIT 1
    ', table_name);
END;
$$ LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, conversation_messages, pg_temp;

-- ============================================================
-- 系统用户（已删除用户占位符）
-- ============================================================
INSERT INTO users (id, uid, username, password_hash, salt, avatar_url, is_bot)
VALUES ('00000000-0000-0000-0000-000000000000', -1, 'deleted_user', '', '', '', FALSE)
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- 注释
-- ============================================================
COMMENT ON SCHEMA conversation_messages IS '存放所有会话消息表的schema';
COMMENT ON TABLE users IS '用户表（含 Bot 用户）';
COMMENT ON TABLE conversations IS '会话表（私聊和群聊）';
COMMENT ON TABLE friendships IS '好友关系表';
COMMENT ON TABLE enrollments IS '用户与会话的关联表';
COMMENT ON TABLE user_settings IS '用户设置表';
COMMENT ON TABLE bots IS 'Bot 定义表';
COMMENT ON TABLE bot_deployments IS 'Bot 部署表';
