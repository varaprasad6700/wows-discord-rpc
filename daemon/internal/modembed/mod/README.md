# World of Warships Telemetry Exporter Mod

In-game Python telemetry script for World of Warships utilizing the official `PnFMods` ModAPI.

## Structure
- `PnFModsLoader.py`: Mod loader script deployed to `bin/<build_id>/res_mods/` on vanilla game installations.
- `PnFMods/TelemetryExporter/Main.py`: Telemetry script extracting real-time battle stats to `wows_live_state.json`.
