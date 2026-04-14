-- 文件元数据表
-- 存储所有上传文件的元信息，用于追踪和管理工作
CREATE TABLE IF NOT EXISTS file_metadata (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    object_key TEXT NOT NULL UNIQUE,                            -- MinIO 对象 key（唯一）
    file_name TEXT NOT NULL,                                     -- 原始文件名
    file_size BIGINT NOT NULL,                                   -- 文件大小（字节）
    content_type VARCHAR(100) NOT NULL,                          -- MIME 类型
    category VARCHAR(20) NOT NULL,                               -- 文件类别：avatar, background, chat-image, file
    usage VARCHAR(20) NOT NULL,                                  -- 文件用途：avatar, background, message, temp
    uploader_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    public_url TEXT,                                             -- 公开访问 URL
    etag TEXT,                                                   -- 文件 ETag
    confirmed BOOLEAN NOT NULL DEFAULT FALSE,                    -- 上传是否已确认
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    confirmed_at TIMESTAMP,                                      -- 确认时间
    CONSTRAINT check_file_category CHECK (category IN ('avatar', 'background', 'chat-image', 'file')),
    CONSTRAINT check_file_usage CHECK (usage IN ('avatar', 'background', 'message', 'temp'))
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_file_metadata_uploader_id ON file_metadata(uploader_id);
CREATE INDEX IF NOT EXISTS idx_file_metadata_category ON file_metadata(category);
CREATE INDEX IF NOT EXISTS idx_file_metadata_usage ON file_metadata(usage);
CREATE INDEX IF NOT EXISTS idx_file_metadata_confirmed ON file_metadata(confirmed);
CREATE INDEX IF NOT EXISTS idx_file_metadata_created_at ON file_metadata(created_at);

COMMENT ON TABLE file_metadata IS '文件元数据表';
COMMENT ON COLUMN file_metadata.object_key IS 'MinIO 对象存储 key';
COMMENT ON COLUMN file_metadata.confirmed IS '标识文件是否已成功上传并确认';
