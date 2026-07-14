-- #36: 存量数据迁移至 bot_installations
-- 1. bot_deployments → bot_installations(target_type=conversation)
-- 2. Bot friendship → bot_installations(target_type=user)
-- 幂等:ON CONFLICT (target_type, target_id, app_id) DO NOTHING

-- 1. 回填群聊部署记录
INSERT INTO bot_installations (app_id, installed_by, target_type, target_id, granted_capabilities, diagnostics_consent, status, installed_at)
SELECT
    d.bot_id,
    d.deployed_by,
    'conversation',
    d.conversation_id,
    COALESCE(b.requested_capabilities, ARRAY[]::text[]),
    CASE
        WHEN b.requested_capabilities @> ARRAY['network:external']::text[] THEN 'granted'
        ELSE 'denied'
    END,
    CASE WHEN d.status = 'paused' THEN 'paused' ELSE 'active' END,
    d.deployed_at
FROM bot_deployments d
JOIN bots b ON d.bot_id = b.id
ON CONFLICT (target_type, target_id, app_id) DO NOTHING;

-- 2. 回填私聊 Bot friendship(friend_id 指向 Bot 的记录)
INSERT INTO bot_installations (app_id, installed_by, target_type, target_id, granted_capabilities, diagnostics_consent, status, installed_at)
SELECT
    f.friend_id,
    f.user_id,
    'user',
    f.user_id,
    COALESCE(b.requested_capabilities, ARRAY[]::text[]),
    CASE
        WHEN b.requested_capabilities @> ARRAY['network:external']::text[] THEN 'granted'
        ELSE 'denied'
    END,
    CASE WHEN f.status = 'accepted' THEN 'active' ELSE 'paused' END,
    f.created_at
FROM friendships f
JOIN users u ON f.friend_id = u.id AND u.is_bot = true
JOIN bots b ON f.friend_id = b.id
ON CONFLICT (target_type, target_id, app_id) DO NOTHING;
