"""SimAdapter gRPC service for the Panopticon adapter.

Mirrors the custom-engine adapter's contract: Initialize stores the scenario,
Run kicks off the sim asynchronously, GetResults blocks until it finishes.
"""

from __future__ import annotations

import logging
import threading

import grpc

from protostubs import adapter_pb2, adapter_pb2_grpc

log = logging.getLogger("panopticon.adapter")


class _RunState:
    def __init__(self):
        self.done = threading.Event()
        self.results = None
        self.error = None


class PanopticonAdapter(adapter_pb2_grpc.SimAdapterServicer):
    def __init__(self, engine, engine_id: str, version: str):
        self._engine = engine
        self._engine_id = engine_id
        self._version = version
        self._lock = threading.Lock()
        self._scenarios = {}
        self._runs = {}

    def Initialize(self, scenario, context):
        if not scenario.id:
            return adapter_pb2.InitResponse(success=False, message="scenario id is required")
        with self._lock:
            self._scenarios[scenario.id] = scenario
        log.info("Initialize: scenario=%s entities=%d missions=%d",
                 scenario.id, len(scenario.entities), len(scenario.missions))
        return adapter_pb2.InitResponse(
            success=True,
            message='scenario "%s" initialized' % scenario.name,
            adapter_version=self._version,
        )

    def Run(self, req, context):
        with self._lock:
            scenario = self._scenarios.get(req.scenario_id)
            if scenario is None:
                return adapter_pb2.RunResponse(
                    accepted=False, message="scenario not initialized; call Initialize first")
            if req.run_id in self._runs:
                return adapter_pb2.RunResponse(accepted=False, message="run already exists")
            state = _RunState()
            self._runs[req.run_id] = state

        log.info("Run: run=%s scenario=%s starting (engine=%s)",
                 req.run_id, req.scenario_id, self._engine.name)
        threading.Thread(
            target=self._execute, args=(scenario, req.run_id, state), daemon=True
        ).start()
        return adapter_pb2.RunResponse(accepted=True, message="run started")

    def _execute(self, scenario, run_id, state):
        try:
            state.results = self._engine.run(scenario, self._engine_id, run_id)
            log.info("Run: run=%s done — %d tracks, %d events, %d kills",
                     run_id, len(state.results.entity_tracks),
                     len(state.results.events), len(state.results.kill_chains))
        except Exception as exc:  # noqa: BLE001 - surfaced to the caller via GetResults
            log.exception("Run: run=%s failed", run_id)
            state.error = exc
        finally:
            state.done.set()

    def GetResults(self, req, context):
        with self._lock:
            state = self._runs.get(req.run_id)
        if state is None:
            context.abort(grpc.StatusCode.NOT_FOUND, "unknown run %s" % req.run_id)

        # Block until the run completes or the caller gives up.
        timeout = context.time_remaining()
        if not state.done.wait(timeout=timeout):
            context.abort(grpc.StatusCode.DEADLINE_EXCEEDED, "run still in progress")
        if state.error is not None:
            context.abort(grpc.StatusCode.INTERNAL, str(state.error))
        return state.results

    def Shutdown(self, req, context):
        log.info("Shutdown requested (graceful=%s)", req.graceful)
        return adapter_pb2.ShutdownResponse(success=True)
