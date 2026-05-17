"""Entry/exit direction determination via ROI line crossing."""

import threading
import time
from typing import Dict, List, Optional, Tuple


class DirectionDetector:
    """Determine whether a tracked face is entering or exiting the building.

    Uses a virtual ROI (region-of-interest) line placed at a configurable
    fraction of the frame width.  When a face track crosses that line
    left-to-right it is classified as *entry*; right-to-left as *exit*.
    """

    def __init__(self, config):
        self.config = config
        self._tracks: Dict[str, List[Tuple[float, float, float]]] = {}
        self._lock = threading.Lock()
        # Tracks older than this (seconds) are removed during cleanup.
        self._track_ttl = 5.0

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def determine(
        self,
        face_id: str,
        face_center_x: float,
        face_center_y: float,
        frame_width: int,
    ) -> Optional[str]:
        """Return ``'entry'``, ``'exit'``, or ``None`` for the given face track.

        Parameters
        ----------
        face_id:
            Unique identifier for the tracked face (e.g. camera_id).
        face_center_x, face_center_y:
            Centre of the face bounding box in pixels.
        frame_width:
            Width of the current frame in pixels (used to compute the ROI line).
        """
        roi_x = frame_width * self.config.roi_line_x
        now = time.time()

        with self._lock:
            if face_id not in self._tracks:
                self._tracks[face_id] = []
            self._tracks[face_id].append((face_center_x, face_center_y, now))

            track = self._tracks[face_id]
            if len(track) < self.config.min_track_points:
                return None

            first_x = track[0][0]
            last_x = track[-1][0]

            # Left → right crosses the ROI line = entering the building
            if first_x < roi_x and last_x >= roi_x:
                return "entry"
            # Right → left = exiting the building
            if first_x >= roi_x and last_x < roi_x:
                return "exit"

        return None

    def cleanup(self):
        """Remove tracked faces that haven't been seen in the last 5 seconds."""
        now = time.time()
        stale_before = now - self._track_ttl
        with self._lock:
            stale_ids = [
                fid
                for fid, track in self._tracks.items()
                if track[-1][2] < stale_before
            ]
            for fid in stale_ids:
                del self._tracks[fid]
