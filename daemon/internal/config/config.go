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

const (
	DefaultBackendURL      = "https://wows-meta.varaprasad.co.in"
	DefaultDiscordClientID = "1562101043670556782"
)

func DefaultConfig() *Config {
	return &Config{
		Backend: BackendConfig{
			URL: DefaultBackendURL,
		},
		Discord: DiscordConfig{
			ClientID:              DefaultDiscordClientID,
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
		cfg.Backend.URL = DefaultBackendURL
	}
	if cfg.Discord.ClientID == "" {
		cfg.Discord.ClientID = DefaultDiscordClientID
	}
	if cfg.Discord.UpdateIntervalSeconds <= 0 {
		cfg.Discord.UpdateIntervalSeconds = 15
	}

	return cfg, nil
}
