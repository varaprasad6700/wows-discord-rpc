package process_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wows-discord-rpc/daemon/internal/process"
)

func TestIsProcessRunning(t *testing.T) {
	// A non-existent process name must return false
	running := process.IsProcessRunning("definitely_not_a_running_process_xyz_123.exe")
	if running {
		t.Error("expected non-existent process to return false")
	}

	// Current executable name must return true
	currentExe, err := os.Executable()
	if err == nil {
		exeName := filepath.Base(currentExe)
		if !process.IsProcessRunning(exeName) {
			t.Errorf("expected current process '%s' to return true", exeName)
		}
	}
}
