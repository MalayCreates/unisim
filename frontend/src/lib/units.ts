import type { Entity, SimResults, Side } from '../types';

// Units are an opt-in, organizational grouping: an entity belongs to a unit
// when its attributes carry a non-empty `unit_id`. Nothing in the sim engine
// interprets units — per-unit rollups are derived here, on the frontend, from
// the same results (tracks/events/kill chains) the UI already has.

export const UNIT_ATTR = 'unit_id';

export function entityUnitId(e: Entity): string | undefined {
  const u = e.attributes?.[UNIT_ATTR]?.trim();
  return u ? u : undefined;
}

export interface UnitSummary {
  unitId: string;
  side: Side; // taken from the first member; units are normally single-side
  total: number;
  alive: number;
  losses: number;
  damaged: number;
  kills: number;
  rounds: number;
  detections: number;
  strengthPct: number; // alive / total * 100
}

// computeUnitSummaries derives a per-unit rollup from a completed run. Returns
// [] when no entity is assigned to a unit.
export function computeUnitSummaries(entities: Entity[], results: SimResults): UnitSummary[] {
  const unitByEntity = new Map<string, string>();
  for (const e of entities) {
    const u = entityUnitId(e);
    if (u) unitByEntity.set(e.id, u);
  }
  if (unitByEntity.size === 0) return [];

  // Final status per entity = the last track point's status.
  const finalStatus = new Map<string, string>();
  for (const tr of results.entity_tracks) {
    const pts = tr.points;
    if (pts.length > 0) finalStatus.set(tr.entity_id, pts[pts.length - 1].status);
  }

  const byUnit = new Map<string, UnitSummary>();
  for (const e of entities) {
    const u = entityUnitId(e);
    if (!u) continue;
    let s = byUnit.get(u);
    if (!s) {
      s = {
        unitId: u,
        side: e.side,
        total: 0,
        alive: 0,
        losses: 0,
        damaged: 0,
        kills: 0,
        rounds: 0,
        detections: 0,
        strengthPct: 0,
      };
      byUnit.set(u, s);
    }
    s.total++;
    const status = finalStatus.get(e.id) ?? 'alive';
    if (status === 'killed') {
      s.losses++;
    } else {
      s.alive++;
      if (status === 'damaged') s.damaged++;
    }
  }

  for (const kc of results.kill_chains) {
    const u = unitByEntity.get(kc.attacker_entity_id);
    if (u) byUnit.get(u)!.kills++;
  }

  for (const ev of results.events) {
    const u = unitByEntity.get(ev.entity_id);
    if (!u) continue;
    const s = byUnit.get(u)!;
    if (ev.type === 'engagement') s.rounds++;
    else if (ev.type === 'detection') s.detections++;
  }

  for (const s of byUnit.values()) {
    s.strengthPct = s.total > 0 ? (100 * s.alive) / s.total : 0;
  }

  return [...byUnit.values()].sort((a, b) => a.unitId.localeCompare(b.unitId));
}
