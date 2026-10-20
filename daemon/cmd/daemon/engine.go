package main

import (
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/wows-discord-rpc/daemon/internal/api"
	"github.com/wows-discord-rpc/daemon/internal/discord"
	"github.com/wows-discord-rpc/daemon/internal/watcher"
)

type ShipResolver interface {
	GetShip(shipID int64) (*api.ShipMetadata, error)
}

type DiscordActivityClient interface {
	SetActivity(act *discord.Activity) error
	ClearActivity() error
	Close() error
}

type Engine struct {
	mu             sync.RWMutex
	wowsPath       string
	isGameRunning  func() bool
	shipResolver   ShipResolver
	discordClient  DiscordActivityClient
	wasGameRunning bool
}

func NewEngine(wowsPath string, isGameRunning func() bool, shipResolver ShipResolver, discordClient DiscordActivityClient) *Engine {
	return &Engine{
		wowsPath:      wowsPath,
		isGameRunning: isGameRunning,
		shipResolver:  shipResolver,
		discordClient: discordClient,
	}
}

func (e *Engine) SetWoWSPath(p string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.wowsPath = p
}

func (e *Engine) GetWoWSPath() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.wowsPath
}

// FindLiveStateFile searches for wows_live_state.json across game root, bin/<build>, and bin/<build>/bin64
func FindLiveStateFile(wowsPath string) string {
	if wowsPath == "" {
		return ""
	}

	// 1. Root directory takes priority
	rootCandidate := filepath.Join(wowsPath, "wows_live_state.json")
	if stat, err := os.Stat(rootCandidate); err == nil && !stat.IsDir() {
		return rootCandidate
	}

	// 2. Scan build directories in bin/
	binDir := filepath.Join(wowsPath, "bin")
	entries, err := os.ReadDir(binDir)
	if err == nil {
		for i := len(entries) - 1; i >= 0; i-- {
			entry := entries[i]
			if !entry.IsDir() {
				continue
			}
			buildCandidate := filepath.Join(binDir, entry.Name(), "wows_live_state.json")
			if stat, err := os.Stat(buildCandidate); err == nil && !stat.IsDir() {
				return buildCandidate
			}
			bin64Candidate := filepath.Join(binDir, entry.Name(), "bin64", "wows_live_state.json")
			if stat, err := os.Stat(bin64Candidate); err == nil && !stat.IsDir() {
				return bin64Candidate
			}
		}
	}

	return ""
}

func (e *Engine) TickOnce() error {
	e.mu.RLock()
	currentPath := e.wowsPath
	e.mu.RUnlock()

	running := e.isGameRunning()
	if !running {
		if e.wasGameRunning {
			log.Println("World of Warships closed. Clearing Discord activity.")
			_ = e.discordClient.ClearActivity()
			e.wasGameRunning = false
		}
		return nil
	}

	e.wasGameRunning = true

	// Check if tempArenaInfo.json exists in replays
	arenaPath := filepath.Join(currentPath, "replays", "tempArenaInfo.json")
	if _, err := os.Stat(arenaPath); os.IsNotExist(err) {
		// In Port
		act := discord.FormatOptionC(nil, nil, nil)
		return e.discordClient.SetActivity(act)
	}

	// In Battle - parse arena info
	arena, err := watcher.ParseArenaInfo(arenaPath)
	if err != nil {
		log.Printf("Warning: failed to parse arena info: %v. Falling back to port.", err)
		act := discord.FormatOptionC(nil, nil, nil)
		return e.discordClient.SetActivity(act)
	}

	// Read live telemetry state
	livePath := FindLiveStateFile(currentPath)
	var live *watcher.LiveState
	if livePath != "" {
		live, _ = watcher.ReadLiveState(livePath)
	}
	if live == nil {
		// Telemetry might not have flushed first tick yet
		live = &watcher.LiveState{
			InBattle:     true,
			ShipID:       arena.PlayerShipID,
			Health:       1000,
			MaxHealth:    1000,
			IsAlive:      true,
			AlliesAlive:  12,
			EnemiesAlive: 12,
		}
	}

	// Resolve ship metadata
	shipID := arena.PlayerShipID
	if shipID == 0 && live != nil {
		shipID = live.ShipID
	}

	shipMeta, _ := e.shipResolver.GetShip(shipID)

	act := discord.FormatOptionC(arena, live, shipMeta)
	return e.discordClient.SetActivity(act)
}
