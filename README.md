# USIP — Unified Simulation Integration Platform

Web front end and orchestration layer for multiple military simulation engines (AFSIM, STORM, JCATS, Panopticon, and others).

## Quick start

### Prerequisites

| Tool | Install |
|---|---|
| Go 1.21+ | `brew install go` |
| Node 20+ | `brew install node` |
| buf (proto codegen) | `brew install bufbuild/buf/buf` |
| SQLite | bundled with `go-sqlite3` (requires cgo / Xcode CLT) |

> `buf` has prebuilt binaries if Homebrew is blocked by outdated Xcode CLT:
> `curl -sSL https://github.com/bufbuild/buf/releases/latest/download/buf-Darwin-arm64 -o $(go env GOPATH)/bin/buf && chmod +x $(go env GOPATH)/bin/buf`
> (the Go protoc plugins also need to be on PATH —
> `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest` and
> `go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest`).

### 1. One-time setup

```sh
make proto             # generate Go stubs into backend/schema/ (gitignored)
make install-frontend  # npm install in frontend/
```

### 2. Run everything

```sh
make dev               # backend (:8080) + custom-engine adapter (:50051) + frontend (:5173)
```

### 3. Load a sample scenario

In a second terminal, once `make dev` is up:

```sh
make seed              # POSTs scripts/sample-scenario.json to the backend
```

Then open **http://localhost:5173**, pick *“Sample — Coastal Strike (San Diego)”*
from the scenario dropdown, and click **Run**. Two blue fighters strike a pair of
red destroyers — scrub the playback bar to watch the tracks and review MOEs,
events, and kill chains.

### Backend only (API / seed testing, no frontend)

```sh
make stack             # backend + adapter together
make seed              # in another terminal
```

To start pieces individually: `make backend`, `make adapter`, `make frontend`.

## Architecture

Six-layer design: React+CesiumJS frontend → Go REST/WebSocket backend → gRPC adapter sidecars → sim engines.

See [USIP_PROJECT_BRIEF.md](../Downloads/USIP_PROJECT_BRIEF.md) for full architecture documentation.

## Proto schema

All five `.proto` files live in `proto/` and are the canonical source of truth for every data structure exchanged between layers.

| File | Purpose |
|---|---|
| `entity.proto` | Unit types, positions, attributes |
| `scenario.proto` | Scenario envelope, sides, timeline |
| `mission.proto` | Per-entity sortie params, ROE, waypoints |
| `results.proto` | Entity tracks, events, kill chains, MOEs |
| `adapter.proto` | gRPC service every sim engine adapter must implement |

## v1 scope

- [x] Proto schema (all 5 files)
- [x] Go backend scaffold + SQLite store
- [x] REST API (scenario CRUD, run lifecycle, results)
- [x] Frontend scaffold (Vite + React + CesiumJS + Zustand)
- [x] Custom engine adapter (Go — reference implementation, gRPC `SimAdapter`)
- [x] Adapter registry (dynamic registration, no hardcoded engine list)
- [x] Orchestrator (registry lookup → gRPC Initialize/Run/GetResults → normalize → persist)
- [x] Frontend UI (Mantine): entity placement, mission config, run trigger, playback, results
- [x] Results + playback (entity track scrubbing on Cesium)
- [x] Mission config panels (mission type, ROE, waypoints, objectives)
- [x] Layer/engine filter controls (domain layers + engine result filter)

**v1 is feature-complete.** Place units on the globe, configure missions, run a
sim, and scrub the resulting tracks — all wired to the Go backend.

## End-to-end run flow (working today)

```
POST /api/v1/scenarios            create a scenario (entities + missions)
POST /api/v1/scenarios/{id}/runs  trigger a run on an engine
        │
        ▼ orchestrator
  registry lookup → gRPC dial adapter → Initialize → Run → GetResults
        │
        ▼ normalizer → SQLite
GET  /api/v1/runs/{runID}          poll status (pending→running→completed)
GET  /api/v1/runs/{runID}/results  entity tracks, events, kill chains, MOEs
```

The custom-engine adapter is a standalone gRPC sidecar that self-registers with
the backend on startup (`POST /api/v1/adapters`). Run a full local stack with
`make dev`, or just the backend + adapter and drive it over REST.

### Batches (Monte Carlo replications)

For stochastic scenarios, run the same scenario N times and get cross-run
statistics instead of a single noisy outcome:

```
POST /api/v1/scenarios/{id}/batches  {"engine_id": "...", "count": N}  -> {batch_id, run_ids}
GET  /api/v1/batches/{batchID}       per-run status + aggregated_moes (mean/stddev/min/max per MOE key)
```

Each replication is a normal run under the hood (own run ID, own RNG seed,
scheduled through the same queue), just tagged with a shared `batch_id`. The
frontend exposes this as a "Replications" count next to the Run button.
