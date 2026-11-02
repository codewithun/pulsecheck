package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAgentHeartbeatIntegration(t *testing.T) {
	db := openTestDB(t)

	probeID := fmt.Sprintf("heartbeat-test-%d", time.Now().UnixNano())
	hostname := "integration-test-host"

	t.Cleanup(func() {
		_, _ = db.Exec(
			`DELETE FROM agents WHERE probe_id = $1`,
			probeID,
		)
	})

	handler := agentHeartbeatHandler(db)

	sendHeartbeat := func(probeID, hostname string) *httptest.ResponseRecorder {
		t.Helper()

		payload, err := json.Marshal(AgentHeartbeatRequest{
			ProbeID:  probeID,
			Hostname: hostname,
		})
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/agents/heartbeat",
			bytes.NewReader(payload),
		)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		return rec
	}

	t.Run("register agent", func(t *testing.T) {
		rec := sendHeartbeat(probeID, hostname)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s",
				rec.Code, http.StatusOK, rec.Body.String())
		}

		var response AgentHeartbeatResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		if response.ProbeID != probeID {
			t.Fatalf("probe_id = %q, want %q", response.ProbeID, probeID)
		}

		if response.Status != "received" {
			t.Fatalf("status = %q, want received", response.Status)
		}
	})

	t.Run("update last seen", func(t *testing.T) {
		var previous time.Time

		if err := db.QueryRow(`
			SELECT last_seen FROM agents WHERE probe_id = $1
		`, probeID).Scan(&previous); err != nil {
			t.Fatalf("read previous last_seen: %v", err)
		}

		rec := sendHeartbeat(probeID, "updated-host")

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		var current time.Time
		var currentHostname string

		if err := db.QueryRow(`
			SELECT last_seen, hostname
			FROM agents
			WHERE probe_id = $1
		`, probeID).Scan(&current, &currentHostname); err != nil {
			t.Fatalf("read updated agent: %v", err)
		}

		if current.Before(previous) {
			t.Fatalf("last_seen was not updated: previous=%v current=%v",
				previous, current)
		}

		if currentHostname != "updated-host" {
			t.Fatalf("hostname = %q, want updated-host", currentHostname)
		}
	})

	t.Run("online status", func(t *testing.T) {
		var status string

		err := db.QueryRow(`
			SELECT CASE
				WHEN last_seen >= NOW() - INTERVAL '90 seconds'
				THEN 'ONLINE'
				ELSE 'OFFLINE'
			END
			FROM agents
			WHERE probe_id = $1
		`, probeID).Scan(&status)

		if err != nil {
			t.Fatalf("read agent status: %v", err)
		}

		if status != "ONLINE" {
			t.Fatalf("status = %q, want ONLINE", status)
		}
	})

	t.Run("offline status", func(t *testing.T) {
		_, err := db.Exec(`
			UPDATE agents
			SET last_seen = NOW() - INTERVAL '120 seconds'
			WHERE probe_id = $1
		`, probeID)

		if err != nil {
			t.Fatalf("set stale heartbeat: %v", err)
		}

		var status string

		err = db.QueryRow(`
			SELECT CASE
				WHEN last_seen >= NOW() - INTERVAL '90 seconds'
				THEN 'ONLINE'
				ELSE 'OFFLINE'
			END
			FROM agents
			WHERE probe_id = $1
		`, probeID).Scan(&status)

		if err != nil {
			t.Fatalf("read agent status: %v", err)
		}

		if status != "OFFLINE" {
			t.Fatalf("status = %q, want OFFLINE", status)
		}
	})
}

func TestAgentHeartbeatRejectsInvalidPayload(t *testing.T) {
	db := openTestDB(t)

	handler := agentHeartbeatHandler(db)

	testCases := []struct {
		name string
		body string
	}{
		{
			name: "invalid JSON",
			body: `{"probe_id":`,
		},
		{
			name: "missing probe ID",
			body: `{"hostname":"test-host"}`,
		},
		{
			name: "missing hostname",
			body: `{"probe_id":"test-probe"}`,
		},
		{
			name: "unknown field",
			body: `{"probe_id":"test-probe","hostname":"test-host","extra":"value"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/agents/heartbeat",
				bytes.NewBufferString(tc.body),
			)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body=%s",
					rec.Code, http.StatusBadRequest, rec.Body.String())
			}
		})
	}
}
