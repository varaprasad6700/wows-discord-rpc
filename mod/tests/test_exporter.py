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

if __name__ == "__main__":
    unittest.main()
