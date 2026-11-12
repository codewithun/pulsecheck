package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAgentDetailIntegration(t *testing.T) {
	db := openTestDB(t)

	probeID := fmt.Sprintf("agent-detail-test-%d", time.Now().UnixNano())

	targetUP := "https://agent-detail-up.invalid"
	targetDOWN := "https://agent-detail-down.invalid"
	targetUNKNOWN := "https://agent-detail-unknown.invalid"

	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM monitoring_state WHERE probe_id = $1`, probeID)
		_, _ = db.Exec(`DELETE FROM check_results WHERE probe_id = $1`, probeID)
		_, _ = db.Exec(`DELETE FROM agents WHERE probe_id = $1`, probeID)
	})

	_, err := db.Exec(`
		INSERT INTO agents (probe_id, hostname, last_seen)
		VALUES ($1, 'agent-detail-host', NOW())
	`, probeID)
	if err != nil {
		t.Fatalf("insert agent: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO monitoring_state (probe_id, target, status)
		VALUES
			($1, $2, 'UP'),
			($1, $3, 'DOWN'),
			($1, $4, 'INVALID')
	`, probeID, targetUP, targetDOWN, targetUNKNOWN)
	if err != nil {
		t.Fatalf("insert monitoring states: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO check_results
			(probe_id, target, status, latency_ms, protocol, error, checked_at)
		VALUES
			($1, $2, 'UP', 10, 'HTTP', NULL, NOW()),
			($1, $3, 'DOWN', 20, 'TCP', 'connection refused', NOW()),
			($1, $4, 'INVALID', 0, 'HTTP', 'invalid response', NOW())
	`, probeID, targetUP, targetDOWN, targetUNKNOWN)
	if err != nil {
		t.Fatalf("insert check results: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle(
		"GET /api/v1/agents/{probe_id}",
		agentDetailHandler(db),
	)

	request := func(path string) (*httptest.ResponseRecorder, AgentDetailResponse) {
		t.Helper()

		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		var response AgentDetailResponse
		if rec.Code == http.StatusOK {
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
		}

		return rec, response
	}

	t.Run("agent detail and target summary", func(t *testing.T) {
		rec, response := request("/api/v1/agents/" + probeID)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s",
				rec.Code, http.StatusOK, rec.Body.String())
		}

		if response.Agent.ProbeID != probeID {
			t.Errorf("probe_id = %q, want %q",
				response.Agent.ProbeID, probeID)
		}

		if response.Agent.Status != "ONLINE" {
			t.Errorf("agent status = %q, want ONLINE",
				response.Agent.Status)
		}

		if response.Summary.Total != 3 ||
			response.Summary.UP != 1 ||
			response.Summary.DOWN != 1 ||
			response.Summary.Unknown != 1 {
			t.Errorf("unexpected summary: %+v", response.Summary)
		}

		if len(response.Targets.Data) != 3 {
			t.Errorf("targets = %d, want 3", len(response.Targets.Data))
		}

		if response.Targets.HasMore {
			t.Error("has_more = true, want false")
		}
	})

	t.Run("pagination", func(t *testing.T) {
		rec, response := request(
			"/api/v1/agents/" + probeID + "?limit=2",
		)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s",
				rec.Code, http.StatusOK, rec.Body.String())
		}

		if len(response.Targets.Data) != 2 {
			t.Errorf("targets = %d, want 2", len(response.Targets.Data))
		}

		if !response.Targets.HasMore {
			t.Error("has_more = false, want true")
		}

		if response.Targets.Limit != 2 ||
			response.Targets.Offset != 0 {
			t.Errorf("unexpected pagination: %+v", response.Targets)
		}
	})

	t.Run("agent not found", func(t *testing.T) {
		rec, _ := request("/api/v1/agents/nonexistent-agent")

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d",
				rec.Code, http.StatusNotFound)
		}
	})

	t.Run("invalid pagination", func(t *testing.T) {
		testCases := []string{
			"?limit=0",
			"?limit=101",
			"?limit=invalid",
			"?offset=-1",
			"?offset=invalid",
		}

		for _, query := range testCases {
			rec, _ := request("/api/v1/agents/" + probeID + query)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("%s: status = %d, want %d",
					query, rec.Code, http.StatusBadRequest)
			}
		}
	})
}
