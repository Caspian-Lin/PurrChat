-- 添加 'rejected' 状态到 friendships 表的 check_status 约束

-- 先删除旧的约束
ALTER TABLE friendships DROP CONSTRAINT IF EXISTS check_status;

-- 添加新的约束，包含 'rejected' 状态
ALTER TABLE friendships ADD CONSTRAINT check_status CHECK (status IN ('pending', 'accepted', 'rejected', 'blocked'));
