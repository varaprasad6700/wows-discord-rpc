//go:build windows

package startup

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

const (
	runKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	appName    = "WoWSPresence"
)

// IsEnabled checks if WoWSPresence is registered in HKCU Run
func IsEnabled() (bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false, fmt.Errorf("opening registry key: %w", err)
	}
	defer k.Close()

	val, _, err := k.GetStringValue(appName)
	if err == registry.ErrNotExist {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("reading registry value: %w", err)
	}

	return val != "", nil
}

// SetEnabled adds or removes WoWSPresence in HKCU Run
func SetEnabled(enabled bool) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("opening registry key for writing: %w", err)
	}
	defer k.Close()

	if enabled {
		exePath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("retrieving executable path: %w", err)
		}
		absPath, err := filepath.Abs(exePath)
		if err != nil {
			return fmt.Errorf("resolving absolute executable path: %w", err)
		}
		val := fmt.Sprintf("\"%s\"", absPath)
		if err := k.SetStringValue(appName, val); err != nil {
			return fmt.Errorf("setting registry value: %w", err)
		}
	} else {
		err := k.DeleteValue(appName)
		if err != nil && err != registry.ErrNotExist {
			return fmt.Errorf("deleting registry value: %w", err)
		}
	}

	return nil
}
