package detector

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/wows-discord-rpc/daemon/internal/modembed"
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
// are deployed into <buildPath>/res_mods/ from an on-disk mod source directory.
func EnsureModDeployed(buildPath, modSourceDir string) error {
	resModsDir := filepath.Join(buildPath, "res_mods")
	telemetryDir := filepath.Join(resModsDir, "PnFMods", "TelemetryExporter")

	if err := os.MkdirAll(telemetryDir, 0755); err != nil {
		return fmt.Errorf("creating mod directories in %s: %w", resModsDir, err)
	}

	// 1. Deploy PnFModsLoader.py
	loaderSrc := filepath.Join(modSourceDir, "PnFModsLoader.py")
	loaderDst := filepath.Join(resModsDir, "PnFModsLoader.py")
	if err := copyIfDifferent(loaderSrc, loaderDst); err != nil {
		return fmt.Errorf("deploying PnFModsLoader.py: %w", err)
	}

	// 2. Deploy TelemetryExporter/Main.py
	mainSrc := filepath.Join(modSourceDir, "PnFMods", "TelemetryExporter", "Main.py")
	mainDst := filepath.Join(telemetryDir, "Main.py")
	if err := copyIfDifferent(mainSrc, mainDst); err != nil {
		return fmt.Errorf("deploying TelemetryExporter Main.py: %w", err)
	}

	return nil
}

// DeployEmbeddedMod extracts the embedded mod assets directly into <buildPath>/res_mods/
func DeployEmbeddedMod(buildPath string) error {
	resModsDir := filepath.Join(buildPath, "res_mods")
	telemetryDir := filepath.Join(resModsDir, "PnFMods", "TelemetryExporter")

	if err := os.MkdirAll(telemetryDir, 0755); err != nil {
		return fmt.Errorf("creating mod directories in %s: %w", resModsDir, err)
	}

	// 1. Deploy PnFModsLoader.py
	loaderData, err := modembed.ReadModFile("PnFModsLoader.py")
	if err != nil {
		return fmt.Errorf("reading embedded PnFModsLoader.py: %w", err)
	}
	loaderDst := filepath.Join(resModsDir, "PnFModsLoader.py")
	if err := writeIfDifferent(loaderData, loaderDst); err != nil {
		return fmt.Errorf("deploying embedded PnFModsLoader.py: %w", err)
	}

	// 2. Deploy TelemetryExporter/Main.py
	mainData, err := modembed.ReadModFile("PnFMods/TelemetryExporter/Main.py")
	if err != nil {
		return fmt.Errorf("reading embedded Main.py: %w", err)
	}
	mainDst := filepath.Join(telemetryDir, "Main.py")
	if err := writeIfDifferent(mainData, mainDst); err != nil {
		return fmt.Errorf("deploying embedded TelemetryExporter Main.py: %w", err)
	}

	return nil
}

func writeIfDifferent(srcData []byte, dst string) error {
	if dstData, err := os.ReadFile(dst); err == nil {
		if bytes.Equal(srcData, dstData) {
			return nil
		}
	}

	tmpDst := dst + ".tmp"
	if err := os.WriteFile(tmpDst, srcData, 0644); err != nil {
		return fmt.Errorf("writing temp destination %s: %w", tmpDst, err)
	}

	_ = os.Remove(dst)
	if err := os.Rename(tmpDst, dst); err != nil {
		return fmt.Errorf("renaming temp to destination %s: %w", dst, err)
	}

	return nil
}

func copyIfDifferent(src, dst string) error {
	srcData, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("reading source file %s: %w", src, err)
	}
	return writeIfDifferent(srcData, dst)
}

// FindWoWSPath attempts to auto-discover World of Warships installation
func FindWoWSPath() (string, error) {
	// 1. Check Wargaming Game Center preferences.xml
	programData := os.Getenv("ProgramData")
	if programData == "" {
		programData = `C:\ProgramData`
	}
	prefPath := filepath.Join(programData, "Wargaming.net", "GameCenter", "preferences.xml")
	if data, err := os.ReadFile(prefPath); err == nil {
		reWows := regexp.MustCompile(`(?i)<WOWS>(.*?)</WOWS>`)
		if matches := reWows.FindSubmatch(data); len(matches) > 1 {
			p := string(matches[1])
			if IsValidWoWSPath(p) {
				return p, nil
			}
		}
		reDir := regexp.MustCompile(`(?i)<working_dir>(.*?World_of_Warships.*?)</working_dir>`)
		if matches := reDir.FindSubmatch(data); len(matches) > 1 {
			p := string(matches[1])
			if IsValidWoWSPath(p) {
				return p, nil
			}
		}
	}

	// 2. Standard candidate paths on Windows
	candidates := []string{
		`E:\WarGaming Games\World_of_Warships`,
		`C:\WarGaming Games\World_of_Warships`,
		`D:\WarGaming Games\World_of_Warships`,
		`C:\Games\World_of_Warships`,
		`D:\Games\World_of_Warships`,
		`E:\Games\World_of_Warships`,
		`C:\Games\World_of_Warships_EU`,
		`C:\Games\World_of_Warships_NA`,
		`C:\Program Files (x86)\Steam\steamapps\common\World of Warships`,
		`D:\SteamLibrary\steamapps\common\World of Warships`,
		`E:\SteamLibrary\steamapps\common\World of Warships`,
	}

	for _, path := range candidates {
		if IsValidWoWSPath(path) {
			return path, nil
		}
	}

	return "", fmt.Errorf("world of warships installation directory not found in common locations")
}

// IsValidWoWSPath checks whether the given path contains WoWS binaries or structures
func IsValidWoWSPath(path string) bool {
	if path == "" {
		return false
	}
	if stat, err := os.Stat(path); err != nil || !stat.IsDir() {
		return false
	}

	binDir := filepath.Join(path, "bin")
	if stat, err := os.Stat(binDir); err == nil && stat.IsDir() {
		return true
	}

	for _, exe := range []string{"WorldOfWarships64.exe", "WorldOfWarships.exe", "wows.exe"} {
		if _, err := os.Stat(filepath.Join(path, exe)); err == nil {
			return true
		}
	}

	return false
}
