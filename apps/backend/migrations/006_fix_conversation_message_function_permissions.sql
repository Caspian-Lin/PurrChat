-- 006: 修复 conversation_messages 分表函数权限
--
-- 消息分表位于 conversation_messages schema，函数内部使用动态 SQL 访问分表。
-- PostgreSQL 函数默认 SECURITY INVOKER，会使用后端连接用户的权限执行动态 SQL。
-- 当迁移由数据库 owner 执行、后端使用受限应用用户连接时，会出现：
--   ERROR: permission denied for schema conversation_messages (SQLSTATE 42501)
--
-- 将这些封装函数改为 SECURITY DEFINER，让它们以函数 owner 权限访问消息分表。

DO $$
BEGIN
    IF to_regprocedure('create_conversation_message_table(uuid)') IS NOT NULL THEN
        ALTER FUNCTION create_conversation_message_table(UUID)
            SECURITY DEFINER
            SET search_path = public, conversation_messages, pg_temp;
    END IF;

    IF to_regprocedure('drop_conversation_message_table(uuid)') IS NOT NULL THEN
        ALTER FUNCTION drop_conversation_message_table(UUID)
            SECURITY DEFINER
            SET search_path = public, conversation_messages, pg_temp;
    END IF;

    IF to_regprocedure('insert_conversation_message(uuid, uuid, text, character varying, uuid, character varying)') IS NOT NULL THEN
        ALTER FUNCTION insert_conversation_message(UUID, UUID, TEXT, VARCHAR, UUID, VARCHAR)
            SECURITY DEFINER
            SET search_path = public, conversation_messages, pg_temp;
    END IF;

    IF to_regprocedure('insert_conversation_message(uuid, uuid, text, character varying, uuid, character varying, character varying)') IS NOT NULL THEN
        ALTER FUNCTION insert_conversation_message(UUID, UUID, TEXT, VARCHAR, UUID, VARCHAR, VARCHAR)
            SECURITY DEFINER
            SET search_path = public, conversation_messages, pg_temp;
    END IF;

    IF to_regprocedure('get_conversation_messages(uuid, integer, integer)') IS NOT NULL THEN
        ALTER FUNCTION get_conversation_messages(UUID, INT, INT)
            SECURITY DEFINER
            SET search_path = public, conversation_messages, pg_temp;
    END IF;

    IF to_regprocedure('get_conversation_messages_incremental(uuid, timestamp without time zone)') IS NOT NULL THEN
        ALTER FUNCTION get_conversation_messages_incremental(UUID, TIMESTAMP)
            SECURITY DEFINER
            SET search_path = public, conversation_messages, pg_temp;
    END IF;

    IF to_regprocedure('get_conversation_message_count(uuid)') IS NOT NULL THEN
        ALTER FUNCTION get_conversation_message_count(UUID)
            SECURITY DEFINER
            SET search_path = public, conversation_messages, pg_temp;
    END IF;

    IF to_regprocedure('get_conversation_last_message(uuid)') IS NOT NULL THEN
        ALTER FUNCTION get_conversation_last_message(UUID)
            SECURITY DEFINER
            SET search_path = public, conversation_messages, pg_temp;
    END IF;

    IF to_regprocedure('get_conversation_message_by_client_id(uuid, character varying)') IS NOT NULL THEN
        ALTER FUNCTION get_conversation_message_by_client_id(UUID, VARCHAR)
            SECURITY DEFINER
            SET search_path = public, conversation_messages, pg_temp;
    END IF;
END;
$$;
