-- Event outbox for at-least-once delivery to external bots.
-- Each event is persisted before WS push; bots can ACK and resume.
CREATE TABLE IF NOT EXISTS bot_event_seq_counter (
    bot_id   UUID PRIMARY KEY REFERENCES bots(id) ON DELETE CASCADE,
    next_seq BIGINT NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS bot_event_outbox (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_id     UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    event_id   TEXT NOT NULL,
    seq        BIGINT NOT NULL,
    payload    JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    acked_at   TIMESTAMP,
    CONSTRAINT uq_bot_event_outbox_bot_seq UNIQUE (bot_id, seq),
    CONSTRAINT uq_bot_event_outbox_bot_event UNIQUE (bot_id, event_id)
);

CREATE INDEX IF NOT EXISTS idx_bot_event_outbox_resume
    ON bot_event_outbox (bot_id, seq);
CREATE INDEX IF NOT EXISTS idx_bot_event_outbox_created
    ON bot_event_outbox (created_at);

-- Per-credential ACK position: the highest seq acknowledged by this credential.
CREATE TABLE IF NOT EXISTS bot_event_ack_state (
    credential_id  UUID NOT NULL REFERENCES bot_api_credentials(id) ON DELETE CASCADE,
    bot_id         UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    last_acked_seq BIGINT NOT NULL DEFAULT 0,
    updated_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (credential_id, bot_id)
);
