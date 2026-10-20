package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/wows-discord-rpc/daemon/internal/api"
	"github.com/wows-discord-rpc/daemon/internal/config"
	"github.com/wows-discord-rpc/daemon/internal/detector"
	"github.com/wows-discord-rpc/daemon/internal/discord"
	"github.com/wows-discord-rpc/daemon/internal/logging"
	"github.com/wows-discord-rpc/daemon/internal/process"
	"github.com/wows-discord-rpc/daemon/internal/server"
	"github.com/wows-discord-rpc/daemon/internal/tray"
	"github.com/wows-discord-rpc/daemon/internal/updater"
)

var (
	version          = "v1.1.0"
	defaultDiscordID = "1562101043670556782"
)

type stateAdapter struct {
	mu            sync.RWMutex
	cfg           *config.Config
	configPath    string
	engine        *Engine
	updater       *updater.Updater
	discordClient *discord.Client
}

func (s *stateAdapter) GetStatus() server.StatusDTO {
	gameRunning, details, state := s.engine.GetStatus()
	ipcState := "Disconnected"
	if s.discordClient != nil && s.discordClient.IsConnected() {
		ipcState = "Connected (\\pipe\\discord-ipc-0)"
	}
	return server.StatusDTO{
		GameRunning:     gameRunning,
		DiscordIPCState: ipcState,
		Details:         details,
		State:           state,
	}
}

func (s *stateAdapter) GetConfig() server.ConfigDTO {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return server.ConfigDTO{
		InstallPath: s.cfg.WoWS.InstallPath,
		BackendURL:  s.cfg.Backend.URL,
	}
}

func (s *stateAdapter) SaveConfig(dto server.ConfigDTO) error {
	s.mu.Lock()
	s.cfg.WoWS.InstallPath = dto.InstallPath
	s.cfg.Backend.URL = dto.BackendURL
	path := s.configPath
	cfgCopy := *s.cfg
	s.mu.Unlock()

	// Hot reload in engine
	s.engine.SetWoWSPath(dto.InstallPath)

	logging.Logf("Configuration updated: Path=%s, Backend=%s", dto.InstallPath, dto.BackendURL)
	return config.Save(&cfgCopy, path)
}

func (s *stateAdapter) CheckUpdate() (*server.UpdateDTO, error) {
	info, err := s.updater.Check()
	if err != nil {
		return nil, err
	}
	return &server.UpdateDTO{
		CurrentVersion: info.CurrentVersion,
		LatestVersion:  info.LatestVersion,
		ReleaseNotes:   info.ReleaseNotes,
		DownloadURL:    info.DownloadURL,
		UpdateReady:    info.UpdateReady,
	}, nil
}

func (s *stateAdapter) ApplyUpdate(downloadURL string) error {
	logging.Logf("Applying in-place update from %s...", downloadURL)
	return s.updater.Apply(downloadURL)
}

func main() {
	headlessFlag := flag.Bool("headless", false, "Run in headless console mode without system tray")
	flag.Parse()

	// 1. Initialize logging
	logFile := logging.DefaultLogPath()
	_ = logging.InitFileLogger(logFile)
	logging.Logf("Starting WoWS Discord Rich Presence Daemon (%s)...", version)

	// 2. Load configuration (try AppData first, then local config.yaml)
	configPath := config.DefaultConfigPath()
	cfg, err := config.Load(configPath)
	if err != nil {
		if localCfg, lerr := config.Load("config.yaml"); lerr == nil {
			cfg = localCfg
			configPath = "config.yaml"
		} else {
			cfg = config.DefaultConfig()
		}
	}

	if cfg.Discord.ClientID == "" {
		cfg.Discord.ClientID = defaultDiscordID
	}

	// 3. Discover WoWS installation directory
	wowsPath := cfg.WoWS.InstallPath
	if wowsPath == "" {
		detected, err := detector.FindWoWSPath()
		if err == nil {
			wowsPath = detected
			cfg.WoWS.InstallPath = detected
			_ = config.Save(cfg, configPath)
		}
	}

	if wowsPath == "" {
		logging.Logf("Notice: World of Warships installation directory not found automatically. Please configure it in settings.")
	} else {
		logging.Logf("World of Warships Path: %s", wowsPath)

		// Deploy embedded mod
		if buildPath, err := detector.FindLatestBuild(wowsPath); err == nil {
			if err := detector.DeployEmbeddedMod(buildPath); err == nil {
				logging.Logf("Embedded telemetry mod successfully deployed to %s", buildPath)
			}
		}
	}

	// 4. Initialize API and Discord IPC clients
	shipClient := api.NewClient(cfg.Backend.URL)
	discordClient := discord.NewClient(cfg.Discord.ClientID)

	if err := discordClient.Connect(); err != nil {
		logging.Logf("Notice: Could not connect to Discord IPC on startup: %v (will retry)", err)
	} else {
		logging.Logf("Connected to Discord IPC named pipe.")
	}

	// 5. Create Engine and Updater
	engine := NewEngine(wowsPath, process.IsGameRunning, shipClient, discordClient)
	up := updater.NewUpdater(version)

	adapter := &stateAdapter{
		cfg:           cfg,
		configPath:    configPath,
		engine:        engine,
		updater:       up,
		discordClient: discordClient,
	}

	// 6. Start Loopback HTTP Server
	srv := server.NewServer(0, adapter)
	port, err := srv.Start()
	if err != nil {
		logging.Logf("Warning: Could not start UI loopback server: %v", err)
	} else {
		logging.Logf("UI loopback server running on http://127.0.0.1:%d", port)
	}

	// 7. Start Game State Polling Loop in background
	interval := time.Duration(cfg.Discord.UpdateIntervalSeconds) * time.Second
	if interval < 5*time.Second {
		interval = 15 * time.Second
	}
	ticker := time.NewTicker(interval)
	go func() {
		_ = engine.TickOnce()
		for range ticker.C {
			if !discordClient.IsConnected() {
				_ = discordClient.Connect()
			}
			if err := engine.TickOnce(); err != nil {
				logging.Logf("Error during presence update: %v", err)
			}
		}
	}()

	cleanup := func() {
		logging.Logf("Stopping daemon cleanly...")
		ticker.Stop()
		_ = discordClient.ClearActivity()
		_ = discordClient.Close()
		_ = srv.Close()
		tray.Remove()
	}

	// 8. If headless or not windows: listen for signals
	if *headlessFlag || runtime.GOOS != "windows" {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		cleanup()
		os.Exit(0)
	}

	// 9. Windows System Tray Mode
	onOpenUI := func() {
		url := fmt.Sprintf("http://127.0.0.1:%d", port)
		_ = tray.LaunchAppMode(url)
	}

	onOpenLogs := func() {
		logPath := logging.GetLogFilePath()
		dir := filepath.Dir(logPath)
		_ = os.MkdirAll(dir, 0755)
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			_ = os.WriteFile(logPath, []byte(""), 0644)
		}
		_ = exec.Command("notepad.exe", logPath).Start()
	}

	onCheckUpdates := func() {
		onOpenUI()
	}

	onExit := func() {
		cleanup()
		os.Exit(0)
	}

	// Start tray message loop (blocks main thread)
	if err := tray.Run(onOpenUI, onOpenLogs, onCheckUpdates, onExit); err != nil {
		logging.Logf("Tray error: %v, falling back to signal wait", err)
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		cleanup()
	}
}
