"""Multi-face IOU tracker — assigns unique track IDs to faces across frames."""

import threading
import time
from dataclasses import dataclass, field
from typing import Dict, List, Optional, Tuple

import numpy as np

from app.detector import Face


# ---------------------------------------------------------------------------
# Track data
# ---------------------------------------------------------------------------

@dataclass
class Track:
    """A tracked face across multiple frames.

    Fields
    ------
    track_id:
        Unique identifier in the format ``cam-{camera_id}-{N}``.
    bbox:
        Most recent bounding box ``(x1, y1, x2, y2)`` in pixel coordinates.
    center:
        Center ``(cx, cy)`` of the most recent bounding box.
    timestamps:
        List of timestamps when this track was observed (newest last).
    positions:
        List of ``(cx, cy)`` center positions (newest last, max 30).
    embedding:
        Optional feature vector associated with this track.
    face:
        Optional ``Face`` object from the most recent detection.
    last_seen:
        Timestamp of the most recent observation.
    frames_alive:
        Number of frames this track has been continuously matched.
    """

    track_id: str
    bbox: Tuple[float, float, float, float]
    center: Tuple[float, float]
    timestamps: List[float] = field(default_factory=list)
    positions: List[Tuple[float, float]] = field(default_factory=list)
    embedding: Optional[np.ndarray] = None
    face: Optional[Face] = None
    last_seen: float = 0.0
    frames_alive: int = 1


# ---------------------------------------------------------------------------
# IoU helper
# ---------------------------------------------------------------------------

def _iou(a: Tuple[float, float, float, float],
         b: Tuple[float, float, float, float]) -> float:
    """Intersection-over-Union between two bounding boxes ``(x1, y1, x2, y2)``."""
    x1 = max(a[0], b[0])
    y1 = max(a[1], b[1])
    x2 = min(a[2], b[2])
    y2 = min(a[3], b[3])

    inter = max(0.0, x2 - x1) * max(0.0, y2 - y1)
    area_a = (a[2] - a[0]) * (a[3] - a[1])
    area_b = (b[2] - b[0]) * (b[3] - b[1])
    union = area_a + area_b - inter

    return inter / (union + 1e-10)


# ---------------------------------------------------------------------------
# FaceTracker
# ---------------------------------------------------------------------------

class FaceTracker:
    """Assign persistent track IDs to faces across consecutive frames.

    Uses greedy IoU matching: detections are sorted by confidence (highest
    first) and each is matched to the existing track with which it has the
    highest IoU above ``iou_threshold``.

    Thread-safe (uses ``threading.Lock``).
    """

    def __init__(self, iou_threshold: float = 0.3,
                 max_tracks: int = 100,
                 track_ttl: float = 5.0):
        self.iou_threshold = iou_threshold
        self.max_tracks = max_tracks
        self.track_ttl = track_ttl

        self._tracks: Dict[str, Track] = {}
        self._lock = threading.Lock()
        self._global_seq: int = 0

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def update(self, faces: List[Face],
               embeddings: Optional[List[np.ndarray]] = None,
               camera_id: str = "",
               timestamp: Optional[float] = None) -> List[Track]:
        """Update tracks with a new set of face detections.

        Parameters
        ----------
        faces:
            Face detections from the current frame.
        embeddings:
            Optional feature vectors, one per face (in the same order).
        camera_id:
            Camera identifier used in track ID generation.
        timestamp:
            Current timestamp (seconds, e.g. ``time.time()``).  Defaults to
            ``time.time()`` when ``None``.

        Returns
        -------
        List[Track]:
            Active tracks after the update (same length as ``faces``).
        """
        now = timestamp if timestamp is not None else time.time()

        if not faces:
            with self._lock:
                self._evict_stale(now)
            return []

        bboxes = [(f.x1, f.y1, f.x2, f.y2) for f in faces]
        confidences = [f.confidence for f in faces]

        order = sorted(range(len(faces)), key=lambda i: confidences[i], reverse=True)

        matched_track_ids: set = set()
        active_tracks: List[Optional[Track]] = [None] * len(faces)

        with self._lock:
            for det_idx in order:
                det_bbox = bboxes[det_idx]
                best_iou = self.iou_threshold
                best_tid = None

                for tid, track in self._tracks.items():
                    if tid in matched_track_ids:
                        continue
                    iou_val = _iou(det_bbox, track.bbox)
                    if iou_val > best_iou:
                        best_iou = iou_val
                        best_tid = tid

                if best_tid is not None:
                    track = self._tracks[best_tid]
                    track.bbox = det_bbox
                    cx = (det_bbox[0] + det_bbox[2]) / 2.0
                    cy = (det_bbox[1] + det_bbox[3]) / 2.0
                    track.center = (cx, cy)
                    track.timestamps.append(now)
                    track.positions.append((cx, cy))
                    if len(track.positions) > 30:
                        track.positions = track.positions[-30:]
                    if len(track.timestamps) > 30:
                        track.timestamps = track.timestamps[-30:]
                    track.last_seen = now
                    track.frames_alive += 1
                    track.face = faces[det_idx]
                    if embeddings is not None and det_idx < len(embeddings):
                        track.embedding = embeddings[det_idx]

                    matched_track_ids.add(best_tid)
                    active_tracks[det_idx] = track
                else:
                    self._global_seq += 1
                    tid = f"cam-{camera_id}-{self._global_seq}"
                    cx = (det_bbox[0] + det_bbox[2]) / 2.0
                    cy = (det_bbox[1] + det_bbox[3]) / 2.0

                    new_track = Track(
                        track_id=tid,
                        bbox=det_bbox,
                        center=(cx, cy),
                        timestamps=[now],
                        positions=[(cx, cy)],
                        embedding=(embeddings[det_idx]
                                   if embeddings is not None
                                   and det_idx < len(embeddings)
                                   else None),
                        face=faces[det_idx],
                        last_seen=now,
                        frames_alive=1,
                    )
                    self._tracks[tid] = new_track
                    matched_track_ids.add(tid)
                    active_tracks[det_idx] = new_track

            while len(self._tracks) > self.max_tracks:
                oldest_tid = min(self._tracks, key=lambda t: self._tracks[t].last_seen)
                del self._tracks[oldest_tid]

            self._evict_stale(now)

        return [t for t in active_tracks if t is not None]

    def get_tracks(self) -> List[Track]:
        """Return a snapshot of all active tracks.

        Note: this does **not** evict stale tracks — call ``cleanup()``
        separately if eviction is desired.
        """
        with self._lock:
            return list(self._tracks.values())

    def cleanup(self):
        """Remove all tracks that have not been seen in the last ``track_ttl``."""
        with self._lock:
            now = time.time()
            self._evict_stale(now)

    # ------------------------------------------------------------------
    # Internal helpers
    # ------------------------------------------------------------------

    def _evict_stale(self, now: float):
        """Remove tracks whose ``last_seen`` is older than ``track_ttl``."""
        cutoff = now - self.track_ttl
        stale_ids = [tid for tid, t in self._tracks.items()
                     if t.last_seen < cutoff]
        for tid in stale_ids:
            del self._tracks[tid]
