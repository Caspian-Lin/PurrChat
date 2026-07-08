-- Keep human trigger messages before bot replies when message timestamps tie.
-- Remove the transient 8-argument variant if it was applied locally before this migration was fixed.
DROP FUNCTION IF EXISTS insert_conversation_message(UUID, UUID, TEXT, VARCHAR(20), UUID, VARCHAR(100), VARCHAR(255), TIMESTAMP);

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
