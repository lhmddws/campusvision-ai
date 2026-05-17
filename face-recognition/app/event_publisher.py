"""Behavior event publisher for Kafka ``t_dorm_event`` topic.

Publishes behavior/action recognition events (loitering, running, zone intrusion,
crowd alerts) to the ``t_dorm_event`` Kafka topic with the ``source: "behavior"``
field to distinguish them from entry/exit events published by the face-recognition
pipeline.

Features
--------
* Event-type mapping (long descriptive name → short ≤8 char code).
* Assertion enforcement: ``event_type`` **must** be ≤ 8 characters (DB constraint).
* Cooldown per ``event_type`` to avoid flooding Kafka with duplicates.
* In-memory buffer (max 1000) when Kafka is unavailable; auto-flush on next
  successful publish.
* Thread-safe via ``threading.Lock()``.
"""

from __future__ import annotations

import json
import logging
import threading
import time
from typing import TYPE_CHECKING, Any

if TYPE_CHECKING:
    from app.config import AppConfig

logger = logging.getLogger(__name__)


class BehaviorEventPublisher:
    """Publishes behavior events to Kafka with buffering on failure.

    Thread-safe: all public methods coordinate via ``self._lock``.

    Parameters
    ----------
    config : AppConfig
        Application configuration.  Uses ``config.kafka.brokers``,
        ``config.kafka.event_topic``, ``config.behavior.*``.
    """

    def __init__(self, config: AppConfig) -> None:
        self.config = config
        self._lock = threading.Lock()
        self._buffer: list[dict[str, Any]] = []
        self._last_publish_time: dict[str, float] = {}
        self._producer = None

        # Only attempt Kafka connection if behavior is enabled
        if config.kafka.brokers and config.behavior.enabled:
            try:
                from kafka import KafkaProducer

                self._producer = KafkaProducer(
                    bootstrap_servers=config.kafka.brokers,
                    value_serializer=lambda v: json.dumps(v).encode("utf-8"),
                )
                logger.info(
                    "Behavior event publisher connected to Kafka",
                    extra={"brokers": config.kafka.brokers},
                )
            except Exception as exc:
                logger.warning("Kafka unavailable, using buffer: %s", exc)

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    @property
    def topic(self) -> str:
        """Kafka topic for behavior events (default ``t_dorm_event``)."""
        return self.config.kafka.event_topic

    def publish_behavior_event(self, event: dict[str, Any]) -> None:
        """Publish a behavior event to Kafka (or buffer on failure).

        Parameters
        ----------
        event : dict
            Dictionary with at least the keys ``camera_id``, ``event_type``,
            and ``timestamp``.  Optional keys: ``track_id``, ``detail``,
            ``confidence``.

        Raises
        ------
        AssertionError
            If ``event_type`` is longer than 8 characters.
        """
        if not self.config.behavior.enabled:
            return

        # Apply event-type mapping (long descriptive name → short code)
        event_type = event["event_type"]
        mapping = self.config.behavior.event.event_type_mapping
        if event_type in mapping:
            event_type = mapping[event_type]

        # Enforce DB constraint: event_type VARCHAR(8)
        assert len(event_type) <= 8, (
            f"event_type '{event_type}' ({len(event_type)} chars) exceeds 8 char limit"
        )

        now = time.time()

        with self._lock:
            # Cooldown: suppress duplicate event_types within the window
            last_time = self._last_publish_time.get(event_type, 0.0)
            if (now - last_time) < self.config.behavior.event_cooldown_seconds:
                return

            kafka_msg: dict[str, Any] = {
                "camera_id": event["camera_id"],
                "event_type": event_type,
                "track_id": event.get("track_id", ""),
                "detail": event.get("detail", ""),
                "timestamp": event["timestamp"],
                "confidence": event.get("confidence", 1.0),
                "source": "behavior",
            }

            # Try Kafka send; buffer on any failure
            if self._producer is not None:
                try:
                    self._producer.send(self.topic, value=kafka_msg)
                    self._last_publish_time[event_type] = now
                    self._flush_buffer()
                    return
                except Exception:
                    logger.warning("Kafka send failed, buffering event", exc_info=True)

            self._buffer.append(kafka_msg)
            if len(self._buffer) > 1000:
                evicted = self._buffer.pop(0)
                logger.warning(
                    "Buffer full, evicted oldest event",
                    extra={"evicted_type": evicted.get("event_type")},
                )

    def close(self) -> None:
        """Flush buffer and close the Kafka producer."""
        with self._lock:
            if self._producer is not None:
                try:
                    self._flush_buffer()
                    self._producer.flush()
                except Exception:
                    logger.exception("Error flushing producer during close")
                try:
                    self._producer.close()
                except Exception:
                    logger.exception("Error closing producer")
                self._producer = None
                logger.info("Behavior event publisher closed")

    # ------------------------------------------------------------------
    # Internal helpers
    # ------------------------------------------------------------------

    def _flush_buffer(self) -> None:
        """Retry sending all buffered events.

        Events that still fail remain in the buffer.  Called automatically
        after a successful Kafka send from :meth:`publish_behavior_event`.
        """
        if not self._buffer or self._producer is None:
            return
        remaining: list[dict[str, Any]] = []
        for msg in self._buffer:
            try:
                self._producer.send(self.topic, value=msg)
            except Exception:
                remaining.append(msg)

        if remaining:
            flushed = len(self._buffer) - len(remaining)
            logger.info("Flushed %d buffered events, %d remain", flushed, len(remaining))
            self._buffer = remaining
        else:
            logger.info("Flushed all %d buffered events", len(self._buffer))
            self._buffer.clear()
