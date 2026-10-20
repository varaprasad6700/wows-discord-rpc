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

	if cfg.Backend.URL != "https://wows-meta.varaprasad.co.in" {
		t.Errorf("expected backend URL 'https://wows-meta.varaprasad.co.in', got '%s'", cfg.Backend.URL)
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

	if cfg.Backend.URL != "https://wows-meta.varaprasad.co.in" {
		t.Errorf("expected default backend URL 'https://wows-meta.varaprasad.co.in', got '%s'", cfg.Backend.URL)
	}

	if cfg.Discord.ClientID != "999" {
		t.Errorf("expected custom client ID '999', got '%s'", cfg.Discord.ClientID)
	}

	if cfg.Discord.UpdateIntervalSeconds != 15 {
		t.Errorf("expected default interval 15, got %d", cfg.Discord.UpdateIntervalSeconds)
	}
}


func TestSaveAndLoadConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wows_cfg_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfgPath := filepath.Join(tmpDir, "config.yaml")
	cfg := config.DefaultConfig()
	cfg.WoWS.InstallPath = `E:\TestPath\WoWS`
	cfg.Backend.URL = "https://custom-meta.example.com"

	if err := config.Save(cfg, cfgPath); err != nil {
		t.Fatalf("config.Save failed: %v", err)
	}

	loaded, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("config.Load failed: %v", err)
	}

	if loaded.WoWS.InstallPath != `E:\TestPath\WoWS` {
		t.Errorf("expected WoWS path 'E:\\TestPath\\WoWS', got '%s'", loaded.WoWS.InstallPath)
	}
	if loaded.Backend.URL != "https://custom-meta.example.com" {
		t.Errorf("expected custom backend URL, got '%s'", loaded.Backend.URL)
	}
}
