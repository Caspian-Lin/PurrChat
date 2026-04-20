-- PurrChat Bot 统一身份模型迁移
-- 将 Bot 作为特殊用户插入 users 表，使其能使用好友体系和 enrollment 体系

-- 0. 扩展 username 列长度以支持 Bot 名称（Bot name max=40，普通用户仍由应用层限制 20）
ALTER TABLE users ALTER COLUMN username TYPE VARCHAR(40);

-- 1. users 表添加 is_bot 标记
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_bot BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX IF NOT EXISTS idx_users_is_bot ON users(is_bot);

-- 2. username 部分唯一索引（Bot 和普通用户可同名，Bot 之间不可同名）
DROP INDEX IF EXISTS idx_users_username_unique;
DROP INDEX IF EXISTS idx_users_bot_username_unique;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username_unique ON users(username) WHERE is_bot = FALSE;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_bot_username_unique ON users(username) WHERE is_bot = TRUE;

-- 3. 清理现有 Bot 数据（不需要历史数据，全新开始）
DELETE FROM bot_deployments;
DELETE FROM bots;
