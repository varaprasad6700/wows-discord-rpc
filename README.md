# World of Warships Discord Rich Presence

Real-time World of Warships combat telemetry and high-density status display directly inside Discord via Discord Rich Presence (IPC named pipe). Ship metadata resolution is offloaded to a dedicated caching backend to prevent client rate-limiting and protect Wargaming API keys.

---

## Architecture Overview

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
|    (Kotlin + PostgreSQL)      |       | Rich Presence       |
+---------------+---------------+       +---------------------+
                | (Sync job)
                v
+-------------------------------+
|  Wargaming EU Encyclopedia    |
|  /wows/encyclopedia/ships/    |
+-------------------------------+
```

---

## Repository Split & Components

This project is organized into two deployment targets:

### 1. Public Client & Mod (`wows-discord-rpc`) — Target: Public GitHub
- **`mod/`**:
  - `PnFModsLoader.py`: Mod bootstrap loader script required for vanilla WoWS clients.
  - `PnFMods/TelemetryExporter/Main.py`: In-game Python telemetry script capturing live health, damage dealt, frags, and living team counts.
- **`daemon/`**:
  - `wows-presence-daemon.exe`: Zero-dependency Go Windows binary that auto-detects WoWS installations, installs/verifies the mod, tracks game process lifecycle, queries ship metadata, and updates Discord via named pipe (`\\.\pipe\discord-ipc-0`).
- **`config.example.yaml`**: Default daemon configuration file.

### 2. Backend Service (`wows-meta-service`) — Target: Private Gitea / Self-Hosted
- **`backend/`**:
  - Kotlin + Spring Boot 3 + PostgreSQL 16 caching service.
  - Syncs ship definitions from the Wargaming EU Encyclopedia API (`/wows/encyclopedia/ships/`).
  - REST endpoints: `GET /api/v1/ships/{shipId}` and `POST /api/v1/ships/sync`.
- **`docker-compose.yml`**: Production-ready containerized deployment for PostgreSQL and the Spring Boot service.

---

## Rich Presence Format (Option C - High-Density)

- **In Battle (Alive)**:
  - **Details**: `[{RomanTier}] {ShipName} • {BattleType} ({MapName})`  
    *Example:* `[X] Yamato • Random Battle (North)`
  - **State**: `{CurrentHP}k HP ({HP_Pct}%) | {DMG}k DMG | {Frags} Kills | {Allies}v{Enemies}`  
    *Example:* `72k HP (74%) | 84k DMG | 2 Kills | 9v7`
  - **Timestamps**: Elapsed match time (`StartTime`)
  - **Assets**: Large ship icon key hover text `Yamato - 72000/97200 HP`, small icon class badge
- **In Battle (Sunk)**:
  - **State**: `Sunk (Spectating) | 84k DMG | 2 Kills | 9v7`
- **In Port / Menu**:
  - **Details**: `In Port`
  - **State**: `Inspecting Fleet & Armory`
- **Game Closed**:
  - Discord Rich Presence cleared immediately.

---

## Quick Start Guide

### Running the Client Daemon (End-User)

1. Download the latest release:
   - `wows-presence-daemon.exe`
   - `mod/` directory
   - `config.example.yaml` (rename to `config.yaml` if customizing)
2. (Optional) Customizing `config.yaml`:
   ```yaml
   wows:
     install_path: "" # Leave empty for automatic registry & disk discovery
   backend:
     url: "http://localhost:8080" # URL of your metadata caching backend
   discord:
     client_id: "123456789012345678" # Your Discord Developer Application Client ID
     update_interval_seconds: 15
   ```
3. Run `wows-presence-daemon.exe`.
   - The daemon automatically detects your latest WoWS build (e.g. `bin/8291045/`) and copies `PnFModsLoader.py` and `TelemetryExporter/Main.py` into `res_mods/`.
4. Launch World of Warships and join any battle!

---

### Running the Self-Hosted Metadata Backend

1. Configure your Wargaming Application ID in `docker-compose.yml` or an `.env` file:
   ```env
   WARGAMING_APPLICATION_ID=your_wargaming_api_key
   ```
2. Start the service and database:
   ```bash
   docker compose up -d
   ```
3. Trigger an encyclopedia sync (optional on initial setup):
   ```bash
   curl -X POST http://localhost:8080/api/v1/ships/sync
   ```
4. Query ship metadata:
   ```bash
   curl http://localhost:8080/api/v1/ships/4181230000
   ```

---

## Development & Test Commands

### 1. Python Telemetry Mod
```bash
python -m unittest discover -s mod/tests
```

### 2. Go Daemon
```bash
cd daemon
go test -v ./...
go build -o ../wows-presence-daemon.exe ./cmd/daemon
```

### 3. Spring Boot Backend
```bash
cd backend
./gradlew test
```
