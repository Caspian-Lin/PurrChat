-- PurrChat Bot 系统数据库迁移
-- 创建 Bot 相关表、索引和兼容函数

-- Bot 表：独立于 users 表的 Bot 身份
CREATE TABLE IF NOT EXISTS bots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(40) NOT NULL,
    avatar_url TEXT DEFAULT '',
    description TEXT DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    visibility VARCHAR(20) NOT NULL DEFAULT 'private',
    trigger_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    reply_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    special_mode_config JSONB DEFAULT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT check_bot_status CHECK (status IN ('active', 'disabled')),
    CONSTRAINT check_bot_visibility CHECK (visibility IN ('private', 'public', 'global'))
);

-- Bot 部署表：记录 Bot 被添加到哪些会话中
CREATE TABLE IF NOT EXISTS bot_deployments (
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

-- 为会话添加 bot_enabled 标记
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS bot_enabled BOOLEAN DEFAULT FALSE;

-- 索引
CREATE INDEX IF NOT EXISTS idx_bots_owner_id ON bots(owner_id);
CREATE INDEX IF NOT EXISTS idx_bots_status ON bots(status);
CREATE INDEX IF NOT EXISTS idx_bots_visibility ON bots(visibility);
CREATE INDEX IF NOT EXISTS idx_bot_deployments_bot_id ON bot_deployments(bot_id);
CREATE INDEX IF NOT EXISTS idx_bot_deployments_conversation_id ON bot_deployments(conversation_id);
CREATE INDEX IF NOT EXISTS idx_bot_deployments_status ON bot_deployments(status);

-- 更新时间触发器
CREATE OR REPLACE FUNCTION update_bots_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS update_bots_updated_at ON bots;
CREATE TRIGGER update_bots_updated_at
    BEFORE UPDATE ON bots
    FOR EACH ROW
    EXECUTE FUNCTION update_bots_updated_at();

-- 注释
COMMENT ON TABLE bots IS 'Bot 定义表';
COMMENT ON TABLE bot_deployments IS 'Bot 部署表（Bot 与会话的关联）';
