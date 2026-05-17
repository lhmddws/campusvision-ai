"""Tests for the multi-face IOU tracker."""

import time

import numpy as np
import pytest

from app.detector import Face
from app.tracker import FaceTracker, Track, _iou


# ---------------------------------------------------------------------------
# IoU helper tests
# ---------------------------------------------------------------------------

class TestIoU:
    def test_full_overlap(self):
        box = (10, 10, 100, 100)
        assert _iou(box, box) == pytest.approx(1.0)

    def test_no_overlap(self):
        a = (10, 10, 50, 50)
        b = (100, 100, 150, 150)
        assert _iou(a, b) == pytest.approx(0.0)

    def test_partial_overlap(self):
        a = (10, 10, 100, 100)
        b = (50, 50, 150, 150)
        # intersection area = 2500, union area = 15600
        assert _iou(a, b) == pytest.approx(2500 / 15600, rel=1e-4)

    def test_edge_touching(self):
        a = (10, 10, 100, 100)
        b = (100, 10, 200, 100)
        assert _iou(a, b) == pytest.approx(0.0)

    def test_one_contained(self):
        a = (0, 0, 200, 200)
        b = (50, 50, 100, 100)
        assert _iou(a, b) == pytest.approx(2500 / 40000, rel=1e-4)

    def test_zero_area_box(self):
        a = (10, 10, 10, 10)
        b = (10, 10, 100, 100)
        assert _iou(a, b) == pytest.approx(0.0)


# ---------------------------------------------------------------------------
# Track dataclass tests
# ---------------------------------------------------------------------------

class TestTrackDataclass:
    def test_default_fields(self):
        track = Track(
            track_id="cam-A-1",
            bbox=(10, 10, 100, 100),
            center=(55, 55),
        )
        assert track.track_id == "cam-A-1"
        assert track.bbox == (10, 10, 100, 100)
        assert track.center == (55, 55)
        assert track.timestamps == []
        assert track.positions == []
        assert track.embedding is None
        assert track.face is None
        assert track.last_seen == 0.0
        assert track.frames_alive == 1

    def test_with_optional_fields(self):
        emb = np.array([0.1, 0.2, 0.3], dtype=np.float32)
        face = Face(x1=10, y1=10, x2=100, y2=100, confidence=0.9)
        track = Track(
            track_id="cam-B-7",
            bbox=(20, 20, 80, 80),
            center=(50, 50),
            timestamps=[100.0, 101.0],
            positions=[(50, 50)],
            embedding=emb,
            face=face,
            last_seen=101.0,
            frames_alive=2,
        )
        assert track.track_id == "cam-B-7"
        assert np.array_equal(track.embedding, emb)
        assert track.face is face
        assert track.frames_alive == 2


# ---------------------------------------------------------------------------
# FaceTracker tests
# ---------------------------------------------------------------------------

