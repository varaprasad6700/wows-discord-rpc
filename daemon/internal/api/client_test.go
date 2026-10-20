package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/wows-discord-rpc/daemon/internal/api"
)

func TestClient_GetShip_Cached(t *testing.T) {
	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)

		if r.URL.Path == "/api/v1/ships/4181230000" {
			ship := api.ShipMetadata{
				ShipID:    4181230000,
				Name:      "Yamato",
				Tier:      10,
				RomanTier: "X",
				ShipClass: "Battleship",
				Nation:    "japan",
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(ship)
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	client := api.NewClient(server.URL)

	// First request: hits HTTP server
	ship1, err := client.GetShip(4181230000)
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	if ship1.Name != "Yamato" {
		t.Errorf("expected Yamato, got %s", ship1.Name)
	}

	// Second request: hits in-memory cache
	ship2, err := client.GetShip(4181230000)
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	if ship2.Name != "Yamato" {
		t.Errorf("expected Yamato, got %s", ship2.Name)
	}

	count := atomic.LoadInt32(&requestCount)
	if count != 1 {
		t.Errorf("expected exactly 1 HTTP request due to caching, got %d", count)
	}
}

func TestClient_GetShip_NotFoundFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := api.NewClient(server.URL)
	ship, err := client.GetShip(9999999999)
	if err != nil {
		t.Fatalf("expected fallback metadata instead of hard error, got: %v", err)
	}

	if ship.ShipID != 9999999999 {
		t.Errorf("expected ship ID 9999999999, got %d", ship.ShipID)
	}
}
