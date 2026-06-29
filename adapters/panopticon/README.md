# panopticon adapter (v2 — placeholder)

Planned Python gRPC adapter for [Panopticon AI](https://github.com/Panopticon-AI-team/panopticon).
**Out of scope for v1** — this directory is a placeholder for the structure.

When implemented it will:
- implement the `SimAdapter` gRPC service from `proto/adapter.proto`
- translate `ScenarioProto` → Panopticon's JSON scenario format (`translator.py`)
- invoke Panopticon's Gymnasium-compatible Python sim engine
- normalize Panopticon output → `SimResultsProto` (`normalizer.py`)
- self-register with the backend on startup, like the custom-engine adapter

Python protobuf stubs will be generated into `adapters/panopticon/proto/` by
adding the Python targets back to `buf.gen.yaml` and running `make proto`.
