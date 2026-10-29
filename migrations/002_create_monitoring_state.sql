CREATE TABLE IF NOT EXISTS monitoring_state (
    probe_id VARCHAR(100) NOT NULL,
    target TEXT NOT NULL,
    status VARCHAR(10) NOT NULL
        CHECK (status IN ('UP', 'DOWN', 'INVALID')),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (probe_id, target)
);
