"""Shared, dependency-free DIS (IEEE 1278.1) listener for USIP adapters.

Decodes Entity State / Fire / Detonation PDUs into USIP-shaped tracks, events,
and kill chains. Reused by any adapter fronting a DIS-capable simulator
(Command: Modern Operations, ArmA/VBS4, JCATS, ...).
"""

from .collector import (
    DISCollector,
    EntityTrack,
    KillChain,
    Results,
    SimEvent,
    TrackPoint,
    default_resolver,
)
from .listener import DEFAULT_DIS_PORT, DISListener
from .pdu import (
    DetonationPDU,
    EntityStatePDU,
    FirePDU,
    Header,
    encode_detonation,
    encode_entity_state,
    encode_fire,
    entity_key,
    parse_pdu,
)

__all__ = [
    "DISCollector",
    "DISListener",
    "DEFAULT_DIS_PORT",
    "EntityTrack",
    "KillChain",
    "Results",
    "SimEvent",
    "TrackPoint",
    "default_resolver",
    "DetonationPDU",
    "EntityStatePDU",
    "FirePDU",
    "Header",
    "parse_pdu",
    "entity_key",
    "encode_entity_state",
    "encode_fire",
    "encode_detonation",
]
