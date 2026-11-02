package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSendHeartbeatSuccess(t *testing.T) {
	token := "test-token"
	requestReceived := false

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			requestReceived = true

			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}

			if r.URL.Path != "/api/v1/agents/heartbeat" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}

			if got := r.Header.Get("Authorization"); got != "Bearer "+token {
				t.Errorf("unexpected Authorization header: %q", got)
			}

			if got := r.Header.Get("Content-Type"); got != "application/json" {
				t.Errorf("unexpected Content-Type: %q", got)
			}

			var payload HeartbeatRequest
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Errorf("decode payload: %v", err)
				return
			}

			if payload.ProbeID != "test-probe" {
				t.Errorf("unexpected probe_id: %q", payload.ProbeID)
			}

			if payload.Hostname != "test-host" {
				t.Errorf("unexpected hostname: %q", payload.Hostname)
			}

			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	client := &http.Client{Timeout: time.Second}

	err := sendHeartbeat(
		context.Background(),
		client,
		server.URL,
		token,
		"test-probe",
		"test-host",
	)

	if err != nil {
		t.Fatalf("expected successful heartbeat, got: %v", err)
	}

	if !requestReceived {
		t.Fatal("expected heartbeat request to be received")
	}
}

func TestSendHeartbeatRejectsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		},
	))
	defer server.Close()

	client := &http.Client{Timeout: time.Second}

	err := sendHeartbeat(
		context.Background(),
		client,
		server.URL,
		"test-token",
		"test-probe",
		"test-host",
	)

	if err == nil {
		t.Fatal("expected error when API returns HTTP 500")
	}
}

func TestRunHeartbeatStopsOnContextCancellation(t *testing.T) {
	heartbeatReceived := make(chan struct{}, 1)

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			heartbeatReceived <- struct{}{}
			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		defer close(done)

		runHeartbeat(
			ctx,
			&http.Client{Timeout: time.Second},
			server.URL,
			"test-token",
			"test-probe",
			"test-host",
			time.Hour,
		)
	}()

	select {
	case <-heartbeatReceived:
	case <-time.After(2 * time.Second):
		cancel()
		t.Fatal("heartbeat was not sent immediately")
	}

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("heartbeat worker did not stop after cancellation")
	}
}
