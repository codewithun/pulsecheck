package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOverviewIntegration(t *testing.T) {
	db := openTestDB(t)

	probeID := fmt.Sprintf("overview-test-%d", time.Now().UnixNano())

	targetUP := "https://overview-up.invalid"
	targetDOWN := "https://overview-down.invalid"
	targetUNKNOWN := "https://overview-unknown.invalid"

	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM alert_outbox WHERE probe_id = $1`, probeID)
		_, _ = db.Exec(`DELETE FROM monitoring_state WHERE probe_id = $1`, probeID)
		_, _ = db.Exec(`DELETE FROM check_results WHERE probe_id = $1`, probeID)
		_, _ = db.Exec(`DELETE FROM agents WHERE probe_id = $1`, probeID)
	})

	// Create one online and one offline agent.
	_, err := db.Exec(`
		INSERT INTO agents (probe_id, hostname, last_seen)
		VALUES
			($1, 'overview-online', NOW()),
			($2, 'overview-offline', NOW() - INTERVAL '120 seconds')
	`, probeID, probeID+"-offline")
	if err != nil {
		t.Fatalf("insert agents: %v", err)
	}

	// Create operational states.
	_, err = db.Exec(`
		INSERT INTO monitoring_state (probe_id, target, status)
		VALUES
			($1, $2, 'UP'),
			($1, $3, 'DOWN')
	`, probeID, targetUP, targetDOWN)
	if err != nil {
		t.Fatalf("insert monitoring states: %v", err)
	}

	// Create four checks: two UP, one DOWN, and one INVALID.
	_, err = db.Exec(`
		INSERT INTO check_results
			(probe_id, target, status, latency_ms, protocol, checked_at)
		VALUES
			($1, $2, 'UP', 10, 'HTTP', NOW()),
			($1, $2, 'UP', 12, 'HTTP', NOW()),
			($1, $3, 'DOWN', 20, 'HTTP', NOW()),
			($1, $4, 'INVALID', 0, 'HTTP', NOW())
	`, probeID, targetUP, targetDOWN, targetUNKNOWN)
	if err != nil {
		t.Fatalf("insert check results: %v", err)
	}

	// Create one pending and one delivered alert.
	_, err = db.Exec(`
		INSERT INTO alert_outbox
			(probe_id, target, previous_status, current_status, delivered_at)
		VALUES
			($1, $2, 'UP', 'DOWN', NULL),
			($1, $3, 'DOWN', 'UP', NOW())
	`, probeID, targetUP, targetDOWN)
	if err != nil {
		t.Fatalf("insert alert events: %v", err)
	}

	handler := overviewHandler(db)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/overview?period=24h",
		nil,
	)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s",
			rec.Code, http.StatusOK, rec.Body.String())
	}

	var response OverviewResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Period != "24h" {
		t.Errorf("period = %q, want 24h", response.Period)
	}

	if response.Agents.Total != 2 ||
		response.Agents.Online != 1 ||
		response.Agents.Offline != 1 {
		t.Errorf("unexpected agent stats: %+v", response.Agents)
	}

	if response.Targets.Total != 3 ||
		response.Targets.UP != 1 ||
		response.Targets.DOWN != 1 ||
		response.Targets.UNKNOWN != 1 {
		t.Errorf("unexpected target stats: %+v", response.Targets)
	}

	if response.Checks.Total != 4 ||
		response.Checks.UP != 2 ||
		response.Checks.DOWN != 1 ||
		response.Checks.INVALID != 1 {
		t.Errorf("unexpected check stats: %+v", response.Checks)
	}

	if response.Availability == nil {
		t.Fatal("availability is nil, want 66.67")
	}

	if math.Abs(*response.Availability-66.67) > 0.001 {
		t.Errorf("availability = %.2f, want 66.67", *response.Availability)
	}

	if response.Alerts.Pending != 1 ||
		response.Alerts.Delivered != 1 {
		t.Errorf("unexpected alert stats: %+v", response.Alerts)
	}
}

func TestOverviewRejectsInvalidPeriod(t *testing.T) {
	db := openTestDB(t)

	handler := overviewHandler(db)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/overview?period=invalid",
		nil,
	)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s",
			rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestOverviewAvailabilityIsNilWithoutValidChecks(t *testing.T) {
	db := openTestDB(t)

	probeID := fmt.Sprintf("overview-invalid-only-%d", time.Now().UnixNano())
	target := "https://overview-invalid-only.invalid"

	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM check_results WHERE probe_id = $1`, probeID)
	})

	_, err := db.Exec(`
		INSERT INTO check_results
			(probe_id, target, status, latency_ms, protocol, checked_at)
		VALUES ($1, $2, 'INVALID', 0, 'HTTP', NOW())
	`, probeID, target)
	if err != nil {
		t.Fatalf("insert invalid check: %v", err)
	}

	handler := overviewHandler(db)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/overview?period=1h",
		nil,
	)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s",
			rec.Code, http.StatusOK, rec.Body.String())
	}

	var response OverviewResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Availability != nil {
		t.Errorf("availability = %v, want nil", *response.Availability)
	}
}
