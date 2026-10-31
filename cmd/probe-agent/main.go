package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
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
	apiURL := flag.String(
		"api",
		"http://localhost:8081",
		"Central API base URL",
	)

	protocol := flag.String(
		"protocol",
		"http",
		"Check protocol: http or tcp",
	)

	target := flag.String(
		"target",
		"",
		"Single monitoring target (backward compatible)",
	)

	targets := flag.String(
		"targets",
		"",
		"Comma-separated list of monitoring targets",
	)

	interval := flag.Duration(
		"interval",
		30*time.Second,
		"Check interval",
	)

	timeout := flag.Duration(
		"timeout",
		5*time.Second,
		"Request or connection timeout",
	)

	flag.Parse()

	if *interval <= 0 || *timeout <= 0 {
		log.Fatal("interval and timeout must be positive")
	}

	selectedProtocol := strings.ToLower(strings.TrimSpace(*protocol))
	if selectedProtocol != "http" && selectedProtocol != "tcp" {
		log.Fatal("protocol must be either http or tcp")
	}

	selectedTargets, err := parseTargets(*target, *targets)
	if err != nil {
		log.Fatal(err)
	}

	if strings.TrimSpace(*apiURL) == "" {
		log.Fatal("API URL must not be empty")
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
		CheckRedirect: func(
			req *http.Request,
			via []*http.Request,
		) error {
			return http.ErrUseLastResponse
		},
	}

	apiClient := &http.Client{
		Timeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	log.Printf(
		"Probe Agent %q started; protocol=%s targets=%d interval=%s timeout=%s",
		probeID,
		strings.ToUpper(selectedProtocol),
		len(selectedTargets),
		*interval,
		*timeout,
	)

	for {
		for _, currentTarget := range selectedTargets {
			var result CheckResult

			switch selectedProtocol {
			case "http":
				result = checkHTTP(client, probeID, currentTarget)

			case "tcp":
				result = checkTCP(probeID, currentTarget, *timeout)
			}

			log.Printf(
				"Check result: protocol=%s target=%s status=%s latency=%.2fms",
				result.Protocol,
				result.Target,
				result.Status,
				result.LatencyMS,
			)

			if result.Error != "" {
				log.Printf(
					"Check error for %s: %s",
					currentTarget,
					result.Error,
				)
			}

			if err := sendResultWithRetry(
				apiClient,
				*apiURL,
				apiToken,
				result,
				3,
				2*time.Second,
			); err != nil {
				log.Printf(
					"Send result failed for %s: %v",
					currentTarget,
					err,
				)
			}
		}

		select {
		case <-ctx.Done():
			log.Println("Probe Agent shutting down")
			return

		case <-ticker.C:
		}
	}
}

func parseTargets(singleTarget string, multiTargets string) ([]string, error) {
	if strings.TrimSpace(multiTargets) == "" {
		singleTarget = strings.TrimSpace(singleTarget)

		if singleTarget == "" {
			return nil, errors.New(
				"target must not be empty; use -target or -targets",
			)
		}

		return []string{singleTarget}, nil
	}

	parts := strings.Split(multiTargets, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		currentTarget := strings.TrimSpace(part)

		if currentTarget == "" {
			return nil, errors.New(
				"targets contains an empty target; check comma-separated values",
			)
		}

		result = append(result, currentTarget)
	}

	if len(result) == 0 {
		return nil, errors.New("at least one target is required")
	}

	return result, nil
}

func checkHTTP(
	client *http.Client,
	probeID string,
	target string,
) CheckResult {
	result := CheckResult{
		ProbeID:   probeID,
		Target:    target,
		Status:    "DOWN",
		Protocol:  "HTTP",
		CheckedAt: time.Now().UTC(),
	}

	start := time.Now()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		client.Timeout,
	)
	defer cancel()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		target,
		nil,
	)
	if err != nil {
		result.Status = "INVALID"
		result.Error = err.Error()
		result.LatencyMS = elapsedMilliseconds(start)
		return result
	}

	resp, err := client.Do(req)
	result.LatencyMS = elapsedMilliseconds(start)

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

func checkTCP(
	probeID string,
	target string,
	timeout time.Duration,
) CheckResult {
	result := CheckResult{
		ProbeID:   probeID,
		Target:    target,
		Status:    "DOWN",
		Protocol:  "TCP",
		CheckedAt: time.Now().UTC(),
	}

	if timeout <= 0 {
		result.Error = "timeout must be positive"
		return result
	}

	if strings.TrimSpace(target) == "" {
		result.Error = "target must not be empty"
		return result
	}

	start := time.Now()

	conn, err := net.DialTimeout("tcp", target, timeout)
	result.LatencyMS = elapsedMilliseconds(start)

	if err != nil {
		result.Error = err.Error()
		return result
	}

	if err := conn.Close(); err != nil {
		result.Error = fmt.Sprintf("close TCP connection: %v", err)
		return result
	}

	result.Status = "UP"
	return result
}

func elapsedMilliseconds(start time.Time) float64 {
	return float64(time.Since(start).Microseconds()) / 1000
}

func sendResult(
	client *http.Client,
	apiURL string,
	token string,
	result CheckResult,
) error {
	payload, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("encode result: %w", err)
	}

	endpoint := strings.TrimRight(apiURL, "/") + "/api/v1/results"

	req, err := http.NewRequest(
		http.MethodPost,
		endpoint,
		bytes.NewReader(payload),
	)
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
		return errors.New(
			"central API returned status " + resp.Status,
		)
	}

	return nil
}

func sendResultWithRetry(
	client *http.Client,
	apiURL string,
	token string,
	result CheckResult,
	maxAttempts int,
	retryDelay time.Duration,
) error {
	if maxAttempts < 1 {
		return errors.New("maxAttempts must be at least 1")
	}

	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		lastErr = sendResult(client, apiURL, token, result)
		if lastErr == nil {
			return nil
		}

		if attempt == maxAttempts {
			break
		}

		time.Sleep(retryDelay)
	}

	return fmt.Errorf(
		"failed to send result after %d attempts: %w",
		maxAttempts,
		lastErr,
	)
}
