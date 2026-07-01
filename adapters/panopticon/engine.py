"""Simulation engine backends for the Panopticon adapter.

Two backends implement the same `run()` contract:

  * BuiltinEngine    — a small, deterministic reference sim (waypoint movement +
                       detect/engage/kill). Lets the adapter run end-to-end today
                       without Panopticon installed, so the cross-language adapter
                       protocol and spawn-on-demand can be exercised.
  * PanopticonEngine — the seam for the real Panopticon Gymnasium engine. Raises
                       a clear error until implemented.

Swap backends with the adapter's --backend-engine flag.
"""

from __future__ import annotations

import hashlib
import math
import random
from typing import List

from protostubs import entity_pb2, mission_pb2, results_pb2, scenario_pb2

import normalizer
import translator

_EARTH_RADIUS_M = 6371000.0
_STEP_S = 2.0
_REENGAGE_COOLDOWN_S = 10.0
_FALLBACK_SPEED_MS = 100.0


class BuiltinEngine:
    """Deterministic placeholder sim. Seeded by run_id for reproducibility."""

    name = "builtin"

    def run(self, scenario, engine_id, run_id) -> "results_pb2.SimResultsProto":
        entities = translator.translate(scenario)
        duration = float(scenario.duration_s) if scenario.duration_s > 0 else 600.0
        start_epoch = scenario.start_time.seconds if scenario.start_time.seconds > 0 else _now_epoch()
        rng = random.Random(_seed_from_run_id(run_id))

        tracks = {e.id: results_pb2.EntityTrack(entity_id=e.id) for e in entities}
        track_order = [e.id for e in entities]
        events: List["results_pb2.SimEvent"] = []
        kill_chains: List["results_pb2.KillChain"] = []
        last_engage = {}

        t = 0.0
        while t <= duration:
            for e in entities:
                if e.alive:
                    _advance(e, t, events, start_epoch)
                _record(e, t, tracks, start_epoch)
            _interactions(entities, t, rng, events, kill_chains, last_engage, start_epoch)
            t += _STEP_S

        return results_pb2.SimResultsProto(
            scenario_id=scenario.id,
            engine_id=engine_id,
            run_id=run_id,
            entity_tracks=[tracks[i] for i in track_order],
            events=events,
            kill_chains=kill_chains,
            moe_metrics=_compute_moes(entities, kill_chains, events),
        )


class PanopticonEngine:
    """SEAM: drive the real Panopticon Gymnasium engine."""

    name = "panopticon"

    def run(self, scenario, engine_id, run_id):
        try:
            import gymnasium  # noqa: F401
            import panopticon  # noqa: F401
        except ImportError as exc:
            raise RuntimeError(
                "Panopticon (and gymnasium) is not installed. Install it and "
                "implement translator.to_panopticon_scenario / "
                "normalizer.normalize_panopticon, or run the adapter with "
                "--backend-engine builtin."
            ) from exc

        # Real integration outline (left as the seam):
        #   pano_scn = translator.to_panopticon_scenario(scenario)
        #   env = gymnasium.make("panopticon/...", scenario=pano_scn)
        #   obs, _ = env.reset(seed=_seed_from_run_id(run_id))
        #   while not done: obs, _, done, _, _ = env.step(policy(obs))
        #   return normalizer.normalize_panopticon(obs_log, scenario.id, engine_id, run_id)
        raise NotImplementedError("Panopticon engine integration is a TODO.")


def get_engine(name: str):
    if name == "panopticon":
        return PanopticonEngine()
    return BuiltinEngine()


# --- simulation steps ---


def _advance(e: "translator.SimEntity", t, events, start_epoch) -> None:
    if e.wp_index >= len(e.waypoints):
        e.speed_ms = 0.0
        return
    if t < e.hold_until_s:
        e.speed_ms = 0.0
        return

    target = e.waypoints[e.wp_index]
    speed = target.speed_ms if target.speed_ms > 0 else _FALLBACK_SPEED_MS
    dist = _haversine_m(e.lat, e.lon, target.lat, target.lon)
    step_dist = speed * _STEP_S

    if step_dist >= dist:
        e.lat, e.lon, e.alt_m = target.lat, target.lon, target.alt_m
        e.speed_ms = speed
        events.append(
            normalizer.event(start_epoch + t, results_pb2.EVENT_TYPE_WAYPOINT_REACHED,
                             e.id, detail="reached waypoint %d" % e.wp_index)
        )
        if target.hold_time_s > 0:
            e.hold_until_s = t + float(target.hold_time_s)
        e.wp_index += 1
        if e.wp_index >= len(e.waypoints):
            events.append(
                normalizer.event(start_epoch + t, results_pb2.EVENT_TYPE_MISSION_COMPLETE,
                                 e.id, detail="final waypoint reached")
            )
        return

    brng = _bearing_deg(e.lat, e.lon, target.lat, target.lon)
    e.lat, e.lon = _move_point(e.lat, e.lon, step_dist, brng)
    e.alt_m = target.alt_m
    e.heading_deg = brng
    e.speed_ms = speed


