import type { EntityTrack, SimResults } from '../types';

export interface InterpolatedPoint {
  lat: number;
  lon: number;
  alt_m: number;
  heading_deg: number;
  status: string;
}

// resultsTimeRange returns the absolute [start, end] timestamps (ms) spanning
// every track point in the results, plus the total duration.
export function resultsTimeRange(results: SimResults): {
  startMs: number;
  endMs: number;
  durationMs: number;
} {
  let startMs = Infinity;
  let endMs = -Infinity;
  for (const track of results.entity_tracks) {
    for (const p of track.points) {
      if (p.timestamp_ms < startMs) startMs = p.timestamp_ms;
      if (p.timestamp_ms > endMs) endMs = p.timestamp_ms;
    }
  }
  if (!isFinite(startMs)) {
    startMs = 0;
    endMs = 0;
  }
  return { startMs, endMs, durationMs: Math.max(0, endMs - startMs) };
}

// interpolateTrack returns the entity's position at absolute time absMs by
// linearly interpolating between surrounding track points. Returns null if the
// track has no points.
export function interpolateTrack(track: EntityTrack, absMs: number): InterpolatedPoint | null {
  const pts = track.points;
  if (pts.length === 0) return null;
  if (absMs <= pts[0].timestamp_ms) return toPoint(pts[0]);
  if (absMs >= pts[pts.length - 1].timestamp_ms) return toPoint(pts[pts.length - 1]);

  // Binary search for the segment containing absMs.
  let lo = 0;
  let hi = pts.length - 1;
  while (hi - lo > 1) {
    const mid = (lo + hi) >> 1;
    if (pts[mid].timestamp_ms <= absMs) lo = mid;
    else hi = mid;
  }
  const a = pts[lo];
  const b = pts[hi];
  const span = b.timestamp_ms - a.timestamp_ms || 1;
  const f = (absMs - a.timestamp_ms) / span;
  return {
    lat: a.lat + (b.lat - a.lat) * f,
    lon: a.lon + (b.lon - a.lon) * f,
    alt_m: a.alt_m + (b.alt_m - a.alt_m) * f,
    heading_deg: a.heading_deg,
    // Status is a step function — use the earlier point's status.
    status: a.status,
  };
}

function toPoint(p: EntityTrack['points'][number]): InterpolatedPoint {
  return {
    lat: p.lat,
    lon: p.lon,
    alt_m: p.alt_m,
    heading_deg: p.heading_deg,
    status: p.status,
  };
}
