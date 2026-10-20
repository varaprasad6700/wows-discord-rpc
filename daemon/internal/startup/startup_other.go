//go:build !windows

package startup

// IsEnabled is a stub for non-windows platforms
func IsEnabled() (bool, error) {
	return false, nil
}

// SetEnabled is a stub for non-windows platforms
func SetEnabled(enabled bool) error {
	return nil
}
