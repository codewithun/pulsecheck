package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
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
