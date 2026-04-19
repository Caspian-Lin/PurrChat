-- PurrChat Bot 机制配置重构
-- 将 trigger_config + reply_config + special_mode_config 统一为 mechanism_config

-- 新增机制配置列
ALTER TABLE bots ADD COLUMN IF NOT EXISTS mechanism_config JSONB NOT NULL DEFAULT '[]'::jsonb;

-- 辅助函数：从条件表达式中提取 pattern
-- 支持 includes('text'), startsWith('text'), equals('text'), regex('pattern')
CREATE OR REPLACE FUNCTION extract_pattern_from_expression(expr TEXT)
RETURNS TEXT AS $$
DECLARE
    result TEXT;
BEGIN
    IF expr IS NULL OR expr = '' THEN
        RETURN '';
    END IF;

    -- includes('text') -> text
    IF expr LIKE 'includes(%)%' THEN
        result := substring(expr FROM 'includes\(''([^'']*)''');
        IF result IS NOT NULL THEN RETURN result; END IF;
        result := substring(expr FROM 'includes\("([^"]*)"\)');
        IF result IS NOT NULL THEN RETURN result; END IF;
        RETURN '';
    END IF;

    -- startsWith('text') -> text
    IF expr LIKE 'startsWith(%)%' THEN
        result := substring(expr FROM 'startsWith\(''([^'']*)''');
        IF result IS NOT NULL THEN RETURN result; END IF;
        result := substring(expr FROM 'startsWith\("([^"]*)"\)');
        IF result IS NOT NULL THEN RETURN result; END IF;
        RETURN '';
    END IF;

    -- equals('text') -> text
    IF expr LIKE 'equals(%)%' THEN
        result := substring(expr FROM 'equals\(''([^'']*)''');
        IF result IS NOT NULL THEN RETURN result; END IF;
        result := substring(expr FROM 'equals\("([^"]*)"\)');
        IF result IS NOT NULL THEN RETURN result; END IF;
        RETURN '';
    END IF;

    -- regex('pattern') -> pattern
    IF expr LIKE 'regex(%)%' THEN
        result := substring(expr FROM 'regex\(''([^'']*)''');
        IF result IS NOT NULL THEN RETURN result; END IF;
        result := substring(expr FROM 'regex\("([^"]*)"\)');
        IF result IS NOT NULL THEN RETURN result; END IF;
        RETURN '';
    END IF;

    RETURN expr;
END;
$$ LANGUAGE plpgsql;

-- 辅助函数：将旧 trigger_config 转换为新的 trigger_spec JSONB
CREATE OR REPLACE FUNCTION migrate_trigger_spec(tc JSONB)
RETURNS JSONB AS $$
DECLARE
    mode TEXT;
    result JSONB;
BEGIN
    IF tc IS NULL OR tc = '{}'::jsonb OR tc = 'null'::jsonb THEN
        RETURN '{"type":"rule","rules":[]}'::jsonb;
    END IF;

    mode := COALESCE(tc->>'mode', 'rule');

    -- 概率模式
    IF mode = 'probability' THEN
        RETURN jsonb_build_object(
            'type', 'probability',
            'probability', COALESCE((tc->>'probability')::float8, 0.3)
        );
    END IF;

    -- 条件模式：转换为规则模式（从 start_expression 提取 pattern）
    IF mode = 'conditional' THEN
        IF tc->'condition' IS NOT NULL THEN
            DECLARE
                start_expr TEXT;
                pattern TEXT;
                rule_type TEXT;
            BEGIN
                start_expr := COALESCE(tc->'condition'->>'start_expression', '');
                pattern := extract_pattern_from_expression(start_expr);

                -- 根据表达式类型推断规则类型
                IF start_expr LIKE 'regex(%)%' THEN
                    rule_type := 'regex';
                ELSIF start_expr LIKE 'equals(%)%' THEN
                    rule_type := 'equals';
                ELSIF start_expr LIKE 'startsWith(%)%' THEN
                    rule_type := 'command';
                ELSE
                    rule_type := 'keyword';
                END IF;

                IF pattern != '' THEN
                    RETURN jsonb_build_object(
                        'type', 'rule',
                        'rules', jsonb_build_array(
                            jsonb_build_object(
                                'type', rule_type,
                                'pattern', pattern,
                                'case_sensitive', false
                            )
                        )
                    );
                END IF;
            END;
        END IF;
        RETURN '{"type":"rule","rules":[]}'::jsonb;
    END IF;

    -- 规则模式（直接复用）
    RETURN jsonb_build_object(
        'type', 'rule',
        'rules', COALESCE(tc->'rules', '[]'::jsonb)
    );
END;
$$ LANGUAGE plpgsql;

-- 辅助函数：将旧 reply_config 转换为新的 reply_spec JSONB
CREATE OR REPLACE FUNCTION migrate_reply_spec(rc JSONB)
RETURNS JSONB AS $$
BEGIN
    IF rc IS NULL OR rc = '{}'::jsonb OR rc = 'null'::jsonb THEN
        RETURN '{"type":"predefined","predefined":{"mode":"random","replies":["..."]}}'::jsonb;
    END IF;
    RETURN rc;
END;
$$ LANGUAGE plpgsql;

-- 迁移现有数据：将旧三字段合并为机制列表
UPDATE bots SET mechanism_config = (
    SELECT jsonb_build_object('mechanisms', jsonb_agg(mech ORDER BY ord))
    FROM (
        -- 主机制：trigger + reply
        SELECT
            jsonb_build_object(
                'id', 'migrated_' || id::text || '_main',
                'name', '默认机制',
                'enabled', true,
                'trigger', migrate_trigger_spec(trigger_config),
                'reply', migrate_reply_spec(reply_config)
            ) AS mech,
            0 AS ord
        FROM bots b2
        WHERE b2.id = bots.id

        UNION ALL

        -- 特殊模式机制（如果存在且有事件）
        SELECT
            jsonb_build_object(
                'id', 'migrated_' || id::text || '_special',
                'name', '特殊模式',
                'enabled', true,
                'trigger', '{"type":"rule","rules":[]}'::jsonb,
                'reply', jsonb_build_object(
                    'type', 'special_mode',
                    'special_mode', special_mode_config
                )
            ) AS mech,
            1 AS ord
        FROM bots b2
        WHERE b2.id = bots.id
          AND special_mode_config IS NOT NULL
          AND special_mode_config != 'null'::jsonb
          AND jsonb_array_length(special_mode_config->'events') > 0
    ) AS sub
)
WHERE mechanism_config = '[]'::jsonb
  AND (trigger_config IS NOT NULL AND trigger_config != '{}'::jsonb
       OR reply_config IS NOT NULL AND reply_config != '{}'::jsonb
       OR special_mode_config IS NOT NULL AND special_mode_config != 'null'::jsonb);

-- 为没有旧配置的 Bot 设置默认机制
UPDATE bots SET mechanism_config = '[{"id":"mech_default","name":"默认机制","enabled":true,"trigger":{"type":"rule","rules":[]},"reply":{"type":"predefined","predefined":{"mode":"random","replies":["..."]}}}]'::jsonb
WHERE mechanism_config = '[]'::jsonb;

COMMENT ON COLUMN bots.mechanism_config IS 'Bot 机制配置（JSON 数组，每个机制包含触发规则和回复设置）';
