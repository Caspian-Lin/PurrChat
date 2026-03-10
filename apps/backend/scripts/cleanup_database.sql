-- ======================================
-- PurrChat 数据库清理脚本
-- ======================================
-- 
-- 此脚本用于删除并重建 PurrChat 数据库。
-- 这是最彻底的清理方式，不依赖于现有的数据库结构。
--
-- 警告：此脚本将删除整个数据库！请谨慎使用！
--
-- 使用方法：
--   psql -U postgres -d postgres -f scripts/cleanup_database.sql
-- ======================================

-- 显示警告信息
DO $$
BEGIN
    RAISE NOTICE '========================================';
    RAISE NOTICE '警告：此脚本将删除整个数据库！';
    RAISE NOTICE '========================================';
    RAISE NOTICE '5秒后开始清理...';
    RAISE NOTICE '按 Ctrl+C 取消操作';
    RAISE NOTICE '========================================';
END $$;

-- 等待5秒
SELECT pg_sleep(5);

-- ======================================
-- 1. 终止所有连接到 purrchat 数据库的会话
-- ======================================
DO $$
DECLARE
    pid INTEGER;
BEGIN
    FOR pid IN
        SELECT pg_terminate_backend(pid)
        FROM pg_stat_activity
        WHERE datname = 'purrchat'
          AND pid <> pg_backend_pid()
    LOOP
        RAISE NOTICE '已终止连接: PID %', pid;
    END LOOP;
END $$;

-- ======================================
-- 2. 删除 purrchat 数据库
-- ======================================
DROP DATABASE IF EXISTS purrchat;
RAISE NOTICE '已删除数据库: purrchat';

-- ======================================
-- 3. 创建新的 purrchat 数据库
-- ======================================
CREATE DATABASE purrchat;
RAISE NOTICE '已创建数据库: purrchat';

-- ======================================
-- 4. 连接到新数据库并设置权限
-- ======================================
\c purrchat

-- ======================================
-- 5. 创建扩展（如果需要）
-- ======================================
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
RAISE NOTICE '已创建扩展: uuid-ossp';

-- ======================================
-- 6. 验证结果
-- ======================================
DO $$
BEGIN
    RAISE NOTICE '========================================';
    RAISE NOTICE '数据库重建完成！';
    RAISE NOTICE '========================================';
    RAISE NOTICE '数据库名称: %', current_database();
    RAISE NOTICE '========================================';
    RAISE NOTICE '现在可以运行迁移脚本：';
    RAISE NOTICE '  make migrate';
    RAISE NOTICE '或直接运行:';
    RAISE NOTICE '  cd apps/backend && go run cmd/server/main.go migrate';
    RAISE NOTICE '========================================';
END $$;

-- ======================================
-- 清理完成
-- ======================================
