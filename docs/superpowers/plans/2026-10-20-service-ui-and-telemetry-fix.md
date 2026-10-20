# WoWS Discord Presence Background Controller, UI & Telemetry Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a windowless Windows background system tray application with an Option A tabbed controller UI, self-updater, and embedded telemetry mod, while fixing in-game Python combat telemetry to accurately report live HP, damage, and kills in Discord.

**Architecture:** The Go daemon will embed the Python mod tree via `//go:embed` and deploy it automatically to the detected WoWS build directory. Built with `-H=windowsgui`, it runs without a console window, registers in Windows User Startup (`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`), and manages a system tray icon. The daemon hosts an internal loopback server serving a sleek Option A tabbed UI in Edge app mode (`msedge.exe --app`), providing live status, settings with native folder picker, diagnostic file logging toggle, and GitHub self-updates.

**Tech Stack:** Go 1.22+ (native Windows binary, `//go:embed`, Win32 system tray / Edge app mode), Python 2.7/3 BigWorld sandbox, Discord IPC named pipes, GitHub Releases API, HTML/CSS/JS.

**Spec:** [`docs/superpowers/specs/2026-10-20-service-ui-and-telemetry-fix-design.md`](file:///C:/Users/pothu/android-projects/wows-discord-rpc/docs/superpowers/specs/2026-10-20-service-ui-and-telemetry-fix-design.md)

## Global Constraints

- Never use `import json`, `import os`, `globals()`, or `locals()` in in-game Python scripts (`Main.py`).
- Windows-first native binary: zero required external dependencies (uses native Windows APIs and built-in Edge app mode).
- Default Discord Client ID: `1562101043670556782`, embedded at compile time via `-ldflags`.
- Default Metadata Cache Backend: `https://wows-meta.varaprasad.co.in`.
- Config file path: `%APPDATA%\wows-rpc\config.yaml`.
- Log file path: `%APPDATA%\wows-rpc\daemon.log` (capped at 5MB with 1 backup).

## Review Focus

1. **Restricted BigWorld Python Environment**: Game engine throws unhandled `NameError` if `globals()` or forbidden standard library modules are accessed.
2. **Missing/Moved Game Directory**: User changes WoWS installation directory or deletes folder; daemon and UI must handle invalid paths gracefully without crashing.
3. **Locked Executable During Update**: Windows locks running `.exe` files from being overwritten; updater must rename the running binary to `.old` before writing the replacement.
4. **Browser Sandboxing on File Picker**: Modern browsers in `--app` mode sanitize absolute file paths; directory browsing must be proxied via loopback endpoint to a native Windows folder picker dialog.
5. **No Terminal / Background Startup**: App launched on Windows boot must not display any console prompt or flash a command window.

---

### Task 1: Fix Python Telemetry Mod Sandbox Compatibility

**Files:**
- Modify: `mod/PnFMods/TelemetryExporter/Main.py`
- Modify: `mod/tests/test_exporter.py`

**Interfaces:**
- Produces: `wows_live_state.json` containing `{"in_battle": bool, "ship_id": int, "health": int, "max_health": int, "damage_dealt": int, "frags": int, "allies_alive": int, "enemies_alive": int, "is_alive": bool}` without relying on `globals()` or forbidden modules.

- [ ] **Step 1: Write unit tests in `mod/tests/test_exporter.py` verifying sandbox compatibility**

```python
def test_no_forbidden_globals_or_builtins():
    with open("mod/PnFMods/TelemetryExporter/Main.py", "r") as f:
        content = f.read()
    assert "globals()" not in content
    assert "locals()" not in content
    assert "import json" not in content
    assert "import os" not in content
```

- [ ] **Step 2: Run test to verify it fails**

Run: `python -m unittest discover -s mod/tests -p "test_exporter.py"`
Expected: FAIL (assertion error: `globals()` found in `Main.py`).

- [ ] **Step 3: Update `mod/PnFMods/TelemetryExporter/Main.py`**

Remove `globals().get("events")` and `globals().get("callbacks")`. Directly reference global `events` and `callbacks` injected by WoWS ModsAPI within a safe `try / except NameError`. Use `xml.sax.saxutils.os` for filesystem operations if available, or write to standard relative files. Also copy the updated mod directly to `E:\WarGaming Games\World_of_Warships\bin\13357625\res_mods\PnFMods\TelemetryExporter\Main.py`.

- [ ] **Step 4: Run test to verify it passes**

Run: `python -m unittest discover -s mod/tests -p "test_exporter.py"`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mod/PnFMods/TelemetryExporter/Main.py mod/tests/test_exporter.py
git commit -m "fix(mod): eliminate globals() and forbidden imports for WoWS sandbox compatibility"
```

---

### Task 2: Embed Mod into Go Binary (`//go:embed`) & Update Engine State Discovery

**Files:**
- Create: `daemon/internal/modembed/modembed.go`
- Modify: `daemon/internal/detector/mod.go`
- Modify: `daemon/cmd/daemon/engine.go`
- Modify: `daemon/cmd/daemon/main.go`
- Test: `daemon/internal/detector/mod_test.go`
- Test: `daemon/cmd/daemon/engine_test.go`

**Interfaces:**
- Consumes: `modembed.ModFS` (embedded filesystem containing `mod/`).
- Produces: `detector.DeployEmbeddedMod(buildPath string) error` which unpacks the embedded mod directly into `res_mods` without requiring an external `mod/` directory on disk.
- Produces: `engine.findLiveStateFile(wowsPath string) string` checking `filepath.Join(wowsPath, "wows_live_state.json")`, `bin/<build>/wows_live_state.json`, and `bin/<build>/bin64/wows_live_state.json`.

- [ ] **Step 1: Write test for embedded mod deployment and multi-path live state lookup**

```go
func TestFindLiveStateFile(t *testing.T) {
    // Tests discovery across root, bin/<build>, and bin/<build>/bin64
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `cd daemon && go test -v ./...`
Expected: FAIL (types/functions not yet defined).

- [ ] **Step 3: Implement `daemon/internal/modembed/modembed.go` and update `mod.go` & `engine.go`**

Implement `embed.FS` in `modembed` package rooting at `../../mod`. In `detector/mod.go`, add `DeployEmbeddedMod(buildPath string) error`. Update `engine.go` to search multiple possible locations for `wows_live_state.json`.

- [ ] **Step 4: Run tests to verify pass**

Run: `cd daemon && go test -v ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add daemon/internal/modembed daemon/internal/detector daemon/cmd/daemon
git commit -m "feat(daemon): embed mod assets into binary and expand live state path resolution"
```

---

### Task 3: Windows Startup Registry & Diagnostic File Logging

**Files:**
- Create: `daemon/internal/startup/startup_windows.go`
- Create: `daemon/internal/startup/startup_other.go`
- Create: `daemon/internal/logging/logger.go`
- Test: `daemon/internal/logging/logger_test.go`

**Interfaces:**
- Produces: `startup.IsEnabled() (bool, error)`
- Produces: `startup.SetEnabled(enabled bool) error`
- Produces: `logging.InitFileLogger(logPath string) error`
- Produces: `logging.SetFileLoggingEnabled(enabled bool)`
- Produces: `logging.GetRecentLogs(limit int) []string`
- Produces: `logging.GetLogFilePath() string`

- [ ] **Step 1: Write unit tests for logger rotation and recent lines buffer**

```go
func TestLoggerRotationAndRecentBuffer(t *testing.T) {
    // Tests file creation, writing, 5MB rotation, and circular buffer
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `cd daemon && go test -v ./internal/logging`
Expected: FAIL.

- [ ] **Step 3: Implement `startup` and `logging` packages**

Implement Windows registry read/write for `HKCU\Software\Microsoft\Windows\CurrentVersion\Run\WoWSPresence` using `golang.org/x/sys/windows/registry`. Implement file logger writing to `%APPDATA%\wows-rpc\daemon.log` with a 50-line in-memory ring buffer for live UI diagnostics and file rotation at 5MB.

- [ ] **Step 4: Run test to verify pass**

Run: `cd daemon && go test -v ./internal/logging`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add daemon/internal/startup daemon/internal/logging
git commit -m "feat(daemon): add windows startup registry manager and diagnostic file logger"
```

---

### Task 4: Loopback HTTP Server, UI Web Assets & Native Folder Picker Bridge

**Files:**
- Create: `daemon/internal/server/server.go`
- Create: `daemon/internal/server/ui/index.html`
- Create: `daemon/internal/server/ui/styles.css`
- Create: `daemon/internal/server/ui/app.js`
- Test: `daemon/internal/server/server_test.go`

**Interfaces:**
- Consumes: `logging.GetRecentLogs()`, `startup.IsEnabled()`, `Engine.GetStatus()`
- Produces: `server.NewServer(port int, stateProvider StateProvider, configHandler ConfigHandler, updateHandler UpdateHandler) *Server`
- Exposes:
  - `GET /` (serves embedded Option A Tabbed UI)
  - `GET /api/status` (game running, presence preview, Discord IPC state)
  - `GET /api/config` & `POST /api/config` (WoWS install path, server URL, startup toggle)
  - `GET /api/logs` & `POST /api/logs/toggle`
  - `POST /api/logs/open` (launches `daemon.log` in Notepad or folder in Explorer)
  - `POST /api/browse-folder` (opens Windows native directory picker and returns chosen path)

- [ ] **Step 1: Write unit test for loopback API routes**

```go
func TestServerAPIRoutes(t *testing.T) {
    // Tests /api/status, /api/config GET/POST, /api/logs
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `cd daemon && go test -v ./internal/server`
Expected: FAIL.

- [ ] **Step 3: Implement Option A Tabbed UI and HTTP server with folder picker bridge**

Embed `ui/` directory with `//go:embed ui/*`. Implement the four tabs (Status, Settings, Logs & Diagnostics, Updates). For `POST /api/browse-folder`, invoke PowerShell `(New-Object -ComObject Shell.Application).BrowseForFolder(0, 'Select World of Warships Folder', 0).Self.Path` or Win32 COM, returning the chosen path.

- [ ] **Step 4: Run test to verify pass**

Run: `cd daemon && go test -v ./internal/server`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add daemon/internal/server
git commit -m "feat(daemon): implement loopback server, tabbed ui, and native folder picker bridge"
```

---

### Task 5: In-App Self-Updater & GitHub Release Workflow

**Files:**
- Create: `daemon/internal/updater/updater.go`
- Modify: `.github/workflows/release.yml`
- Test: `daemon/internal/updater/updater_test.go`

**Interfaces:**
- Produces: `updater.CheckForUpdate(currentVersion string) (*UpdateInfo, error)`
- Produces: `updater.ApplyUpdate(downloadURL string) error`

- [ ] **Step 1: Write unit test for version comparison and GitHub Release parsing**

```go
func TestVersionComparison(t *testing.T) {
    // Tests v1.1.0 vs v1.1.1 semver comparisons
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `cd daemon && go test -v ./internal/updater`
Expected: FAIL.

- [ ] **Step 3: Implement updater and update GitHub Actions release workflow**

In `updater.go`, query GitHub Releases API for latest release, locate the `wows-presence-daemon.exe` asset, download to a temporary file, rename active binary to `.old`, move new binary into place, and trigger relaunch. Update `.github/workflows/release.yml` to attach standalone `wows-presence-daemon.exe` alongside `.zip`.

- [ ] **Step 4: Run test to verify pass**

Run: `cd daemon && go test -v ./internal/updater`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add daemon/internal/updater .github/workflows/release.yml
git commit -m "feat(updater): implement in-place executable self-update and standalone release asset"
```

---

### Task 6: System Tray Integration & Headless GUI Launcher

**Files:**
- Create: `daemon/internal/tray/tray_windows.go`
- Create: `daemon/internal/tray/tray_other.go`
- Modify: `daemon/internal/config/config.go`
- Modify: `daemon/cmd/daemon/main.go`
- Modify: `daemon/cmd/daemon/engine.go`

**Interfaces:**
- Produces: `tray.Run(onOpenUI func(), onOpenLogs func(), onCheckUpdates func(), onExit func())`
- Produces: `tray.UpdateStatus(text string)`
- Supports: `-ldflags "-H=windowsgui -X main.version=... -X main.defaultDiscordID=..."`

- [ ] **Step 1: Write test for config file persistence and hot-reload in `%APPDATA%\wows-rpc\config.yaml`**

```go
func TestConfigPersistenceAndHotReload(t *testing.T) {
    // Tests saving to AppData and atomic in-memory reload
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `cd daemon && go test -v ./internal/config`
Expected: FAIL.

- [ ] **Step 3: Implement Win32 System Tray & Main Controller Launcher**

Implement system tray icon using Windows Shell_NotifyIcon API (pure Go, zero CGO). When user clicks "Open Controller UI", spawn Edge in app mode: `msedge.exe --app=http://127.0.0.1:<port> --window-size=540,680`. Thread-safe config update in `Engine`.

- [ ] **Step 4: Run test to verify pass**

Run: `cd daemon && go test -v ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add daemon/internal/tray daemon/internal/config daemon/cmd/daemon
git commit -m "feat(tray): implement native system tray, edge app mode launcher, and hot-reload"
```

---

### Task 7: End-to-End Verification & Documentation Update

**Files:**
- Modify: `README.md`
- Modify: `AGENTS.md`

- [ ] **Step 1: Execute full test suite**

Run:
```powershell
python -m unittest discover -s mod/tests
cd daemon
go test -v ./...
```
Expected: All tests PASS.

- [ ] **Step 2: Compile native GUI binary**

Run:
```powershell
go build -ldflags "-H=windowsgui -X main.version=v1.1.0 -X main.defaultDiscordID=1562101043670556782" -o ../wows-presence-daemon.exe ./cmd/daemon
```
Expected: Successfully generated `wows-presence-daemon.exe` with zero console window.

- [ ] **Step 3: Update README.md and AGENTS.md**

Document the new system tray, controller UI, self-updater, `%APPDATA%\wows-rpc` configuration/logging paths, and Windows Startup settings.

- [ ] **Step 4: Commit and Push**

```bash
git add README.md AGENTS.md
git commit -m "docs: document system tray, controller ui, file logging, and self-updater"
git push origin master
```
