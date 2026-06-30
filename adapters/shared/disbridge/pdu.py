"""Minimal IEEE 1278.1 (DIS) PDU decoder.

Decodes only the three PDU types USIP needs to reconstruct simulation results:

  * Entity State (type 1)  -> position / velocity / status track points
  * Fire (type 2)          -> engagement events
  * Detonation (type 3)    -> kill / miss events

DIS is big-endian ("network order"). All offsets below are relative to the
start of the PDU; the common 12-byte header precedes every body. This is a
focused decoder, not a full DIS stack — unknown PDU types decode to None.
"""

from __future__ import annotations

import struct
from dataclasses import dataclass, field
from typing import List, Optional, Tuple, Union

from . import geo

# PDU type enumerations we handle.
PDU_ENTITY_STATE = 1
PDU_FIRE = 2
PDU_DETONATION = 3

_HEADER = struct.Struct(">BBBBIHBB")  # 12 bytes
HEADER_LEN = _HEADER.size

EntityID = Tuple[int, int, int]  # (site, application, entity)


def entity_key(eid: EntityID) -> str:
    """Stable string key for an entity id, e.g. "1:1:42"."""
    return "%d:%d:%d" % eid


@dataclass
class Header:
    protocol_version: int
    exercise_id: int
    pdu_type: int
    protocol_family: int
    timestamp: int
    length: int


@dataclass
class EntityStatePDU:
    header: Header
    entity_id: EntityID
    marking: str  # human-readable 11-char name, if present
    lat: float
    lon: float
    alt_m: float
    speed_ms: float
    heading_deg: float
    destroyed: bool


@dataclass
class FirePDU:
    header: Header
    firing_id: EntityID
    target_id: EntityID


@dataclass
class DetonationPDU:
    header: Header
    firing_id: EntityID
    target_id: EntityID
    result: int  # raw DIS detonation-result code

    @property
    def is_hit(self) -> bool:
        # 1 = Entity Impact, 2 = Entity Proximate Detonation.
        return self.result in (1, 2)


PDU = Union[EntityStatePDU, FirePDU, DetonationPDU]


def parse_header(data: bytes) -> Optional[Header]:
    if len(data) < HEADER_LEN:
        return None
    pv, ex, pt, pf, ts, ln, _status, _pad = _HEADER.unpack_from(data, 0)
    return Header(pv, ex, pt, pf, ts, ln)


def _eid(data: bytes, off: int) -> EntityID:
    return struct.unpack_from(">HHH", data, off)


def parse_pdu(data: bytes) -> Optional[PDU]:
    """Decode a single PDU. Returns None for unsupported types or short buffers."""
    h = parse_header(data)
    if h is None:
        return None
    if h.pdu_type == PDU_ENTITY_STATE:
        return _parse_entity_state(data, h)
    if h.pdu_type == PDU_FIRE:
        return _parse_fire(data, h)
    if h.pdu_type == PDU_DETONATION:
        return _parse_detonation(data, h)
    return None


def _parse_entity_state(data: bytes, h: Header) -> Optional[EntityStatePDU]:
    # Fixed portion runs through capabilities at offset 144.
    if len(data) < 144:
        return None
    eid = _eid(data, 12)
    vx, vy, vz = struct.unpack_from(">fff", data, 36)
    px, py, pz = struct.unpack_from(">ddd", data, 48)
    appearance = struct.unpack_from(">I", data, 84)[0]

    charset = data[128]
    raw = data[129:140]
    marking = raw.decode("ascii", "ignore").rstrip("\x00 ").strip() if charset in (0, 1) else ""

    lat, lon, alt = geo.ecef_to_geodetic(px, py, pz)
    speed, heading = geo.ecef_velocity_to_ground(vx, vy, vz, lat, lon)
    # Appearance bits 3-4 hold the damage state; 3 = destroyed.
    destroyed = ((appearance >> 3) & 0x3) == 0x3

    return EntityStatePDU(
        header=h,
        entity_id=eid,
        marking=marking,
        lat=lat,
        lon=lon,
        alt_m=alt,
        speed_ms=speed,
        heading_deg=heading,
        destroyed=destroyed,
    )


def _parse_fire(data: bytes, h: Header) -> Optional[FirePDU]:
    if len(data) < 96:
        return None
    return FirePDU(header=h, firing_id=_eid(data, 12), target_id=_eid(data, 18))


def _parse_detonation(data: bytes, h: Header) -> Optional[DetonationPDU]:
    if len(data) < 104:
        return None
    result = data[100]
    return DetonationPDU(
        header=h,
        firing_id=_eid(data, 12),
        target_id=_eid(data, 18),
        result=result,
    )


# --- encoders (used by tests and by adapters that need to emit PDUs) ---


def encode_entity_state(
    eid: EntityID,
    lat_deg: float,
    lon_deg: float,
    alt_m: float,
    velocity_ecef: Tuple[float, float, float] = (0.0, 0.0, 0.0),
    marking: str = "",
    destroyed: bool = False,
    exercise_id: int = 1,
) -> bytes:
    """Build a minimal Entity State PDU (144-byte fixed portion)."""
    px, py, pz = geo.geodetic_to_ecef(lat_deg, lon_deg, alt_m)
    buf = bytearray(144)
    appearance = (0x3 << 3) if destroyed else 0
    struct.pack_into(">HHH", buf, 12, *eid)
    struct.pack_into(">fff", buf, 36, *velocity_ecef)
    struct.pack_into(">ddd", buf, 48, px, py, pz)
    struct.pack_into(">I", buf, 84, appearance)
    buf[128] = 1  # ASCII marking charset
    name = marking.encode("ascii", "ignore")[:11]
    buf[129 : 129 + len(name)] = name
    _write_header(buf, PDU_ENTITY_STATE, 1, len(buf), exercise_id)
    return bytes(buf)


def encode_fire(firing: EntityID, target: EntityID, exercise_id: int = 1) -> bytes:
    buf = bytearray(96)
    struct.pack_into(">HHH", buf, 12, *firing)
    struct.pack_into(">HHH", buf, 18, *target)
    _write_header(buf, PDU_FIRE, 2, len(buf), exercise_id)
    return bytes(buf)


def encode_detonation(
    firing: EntityID, target: EntityID, result: int, exercise_id: int = 1
) -> bytes:
    buf = bytearray(104)
    struct.pack_into(">HHH", buf, 12, *firing)
    struct.pack_into(">HHH", buf, 18, *target)
    buf[100] = result & 0xFF
    _write_header(buf, PDU_DETONATION, 2, len(buf), exercise_id)
    return bytes(buf)


def _write_header(buf: bytearray, pdu_type: int, family: int, length: int, exercise_id: int) -> None:
    _HEADER.pack_into(buf, 0, 6, exercise_id, pdu_type, family, 0, length, 0, 0)
