"""Accumulate decoded DIS PDUs into USIP-shaped simulation results.

The collector is deliberately schema-agnostic: it emits plain dataclasses, not
generated protobuf types, so it can be reused outside USIP and unit-tested
without the proto stubs. Adapters convert Results into SimResultsProto.

Entity identity: DIS keys entities by (site, application, entity) and also
carries an 11-char "marking" string. The collector resolves both to a stable
USIP entity id via an id_resolver callback so tracks/events line up with the
scenario the adapter injected.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Callable, Dict, List, Optional

from .pdu import (
    DetonationPDU,
    EntityStatePDU,
    FirePDU,
    PDU,
    entity_key,
)

# id_resolver(entity_key, marking) -> USIP entity id.
IDResolver = Callable[[str, str], str]


def default_resolver(key: str, marking: str) -> str:
    """Prefer the human-readable marking; fall back to the numeric key."""
    return marking or key


@dataclass
class TrackPoint:
    timestamp_ms: int
    lat: float
    lon: float
    alt_m: float
    heading_deg: float
    speed_ms: float
    status: str  # "alive" | "killed"


@dataclass
class EntityTrack:
    entity_id: str
    points: List[TrackPoint] = field(default_factory=list)


@dataclass
class SimEvent:
    timestamp_ms: int
    type: str  # "engagement" | "kill" | "damage"
    entity_id: str
    target_entity_id: str = ""
    detail: str = ""


@dataclass
class KillChain:
    attacker_entity_id: str
    target_entity_id: str
    engaged_at_ms: int
    killed_at_ms: int


@dataclass
class Results:
    entity_tracks: List[EntityTrack] = field(default_factory=list)
    events: List[SimEvent] = field(default_factory=list)
    kill_chains: List[KillChain] = field(default_factory=list)


class DISCollector:
    def __init__(self, id_resolver: Optional[IDResolver] = None):
        self._resolve = id_resolver or default_resolver
        self._tracks: Dict[str, EntityTrack] = {}
        self._track_order: List[str] = []
        self._events: List[SimEvent] = []
        self._kill_chains: List[KillChain] = []
        # last engagement time per "attacker->target", to pair into kill chains.
        self._engaged_at: Dict[str, int] = {}
        # marking learned per numeric key from Entity State PDUs. Fire and
        # Detonation PDUs reference entities by numeric id only, so we backfill
        # the learned marking to keep ids consistent across PDU types.
        self._marking_by_key: Dict[str, str] = {}

    def on_pdu(self, pdu: PDU, recv_ms: int) -> None:
        if isinstance(pdu, EntityStatePDU):
            self._on_entity_state(pdu, recv_ms)
        elif isinstance(pdu, FirePDU):
            self._on_fire(pdu, recv_ms)
        elif isinstance(pdu, DetonationPDU):
            self._on_detonation(pdu, recv_ms)

    def _id(self, eid, marking: str = "") -> str:
        key = entity_key(eid)
        if not marking:
            marking = self._marking_by_key.get(key, "")
        return self._resolve(key, marking)

    def _on_entity_state(self, p: EntityStatePDU, recv_ms: int) -> None:
        if p.marking:
            self._marking_by_key[entity_key(p.entity_id)] = p.marking
        eid = self._id(p.entity_id, p.marking)
        track = self._tracks.get(eid)
        if track is None:
            track = EntityTrack(entity_id=eid)
            self._tracks[eid] = track
            self._track_order.append(eid)
        track.points.append(
            TrackPoint(
                timestamp_ms=recv_ms,
                lat=p.lat,
                lon=p.lon,
                alt_m=p.alt_m,
                heading_deg=p.heading_deg,
                speed_ms=p.speed_ms,
                status="killed" if p.destroyed else "alive",
            )
        )

    def _on_fire(self, p: FirePDU, recv_ms: int) -> None:
        attacker = self._id(p.firing_id)
        target = self._id(p.target_id)
        self._engaged_at["%s->%s" % (attacker, target)] = recv_ms
        self._events.append(
            SimEvent(
                timestamp_ms=recv_ms,
                type="engagement",
                entity_id=attacker,
                target_entity_id=target,
                detail="fired on %s" % target,
            )
        )

    def _on_detonation(self, p: DetonationPDU, recv_ms: int) -> None:
        attacker = self._id(p.firing_id)
        target = self._id(p.target_id)
        if p.is_hit:
            self._events.append(
                SimEvent(
                    timestamp_ms=recv_ms,
                    type="kill",
                    entity_id=attacker,
                    target_entity_id=target,
                    detail="destroyed %s" % target,
                )
            )
            engaged = self._engaged_at.get("%s->%s" % (attacker, target), recv_ms)
            self._kill_chains.append(
                KillChain(
                    attacker_entity_id=attacker,
                    target_entity_id=target,
                    engaged_at_ms=engaged,
                    killed_at_ms=recv_ms,
                )
            )
        else:
            self._events.append(
                SimEvent(
                    timestamp_ms=recv_ms,
                    type="damage",
                    entity_id=attacker,
                    target_entity_id=target,
                    detail="missed %s (result %d)" % (target, p.result),
                )
            )

    def results(self) -> Results:
        return Results(
            entity_tracks=[self._tracks[k] for k in self._track_order],
            events=list(self._events),
            kill_chains=list(self._kill_chains),
        )
