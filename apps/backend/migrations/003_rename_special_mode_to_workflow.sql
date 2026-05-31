-- 003_rename_special_mode_to_workflow.sql
-- 将 bot_deployments 表的 special_mode 相关字段重命名为 workflow
-- 使用条件判断避免在已迁移的数据库上重复执行时报错

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'bot_deployments' AND column_name = 'special_mode_active'
    ) THEN
        ALTER TABLE bot_deployments RENAME COLUMN special_mode_active TO workflow_active;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'bot_deployments' AND column_name = 'special_mode_started_at'
    ) THEN
        ALTER TABLE bot_deployments RENAME COLUMN special_mode_started_at TO workflow_started_at;
    END IF;
END $$;
