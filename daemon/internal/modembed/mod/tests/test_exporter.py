import json
import os
import shutil
import sys
import tempfile
import unittest

sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))

class TestTelemetryExporter(unittest.TestCase):
    def setUp(self):
        self.test_dir = tempfile.mkdtemp()
        self.state_file = os.path.join(self.test_dir, "wows_live_state.json")

    def tearDown(self):
        shutil.rmtree(self.test_dir)

    def test_atomic_write_creates_valid_json(self):
        from PnFMods.TelemetryExporter.Main import atomic_write_json

        data = {
            "in_battle": True,
            "ship_id": 4181230000,
            "health": 72000,
            "max_health": 97200,
            "damage_dealt": 84200,
            "frags": 2,
            "allies_alive": 9,
            "enemies_alive": 7,
            "is_alive": True,
        }

        atomic_write_json(self.state_file, data)

        self.assertTrue(os.path.exists(self.state_file))
        self.assertFalse(os.path.exists(self.state_file + ".tmp"))

        with open(self.state_file, "r") as f:
            loaded = json.loads(f.read())

        self.assertEqual(loaded, data)

    def test_on_receive_damages_on_ship_adds_fire_and_flood_damage(self):
        import PnFMods.TelemetryExporter.Main as mod

        mod.current_damage = 0
        mod.self_ship_id = 4181230000
        mod.self_avatar_id = 100

        class MockPlayer:
            def __init__(self, avatar_id, ship_id):
                self.id = avatar_id
                self.shipId = ship_id

        class MockBattleDamages:
            def getSelfPlayerInfo(self):
                return MockPlayer(100, 4181230000)
            def getSelfPlayerShip(self):
                class MockShipObj:
                    id = 10001
                return MockShipObj()
            def getPlayerByVehicleId(self, veh_id):
                if veh_id == 10001:
                    return MockPlayer(100, 4181230000)
                elif veh_id == 10002:
                    return MockPlayer(200, 4281230000)
                return None

        orig_battle = getattr(mod, "battle", None)
        try:
            mod.battle = MockBattleDamages()

            # Fire damage tick from local player (veh_id 10001) onto enemy (veh_id 10002)
            damages_fire = [{"vehicleID": 10001, "damage": 1250.0}]
            mod.on_receive_damages_on_ship(10002, damages_fire)
            self.assertEqual(int(mod.current_damage), 1250)

            # Flooding damage tick from local player onto enemy
            damages_flood = [{"vehicleID": 10001, "damage": 2300.0}]
            mod.on_receive_damages_on_ship(10002, damages_flood)
            self.assertEqual(int(mod.current_damage), 3550)

            # Damage from another player onto enemy - should NOT be added
            damages_other = [{"vehicleID": 99999, "damage": 5000.0}]
            mod.on_receive_damages_on_ship(10002, damages_other)
            self.assertEqual(int(mod.current_damage), 3550)

            # Damage where victim is self ship - should NOT be added
            damages_self_victim = [{"vehicleID": 10001, "damage": 500.0}]
            mod.on_receive_damages_on_ship(10001, damages_self_victim)
            self.assertEqual(int(mod.current_damage), 3550)
        finally:
            if orig_battle is not None:
                mod.battle = orig_battle
            elif hasattr(mod, "battle"):
                delattr(mod, "battle")

    def test_on_receive_shell_info_does_not_double_count_when_damages_hooked(self):
        import PnFMods.TelemetryExporter.Main as mod

        mod.current_damage = 1000
        mod.current_frags = 0
        mod.self_ship_id = 4181230000
        mod.has_damages_hook = True
        mod.killed_victims.clear()

        # Shell info fired - because has_damages_hook is True, damage is not double-counted
        mod.on_receive_shell_info(victim_id=2000, shooter_id=4181230000, ammo_id=1, mat_id=1, shoot_id=1, booleans=0, damage=500)
        self.assertEqual(int(mod.current_damage), 1000)

        # But frag flag (booleans & 8) is still captured
        mod.on_receive_shell_info(victim_id=2000, shooter_id=4181230000, ammo_id=1, mat_id=1, shoot_id=1, booleans=8, damage=500)
        self.assertEqual(mod.current_frags, 1)
        self.assertIn(2000, mod.killed_victims)

        # Same victim again should not increment frags twice
        mod.on_receive_shell_info(victim_id=2000, shooter_id=4181230000, ammo_id=1, mat_id=1, shoot_id=1, booleans=8, damage=0)
        self.assertEqual(mod.current_frags, 1)

    def test_on_receive_vehicle_death_tracks_frags_from_fire_and_floods(self):
        import PnFMods.TelemetryExporter.Main as mod

        mod.current_frags = 0
        mod.self_ship_id = 4181230000
        mod.killed_victims.clear()

        # Fire kills victim 3000
        mod.on_receive_vehicle_death(victim_id=3000, killer_id=4181230000, reason=2)
        self.assertEqual(mod.current_frags, 1)
        self.assertIn(3000, mod.killed_victims)

        # Duplicate event for same victim
        mod.on_receive_vehicle_death(victim_id=3000, killer_id=4181230000, reason=2)
        self.assertEqual(mod.current_frags, 1)

        # Kill by another player should not increment
        mod.on_receive_vehicle_death(victim_id=4000, killer_id=999999, reason=1)
        self.assertEqual(mod.current_frags, 1)

    def test_calculate_telemetry_with_mock_battle(self):
        from PnFMods.TelemetryExporter.Main import build_telemetry_payload

        class MockHealthData:
            def __init__(self, hp, max_hp):
                self.health = hp
                self.maxHealth = max_hp

        class MockShip:
            def __init__(self, team_id, alive, hp=50000, max_hp=50000):
                self.teamId = team_id
                self._alive = alive
                self._health_data = MockHealthData(hp, max_hp)

            def isAlive(self):
                return self._alive

            def getHealthData(self):
                return self._health_data

        class MockBattle:
            def __init__(self, self_ship, all_ships):
                self._self_ship = self_ship
                self._all_ships = all_ships

            def getSelfPlayerShip(self):
                return self._self_ship

            def getAllShips(self):
                return self._all_ships

        # Local player is team 0 (friendly)
        self_ship = MockShip(team_id=0, alive=True, hp=72000, max_hp=97200)
        all_ships = [
            self_ship,
            MockShip(team_id=0, alive=True),
            MockShip(team_id=0, alive=False), # Dead ally
            MockShip(team_id=1, alive=True), # Alive enemy
            MockShip(team_id=1, alive=True), # Alive enemy
            MockShip(team_id=1, alive=False), # Dead enemy
        ]

        mock_battle = MockBattle(self_ship, all_ships)
        payload = build_telemetry_payload(mock_battle, ship_id=4181230000, damage_dealt=45000, frags=1)

        self.assertTrue(payload["in_battle"])
        self.assertEqual(payload["ship_id"], 4181230000)
        self.assertEqual(payload["health"], 72000)
        self.assertEqual(payload["max_health"], 97200)
        self.assertEqual(payload["allies_alive"], 2)
        self.assertEqual(payload["enemies_alive"], 2)
        self.assertEqual(payload["damage_dealt"], 45000)
        self.assertEqual(payload["frags"], 1)
        self.assertTrue(payload["is_alive"])

    def test_build_telemetry_fallback_when_self_ship_none(self):
        from PnFMods.TelemetryExporter.Main import build_telemetry_payload

        class MockBattleEmpty:
            def getSelfPlayerShip(self):
                return None
            def getAllShips(self):
                return []

        payload = build_telemetry_payload(MockBattleEmpty(), ship_id=12345, damage_dealt=0, frags=0)
        self.assertTrue(payload["in_battle"])
        self.assertEqual(payload["health"], 0)
        self.assertEqual(payload["max_health"], 0)
        self.assertFalse(payload["is_alive"])
        self.assertEqual(payload["allies_alive"], 0)
        self.assertEqual(payload["enemies_alive"], 0)

    def test_no_forbidden_globals_or_builtins_in_mod(self):
        mod_file = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "PnFMods", "TelemetryExporter", "Main.py"))
        with open(mod_file, "r") as f:
            content = f.read()
        self.assertNotIn("globals()", content, "globals() is forbidden in WoWS Python sandbox")
        self.assertNotIn("locals()", content, "locals() is forbidden in WoWS Python sandbox")
        self.assertNotIn("import json", content, "import json is forbidden in WoWS Python sandbox")
        self.assertNotIn("import os", content, "top-level import os is forbidden in WoWS Python sandbox")

if __name__ == "__main__":
    unittest.main()
