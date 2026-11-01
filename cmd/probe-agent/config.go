package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

type TargetConfig struct {
	Name     string        `json:"name"`
	Target   string        `json:"target"`
	Protocol string        `json:"protocol"`
	Interval time.Duration `json:"-"`
	Timeout  time.Duration `json:"-"`
}

type rawTargetConfig struct {
	Name     string `json:"name"`
	Target   string `json:"target"`
	Protocol string `json:"protocol"`
	Interval string `json:"interval"`
	Timeout  string `json:"timeout"`
}

type rawConfig struct {
	Targets []rawTargetConfig `json:"targets"`
}

func LoadConfig(path string) ([]TargetConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var raw rawConfig

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("decode config JSON: %w", err)
	}

	if len(raw.Targets) == 0 {
		return nil, errors.New("configuration must contain at least one target")
	}

	configs := make([]TargetConfig, 0, len(raw.Targets))

	for i, item := range raw.Targets {
		config, err := parseTargetConfig(item)
		if err != nil {
			return nil, fmt.Errorf("target at index %d: %w", i, err)
		}

		configs = append(configs, config)
	}

	return configs, nil
}

func parseTargetConfig(raw rawTargetConfig) (TargetConfig, error) {
	config := TargetConfig{
		Name:     strings.TrimSpace(raw.Name),
		Target:   strings.TrimSpace(raw.Target),
		Protocol: strings.ToLower(strings.TrimSpace(raw.Protocol)),
	}

	if config.Name == "" {
		return TargetConfig{}, errors.New("name must not be empty")
	}

	if config.Target == "" {
		return TargetConfig{}, errors.New("target must not be empty")
	}

	if config.Protocol != "http" && config.Protocol != "tcp" {
		return TargetConfig{}, errors.New("protocol must be http or tcp")
	}

	interval, err := time.ParseDuration(strings.TrimSpace(raw.Interval))
	if err != nil {
		return TargetConfig{}, fmt.Errorf("invalid interval: %w", err)
	}

	if interval <= 0 {
		return TargetConfig{}, errors.New("interval must be positive")
	}

	timeout, err := time.ParseDuration(strings.TrimSpace(raw.Timeout))
	if err != nil {
		return TargetConfig{}, fmt.Errorf("invalid timeout: %w", err)
	}

	if timeout <= 0 {
		return TargetConfig{}, errors.New("timeout must be positive")
	}

	config.Interval = interval
	config.Timeout = timeout

	return config, nil
}
