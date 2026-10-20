# -*- coding: utf-8 -*-
API_VERSION = "API_v1.0"
MOD_NAME = "TelemetryExporter"

import json
import os
import sys

TICK_INTERVAL = 5.0
callback_handle = None
current_damage = 0
current_frags = 0

def atomic_write_json(file_path, payload):
    tmp_path = file_path + ".tmp"
    data_str = json.dumps(payload)
    with open(tmp_path, "w") as f:
        f.write(data_str)
        f.flush()
        os.fsync(f.fileno())

    # Python 3.3+ supports atomic os.replace on Windows; Python 2.7 requires remove + rename
    if hasattr(os, "replace"):
        os.replace(tmp_path, file_path)
    else:
        if os.path.exists(file_path):
            try:
                os.remove(file_path)
            except Exception:
                pass
        os.rename(tmp_path, file_path)

def build_telemetry_payload(battle_api, ship_id=None, damage_dealt=0, frags=0):
    payload = {
        "in_battle": True,
        "ship_id": ship_id,
        "health": 0,
        "max_health": 0,
        "damage_dealt": int(damage_dealt),
        "frags": int(frags),
        "allies_alive": 0,
        "enemies_alive": 0,
        "is_alive": False,
    }

    if not battle_api:
        return payload

    self_ship = None
    try:
        self_ship = battle_api.getSelfPlayerShip()
    except Exception:
        self_ship = None

    self_team = None
    if self_ship:
        try:
            self_team = self_ship.teamId
            health_data = self_ship.getHealthData()
            if health_data:
                payload["health"] = int(health_data.health)
                payload["max_health"] = int(health_data.maxHealth)
                payload["is_alive"] = bool(payload["health"] > 0)
        except Exception:
            pass

    try:
        all_ships = battle_api.getAllShips() or []
        allies = 0
        enemies = 0
        for ship in all_ships:
            try:
                if ship.isAlive():
                    if self_team is not None and ship.teamId == self_team:
                        allies += 1
                    else:
                        enemies += 1
            except Exception:
                continue
        payload["allies_alive"] = allies
        payload["enemies_alive"] = enemies
    except Exception:
        pass

    return payload

def tick_telemetry():
    global callback_handle, current_damage, current_frags
    try:
        battle_module = sys.modules.get("battle")
        player_info = None
        ship_id = None
        if battle_module and hasattr(battle_module, "getSelfPlayerInfo"):
            try:
                player_info = battle_module.getSelfPlayerInfo()
                if player_info:
                    ship_id = player_info.get("shipId")
            except Exception:
                pass

        payload = build_telemetry_payload(
            battle_api=battle_module,
            ship_id=ship_id,
            damage_dealt=current_damage,
            frags=current_frags,
        )
        atomic_write_json("wows_live_state.json", payload)
    except Exception:
        pass

    callbacks_module = sys.modules.get("callbacks")
    if callbacks_module and hasattr(callbacks_module, "callback"):
        callback_handle = callbacks_module.callback(TICK_INTERVAL, tick_telemetry)

def on_battle_start(*args, **kwargs):
    global current_damage, current_frags
    current_damage = 0
    current_frags = 0
    tick_telemetry()

def on_battle_end(*args, **kwargs):
    global callback_handle
    callbacks_module = sys.modules.get("callbacks")
    if callback_handle is not None and callbacks_module and hasattr(callbacks_module, "cancel"):
        try:
            callbacks_module.cancel(callback_handle)
        except Exception:
            pass
        callback_handle = None

    try:
        atomic_write_json("wows_live_state.json", {"in_battle": False})
    except Exception:
        pass

# Hook into WoWS game events if running inside client environment
events_module = sys.modules.get("events")
if events_module:
    try:
        if hasattr(events_module, "onBattleStart"):
            events_module.onBattleStart(on_battle_start)
        if hasattr(events_module, "onBattleEnd"):
            events_module.onBattleEnd(on_battle_end)
        if hasattr(events_module, "onBattleQuit"):
            events_module.onBattleQuit(on_battle_end)
    except Exception:
        pass
