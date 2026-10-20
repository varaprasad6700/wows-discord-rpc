package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wows-discord-rpc/daemon/internal/api"
	"github.com/wows-discord-rpc/daemon/internal/discord"
)

type mockDiscordClient struct {
	lastActivity *discord.Activity
	cleared      bool
}

func (m *mockDiscordClient) SetActivity(act *discord.Activity) error {
	m.lastActivity = act
	m.cleared = false
	return nil
}

func (m *mockDiscordClient) ClearActivity() error {
	m.cleared = true
	m.lastActivity = nil
	return nil
}

func (m *mockDiscordClient) Close() error {
	return nil
}

type mockShipResolver struct{}

func (m *mockShipResolver) GetShip(shipID int64) (*api.ShipMetadata, error) {
	return &api.ShipMetadata{
		ShipID:    shipID,
		Name:      "Yamato",
		Tier:      10,
		RomanTier: "X",
		ShipClass: "Battleship",
	}, nil
}

func TestEngine_TickOnce_GameNotRunning(t *testing.T) {
	mockDiscord := &mockDiscordClient{cleared: false}
	engine := &Engine{
		wowsPath:       t.TempDir(),
		isGameRunning:  func() bool { return false },
		shipResolver:   &mockShipResolver{},
		discordClient:  mockDiscord,
		wasGameRunning: true,
	}

	err := engine.TickOnce()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !mockDiscord.cleared {
		t.Errorf("expected discord presence to be cleared when game is not running")
	}
}

func TestEngine_TickOnce_InPort(t *testing.T) {
	tempDir := t.TempDir()
	mockDiscord := &mockDiscordClient{}
	engine := &Engine{
		wowsPath:      tempDir,
		isGameRunning: func() bool { return true },
		shipResolver:  &mockShipResolver{},
		discordClient: mockDiscord,
	}

	err := engine.TickOnce()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mockDiscord.lastActivity == nil {
		t.Fatal("expected discord activity to be set")
	}

	if mockDiscord.lastActivity.Details != "In Port" {
		t.Errorf("expected 'In Port', got '%s'", mockDiscord.lastActivity.Details)
	}
}

func TestEngine_TickOnce_InBattle(t *testing.T) {
	tempDir := t.TempDir()
	replaysDir := filepath.Join(tempDir, "replays")
	_ = os.MkdirAll(replaysDir, 0755)

	// Create tempArenaInfo.json fixture
	arenaContent := `{
		"mapDisplayName": "spaces/15_NE_north",
		"matchGroup": "pvp",
		"dateTime": "20.10.2026 15:30:00",
		"playerName": "Commander",
		"vehicles": [
			{"name": "Commander", "shipId": 4181230000, "relation": 0}
		]
	}`
	_ = os.WriteFile(filepath.Join(replaysDir, "tempArenaInfo.json"), []byte(arenaContent), 0644)

	// Create wows_live_state.json fixture
	liveContent := `{
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
	_ = os.WriteFile(filepath.Join(tempDir, "wows_live_state.json"), []byte(liveContent), 0644)

	mockDiscord := &mockDiscordClient{}
	engine := &Engine{
		wowsPath:      tempDir,
		isGameRunning: func() bool { return true },
		shipResolver:  &mockShipResolver{},
		discordClient: mockDiscord,
	}

	err := engine.TickOnce()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mockDiscord.lastActivity == nil {
		t.Fatal("expected discord activity to be set")
	}

	expectedDetails := "[X] Yamato • Random Battle (North)"
	if mockDiscord.lastActivity.Details != expectedDetails {
		t.Errorf("expected details '%s', got '%s'", expectedDetails, mockDiscord.lastActivity.Details)
	}

	expectedState := "72k HP (74%) | 84k DMG | 2 Kills | 9v7"
	if mockDiscord.lastActivity.State != expectedState {
		t.Errorf("expected state '%s', got '%s'", expectedState, mockDiscord.lastActivity.State)
	}

	// Now simulate battle end: tempArenaInfo.json removed
	_ = os.Remove(filepath.Join(replaysDir, "tempArenaInfo.json"))
	err = engine.TickOnce()
	if err != nil {
		t.Fatalf("unexpected error on second tick: %v", err)
	}

	if mockDiscord.lastActivity.Details != "In Port" {
		t.Errorf("expected transition back to 'In Port', got '%s'", mockDiscord.lastActivity.Details)
	}
}
