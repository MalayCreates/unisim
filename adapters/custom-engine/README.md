# custom-engine adapter

The reference adapter and fallback sim engine. A standalone gRPC sidecar that
implements the `SimAdapter` service from `proto/adapter.proto`.

## What it does

1. Serves the `SimAdapter` gRPC service (`Initialize` / `Run` / `GetResults` / `Shutdown`).
2. Self-registers with the backend on startup (`POST /api/v1/adapters`).
3. On `Run`, executes a lightweight physics model:
   - entities move along ordered waypoints at their defined speeds
   - **detect**: range-based detection within sensor radius
   - **engage**: ROE-gated engagement within weapons radius
   - **kill**: probabilistic kill from a per-type lethality table
4. Returns a normalized `SimResultsProto` (tracks, events, kill chains, MOEs).

Runs are deterministic for a given run ID (the RNG is seeded from it).

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
