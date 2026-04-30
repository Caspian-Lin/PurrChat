-- 群聊头像字段
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS avatar_url TEXT;
