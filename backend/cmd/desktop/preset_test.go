package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSeedDefaultModelConfigWritesPresetWhenAbsent(t *testing.T) {
	dataDir := t.TempDir()
	seedDefaultModelConfig(dataDir)
	body, err := os.ReadFile(filepath.Join(dataDir, "local-model-config.json"))
	if err != nil {
		t.Fatalf("预设未写入: %v", err)
	}
	var document struct {
		SchemaVersion int `json:"schemaVersion"`
		Config        struct {
			Channels []struct {
				ID      string `json:"id"`
				APIKey  string `json:"apiKey"`
				BaseURL string `json:"baseUrl"`
				Models  []struct {
					Model string `json:"model"`
				} `json:"modelProfiles"`
			} `json:"channels"`
		} `json:"config"`
	}
	if err := json.Unmarshal(body, &document); err != nil {
		t.Fatalf("预设不是合法 JSON: %v", err)
	}
	if document.SchemaVersion != 1 {
		t.Fatalf("schemaVersion = %d, want 1", document.SchemaVersion)
	}
	if len(document.Config.Channels) != 1 {
		t.Fatalf("channels = %d, want 1", len(document.Config.Channels))
	}
	channel := document.Config.Channels[0]
	if channel.ID != "tudou" || channel.BaseURL != "https://api.ai-tudou.net" {
		t.Fatalf("渠道预设不符: id=%q baseUrl=%q", channel.ID, channel.BaseURL)
	}
	if channel.APIKey != "" {
		t.Fatalf("预设不得携带密钥, got %q", channel.APIKey)
	}
	if len(channel.Models) != 11 {
		t.Fatalf("模型数 = %d, want 11", len(channel.Models))
	}
}

func TestSeedDefaultModelConfigKeepsExistingConfig(t *testing.T) {
	dataDir := t.TempDir()
	path := filepath.Join(dataDir, "local-model-config.json")
	existing := []byte(`{"schemaVersion":1,"revision":7,"config":{"channels":[]}}`)
	if err := os.WriteFile(path, existing, 0o600); err != nil {
		t.Fatal(err)
	}
	seedDefaultModelConfig(dataDir)
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != string(existing) {
		t.Fatalf("已有配置被覆盖: %s", body)
	}
}
