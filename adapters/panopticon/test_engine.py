"""Smoke + determinism tests for the built-in placeholder engine.

Run: python3 adapters/panopticon/test_engine.py   (or: python3 -m unittest)
"""

import os
import sys
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from protostubs import entity_pb2, mission_pb2, results_pb2, scenario_pb2  # noqa: E402
from engine import BuiltinEngine  # noqa: E402


def _scenario():
    s = scenario_pb2.ScenarioProto(id="s1", name="duel", duration_s=120)
    s.entities.add(
        id="blue-1", name="Blue 1", type=entity_pb2.ENTITY_TYPE_FIXED_WING,
        side=entity_pb2.SIDE_FRIENDLY, position=entity_pb2.Position(lat=26.0, lon=56.0, alt_m=8000),
    )
    s.entities.add(
        id="red-1", name="Red 1", type=entity_pb2.ENTITY_TYPE_SURFACE_VESSEL,
        side=entity_pb2.SIDE_ENEMY, position=entity_pb2.Position(lat=26.05, lon=56.05, alt_m=0),
    )
    m = s.missions.add(entity_id="blue-1", mission_type=mission_pb2.MISSION_TYPE_STRIKE,
                       roe=mission_pb2.ROE_WEAPONS_FREE)
    m.waypoints.add(lat=26.05, lon=56.05, alt_m=4000, speed_ms=250)
    return s


class TestBuiltinEngine(unittest.TestCase):
    def test_produces_tracks_and_events(self):
        res = BuiltinEngine().run(_scenario(), "panopticon", "run-A")
        self.assertEqual(res.engine_id, "panopticon")
        self.assertEqual(len(res.entity_tracks), 2)
        self.assertTrue(all(len(t.points) > 0 for t in res.entity_tracks))
        # The two entities are close and weapons-free/tight -> at least detection.
        types = {e.type for e in res.events}
        self.assertIn(results_pb2.EVENT_TYPE_DETECTION, types)
        keys = {m.key for m in res.moe_metrics}
        self.assertEqual(keys, {
            "blue_losses", "red_losses", "blue_kills", "red_kills", "total_kills",
            "detections_total", "rounds_expended", "avg_health_pct",
        })

    def test_deterministic_for_same_run_id(self):
        a = BuiltinEngine().run(_scenario(), "panopticon", "run-X")
        b = BuiltinEngine().run(_scenario(), "panopticon", "run-X")
        self.assertEqual(len(a.kill_chains), len(b.kill_chains))
        self.assertEqual([e.type for e in a.events], [e.type for e in b.events])


if __name__ == "__main__":
    unittest.main()
