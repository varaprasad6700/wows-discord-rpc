# -*- coding: utf-8 -*-
API_VERSION = "API_v1.0"
MOD_NAME = "TelemetryExporter"

try:
    long
except NameError:
    long = int

try:
    basestring
except NameError:
    basestring = (str, bytes)

TICK_INTERVAL = 3.0
callback_handle = None
current_damage = 0
current_frags = 0
self_ship_id = None

def _simple_json_dump(obj):
    if obj is True:
        return "true"
    elif obj is False:
        return "false"
    elif obj is None:
        return "null"
    elif isinstance(obj, (int, long)):
        return str(obj)
    elif isinstance(obj, float):
        return "%.2f" % obj
    elif isinstance(obj, basestring):
        escaped = obj.replace('\\', '\\\\').replace('"', '\\"')
        return '"%s"' % escaped
    elif isinstance(obj, dict):
        parts = []
        for k in sorted(obj.keys()):
            parts.append('"%s": %s' % (k, _simple_json_dump(obj[k])))
        return "{" + ", ".join(parts) + "}"
    elif isinstance(obj, (list, tuple)):
        return "[" + ", ".join([_simple_json_dump(x) for x in obj]) + "]"
    return "null"

def to_json_str(payload):
    try:
        import utils
        if hasattr(utils, "jsonEncode"):
            return utils.jsonEncode(payload)
    except Exception:
        pass

    try:
        import json
        return json.dumps(payload)
    except Exception:
        pass

    return _simple_json_dump(payload)

def log_msg(msg):
    try:
        import utils
        if hasattr(utils, "logInfo"):
            utils.logInfo("[TelemetryExporter] " + str(msg))
            return
    except Exception:
        pass
    try:
        print("[TelemetryExporter] " + str(msg))
    except Exception:
        pass

def atomic_write_json(file_path, payload):
    data_str = to_json_str(payload)
    written = False

    # Standalone environment with os support
    try:
        import os
        tmp_path = file_path + ".tmp"
        with open(tmp_path, "w") as f:
            f.write(data_str)
            f.flush()
        try:
            if hasattr(os, "replace"):
                os.replace(tmp_path, file_path)
            else:
                if os.path.exists(file_path):
                    try:
                        os.remove(file_path)
                    except Exception:
                        pass
                os.rename(tmp_path, file_path)
            written = True
        except Exception:
            pass
    except Exception:
        pass

    # Direct write fallback for restricted WoWS sandbox
    if not written:
        try:
            with open(file_path, "w") as f:
                f.write(data_str)
        except Exception as e:
            log_msg("Write failed: " + str(e))

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
    if hasattr(battle_api, "getSelfPlayerShip"):
        try:
            self_ship = battle_api.getSelfPlayerShip()
        except Exception:
            self_ship = None

    self_team = None
    if self_ship:
        try:
            self_team = getattr(self_ship, "teamId", None)
            if hasattr(self_ship, "getHealthData"):
                health_data = self_ship.getHealthData()
                if health_data:
                    payload["health"] = int(getattr(health_data, "health", 0))
                    payload["max_health"] = int(getattr(health_data, "maxHealth", 0))
                    payload["is_alive"] = bool(payload["health"] > 0)
        except Exception:
            pass

    if hasattr(battle_api, "getAllShips"):
        try:
            all_ships = battle_api.getAllShips() or []
            allies = 0
            enemies = 0
            for ship in all_ships:
                try:
                    if hasattr(ship, "isAlive") and ship.isAlive():
                        if self_team is not None and getattr(ship, "teamId", None) == self_team:
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
    global callback_handle, current_damage, current_frags, self_ship_id
    try:
        battle_module = None
        try:
            import battle as battle_module
        except Exception:
            battle_module = globals().get("battle")

        if battle_module:
            if hasattr(battle_module, "getSelfPlayerInfo"):
                try:
                    pinfo = battle_module.getSelfPlayerInfo()
                    if pinfo:
                        self_ship_id = pinfo.get("shipId", self_ship_id)
                except Exception:
                    pass

            if hasattr(battle_module, "getBattleStatistics"):
                try:
                    stats = battle_module.getBattleStatistics()
                    if stats:
                        for attr in ("damageDealt", "damage", "totalDamage"):
                            if hasattr(stats, attr):
                                val = getattr(stats, attr)
                                if val is not None:
                                    current_damage = max(current_damage, int(val))
                        for attr in ("frags", "kills", "shipsKilled"):
                            if hasattr(stats, attr):
                                val = getattr(stats, attr)
                                if val is not None:
                                    current_frags = max(current_frags, int(val))
                except Exception:
                    pass

        payload = build_telemetry_payload(
            battle_api=battle_module,
            ship_id=self_ship_id,
            damage_dealt=current_damage,
            frags=current_frags,
        )
        atomic_write_json("wows_live_state.json", payload)
    except Exception as e:
        log_msg("tick error: " + str(e))

    callbacks_module = None
    try:
        import callbacks as callbacks_module
    except Exception:
        callbacks_module = globals().get("callbacks")

    if callbacks_module and hasattr(callbacks_module, "callback"):
        try:
            callback_handle = callbacks_module.callback(TICK_INTERVAL, tick_telemetry)
        except Exception:
            pass

def on_receive_shell_info(victim_id, shooter_id, ammo_id, mat_id, shoot_id, booleans, damage, *args, **kwargs):
    global current_damage, current_frags, self_ship_id
    try:
        if self_ship_id is not None and shooter_id == self_ship_id:
            current_damage += int(damage)
            if isinstance(booleans, (int, long)) and (booleans & 8):
                current_frags += 1
    except Exception:
        pass

def on_battle_start(*args, **kwargs):
    global current_damage, current_frags
    log_msg("Battle started, initiating telemetry loop.")
    current_damage = 0
    current_frags = 0
    tick_telemetry()

def on_battle_end(*args, **kwargs):
    global callback_handle
    log_msg("Battle ended, clearing telemetry loop.")
    callbacks_module = None
    try:
        import callbacks as callbacks_module
    except Exception:
        callbacks_module = globals().get("callbacks")

    if callback_handle is not None and callbacks_module and hasattr(callbacks_module, "cancel"):
        try:
            callbacks_module.cancel(callback_handle)
        except Exception:
            pass
        callback_handle = None

    atomic_write_json("wows_live_state.json", {"in_battle": False})

# Register event callbacks
events_module = None
try:
    import events as events_module
except Exception:
    events_module = globals().get("events")

if events_module:
    try:
        if hasattr(events_module, "onBattleStart"):
            events_module.onBattleStart(on_battle_start)
        if hasattr(events_module, "onBattleEnd"):
            events_module.onBattleEnd(on_battle_end)
        if hasattr(events_module, "onBattleQuit"):
            events_module.onBattleQuit(on_battle_end)
        if hasattr(events_module, "onReceiveShellInfo"):
            events_module.onReceiveShellInfo(on_receive_shell_info)
        log_msg("TelemetryExporter loaded and hooked to game events.")
    except Exception as e:
        log_msg("Hook error: " + str(e))
