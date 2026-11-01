package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfigValid(t *testing.T) {
	content := `{
		"targets": [
			{
				"name": "Google",
				"target": "https://google.com",
				"protocol": "http",
				"interval": "30s",
				"timeout": "5s"
			},
			{
				"name": "PostgreSQL",
				"target": "localhost:5433",
				"protocol": "tcp",
				"interval": "10s",
				"timeout": "3s"
			}
		]
	}`

	path := writeConfigFile(t, content)

	configs, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if len(configs) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(configs))
	}

	if configs[0].Name != "Google" {
		t.Errorf("expected Google, got %s", configs[0].Name)
	}

	if configs[0].Interval != 30*time.Second {
		t.Errorf("unexpected interval: %v", configs[0].Interval)
	}

	if configs[0].Timeout != 5*time.Second {
		t.Errorf("unexpected timeout: %v", configs[0].Timeout)
	}

	if configs[1].Protocol != "tcp" {
		t.Errorf("expected tcp, got %s", configs[1].Protocol)
	}
}

func TestLoadConfigInvalid(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "invalid JSON",
			content: `{invalid json}`,
		},
		{
			name:    "empty targets",
			content: `{"targets":[]}`,
		},
		{
			name: "empty name",
			content: `{"targets":[{
				"name":"",
				"target":"https://google.com",
				"protocol":"http",
				"interval":"30s",
				"timeout":"5s"
			}]}`,
		},
		{
			name: "empty target",
			content: `{"targets":[{
				"name":"Google",
				"target":"",
				"protocol":"http",
				"interval":"30s",
				"timeout":"5s"
			}]}`,
		},
		{
			name: "invalid protocol",
			content: `{"targets":[{
				"name":"Google",
				"target":"https://google.com",
				"protocol":"ftp",
				"interval":"30s",
				"timeout":"5s"
			}]}`,
		},
		{
			name: "invalid interval",
			content: `{"targets":[{
				"name":"Google",
				"target":"https://google.com",
				"protocol":"http",
				"interval":"invalid",
				"timeout":"5s"
			}]}`,
		},
		{
			name: "zero interval",
			content: `{"targets":[{
				"name":"Google",
				"target":"https://google.com",
				"protocol":"http",
				"interval":"0s",
				"timeout":"5s"
			}]}`,
		},
		{
			name: "negative timeout",
			content: `{"targets":[{
				"name":"Google",
				"target":"https://google.com",
				"protocol":"http",
				"interval":"30s",
				"timeout":"-5s"
			}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeConfigFile(t, tt.content)

			_, err := LoadConfig(path)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestLoadConfigFileNotFound(t *testing.T) {
	_, err := LoadConfig(filepath.Join(t.TempDir(), "missing.json"))

	if err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "targets.json")

	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	return path
}
