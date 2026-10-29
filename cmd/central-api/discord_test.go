package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDiscordSenderSendsDownAlert(t *testing.T) {
	var received discordMessage

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected Content-Type: %s", r.Header.Get("Content-Type"))
		}

		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("decode request body: %v", err)
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	sender := newDiscordSender(server.URL)
	event := alertEvent{
		ID:             1,
		ProbeID:        "probe-test",
		Target:         "https://example.com",
		PreviousStatus: "UP",
		CurrentStatus:  "DOWN",
		Error:          "connection timeout",
	}

	if err := sender.send(context.Background(), event); err != nil {
		t.Fatalf("send returned error: %v", err)
	}

	for _, expected := range []string{
		"Target DOWN",
		"probe-test",
		"https://example.com",
		"UP → DOWN",
		"connection timeout",
	} {
		if !strings.Contains(received.Content, expected) {
			t.Errorf("message %q does not contain %q", received.Content, expected)
		}
	}
}

func TestDiscordSenderSendsRecoveryAlert(t *testing.T) {
	var received discordMessage

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("decode request body: %v", err)
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	sender := newDiscordSender(server.URL)
	event := alertEvent{
		ProbeID:        "probe-test",
		Target:         "https://example.com",
		PreviousStatus: "DOWN",
		CurrentStatus:  "UP",
	}

	if err := sender.send(context.Background(), event); err != nil {
		t.Fatalf("send returned error: %v", err)
	}

	for _, expected := range []string{
		"RECOVERED",
		"DOWN → UP",
		"probe-test",
	} {
		if !strings.Contains(received.Content, expected) {
			t.Errorf("message %q does not contain %q", received.Content, expected)
		}
	}
}

func TestDiscordSenderRejectsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "simulated failure", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	sender := newDiscordSender(server.URL)
	err := sender.send(context.Background(), alertEvent{
		ProbeID:       "probe-test",
		Target:        "https://example.com",
		CurrentStatus: "DOWN",
	})

	if err == nil {
		t.Fatal("expected error for HTTP 503")
	}
	if !strings.Contains(err.Error(), "503") {
		t.Errorf("error = %q, want HTTP status 503", err)
	}
}

func TestDiscordSenderRejectsEmptyWebhook(t *testing.T) {
	sender := newDiscordSender("")

	err := sender.send(context.Background(), alertEvent{
		CurrentStatus: "DOWN",
	})

	if err == nil {
		t.Fatal("expected error for empty webhook URL")
	}
}

func TestRetryDelayIncreasesAndCaps(t *testing.T) {
	tests := []struct {
		attempts int
		want     time.Duration
	}{
		{attempts: 0, want: 1 * time.Second},
		{attempts: 1, want: 2 * time.Second},
		{attempts: 2, want: 4 * time.Second},
		{attempts: 4, want: 16 * time.Second},
		{attempts: 10, want: 900 * time.Second},
		{attempts: 100, want: 900 * time.Second},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempts_%d", tt.attempts), func(t *testing.T) {
			if got := retryDelay(tt.attempts); got != tt.want {
				t.Errorf("retryDelay(%d) = %v, want %v", tt.attempts, got, tt.want)
			}
		})
	}
}

func TestRetryDelayIsExponentialAndCapped(t *testing.T) {
	tests := []struct {
		attempts int
		want     time.Duration
	}{
		{attempts: 0, want: time.Second},
		{attempts: 1, want: 2 * time.Second},
		{attempts: 2, want: 4 * time.Second},
		{attempts: 3, want: 8 * time.Second},
		{attempts: 9, want: 512 * time.Second},
		{attempts: 100, want: 900 * time.Second},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempts_%d", tt.attempts), func(t *testing.T) {
			if got := retryDelay(tt.attempts); got != tt.want {
				t.Errorf("retryDelay(%d) = %v, want %v", tt.attempts, got, tt.want)
			}
		})
	}
}
