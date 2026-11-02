package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type HeartbeatRequest struct {
	ProbeID  string `json:"probe_id"`
	Hostname string `json:"hostname"`
}

func sendHeartbeat(
	ctx context.Context,
	client *http.Client,
	apiURL string,
	apiToken string,
	probeID string,
	hostname string,
) error {
	payload := HeartbeatRequest{
		ProbeID:  probeID,
		Hostname: hostname,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal heartbeat: %w", err)
	}

	endpoint := strings.TrimRight(apiURL, "/") +
		"/api/v1/agents/heartbeat"

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("create heartbeat request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiToken)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send heartbeat: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("heartbeat API returned status %d", resp.StatusCode)
	}

	return nil
}

func runHeartbeat(
	ctx context.Context,
	client *http.Client,
	apiURL string,
	apiToken string,
	probeID string,
	hostname string,
	interval time.Duration,
) {
	if interval <= 0 {
		log.Printf("Invalid heartbeat interval: %s", interval)
		return
	}

	send := func() {
		if err := sendHeartbeat(
			ctx,
			client,
			apiURL,
			apiToken,
			probeID,
			hostname,
		); err != nil {
			if ctx.Err() == nil {
				log.Printf("Heartbeat failed: %v", err)
			}
			return
		}

		log.Printf("Heartbeat sent: probe_id=%s hostname=%s", probeID, hostname)
	}

	send()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Heartbeat worker stopped: probe_id=%s", probeID)
			return

		case <-ticker.C:
			send()
		}
	}
}
