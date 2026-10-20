# AGENTS.md

Welcome to the **World of Warships Discord Rich Presence** workspace. This document provides essential architecture context, component boundaries, development conventions, and testing instructions for AI agents and human contributors.

---

## 1. System Overview & Architecture

This project provides real-time World of Warships combat telemetry and status display directly inside Discord via Discord Rich Presence (IPC named pipe). To prevent client rate-limiting and protect API keys, ship metadata resolution is offloaded to a dedicated caching backend.

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
             |   (System Tray + Controller)  |
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

## 2. Component Structure & Repository Split

The project is structured into two logical repositories:

### A. Public Client & Mod (`wows-discord-rpc`) — Target: Public GitHub
Contains the lightweight end-user components distributed via GitHub Releases:
- **`mod/`**:
  - `PnFModsLoader.py`: Mod bootstrap loader script required for vanilla WoWS clients.
  - `PnFMods/TelemetryExporter/Main.py`: In-game Python telemetry script capturing live health, damage dealt, frags, and living team numbers.
- **`daemon/`**:
  - Go application (`wows-presence-daemon.exe`) that detects the game installation, installs/verifies the mod, watches game state, connects to Discord IPC named pipe (`\\.\pipe\discord-ipc-0`), and resolves ship metadata.
  - Includes native Windows system tray icon, Edge app mode launcher, loopback server for Option A tabbed controller, and GitHub self-updater.
- **`assets/discord/`**:
  - 512×512 RGBA PNG art assets for Discord Rich Presence (`wows_logo`, `cruiser`, `battleship`, `destroyer`, `aircarrier`, `submarine`, `sunk`).

### B. Backend Service (`wows-meta-service`) — Target: Private Gitea
Contains the self-hosted ship encyclopedia metadata cache:
- **`backend/`**:
  - Kotlin + Spring Boot 3 application with Gradle.
  - PostgreSQL 16 integration via Spring Data JPA / Flyway.
  - Wargaming EU Encyclopedia API client (`/wows/encyclopedia/ships/`) with scheduled background sync and on-demand cache miss resolution.
  - REST endpoints: `GET /api/v1/ships/{shipId}` and `POST /api/v1/ships/sync`.
- **`docker-compose.yml`**: Compose configuration defining the Spring Boot service and PostgreSQL container.

---

## 3. Technology Stack & Environment

- **Operating System**: Windows (primary target for WoWS client and Go daemon).
- **Go Daemon**: Go 1.22+ (native binary, zero runtime dependencies, `//go:embed` for UI and mod assets).
- **In-Game Mod**: Python 2.7 & 3 compatible polyglot (WoWS BigWorld client uses embedded Python).
- **Backend Service**: Kotlin 1.9+, Java/JDK 21, Spring Boot 3.x, Gradle.
- **Database**: PostgreSQL 16.
- **Discord Integration**: Local Discord IPC named pipe (`\\.\pipe\discord-ipc-0`).

---

## 4. Key Design Decisions

1. **Presence Format (Option C - High-Density)**:
   - **Details**: `[{RomanTier}] {ShipName} • {BattleType} ({MapName})` (e.g. `[X] Yamato • Random (North)`)
   - **State**: `{CurrentHP}k HP ({HP_Pct}%) | {DMG}k DMG | {Frags} Kills | {Allies}v{Enemies}` (e.g. `72k HP (74%) | 84k DMG | 2 Kills | 9v7`)
   - **Fallback**: `In Port - Inspecting Fleet & Armory`
2. **Atomic State Serialization**:
   - The Python mod writes to `wows_live_state.json.tmp` and renames/replaces it atomically to `wows_live_state.json` to prevent partial reads by the daemon.
3. **Mod Loader Provisioning & Embedding**:
   - The daemon embeds the full mod tree (`//go:embed`) and automatically places `PnFModsLoader.py` and `TelemetryExporter/Main.py` into `bin/<build_id>/res_mods/` if missing. No external `mod/` folder on disk is required.
4. **Python Mod Sandbox Rules**:
   - NEVER use `import json`, `import os`, `globals()`, or `locals()` in in-game Python scripts (`Main.py`). WoWS's BigWorld sandbox will raise unhandled `NameError` or `ImportError`. Always use injected `utils`, `events`, `callbacks`, `battle`, and `xml.sax.saxutils.os`.
5. **Windows Background Tray & Controller**:
   - Compiled with `-H=windowsgui` so no console prompt opens. Runs in the system tray next to the Windows clock with a right-click context menu.
   - Option A Tabbed Controller UI is served on an internal loopback server (`http://127.0.0.1:<port>`) and launched in standalone Edge application mode (`msedge.exe --app`).
6. **Logging & Startup Paths**:
   - File logs: `%APPDATA%\wows-rpc\daemon.log` (capped at 5MB with 1 backup).
   - Config: `%APPDATA%\wows-rpc\config.yaml`.
   - Windows Startup: `HKCU\Software\Microsoft\Windows\CurrentVersion\Run\WoWSPresence`.

---

## 5. Verification & Test Commands

- **Python Mod**:
  ```bash
  python -m unittest discover -s mod/tests
  ```
- **Go Daemon**:
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
