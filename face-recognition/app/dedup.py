"""Duplicate event suppression within a configurable time window."""

import threading
import time
from collections import OrderedDict
from typing import Tuple


class DedupFilter:
    """Suppress duplicate events for the same (student, direction) pair.

    Once an event has been emitted for a given ``(student_id, direction)``
    no further event with the same key will be returned until
    ``window_seconds`` have elapsed.

    Thread-safe (uses ``threading.Lock``).
    """

    def __init__(self, config):
        self.config = config
        self._seen: OrderedDict[Tuple[str, str], float] = OrderedDict()
        self._lock = threading.Lock()

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def is_duplicate(self, student_id: str, direction: str) -> bool:
        """Return ``True`` if this (student, direction) was seen recently."""
        key = (student_id, direction)
        with self._lock:
            ts = self._seen.get(key)
            if ts is not None:
                if time.time() - ts < self.config.window_seconds:
                    return True
                # Entry expired — remove so ``mark_seen`` can record fresh
                del self._seen[key]
        return False

    def mark_seen(self, student_id: str, direction: str):
        """Record this (student, direction) with the current timestamp."""
        key = (student_id, direction)
        with self._lock:
            self._seen[key] = time.time()
            self._seen.move_to_end(key)  # LRU: promote to end
            # Enforce maximum cache size (evict oldest)
            while len(self._seen) > self.config.max_cache_size:
                self._seen.popitem(last=False)

    def cleanup(self):
        """Remove all entries older than ``window_seconds``."""
        cutoff = time.time() - self.config.window_seconds
        with self._lock:
            stale = [k for k, ts in self._seen.items() if ts < cutoff]
            for k in stale:
                del self._seen[k]
