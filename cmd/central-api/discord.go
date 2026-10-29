package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type discordMessage struct {
	Content         string `json:"content"`
	AllowedMentions struct {
		Parse []string `json:"parse"`
	} `json:"allowed_mentions"`
}

type discordSender struct {
	webhookURL string
	client     *http.Client
}

func newDiscordSender(webhookURL string) *discordSender {
	return &discordSender{
		webhookURL: strings.TrimSpace(webhookURL),
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (s *discordSender) send(ctx context.Context, event alertEvent) error {
	if s.webhookURL == "" {
		return fmt.Errorf("Discord webhook is not configured")
	}

	title := "PulseCheck: Target DOWN"
	if event.CurrentStatus == "UP" {
		title = "PulseCheck: Target RECOVERED"
	}

	content := fmt.Sprintf(
		"%s\nProbe: %s\nTarget: %s\nTransition: %s → %s",
		title,
		event.ProbeID,
		event.Target,
		event.PreviousStatus,
		event.CurrentStatus,
	)

	if event.Error != "" && event.CurrentStatus == "DOWN" {
		content += "\nError: " + event.Error
	}

	message := discordMessage{Content: content}
	message.AllowedMentions.Parse = []string{}
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("encode Discord message: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		s.webhookURL,
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("create Discord request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send Discord webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Discord webhook returned HTTP %d", resp.StatusCode)
	}

	return nil
}
