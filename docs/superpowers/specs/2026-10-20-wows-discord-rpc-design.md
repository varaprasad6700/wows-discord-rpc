# World of Warships Discord Rich Presence System Design

## Intent & The Why
World of Warships players often want their Discord status to reflect their active gameplay in real time—not just generic game presence, but specific combat status including the ship they are commanding, its tier, current match type, map, and dynamic battle performance (health percentage, damage dealt, frags, and team standing).

Rather than querying Wargaming's public API directly from every player client (which risks rate-limiting, leaks API keys, and incurs latency), this system utilizes an intermediate caching backend in Kotlin/Spring Boot with PostgreSQL, paired with a lightweight, zero-dependency Go local daemon and an in-game telemetry mod.

---

## Repository & Hosting Strategy
As requested:
1. **Client / Daemon / Mod (Public GitHub)**: `wows-discord-rpc`
   - Contains the Go local daemon (`wows-presence-daemon`), configuration templates, and the Python telemetry mod files (`PnFModsLoader.py` & `TelemetryExporter/Main.py`).
   - Packaged and distributed via GitHub Releases as a standalone Windows executable.
2. **Backend Service (Self-Hosted / Gitea)**: `wows-meta-service`
   - Contains the Kotlin Spring Boot caching service, PostgreSQL schema/Flyway migrations, and `docker-compose.yml`.
   - Hosted privately on the user's Gitea instance and deployable via Docker.

---

## Goals, Non-Goals, and Anti-Goals

### Goals
1. **Auto-Discovery & Mod Deployment**: The local daemon automatically detects the active World of Warships installation path (via Windows registry for Wargaming Game Center and Steam, with a `config.yaml` override), locates the latest build directory under `bin/`, and ensures both `res_mods/PnFModsLoader.py` and `res_mods/PnFMods/TelemetryExporter/Main.py` are installed and up-to-date.
2. **Combat Telemetry Extraction**: The Python mod hooks into the official `PnFMods` ModAPI to extract real-time player health, maximum health, damage dealt, frags (kills), and living friendly vs. enemy ship counts, writing atomically to `wows_live_state.json` every 5 seconds.
3. **Battle Lifecycle & Process Tracking**: The daemon watches `<WoWS_root>/replays/tempArenaInfo.json` to detect battle entry and read static match metadata (map identifier, battle type/game mode, player ship ID, and battle start timestamp). It also monitors the `WorldOfWarships64.exe` process to clear Discord presence when the game exits.
4. **Intermediate Caching Backend**: A Kotlin Spring Boot application backed by PostgreSQL syncs ship metadata from the Wargaming EU Encyclopedia API (`/wows/encyclopedia/ships/`) and serves localized ship names, Roman numeral tiers, and ship classes to the client daemon via a lightweight REST endpoint (`GET /api/v1/ships/{shipId}`).
5. **Discord Rich Presence (Option C)**: The Go daemon formats telemetry into a high-density status card and pushes updates to the local Discord client via Discord IPC named pipe (`\\.\pipe\discord-ipc-0`) every 15 seconds.
6. **Graceful State Transitions**: Automatically falls back to "In Port" status when `tempArenaInfo.json` is deleted or the match ends, clears Discord status when the game closes, and reconnects smoothly if Discord or the game restarts.
7. **Local Reproducibility**: Includes a `docker-compose.yml` to spin up the Kotlin backend and PostgreSQL database locally for personal use and testing.

