-- #13: 版本化 Workflow Document 存储
-- 1. bots 表新增 workflow_document (草稿) + workflow_revision (乐观锁)
-- 2. 新建 workflow_versions 表（发布历史）
-- 向后兼容：mechanism_config 保持不变，API 优先读 workflow_document

ALTER TABLE bots ADD COLUMN IF NOT EXISTS workflow_document jsonb;
ALTER TABLE bots ADD COLUMN IF NOT EXISTS workflow_revision int DEFAULT 0;

CREATE TABLE IF NOT EXISTS workflow_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    revision INTEGER NOT NULL,
    document JSONB NOT NULL,
    capabilities TEXT[] DEFAULT '{}',
    published_by UUID REFERENCES users(id),
    published_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(bot_id, revision)
);

CREATE INDEX IF NOT EXISTS idx_workflow_versions_bot_id ON workflow_versions(bot_id);
