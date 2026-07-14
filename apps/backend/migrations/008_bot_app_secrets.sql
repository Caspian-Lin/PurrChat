-- ============================================================
-- Secret 引用机制:bot_app_secrets 加密密钥存储 + 出站端点白名单骨架
-- issue #35, 设计见 docs/bot-engine/BOT_APP_MODEL.md §5
-- ============================================================

-- 1. BotApp 级 secret store(owner 管理,密文存储)
CREATE TABLE IF NOT EXISTS bot_app_secrets (
    app_id     UUID        NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    key_name   VARCHAR(64) NOT NULL,
    ciphertext TEXT        NOT NULL,  -- base64(IV || AES-256-GCM ciphertext+tag)
    created_at TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (app_id, key_name)
);

-- 2. 出站端点白名单(骨架,P3 强制拦截)
ALTER TABLE bots
    ADD COLUMN IF NOT EXISTS allowed_endpoints TEXT[] NOT NULL DEFAULT '{}';
