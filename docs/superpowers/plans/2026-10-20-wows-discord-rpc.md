# World of Warships Discord Rich Presence Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a complete World of Warships Discord Rich Presence system consisting of an in-game telemetry mod, a zero-dependency Go local daemon for Windows, and a Kotlin/Spring Boot caching backend backed by PostgreSQL.

**Architecture:** The in-game Python mod extracts combat metrics (HP, damage, frags, alive counts) to an atomic live JSON file. The Go daemon monitors match arena replay data and live telemetry, queries ship metadata from the Kotlin backend, and streams high-density Option C Rich Presence updates over Discord IPC named pipes.

**Tech Stack:**
- Python 2.7 / 3 polyglot (WoWS BigWorld ModAPI)
- Go 1.22+ (Windows native daemon, named pipe IPC)
- Kotlin 1.9+, Java 21, Spring Boot 3.x, Gradle (Backend caching service)
- PostgreSQL 16 (Flyway database migrations, Docker Compose)

**Spec:** [`docs/superpowers/specs/2026-10-20-wows-discord-rpc-design.md`](file:///C:/Users/pothu/android-projects/wows-discord-rpc/docs/superpowers/specs/2026-10-20-wows-discord-rpc-design.md)

---

## Global Constraints

- Windows is the primary target OS for the Go daemon and WoWS client.
- Mod code must remain compatible with WoWS's embedded Python 2.7 runtime (no f-strings, no Python 3.10+ syntax).
- State file serialization must use atomic replacement (`wows_live_state.json.tmp` -> `wows_live_state.json`) to prevent partial reads.
- Discord IPC named pipe path on Windows: `\\.\pipe\discord-ipc-0`.
- Presence format must strictly match Option C (Details: `[{RomanTier}] {ShipName} • {BattleType} ({MapName})`, State: `{HP}k HP ({%}) | {DMG}k DMG | {Frags} Kills | {Allies}v{Enemies}`).
- Repository split: Public client & mod in `wows-discord-rpc`; backend service in `wows-meta-service`.

## Review Focus

1. **Atomic File Write Race**: Mod writing `wows_live_state.json` while the Go daemon attempts to read it; expected: no JSON decode errors due to atomic rename.
2. **Game Process Termination**: `WorldOfWarships64.exe` closes during battle or port; expected: Discord Rich Presence is cleared immediately rather than lingering indefinitely in "In Port".
3. **Vanilla Client Mod Loading**: Player does not use ModStation or Aslain modpack; expected: daemon automatically deploys `PnFModsLoader.py` into `res_mods/` so `TelemetryExporter/Main.py` executes without manual configuration.
4. **Backend Cache Miss / Offline Handling**: Wargaming API is temporarily down or shipId is unrecognized; expected: daemon falls back to displaying the ship ID or a graceful placeholder without crashing.
5. **Discord Client Disconnect/Reconnect**: Discord client is opened after the game or restarted mid-match; expected: daemon detects pipe disconnection and automatically reconnects without crashing.

---

### Task 1: Repository Structure & Configuration Scaffolding

**Files:**
- Create: `config.example.yaml`
- Create: `docker-compose.yml`
- Create: `mod/README.md`
- Create: `daemon/README.md`
- Create: `backend/README.md`
- Modify: `.gitignore`

**Interfaces:**
- Produces: Base configuration template and directory structure for daemon, mod, and backend.

- [ ] **Step 1: Write test verifying configuration template schema**

Create `daemon/internal/config/config_test.go` checking that `config.example.yaml` parses into expected Go config struct with default values.

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./daemon/internal/config/...`
Expected: FAIL (package / files do not exist yet).

- [ ] **Step 3: Create directory structure and configuration files**

- Create `config.example.yaml` containing:
  ```yaml
  wows:
    install_path: ""
  backend:
    url: "http://localhost:8080"
  discord:
    client_id: "123456789012345678"
    update_interval_seconds: 15
  ```
- Create `docker-compose.yml` defining PostgreSQL 16 container and backend service container.
- Update `.gitignore` for Go, Gradle, and Python artifacts.

- [ ] **Step 4: Implement minimal Config parser and verify test passes**

Run: `go test ./daemon/internal/config/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add config.example.yaml docker-compose.yml .gitignore daemon/ mod/ backend/
git commit -m "chore: scaffold project structure, config templates, and docker-compose"
```

---

### Task 2: In-Game Python Telemetry Mod (`TelemetryExporter`)

**Files:**
- Create: `mod/PnFModsLoader.py`
- Create: `mod/PnFMods/TelemetryExporter/Main.py`
- Create: `mod/tests/test_exporter.py`

**Interfaces:**
- Consumes: WoWS ModAPI (`events.onBattleStart`, `events.onBattleEnd`, `callbacks.callback`, `battle.getSelfPlayerShip()`, `battle.getAllShips()`).
- Produces: Atomically replaced file `<WoWS_root>/wows_live_state.json` with keys: `in_battle`, `ship_id`, `health`, `max_health`, `damage_dealt`, `frags`, `allies_alive`, `enemies_alive`, `is_alive`.

- [ ] **Step 1: Write unit tests in `mod/tests/test_exporter.py`**

Test atomic serialization logic, living allies/enemies tallying from mock ship collections, and fallback behavior when `battle.getSelfPlayerShip()` returns `None`.

- [ ] **Step 2: Run test to verify failure**

Run: `python -m unittest discover -s mod/tests`
Expected: FAIL (module not implemented).

- [ ] **Step 3: Implement `PnFModsLoader.py` and `Main.py`**

- `mod/PnFModsLoader.py`: Standard bootstrap script that scans `res_mods/PnFMods/*/Main.py` and executes `init()` or module-level entrypoints.
- `mod/PnFMods/TelemetryExporter/Main.py`:
  - Polyglot Python 2.7 / 3 syntax.
  - Registers `events.onBattleStart`, `events.onBattleEnd`, `events.onBattleQuit`.
  - Ticks every 5.0s using `callbacks.callback`.
  - Tally: friendly vs. enemy alive counts via `battle.getAllShips()` and `ship.isAlive()`.
  - Reads health data via `battle.getSelfPlayerShip().getHealthData()`.
  - Writes to `wows_live_state.json.tmp` and renames to `wows_live_state.json`.

- [ ] **Step 4: Run unit tests to verify they pass**

Run: `python -m unittest discover -s mod/tests`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mod/
git commit -m "feat(mod): implement PnFModsLoader and TelemetryExporter script"
```

---

### Task 3: Backend PostgreSQL Schema & Data Model (Kotlin + Spring Boot)

**Files:**
- Create: `backend/build.gradle.kts`
- Create: `backend/settings.gradle.kts`
- Create: `backend/src/main/resources/application.yml`
- Create: `backend/src/main/resources/db/migration/V1__create_ships_table.sql`
- Create: `backend/src/main/kotlin/com/wows/meta/model/ShipEntity.kt`
- Create: `backend/src/main/kotlin/com/wows/meta/repository/ShipRepository.kt`
- Create: `backend/src/test/kotlin/com/wows/meta/repository/ShipRepositoryTest.kt`

**Interfaces:**
- Produces: `ShipEntity` with `shipId` (Long), `name` (String), `tier` (Int), `romanTier` (String), `shipClass` (String), `nation` (String), `iconUrl` (String), `lastUpdated` (Instant).
- Produces: `ShipRepository: JpaRepository<ShipEntity, Long>`.

- [ ] **Step 1: Write JPA repository test in `ShipRepositoryTest.kt`**

Test saving and retrieving a `ShipEntity` (e.g. `shipId = 4181230000`, `name = "Yamato"`, `tier = 10`, `romanTier = "X"`, `shipClass = "Battleship"`).

- [ ] **Step 2: Run Gradle test to verify failure**

Run: `cd backend && ./gradlew test` (or `gradlew.bat test`)
Expected: FAIL (project and entities do not exist).

- [ ] **Step 3: Implement Flyway migration, Entity, and Repository**

- Create `V1__create_ships_table.sql` defining `ships` table with indexed `ship_id`.
- Implement `ShipEntity` and `ShipRepository`.
- Configure H2 in-memory test configuration in `backend/src/test/resources/application-test.yml`.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && ./gradlew test`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/
git commit -m "feat(backend): set up Spring Boot project, Flyway schema, and ShipRepository"
```

---

### Task 4: Backend Wargaming EU Encyclopedia Sync & REST API

**Files:**
- Create: `backend/src/main/kotlin/com/wows/meta/dto/WargamingShipDto.kt`
- Create: `backend/src/main/kotlin/com/wows/meta/service/WargamingSyncService.kt`
- Create: `backend/src/main/kotlin/com/wows/meta/controller/ShipController.kt`
- Create: `backend/src/test/kotlin/com/wows/meta/controller/ShipControllerTest.kt`

**Interfaces:**
- Consumes: `https://api.worldofwarships.eu/wows/encyclopedia/ships/?application_id=...&page_no=...`
- Produces:
  - `GET /api/v1/ships/{shipId}` -> returns `ShipResponseDto(shipId, name, tier, romanTier, shipClass, nation, iconUrl)`.
  - `POST /api/v1/ships/sync` -> triggers asynchronous catalog sync and returns sync status.

- [ ] **Step 1: Write MockMvc controller test in `ShipControllerTest.kt`**

Test `GET /api/v1/ships/4181230000` returning `200 OK` with JSON `{ "shipId": 4181230000, "name": "Yamato", "romanTier": "X", "shipClass": "Battleship" }`, and `404 Not Found` when a shipId does not exist.

- [ ] **Step 2: Run test to verify failure**

Run: `cd backend && ./gradlew test --tests ShipControllerTest`
Expected: FAIL (controller not implemented).

- [ ] **Step 3: Implement Roman numeral converter, `WargamingSyncService`, and `ShipController`**

- Implement helper `toRoman(tier: Int): String` (1 -> "I", 10 -> "X", 11 -> "★").
- Implement `WargamingSyncService`: queries Wargaming EU API with pagination, upserts ships into `ShipRepository`.
- Implement `ShipController`: handles `GET /api/v1/ships/{shipId}` and `POST /api/v1/ships/sync`.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && ./gradlew test`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/
git commit -m "feat(backend): implement Wargaming encyclopedia sync and ShipController"
```

---

### Task 5: Go Daemon Config & WoWS Game Installation Detector

**Files:**
- Create: `daemon/go.mod`
- Create: `daemon/internal/config/config.go`
- Create: `daemon/internal/detector/detector.go`
- Create: `daemon/internal/detector/detector_test.go`

**Interfaces:**
- Produces:
  - `detector.FindWoWSPath() (string, error)`: checks Registry/Steam/Defaults/Config.
  - `detector.FindLatestBuild(wowsPath string) (string, error)`: finds highest numerical folder in `bin/`.
  - `detector.EnsureModDeployed(buildPath string, modSourceDir string) error`: deploys `PnFModsLoader.py` and `TelemetryExporter/Main.py`.

- [ ] **Step 1: Write detector tests in `detector_test.go`**

Test finding highest build number from a mock `bin/` directory structure containing `bin/100000`, `bin/105000`, `bin/99999` (expected `105000`), and deploying mod files into `res_mods/`.

- [ ] **Step 2: Run test to verify failure**

Run: `cd daemon && go test ./internal/detector/...`
Expected: FAIL (functions not defined).

- [ ] **Step 3: Implement `detector.go`**

- Implement `FindLatestBuild(wowsPath string) (string, error)`: lists directories in `bin/`, parses names as integers, selects maximum.
- Implement `EnsureModDeployed(buildPath, modSourceDir string) error`: creates `res_mods/PnFMods/TelemetryExporter/`, copies `PnFModsLoader.py` and `Main.py` if missing or differing in checksum.
- Implement `FindWoWSPath()`: checks Windows Registry keys (`HKCU\Software\Wargaming.net\GameCenter` and `HKLM\Software\WOW6432Node\Valve\Steam`) with fallback to standard `C:\Games\World_of_Warships`.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd daemon && go test ./internal/detector/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add daemon/
git commit -m "feat(daemon): implement WoWS installation detector and mod deployer"
```

---

### Task 6: Go Daemon Process Tracker & Replay / State Watcher

**Files:**
- Create: `daemon/internal/process/tracker.go`
- Create: `daemon/internal/process/tracker_test.go`
- Create: `daemon/internal/watcher/arena.go`
- Create: `daemon/internal/watcher/state.go`
- Create: `daemon/internal/watcher/watcher_test.go`

**Interfaces:**
- Produces:
  - `process.IsGameRunning() bool`: checks for `WorldOfWarships64.exe` or `WorldOfWarships.exe`.
  - `watcher.ParseArenaInfo(path string) (*ArenaInfo, error)`: parses `tempArenaInfo.json` to extract `MapName`, `BattleType`, `ShipID`, `StartTime`.
  - `watcher.ReadLiveState(path string) (*LiveState, error)`: parses `wows_live_state.json`.

- [ ] **Step 1: Write parser tests in `watcher_test.go`**

Test parsing actual WoWS `tempArenaInfo.json` sample fixture (mapping map ID `spaces/15_NE_north` to `North`, `matchGroup: "pvp"` to `Random Battle`) and `wows_live_state.json` fixture.

- [ ] **Step 2: Run test to verify failure**

Run: `cd daemon && go test ./internal/watcher/...`
Expected: FAIL (parser not defined).

- [ ] **Step 3: Implement `process/tracker.go` and `watcher/*.go`**

- `tracker.go`: uses Windows API or process list to check if WoWS client is running.
- `arena.go`: parses `tempArenaInfo.json`, maps space names to friendly names, extracts player vehicle ID and battle start Unix timestamp.
- `state.go`: reads `wows_live_state.json` safely with retry on file-locking.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd daemon && go test ./internal/process/... ./internal/watcher/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add daemon/internal/process/ daemon/internal/watcher/
git commit -m "feat(daemon): implement process tracker and game state parsers"
```

---

### Task 7: Go Daemon Backend API Client & Discord IPC Presence (Option C)

**Files:**
- Create: `daemon/internal/api/client.go`
- Create: `daemon/internal/api/client_test.go`
- Create: `daemon/internal/discord/ipc.go`
- Create: `daemon/internal/discord/presence.go`
- Create: `daemon/internal/discord/presence_test.go`

**Interfaces:**
- Produces:
  - `api.Client`: `GetShip(shipID int64) (*ShipMetadata, error)` with internal thread-safe cache.
  - `discord.Client`: `Connect() error`, `SetActivity(activity *Activity) error`, `ClearActivity() error`.
  - `discord.FormatOptionC(arena *ArenaInfo, live *LiveState, ship *ShipMetadata) *Activity`.

- [ ] **Step 1: Write presence formatter tests in `presence_test.go`**

Test `FormatOptionC` producing:
- In-Battle Alive:
  - Details: `[X] Yamato • Random Battle (North)`
  - State: `72k HP (74%) | 84k DMG | 2 Kills | 9v7`
- In-Battle Sunk:
  - Details: `[X] Yamato • Random Battle (North)`
  - State: `Sunk (Spectating) | 84k DMG | 2 Kills | 9v7`
- In Port:
  - Details: `In Port`
  - State: `Inspecting Fleet & Armory`

- [ ] **Step 2: Run test to verify failure**

Run: `cd daemon && go test ./internal/discord/...`
Expected: FAIL (functions not defined).

- [ ] **Step 3: Implement `api/client.go` and `discord/presence.go`**

- `api/client.go`: HTTP client with `sync.RWMutex` cache querying `GET /api/v1/ships/{shipId}`.
- `discord/ipc.go`: Handles Windows named pipe handshake (`\\.\pipe\discord-ipc-0`) using JSON-RPC opcodes.
- `discord/presence.go`: Formats Option C activity cards and handles automatic reconnect on pipe failure.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd daemon && go test ./internal/api/... ./internal/discord/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add daemon/internal/api/ daemon/internal/discord/
git commit -m "feat(daemon): implement cached metadata API client and Discord IPC Option C presence"
```

---

### Task 8: Daemon Main Loop Integration & Docker Compose Verification

**Files:**
- Create: `daemon/cmd/daemon/main.go`
- Modify: `docker-compose.yml`
- Create: `daemon/cmd/daemon/main_test.go`

**Interfaces:**
- Produces: Standalone executable `wows-presence-daemon.exe`.

- [ ] **Step 1: Write integration test for the daemon loop state machine**

Test state transitions:
1. Game Not Running -> Presence cleared.
2. Game Running, no `tempArenaInfo.json` -> "In Port" activity set.
3. Battle Started (`tempArenaInfo.json` created) -> Option C in-battle activity set.
4. Battle Ended (`tempArenaInfo.json` removed) -> transitions back to "In Port".

- [ ] **Step 2: Run integration test to verify failure**

Run: `cd daemon && go test ./cmd/daemon/...`
Expected: FAIL (main loop integration not implemented).

- [ ] **Step 3: Implement `cmd/daemon/main.go`**

- Loads `config.yaml` (or `config.example.yaml` fallback).
- Runs `detector.FindWoWSPath()` and `detector.EnsureModDeployed()`.
- Connects to Discord IPC.
- Starts ticker every `update_interval_seconds` (15s):
  - Checks `process.IsGameRunning()`.
  - Checks if `tempArenaInfo.json` exists.
  - Resolves ship metadata via backend client.
  - Pushes Rich Presence activity.
- Handles graceful shutdown on `os.Interrupt` / `SIGTERM` clearing Discord presence.

- [ ] **Step 4: Run integration test and build standalone binary**

Run:
```bash
cd daemon
go test -v ./...
go build -o ../wows-presence-daemon.exe ./cmd/daemon
```
Expected: PASS, and binary `wows-presence-daemon.exe` is generated.

- [ ] **Step 5: Commit**

```bash
git add daemon/cmd/daemon/ docker-compose.yml
git commit -m "feat(daemon): implement main daemon loop and build verification"
```

---

### Task 9: End-to-End Verification & Documentation

**Files:**
- Modify: `README.md`
- Modify: `AGENTS.md`

**Interfaces:**
- Produces: Complete end-to-end documentation, build and run instructions for both public client distribution and private Gitea backend hosting.

- [ ] **Step 1: Run all test suites across mod, backend, and daemon**

Run:
```bash
python -m unittest discover -s mod/tests
cd daemon && go test ./...
cd ../backend && ./gradlew test
```
Expected: All suites PASS.

- [ ] **Step 2: Update project `README.md`**

Provide setup and usage instructions:
- How to start the caching backend via `docker compose up -d`.
- How to run `wows-presence-daemon.exe`.
- How mod auto-deployment works.

- [ ] **Step 3: Commit and tag**

```bash
git add README.md AGENTS.md
git commit -m "docs: add comprehensive setup guide and end-to-end verification instructions"
```
