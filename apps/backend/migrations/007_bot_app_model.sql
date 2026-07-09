-- ============================================================
-- Bot App 化模型:BotIdentity / BotInstallation + bots 表演进
-- issue #33, 设计见 docs/bot-engine/BOT_APP_MODEL.md
-- ============================================================

-- 1. 演进 bots 表为 BotApp 等价物(加 App 化字段,保留旧字段供 #36 迁移清理)
ALTER TABLE bots
    ADD COLUMN IF NOT EXISTS bot_type VARCHAR(20) NOT NULL DEFAULT 'workflow',
    ADD COLUMN IF NOT EXISTS discoverability VARCHAR(20) NOT NULL DEFAULT 'unlisted',
    ADD COLUMN IF NOT EXISTS is_system BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS published_version INTEGER,
    ADD COLUMN IF NOT EXISTS requested_capabilities TEXT[] NOT NULL DEFAULT '{}';

ALTER TABLE bots
    DROP CONSTRAINT IF EXISTS check_bot_type;
ALTER TABLE bots
    ADD CONSTRAINT check_bot_type CHECK (bot_type IN ('builtin', 'workflow', 'external'));

ALTER TABLE bots
    DROP CONSTRAINT IF EXISTS check_bot_discoverability;
ALTER TABLE bots
    ADD CONSTRAINT check_bot_discoverability CHECK (discoverability IN ('unlisted', 'listed', 'featured'));

-- 把现有 visibility 值映射到新维度(visibility 字段保留,#36 负责删除)
UPDATE bots SET discoverability = 'listed' WHERE visibility IN ('public', 'global') AND discoverability = 'unlisted';
UPDATE bots SET is_system = TRUE WHERE visibility = 'global';

-- 2. BotIdentity 系统身份投影(不可登录、不可好友;仅用于 message.sender_id)
CREATE TABLE IF NOT EXISTS bot_identities (
    app_id UUID PRIMARY KEY REFERENCES bots(id) ON DELETE CASCADE,
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    display_name VARCHAR(40) NOT NULL,
    avatar_url TEXT DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 为现有 Bot 回填 bot_identities(app_id = bot.id = user_id 投影)
INSERT INTO bot_identities (app_id, user_id, display_name, avatar_url)
SELECT b.id, b.id, b.name, b.avatar_url
FROM bots b
WHERE NOT EXISTS (SELECT 1 FROM bot_identities bi WHERE bi.app_id = b.id);

-- 3. BotInstallation 统一安装(替代 friendship + bot_deployments 的安装语义)
CREATE TABLE IF NOT EXISTS bot_installations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    installed_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_type VARCHAR(20) NOT NULL,
    target_id UUID NOT NULL,
    granted_capabilities TEXT[] NOT NULL DEFAULT '{}',
    diagnostics_consent VARCHAR(20) NOT NULL DEFAULT 'denied',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    config JSONB DEFAULT '{}'::jsonb,
    installed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(target_type, target_id, app_id),
    CONSTRAINT check_installation_target CHECK (target_type IN ('user', 'conversation')),
    CONSTRAINT check_installation_diag CHECK (diagnostics_consent IN ('denied', 'granted')),
    CONSTRAINT check_installation_status CHECK (status IN ('active', 'paused', 'disabled'))
);

CREATE INDEX IF NOT EXISTS idx_bot_installations_app ON bot_installations (app_id);
CREATE INDEX IF NOT EXISTS idx_bot_installations_target ON bot_installations (target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_bot_installations_installed_by ON bot_installations (installed_by);
