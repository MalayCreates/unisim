"""Loads the generated USIP protobuf stubs from adapters/_proto/.

Run `make proto-py` (or scripts/gen-proto-py.sh) to generate them first. All
adapter modules import the proto types from here so the sys.path wiring lives
in exactly one place.
"""

import os
import sys

_HERE = os.path.dirname(os.path.abspath(__file__))
_PROTO_DIR = os.path.abspath(os.path.join(_HERE, "..", "_proto"))

if not os.path.isdir(_PROTO_DIR):
    raise ImportError(
        "Generated proto stubs not found at %s.\n"
        "Generate them with:  make proto-py   (or ./scripts/gen-proto-py.sh)" % _PROTO_DIR
    )

if _PROTO_DIR not in sys.path:
    sys.path.insert(0, _PROTO_DIR)

import adapter_pb2  # noqa: E402,F401
import adapter_pb2_grpc  # noqa: E402,F401
import entity_pb2  # noqa: E402,F401
import mission_pb2  # noqa: E402,F401
import results_pb2  # noqa: E402,F401
import scenario_pb2  # noqa: E402,F401
