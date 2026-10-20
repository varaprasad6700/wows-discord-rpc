package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wows-discord-rpc/daemon/internal/api"
	"github.com/wows-discord-rpc/daemon/internal/config"
	"github.com/wows-discord-rpc/daemon/internal/detector"
	"github.com/wows-discord-rpc/daemon/internal/discord"
	"github.com/wows-discord-rpc/daemon/internal/process"
)

var (
	version          = "dev"
	defaultDiscordID = "1562101043670556782"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	log.Printf("Starting WoWS Discord Rich Presence Daemon (%s)...", version)

	// 1. Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Printf("Warning: Failed to load config from %s: %v. Using defaults.", *configPath, err)
		cfg = config.DefaultConfig()
	}

	if cfg.Discord.ClientID == "" {
		cfg.Discord.ClientID = defaultDiscordID
	}

	// 2. Discover WoWS installation directory
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
			if err := detector.DeployEmbeddedMod(buildPath); err != nil {
				log.Printf("Warning: Failed to deploy embedded mod to WoWS: %v", err)
			} else {
				log.Printf("Telemetry mod successfully deployed to %s", buildPath)
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