### Non-Goals
- In-game HUD modifications or visual overlays on the screen.
- Direct memory reading, code injection, or hooking into unapproved game memory addresses (strictly adheres to Wargaming's `PnFMods` standard ModAPI).
- Interactive Discord bot commands (only client-side Discord IPC Rich Presence updates).
- Dynamic multi-region switching per match (defaults to EU encyclopedia, configurable in backend).

### Anti-Goals
- Any invasive mod techniques that could trigger anti-cheat or violate Wargaming Terms of Service.
- Leaking Wargaming API keys into distributed client binaries.
- High memory or CPU footprint on the gaming client (daemon must idle under 20MB RAM in Go).
- Stale or corrupted Discord status caused by race conditions when reading files during game writes.

---

## Technical Specifications

### 1. In-Game Mod: `TelemetryExporter`
- **Location**: `<WoWS_root>/bin/<build_id>/res_mods/PnFMods/TelemetryExporter/Main.py`
- **Loader**: `<WoWS_root>/bin/<build_id>/res_mods/PnFModsLoader.py` (deployed automatically by the daemon if missing so vanilla game installs execute the mod).
- **Engine Environment**: Python 2.7 & 3 compatible polyglot syntax (standard JSON, safe string formatting).
- **Events Hooked**:
  - `events.onBattleStart`: Initializes telemetry loop.
  - `events.onBattleEnd`, `events.onBattleQuit`: Clears combat state and writes `{"in_battle": false}`.
  - `callbacks.callback(5.0, dump_telemetry)`: Periodic 5-second polling loop.
- **Metrics Collected**:
  - `in_battle`: Boolean.
  - `ship_id`: Integer local player ship identifier.
  - `health`: Current HP integer.
  - `max_health`: Maximum HP integer.
  - `damage_dealt`: Integer total damage inflicted.
  - `frags`: Integer number of enemies destroyed.
  - `allies_alive`: Integer count of living friendly ships.
  - `enemies_alive`: Integer count of living enemy ships.
  - `is_alive`: Boolean (`health > 0`).
- **File Output**: Writes to `<WoWS_root>/wows_live_state.json.tmp` and performs an atomic replace to `<WoWS_root>/wows_live_state.json` to prevent partial reads by the daemon.

### 2. Local Daemon: Go (`wows-presence-daemon`)
- **Runtime**: Go (native Windows executable, zero external dependencies).
- **WoWS Installation Auto-Detection**:
  1. Registry check: `HKCU\Software\Wargaming.net\GameCenter` and `HKLM\Software\WOW6432Node\Valve\Steam` (Steam library folders `steamapps/common/World of Warships`).
  2. Fallback check: `C:\Games\World_of_Warships` and custom path from `config.yaml`.
  3. Locates highest numerical directory in `bin/` (e.g. `bin/8291045/`).
  4. Deploys both `PnFModsLoader.py` and `PnFMods/TelemetryExporter/Main.py` if missing or updated.
- **Process & Lifecycle Management**:
  - Checks if `WorldOfWarships64.exe` (or `WorldOfWarships.exe`) is active.
  - If process is not running: Clears Discord Rich Presence.
  - If process is running and no match is active: Displays "In Port".
- **Replay & State Watcher**:
  - Watches `<WoWS_root>/replays/tempArenaInfo.json` and polls `<WoWS_root>/wows_live_state.json`.
  - Parses match metadata:
    - Map ID: Mapped via dictionary to human-readable map name (e.g., `spaces/15_NE_north` -> `North`).
    - Match Group / Game Mode: e.g. `pvp` -> `Random Battle`, `ranked` -> `Ranked Battle`, `cooperative` -> `Co-op Battle`.
    - Battle Start Time: Unix timestamp for Discord elapsed time clock.
    - Player Vehicle: Identifies player's vehicle ID and team relation.
- **Discord IPC Client**:
  - Connects to Windows named pipe `\\.\pipe\discord-ipc-0`.
  - Updates Discord Rich Presence every 15 seconds.
  - Presence Layout (Option C):
    - **Details**: `[{RomanTier}] {ShipName} • {BattleType} ({MapName})`  
      *Example:* `[X] Yamato • Random (North)`
    - **State (Alive)**: `{CurrentHP_k}k HP ({HP_Pct}%) | {DMG_k}k DMG | {Frags} Kills | {Allies}v{Enemies}`  
      *Example:* `72k HP (74%) | 84k DMG | 2 Kills | 9v7`
    - **State (Sunk)**: `Sunk (Spectating) | {DMG_k}k DMG | {Frags} Kills | {Allies}v{Enemies}`
    - **Timestamps**: `start: {battle_start_unix}`
    - **Assets**:
      - `large_image`: Ship or class icon key (hover: `{ShipName} - {CurrentHP}/{MaxHP} HP`)
      - `small_image`: Alive checkmark or class badge (hover: `{ShipClass}`)
    - **In Port Fallback**:
      - Details: `In Port`
      - State: `Inspecting Fleet & Armory`
      - Large Image: `wows_logo` (hover: `World of Warships`)
- **Backend Communication**:
  - Queries `http://{backend_host}:{port}/api/v1/ships/{shipId}`.
  - Caches results locally in memory to minimize HTTP round-trips.

### 3. Backend Service: Kotlin + Spring Boot (`wows-meta-service`)
- **Runtime**: Kotlin, Spring Boot 3.x, Gradle.
- **Database**: PostgreSQL 16 (configured via `docker-compose.yml`).
- **Data Model**:
  - `ShipEntity`:
    - `shipId` (Long, Primary Key)
    - `name` (String, localized display name, e.g. "Yamato")
    - `tier` (Int, 1-11)
    - `romanTier` (String, e.g. "X")
    - `shipClass` (Enum: `DESTROYER`, `CRUISER`, `BATTLESHIP`, `AIR_CARRIER`, `SUBMARINE`)
    - `nation` (String, e.g. "japan")
    - `iconUrl` (String)
    - `lastUpdated` (Instant)
- **Wargaming API Integration**:
  - Endpoint: `https://api.worldofwarships.eu/wows/encyclopedia/ships/`
  - Parameters: `application_id`, `fields=name,tier,type,nation,images.small`
  - Sync Mechanism: Scheduled sync on startup / weekly background cron, plus on-demand cache miss fetching.
- **REST Endpoints**:
  - `GET /api/v1/ships/{shipId}` -> Returns cached ship JSON.
  - `POST /api/v1/ships/sync` -> Triggers full encyclopedia refresh from Wargaming.

### 4. Configuration & Deployment
- `config.yaml` for Go Daemon:
  ```yaml
  wows:
    install_path: "" # Auto-detected if empty
  backend:
    url: "http://localhost:8080"
  discord:
    client_id: "DEFAULT_APPLICATION_ID"
    update_interval_seconds: 15
  ```
- `docker-compose.yml` for Backend:
  - Service 1: `postgres:16-alpine` with healthcheck.
  - Service 2: `wows-meta-service` (Spring Boot app) with environment variables for database credentials and Wargaming `APPLICATION_ID`.

---

## Left to the Builder
- Specific implementation of Wargaming map identifier dictionary mapping (mapping internal space names like `spaces/15_NE_north` to `North`).
- Specific Discord Application asset upload scripts or hosting icon CDN URLs.
- Exact Go package layout (`cmd/daemon`, `internal/wows`, `internal/discord`, `internal/api`).
- Exact Spring Boot package layout (`com.wows.meta.controller`, `service`, `repository`, `model`).

---

## References & Official Resources
- **Wargaming ModSDK**: [https://github.com/wgmods/ModSDK](https://github.com/wgmods/ModSDK) (Official modding SDK data, assets, and localization definitions)
- **Wargaming Mods-API-Documentation**: [https://github.com/wgmods/Mods-API-Documentation](https://github.com/wgmods/Mods-API-Documentation) (Official Python ModsAPI event hooks, callback specifications, and runtime environment details)
- **Wargaming Developer Room (Encyclopedia API)**: [https://developers.wargaming.net/reference/all/wows/encyclopedia/ships/](https://developers.wargaming.net/reference/all/wows/encyclopedia/ships/)

