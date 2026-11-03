package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTargetsIntegration(t *testing.T) {
	db := openTestDB(t)

	probeID := fmt.Sprintf("targets-test-%d", time.Now().UnixNano())

	targetUP := "https://targets-up.invalid"
	targetDOWN := "https://targets-down.invalid"
	targetUNKNOWN := "https://targets-unknown.invalid"
	targetOld := "https://targets-old.invalid"

	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM monitoring_state WHERE probe_id = $1`, probeID)
		_, _ = db.Exec(`DELETE FROM check_results WHERE probe_id = $1`, probeID)
	})

	// Operational states.
	_, err := db.Exec(`
		INSERT INTO monitoring_state (probe_id, target, status)
		VALUES
			($1, $2, 'UP'),
			($1, $3, 'DOWN'),
			($1, $4, 'UP')
	`, probeID, targetUP, targetDOWN, targetOld)
	if err != nil {
		t.Fatalf("insert monitoring states: %v", err)
	}

	// Insert recent checks.
	_, err = db.Exec(`
		INSERT INTO check_results
			(probe_id, target, status, latency_ms, protocol, error, checked_at)
		VALUES
			($1, $2, 'UP', 10, 'HTTP', NULL, NOW() - INTERVAL '2 minutes'),
			($1, $2, 'INVALID', 0, 'HTTP', 'temporary error', NOW()),
			($1, $3, 'DOWN', 20, 'TCP', 'connection refused', NOW()),
			($1, $4, 'INVALID', 0, 'HTTP', 'invalid response', NOW()),
			($1, $5, 'UP', 30, 'HTTP', NULL, NOW() - INTERVAL '2 hours')
	`, probeID, targetUP, targetDOWN, targetUNKNOWN, targetOld)
	if err != nil {
		t.Fatalf("insert check results: %v", err)
	}

	handler := targetsHandler(db)

	request := func(path string) (*httptest.ResponseRecorder, TargetsResponse) {
		t.Helper()

		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		var response TargetsResponse
		if rec.Code == http.StatusOK {
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
		}

		return rec, response
	}

	t.Run("all historical targets", func(t *testing.T) {
		rec, response := request("/api/v1/targets?probe_id=" + probeID)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s",
				rec.Code, http.StatusOK, rec.Body.String())
		}

		if len(response.Data) != 4 {
			t.Fatalf("targets = %d, want 4", len(response.Data))
		}

		if response.HasMore {
			t.Error("has_more = true, want false")
		}

		found := make(map[string]TargetItem)
		for _, item := range response.Data {
			found[item.Target] = item
		}

		up, ok := found[targetUP]
		if !ok {
			t.Fatal("UP target not found")
		}

		if up.Status != "UP" {
			t.Errorf("UP target status = %q, want UP", up.Status)
		}

		if up.LastCheckStatus != "INVALID" {
			t.Errorf("UP target last_check_status = %q, want INVALID",
				up.LastCheckStatus)
		}

		if up.Error != "temporary error" {
			t.Errorf("UP target error = %q, want temporary error", up.Error)
		}

		down, ok := found[targetDOWN]
		if !ok {
			t.Fatal("DOWN target not found")
		}

		if down.Status != "DOWN" || down.LastCheckStatus != "DOWN" {
			t.Errorf("unexpected DOWN target: %+v", down)
		}

		unknown, ok := found[targetUNKNOWN]
		if !ok {
			t.Fatal("UNKNOWN target not found")
		}

		if unknown.Status != "UNKNOWN" ||
			unknown.LastCheckStatus != "INVALID" {
			t.Errorf("unexpected UNKNOWN target: %+v", unknown)
		}

		if _, ok := found[targetOld]; !ok {
			t.Error("historical target not found")
		}
	})

	t.Run("period filter", func(t *testing.T) {
		rec, response := request(
			"/api/v1/targets?probe_id=" + probeID + "&period=1h",
		)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s",
				rec.Code, http.StatusOK, rec.Body.String())
		}

		if len(response.Data) != 3 {
			t.Errorf("targets = %d, want 3", len(response.Data))
		}

		for _, item := range response.Data {
			if item.Target == targetOld {
				t.Error("old target should not appear in 1h period")
			}
		}
	})

	t.Run("status filter", func(t *testing.T) {
		rec, response := request(
			"/api/v1/targets?probe_id=" + probeID + "&status=DOWN",
		)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s",
				rec.Code, http.StatusOK, rec.Body.String())
		}

		if len(response.Data) != 1 ||
			response.Data[0].Target != targetDOWN {
			t.Errorf("unexpected DOWN filter response: %+v", response.Data)
		}
	})

	t.Run("target search", func(t *testing.T) {
		rec, response := request(
			"/api/v1/targets?probe_id=" + probeID + "&target=targets-up",
		)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s",
				rec.Code, http.StatusOK, rec.Body.String())
		}

		if len(response.Data) != 1 ||
			response.Data[0].Target != targetUP {
			t.Errorf("unexpected target search response: %+v", response.Data)
		}
	})

	t.Run("pagination", func(t *testing.T) {
		rec, response := request(
			"/api/v1/targets?probe_id=" + probeID + "&limit=2",
		)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s",
				rec.Code, http.StatusOK, rec.Body.String())
		}

		if len(response.Data) != 2 {
			t.Errorf("targets = %d, want 2", len(response.Data))
		}

		if !response.HasMore {
			t.Error("has_more = false, want true")
		}

		if response.Limit != 2 || response.Offset != 0 {
			t.Errorf("unexpected pagination: %+v", response)
		}
	})
}

func TestTargetsRejectsInvalidParameters(t *testing.T) {
	db := openTestDB(t)
	handler := targetsHandler(db)

	testCases := []struct {
		name string
		path string
	}{
		{"invalid limit", "/api/v1/targets?limit=101"},
		{"zero limit", "/api/v1/targets?limit=0"},
		{"negative offset", "/api/v1/targets?offset=-1"},
		{"invalid status", "/api/v1/targets?status=INVALID"},
		{"invalid period", "/api/v1/targets?period=invalid"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d; body=%s",
					rec.Code, http.StatusBadRequest, rec.Body.String())
			}
		})
	}
}
