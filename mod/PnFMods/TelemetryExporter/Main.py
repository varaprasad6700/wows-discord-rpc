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

def _get_os():
    try:
        import xml.sax.saxutils as _sax
        if hasattr(_sax, "os"):
            return _sax.os
    except Exception:
        pass
    return None

def _get_events():
    try:
        return events
    except NameError:
        return None

def _get_callbacks():
    try:
        return callbacks
    except NameError:
        return None

def _get_battle():
    try:
        return battle
    except NameError:
        return None

def _get_utils():
    try:
        return utils
    except NameError:
        return None

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
    u = _get_utils()
    if u and hasattr(u, "jsonEncode"):
        try:
            return u.jsonEncode(payload)
        except Exception:
            pass
    return _simple_json_dump(payload)

def log_msg(msg):
    u = _get_utils()
    if u and hasattr(u, "logInfo"):
        try:
            u.logInfo("[TelemetryExporter] " + str(msg))
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
    _os = _get_os()

    if _os:
        tmp_path = file_path + ".tmp"
        try:
            with open(tmp_path, "w") as f:
                f.write(data_str)
                f.flush()
            if hasattr(_os, "replace"):
                _os.replace(tmp_path, file_path)
            else:
                if hasattr(_os, "path") and _os.path.exists(file_path):
                    try:
                        _os.remove(file_path)
                    except Exception:
                        pass
                _os.rename(tmp_path, file_path)
            written = True
        except Exception:
            pass

    if not written:
        try:
            with open(file_path, "w") as f:
                f.write(data_str)
        except Exception as e:
            log_msg("Write failed: " + str(e))

def export_state_to_all_paths(payload):
    # 1. Write to cwd
    atomic_write_json("wows_live_state.json", payload)

    # 2. If utils and os available, resolve build and root directories
    u = _get_utils()
    _os = _get_os()
    if u and hasattr(u, "getModDir") and _os and hasattr(_os, "path"):
        try:
            mod_dir = u.getModDir() # .../bin/<build>/res_mods/PnFMods
            build_dir = _os.path.abspath(_os.path.join(mod_dir, "..", ".."))
            root_dir = _os.path.abspath(_os.path.join(build_dir, "..", ".."))
            atomic_write_json(_os.path.join(build_dir, "wows_live_state.json"), payload)
            atomic_write_json(_os.path.join(root_dir, "wows_live_state.json"), payload)
        except Exception:
            pass

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
        b = _get_battle()
        if b:
            if hasattr(b, "getSelfPlayerInfo"):
                try:
                    pinfo = b.getSelfPlayerInfo()
                    if pinfo and isinstance(pinfo, dict):
                        self_ship_id = pinfo.get("shipId", self_ship_id)
                except Exception:
                    pass

            if hasattr(b, "getBattleStatistics"):
                try:
                    stats = b.getBattleStatistics()
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
            battle_api=b,
            ship_id=self_ship_id,
            damage_dealt=current_damage,
            frags=current_frags,
        )
        export_state_to_all_paths(payload)
    except Exception as e:
        log_msg("tick error: " + str(e))

    c = _get_callbacks()
    if c and hasattr(c, "callback"):
        try:
            callback_handle = c.callback(TICK_INTERVAL, tick_telemetry)
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
    c = _get_callbacks()
    if callback_handle is not None and c and hasattr(c, "cancel"):
        try:
            c.cancel(callback_handle)
        except Exception:
            pass
        callback_handle = None

    export_state_to_all_paths({"in_battle": False})

# Register event callbacks directly from injected scope
ev = _get_events()
if ev:
    try:
        if hasattr(ev, "onBattleStart"):
            ev.onBattleStart(on_battle_start)
        if hasattr(ev, "onBattleEnd"):
            ev.onBattleEnd(on_battle_end)
        if hasattr(ev, "onBattleQuit"):
            ev.onBattleQuit(on_battle_end)
        if hasattr(ev, "onReceiveShellInfo"):
            ev.onReceiveShellInfo(on_receive_shell_info)
        log_msg("TelemetryExporter loaded and hooked to game events successfully.")
    except Exception as e:
        log_msg("Hook error: " + str(e))
