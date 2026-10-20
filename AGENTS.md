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
             |       Go Local Daemon         |
             |   (wows-presence-daemon.exe)  |
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
- **`config.example.yaml`**: Default daemon configuration file.

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
- **Go Daemon**: Go 1.22+ (native binary, zero runtime dependencies).
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
3. **Mod Loader Provisioning**:
   - The daemon automatically places `PnFModsLoader.py` into `bin/<build_id>/res_mods/` if missing, allowing vanilla installations without modpacks to run the telemetry exporter.
4. **Process Tracking**:
   - The daemon checks for `WorldOfWarships64.exe`. If the game closes, Discord Rich Presence is cleared immediately rather than lingering as "In Port".

---

## 5. Specifications & Reference Links

- **Design Document**: [`docs/superpowers/specs/2026-10-20-wows-discord-rpc-design.md`](docs/superpowers/specs/2026-10-20-wows-discord-rpc-design.md)
- **Official ModSDK**: [https://github.com/wgmods/ModSDK](https://github.com/wgmods/ModSDK)
- **Official Mods-API Documentation**: [https://github.com/wgmods/Mods-API-Documentation](https://github.com/wgmods/Mods-API-Documentation)
- **Wargaming Developer Portal**: [https://developers.wargaming.net/reference/all/wows/encyclopedia/ships/](https://developers.wargaming.net/reference/all/wows/encyclopedia/ships/)

---

## 6. Verification & Test Commands

- **Python Mod**:
  ```bash
  python -m unittest discover -s mod/tests
  ```
- **Go Daemon**:
  ```bash
  cd daemon
  go test -v ./...
  go build -o ../wows-presence-daemon.exe ./cmd/daemon
  ```
- **Backend Service**:
  ```powershell
  $env:JAVA_HOME = "C:\Users\pothu\AppData\Local\Programs\IntelliJ IDEA\jbr"
  cd backend
  .\gradlew.bat test
  ```
