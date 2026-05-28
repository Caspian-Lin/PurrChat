-- 003_rename_special_mode_to_workflow.sql
-- 将 bot_deployments 表的 special_mode 相关字段重命名为 workflow

ALTER TABLE bot_deployments RENAME COLUMN special_mode_active TO workflow_active;
ALTER TABLE bot_deployments RENAME COLUMN special_mode_started_at TO workflow_started_at;
