package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wows-discord-rpc/daemon/internal/server"
)

type mockStateProvider struct {
	appliedURL string
}

func (m *mockStateProvider) GetStatus() server.StatusDTO {
	return server.StatusDTO{
		GameRunning:     true,
		DiscordIPCState: "Connected",
		Details:         "[X] Yamato • Random (North)",
		State:           "72k HP (74%) | 84k DMG",
	}
}

func (m *mockStateProvider) GetConfig() server.ConfigDTO {
	return server.ConfigDTO{
		InstallPath: `E:\WarGaming Games\World_of_Warships`,
		BackendURL:  "https://wows-meta.varaprasad.co.in",
		AutoStart:   true,
		FileLogging: true,
	}
}

func (m *mockStateProvider) SaveConfig(cfg server.ConfigDTO) error {
	return nil
}

func (m *mockStateProvider) CheckUpdate() (*server.UpdateDTO, error) {
	return &server.UpdateDTO{
		CurrentVersion: "v1.1.0",
		LatestVersion:  "v1.1.0",
		UpdateReady:    false,
	}, nil
}

func (m *mockStateProvider) ApplyUpdate(downloadURL string) error {
	m.appliedURL = downloadURL
	return nil
}

func TestServerEndpoints(t *testing.T) {
	mockState := &mockStateProvider{}
	srv := server.NewServer(0, mockState)

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// 1. Test GET /api/status
	res, err := http.Get(ts.URL + "/api/status")
	if err != nil {
		t.Fatalf("GET /api/status failed: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", res.StatusCode)
	}

	var status server.StatusDTO
	if err := json.NewDecoder(res.Body).Decode(&status); err != nil {
		t.Fatalf("decoding status json failed: %v", err)
	}
	if !status.GameRunning || status.DiscordIPCState != "Connected" {
		t.Errorf("unexpected status dto: %+v", status)
	}

	// 2. Test GET /api/config
	res, err = http.Get(ts.URL + "/api/config")
	if err != nil {
		t.Fatalf("GET /api/config failed: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", res.StatusCode)
	}

	// 3. Test POST /api/config
	cfgPayload := `{"install_path":"E:\\NewPath","backend_url":"https://meta.example.com","auto_start":true,"file_logging":true}`
	res, err = http.Post(ts.URL+"/api/config", "application/json", strings.NewReader(cfgPayload))
	if err != nil {
		t.Fatalf("POST /api/config failed: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", res.StatusCode)
	}

	// 4. Test POST /api/update/apply
	applyPayload := `{"download_url":"https://example.com/new.exe"}`
	res, err = http.Post(ts.URL+"/api/update/apply", "application/json", strings.NewReader(applyPayload))
	if err != nil {
		t.Fatalf("POST /api/update/apply failed: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", res.StatusCode)
	}
	if mockState.appliedURL != "https://example.com/new.exe" {
		t.Errorf("expected appliedURL to be recorded, got: %s", mockState.appliedURL)
	}
}
