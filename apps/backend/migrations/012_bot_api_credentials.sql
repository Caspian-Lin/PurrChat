-- External Bot API credentials. Plaintext tokens are never persisted.
CREATE TABLE IF NOT EXISTS bot_api_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    name VARCHAR(64) NOT NULL,
    token_hash BYTEA NOT NULL UNIQUE,
    token_prefix VARCHAR(20) NOT NULL,
    last_used_at TIMESTAMP,
    expires_at TIMESTAMP,
    revoked_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT check_bot_api_credential_name CHECK (char_length(trim(name)) > 0)
);

CREATE INDEX IF NOT EXISTS idx_bot_api_credentials_bot ON bot_api_credentials (bot_id, created_at DESC);

CREATE TABLE IF NOT EXISTS bot_api_credential_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    credential_id UUID NOT NULL REFERENCES bot_api_credentials(id) ON DELETE CASCADE,
    bot_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
    event_type VARCHAR(32) NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT check_bot_api_credential_audit_event CHECK (
        event_type IN ('created', 'rotated', 'revoked', 'connected', 'invoked')
    )
);

CREATE INDEX IF NOT EXISTS idx_bot_api_credential_audit_credential
    ON bot_api_credential_audit_logs (credential_id, created_at DESC);
