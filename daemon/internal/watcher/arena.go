package watcher

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type RawArenaInfo struct {
	DateTime       string           `json:"dateTime"`
	MatchGroup     string           `json:"matchGroup"`
	MapDisplayName string           `json:"mapDisplayName"`
	PlayerName     string           `json:"playerName"`
	PlayerVehicle  string           `json:"playerVehicle"`
	Vehicles       []RawArenaPlayer `json:"vehicles"`
}

type RawArenaPlayer struct {
	Name     string `json:"name"`
	ShipID   int64  `json:"shipId"`
	Relation int    `json:"relation"` // 0: self, 1: ally, 2: enemy
}

type ArenaInfo struct {
	MapName       string
	BattleType    string
	PlayerShipID  int64
	PlayerVehicle string
	StartTime     time.Time
}

var mapDictionary = map[string]string{
	"spaces/15_NE_north":              "North",
	"spaces/01_solomon_islands":       "Solomon Islands",
	"spaces/04_archipelago":           "Archipelago",
	"spaces/05_ring":                  "Ring",
	"spaces/08_ruins":                 "Haven",
	"spaces/10_NE_big_race":           "Big Race",
	"spaces/13_OC_new_dawn":           "New Dawn",
	"spaces/14_Atlantic":              "Atlantic",
	"spaces/16_OC_bees_to_honey":      "Fault Line",
	"spaces/17_NA_two_brothers":       "Two Brothers",
	"spaces/18_NE_ice_islands":        "Islands of Ice",
	"spaces/19_OC_prey":               "Trap",
	"spaces/20_NE_sunken_islands":     "Tears of the Desert",
	"spaces/22_tierra_del_fuego":      "Land of Fire",
	"spaces/23_shards":                "Shards",
	"spaces/25_sea_hope":              "Sea of Fortune",
	"spaces/28_naval_mission":         "Warrior's Path",
	"spaces/33_two_path":              "Loop",
	"spaces/34_OC_bee_and_wasp":       "Estuary",
	"spaces/35_Sleeping_Giant":        "Sleeping Giant",
	"spaces/37_Ridge":                 "Mountain Range",
	"spaces/38_Canada":                "Crash Zone Alpha",
	"spaces/40_Okinawa":               "Okinawa",
	"spaces/41_SZ_Northern_Waters":    "Northern Waters",
	"spaces/42_SZ_Hotspot":            "Hotspot",
	"spaces/44_epicenter":             "Trident",
	"spaces/45_Greece":                "Greece",
	"spaces/46_Ice_Islands":           "Riposte",
	"spaces/47_Roosevelt":             "Faroe Islands",
	"spaces/50_Gold_island":           "Seychelles",
}

var battleTypeMap = map[string]string{
	"pvp":         "Random Battle",
	"ranked":      "Ranked Battle",
	"cooperative": "Co-op Battle",
	"clan":        "Clan Battle",
	"brawl":       "Brawl",
	"training":    "Training Room",
	"scenario":    "Operation",
}

func CleanMapName(rawMap string) string {
	cleaned := strings.TrimPrefix(rawMap, "spaces/")
	if name, ok := mapDictionary["spaces/"+cleaned]; ok {
		return name
	}
	if name, ok := mapDictionary[rawMap]; ok {
		return name
	}
	// Fallback cleanup if not in explicit dictionary
	parts := strings.Split(cleaned, "_")
	if len(parts) > 0 {
		return strings.Title(parts[len(parts)-1])
	}
	return rawMap
}

func CleanBattleType(rawType string) string {
	low := strings.ToLower(rawType)
	if name, ok := battleTypeMap[low]; ok {
		return name
	}
	if rawType == "" {
		return "Battle"
	}
	return strings.Title(rawType)
}

// CleanVehicleName extracts a clean, human-readable ship name from an in-game identifier.
// e.g. "PHSC019-Menno-Van-Coehoorn" -> "Menno Van Coehoorn"
// e.g. "PWSB108_Musashi_1942" -> "Musashi"
func CleanVehicleName(rawVehicle string) string {
	if rawVehicle == "" {
		return ""
	}

	// Normalize separator
	s := strings.ReplaceAll(rawVehicle, "_", "-")
	parts := strings.Split(s, "-")
	if len(parts) > 1 {
		// Drop internal ID prefix (e.g. PHSC019, PWSB108)
		nameParts := parts[1:]
		// Strip trailing 4-digit build year if present (e.g. 1942, 1943)
		if len(nameParts) > 1 {
			last := nameParts[len(nameParts)-1]
			if len(last) == 4 {
				if _, err := strconv.Atoi(last); err == nil {
					nameParts = nameParts[:len(nameParts)-1]
				}
			}
		}
		return strings.Join(nameParts, " ")
	}
	return rawVehicle
}

func ParseArenaInfo(filePath string) (*ArenaInfo, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading arena info %s: %w", filePath, err)
	}

	var raw RawArenaInfo
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("decoding arena info json: %w", err)
	}

	info := &ArenaInfo{
		MapName:       CleanMapName(raw.MapDisplayName),
		BattleType:    CleanBattleType(raw.MatchGroup),
		PlayerVehicle: raw.PlayerVehicle,
	}

	// Parse date time format "DD.MM.YYYY HH:MM:SS"
	// Example: "20.10.2026 14:30:00"
	t, err := time.ParseInLocation("02.01.2006 15:04:05", raw.DateTime, time.Local)
	if err == nil {
		info.StartTime = t
	} else {
		info.StartTime = time.Now()
	}

	// Find player ship ID
	for _, v := range raw.Vehicles {
		if v.Relation == 0 || (raw.PlayerName != "" && v.Name == raw.PlayerName) {
			info.PlayerShipID = v.ShipID
			break
		}
	}

	return info, nil
}