def _interactions(entities, t, rng, events, kill_chains, last_engage, start_epoch) -> None:
    for a in entities:
        if not a.alive or a.sensor_range_m <= 0:
            continue
        for b in entities:
            if b is a or not b.alive or not _are_enemies(a.side, b.side):
                continue
            d = _haversine_m(a.lat, a.lon, b.lat, b.lon)

            if d <= a.sensor_range_m and not a.detected.get(b.id):
                a.detected[b.id] = True
                events.append(
                    normalizer.event(start_epoch + t, results_pb2.EVENT_TYPE_DETECTION,
                                     a.id, b.id, "detected %s at %.0f m" % (b.name, d))
                )

            if a.weapon_range_m > 0 and _roe_allows(a.roe) and d <= a.weapon_range_m:
                key = a.id + "|" + b.id
                last = last_engage.get(key)
                if last is not None and t - last < _REENGAGE_COOLDOWN_S:
                    continue
                last_engage[key] = t
                events.append(
                    normalizer.event(start_epoch + t, results_pb2.EVENT_TYPE_ENGAGEMENT,
                                     a.id, b.id, "engaging %s at %.0f m" % (b.name, d))
                )
                pk = translator.capability_for(a.etype).base_pk
                if rng.random() < pk:
                    b.alive = False
                    events.append(
                        normalizer.event(start_epoch + t, results_pb2.EVENT_TYPE_KILL,
                                         a.id, b.id, "killed %s" % b.name)
                    )
                    ts = normalizer.make_timestamp(start_epoch + t)
                    kill_chains.append(
                        results_pb2.KillChain(
                            attacker_entity_id=a.id,
                            target_entity_id=b.id,
                            engaged_at=ts,
                            killed_at=ts,
                        )
                    )


def _record(e, t, tracks, start_epoch) -> None:
    tracks[e.id].points.append(
        normalizer.track_point(
            start_epoch + t, e.lat, e.lon, e.alt_m, e.heading_deg, e.speed_ms,
            "alive" if e.alive else "killed",
        )
    )


def _compute_moes(entities, kill_chains, events):
    """Canonical cross-engine MOE set, see docs/moe-taxonomy.md.

    BuiltinEngine has no partial-health model (binary alive/dead), so
    avg_health_pct degrades to 100/0 per entity per the taxonomy's documented
    fallback for engines without hit points.
    """
    blue_losses = red_losses = blue_kills = red_kills = 0.0
    by_id = {e.id: e for e in entities}
    for e in entities:
        if not e.alive:
            if e.side == entity_pb2.SIDE_FRIENDLY:
                blue_losses += 1
            elif e.side == entity_pb2.SIDE_ENEMY:
                red_losses += 1
    for kc in kill_chains:
        attacker = by_id.get(kc.attacker_entity_id)
        if attacker is None:
            continue
        if attacker.side == entity_pb2.SIDE_FRIENDLY:
            blue_kills += 1
        elif attacker.side == entity_pb2.SIDE_ENEMY:
            red_kills += 1

    detections = sum(1 for e in events if e.type == results_pb2.EVENT_TYPE_DETECTION)
    engagements = sum(1 for e in events if e.type == results_pb2.EVENT_TYPE_ENGAGEMENT)
    avg_health_pct = (
        100.0 * sum(1 for e in entities if e.alive) / len(entities) if entities else 0.0
    )

    return [
        results_pb2.MOEMetric(key="blue_losses", value=blue_losses, unit="entities"),
        results_pb2.MOEMetric(key="red_losses", value=red_losses, unit="entities"),
        results_pb2.MOEMetric(key="blue_kills", value=blue_kills, unit="entities"),
        results_pb2.MOEMetric(key="red_kills", value=red_kills, unit="entities"),
        results_pb2.MOEMetric(key="total_kills", value=float(len(kill_chains)), unit="entities"),
        results_pb2.MOEMetric(key="detections_total", value=float(detections), unit="events"),
        results_pb2.MOEMetric(key="rounds_expended", value=float(engagements), unit="rounds"),
        results_pb2.MOEMetric(key="avg_health_pct", value=avg_health_pct, unit="percent"),
    ]


# --- helpers ---


def _are_enemies(a: int, b: int) -> bool:
    return (a == entity_pb2.SIDE_FRIENDLY and b == entity_pb2.SIDE_ENEMY) or (
        a == entity_pb2.SIDE_ENEMY and b == entity_pb2.SIDE_FRIENDLY
    )


def _roe_allows(roe: int) -> bool:
    return roe in (mission_pb2.ROE_WEAPONS_TIGHT, mission_pb2.ROE_WEAPONS_FREE)


def _seed_from_run_id(run_id: str) -> int:
    return int(hashlib.sha256(run_id.encode("utf-8")).hexdigest()[:16], 16)


def _now_epoch() -> float:
    import time

    return time.time()


def _haversine_m(lat1, lon1, lat2, lon2) -> float:
    p1, p2 = math.radians(lat1), math.radians(lat2)
    dlat = math.radians(lat2 - lat1)
    dlon = math.radians(lon2 - lon1)
    a = math.sin(dlat / 2) ** 2 + math.cos(p1) * math.cos(p2) * math.sin(dlon / 2) ** 2
    return _EARTH_RADIUS_M * 2 * math.atan2(math.sqrt(a), math.sqrt(1 - a))


def _bearing_deg(lat1, lon1, lat2, lon2) -> float:
    p1, p2 = math.radians(lat1), math.radians(lat2)
    dlon = math.radians(lon2 - lon1)
    y = math.sin(dlon) * math.cos(p2)
    x = math.cos(p1) * math.sin(p2) - math.sin(p1) * math.cos(p2) * math.cos(dlon)
    return (math.degrees(math.atan2(y, x)) + 360) % 360


def _move_point(lat, lon, d, bearing):
    ad = d / _EARTH_RADIUS_M
    br = math.radians(bearing)
    p1 = math.radians(lat)
    l1 = math.radians(lon)
    p2 = math.asin(math.sin(p1) * math.cos(ad) + math.cos(p1) * math.sin(ad) * math.cos(br))
    l2 = l1 + math.atan2(
        math.sin(br) * math.sin(ad) * math.cos(p1),
        math.cos(ad) - math.sin(p1) * math.sin(p2),
    )
    return math.degrees(p2), math.degrees(l2)
