-- 添加 conversation_id 字段到 friendships 表（如果不存在）
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name='friendships' 
        AND column_name='conversation_id'
    ) THEN
        ALTER TABLE friendships ADD COLUMN conversation_id UUID REFERENCES conversations(id) ON DELETE CASCADE;
    END IF;
END $$;

-- 为现有的好友请求设置 conversation_id
-- 通过查找对应的会话（通过 user_id 和 friend_id 在 enrollments 表中查找）
UPDATE friendships f
SET conversation_id = (
    SELECT e1.conversation_id
    FROM enrollments e1
    JOIN enrollments e2 ON e1.conversation_id = e2.conversation_id AND e1.user_id != e2.user_id
    WHERE e1.user_id = f.user_id
    AND e2.user_id = f.friend_id
    LIMIT 1
)
WHERE f.conversation_id IS NULL;

-- 为 conversation_id 创建索引（如果不存在）
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 
        FROM pg_indexes 
        WHERE tablename='friendships' 
        AND indexname='idx_friendships_conversation_id'
    ) THEN
        CREATE INDEX idx_friendships_conversation_id ON friendships(conversation_id);
    END IF;
END $$;
