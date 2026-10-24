package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type CheckResult struct {
	ProbeID    string    `json:"probe_id"`
	Target     string    `json:"target"`
	Status     string    `json:"status"`
	StatusCode *int      `json:"status_code,omitempty"`
	LatencyMS  float64   `json:"latency_ms"`
	Error      string    `json:"error,omitempty"`
	Protocol   string    `json:"protocol"`
	CheckedAt  time.Time `json:"checked_at"`
}

func main() {
	apiURL := flag.String("api", "http://localhost:8081", "Central API base URL")
	target := flag.String("target", "https://example.com", "HTTP/HTTPS target to check")
	interval := flag.Duration("interval", 30*time.Second, "Check interval")
	timeout := flag.Duration("timeout", 5*time.Second, "Request timeout")
	flag.Parse()

	if *interval <= 0 || *timeout <= 0 {
		log.Fatal("interval and timeout must be positive")
	}

	apiToken := os.Getenv("API_TOKEN")
	if apiToken == "" {
		log.Fatal("API_TOKEN is required")
	}

	probeID, err := os.Hostname()
	if err != nil || strings.TrimSpace(probeID) == "" {
		probeID = "probe-local"
	}

	client := &http.Client{
		Timeout: *timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	apiClient := &http.Client{Timeout: 5 * time.Second}

	log.Printf("Probe Agent %q started; target=%s interval=%s",
		probeID, *target, *interval)

	for {
		result := checkHTTP(client, probeID, *target)
		log.Printf("Check result: target=%s status=%s latency=%.2fms",
			result.Target, result.Status, result.LatencyMS)

		if err := sendResult(apiClient, *apiURL, apiToken, result); err != nil {
			log.Printf("Send result failed: %v", err)
		}

		time.Sleep(*interval)
	}
}

func checkHTTP(client *http.Client, probeID, target string) CheckResult {
	result := CheckResult{
		ProbeID:   probeID,
		Target:    target,
		Status:    "DOWN",
		Protocol:  "HTTP",
		CheckedAt: time.Now().UTC(),
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), client.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		result.Status = "INVALID"
		result.Error = err.Error()
		result.LatencyMS = float64(time.Since(start).Microseconds()) / 1000
		return result
	}

	resp, err := client.Do(req)
	result.LatencyMS = float64(time.Since(start).Microseconds()) / 1000

	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()

	statusCode := resp.StatusCode
	result.StatusCode = &statusCode

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		result.Status = "UP"
	} else {
		result.Error = fmt.Sprintf("HTTP status %d", resp.StatusCode)
	}

	return result
}

func sendResult(client *http.Client, apiURL, token string, result CheckResult) error {
	payload, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("encode result: %w", err)
	}

	endpoint := strings.TrimRight(apiURL, "/") + "/api/v1/results"
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request central API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return errors.New("central API returned status " + resp.Status)
	}

	return nil
}
