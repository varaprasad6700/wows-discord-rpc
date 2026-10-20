# Design Document: WoWS Discord Presence Background Controller, UI & Telemetry Fix

**Date:** 2026-10-20  
**Status:** Draft  
**Target:** Monorepo (`wows-discord-rpc`)

---

## 1. Intent & Context

The World of Warships Discord Rich Presence application enables players to display their live combat telemetry (health, percentage, damage dealt, frags, team status) directly on their Discord profile.

Currently:
1. The daemon must be launched manually from a terminal, keeping an unwanted console window open.
2. The in-game Python telemetry mod failed to load due to a `NameError: name 'globals' is not defined` inside the game's embedded sandbox, causing the daemon to permanently show fallback default values (`1k HP (100%) | 0k DMG | 0 Kills`).
3. Players have no visual interface to view status, configure custom paths, toggle diagnostic file logging, or check for application updates.

This project delivers a hands-off, set-and-forget Windows background experience with a native system tray icon, a compact tabbed controller window (Option A), an in-app updater, and a complete fix for the in-game Python combat telemetry script.

---

## 2. Goals, Non-Goals, and Anti-Goals

### Goals
- **Fix Live Combat Telemetry**: Ensure the Python telemetry script (`Main.py`) loads cleanly within WoWS's restricted BigWorld sandbox, hooks battle events, and exports accurate live HP, percentage, damage, and kill counts.
- **Zero-Touch Background Operation**: Run silently in the background on Windows startup without opening a terminal window, living as a responsive icon in the system notification tray.
- **Option A Tabbed Controller UI**: Provide a compact, dark-mode desktop window with four focused tabs:
  1. *Status*: Real-time game process detection, Discord IPC connection status, and current presence preview.
  2. *Settings*: Configurable WoWS install path (with directory picker), metadata server URL, and "Launch on Windows startup" toggle.
  3. *Diagnostics & Logs*: Checkbox to toggle file logging (`%APPDATA%\wows-rpc\daemon.log`), with quick buttons to open the log file or folder.
  4. *Updates*: One-click update check against GitHub Releases API and in-place executable update.
- **Embedded Client ID**: Embed the default Discord Client ID at compile time via Go linker flags (`-ldflags`), removing credential configuration from the user interface.

### Non-Goals
- **Session 0 Windows Service**: We will not implement a true Windows SCM Service. Windows Services run in isolated Session 0, which cannot connect to user-session named pipes (`\\.\pipe\discord-ipc-0`) without complex cross-session proxies. A User Startup / Task Scheduler background app delivers identical automated behavior without IPC barriers.
- **Third-Party Game Overlays**: No DirectX or in-game screen overlays. Rich Presence remains strictly within Discord.

### Anti-Goals
- **Heavy Framework Dependencies**: Avoid requiring huge runtimes (e.g. Electron or .NET 8 desktop runtimes). Keep the single-binary deployment lightweight and fast.
- **Blocking / Lagging Game Client**: The mod and daemon must never block game rendering or telemetry writes.

---

## 3. Architecture & Technical Design

### A. Python Telemetry Sandbox Fix
**Root Cause**: BigWorld's embedded Python sandbox explicitly removes `globals()`, `locals()`, and `eval` from `__builtins__`. When `Main.py` executed `globals().get("events")`, Python raised `NameError: name 'globals' is not defined`, aborting mod loading.

**Resolution**:
1. Remove all references to `globals()`, `locals()`, and forbidden imports (`json`, `os`).
2. Utilize the globals injected directly into the mod scope by `ModsAPI` / `PnFModsLoader.py`:
   - `events.onBattleStart(...)`
   - `events.onBattleEnd(...)`
   - `events.onBattleQuit(...)`
   - `events.onReceiveShellInfo(...)`
   - `utils.getModDir()`
   - `utils.jsonEncode(...)`
   - `utils.logInfo(...)`
3. Fall back gracefully to `xml.sax.saxutils.os` for filesystem operations if needed, or write to standard relative paths.
4. Export `wows_live_state.json` to both the game root and `bin/<build_id>/`, ensuring the Go daemon locates it under any game launch directory.

### B. Background Execution & System Tray
1. **Windowless Execution**:
   - Built with `-H=windowsgui` so no console prompt opens on launch.
2. **System Tray Integration**:
   - Uses native Win32 `Shell_NotifyIconW` or pure-Go `systray` to show a WoWS anchor/shield icon next to the Windows clock.
   - Context Menu:
     - `● Status: Active / Idle`
     - `🖥️ Open Controller`
     - `📝 View Log File`
     - `🔄 Check for Updates...`
     - `────────────────────`
     - `❌ Exit`
3. **Windows Startup Registration**:
   - Controlled via standard user registry key: `HKCU\Software\Microsoft\Windows\CurrentVersion\Run\WoWSPresence`.
   - Toggleable from the Settings tab.

