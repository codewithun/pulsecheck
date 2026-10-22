package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestJSONOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	cmd := exec.Command(
		"go",
		"run",
		".",
		"-json",
		server.URL,
	)

	cmd.Dir = "."

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %v\nOutput: %s", err, output)
	}

	var result struct {
		Results []CheckResult `json:"results"`
		Summary struct {
			Up   int `json:"up"`
			Down int `json:"down"`
		} `json:"summary"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("invalid JSON output: %v\nOutput: %s", err, output)
	}

	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}

	if result.Results[0].Status != "UP" {
		t.Errorf("expected UP, got %s", result.Results[0].Status)
	}

	if result.Summary.Up != 1 || result.Summary.Down != 0 {
		t.Errorf("unexpected summary: %+v", result.Summary)
	}
}

func TestJSONOutputContainsNoTextSummary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	cmd := exec.Command(
		"go",
		"run",
		".",
		"-json",
		server.URL,
	)

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	if strings.Contains(string(output), "Summary:") {
		t.Error("JSON output should not contain the plain-text summary")
	}

	_ = os.Stdout
}

func TestJSONOutputHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		},
	))
	defer server.Close()

	cmd := exec.Command("go", "run", ".", "-json", server.URL)
	output, err := cmd.Output()

	if err == nil {
		t.Fatal("expected non-zero exit code for failed check")
	}

	var result struct {
		Results []CheckResult `json:"results"`
		Summary struct {
			Up   int `json:"up"`
			Down int `json:"down"`
		} `json:"summary"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("invalid JSON output: %v\nOutput: %s", err, output)
	}

	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}

	if result.Results[0].Status != "DOWN" {
		t.Errorf("expected DOWN, got %s", result.Results[0].Status)
	}

	if result.Summary.Down != 1 {
		t.Errorf("expected 1 DOWN, got %d", result.Summary.Down)
	}
}

func TestJSONOutputTCPFailure(t *testing.T) {
	cmd := exec.Command(
		"go", "run", ".", "-tcp", "-json",
		"invalid-address",
	)

	output, err := cmd.Output()
	if err == nil {
		t.Fatal("expected non-zero exit code for invalid TCP target")
	}

	var result struct {
		Results []CheckResult `json:"results"`
		Summary struct {
			Up   int `json:"up"`
			Down int `json:"down"`
		} `json:"summary"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("invalid JSON output: %v\nOutput: %s", err, output)
	}

	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}

	if result.Results[0].Status != "INVALID" {
		t.Errorf("expected INVALID, got %s", result.Results[0].Status)
	}

	if result.Summary.Down != 1 {
		t.Errorf("expected 1 DOWN, got %d", result.Summary.Down)
	}
}
