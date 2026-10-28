package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthenticateRejectsMissingToken(t *testing.T) {
	handler := authenticate("test-secret", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/results", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d",
			http.StatusUnauthorized, rec.Code)
	}
}

func TestAuthenticateAcceptsValidToken(t *testing.T) {
	handler := authenticate("test-secret", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/results", nil)
	req.Header.Set("Authorization", "Bearer test-secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d",
			http.StatusOK, rec.Code)
	}
}

func TestAuthenticateRejectsWrongToken(t *testing.T) {
	handler := authenticate("test-secret", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/results", nil)
	req.Header.Set("Authorization", "Bearer wrong-secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d",
			http.StatusUnauthorized, rec.Code)
	}
}

func TestSummaryEndpointRejectsMissingToken(t *testing.T) {
	handler := authenticate("test-secret", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	))

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/summary",
		nil,
	)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d",
			http.StatusUnauthorized, rec.Code)
	}
}

func TestSummaryEndpointAcceptsValidToken(t *testing.T) {
	handler := authenticate("test-secret", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	))

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/summary",
		nil,
	)
	req.Header.Set("Authorization", "Bearer test-secret")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d",
			http.StatusOK, rec.Code)
	}
}
