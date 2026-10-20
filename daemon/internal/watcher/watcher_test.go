package watcher_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wows-discord-rpc/daemon/internal/watcher"
)

func TestParseArenaInfo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wows-arena-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	arenaJson := `{
  "dateTime": "20.10.2026 14:30:00",
  "matchGroup": "pvp",
  "mapDisplayName": "spaces/15_NE_north",
  "playerVehicle": "PWSB108_Musashi_1942",
  "playerName": "AdmiralPlayer",
  "vehicles": [
    {
      "name": "AdmiralPlayer",
      "shipId": 4181230000,
      "relation": 0
    },
    {
      "name": "RedEnemy",
      "shipId": 4179110000,
      "relation": 2
    }
  ]
}`

	filePath := filepath.Join(tmpDir, "tempArenaInfo.json")
	if err := os.WriteFile(filePath, []byte(arenaJson), 0644); err != nil {
		t.Fatal(err)
	}

	arena, err := watcher.ParseArenaInfo(filePath)
	if err != nil {
		t.Fatalf("ParseArenaInfo failed: %v", err)
	}

	if arena.MapName != "North" {
		t.Errorf("expected map 'North', got '%s'", arena.MapName)
	}

	if arena.BattleType != "Random Battle" {
		t.Errorf("expected battle type 'Random Battle', got '%s'", arena.BattleType)
	}

	if arena.PlayerShipID != 4181230000 {
		t.Errorf("expected player shipId 4181230000, got %d", arena.PlayerShipID)
	}

	if arena.StartTime.IsZero() {
		t.Error("expected non-zero start time")
	}
}

func TestReadLiveState(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wows-state-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	stateJson := `{
  "in_battle": true,
  "ship_id": 4181230000,
  "health": 72000,
  "max_health": 97200,
  "damage_dealt": 84200,
  "frags": 2,
  "allies_alive": 9,
  "enemies_alive": 7,
  "is_alive": true
}`

	filePath := filepath.Join(tmpDir, "wows_live_state.json")
	if err := os.WriteFile(filePath, []byte(stateJson), 0644); err != nil {
		t.Fatal(err)
	}

	state, err := watcher.ReadLiveState(filePath)
	if err != nil {
		t.Fatalf("ReadLiveState failed: %v", err)
	}

	if !state.InBattle {
		t.Error("expected in_battle = true")
	}

	if state.Health != 72000 {
		t.Errorf("expected health 72000, got %d", state.Health)
	}

	if state.HealthPercent() != 74 {
		t.Errorf("expected health percent 74, got %d", state.HealthPercent())
	}

	if state.AlliesAlive != 9 || state.EnemiesAlive != 7 {
		t.Errorf("expected 9v7, got %dv%d", state.AlliesAlive, state.EnemiesAlive)
	}
}
