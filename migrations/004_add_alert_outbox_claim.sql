ALTER TABLE alert_outbox
    ADD COLUMN IF NOT EXISTS claimed_until TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_alert_outbox_claimable
    ON alert_outbox (next_attempt_at, id)
    WHERE delivered_at IS NULL;
