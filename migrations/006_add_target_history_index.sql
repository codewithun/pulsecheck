CREATE INDEX IF NOT EXISTS idx_check_results_probe_target_checked_at
ON check_results (probe_id, target, checked_at DESC, id DESC);
