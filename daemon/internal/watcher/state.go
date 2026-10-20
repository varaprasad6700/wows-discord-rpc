package watcher

import (
	"encoding/json"
	"fmt"
	"os"
)

type LiveState struct {
	InBattle     bool  `json:"in_battle"`
	ShipID       int64 `json:"ship_id"`
	Health       int   `json:"health"`
	MaxHealth    int   `json:"max_health"`
	DamageDealt  int   `json:"damage_dealt"`
	Frags        int   `json:"frags"`
	AlliesAlive  int   `json:"allies_alive"`
	EnemiesAlive int   `json:"enemies_alive"`
	IsAlive      bool  `json:"is_alive"`
}

func (s *LiveState) HealthPercent() int {
	if s.MaxHealth <= 0 || s.Health <= 0 {
		return 0
	}
	pct := (float64(s.Health) / float64(s.MaxHealth)) * 100.0
	return int(pct + 0.5) // round to nearest integer
}

func ReadLiveState(filePath string) (*LiveState, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading live state file %s: %w", filePath, err)
	}

	var state LiveState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("decoding live state json: %w", err)
	}

	return &state, nil
}
