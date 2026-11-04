package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestTargetHistoryIntegration(t *testing.T) {
	db := openTestDB(t)

	probeID := fmt.Sprintf("history-test-%d", time.Now().UnixNano())
	otherProbeID := probeID + "-other"
	target := "https://history-test.invalid"

	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM check_results WHERE probe_id IN ($1, $2)`,
			probeID, otherProbeID)
	})

	now := time.Now().UTC()

	fixtures := []struct {
		status     string
		statusCode any
		latency    float64
		errorText  any
		protocol   string
		checkedAt  time.Time
	}{
		{"UP", 200, 12.5, nil, "HTTP", now.Add(-time.Minute)},
		{"INVALID", nil, 0, "temporary error", "HTTP", now.Add(-2 * time.Minute)},
		{"DOWN", nil, 25.5, "connection refused", "TCP", now.Add(-3 * time.Minute)},
		{"UP", 200, 10, nil, "HTTP", now.Add(-2 * time.Hour)},
	}

	for _, fixture := range fixtures {
		_, err := db.Exec(`
			INSERT INTO check_results
				(probe_id, target, status, status_code, latency_ms,
				 error, protocol, checked_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`,
			probeID,
			target,
			fixture.status,
			fixture.statusCode,
			fixture.latency,
			fixture.errorText,
			fixture.protocol,
			fixture.checkedAt,
		)
		if err != nil {
			t.Fatalf("insert history fixture: %v", err)
		}
	}

	// Same target, different probe. This record must not appear.
	_, err := db.Exec(`
		INSERT INTO check_results
			(probe_id, target, status, latency_ms, protocol, checked_at)
		VALUES ($1, $2, 'UP', 5, 'HTTP', $3)
	`, otherProbeID, target, now)
	if err != nil {
		t.Fatalf("insert other probe fixture: %v", err)
	}

	handler := targetHistoryHandler(db)

	request := func(query string) (*httptest.ResponseRecorder, TargetHistoryResponse) {
		t.Helper()

		req := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/targets/history?"+query,
			nil,
		)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		var response TargetHistoryResponse
		if rec.Code == http.StatusOK {
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
		}

		return rec, response
	}

	baseQuery := "probe_id=" + url.QueryEscape(probeID) +
		"&target=" + url.QueryEscape(target)

	t.Run("returns only requested probe and target", func(t *testing.T) {
		rec, response := request(baseQuery)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s",
				rec.Code, http.StatusOK, rec.Body.String())
		}

		if len(response.Data) != 4 {
			t.Fatalf("history records = %d, want 4", len(response.Data))
		}

		for _, item := range response.Data {
			if item.ProbeID != probeID || item.Target != target {
				t.Errorf("unexpected history item: %+v", item)
			}
		}

		if response.Data[0].Status != "UP" {
			t.Errorf("latest status = %q, want UP", response.Data[0].Status)
		}

		if response.Data[0].StatusCode == nil ||
			*response.Data[0].StatusCode != 200 {
			t.Errorf("latest status_code = %v, want 200", response.Data[0].StatusCode)
		}

		if response.Data[1].Status != "INVALID" {
			t.Errorf("second status = %q, want INVALID", response.Data[1].Status)
		}

		if response.Data[1].Error != "temporary error" {
			t.Errorf("second error = %q, want temporary error",
				response.Data[1].Error)
		}

		if response.Data[1].StatusCode != nil {
			t.Errorf("INVALID status_code = %v, want nil", response.Data[1].StatusCode)
		}

		if !response.Data[0].CheckedAt.After(response.Data[1].CheckedAt) {
			t.Error("history is not ordered newest first")
		}

		if response.Data[0].ReceivedAt.IsZero() {
			t.Error("received_at should not be zero")
		}

		if response.HasMore {
			t.Error("has_more = true, want false")
		}
	})

	t.Run("period filter", func(t *testing.T) {
		rec, response := request(baseQuery + "&period=1h")

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s",
				rec.Code, http.StatusOK, rec.Body.String())
		}

		if len(response.Data) != 3 {
			t.Errorf("history records = %d, want 3", len(response.Data))
		}

		for _, item := range response.Data {
			if item.CheckedAt.Before(now.Add(-time.Hour)) {
				t.Errorf("old record appeared in 1h period: %+v", item)
			}
		}
	})

	t.Run("pagination", func(t *testing.T) {
		rec, response := request(baseQuery + "&limit=2")

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s",
				rec.Code, http.StatusOK, rec.Body.String())
		}

		if len(response.Data) != 2 {
			t.Errorf("history records = %d, want 2", len(response.Data))
		}

		if !response.HasMore {
			t.Error("has_more = false, want true")
		}

		if response.Limit != 2 || response.Offset != 0 {
			t.Errorf("unexpected pagination: %+v", response)
		}

		rec, nextPage := request(baseQuery + "&limit=2&offset=2")

		if rec.Code != http.StatusOK {
			t.Fatalf("second page status = %d, want %d; body=%s",
				rec.Code, http.StatusOK, rec.Body.String())
		}

		if len(nextPage.Data) != 2 {
			t.Errorf("second page records = %d, want 2", len(nextPage.Data))
		}

		if nextPage.HasMore {
			t.Error("second page has_more = true, want false")
		}
	})
}

func TestTargetHistoryRejectsInvalidParameters(t *testing.T) {
	db := openTestDB(t)
	handler := targetHistoryHandler(db)

	testCases := []struct {
		name string
		path string
	}{
		{"missing probe_id", "/api/v1/targets/history?target=https://example.com"},
		{"missing target", "/api/v1/targets/history?probe_id=probe-1"},
		{"empty parameters", "/api/v1/targets/history?probe_id=&target="},
		{"invalid limit", "/api/v1/targets/history?probe_id=p&target=t&limit=101"},
		{"zero limit", "/api/v1/targets/history?probe_id=p&target=t&limit=0"},
		{"negative offset", "/api/v1/targets/history?probe_id=p&target=t&offset=-1"},
		{"invalid period", "/api/v1/targets/history?probe_id=p&target=t&period=invalid"},
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
