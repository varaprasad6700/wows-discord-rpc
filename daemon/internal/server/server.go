package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/wows-discord-rpc/daemon/internal/logging"
	"github.com/wows-discord-rpc/daemon/internal/startup"
)

//go:embed all:ui
var uiFS embed.FS

type StatusDTO struct {
	GameRunning     bool   `json:"game_running"`
	DiscordIPCState string `json:"discord_state"`
	Details         string `json:"details"`
	State           string `json:"state"`
}

type ConfigDTO struct {
	InstallPath string `json:"install_path"`
	BackendURL  string `json:"backend_url"`
	AutoStart   bool   `json:"auto_start"`
	FileLogging bool   `json:"file_logging"`
}

type UpdateDTO struct {
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	ReleaseNotes   string `json:"release_notes"`
	DownloadURL    string `json:"download_url"`
	UpdateReady    bool   `json:"update_ready"`
}

type StateProvider interface {
	GetStatus() StatusDTO
	GetConfig() ConfigDTO
	SaveConfig(cfg ConfigDTO) error
	CheckUpdate() (*UpdateDTO, error)
}

type Server struct {
	port          int
	listener      net.Listener
	stateProvider StateProvider
	mux           *http.ServeMux
}

func NewServer(port int, stateProvider StateProvider) *Server {
	s := &Server{
		port:          port,
		stateProvider: stateProvider,
		mux:           http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) routes() {
	// 1. Static UI files
	uiSub, err := fs.Sub(uiFS, "ui")
	if err == nil {
		s.mux.Handle("/", http.FileServer(http.FS(uiSub)))
	}

	// 2. Status
	s.mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		status := s.stateProvider.GetStatus()
		_ = json.NewEncoder(w).Encode(status)
	})

	// 3. Config
	s.mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			var req ConfigDTO
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := s.stateProvider.SaveConfig(req); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			_ = startup.SetEnabled(req.AutoStart)
			logging.SetFileLoggingEnabled(req.FileLogging)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "saved"})
			return
		}

		cfg := s.stateProvider.GetConfig()
		autoStart, _ := startup.IsEnabled()
		cfg.AutoStart = autoStart
		cfg.FileLogging = logging.IsFileLoggingEnabled()
		_ = json.NewEncoder(w).Encode(cfg)
	})

	// 4. Logs
	s.mux.HandleFunc("/api/logs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		lines := logging.GetRecentLogs(50)
		_ = json.NewEncoder(w).Encode(map[string]any{"lines": lines})
	})

	s.mux.HandleFunc("/api/logs/toggle", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			logging.SetFileLoggingEnabled(req.Enabled)
		}
		w.WriteHeader(http.StatusOK)
	})

	s.mux.HandleFunc("/api/logs/open", func(w http.ResponseWriter, r *http.Request) {
		openType := r.URL.Query().Get("type")
		logPath := logging.GetLogFilePath()
		if openType == "folder" {
			dir := filepath.Dir(logPath)
			if runtime.GOOS == "windows" {
				_ = exec.Command("explorer.exe", dir).Start()
			}
		} else {
			if _, err := os.Stat(logPath); os.IsNotExist(err) {
				_ = os.WriteFile(logPath, []byte(""), 0644)
			}
			if runtime.GOOS == "windows" {
				_ = exec.Command("notepad.exe", logPath).Start()
			}
		}
		w.WriteHeader(http.StatusOK)
	})

	// 5. Browse Folder Dialog Bridge
	s.mux.HandleFunc("/api/browse-folder", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if runtime.GOOS != "windows" {
			_ = json.NewEncoder(w).Encode(map[string]string{"path": ""})
			return
		}

		cmdStr := `(New-Object -ComObject Shell.Application).BrowseForFolder(0, 'Select World of Warships Folder', 0).Self.Path`
		out, err := exec.Command("powershell", "-NoProfile", "-Command", cmdStr).Output()
		if err != nil {
			_ = json.NewEncoder(w).Encode(map[string]string{"path": ""})
			return
		}
		chosen := strings.TrimSpace(string(out))
		_ = json.NewEncoder(w).Encode(map[string]string{"path": chosen})
	})

	// 6. Updates
	s.mux.HandleFunc("/api/update/check", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		up, err := s.stateProvider.CheckUpdate()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(up)
	})
}

func (s *Server) Start() (int, error) {
	addr := fmt.Sprintf("127.0.0.1:%d", s.port)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return 0, fmt.Errorf("starting loopback server: %w", err)
	}
	s.listener = l
	s.port = l.Addr().(*net.TCPAddr).Port

	go func() {
		_ = http.Serve(l, s.mux)
	}()

	return s.port, nil
}

func (s *Server) Port() int {
	return s.port
}

func (s *Server) Close() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}
