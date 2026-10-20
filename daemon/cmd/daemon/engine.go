package main

import (
	"log"
	"os"
	"path/filepath"

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

func (e *Engine) TickOnce() error {
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
	arenaPath := filepath.Join(e.wowsPath, "replays", "tempArenaInfo.json")
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
	livePath := filepath.Join(e.wowsPath, "wows_live_state.json")
	live, err := watcher.ReadLiveState(livePath)
	if err != nil {
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
