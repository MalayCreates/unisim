"""Panopticon adapter entry point: gRPC server + backend self-registration.

Usage:
  python3 adapters/panopticon/main.py \
      --addr :50052 --host localhost --port 50052 \
      --backend http://localhost:8080 [--backend-engine builtin|panopticon]

By default it runs the built-in placeholder engine so the adapter works without
Panopticon installed. The backend's runner manager can also spawn it on demand
(see runner_config.json.example).
"""

from __future__ import annotations

import argparse
import json
import logging
import signal
import threading
import time
import urllib.error
import urllib.request
from concurrent import futures

import grpc

from protostubs import adapter_pb2_grpc
from adapter import PanopticonAdapter
from engine import get_engine

ADAPTER_VERSION = "panopticon/0.1.0"
DEFAULT_ENGINE_ID = "panopticon"

log = logging.getLogger("panopticon.main")


def register_with_backend(backend_url, engine_id, host, port):
    payload = json.dumps(
        {"engine_id": engine_id, "host": host, "port": port, "version": ADAPTER_VERSION}
    ).encode("utf-8")
    url = backend_url.rstrip("/") + "/api/v1/adapters"

    for attempt in range(1, 31):
        try:
            req = urllib.request.Request(
                url, data=payload, headers={"Content-Type": "application/json"}, method="POST"
            )
            with urllib.request.urlopen(req, timeout=3) as resp:
                if resp.status < 300:
                    log.info("registered with backend as %r (%s:%d)", engine_id, host, port)
                    return
        except (urllib.error.URLError, OSError) as exc:
            log.info("registration attempt %d failed (%s); retrying...", attempt, exc)
        time.sleep(2)
    log.warning("could not register with backend at %s after retries; still serving", url)


def main():
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s: %(message)s")
    parser = argparse.ArgumentParser(description="USIP Panopticon adapter")
    parser.add_argument("--addr", default=":50052", help="gRPC listen address")
    parser.add_argument("--host", default="localhost", help="host the backend should dial")
    parser.add_argument("--port", type=int, default=50052, help="port the backend should dial")
    parser.add_argument("--backend", default="http://localhost:8080", help="backend base URL")
    parser.add_argument("--engine-id", default=DEFAULT_ENGINE_ID, help="engine id to register as")
    parser.add_argument(
        "--backend-engine",
        default="builtin",
        choices=["builtin", "panopticon"],
        help="simulation backend (builtin placeholder or real panopticon)",
    )
    args = parser.parse_args()

    engine = get_engine(args.backend_engine)
    if engine.name != "panopticon":
        log.warning(
            "using built-in placeholder engine; real Panopticon integration is a TODO "
            "(translator.to_panopticon_scenario / normalizer.normalize_panopticon)"
        )

    server = grpc.server(futures.ThreadPoolExecutor(max_workers=8))
    adapter_pb2_grpc.add_SimAdapterServicer_to_server(
        PanopticonAdapter(engine, args.engine_id, ADAPTER_VERSION), server
    )
    bind = args.addr if ":" in args.addr else ":" + args.addr
    server.add_insecure_port("[::]" + bind if bind.startswith(":") else bind)
    server.start()
    log.info("panopticon adapter (%s) listening on %s, advertising %s:%d, engine=%s",
             ADAPTER_VERSION, bind, args.host, args.port, engine.name)

    threading.Thread(
        target=register_with_backend,
        args=(args.backend, args.engine_id, args.host, args.port),
        daemon=True,
    ).start()

    stop = threading.Event()
    for sig in (signal.SIGINT, signal.SIGTERM):
        signal.signal(sig, lambda *_: stop.set())
    stop.wait()
    log.info("shutting down gRPC server...")
    server.stop(grace=2).wait()


if __name__ == "__main__":
    main()
