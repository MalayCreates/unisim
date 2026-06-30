"""Unit tests for the shared DIS listener. Run with: python3 -m unittest -v
from the adapters/shared directory (or python3 adapters/shared/test_dis.py)."""

import os
import socket
import sys
import time
import unittest

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from shared.disbridge import geo  # noqa: E402
from shared.disbridge import (  # noqa: E402
    DISCollector,
    DISListener,
    encode_detonation,
    encode_entity_state,
    encode_fire,
    parse_pdu,
)
from shared.disbridge.pdu import DetonationPDU, EntityStatePDU, FirePDU  # noqa: E402


class TestGeo(unittest.TestCase):
    def test_roundtrip(self):
        for lat, lon, alt in [
            (0.0, 0.0, 0.0),
            (26.6, 56.4, 9000.0),
            (-33.9, 151.2, 120.0),
            (51.5, -0.12, 35.0),
        ]:
            x, y, z = geo.geodetic_to_ecef(lat, lon, alt)
            lat2, lon2, alt2 = geo.ecef_to_geodetic(x, y, z)
            self.assertAlmostEqual(lat, lat2, places=6)
            self.assertAlmostEqual(lon, lon2, places=6)
            self.assertAlmostEqual(alt, alt2, places=3)

    def test_heading_due_east(self):
        # An ECEF velocity pointing local-east at (0N, 0E) is +Y.
        speed, heading = geo.ecef_velocity_to_ground(0.0, 250.0, 0.0, 0.0, 0.0)
        self.assertAlmostEqual(speed, 250.0, places=3)
        self.assertAlmostEqual(heading, 90.0, places=3)

    def test_heading_due_north(self):
        # Local-north at (0N, 0E) is +Z.
        speed, heading = geo.ecef_velocity_to_ground(0.0, 0.0, 200.0, 0.0, 0.0)
        self.assertAlmostEqual(speed, 200.0, places=3)
        self.assertAlmostEqual(heading, 0.0, places=3)


class TestPDUCodec(unittest.TestCase):
    def test_entity_state_roundtrip(self):
        raw = encode_entity_state(
            (1, 1, 42), 26.6, 56.4, 9000.0, marking="Hammer 1"
        )
        pdu = parse_pdu(raw)
        self.assertIsInstance(pdu, EntityStatePDU)
        self.assertEqual(pdu.entity_id, (1, 1, 42))
        self.assertEqual(pdu.marking, "Hammer 1")
        self.assertAlmostEqual(pdu.lat, 26.6, places=4)
        self.assertAlmostEqual(pdu.lon, 56.4, places=4)
        self.assertAlmostEqual(pdu.alt_m, 9000.0, places=1)
        self.assertFalse(pdu.destroyed)

    def test_entity_state_destroyed_flag(self):
        raw = encode_entity_state((1, 1, 7), 0, 0, 0, destroyed=True)
        pdu = parse_pdu(raw)
        self.assertTrue(pdu.destroyed)

    def test_fire_and_detonation(self):
        fire = parse_pdu(encode_fire((1, 1, 1), (1, 1, 2)))
        self.assertIsInstance(fire, FirePDU)
        self.assertEqual(fire.firing_id, (1, 1, 1))
        self.assertEqual(fire.target_id, (1, 1, 2))

        hit = parse_pdu(encode_detonation((1, 1, 1), (1, 1, 2), result=1))
        self.assertIsInstance(hit, DetonationPDU)
        self.assertTrue(hit.is_hit)

        miss = parse_pdu(encode_detonation((1, 1, 1), (1, 1, 2), result=3))
        self.assertFalse(miss.is_hit)

    def test_unknown_pdu_is_none(self):
        # A header with an unsupported pdu_type decodes to None.
        raw = bytearray(encode_fire((1, 1, 1), (1, 1, 2)))
        raw[2] = 99  # pdu_type
        self.assertIsNone(parse_pdu(bytes(raw)))


class TestCollector(unittest.TestCase):
    def test_builds_tracks_events_and_killchain(self):
        # Resolver maps DIS markings to USIP ids.
        resolver = lambda key, marking: marking or key  # noqa: E731
        c = DISCollector(id_resolver=resolver)

        c.on_pdu(parse_pdu(encode_entity_state((1, 1, 1), 26.0, 56.0, 8000, marking="blue-1")), 1000)
        c.on_pdu(parse_pdu(encode_entity_state((1, 1, 2), 26.5, 56.5, 0, marking="red-1")), 1000)
        c.on_pdu(parse_pdu(encode_fire((1, 1, 1), (1, 1, 2))), 2000)
        c.on_pdu(parse_pdu(encode_detonation((1, 1, 1), (1, 1, 2), result=1)), 3000)
        c.on_pdu(parse_pdu(encode_entity_state((1, 1, 2), 26.5, 56.5, 0, marking="red-1", destroyed=True)), 4000)

        res = c.results()
        self.assertEqual(len(res.entity_tracks), 2)
        ids = {t.entity_id for t in res.entity_tracks}
        self.assertEqual(ids, {"blue-1", "red-1"})

        types = [e.type for e in res.events]
        self.assertIn("engagement", types)
        self.assertIn("kill", types)

        self.assertEqual(len(res.kill_chains), 1)
        kc = res.kill_chains[0]
        self.assertEqual(kc.attacker_entity_id, "blue-1")
        self.assertEqual(kc.target_entity_id, "red-1")
        self.assertEqual(kc.engaged_at_ms, 2000)  # paired from the Fire PDU
        self.assertEqual(kc.killed_at_ms, 3000)

        # red-1's final track point reflects the destroyed status.
        red = next(t for t in res.entity_tracks if t.entity_id == "red-1")
        self.assertEqual(red.points[-1].status, "killed")


class TestListenerLoopback(unittest.TestCase):
    def test_receives_over_udp(self):
        # Pick a free UDP port and round-trip a few PDUs through the socket.
        probe = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        probe.bind(("127.0.0.1", 0))
        port = probe.getsockname()[1]
        probe.close()

        listener = DISListener(host="127.0.0.1", port=port)
        listener.start()
        try:
            tx = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            tx.sendto(encode_entity_state((1, 1, 1), 26.0, 56.0, 8000, marking="blue-1"), ("127.0.0.1", port))
            tx.sendto(encode_fire((1, 1, 1), (1, 1, 2)), ("127.0.0.1", port))
            tx.sendto(encode_detonation((1, 1, 1), (1, 1, 2), result=1), ("127.0.0.1", port))
            tx.close()

            deadline = time.time() + 2.0
            while time.time() < deadline:
                res = listener.results()
                if res.kill_chains:
                    break
                time.sleep(0.02)
        finally:
            listener.stop()

        res = listener.results()
        self.assertTrue(any(t.entity_id == "blue-1" for t in res.entity_tracks))
        self.assertEqual(len(res.kill_chains), 1)


if __name__ == "__main__":
    unittest.main()
