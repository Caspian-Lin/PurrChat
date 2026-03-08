-- ======================================
-- PurrChat 数据库清理脚本
-- ======================================
-- 
-- 此脚本用于清理现有的 PurrChat 数据库，
-- 删除旧的表和 schema，以便重新初始化数据库。
--
-- 警告：此脚本将删除所有数据，请谨慎使用！
--
-- 使用方法：
--   psql -U your_username -d purrchat -f scripts/cleanup_database.sql
-- ======================================

-- 开始事务
BEGIN;

-- 显示警告信息
DO $$
BEGIN
    RAISE NOTICE '========================================';
    RAISE NOTICE '警告：此脚本将删除所有数据！';
    RAISE NOTICE '========================================';
    RAISE NOTICE '5秒后开始清理...';
    RAISE NOTICE '按 Ctrl+C 取消操作';
    RAISE NOTICE '========================================';
END $$;

-- 等待5秒
SELECT pg_sleep(5);

-- ======================================
-- 1. 删除 conversation_messages schema 及其所有表
-- ======================================
DO $$
DECLARE
    table_name TEXT;
BEGIN
    -- 删除 schema 中的所有表
    FOR table_name IN
        SELECT tablename
        FROM pg_tables
        WHERE schemaname = 'conversation_messages'
    LOOP
        EXECUTE format('DROP TABLE IF EXISTS conversation_messages.%I CASCADE', table_name);
        RAISE NOTICE '已删除表: conversation_messages.%', table_name;
    END LOOP;
END $$;

-- 删除 conversation_messages schema
DROP SCHEMA IF EXISTS conversation_messages CASCADE;
RAISE NOTICE '已删除 schema: conversation_messages';

-- ======================================
-- 2. 删除旧的表
-- ======================================

-- 删除 groups 表
DROP TABLE IF EXISTS groups CASCADE;
RAISE NOTICE '已删除表: groups';

-- 删除 group_members 表
DROP TABLE IF EXISTS group_members CASCADE;
RAISE NOTICE '已删除表: group_members';

-- 删除 messages 表
DROP TABLE IF EXISTS messages CASCADE;
RAISE NOTICE '已删除表: messages';

-- ======================================
-- 3. 删除旧的迁移记录（可选）
-- ======================================

-- 如果存在迁移表，删除旧的迁移记录
DELETE FROM schema_migrations WHERE version IN ('002', '003', '004');
RAISE NOTICE '已删除旧的迁移记录';

-- ======================================
-- 4. 清理其他可能的残留
-- ======================================

-- 删除可能存在的序列
DROP SEQUENCE IF EXISTS user_uid_seq CASCADE;
RAISE NOTICE '已删除序列: user_uid_seq';

-- ======================================
-- 5. 验证清理结果
-- ======================================

DO $$
BEGIN
    RAISE NOTICE '========================================';
    RAISE NOTICE '清理完成！';
    RAISE NOTICE '========================================';
    RAISE NOTICE '剩余的表：';
    
    FOR tbl IN
        SELECT tablename
        FROM pg_tables
        WHERE schemaname = 'public'
        ORDER BY tablename
    LOOP
        RAISE NOTICE '  - %', tbl;
    END LOOP;
    
    RAISE NOTICE '========================================';
    RAISE NOTICE '剩余的 schema：';
    
    FOR sch IN
        SELECT nspname
        FROM pg_namespace
        WHERE nspname NOT LIKE 'pg_%'
          AND nspname != 'information_schema'
        ORDER BY nspname
    LOOP
        RAISE NOTICE '  - %', sch;
    END LOOP;
    
    RAISE NOTICE '========================================';
    RAISE NOTICE '现在可以运行新的迁移脚本：';
    RAISE NOTICE '  psql -U your_username -d purrchat -f migrations/001_init_schema.sql';
    RAISE NOTICE '========================================';
END $$;

-- 提交事务
COMMIT;

-- ======================================
-- 清理完成
-- ======================================
