"""Build SimResultsProto from engine output.

For the built-in placeholder engine the results are assembled directly in
engine.py using the small helpers here. `normalize_panopticon` is the seam where
a real Panopticon run's observations would be converted into the same proto.
"""

from __future__ import annotations

from google.protobuf.timestamp_pb2 import Timestamp

from protostubs import results_pb2


def make_timestamp(epoch_s: float) -> Timestamp:
    ts = Timestamp()
    ts.seconds = int(epoch_s)
    ts.nanos = int((epoch_s - int(epoch_s)) * 1e9)
    return ts


def track_point(epoch_s, lat, lon, alt_m, heading_deg, speed_ms, status) -> "results_pb2.TrackPoint":
    return results_pb2.TrackPoint(
        ts=make_timestamp(epoch_s),
        lat=lat,
        lon=lon,
        alt_m=alt_m,
        heading_deg=heading_deg,
        speed_ms=speed_ms,
        status=status,
    )


def event(epoch_s, etype, entity_id, target_id="", detail="") -> "results_pb2.SimEvent":
    return results_pb2.SimEvent(
        ts=make_timestamp(epoch_s),
        type=etype,
        entity_id=entity_id,
        target_entity_id=target_id,
        detail=detail,
    )


def normalize_panopticon(observations, scenario_id, engine_id, run_id) -> "results_pb2.SimResultsProto":
    """SEAM: convert Panopticon step observations into SimResultsProto.

    Panopticon's Gymnasium env yields per-step observation arrays (positions,
    health, weapon events). Accumulate them into EntityTracks/SimEvents/
    KillChains/MOEMetrics here, mirroring what engine.py builds for the
    placeholder engine.
    """
    raise NotImplementedError(
        "Panopticon observation normalization is not implemented yet."
    )
