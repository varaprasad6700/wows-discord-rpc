package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	WoWS    WoWSConfig    `yaml:"wows"`
	Backend BackendConfig `yaml:"backend"`
	Discord DiscordConfig `yaml:"discord"`
}

type WoWSConfig struct {
	InstallPath string `yaml:"install_path"`
}

type BackendConfig struct {
	URL string `yaml:"url"`
}

type DiscordConfig struct {
	ClientID              string `yaml:"client_id"`
	UpdateIntervalSeconds int    `yaml:"update_interval_seconds"`
}

func DefaultConfig() *Config {
	return &Config{
		Backend: BackendConfig{
			URL: "http://localhost:8080",
		},
		Discord: DiscordConfig{
			UpdateIntervalSeconds: 15,
		},
	}
}

func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config yaml %s: %w", path, err)
	}

	// Apply fallbacks if empty
	if cfg.Backend.URL == "" {
		cfg.Backend.URL = "http://localhost:8080"
	}
	if cfg.Discord.UpdateIntervalSeconds <= 0 {
		cfg.Discord.UpdateIntervalSeconds = 15
	}

	return cfg, nil
}