### C. Controller UI (Option A: Tabbed Compact Window)
1. **Interface Engine**:
   - Zero-external-dependency local web app mode served by the daemon's internal lightweight loopback server on a dynamic port, launched via `msedge.exe --app=http://127.0.0.1:<port>` or embedded webview.
   - Native look and feel matching the approved Option A dark naval mockups.
2. **Tab Breakdown**:
   - **Status Tab**:
     - Game Process: `● WorldOfWarships64.exe (Running / Not Running)`
     - Discord IPC: `● Connected (\pipe\discord-ipc-0)`
     - Presence Preview Card: Roman tier, ship name, battle mode, map, HP %, damage, frags.
   - **Settings Tab**:
     - Game Installation Path input + "Browse" folder button.
     - Metadata Server URL (default: `https://wows-meta.varaprasad.co.in`).
     - "Launch on Windows Startup" checkbox.
     - "Save & Apply" button.
   - **Diagnostics & Logs Tab**:
     - "Enable File Logging" toggle.
     - Current log path: `%APPDATA%\wows-rpc\daemon.log`.
     - "Open Log File" (opens in Notepad) and "Open Logs Folder" (opens Explorer).
     - Live recent log excerpt view (last 50 lines).
   - **Updates Tab**:
     - Displays Current Version vs Latest GitHub Release.
     - Release notes snippet.
     - "Check for Updates" and "Update Now" buttons.

### D. Compile-Time Credentials
- Default Discord Client ID (`1562101043670556782`) and version (`v1.1.0`) injected via `-ldflags`:
  ```bash
  go build -ldflags "-H=windowsgui -X main.version=v1.1.0 -X main.defaultDiscordID=1562101043670556782" ...
  ```

---

## 4. Architectural & Implementation Decisions (Builder Review Settle-Outs)

Following a builder review of the codebase and workflow assets, the following architecture decisions have been finalized:

1. **Embedded Mod Packaging (`//go:embed`)**:
   - The entire `mod/` directory (`PnFModsLoader.py` and `PnFMods/TelemetryExporter/Main.py`) will be embedded directly into the Go daemon binary at compile time via Go's standard `embed.FS`.
   - **Benefit**: The daemon becomes completely self-contained as a single `.exe`. It does not rely on an external `mod/` folder on disk and can extract/deploy the mod directly into `res_mods` on any PC.
   - When the daemon updates, the telemetry mod updates simultaneously and is automatically redeployed to the WoWS installation.

2. **In-Place Self-Updater Pipeline**:
   - The GitHub Actions release workflow (`.github/workflows/release.yml`) will publish `wows-presence-daemon.exe` directly as a standalone asset alongside the `.zip` archive.
   - The daemon's updater will query the GitHub Releases API (`GET /repos/varaprasad6700/wows-discord-rpc/releases/latest`), compare semver strings, download the new executable, rename the running binary to `.old`, write the new executable, and relaunch.

3. **Dynamic In-Memory Configuration Reload**:
   - Saving settings from the UI updates the persisted `%APPDATA%\wows-rpc\config.yaml` and hot-reloads the in-memory configuration using a synchronized mutex on `Engine`. No process kill/restart or window flickering is required.

4. **Native Folder Selection Bridge**:
   - Because modern browsers running in `--app` mode do not expose full local Windows filesystem paths to web pages, the daemon's loopback server will expose `POST /api/browse-folder`.
   - This endpoint opens a native Windows folder browser dialog (using PowerShell `Shell.Application.BrowseForFolder` or Win32 COM) and returns the selected absolute path to the UI text input.

5. **Diagnostic File Logging & Rotation**:
   - Diagnostic logging to `%APPDATA%\wows-rpc\daemon.log` is capped at 5MB with single `.log.old` rotation.
   - The UI includes "Open Log File" (launches default text editor) and "Open Folder" (opens Windows Explorer).

---

## 5. What's Left to the Builder
- Selection of specific lightweight tray library (pure Go `systray` without CGO or direct Win32 API).
- Internal loopback API routes for the UI (`/api/status`, `/api/config`, `/api/logs`, `/api/update`, `/api/browse-folder`).

---

## 6. Verification & Acceptance Criteria
1. **Telemetry Verification**: Start WoWS, enter Co-op or Random battle, verify `wows_live_state.json` is generated, and verify Discord profile displays live HP, percentage, damage, and frags.
2. **Background & Startup**: Launch executable with double-click; verify no console window appears, tray icon appears, and "Launch on Windows startup" registry entry is set/unset properly.
3. **Controller UI**: Click "Open Controller" from tray, verify all four tabs function, settings persist to config file, file logging toggle writes to `%APPDATA%\wows-rpc\daemon.log`, and "Open Log File" works.
4. **Update Check**: Click "Check for Updates" and verify it queries GitHub API and reports current status.
