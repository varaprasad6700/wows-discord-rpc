package discord_test

import (
	"testing"
	"time"

	"github.com/wows-discord-rpc/daemon/internal/api"
	"github.com/wows-discord-rpc/daemon/internal/discord"
	"github.com/wows-discord-rpc/daemon/internal/watcher"
)

func TestFormatOptionC_InBattleAlive(t *testing.T) {
	arena := &watcher.ArenaInfo{
		MapName:      "North",
		BattleType:   "Random Battle",
		PlayerShipID: 4181230000,
		StartTime:    time.Unix(1700000000, 0),
	}

	live := &watcher.LiveState{
		InBattle:     true,
		ShipID:       4181230000,
		Health:       72000,
		MaxHealth:    97200,
		DamageDealt:  84200,
		Frags:        2,
		AlliesAlive:  9,
		EnemiesAlive: 7,
		IsAlive:      true,
	}

	ship := &api.ShipMetadata{
		ShipID:    4181230000,
		Name:      "Yamato",
		Tier:      10,
		RomanTier: "X",
		ShipClass: "Battleship",
		Nation:    "japan",
	}

	activity := discord.FormatOptionC(arena, live, ship)

	expectedDetails := "[X] Yamato • Random Battle (North)"
	if activity.Details != expectedDetails {
		t.Errorf("expected details '%s', got '%s'", expectedDetails, activity.Details)
	}

	expectedState := "72k HP (74%) | 84k DMG | 2 Kills | 9v7"
	if activity.State != expectedState {
		t.Errorf("expected state '%s', got '%s'", expectedState, activity.State)
	}
}

func TestFormatOptionC_InBattleSunk(t *testing.T) {
	arena := &watcher.ArenaInfo{
		MapName:      "North",
		BattleType:   "Random Battle",
		PlayerShipID: 4181230000,
		StartTime:    time.Unix(1700000000, 0),
	}

	live := &watcher.LiveState{
		InBattle:     true,
		ShipID:       4181230000,
		Health:       0,
		MaxHealth:    97200,
		DamageDealt:  84200,
		Frags:        2,
		AlliesAlive:  9,
		EnemiesAlive: 7,
		IsAlive:      false,
	}

	ship := &api.ShipMetadata{
		ShipID:    4181230000,
		Name:      "Yamato",
		Tier:      10,
		RomanTier: "X",
		ShipClass: "Battleship",
		Nation:    "japan",
	}

	activity := discord.FormatOptionC(arena, live, ship)

	expectedState := "Sunk (Spectating) | 84k DMG | 2 Kills | 9v7"
	if activity.State != expectedState {
		t.Errorf("expected state '%s', got '%s'", expectedState, activity.State)
	}
}

func TestFormatOptionC_InPort(t *testing.T) {
	activity := discord.FormatOptionC(nil, nil, nil)

	if activity.Details != "In Port" {
		t.Errorf("expected details 'In Port', got '%s'", activity.Details)
	}

	if activity.State != "Inspecting Fleet & Armory" {
		t.Errorf("expected state 'Inspecting Fleet & Armory', got '%s'", activity.State)
	}
}
