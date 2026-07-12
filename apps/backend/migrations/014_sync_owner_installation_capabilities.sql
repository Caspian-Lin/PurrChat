-- 修复首次发布工作流前创建的 owner 私聊/群聊安装权限为空。
-- 仅同步 Bot 创建者本人安装的记录，避免给第三方安装自动扩权。
UPDATE bot_installations AS i
SET granted_capabilities = COALESCE(b.requested_capabilities, ARRAY[]::text[]),
    diagnostics_consent = CASE
        WHEN b.requested_capabilities @> ARRAY['network:external']::text[] THEN 'granted'
        ELSE i.diagnostics_consent
    END,
    updated_at = NOW()
FROM bots AS b
WHERE i.app_id = b.id
  AND i.installed_by = b.owner_id
  AND (
      i.granted_capabilities IS DISTINCT FROM COALESCE(b.requested_capabilities, ARRAY[]::text[])
      OR (
          b.requested_capabilities @> ARRAY['network:external']::text[]
          AND i.diagnostics_consent <> 'granted'
      )
  );
