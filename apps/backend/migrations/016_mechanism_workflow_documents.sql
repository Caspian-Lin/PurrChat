-- #87: 工作流文档存储从 bot 级改为 mechanism 级
-- 每个 (bot_id, mechanism_id) 拥有独立的草稿与发布版本历史。
-- 内部测试阶段不迁移历史数据（owner 决策），bots 表的 bot 级工作流列直接删除。

-- 1. 草稿表：每个 mechanism 一份草稿 + 乐观锁 revision
CREATE TABLE IF NOT EXISTS bot_workflow_documents (
    bot_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    mechanism_id VARCHAR(100) NOT NULL,
    document JSONB,
    revision INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (bot_id, mechanism_id)
);

-- 2. workflow_versions 升级为 mechanism 级唯一键
ALTER TABLE workflow_versions ADD COLUMN IF NOT EXISTS mechanism_id VARCHAR(100) NOT NULL DEFAULT '';
ALTER TABLE workflow_versions DROP CONSTRAINT IF EXISTS workflow_versions_bot_id_revision_key;
ALTER TABLE workflow_versions ADD CONSTRAINT workflow_versions_bot_id_mechanism_revision_key UNIQUE (bot_id, mechanism_id, revision);
DROP INDEX IF EXISTS idx_workflow_versions_bot_id;
CREATE INDEX IF NOT EXISTS idx_workflow_versions_bot_mechanism ON workflow_versions(bot_id, mechanism_id);

-- 3. 删除 bots 表的 bot 级工作流列（草稿 / 乐观锁 / published_version 均由 mechanism 级表取代）
ALTER TABLE bots DROP COLUMN IF EXISTS workflow_document;
ALTER TABLE bots DROP COLUMN IF EXISTS workflow_revision;
ALTER TABLE bots DROP COLUMN IF EXISTS published_version;
