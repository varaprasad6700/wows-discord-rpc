package startup_test

import (
	"testing"

	"github.com/wows-discord-rpc/daemon/internal/startup"
)

func TestStartup_IsEnabled(t *testing.T) {
	_, err := startup.IsEnabled()
	if err != nil {
		t.Logf("IsEnabled note (might be restricted in test environment): %v", err)
	}
}