class TestFaceTracker:
    def _make_face(self, x1, y1, x2, y2, confidence=0.9):
        return Face(x1=x1, y1=y1, x2=x2, y2=y2, confidence=confidence)

    # -- Same face across frames ---------------------------------------

    def test_same_face_gets_same_track_id(self):
        """A face in a similar position across frames gets the same track_id."""
        tracker = FaceTracker(iou_threshold=0.3)
        ts = 1000.0

        f1 = [self._make_face(10, 10, 100, 100, confidence=0.9)]
        result1 = tracker.update(f1, camera_id="A", timestamp=ts)

        assert len(result1) == 1
        tid1 = result1[0].track_id

        f2 = [self._make_face(15, 15, 105, 105, confidence=0.9)]
        result2 = tracker.update(f2, camera_id="A", timestamp=ts + 0.1)

        assert len(result2) == 1
        assert result2[0].track_id == tid1, "Same face should keep same track_id"

    def test_track_frames_alive_increments(self):
        """frames_alive increments each time the same track is matched."""
        tracker = FaceTracker(iou_threshold=0.3)
        ts = 1000.0

        face = [self._make_face(10, 10, 100, 100)]
        for i in range(3):
            result = tracker.update(face, camera_id="A", timestamp=ts + i * 0.1)
            assert result[0].frames_alive == i + 1

    def test_position_history_updated(self):
        """Track accumulates position history across updates."""
        tracker = FaceTracker(iou_threshold=0.3)
        ts = 1000.0

        tracker.update([self._make_face(10, 10, 100, 100)], camera_id="A", timestamp=ts)
        tracker.update([self._make_face(20, 20, 110, 110)], camera_id="A", timestamp=ts + 0.1)

        tracks = tracker.get_tracks()
        assert len(tracks) == 1
        assert len(tracks[0].positions) == 2
        assert len(tracks[0].timestamps) == 2

    def test_position_history_capped_at_30(self):
        """Track keeps at most 30 recent positions."""
        tracker = FaceTracker(iou_threshold=0.3)
        ts = 1000.0

        face = [self._make_face(10, 10, 100, 100)]
        for i in range(35):
            tracker.update(face, camera_id="A", timestamp=ts + i * 0.1)

        tracks = tracker.get_tracks()
        assert len(tracks[0].positions) == 30

    # -- New face gets new track_id -----------------------------------

    def test_new_face_gets_new_track_id(self):
        """A completely new face in a different location gets a new track_id."""
        tracker = FaceTracker(iou_threshold=0.3)
        ts = 1000.0

        result1 = tracker.update(
            [self._make_face(10, 10, 100, 100)], camera_id="A", timestamp=ts,
        )
        tid1 = result1[0].track_id

        # Face in completely different position (IoU ≈ 0)
        result2 = tracker.update(
            [self._make_face(300, 300, 400, 400)], camera_id="A", timestamp=ts + 0.1,
        )
        assert len(result2) == 1
        assert result2[0].track_id != tid1

    def test_track_id_format(self):
        """track_id follows cam-{camera_id}-{N} format."""
        tracker = FaceTracker(iou_threshold=0.3)
        ts = 1000.0

        result = tracker.update(
            [self._make_face(10, 10, 100, 100)], camera_id="B", timestamp=ts,
        )
        assert result[0].track_id == "cam-B-1"

        result2 = tracker.update(
            [self._make_face(300, 300, 400, 400)], camera_id="B", timestamp=ts + 0.1,
        )
        assert result2[0].track_id == "cam-B-2"

    def test_track_id_global_sequence(self):
        """Global sequence increments across different camera_ids."""
        tracker = FaceTracker(iou_threshold=0.3)
        ts = 1000.0

        r1 = tracker.update([self._make_face(10, 10, 100, 100)], camera_id="A", timestamp=ts)
        r2 = tracker.update([self._make_face(300, 300, 400, 400)], camera_id="B", timestamp=ts + 0.1)

        assert r1[0].track_id == "cam-A-1"
        assert r2[0].track_id == "cam-B-2"

    # -- Track expiry --------------------------------------------------

    def test_track_expires_after_ttl(self):
        """Track is removed if not seen within track_ttl seconds.

        Note: because matching happens *before* eviction in ``update()``,
        a face re-appearing in the same position re-activates the old track.
        To test expiry, the new face must be in a *different* position
        (IoU below threshold) so a new track is created.
        """
        tracker = FaceTracker(iou_threshold=0.3, track_ttl=0.5)
        ts = 1000.0

        tracker.update([self._make_face(10, 10, 100, 100)], camera_id="A", timestamp=ts)
        assert len(tracker.get_tracks()) == 1

        # New face in a different position (IoU ≈ 0) — old track is stale
        # so a new track is created; old one is evicted by _evict_stale.
        tracker.update(
            [self._make_face(300, 300, 400, 400)],
            camera_id="A", timestamp=ts + 2.0,
        )
        tracks = tracker.get_tracks()
        assert len(tracks) == 1
        # Old track was evicted (last_seen=1000 < 1002-0.5), new track created
        assert tracks[0].track_id == "cam-A-2"

    def test_track_not_expired_within_ttl(self):
        """Track stays alive if updated within track_ttl."""
        tracker = FaceTracker(iou_threshold=0.3, track_ttl=5.0)
        ts = 1000.0

        tracker.update([self._make_face(10, 10, 100, 100)], camera_id="A", timestamp=ts)
        tracker.update([self._make_face(15, 15, 105, 105)], camera_id="A", timestamp=ts + 1.0)

        tracks = tracker.get_tracks()
        assert len(tracks) == 1

    # -- Multiple faces ------------------------------------------------

    def test_multiple_faces(self):
        """Multiple distinct faces are tracked simultaneously."""
        tracker = FaceTracker(iou_threshold=0.3)
        ts = 1000.0

        faces = [
            self._make_face(10, 10, 100, 100, confidence=0.9),
            self._make_face(200, 200, 300, 300, confidence=0.85),
        ]
        result = tracker.update(faces, camera_id="A", timestamp=ts)

        assert len(result) == 2
        assert result[0].track_id != result[1].track_id

    def test_multiple_faces_across_frames(self):
        """Multiple faces keep their track IDs across consecutive frames."""
        tracker = FaceTracker(iou_threshold=0.3)
        ts = 1000.0

        faces1 = [
            self._make_face(10, 10, 100, 100, confidence=0.9),
            self._make_face(200, 200, 300, 300, confidence=0.85),
        ]
        result1 = tracker.update(faces1, camera_id="A", timestamp=ts)
        tid_a = result1[0].track_id
        tid_b = result1[1].track_id

        # Same two faces shifted a bit
        faces2 = [
            self._make_face(15, 15, 105, 105, confidence=0.9),
            self._make_face(205, 205, 305, 305, confidence=0.85),
        ]
        result2 = tracker.update(faces2, camera_id="A", timestamp=ts + 0.1)

        assert len(result2) == 2
        assert result2[0].track_id == tid_a
        assert result2[1].track_id == tid_b

    def test_greedy_matching_assigns_highest_confidence_first(self):
        """Greedy matching: highest-confidence detection gets best match.

        Face B (0.95 conf) overlaps Track 1 (IoU ~0.43) and is processed
        first by greedy matching, taking Track 1. Face A (0.90 conf) then
        has no match and creates a new track (cam-A-3).
        """
        tracker = FaceTracker(iou_threshold=0.3)
        ts = 1000.0

        # First frame: two faces
        faces1 = [
            self._make_face(10, 10, 100, 100, confidence=0.9),
            self._make_face(200, 200, 300, 300, confidence=0.85),
        ]
        result1 = tracker.update(faces1, camera_id="A", timestamp=ts)
        # result1[0] = cam-A-1 (face at 10,10,100,100, conf 0.9)
        # result1[1] = cam-A-2 (face at 200,200,300,300, conf 0.85)

        # Second frame: face B (0.95, overlaps Track 1) is processed first
        # and takes Track 1. Face A (0.90, common position) has no match
        # and gets a new track (cam-A-3).
        faces2 = [
            self._make_face(10, 10, 100, 100, confidence=0.9),
            self._make_face(30, 30, 120, 120, confidence=0.95),
        ]
        result2 = tracker.update(faces2, camera_id="A", timestamp=ts + 0.1)

        # Face B (higher confidence, result2[1]) stole Track 1 from Face A
        assert result2[1].track_id == result1[0].track_id
        # Face A (result2[0]) had to create a new track
        assert result2[0].track_id == "cam-A-3"

    # -- Edge cases ----------------------------------------------------

    def test_empty_input_returns_empty_list(self):
        """update() with empty faces returns []."""
        tracker = FaceTracker()
        result = tracker.update([], camera_id="A", timestamp=1000.0)
        assert result == []

    def test_get_tracks_after_empty_update(self):
        """get_tracks() after feeding faces then empty still works."""
        tracker = FaceTracker(track_ttl=1.0)
        ts = 1000.0

        tracker.update([self._make_face(10, 10, 100, 100)], camera_id="A", timestamp=ts)
        assert len(tracker.get_tracks()) == 1

        # Empty update should not crash and stale tracks are evicted
        tracker.update([], camera_id="A", timestamp=ts + 0.1)
        assert len(tracker.get_tracks()) == 1  # still within TTL

    def test_cleanup_removes_stale_tracks(self):
        """cleanup() removes expired tracks."""
        tracker = FaceTracker(track_ttl=0.3)
        ts = 1000.0

        tracker.update([self._make_face(10, 10, 100, 100)], camera_id="A", timestamp=ts)
        assert len(tracker.get_tracks()) == 1

        time.sleep(0.4)
        tracker.cleanup()
        assert len(tracker.get_tracks()) == 0

    def test_max_tracks_enforced(self):
        """When track count exceeds max_tracks, oldest tracks are evicted."""
        tracker = FaceTracker(iou_threshold=0.3, max_tracks=3)
        ts = 1000.0

        # Create three tracks
        for i in range(3):
            x = 10 + i * 300
            tracker.update(
                [self._make_face(x, x, x + 100, x + 100)],
                camera_id="A", timestamp=ts + i * 0.1,
            )

        assert len(tracker.get_tracks()) == 3

        # Add a fourth face — oldest track should be evicted
        tracker.update(
            [self._make_face(1000, 1000, 1100, 1100)],
            camera_id="A", timestamp=ts + 0.4,
        )
        assert len(tracker.get_tracks()) == 3

    def test_low_iou_below_threshold_creates_new_track(self):
        """When IoU between detection and all tracks is below threshold, new
        tracks are created."""
        tracker = FaceTracker(iou_threshold=0.5)
        ts = 1000.0

        tracker.update([self._make_face(10, 10, 100, 100)], camera_id="A", timestamp=ts)

        # Overlap ~56% area, but IoU ~0.36 (two similar sized boxes shifted)
        tracker.update(
            [self._make_face(60, 60, 150, 150)],
            camera_id="A", timestamp=ts + 0.1,
        )

        tracks = tracker.get_tracks()
        assert len(tracks) == 2

    def test_track_gets_face_reference(self):
        """Matched track's face attribute is updated to latest Face object."""
        tracker = FaceTracker(iou_threshold=0.3)
        ts = 1000.0

        face1 = self._make_face(10, 10, 100, 100, confidence=0.9)
        tracker.update([face1], camera_id="A", timestamp=ts)

        face2 = self._make_face(15, 15, 105, 105, confidence=0.95)
        tracker.update([face2], camera_id="A", timestamp=ts + 0.1)

        tracks = tracker.get_tracks()
        assert tracks[0].face is face2
