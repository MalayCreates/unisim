# panopticon adapter

A Python gRPC adapter implementing the USIP `SimAdapter` contract
(`proto/adapter.proto`). It self-registers with the backend on startup, exactly
like the Go custom-engine adapter, and can also be spawned on demand by the
backend's runner manager (see `runner_config.json.example`).

## Status

- **Adapter plumbing: working.** Initialize → Run → GetResults → Shutdown over
  gRPC, cross-language with the Go backend. Verified end-to-end against the
  Hormuz scenario.
- **Engine: built-in placeholder.** Ships with a small deterministic reference
  sim (`engine.py: BuiltinEngine`) — waypoint movement + detect/engage/kill —
  so the adapter runs today without Panopticon installed.
- **Real Panopticon engine: TODO.** [`PanopticonEngine`](engine.py) is the seam;
  the translator/normalizer seams are marked below.

## Layout

| File | Role |
|------|------|
| `main.py` | gRPC server, CLI, backend self-registration |
| `adapter.py` | `SimAdapter` servicer (Initialize/Run/GetResults/Shutdown) |
| `engine.py` | `BuiltinEngine` (placeholder) + `PanopticonEngine` (seam) |
| `translator.py` | `ScenarioProto` → internal model; `to_panopticon_scenario` seam |
| `normalizer.py` | results helpers; `normalize_panopticon` seam |
| `protostubs.py` | loads generated proto stubs from `adapters/_proto/` |
| `test_engine.py` | smoke + determinism tests for the built-in engine |

## Setup

```bash
python3 -m pip install -r adapters/panopticon/requirements.txt
make proto-py                      # generate stubs into adapters/_proto/
```

## Run

```bash
# Built-in placeholder engine (default), registering with a local backend:
python3 adapters/panopticon/main.py \
    --addr :50052 --host localhost --port 50052 \
    --backend http://localhost:8080

# Or let the backend spawn it on demand:
cp runner_config.json.example runner_config.json   # has a "panopticon" recipe
# then trigger a run with engine_id "panopticon"
```

Test the engine standalone:

```bash
python3 adapters/panopticon/test_engine.py
```

## Wiring the real Panopticon engine

[Panopticon](https://github.com/Panopticon-AI-team/panopticon) is a
Gymnasium-based multi-domain sim. To use it instead of the placeholder:

1. `pip install gymnasium panopticon` (uncomment in `requirements.txt`).
2. Implement `translator.to_panopticon_scenario` — map `ScenarioProto` entities
   and missions onto Panopticon's JSON scenario schema and platform types.
3. Implement the `gymnasium.make(...)` / `reset` / `step` loop in
   `engine.PanopticonEngine.run` with a scripted action policy derived from the
   missions/waypoints.
4. Implement `normalizer.normalize_panopticon` — accumulate per-step
   observations into `EntityTrack`/`SimEvent`/`KillChain`/`MOEMetric`.
5. Run with `--backend-engine panopticon`.

> Panopticon is lower-fidelity than the eventual Command: MO / VBS4 (ArmA-proxy)
> adapters, which will share the DIS listener in `adapters/shared/disbridge/`.
> Panopticon emits its own observation arrays, not DIS, so it has a bespoke
> normalizer rather than using that shared path.
