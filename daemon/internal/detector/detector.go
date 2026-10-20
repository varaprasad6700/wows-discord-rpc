package detector

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// FindLatestBuild locates the highest numbered build directory inside <wowsPath>/bin
func FindLatestBuild(wowsPath string) (string, error) {
	binDir := filepath.Join(wowsPath, "bin")
	entries, err := os.ReadDir(binDir)
	if err != nil {
		return "", fmt.Errorf("reading bin directory %s: %w", binDir, err)
	}

	maxBuild := -1
	var latestDir string

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		buildNum, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue // Skip non-numeric folders
		}
		if buildNum > maxBuild {
			maxBuild = buildNum
			latestDir = filepath.Join(binDir, entry.Name())
		}
	}

	if maxBuild == -1 || latestDir == "" {
		return "", fmt.Errorf("no numerical build directories found in %s", binDir)
	}

	return latestDir, nil
}

// EnsureModDeployed ensures that PnFModsLoader.py and TelemetryExporter/Main.py
// are deployed into <buildPath>/res_mods/
func EnsureModDeployed(buildPath, modSourceDir string) error {
	resModsDir := filepath.Join(buildPath, "res_mods")
	telemetryDir := filepath.Join(resModsDir, "PnFMods", "TelemetryExporter")

	if err := os.MkdirAll(telemetryDir, 0755); err != nil {
		return fmt.Errorf("creating mod directories in %s: %w", resModsDir, err)
	}

	// 1. Deploy PnFModsLoader.py
	srcLoader := filepath.Join(modSourceDir, "PnFModsLoader.py")
	dstLoader := filepath.Join(resModsDir, "PnFModsLoader.py")
	if err := syncFile(srcLoader, dstLoader); err != nil {
		return fmt.Errorf("syncing PnFModsLoader.py: %w", err)
	}

	// 2. Deploy TelemetryExporter/Main.py
	srcMain := filepath.Join(modSourceDir, "PnFMods", "TelemetryExporter", "Main.py")
	dstMain := filepath.Join(telemetryDir, "Main.py")
	if err := syncFile(srcMain, dstMain); err != nil {
		return fmt.Errorf("syncing TelemetryExporter/Main.py: %w", err)
	}

	return nil
}

// syncFile copies src to dst if dst does not exist or contents differ
func syncFile(src, dst string) error {
	srcData, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("reading source file %s: %w", src, err)
	}

	dstData, err := os.ReadFile(dst)
	if err == nil && bytes.Equal(srcData, dstData) {
		return nil // Already up to date
	}

	tmpDst := dst + ".tmp"
	if err := os.WriteFile(tmpDst, srcData, 0644); err != nil {
		return fmt.Errorf("writing temp destination %s: %w", tmpDst, err)
	}

	if err := os.Rename(tmpDst, dst); err != nil {
		// On windows rename might fail if dst exists, remove first
		_ = os.Remove(dst)
		if err := os.Rename(tmpDst, dst); err != nil {
			return fmt.Errorf("renaming temp to destination %s: %w", dst, err)
		}
	}

	return nil
}

// FindWoWSPath attempts to auto-discover World of Warships installation
func FindWoWSPath() (string, error) {
	// Standard candidate paths on Windows
	candidates := []string{
		`C:\Games\World_of_Warships`,
		`D:\Games\World_of_Warships`,
		`E:\Games\World_of_Warships`,
		`C:\Program Files (x86)\Steam\steamapps\common\World of Warships`,
		`D:\SteamLibrary\steamapps\common\World of Warships`,
		`E:\SteamLibrary\steamapps\common\World of Warships`,
		`C:\Games\World_of_Warships_EU`,
		`C:\Games\World_of_Warships_NA`,
	}

	for _, path := range candidates {
		if IsValidWoWSPath(path) {
			return path, nil
		}
	}

	return "", fmt.Errorf("World of Warships directory not found in standard paths; please specify wows.install_path in config.yaml")
}

// IsValidWoWSPath checks whether a folder looks like a WoWS installation
func IsValidWoWSPath(path string) bool {
	if path == "" {
		return false
	}
	binDir := filepath.Join(path, "bin")
	if info, err := os.Stat(binDir); err == nil && info.IsDir() {
		return true
	}
	return false
}
