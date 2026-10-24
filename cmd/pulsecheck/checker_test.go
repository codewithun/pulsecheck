package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantStatus string
	}{
		{
			name:       "HTTP 200 is UP",
			statusCode: http.StatusOK,
			wantStatus: "UP",
		},
		{
			name:       "HTTP 404 is DOWN",
			statusCode: http.StatusNotFound,
			wantStatus: "DOWN",
		},
		{
			name:       "HTTP 500 is DOWN",
			statusCode: http.StatusInternalServerError,
			wantStatus: "DOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.statusCode)
				},
			))
			defer server.Close()

			result := Check(server.URL)

			if result.Status != tt.wantStatus {
				t.Errorf("got status %q, want %q",
					result.Status, tt.wantStatus)
			}

			if result.StatusCode != tt.statusCode {
				t.Errorf("got HTTP code %d, want %d",
					result.StatusCode, tt.statusCode)
			}
		})
	}
}

func TestCheckInvalidURL(t *testing.T) {
	result := Check("google.com")

	if result.Status != "INVALID" {
		t.Errorf("got status %q, want INVALID", result.Status)
	}
}

func TestCheckInvalidScheme(t *testing.T) {
	result := Check("ftp://example.com")

	if result.Status != "INVALID" {
		t.Errorf("got status %q, want INVALID", result.Status)
	}
}

func TestCheckTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(300 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	result := CheckWithTimeout(server.URL, 50*time.Millisecond)

	if result.Status != "DOWN" {
		t.Errorf("got status %q, want DOWN", result.Status)
	}

	if result.Error == "" {
		t.Error("expected timeout error, got no error")
	}
}

func TestCheckRedirect(t *testing.T) {
	finalServer := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	))
	defer finalServer.Close()

	redirectServer := httptest.NewServer(http.RedirectHandler(
		finalServer.URL,
		http.StatusFound,
	))
	defer redirectServer.Close()

	result := Check(redirectServer.URL)

	if result.Status != "UP" {
		t.Errorf("got status %q, want UP", result.Status)
	}

	if result.StatusCode != http.StatusOK {
		t.Errorf("got HTTP code %d, want 200", result.StatusCode)
	}
}

func TestCheckTCPAvailablePort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test listener: %v", err)
	}
	defer listener.Close()

	result := CheckTCP(listener.Addr().String(), time.Second)

	if result.Status != "UP" {
		t.Errorf("expected UP, got %s: %s", result.Status, result.Error)
	}

	if result.Protocol != "TCP" {
		t.Errorf("expected TCP protocol, got %q", result.Protocol)
	}
}

func TestCheckTCPInvalidAddress(t *testing.T) {
	result := CheckTCP("invalid-address", time.Second)

	if result.Status != "INVALID" {
		t.Errorf("expected INVALID, got %s", result.Status)
	}
}

func TestCheckTargetsHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	report := CheckTargets(
		[]string{server.URL},
		time.Second,
		false,
		10,
		0,
	)

	if len(report.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(report.Results))
	}

	if report.Results[0].Status != "UP" {
		t.Errorf("expected UP, got %s", report.Results[0].Status)
	}

	if report.Summary.Up != 1 || report.Summary.Down != 0 {
		t.Errorf("unexpected summary: %+v", report.Summary)
	}

	if report.Results[0].LatencyMS < 0 {
		t.Errorf("latency_ms cannot be negative")
	}
}

func TestCheckTargetsTCP(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test listener: %v", err)
	}
	defer listener.Close()

	report := CheckTargets(
		[]string{listener.Addr().String()},
		time.Second,
		true,
		10,
		0,
	)

	if len(report.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(report.Results))
	}

	if report.Results[0].Status != "UP" {
		t.Errorf("expected UP, got %s", report.Results[0].Status)
	}

	if report.Results[0].Protocol != "TCP" {
		t.Errorf("expected TCP, got %s", report.Results[0].Protocol)
	}

	if report.Summary.Up != 1 || report.Summary.Down != 0 {
		t.Errorf("unexpected summary: %+v", report.Summary)
	}
}

func TestCheckTargetsInvalidTarget(t *testing.T) {
	report := CheckTargets(
		[]string{"invalid-address"},
		time.Second,
		true,
		10,
		0,
	)

	if len(report.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(report.Results))
	}

	if report.Results[0].Status != "INVALID" {
		t.Errorf("expected INVALID, got %s", report.Results[0].Status)
	}

	if report.Summary.Up != 0 || report.Summary.Down != 1 {
		t.Errorf("unexpected summary: %+v", report.Summary)
	}
}

func TestCheckTargetsWithConcurrency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	targets := []string{
		server.URL,
		server.URL,
		server.URL,
	}

	report := CheckTargets(targets, time.Second, false, 2, 0)

	if len(report.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(report.Results))
	}

	if report.Summary.Up != 3 {
		t.Errorf("expected 3 UP, got %d", report.Summary.Up)
	}

	if report.Summary.Down != 0 {
		t.Errorf("expected 0 DOWN, got %d", report.Summary.Down)
	}
}

func TestCheckTargetsRetriesFailedCheck(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if attempts.Add(1) < 3 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	report := CheckTargets(
		[]string{server.URL},
		time.Second,
		false,
		1,
		2,
	)

	if report.Results[0].Status != "UP" {
		t.Errorf("expected UP after retries, got %s", report.Results[0].Status)
	}

	if got := attempts.Load(); got != 3 {
		t.Errorf("expected 3 attempts, got %d", got)
	}
}
