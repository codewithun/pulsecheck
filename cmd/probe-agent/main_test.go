package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCheckHTTP(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		expectedStatus string
	}{
		{
			name:           "HTTP 200 is UP",
			statusCode:     http.StatusOK,
			expectedStatus: "UP",
		},
		{
			name:           "HTTP 500 is DOWN",
			statusCode:     http.StatusInternalServerError,
			expectedStatus: "DOWN",
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

			client := &http.Client{
				Timeout: 2 * time.Second,
			}

			result := checkHTTP(client, "test-probe", server.URL)

			if result.Status != tt.expectedStatus {
				t.Errorf(
					"expected status %q, got %q",
					tt.expectedStatus,
					result.Status,
				)
			}

			if result.StatusCode == nil ||
				*result.StatusCode != tt.statusCode {
				t.Errorf(
					"expected HTTP status code %d, got %v",
					tt.statusCode,
					result.StatusCode,
				)
			}

			if result.LatencyMS < 0 {
				t.Errorf("latency must not be negative")
			}

			if result.Protocol != "HTTP" {
				t.Errorf(
					"expected protocol HTTP, got %q",
					result.Protocol,
				)
			}
		})
	}
}

func TestCheckHTTPTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(200 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	client := &http.Client{
		Timeout: 20 * time.Millisecond,
	}

	result := checkHTTP(client, "test-probe", server.URL)

	if result.Status != "DOWN" {
		t.Errorf("expected DOWN after timeout, got %q", result.Status)
	}

	if result.Error == "" {
		t.Error("expected an error message for timeout")
	}
}

func TestCheckHTTPInvalidTarget(t *testing.T) {
	client := &http.Client{
		Timeout: time.Second,
	}

	result := checkHTTP(client, "test-probe", "://invalid-url")

	if result.Status != "INVALID" {
		t.Errorf("expected INVALID, got %q", result.Status)
	}

	if result.Error == "" {
		t.Error("expected an error message for invalid target")
	}
}

func TestSendResultSuccess(t *testing.T) {
	token := "test-token"

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}

			if r.URL.Path != "/api/v1/results" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}

			if got := r.Header.Get("Authorization"); got != "Bearer "+token {
				t.Errorf("unexpected Authorization header: %q", got)
			}

			if got := r.Header.Get("Content-Type"); got != "application/json" {
				t.Errorf("unexpected Content-Type: %q", got)
			}

			if r.ContentLength == 0 {
				t.Error("expected a non-empty JSON request body")
			}

			w.WriteHeader(http.StatusCreated)
		},
	))
	defer server.Close()

	client := &http.Client{Timeout: time.Second}

	result := CheckResult{
		ProbeID:   "test-probe",
		Target:    "https://example.com",
		Status:    "UP",
		LatencyMS: 12.5,
		Protocol:  "HTTP",
		CheckedAt: time.Now().UTC(),
	}

	err := sendResult(client, server.URL, token, result)
	if err != nil {
		t.Fatalf("expected successful send, got error: %v", err)
	}
}

func TestSendResultRejectsUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		},
	))
	defer server.Close()

	client := &http.Client{Timeout: time.Second}

	result := CheckResult{
		ProbeID:   "test-probe",
		Target:    "https://example.com",
		Status:    "UP",
		Protocol:  "HTTP",
		CheckedAt: time.Now().UTC(),
	}

	err := sendResult(client, server.URL, "wrong-token", result)
	if err == nil {
		t.Fatal("expected error when API returns 401 Unauthorized")
	}
}

func TestCheckTCPSuccess(t *testing.T) {
	server, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("start TCP listener: %v", err)
	}
	defer server.Close()

	go func() {
		conn, err := server.Accept()
		if err == nil {
			_ = conn.Close()
		}
	}()

	result := checkTCP(
		"test-probe",
		server.Addr().String(),
		time.Second,
	)

	if result.Status != "UP" {
		t.Fatalf(
			"expected UP, got %q; error=%s",
			result.Status,
			result.Error,
		)
	}

	if result.Protocol != "TCP" {
		t.Errorf(
			"expected protocol TCP, got %q",
			result.Protocol,
		)
	}

	if result.LatencyMS < 0 {
		t.Error("latency must not be negative")
	}
}

func TestCheckTCPInvalidTarget(t *testing.T) {
	result := checkTCP(
		"test-probe",
		"127.0.0.1:0",
		200*time.Millisecond,
	)

	if result.Status != "DOWN" {
		t.Errorf("expected DOWN, got %q", result.Status)
	}

	if result.Error == "" {
		t.Error("expected an error message for failed TCP connection")
	}
}

func TestSendResultWithRetryEventuallySucceeds(t *testing.T) {
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			attempts++

			if attempts < 3 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}

			w.WriteHeader(http.StatusCreated)
		},
	))
	defer server.Close()

	client := &http.Client{Timeout: time.Second}

	result := CheckResult{
		ProbeID:   "test-probe",
		Target:    "https://example.com",
		Status:    "UP",
		Protocol:  "HTTP",
		CheckedAt: time.Now().UTC(),
	}

	err := sendResultWithRetry(
		client,
		server.URL,
		"test-token",
		result,
		3,
		time.Millisecond,
	)

	if err != nil {
		t.Fatalf("expected retry to succeed, got: %v", err)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestSendResultWithRetryExhausted(t *testing.T) {
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			attempts++
			w.WriteHeader(http.StatusServiceUnavailable)
		},
	))
	defer server.Close()

	client := &http.Client{Timeout: time.Second}

	result := CheckResult{
		ProbeID:   "test-probe",
		Target:    "https://example.com",
		Status:    "UP",
		Protocol:  "HTTP",
		CheckedAt: time.Now().UTC(),
	}

	err := sendResultWithRetry(
		client,
		server.URL,
		"test-token",
		result,
		3,
		time.Millisecond,
	)

	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}
