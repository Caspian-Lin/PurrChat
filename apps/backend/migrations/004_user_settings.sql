-- 用户设置表（单行 JSON 存储，每用户一条记录）
CREATE TABLE IF NOT EXISTS user_settings (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    settings JSONB NOT NULL DEFAULT '{}'::jsonb,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 更新时间触发器（复用已有的 update_updated_at_column 函数）
CREATE TRIGGER update_user_settings_updated_at
    BEFORE UPDATE ON user_settings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE user_settings IS '用户设置表（单行 JSON 存储，每用户一条记录）';
COMMENT ON COLUMN user_settings.settings IS '用户设置 JSON 对象，包含 panels/notifications/general 等分类';
COMMENT ON COLUMN user_settings.updated_at IS '最后更新时间';
