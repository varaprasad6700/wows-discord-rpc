package logging_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wows-discord-rpc/daemon/internal/logging"
)

func TestLoggerWritingAndRingBuffer(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "wows_log_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logFile := filepath.Join(tempDir, "test.log")
	if err := logging.InitFileLogger(logFile); err != nil {
		t.Fatalf("InitFileLogger failed: %v", err)
	}

	logging.SetFileLoggingEnabled(true)
	if !logging.IsFileLoggingEnabled() {
		t.Errorf("expected file logging to be enabled")
	}

	logging.Logf("Hello Test Line 1")
	logging.Logf("Hello Test Line 2")

	recent := logging.GetRecentLogs(10)
	if len(recent) < 2 {
		t.Fatalf("expected at least 2 lines in recent logs, got %d", len(recent))
	}
	if !strings.Contains(recent[len(recent)-1], "Hello Test Line 2") {
		t.Errorf("expected last line to contain 'Hello Test Line 2', got: %s", recent[len(recent)-1])
	}

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("reading log file failed: %v", err)
	}
	if !strings.Contains(string(data), "Hello Test Line 1") {
		t.Errorf("expected log file to contain 'Hello Test Line 1'")
	}
}
