-- 创建系统保留用户，用于注销账号时消息 sender_id 的匿名化目标
-- 使用全零 UUID 避免与正常用户冲突，uid 设为 -1 跳过序列
INSERT INTO users (id, uid, username, password_hash, salt, avatar_url, is_bot)
VALUES ('00000000-0000-0000-0000-000000000000'::uuid, -1, 'deleted_user', '', '', '', FALSE)
ON CONFLICT (id) DO NOTHING;

COMMENT ON TABLE users IS '用户表（包含 id=00000000... 的系统保留用户，用于消息匿名化）';
