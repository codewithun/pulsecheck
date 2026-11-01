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
	"sync"
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

	configPath := flag.String(
		"config",
		"",
		"Path to JSON target configuration file",
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
		"Check interval for legacy target flags",
	)

	timeout := flag.Duration(
		"timeout",
		5*time.Second,
		"Request or connection timeout for legacy target flags",
	)

	flag.Parse()

	var selectedTargets []TargetConfig

	if strings.TrimSpace(*configPath) != "" {
		configs, err := LoadConfig(*configPath)
		if err != nil {
			log.Fatalf("Load configuration: %v", err)
		}

		selectedTargets = configs

	} else {
		if *interval <= 0 || *timeout <= 0 {
			log.Fatal("interval and timeout must be positive")
		}

		selectedProtocol := strings.ToLower(strings.TrimSpace(*protocol))

		if selectedProtocol != "http" && selectedProtocol != "tcp" {
			log.Fatal("protocol must be either http or tcp")
		}

		legacyTargets, err := parseTargets(*target, *targets)
		if err != nil {
			log.Fatal(err)
		}

		for i, currentTarget := range legacyTargets {
			selectedTargets = append(selectedTargets, TargetConfig{
				Name:     fmt.Sprintf("Target-%d", i+1),
				Target:   currentTarget,
				Protocol: selectedProtocol,
				Interval: *interval,
				Timeout:  *timeout,
			})
		}
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

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	apiClient := &http.Client{
		Timeout: 5 * time.Second,
	}

	var wg sync.WaitGroup

	log.Printf(
		"Probe Agent %q started; targets=%d",
		probeID,
		len(selectedTargets),
	)

	for _, config := range selectedTargets {
		wg.Add(1)

		go func(config TargetConfig) {
			defer wg.Done()

			runTarget(
				ctx,
				probeID,
				config,
				*apiURL,
				apiToken,
				apiClient,
			)
		}(config)
	}

	<-ctx.Done()

	log.Println("Probe Agent shutting down")

	wg.Wait()

	log.Println("All target workers stopped")
}

func runTarget(
	ctx context.Context,
	probeID string,
	config TargetConfig,
	apiURL string,
	apiToken string,
	apiClient *http.Client,
) {
	client := &http.Client{
		Timeout: config.Timeout,
		CheckRedirect: func(
			req *http.Request,
			via []*http.Request,
		) error {
			return http.ErrUseLastResponse
		},
	}

	log.Printf(
		"Target worker started: name=%s protocol=%s target=%s interval=%s timeout=%s",
		config.Name,
		strings.ToUpper(config.Protocol),
		config.Target,
		config.Interval,
		config.Timeout,
	)

	for {
		if ctx.Err() != nil {
			return
		}

		var result CheckResult

		switch config.Protocol {
		case "http":
			result = checkHTTP(client, probeID, config.Target)

		case "tcp":
			result = checkTCP(probeID, config.Target, config.Timeout)
		}

		log.Printf(
			"Check result: name=%s protocol=%s target=%s status=%s latency=%.2fms",
			config.Name,
			result.Protocol,
			result.Target,
			result.Status,
			result.LatencyMS,
		)

		if result.Error != "" {
			log.Printf(
				"Check error for %s: %s",
				config.Name,
				result.Error,
			)
		}

		if err := sendResultWithRetry(
			apiClient,
			apiURL,
			apiToken,
			result,
			3,
			2*time.Second,
		); err != nil {
			log.Printf(
				"Send result failed for %s: %v",
				config.Name,
				err,
			)
		}

		timer := time.NewTimer(config.Interval)

		select {
		case <-ctx.Done():
			timer.Stop()
			return

		case <-timer.C:
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
