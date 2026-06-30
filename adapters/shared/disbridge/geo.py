"""WGS-84 geodetic <-> ECEF conversions.

DIS carries entity positions and velocities in Earth-Centered, Earth-Fixed
(ECEF) coordinates. USIP works in geodetic lat/lon/alt plus ground speed and
heading, so the listener converts on the way in. These functions are pure and
dependency-free so they can be unit-tested directly.
"""

from __future__ import annotations

import math
from typing import Tuple

# WGS-84 ellipsoid constants.
_A = 6378137.0  # semi-major axis (m)
_F = 1.0 / 298.257223563  # flattening
_B = _A * (1.0 - _F)  # semi-minor axis
_E2 = _F * (2.0 - _F)  # first eccentricity squared
_EP2 = (_A * _A - _B * _B) / (_B * _B)  # second eccentricity squared


def geodetic_to_ecef(lat_deg: float, lon_deg: float, alt_m: float) -> Tuple[float, float, float]:
    """Convert geodetic lat/lon/alt to ECEF X/Y/Z (meters)."""
    lat = math.radians(lat_deg)
    lon = math.radians(lon_deg)
    sin_lat = math.sin(lat)
    cos_lat = math.cos(lat)
    n = _A / math.sqrt(1.0 - _E2 * sin_lat * sin_lat)
    x = (n + alt_m) * cos_lat * math.cos(lon)
    y = (n + alt_m) * cos_lat * math.sin(lon)
    z = (n * (1.0 - _E2) + alt_m) * sin_lat
    return x, y, z


def ecef_to_geodetic(x: float, y: float, z: float) -> Tuple[float, float, float]:
    """Convert ECEF X/Y/Z (meters) to geodetic lat/lon (degrees) and alt (m).

    Uses Bowring's closed-form approximation, which is accurate to well under a
    millimeter for near-surface points.
    """
    lon = math.atan2(y, x)
    p = math.hypot(x, y)
    if p == 0.0:  # at a pole
        lat = math.copysign(math.pi / 2.0, z)
        alt = abs(z) - _B
        return math.degrees(lat), math.degrees(lon), alt

    theta = math.atan2(z * _A, p * _B)
    sin_t = math.sin(theta)
    cos_t = math.cos(theta)
    lat = math.atan2(z + _EP2 * _B * sin_t**3, p - _E2 * _A * cos_t**3)
    sin_lat = math.sin(lat)
    n = _A / math.sqrt(1.0 - _E2 * sin_lat * sin_lat)
    alt = p / math.cos(lat) - n
    return math.degrees(lat), math.degrees(lon), alt


def ecef_velocity_to_ground(
    vx: float, vy: float, vz: float, lat_deg: float, lon_deg: float
) -> Tuple[float, float]:
    """Project an ECEF velocity into the local ENU frame at lat/lon.

    Returns (ground_speed_ms, heading_deg) where heading is degrees clockwise
    from true north (0..360).
    """
    lat = math.radians(lat_deg)
    lon = math.radians(lon_deg)
    sin_lat, cos_lat = math.sin(lat), math.cos(lat)
    sin_lon, cos_lon = math.sin(lon), math.cos(lon)

    east = -sin_lon * vx + cos_lon * vy
    north = -sin_lat * cos_lon * vx - sin_lat * sin_lon * vy + cos_lat * vz

    ground_speed = math.hypot(east, north)
    heading = (math.degrees(math.atan2(east, north)) + 360.0) % 360.0
    return ground_speed, heading
