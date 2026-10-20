package updater_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wows-discord-rpc/daemon/internal/updater"
)

func TestIsNewerVersion(t *testing.T) {
	cases := []struct {
		current  string
		latest   string
		expected bool
	}{
		{"v1.1.0", "v1.1.1", true},
		{"v1.1.0", "v1.2.0", true},
		{"v1.1.0", "v2.0.0", true},
		{"v1.1.1", "v1.1.0", false},
		{"v1.1.0", "v1.1.0", false},
		{"dev", "v1.1.0", true},
	}

	for _, tc := range cases {
		got := updater.IsNewerVersion(tc.current, tc.latest)
		if got != tc.expected {
			t.Errorf("IsNewerVersion(%s, %s): expected %v, got %v", tc.current, tc.latest, tc.expected, got)
		}
	}
}

func TestCheckForUpdate_SelectsExeOverZip(t *testing.T) {
	fakeRelease := map[string]any{
		"tag_name": "v1.2.0",
		"body":     "Added Option A UI and self updater",
		"assets": []map[string]any{
			{
				"name":                 "wows-discord-rpc-windows-x64.zip",
				"browser_download_url": "https://example.com/archive.zip",
			},
			{
				"name":                 "wows-presence-daemon.exe",
				"browser_download_url": "https://example.com/wows-presence-daemon.exe",
			},
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(fakeRelease)
	}))
	defer ts.Close()

	u := updater.NewUpdaterWithURL("v1.1.0", ts.URL)
	info, err := u.Check()
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if !info.UpdateReady {
		t.Errorf("expected update to be ready")
	}
	if info.DownloadURL != "https://example.com/wows-presence-daemon.exe" {
		t.Errorf("expected download url to be the .exe, got %s", info.DownloadURL)
	}
}
