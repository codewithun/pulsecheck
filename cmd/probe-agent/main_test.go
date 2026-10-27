package main

import (
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
