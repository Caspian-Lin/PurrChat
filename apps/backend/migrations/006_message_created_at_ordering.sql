-- Ensure realtime responses and persisted history use the same backend-created timestamp,
-- and keep human trigger messages before bot replies when timestamps tie.

DROP FUNCTION IF EXISTS insert_conversation_message(UUID, UUID, TEXT, VARCHAR(20), UUID, VARCHAR(100), VARCHAR(255));

CREATE OR REPLACE FUNCTION insert_conversation_message(
    conversation_uuid UUID,
    sender_uuid UUID,
    msg_content TEXT,
    msg_type VARCHAR(20),
    bot_id UUID DEFAULT NULL,
    bot_name VARCHAR(100) DEFAULT NULL,
    client_message_id VARCHAR(255) DEFAULT NULL,
    message_created_at TIMESTAMP DEFAULT NULL
)
RETURNS UUID AS $$
DECLARE
    new_message_id UUID;
    table_name TEXT;
    created_at_value TIMESTAMP;
BEGIN
    table_name := replace(conversation_uuid::TEXT, '-', '_');
    created_at_value := COALESCE(message_created_at, CURRENT_TIMESTAMP);

    IF client_message_id IS NOT NULL THEN
        EXECUTE format('
            SELECT id FROM conversation_messages.%I
            WHERE client_message_id = $1
        ', table_name)
        INTO new_message_id
        USING client_message_id;

        IF new_message_id IS NOT NULL THEN
            RETURN new_message_id;
        END IF;
    END IF;

    EXECUTE format('
        INSERT INTO conversation_messages.%I (sender_id, content, msg_type, bot_id, bot_name, client_message_id, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id
    ', table_name)
    INTO new_message_id
    USING sender_uuid, msg_content, msg_type, bot_id, bot_name, client_message_id, created_at_value;

    RETURN new_message_id;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION get_conversation_messages(
    conversation_uuid UUID,
    msg_limit INT DEFAULT 50,
    msg_offset INT DEFAULT 0
)
RETURNS TABLE (
    id UUID,
    sender_id UUID,
    content TEXT,
    msg_type VARCHAR(20),
    created_at TIMESTAMP,
    bot_id UUID,
    bot_name VARCHAR(100)
) AS $$
DECLARE
    table_name TEXT;
BEGIN
    table_name := replace(conversation_uuid::TEXT, '-', '_');
    RETURN QUERY EXECUTE format('
        SELECT id, sender_id, content, msg_type, created_at, bot_id, bot_name
        FROM conversation_messages.%I
        ORDER BY created_at DESC, bot_id NULLS LAST
        LIMIT $1 OFFSET $2
    ', table_name)
    USING msg_limit, msg_offset;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION get_conversation_messages_incremental(
    conversation_uuid UUID,
    since_timestamp TIMESTAMP
)
RETURNS TABLE (
    id UUID,
    sender_id UUID,
    content TEXT,
    msg_type VARCHAR(20),
    created_at TIMESTAMP,
    bot_id UUID,
    bot_name VARCHAR(100)
) AS $$
DECLARE
    table_name TEXT;
BEGIN
    table_name := replace(conversation_uuid::TEXT, '-', '_');
    RETURN QUERY EXECUTE format('
        SELECT id, sender_id, content, msg_type, created_at, bot_id, bot_name
        FROM conversation_messages.%I
        WHERE created_at > $1
        ORDER BY created_at ASC, bot_id NULLS FIRST
    ', table_name)
    USING since_timestamp;
END;
$$ LANGUAGE plpgsql;
