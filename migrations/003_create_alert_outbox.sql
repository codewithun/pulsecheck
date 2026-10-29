CREATE TABLE IF NOT EXISTS alert_outbox (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    probe_id VARCHAR(100) NOT NULL,
    target TEXT NOT NULL,
    previous_status VARCHAR(10) NOT NULL,
    current_status VARCHAR(10) NOT NULL
        CHECK (current_status IN ('UP', 'DOWN')),
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    delivered_at TIMESTAMPTZ,
    attempts INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_alert_outbox_pending
    ON alert_outbox (next_attempt_at, id)
    WHERE delivered_at IS NULL;
