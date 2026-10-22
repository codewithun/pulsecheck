package main

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type CheckResult struct {
	URL        string        `json:"url"`
	Status     string        `json:"status"`
	StatusCode int           `json:"status_code,omitempty"`
	Latency    time.Duration `json:"-"`
	LatencyMS  float64       `json:"latency_ms"`
	Error      string        `json:"error,omitempty"`
	Protocol   string        `json:"protocol"`
}

func Check(target string) CheckResult {
	return CheckWithTimeout(target, 5*time.Second)
}

func CheckWithTimeout(target string, timeout time.Duration) CheckResult {
	target = strings.TrimSpace(target)

	parsedURL, err := url.ParseRequestURI(target)
	if err != nil || parsedURL.Host == "" ||
		(parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return CheckResult{
			URL:    target,
			Status: "INVALID",
			Error:  "URL must start with http:// or https://",
		}
	}

	if timeout <= 0 {
		return CheckResult{
			URL:    target,
			Status: "INVALID",
			Error:  "Timeout must be greater than zero",
		}
	}

	client := &http.Client{Timeout: timeout}

	start := time.Now()
	resp, err := client.Get(parsedURL.String())
	latency := time.Since(start)

	if err != nil {
		return CheckResult{
			URL:     target,
			Status:  "DOWN",
			Latency: latency,
			Error:   err.Error(),
		}
	}
	defer resp.Body.Close()

	status := "UP"
	if resp.StatusCode >= 400 {
		status = "DOWN"
	}

	return CheckResult{
		URL:        target,
		Status:     status,
		StatusCode: resp.StatusCode,
		Latency:    latency,
		Protocol:   "HTTP",
	}
}

func CheckTCP(target string, timeout time.Duration) CheckResult {
	target = strings.TrimSpace(target)

	host, port, err := net.SplitHostPort(target)
	if err != nil || host == "" {
		return CheckResult{
			URL:      target,
			Status:   "INVALID",
			Error:    "Address must use host:port format, e.g. localhost:8080",
			Protocol: "TCP",
		}
	}

	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return CheckResult{
			URL:      target,
			Status:   "INVALID",
			Error:    "Port must be a number between 1 and 65535",
			Protocol: "TCP",
		}
	}

	if timeout <= 0 {
		return CheckResult{
			URL:      target,
			Status:   "INVALID",
			Error:    "Timeout must be greater than zero",
			Protocol: "TCP",
		}
	}

	start := time.Now()
	conn, err := net.DialTimeout("tcp", target, timeout)
	latency := time.Since(start)

	if err != nil {
		return CheckResult{
			URL:      target,
			Status:   "DOWN",
			Latency:  latency,
			Error:    err.Error(),
			Protocol: "TCP",
		}
	}
	conn.Close()

	return CheckResult{
		URL:      target,
		Status:   "UP",
		Latency:  latency,
		Protocol: "TCP",
	}
}

func (r CheckResult) String() string {
	if r.Error != "" {
		if r.Status == "INVALID" {
			return fmt.Sprintf("Invalid %s target: %s | %s",
				r.Protocol, r.URL, r.Error)
		}

		return fmt.Sprintf("%s %s | Error: %s | Latency: %s",
			r.Status, r.URL, r.Error, r.Latency.Round(time.Millisecond))
	}

	if r.Protocol == "TCP" {
		return fmt.Sprintf("%s %s | TCP connection successful | Latency: %s",
			r.Status, r.URL, r.Latency.Round(time.Millisecond))
	}

	return fmt.Sprintf("%s %s | HTTP %d | Latency %s",
		r.Status, r.URL, r.StatusCode, r.Latency.Round(time.Millisecond))
}
