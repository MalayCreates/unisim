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

### 1. Generate protobuf stubs

```sh
make proto
```

This runs `buf generate proto` and writes Go/Python/TypeScript stubs into:
- `backend/internal/schema/`
- `adapters/panopticon/proto/`
- `frontend/src/proto/`

### 2. Start the backend

```sh
make backend
# Listens on :8080, creates ~/.usip/usip.db
```

### 3. Start the frontend

```sh
make install-frontend
make frontend
# Vite dev server on :5173, proxies /api → :8080
```

### Combined dev

```sh
make dev
```

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
