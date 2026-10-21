package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type CheckResult struct {
	URL        string
	Status     string
	StatusCode int
	Latency    time.Duration
	Error      string
}

func Check(target string) CheckResult {
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

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

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
	}
}

func (r CheckResult) String() string {
	if r.Error != "" {
		if r.Status == "INVALID" {
			return fmt.Sprintf("Invalid URL: %s\n%s", r.URL, r.Error)
		}

		return fmt.Sprintf("%s %s | Error: %s | Latency: %s",
			r.Status, r.URL, r.Error, r.Latency.Round(time.Millisecond))
	}

	return fmt.Sprintf("%s %s | HTTP %d | Latency %s",
		r.Status, r.URL, r.StatusCode, r.Latency.Round(time.Millisecond))
}
