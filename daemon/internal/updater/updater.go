package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultGitHubRepo = "varaprasad6700/wows-discord-rpc"
)

type UpdateInfo struct {
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	ReleaseNotes   string `json:"release_notes"`
	DownloadURL    string `json:"download_url"`
	UpdateReady    bool   `json:"update_ready"`
}

type Updater struct {
	currentVersion string
	apiURL         string
	httpClient     *http.Client
}

func NewUpdater(currentVersion string) *Updater {
	return &Updater{
		currentVersion: currentVersion,
		apiURL:         fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", DefaultGitHubRepo),
		httpClient:     &http.Client{Timeout: 10 * time.Second},
	}
}

func NewUpdaterWithURL(currentVersion, url string) *Updater {
	return &Updater{
		currentVersion: currentVersion,
		apiURL:         url,
		httpClient:     &http.Client{Timeout: 10 * time.Second},
	}
}

func (u *Updater) Check() (*UpdateInfo, error) {
	req, err := http.NewRequest(http.MethodGet, u.apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "WoWS-Discord-RPC-Daemon")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("checking github release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
		Body    string `json:"body"`
		Assets  []struct {
			Name        string `json:"name"`
			DownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decoding release json: %w", err)
	}

	info := &UpdateInfo{
		CurrentVersion: u.currentVersion,
		LatestVersion:  release.TagName,
		ReleaseNotes:   release.Body,
		UpdateReady:    IsNewerVersion(u.currentVersion, release.TagName),
	}

	// Look for standalone executable or release package
	for _, asset := range release.Assets {
		if strings.EqualFold(asset.Name, "wows-presence-daemon.exe") {
			info.DownloadURL = asset.DownloadURL
			break
		}
	}
	if info.DownloadURL == "" && len(release.Assets) > 0 {
		info.DownloadURL = release.Assets[0].DownloadURL
	}

	return info, nil
}

// IsNewerVersion compares semantic version strings (e.g. v1.1.0 vs v1.2.0)
func IsNewerVersion(current, latest string) bool {
	if current == "dev" {
		return true
	}
	curParts := parseSemver(current)
	latParts := parseSemver(latest)

	for i := 0; i < 3; i++ {
		if latParts[i] > curParts[i] {
			return true
		}
		if latParts[i] < curParts[i] {
			return false
		}
	}
	return false
}

func parseSemver(v string) [3]int {
	var result [3]int
	clean := strings.TrimPrefix(strings.TrimSpace(v), "v")
	parts := strings.Split(clean, ".")
	for i := 0; i < len(parts) && i < 3; i++ {
		num, _ := strconv.Atoi(parts[i])
		result[i] = num
	}
	return result
}

// ApplyUpdate downloads the new executable, swaps with running binary, and restarts
func (u *Updater) Apply(downloadURL string) error {
	if downloadURL == "" {
		return fmt.Errorf("no download URL provided")
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolving executable path: %w", err)
	}
	absExe, err := filepath.Abs(exePath)
	if err != nil {
		return fmt.Errorf("resolving absolute path: %w", err)
	}

	// 1. Download new binary to temporary file
	resp, err := u.httpClient.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("downloading update: %w", err)
	}
	defer resp.Body.Close()

	tmpFile, err := os.CreateTemp(filepath.Dir(absExe), "daemon-update-*.exe")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
	}()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return fmt.Errorf("saving update payload: %w", err)
	}
	_ = tmpFile.Close()

	// 2. Rename running binary to .old (allowed on Windows)
	oldPath := absExe + ".old"
	_ = os.Remove(oldPath)
	if err := os.Rename(absExe, oldPath); err != nil {
		return fmt.Errorf("renaming running binary to .old: %w", err)
	}

	// 3. Move new binary into target position
	if err := os.Rename(tmpPath, absExe); err != nil {
		// Rollback
		_ = os.Rename(oldPath, absExe)
		return fmt.Errorf("moving new binary into place: %w", err)
	}

	// 4. Spawn new process detached
	cmd := exec.Command(absExe)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting updated process: %w", err)
	}

	// 5. Exit current process
	go func() {
		time.Sleep(500 * time.Millisecond)
		os.Exit(0)
	}()

	return nil
}
