package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	databaseURL := strings.TrimSpace(os.Getenv("PULSECHECK_TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("PULSECHECK_TEST_DATABASE_URL is not set")
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var databaseName string
	if err := db.QueryRowContext(ctx, "SELECT current_database()").Scan(&databaseName); err != nil {
		t.Fatalf("identify test database: %v", err)
	}
	if databaseName != "pulsecheck_test" {
		t.Fatalf("refusing to run integration test against database %q", databaseName)
	}

	for _, table := range []string{"check_results", "monitoring_state", "alert_outbox"} {
		var exists bool
		err := db.QueryRowContext(ctx, `
			SELECT to_regclass($1) IS NOT NULL
		`, "public."+table).Scan(&exists)
		if err != nil || !exists {
			t.Fatalf("required test table %q is missing (exists=%v, err=%v)", table, exists, err)
		}
	}

	return db
}

func TestSaveCheckResultTransitions(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	probeID := fmt.Sprintf("integration-test-%d", time.Now().UnixNano())
	target := "https://integration-test.invalid"

	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM alert_outbox WHERE probe_id = $1`, probeID)
		_, _ = db.Exec(`DELETE FROM monitoring_state WHERE probe_id = $1`, probeID)
		_, _ = db.Exec(`DELETE FROM check_results WHERE probe_id = $1`, probeID)
	})

	save := func(status string) {
		t.Helper()
		err := saveCheckResult(ctx, db, CheckResult{
			ProbeID:   probeID,
			Target:    target,
			Status:    status,
			LatencyMS: 10,
			Protocol:  "HTTP",
			CheckedAt: time.Now().UTC(),
		})
		if err != nil {
			t.Fatalf("save status %s: %v", status, err)
		}
	}

	assertCounts := func(wantResults, wantAlerts int) {
		t.Helper()

		var results, alerts int
		if err := db.QueryRow(`
			SELECT COUNT(*) FROM check_results WHERE probe_id = $1
		`, probeID).Scan(&results); err != nil {
			t.Fatalf("count results: %v", err)
		}
		if err := db.QueryRow(`
			SELECT COUNT(*) FROM alert_outbox WHERE probe_id = $1
		`, probeID).Scan(&alerts); err != nil {
			t.Fatalf("count alerts: %v", err)
		}

		if results != wantResults || alerts != wantAlerts {
			t.Fatalf("got results=%d alerts=%d; want results=%d alerts=%d",
				results, alerts, wantResults, wantAlerts)
		}
	}

	save("UP")
	assertCounts(1, 0)

	save("DOWN")
	assertCounts(2, 1)

	save("DOWN")
	assertCounts(3, 1)

	save("UP")
	assertCounts(4, 2)

	var status string
	if err := db.QueryRow(`
		SELECT status FROM monitoring_state
		WHERE probe_id = $1 AND target = $2
	`, probeID, target).Scan(&status); err != nil {
		t.Fatalf("read final monitoring state: %v", err)
	}
	if status != "UP" {
		t.Fatalf("final monitoring status = %q, want UP", status)
	}

	var transitions string
	if err := db.QueryRow(`
		SELECT string_agg(previous_status || '->' || current_status, ',' ORDER BY id)
		FROM alert_outbox
		WHERE probe_id = $1
	`, probeID).Scan(&transitions); err != nil {
		t.Fatalf("read transitions: %v", err)
	}
	if transitions != "UP->DOWN,DOWN->UP" {
		t.Fatalf("transitions = %q, want UP->DOWN,DOWN->UP", transitions)
	}
}

func TestInvalidResultDoesNotChangeMonitoringState(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	probeID := fmt.Sprintf("invalid-test-%d", time.Now().UnixNano())
	target := "https://invalid-test.invalid"

	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM alert_outbox WHERE probe_id = $1`, probeID)
		_, _ = db.Exec(`DELETE FROM monitoring_state WHERE probe_id = $1`, probeID)
		_, _ = db.Exec(`DELETE FROM check_results WHERE probe_id = $1`, probeID)
	})

	save := func(status string) {
		t.Helper()
		err := saveCheckResult(ctx, db, CheckResult{
			ProbeID:   probeID,
			Target:    target,
			Status:    status,
			LatencyMS: 0,
			Protocol:  "HTTP",
			CheckedAt: time.Now().UTC(),
		})
		if err != nil {
			t.Fatalf("save status %s: %v", status, err)
		}
	}

	save("UP")
	save("INVALID")

	var state string
	if err := db.QueryRow(`
		SELECT status FROM monitoring_state
		WHERE probe_id = $1 AND target = $2
	`, probeID, target).Scan(&state); err != nil {
		t.Fatalf("read monitoring state: %v", err)
	}
	if state != "UP" {
		t.Fatalf("monitoring state after INVALID = %q, want UP", state)
	}

	var results, alerts int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM check_results WHERE probe_id = $1
	`, probeID).Scan(&results); err != nil {
		t.Fatalf("count results: %v", err)
	}
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM alert_outbox WHERE probe_id = $1
	`, probeID).Scan(&alerts); err != nil {
		t.Fatalf("count alerts: %v", err)
	}
	if results != 2 || alerts != 0 {
		t.Fatalf("got results=%d alerts=%d; want results=2 alerts=0", results, alerts)
	}
}
