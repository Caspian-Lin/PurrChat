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
--   psql -U postgres -d postgres \
--     -v app_db_name=purrchat \
--     -v app_db_user=purrchat \
--     -v app_db_password=purrchat_pw \
--     -f scripts/cleanup_database.sql
-- ======================================

\if :{?app_db_name}
\else
\set app_db_name purrchat
\endif

\if :{?app_db_user}
\else
\set app_db_user purrchat
\endif

\if :{?app_db_password}
\else
\set app_db_password purrchat_pw
\endif

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
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE datname = :'app_db_name'
  AND pid <> pg_backend_pid();

-- ======================================
-- 2. 删除 purrchat 数据库
-- ======================================
DROP DATABASE IF EXISTS :"app_db_name";
\echo 已删除数据库: :app_db_name

-- ======================================
-- 3. 创建或更新应用数据库用户
-- ======================================
SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', :'app_db_user', :'app_db_password')
WHERE NOT EXISTS (
    SELECT 1 FROM pg_roles WHERE rolname = :'app_db_user'
)\gexec

SELECT format('ALTER ROLE %I WITH LOGIN PASSWORD %L', :'app_db_user', :'app_db_password')
WHERE EXISTS (
    SELECT 1 FROM pg_roles WHERE rolname = :'app_db_user'
)\gexec

\echo 已确保应用数据库用户存在: :app_db_user

-- ======================================
-- 4. 创建新的 purrchat 数据库，并设置 owner 为应用用户
-- ======================================
CREATE DATABASE :"app_db_name" OWNER :"app_db_user";
\echo 已创建数据库: :app_db_name

-- ======================================
-- 5. 连接到新数据库并设置权限
-- ======================================
\connect :app_db_name

-- ======================================
-- 6. 创建扩展（如果需要）
-- ======================================
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
\echo 已创建扩展: uuid-ossp

-- PostgreSQL 15+ 默认不再向普通用户授予 public schema 的 CREATE 权限。
-- make migrate 使用应用用户执行 DDL，因此必须显式授予 schema 权限。
GRANT ALL PRIVILEGES ON DATABASE :"app_db_name" TO :"app_db_user";
GRANT USAGE, CREATE ON SCHEMA public TO :"app_db_user";
ALTER SCHEMA public OWNER TO :"app_db_user";
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO :"app_db_user";
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO :"app_db_user";
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON FUNCTIONS TO :"app_db_user";
\echo 已授予应用用户 :app_db_user 在数据库 :app_db_name 上的迁移权限

-- ======================================
-- 7. 验证结果
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
    RAISE NOTICE '  cd apps/backend && go run ./cmd/migrate up';
    RAISE NOTICE '========================================';
END $$;

-- ======================================
-- 清理完成
-- ======================================
