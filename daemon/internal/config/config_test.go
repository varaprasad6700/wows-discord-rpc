package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wows-discord-rpc/daemon/internal/config"
)

func TestLoadConfig_ExampleFile(t *testing.T) {
	// Look for config.example.yaml at project root
	rootPath := filepath.Join("..", "..", "..", "config.example.yaml")
	cfg, err := config.Load(rootPath)
	if err != nil {
		t.Fatalf("expected config.example.yaml to load successfully, got: %v", err)
	}

	if cfg.Backend.URL != "http://localhost:8080" {
		t.Errorf("expected backend URL 'http://localhost:8080', got '%s'", cfg.Backend.URL)
	}

	if cfg.Discord.ClientID != "1562101043670556782" {
		t.Errorf("expected discord client ID '1562101043670556782', got '%s'", cfg.Discord.ClientID)
	}

	if cfg.Discord.UpdateIntervalSeconds != 15 {
		t.Errorf("expected update interval 15, got %d", cfg.Discord.UpdateIntervalSeconds)
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	content := []byte(`
discord:
  client_id: "999"
`)
	if _, err := tmpFile.Write(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	cfg, err := config.Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Backend.URL != "http://localhost:8080" {
		t.Errorf("expected default backend URL 'http://localhost:8080', got '%s'", cfg.Backend.URL)
	}

	if cfg.Discord.ClientID != "999" {
		t.Errorf("expected custom client ID '999', got '%s'", cfg.Discord.ClientID)
	}

	if cfg.Discord.UpdateIntervalSeconds != 15 {
		t.Errorf("expected default interval 15, got %d", cfg.Discord.UpdateIntervalSeconds)
	}
}
