"""Behavior analysis — loitering, running, zone intrusion, and crowd detection."""

import math
import time
from typing import Dict, List, Optional

import cv2
import numpy as np

from app.config import BehaviorConfig
from app.tracker import Track


class BehaviorAnalyzer:
    """Analyze face tracks for suspicious or noteworthy behavior.

    Currently supports:

    * **Loitering** — a person stays within a small radius for longer than a
      configurable threshold.
    * **Running** — a person moves faster than a configurable speed threshold.
    * **Zone intrusion** — tracks detected inside configured zone polygons.
    * **Crowd alert** — debounce-based detection when face count stays at or
      above threshold for multiple consecutive frames.

    Events are automatically de-duplicated via a per-(event_type, track_id)
    cooldown cache.
    """

    def __init__(self, config: BehaviorConfig):
        self.config = config

        # Cooldown map: key = "{event_type}:{track_id}", value = timestamp
        self._cooldowns: Dict[str, float] = {}

        # Crowd detection state
        self._crowd_frames: List[float] = []
        self._crowd_active: bool = False

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def analyze(
        self,
        tracks: List[Track],
        frame_face_count: int,
        timestamp: Optional[float] = None,
    ) -> List[dict]:
        """Analyze tracks for behavior events.

        Parameters
        ----------
        tracks:
            Active face tracks from the tracker.
        frame_face_count:
            Number of faces detected in the current frame (used for crowd
            detection).
        timestamp:
            Current timestamp in seconds.  Defaults to ``time.time()``.

        Returns
        -------
        List[dict]
            Event dictionaries with keys ``event_type``, ``camera_id``,
            ``track_id``, ``detail``, ``timestamp``, ``confidence``.
        """
        if not self.config.enabled:
            return []

        now = timestamp if timestamp is not None else time.time()
        events: List[dict] = []

        events.extend(self._check_loitering(tracks, now))
        events.extend(self._check_running(tracks, now))
        events.extend(self._check_zone_intrusion(tracks))
        events.extend(self._check_crowd(frame_face_count, now))

        return events

    # ------------------------------------------------------------------
    # Per-behavior checks
    # ------------------------------------------------------------------

    def _check_loitering(self, tracks: List[Track], now: float) -> List[dict]:
        """Detect tracks that have remained within a small radius too long.

        A track is considered loitering when:

        1. It has been observed for at least ``loitering_threshold_seconds``.
        2. The distance between its earliest and latest positions does not
           exceed ``loitering_radius_px``.
        """
        events: List[dict] = []
        threshold = self.config.loitering_threshold_seconds
        radius = self.config.loitering_radius_px
        event_type = self.config.event.event_type_mapping.get(
            "loitering", "loiter"
        )

        for track in tracks:
            if len(track.timestamps) < 2 or len(track.positions) < 2:
                continue

            duration = track.timestamps[-1] - track.timestamps[0]
            if duration < threshold:
                continue

            cx0, cy0 = track.positions[0]
            cx1, cy1 = track.positions[-1]
            distance = math.hypot(cx1 - cx0, cy1 - cy0)

            if distance <= radius:
                if self._is_on_cooldown(event_type, track.track_id, now):
                    continue
                self._mark_cooldown(event_type, track.track_id, now)
                events.append(
                    self._make_event(
                        event_type=event_type,
                        track=track,
                        detail=(
                            f"Loitering detected: stayed {duration:.1f}s "
                            f"within {distance:.0f}px radius"
                        ),
                        timestamp=now,
                    )
                )

        return events

    def _check_running(self, tracks: List[Track], now: float) -> List[dict]:
        """Detect tracks moving faster than the speed threshold.

        Speed is calculated as the total displacement between the first and
        last tracked position divided by the total elapsed time.
        """
        events: List[dict] = []
        threshold = self.config.running_speed_threshold_px_per_sec
        event_type = self.config.event.event_type_mapping.get(
            "running", "running"
        )

        for track in tracks:
            if len(track.timestamps) < 2 or len(track.positions) < 2:
                continue

            duration = track.timestamps[-1] - track.timestamps[0]
            if duration <= 0:
                continue

            cx0, cy0 = track.positions[0]
            cx1, cy1 = track.positions[-1]
            distance = math.hypot(cx1 - cx0, cy1 - cy0)
            speed = distance / duration

            if speed > threshold:
                if self._is_on_cooldown(event_type, track.track_id, now):
                    continue
                self._mark_cooldown(event_type, track.track_id, now)
                events.append(
                    self._make_event(
                        event_type=event_type,
                        track=track,
                        detail=(
                            f"Running detected: speed {speed:.1f} px/s "
                            f"exceeds threshold {threshold:.0f} px/s"
                        ),
                        timestamp=now,
                    )
                )

        return events

    def _check_zone_intrusion(self, tracks: List[Track]) -> List[dict]:
        """Check if track centers are inside defined zone polygons."""
        if not self.config.enabled or not self.config.zones:
            return []
        events: List[dict] = []
        for track in tracks:
            for zone in self.config.zones:
                name = zone.get("name", "unknown")
                points = zone.get("points", [])
                if len(points) < 3:
                    continue
                polygon = np.array(points, dtype=np.int32)
                cx, cy = int(track.center[0]), int(track.center[1])
                dist = cv2.pointPolygonTest(polygon, (cx, cy), False)
                if dist >= 0:
                    event_type = self.config.event.event_type_mapping.get(
                        "zone_intrusion", "zone"
                    )
                    if not self._is_on_cooldown(event_type, track.track_id, track.last_seen):
                        self._mark_cooldown(event_type, track.track_id, track.last_seen)
                        events.append(
                            self._make_event(
                                event_type=event_type,
                                track=track,
                                detail=f"Track entered zone '{name}'",
                                timestamp=track.last_seen,
                            )
                        )
        return events

    def _check_crowd(self, face_count: int, now: float) -> List[dict]:
        """Detect crowd conditions using debounce logic."""
        if not self.config.enabled:
            return []
        events: List[dict] = []
        threshold = self.config.crowd_threshold_count
        debounce = self.config.crowd_debounce_frames

        if face_count >= threshold:
            self._crowd_frames.append(now)
            self._crowd_frames = [t for t in self._crowd_frames if now - t < 5.0]

            if len(self._crowd_frames) >= debounce and not self._crowd_active:
                self._crowd_active = True
                event_type = self.config.event.event_type_mapping.get(
                    "crowd_alert", "crowd"
                )
                if not self._is_on_cooldown(event_type, "crowd", now):
                    self._mark_cooldown(event_type, "crowd", now)
                    events.append(
                        {
                            "event_type": event_type,
                            "camera_id": "0",
                            "track_id": "",
                            "detail": f"Crowd detected: {face_count} faces",
                            "timestamp": now,
                            "confidence": 1.0,
                        }
                    )
        else:
            self._crowd_frames = []
            self._crowd_active = False

        return events

    # ------------------------------------------------------------------
    # Cooldown helpers
    # ------------------------------------------------------------------

    def _is_on_cooldown(
        self, event_type: str, track_id: str, now: float
    ) -> bool:
        """Return ``True`` if this *(event_type, track_id)* is still on cooldown."""
        key = f"{event_type}:{track_id}"
        ts = self._cooldowns.get(key)
        if ts is not None and (now - ts) < self.config.event_cooldown_seconds:
            return True
        return False

    def _mark_cooldown(self, event_type: str, track_id: str, now: float):
        """Record the cooldown timestamp for the given *(event_type, track_id)* pair."""
        key = f"{event_type}:{track_id}"
        self._cooldowns[key] = now

    # ------------------------------------------------------------------
    # Event builder
    # ------------------------------------------------------------------

    @staticmethod
    def _make_event(
        event_type: str,
        track: Track,
        detail: str,
        timestamp: float,
    ) -> dict:
        """Build a standardised event dictionary for downstream consumers.

        The ``camera_id`` is extracted from the track ID which has the format
        ``cam-{camera_id}-{N}``.
        """
        # track_id format: cam-{camera_id}-{N}
        parts = track.track_id.split("-")
        camera_id = parts[1] if len(parts) >= 3 else "unknown"

        return {
            "event_type": event_type,
            "camera_id": camera_id,
            "track_id": track.track_id,
            "detail": detail,
            "timestamp": timestamp,
            "confidence": 1.0,
        }
