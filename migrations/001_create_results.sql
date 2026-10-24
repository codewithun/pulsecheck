
CREATE TABLE IF NOT EXISTS check_results (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    probe_id VARCHAR(100) NOT NULL,
    target TEXT NOT NULL,
    status VARCHAR(10) NOT NULL
        CHECK (status IN ('UP', 'DOWN', 'INVALID')),
    status_code INTEGER,
    latency_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
    error TEXT,
    protocol VARCHAR(10) NOT NULL,
    checked_at TIMESTAMPTZ NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_check_results_probe_checked_at
    ON check_results (probe_id, checked_at DESC);

CREATE INDEX IF NOT EXISTS idx_check_results_target_checked_at
    ON check_results (target, checked_at DESC);
