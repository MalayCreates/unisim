"""Translate a USIP ScenarioProto into the engine's internal model.

The built-in placeholder engine consumes the SimEntity model below. The
`to_panopticon_scenario` function is the seam where a real Panopticon
integration would instead emit Panopticon's own scenario format.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Dict, List

from protostubs import entity_pb2, mission_pb2, scenario_pb2


@dataclass
class Capability:
    sensor_range_m: float
    weapon_range_m: float
    base_pk: float  # probability of kill per engagement


# Default sensor/weapon ranges and lethality by entity type. Mirrors the
# custom-engine's table so the two reference engines are roughly comparable.
CAPABILITY_TABLE: Dict[int, Capability] = {
    entity_pb2.ENTITY_TYPE_FIXED_WING: Capability(80000, 40000, 0.7),
    entity_pb2.ENTITY_TYPE_ROTARY_WING: Capability(30000, 8000, 0.6),
    entity_pb2.ENTITY_TYPE_UAV: Capability(60000, 15000, 0.5),
    entity_pb2.ENTITY_TYPE_GROUND_VEHICLE: Capability(8000, 4000, 0.6),
    entity_pb2.ENTITY_TYPE_SURFACE_VESSEL: Capability(100000, 50000, 0.65),
    entity_pb2.ENTITY_TYPE_SUBMARINE: Capability(40000, 20000, 0.7),
    entity_pb2.ENTITY_TYPE_RADAR_SENSOR: Capability(150000, 0, 0.0),
    entity_pb2.ENTITY_TYPE_BASE_FOB: Capability(50000, 0, 0.0),
    entity_pb2.ENTITY_TYPE_MISSILE: Capability(20000, 10000, 0.85),
}
DEFAULT_CAPABILITY = Capability(10000, 5000, 0.5)


def capability_for(etype: int) -> Capability:
    return CAPABILITY_TABLE.get(etype, DEFAULT_CAPABILITY)


@dataclass
class Waypoint:
    lat: float
    lon: float
    alt_m: float
    speed_ms: float
    hold_time_s: int


@dataclass
class SimEntity:
    id: str
    name: str
    side: int
    etype: int
    lat: float
    lon: float
    alt_m: float
    roe: int
    waypoints: List[Waypoint] = field(default_factory=list)
    # mutable run state
    alive: bool = True
    heading_deg: float = 0.0
    speed_ms: float = 0.0
    wp_index: int = 0
    hold_until_s: float = 0.0
    sensor_range_m: float = 0.0
    weapon_range_m: float = 0.0
    detected: Dict[str, bool] = field(default_factory=dict)


def translate(scenario: "scenario_pb2.ScenarioProto") -> List[SimEntity]:
    """ScenarioProto -> internal SimEntity list with missions attached."""
    mission_by_entity: Dict[str, "mission_pb2.EntityMission"] = {
        m.entity_id: m for m in scenario.missions
    }

    out: List[SimEntity] = []
    for e in scenario.entities:
        cap = capability_for(e.type)
        se = SimEntity(
            id=e.id,
            name=e.name,
            side=e.side,
            etype=e.type,
            lat=e.position.lat,
            lon=e.position.lon,
            alt_m=e.position.alt_m,
            roe=mission_pb2.ROE_WEAPONS_TIGHT,
            sensor_range_m=cap.sensor_range_m,
            weapon_range_m=cap.weapon_range_m,
        )
        m = mission_by_entity.get(e.id)
        if m is not None:
            se.roe = m.roe
            for w in m.waypoints:
                se.waypoints.append(
                    Waypoint(w.lat, w.lon, w.alt_m, w.speed_ms, w.hold_time_s)
                )
        out.append(se)
    return out


def to_panopticon_scenario(scenario: "scenario_pb2.ScenarioProto") -> dict:
    """SEAM: convert ScenarioProto to Panopticon's scenario format.

    Implement this when wiring the real Panopticon engine. Panopticon scenarios
    are JSON documents describing sides, aircraft/ships/facilities, and their
    routes/missions; map ScenarioProto entities + missions onto that schema and
    map our EntityType enum to Panopticon platform types.
    """
    raise NotImplementedError(
        "Panopticon scenario translation is not implemented yet. "
        "The built-in placeholder engine is used instead (see engine.py)."
    )
