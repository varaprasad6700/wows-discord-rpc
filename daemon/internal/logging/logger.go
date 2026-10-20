package logging

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	MaxLogFileSize = 5 * 1024 * 1024 // 5MB
	MaxRecentLines = 100
)

type Logger struct {
	mu          sync.RWMutex
	filePath    string
	fileEnabled bool
	recentLines []string
}

var defaultLogger = &Logger{
	fileEnabled: true,
	recentLines: make([]string, 0, MaxRecentLines),
}

// DefaultLogPath returns %APPDATA%\wows-rpc\daemon.log on Windows
func DefaultLogPath() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".wows-rpc", "daemon.log")
	}
	return filepath.Join(appData, "wows-rpc", "daemon.log")
}

// InitFileLogger sets the active log file path
func InitFileLogger(path string) error {
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()

	defaultLogger.filePath = path
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating log directory %s: %w", dir, err)
	}
	return nil
}

func SetFileLoggingEnabled(enabled bool) {
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()
	defaultLogger.fileEnabled = enabled
}

func IsFileLoggingEnabled() bool {
	defaultLogger.mu.RLock()
	defer defaultLogger.mu.RUnlock()
	return defaultLogger.fileEnabled
}

func GetLogFilePath() string {
	defaultLogger.mu.RLock()
	defer defaultLogger.mu.RUnlock()
	if defaultLogger.filePath == "" {
		return DefaultLogPath()
	}
	return defaultLogger.filePath
}

func GetRecentLogs(limit int) []string {
	defaultLogger.mu.RLock()
	defer defaultLogger.mu.RUnlock()

	if limit <= 0 || limit > len(defaultLogger.recentLines) {
		limit = len(defaultLogger.recentLines)
	}

	start := len(defaultLogger.recentLines) - limit
	result := make([]string, limit)
	copy(result, defaultLogger.recentLines[start:])
	return result
}

// Logf writes a formatted log entry to stdout, recent buffer, and optionally file
func Logf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	entry := fmt.Sprintf("[%s] %s", timestamp, msg)

	// Always print to standard logger
	log.Println(msg)

	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()

	// Append to in-memory recent lines
	if len(defaultLogger.recentLines) >= MaxRecentLines {
		defaultLogger.recentLines = defaultLogger.recentLines[1:]
	}
	defaultLogger.recentLines = append(defaultLogger.recentLines, entry)

	// Append to file if enabled
	if defaultLogger.fileEnabled && defaultLogger.filePath != "" {
		// Check size for rotation
		if stat, err := os.Stat(defaultLogger.filePath); err == nil && stat.Size() > MaxLogFileSize {
			oldPath := defaultLogger.filePath + ".old"
			_ = os.Remove(oldPath)
			_ = os.Rename(defaultLogger.filePath, oldPath)
		}

		f, err := os.OpenFile(defaultLogger.filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil {
			_, _ = f.WriteString(entry + "\n")
			_ = f.Close()
		}
	}
}
