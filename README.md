# World of Warships Discord Rich Presence

Real-time World of Warships combat telemetry and high-density status display directly inside Discord via Discord Rich Presence (IPC named pipe). Ship metadata resolution is offloaded to a dedicated caching backend to prevent client rate-limiting and protect Wargaming API keys.

---

## 🚀 Quick Start Guide

### Running the Client Daemon (End-User)

1. **Download**: Grab **`wows-presence-daemon.exe`** from [GitHub Releases](https://github.com/varaprasad6700/wows-discord-rpc/releases).
2. **Run**: Double-click `wows-presence-daemon.exe`.
   - The application runs silently in the background with zero terminal prompts.
   - A naval anchor icon appears in your Windows system notification tray next to the clock.
   - It automatically detects your World of Warships installation and provisions the embedded telemetry mod.
3. **Control & Settings**: Right-click the system tray icon to:
   - **Open Controller UI**: Launch the Option A tabbed control center to monitor live presence, adjust paths/URLs, or check for updates.
   - **View Log File**: Open `%APPDATA%\wows-rpc\daemon.log` in Notepad for diagnostics.
   - **Check for Updates**: Query GitHub Releases for the latest version.
   - **Exit Presence**: Remove Discord status and stop the daemon cleanly.

---

## ✨ Features

- **Set-and-Forget Windows Startup**: Automatically launches silently on Windows login with no command prompt window, living in the system notification tray.
- **Embedded In-Game Mod**: Telemetry mod is embedded directly inside the single executable (`//go:embed`) and automatically deployed into your WoWS `res_mods` directory without needing external mod files on disk.
- **Option A Tabbed Controller UI**: Clean dark-mode desktop window with four focused tabs:
  - **Status**: Live WoWS process detection, Discord IPC connection state, and real-time presence preview.
  - **Settings**: WoWS install path with native folder browser, metadata server URL, and Windows startup toggle.
  - **Logs & Diagnostics**: One-click toggle for file logging (`%APPDATA%\wows-rpc\daemon.log`) with "Open Log File" and "Open Folder" actions.
  - **Updates**: In-app self-updater that checks GitHub Releases and updates the executable in place.
- **High-Density Combat Presence (Option C)**:
  - **In Battle (Alive)**: `[IX] Menno van Coehoorn • Random (North)` | `54k HP (74%) | 84k DMG | 2 Kills | 8v9`
  - **In Battle (Sunk)**: `[IX] Menno van Coehoorn • Random (North)` | `Sunk (Spectating) | 84k DMG | 2 Kills | 8v9`
  - **In Port**: `In Port` | `Inspecting Fleet & Armory`
  - **Game Closed**: Discord Rich Presence cleared immediately.

---

## 🏛️ System Architecture

```
+-------------------------------------------------------------+
|                     World of Warships Client                 |
|                                                             |
|  +--------------------+         +------------------------+  |
|  | Engine Replays     |         | PnFMods Telemetry Mod  |  |
|  | tempArenaInfo.json |         | Main.py                |  |
|  +---------+----------+         +-----------+------------+  |
+------------|--------------------------------|---------------+
             |                                | (Atomic replace)
             v                                v
    replays/tempArenaInfo.json       wows_live_state.json
             |                                |
             +---------------+----------------+
                             | (File watcher / polling)
                             v
             +-------------------------------+
             |    Go Background Daemon       |
             |  (System Tray + Controller)   |
             +---------------+---------------+
                             |
         +-------------------+-------------------+
         | (HTTP Cache Query)                    | (Discord IPC Pipe)
         v                                       v
+-------------------------------+       +---------------------+
|    Spring Boot Backend        |       | Local Discord Client|
|    (Kotlin + PostgreSQL)      |       | Rich Presence (Opt C|
+---------------+---------------+       +---------------------+
                | (Sync job)
                v
+-------------------------------+
|  Wargaming EU Encyclopedia    |
|  /wows/encyclopedia/ships/    |
+-------------------------------+
```

---

## 🛠️ Verification & Build Commands

- **Python Mod**:
  ```bash
  python -m unittest discover -s mod/tests
  ```
- **Go Daemon (Headless / Tray)**:
  ```powershell
  cd daemon
  go test -v ./...
  go build -ldflags "-H=windowsgui -X main.version=v1.1.0 -X main.defaultDiscordID=1562101043670556782" -o ../wows-presence-daemon.exe ./cmd/daemon
  ```
- **Backend Service**:
  ```powershell
  $env:JAVA_HOME = "C:\Users\pothu\AppData\Local\Programs\IntelliJ IDEA\jbr"
  cd backend
  .\gradlew.bat test
  ```
