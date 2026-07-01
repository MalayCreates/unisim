# MOE key taxonomy

`MOEMetric{key, value, unit}` (`proto/results.proto`) is intentionally
engine-agnostic — any adapter can emit any key. This doc is the shared
vocabulary so different engines describe the same concept with the same key,
unit, and meaning, and so the frontend can label/group known keys instead of
just showing raw snake_case. It is a convention, not a schema: an engine is
free to emit additional engine-specific keys beyond this table, and the
frontend falls back to a prettified raw key for anything it doesn't recognize.

| Key | Category | Unit | Meaning |
|---|---|---|---|
| `blue_losses` | attrition | `entities` | Friendly entities destroyed by run end |
| `red_losses` | attrition | `entities` | Enemy entities destroyed by run end |
| `blue_kills` | effectiveness | `entities` | Kills credited to friendly attackers |
| `red_kills` | effectiveness | `entities` | Kills credited to enemy attackers |
| `total_kills` | effectiveness | `entities` | Total kill chains recorded |
| `detections_total` | sensor | `events` | Total detection events across all entities |
| `rounds_expended` | logistics | `rounds` | Total engagement (shot) events fired across all entities |
| `avg_health_pct` | attrition | `percent` | Mean remaining health across all entities at run end |

Notes:
- `avg_health_pct` is 0 for a dead entity and 100 for an entity at full
  starting health, averaged across all entities. Engines without a partial-health
  model (only binary alive/dead) report each entity as exactly 0 or 100 — the
  metric degrades gracefully rather than requiring every engine to model
  hit points.
- `rounds_expended` counts fired shots (engagement attempts), not hits —
  it's independent of whether ammo is finite or unlimited for the firing
  entity.
- Currently emitted by: `adapters/custom-engine/engine.go:computeMOEs()` and
  `adapters/panopticon/engine.py:_compute_moes()` (the `BuiltinEngine`
  placeholder; the real Panopticon seam should emit the same keys once
  implemented).
- Adding a new cross-engine metric: add a row here first, then implement it
  in each engine that can compute it. Keys are flat snake_case (no category
  prefix) for backward compatibility with the original five keys; the
  "Category" column here is what the frontend uses to group metrics for
  display (see `frontend/src/lib/moeTaxonomy.ts`).
- Batch runs (`POST /api/v1/scenarios/{id}/batches`, see root `README.md`)
  aggregate these same keys across N replications (mean/stddev/min/max per
  key) — the taxonomy and its category grouping apply identically there.
