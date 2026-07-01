# custom-engine adapter

The reference adapter and fallback sim engine. A standalone gRPC sidecar that
implements the `SimAdapter` service from `proto/adapter.proto`.

## What it does

1. Serves the `SimAdapter` gRPC service (`Initialize` / `Run` / `GetResults` / `Shutdown`).
2. Self-registers with the backend on startup (`POST /api/v1/adapters`).
3. On `Run`, executes a lightweight physics model:
   - entities move along ordered waypoints at their defined speeds
   - **detect**: range-based detection within sensor radius (optionally
     probabilistic, see below)
   - **engage**: ROE-gated, ammo-gated engagement within weapons radius
   - **damage/kill**: probabilistic hit from a per-type lethality table;
     entities with more than 1 HP take repeated hits (`DAMAGE` events) before
     a `KILL`
4. Returns a normalized `SimResultsProto` (tracks, events, kill chains, MOEs).

Runs are deterministic for a given run ID (the RNG is seeded from it).

## Per-entity attribute overrides

`Entity.attributes` (a free-form `map<string,string>`) lets a scenario
override the per-type defaults in `translator.go`'s `capabilityTable` for a
single entity. All keys are optional; an entity with none of them behaves
exactly like the type default (single-hit kill, unlimited ammo, hard
range-cutoff detection).

| Key | Type | Default | Effect |
|---|---|---|---|
| `sensor_range_m` | float | per-type | detection radius, meters |
| `weapon_range_m` | float | per-type | engagement radius, meters |
| `base_pk` | float | per-type | probability of a hit per engagement roll |
| `health_hp` | float | `1` | hits required to kill; each hit emits `DAMAGE` until HP reaches 0, then `KILL` |
| `ammo` | int | unlimited (`-1`) | rounds available; engagement stops once exhausted |
| `sensor_pd_min` | float | unset (hard cutoff) | if set, enables linear probability-of-detection falloff from `1.0` at 0m to this value at `sensor_range_m` |

## Files

| File | Purpose |
|---|---|
| `main.go` | gRPC server + backend self-registration |
| `adapter.go` | `SimAdapter` service implementation (async run + blocking GetResults) |
| `engine.go` | physics model: movement, detect→engage→kill, MOEs, geo helpers |
| `translator.go` | `ScenarioProto` → internal entity/mission structs + capability table |
| `engine_test.go` | engine regression tests |

## Run

```sh
go run . -addr :50051 -host localhost -port 50051 -backend http://localhost:8080
```

Flags: `-addr` (gRPC listen), `-host`/`-port` (coordinates advertised to the
backend), `-backend` (backend base URL for registration).

The shared protobuf types come from the backend module via the repo `go.work`
workspace (and a `replace` in `go.mod`). Run `make proto` from the repo root if
`backend/schema/` is missing.
