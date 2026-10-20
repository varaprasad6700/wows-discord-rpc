package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type ShipMetadata struct {
	ShipID    int64  `json:"shipId"`
	Name      string `json:"name"`
	Tier      int    `json:"tier"`
	RomanTier string `json:"romanTier"`
	ShipClass string `json:"shipClass"`
	Nation    string `json:"nation"`
	IconURL   string `json:"iconUrl"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
	cache      map[int64]*ShipMetadata
	mu         sync.RWMutex
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 4 * time.Second,
		},
		cache: make(map[int64]*ShipMetadata),
	}
}

func (c *Client) GetShip(shipID int64) (*ShipMetadata, error) {
	if shipID <= 0 {
		return &ShipMetadata{Name: "Unknown Ship"}, nil
	}

	c.mu.RLock()
	if cached, ok := c.cache[shipID]; ok {
		c.mu.RUnlock()
		return cached, nil
	}
	c.mu.RUnlock()

	url := fmt.Sprintf("%s/api/v1/ships/%d", c.baseURL, shipID)
	resp, err := c.httpClient.Get(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		// Fallback for cache misses or offline backend
		fallback := &ShipMetadata{
			ShipID:    shipID,
			Name:      fmt.Sprintf("Ship #%d", shipID),
			RomanTier: "?",
			ShipClass: "Warship",
		}
		c.mu.Lock()
		c.cache[shipID] = fallback
		c.mu.Unlock()
		return fallback, nil
	}
	defer resp.Body.Close()

	var metadata ShipMetadata
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return nil, fmt.Errorf("decoding ship metadata: %w", err)
	}

	c.mu.Lock()
	c.cache[shipID] = &metadata
	c.mu.Unlock()

	return &metadata, nil
}
