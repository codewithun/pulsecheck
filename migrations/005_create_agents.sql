CREATE TABLE IF NOT EXISTS agents (
    probe_id VARCHAR(100) PRIMARY KEY,
    hostname VARCHAR(255) NOT NULL,
    last_seen TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agents_last_seen
ON agents (last_seen DESC);
