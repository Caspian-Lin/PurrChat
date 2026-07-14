-- Backfill permissions for Bots configured through the legacy mechanism editor.
-- These Bots do not pass through Workflow Document publication, so earlier versions
-- left requested_capabilities empty even though message execution requires them.
WITH derived AS (
    SELECT
        b.id,
        ARRAY_REMOVE(ARRAY[
            CASE WHEN EXISTS (
                SELECT 1
                FROM jsonb_array_elements(COALESCE(b.mechanism_config::jsonb -> 'mechanisms', '[]'::jsonb)) AS m
                WHERE COALESCE((m ->> 'enabled')::boolean, TRUE)
            ) THEN 'messages:read_trigger' END,
            CASE WHEN EXISTS (
                SELECT 1
                FROM jsonb_array_elements(COALESCE(b.mechanism_config::jsonb -> 'mechanisms', '[]'::jsonb)) AS m
                WHERE COALESCE((m ->> 'enabled')::boolean, TRUE)
                  AND m -> 'reply' ->> 'type' IN ('predefined', 'llm')
            ) OR EXISTS (
                SELECT 1
                FROM jsonb_array_elements(COALESCE(b.mechanism_config::jsonb -> 'mechanisms', '[]'::jsonb)) AS m,
                     jsonb_array_elements(COALESCE(m -> 'reply' -> 'workflow' -> 'events', '[]'::jsonb)) AS e
                WHERE COALESCE((m ->> 'enabled')::boolean, TRUE)
                  AND e ->> 'type' IN ('reply', 'template')
            ) THEN 'messages:send' END,
            CASE WHEN EXISTS (
                SELECT 1
                FROM jsonb_array_elements(COALESCE(b.mechanism_config::jsonb -> 'mechanisms', '[]'::jsonb)) AS m
                WHERE COALESCE((m ->> 'enabled')::boolean, TRUE)
                  AND m -> 'reply' ->> 'type' = 'llm'
            ) OR EXISTS (
                SELECT 1
                FROM jsonb_array_elements(COALESCE(b.mechanism_config::jsonb -> 'mechanisms', '[]'::jsonb)) AS m,
                     jsonb_array_elements(COALESCE(m -> 'reply' -> 'workflow' -> 'events', '[]'::jsonb)) AS e
                WHERE COALESCE((m ->> 'enabled')::boolean, TRUE)
                  AND e ->> 'type' IN ('llm', 'history')
            ) THEN 'messages:read_history' END,
            CASE WHEN EXISTS (
                SELECT 1
                FROM jsonb_array_elements(COALESCE(b.mechanism_config::jsonb -> 'mechanisms', '[]'::jsonb)) AS m
                WHERE COALESCE((m ->> 'enabled')::boolean, TRUE)
                  AND m -> 'reply' ->> 'type' = 'llm'
            ) OR EXISTS (
                SELECT 1
                FROM jsonb_array_elements(COALESCE(b.mechanism_config::jsonb -> 'mechanisms', '[]'::jsonb)) AS m,
                     jsonb_array_elements(COALESCE(m -> 'reply' -> 'workflow' -> 'events', '[]'::jsonb)) AS e
                WHERE COALESCE((m ->> 'enabled')::boolean, TRUE)
                  AND e ->> 'type' IN ('llm', 'tool', 'dify', 'n8n')
            ) THEN 'network:external' END
        ]::text[], NULL) AS capabilities
    FROM bots b
    WHERE cardinality(b.requested_capabilities) = 0
)
UPDATE bots b
SET requested_capabilities = d.capabilities,
    updated_at = NOW()
FROM derived d
WHERE b.id = d.id
  AND cardinality(d.capabilities) > 0;

UPDATE bot_installations i
SET granted_capabilities = b.requested_capabilities,
    diagnostics_consent = CASE
        WHEN b.requested_capabilities @> ARRAY['network:external']::text[] THEN 'granted'
        ELSE i.diagnostics_consent
    END,
    updated_at = NOW()
FROM bots b
WHERE i.app_id = b.id
  AND i.installed_by = b.owner_id
  AND cardinality(i.granted_capabilities) = 0
  AND cardinality(b.requested_capabilities) > 0;
