package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/wows-discord-rpc/daemon/internal/api"
	"github.com/wows-discord-rpc/daemon/internal/config"
	"github.com/wows-discord-rpc/daemon/internal/detector"
	"github.com/wows-discord-rpc/daemon/internal/discord"
	"github.com/wows-discord-rpc/daemon/internal/process"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to daemon configuration file")
	flag.Parse()

	log.Println("=====================================================")
	log.Println("  World of Warships Discord Rich Presence Daemon")
	log.Println("=====================================================")

	// 1. Load Configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Printf("Could not load '%s' (%v). Attempting config.example.yaml...", *configPath, err)
		cfg, err = config.Load("config.example.yaml")
		if err != nil {
			log.Println("Using built-in defaults.")
			cfg = &config.Config{
				Backend: config.BackendConfig{URL: "http://localhost:8080"},
				Discord: config.DiscordConfig{
					ClientID:              "123456789012345678",
					UpdateIntervalSeconds: 15,
				},
			}
		}
	}

	log.Printf("Backend URL: %s", cfg.Backend.URL)
	log.Printf("Discord Client ID: %s", cfg.Discord.ClientID)
	log.Printf("Update Interval: %ds", cfg.Discord.UpdateIntervalSeconds)

	// 2. Resolve WoWS installation path
	wowsPath := cfg.WoWS.InstallPath
	if wowsPath == "" {
		detected, err := detector.FindWoWSPath()
		if err == nil {
			wowsPath = detected
		}
	}

	if wowsPath == "" {
		log.Println("Warning: World of Warships installation directory not found automatically.")
		log.Println("Please set 'wows.install_path' in your config.yaml.")
	} else {
		log.Printf("World of Warships Path: %s", wowsPath)

		// 3. Locate latest build and ensure mod is deployed
		buildPath, err := detector.FindLatestBuild(wowsPath)
		if err != nil {
			log.Printf("Warning: Failed to detect latest WoWS build directory: %v", err)
		} else {
			log.Printf("Detected latest WoWS build: %s", buildPath)

			// Look for mod directory relative to current working directory or binary
			modDir := findModSourceDir()
			if modDir != "" {
				if err := detector.EnsureModDeployed(buildPath, modDir); err != nil {
					log.Printf("Warning: Failed to deploy mod to WoWS: %v", err)
				} else {
					log.Printf("Telemetry mod successfully deployed to %s", buildPath)
				}
			} else {
				log.Println("Notice: 'mod' directory not found locally; skipping auto-deployment.")
			}
		}
	}

	// 4. Initialize API and Discord clients
	shipClient := api.NewClient(cfg.Backend.URL)
	discordClient := discord.NewClient(cfg.Discord.ClientID)

	if err := discordClient.Connect(); err != nil {
		log.Printf("Notice: Could not connect to Discord IPC on startup: %v (will retry during loop)", err)
	} else {
		log.Println("Connected to Discord IPC named pipe.")
	}

	// 5. Setup graceful shutdown handler
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Printf("Received signal %v. Cleaning up Discord Rich Presence...", sig)
		_ = discordClient.ClearActivity()
		_ = discordClient.Close()
		log.Println("Daemon stopped cleanly.")
		os.Exit(0)
	}()

	// 6. Create Engine and Start Main Polling Loop
	engine := NewEngine(wowsPath, process.IsGameRunning, shipClient, discordClient)

	// Perform initial tick
	_ = engine.TickOnce()

	interval := time.Duration(cfg.Discord.UpdateIntervalSeconds) * time.Second
	if interval < 5*time.Second {
		interval = 15 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Daemon running. Polling game state every %v...", interval)
	for range ticker.C {
		if err := engine.TickOnce(); err != nil {
			log.Printf("Error during presence update: %v", err)
		}
	}
}

func findModSourceDir() string {
	candidates := []string{
		"mod",
		"../mod",
		"../../mod",
		filepath.Join(filepath.Dir(os.Args[0]), "mod"),
		filepath.Join(filepath.Dir(os.Args[0]), "..", "mod"),
	}

	for _, c := range candidates {
		if stat, err := os.Stat(c); err == nil && stat.IsDir() {
			if _, err := os.Stat(filepath.Join(c, "PnFModsLoader.py")); err == nil {
				return c
			}
		}
	}
	return ""
}
