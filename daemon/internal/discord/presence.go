package discord

import (
	"fmt"
	"strings"

	"github.com/wows-discord-rpc/daemon/internal/api"
	"github.com/wows-discord-rpc/daemon/internal/watcher"
)

type Timestamps struct {
	Start int64 `json:"start,omitempty"`
	End   int64 `json:"end,omitempty"`
}

type Assets struct {
	LargeImage string `json:"large_image,omitempty"`
	LargeText  string `json:"large_text,omitempty"`
	SmallImage string `json:"small_image,omitempty"`
	SmallText  string `json:"small_text,omitempty"`
}

type Activity struct {
	Details    string      `json:"details,omitempty"`
	State      string      `json:"state,omitempty"`
	Timestamps *Timestamps `json:"timestamps,omitempty"`
	Assets     *Assets     `json:"assets,omitempty"`
}

// FormatOptionC builds a high-density Option C Discord Rich Presence activity.
func FormatOptionC(arena *watcher.ArenaInfo, live *watcher.LiveState, ship *api.ShipMetadata) *Activity {
	if arena == nil || live == nil || !live.InBattle {
		return &Activity{
			Details: "In Port",
			State:   "Inspecting Fleet & Armory",
			Assets: &Assets{
				LargeImage: "wows_logo",
				LargeText:  "World of Warships",
			},
		}
	}

	// Resolve ship tier and display name
	tier := "?"
	name := fmt.Sprintf("Ship #%d", live.ShipID)
	if arena.PlayerVehicle != "" {
		if cleanName := watcher.CleanVehicleName(arena.PlayerVehicle); cleanName != "" {
			name = cleanName
		}
	}

	var shipClass string

	if ship != nil {
		if ship.RomanTier != "" {
			tier = ship.RomanTier
		} else if ship.Tier > 0 {
			tier = fmt.Sprintf("%d", ship.Tier)
		}
		if ship.Name != "" && !strings.HasPrefix(ship.Name, "Ship #") {
			name = ship.Name
		}
		shipClass = ship.ShipClass
	}

	// Details line: [{RomanTier}] {ShipName} • {BattleType} ({MapName})
	var shipDisplay string
	if tier != "" && tier != "?" {
		shipDisplay = fmt.Sprintf("[%s] %s", tier, name)
	} else {
		shipDisplay = name
	}

	var details string
	if arena.BattleType != "" && arena.MapName != "" {
		details = fmt.Sprintf("%s • %s (%s)", shipDisplay, arena.BattleType, arena.MapName)
	} else if arena.BattleType != "" {
		details = fmt.Sprintf("%s • %s", shipDisplay, arena.BattleType)
	} else {
		details = shipDisplay
	}

	// State line:
	// Alive: {HP}k HP ({%}%) | {DMG}k DMG | {Frags} Kills | {Allies}v{Enemies}
	// Sunk:  Sunk (Spectating) | {DMG}k DMG | {Frags} Kills | {Allies}v{Enemies}
	dmgK := live.DamageDealt / 1000
	killsText := fmt.Sprintf("%d Kills", live.Frags)
	if live.Frags == 1 {
		killsText = "1 Kill"
	}

	var state string
	if live.IsAlive && live.Health > 0 {
		hpK := live.Health / 1000
		hpPct := live.HealthPercent()
		state = fmt.Sprintf("%dk HP (%d%%) | %dk DMG | %s | %dv%d", hpK, hpPct, dmgK, killsText, live.AlliesAlive, live.EnemiesAlive)
	} else {
		state = fmt.Sprintf("Sunk (Spectating) | %dk DMG | %s | %dv%d", dmgK, killsText, live.AlliesAlive, live.EnemiesAlive)
	}

	var timestamps *Timestamps
	if !arena.StartTime.IsZero() {
		timestamps = &Timestamps{
			Start: arena.StartTime.Unix(),
		}
	}

	assets := &Assets{
		LargeImage: "wows_logo",
		LargeText:  fmt.Sprintf("%s - %d/%d HP", name, live.Health, live.MaxHealth),
	}
	if shipClass != "" {
		assets.SmallText = shipClass
	}

	return &Activity{
		Details:    details,
		State:      state,
		Timestamps: timestamps,
		Assets:     assets,
	}
}
