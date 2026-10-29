package main

import (
	"context"
	"database/sql"
	"fmt"
)

func saveCheckResult(ctx context.Context, db *sql.DB, result CheckResult) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO check_results
			(probe_id, target, status, status_code,
			 latency_ms, error, protocol, checked_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`,
		result.ProbeID,
		result.Target,
		result.Status,
		result.StatusCode,
		result.LatencyMS,
		nullIfEmpty(result.Error),
		result.Protocol,
		result.CheckedAt,
	)
	if err != nil {
		return fmt.Errorf("insert check result: %w", err)
	}

	// INVALID is stored as a check result, but never changes operational state.
	if result.Status != "INVALID" {
		var inserted bool

		err = tx.QueryRowContext(ctx, `
			INSERT INTO monitoring_state (probe_id, target, status)
			VALUES ($1, $2, $3)
			ON CONFLICT (probe_id, target) DO NOTHING
			RETURNING TRUE
		`, result.ProbeID, result.Target, result.Status).Scan(&inserted)

		if err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("initialize monitoring state: %w", err)
		}

		// A new state is the baseline; do not emit an initial alert.
		if err == sql.ErrNoRows {
			var previousStatus string

			if err := tx.QueryRowContext(ctx, `
				SELECT status
				FROM monitoring_state
				WHERE probe_id = $1 AND target = $2
				FOR UPDATE
			`, result.ProbeID, result.Target).Scan(&previousStatus); err != nil {
				return fmt.Errorf("read monitoring state: %w", err)
			}

			if previousStatus != result.Status {
				_, err = tx.ExecContext(ctx, `
					UPDATE monitoring_state
					SET status = $3, updated_at = NOW()
					WHERE probe_id = $1 AND target = $2
				`, result.ProbeID, result.Target, result.Status)
				if err != nil {
					return fmt.Errorf("update monitoring state: %w", err)
				}

				if (previousStatus == "UP" && result.Status == "DOWN") ||
					(previousStatus == "DOWN" && result.Status == "UP") {
					_, err = tx.ExecContext(ctx, `
						INSERT INTO alert_outbox
							(probe_id, target, previous_status, current_status, error)
						VALUES ($1, $2, $3, $4, $5)
					`,
						result.ProbeID,
						result.Target,
						previousStatus,
						result.Status,
						nullIfEmpty(result.Error),
					)
					if err != nil {
						return fmt.Errorf("queue alert: %w", err)
					}
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit monitoring result: %w", err)
	}

	return nil
}
