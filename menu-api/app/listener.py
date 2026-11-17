"""Background LISTEN on Postgres 'menu_changed' notifications to invalidate the cache."""

from __future__ import annotations

import logging
import threading
from collections.abc import Callable

import psycopg

log = logging.getLogger(__name__)

CHANNEL = "menu_changed"


class MenuChangeListener:
    def __init__(self, conninfo: str, on_change: Callable[[], None]) -> None:
        self._conninfo = conninfo
        self._on_change = on_change
        self._stop = threading.Event()
        self._thread = threading.Thread(target=self._run, name="menu-listener", daemon=True)

    def start(self) -> None:
        self._thread.start()

    def stop(self) -> None:
        self._stop.set()
        self._thread.join(timeout=5)

    def _run(self) -> None:
        backoff = 1.0
        while not self._stop.is_set():
            try:
                with psycopg.connect(self._conninfo, autocommit=True) as conn:
                    conn.execute(f"LISTEN {CHANNEL}")
                    log.info("listening for %s notifications", CHANNEL)
                    # Anything may have changed while we were disconnected.
                    self._on_change()
                    backoff = 1.0
                    while not self._stop.is_set():
                        changed = False
                        for _ in conn.notifies(timeout=1.0):
                            changed = True
                        if changed:
                            self._on_change()
            except psycopg.Error as exc:
                log.warning("menu listener disconnected: %s; retrying in %.0fs", exc, backoff)
                self._stop.wait(backoff)
                backoff = min(backoff * 2, 30.0)
