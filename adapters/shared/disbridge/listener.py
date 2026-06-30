"""UDP DIS listener.

Binds a UDP socket (DIS commonly broadcasts on port 3000) and feeds each
received datagram through the decoder into a DISCollector on a background
thread. Used by adapters that drive an external sim (Command: MO, ArmA/VBS4)
which emit DIS Entity State / Fire / Detonation PDUs.

The socket layer is a thin shell over the pure decoder + collector, which are
where the testable logic lives.
"""

from __future__ import annotations

import socket
import threading
import time
from typing import Optional

from .collector import DISCollector, IDResolver
from .pdu import parse_pdu

DEFAULT_DIS_PORT = 3000


class DISListener:
    def __init__(
        self,
        collector: Optional[DISCollector] = None,
        id_resolver: Optional[IDResolver] = None,
        host: str = "0.0.0.0",
        port: int = DEFAULT_DIS_PORT,
    ):
        self.collector = collector or DISCollector(id_resolver)
        self._host = host
        self._port = port
        self._sock: Optional[socket.socket] = None
        self._thread: Optional[threading.Thread] = None
        self._running = threading.Event()

    def start(self) -> None:
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        try:
            sock.setsockopt(socket.SOL_SOCKET, socket.SO_BROADCAST, 1)
        except OSError:
            pass
        sock.bind((self._host, self._port))
        sock.settimeout(0.5)
        self._sock = sock
        self._running.set()
        self._thread = threading.Thread(target=self._loop, name="dis-listener", daemon=True)
        self._thread.start()

    def _loop(self) -> None:
        assert self._sock is not None
        while self._running.is_set():
            try:
                data, _addr = self._sock.recvfrom(8192)
            except socket.timeout:
                continue
            except OSError:
                break
            recv_ms = int(time.time() * 1000)
            pdu = parse_pdu(data)
            if pdu is not None:
                self.collector.on_pdu(pdu, recv_ms)

    def stop(self) -> None:
        self._running.clear()
        if self._thread is not None:
            self._thread.join(timeout=2.0)
        if self._sock is not None:
            self._sock.close()
            self._sock = None

    def results(self):
        return self.collector.results()
