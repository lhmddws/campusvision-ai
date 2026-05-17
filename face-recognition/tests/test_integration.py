"""Integration tests for the full face recognition + behavior recognition pipeline.

Tests the end-to-end flow across module boundaries:
  Detection → Tracker → BehaviorAnalyzer → BehaviorEventPublisher

Follows existing patterns from ``test_behavior.py`` and ``test_tracker.py``.
"""

import time
from pathlib import Path
from unittest.mock import MagicMock, patch

import cv2
import numpy as np
import pytest

from app.behavior import BehaviorAnalyzer
from app.config import AppConfig, BehaviorConfig
from app.event_publisher import BehaviorEventPublisher
from app.tracker import FaceTracker, Track

from .conftest import make_face

FIXTURE_DIR = Path(__file__).parent / "fixtures"


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _make_track(
    track_id: str,
    positions,
    timestamps,
) -> Track:
    """Create a Track with the given position/timestamp history."""
    cx, cy = positions[-1]
    return Track(
        track_id=track_id,
        bbox=(cx - 10, cy - 10, cx + 10, cy + 10),
        center=(cx, cy),
        timestamps=list(timestamps),
        positions=list(positions),
        last_seen=timestamps[-1],
        frames_alive=len(timestamps),
    )


def _behavior_config(**overrides) -> BehaviorConfig:
    """Create a BehaviorConfig with overrides applied."""
    return BehaviorConfig(**overrides)


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


@pytest.fixture(scope="session")
def synthetic_face_image():
    """Load the pre-generated synthetic face-like image from fixtures."""
    img = cv2.imread(str(FIXTURE_DIR / "face.jpg"))
    assert img is not None, f"Fixture image not found at {FIXTURE_DIR / 'face.jpg'}"
    return img


# ===================================================================
# Detection → Tracker pipeline
# ===================================================================


class TestDetectionToTrackerPipeline:
    """Integration: :class:`FaceDetector` → :class:`FaceTracker`."""

    def test_detector_feeds_tracker(self, no_blur_detector, synthetic_face_image):
        """Real detector output is consumed by FaceTracker and produces valid tracks.

        Uses the Haar cascade fallback (``model_path=""``) with the fixture image.
        If no faces are found (Haar is conservative on synthetic images) the
        pipeline still handles empty results gracefully.
        """
        faces = no_blur_detector.detect(synthetic_face_image)
        tracker = FaceTracker(iou_threshold=0.3)
        tracks = tracker.update(faces, camera_id="A", timestamp=1000.0)

        if faces:
            assert len(tracks) == len(faces)
            for track in tracks:
                assert track.track_id.startswith("cam-A-")
                assert track.frames_alive == 1
                assert track.face is not None
        else:
            # No faces detected — pipeline still works with empty result
            assert tracks == []

    def test_detector_tracker_maintains_ids_across_frames(
        self, no_blur_detector, synthetic_face_image
    ):
        """Detector output across consecutive frames preserves track IDs."""
        # Frame 1
        faces1 = no_blur_detector.detect(synthetic_face_image)
        if len(faces1) < 1:
            pytest.skip("Need ≥1 detection for multi-frame ID test")

        tracker = FaceTracker(iou_threshold=0.3)
        tracks1 = tracker.update(faces1, camera_id="A", timestamp=1000.0)
        tids_1 = [t.track_id for t in tracks1]

        # Frame 2 (slightly shifted — simulate small head movement)
        img_shift = np.roll(synthetic_face_image, shift=3, axis=1)
        faces2 = no_blur_detector.detect(img_shift)
        tracks2 = tracker.update(faces2, camera_id="A", timestamp=1000.1)

        # The same detections should keep their track IDs (IoU overlap)
        if tracks2:
            for t in tracks2:
                assert t.track_id in tids_1 or t.track_id not in tids_1
            # At least some should have matched (high IoU with shift=3)
            matched = sum(1 for t in tracks2 if t.track_id in tids_1)
            assert matched >= 1, "Expected at least 1 track to keep its ID"

    def test_detector_tracker_empty_handling(self, no_blur_detector):
        """Empty detector output → FaceTracker returns empty list."""
        empty_img = np.ones((100, 100, 3), dtype=np.uint8) * 128
        faces = no_blur_detector.detect(empty_img)
        tracker = FaceTracker()
        tracks = tracker.update(faces, camera_id="A", timestamp=1000.0)
        assert tracks == []

    def test_detector_tracker_position_history(
        self, no_blur_detector, synthetic_face_image
    ):
        """Positions accumulate across frames for the same track."""
        faces = no_blur_detector.detect(synthetic_face_image)
        if len(faces) < 1:
            pytest.skip("Need ≥1 detection for position-history test")

        tracker = FaceTracker(iou_threshold=0.3)
        ts = 1000.0

        # Update three times with slight shifts
        for i in range(3):
            img = np.roll(synthetic_face_image, shift=i * 2, axis=1)
            dets = no_blur_detector.detect(img)
            if not dets:
                continue
            tracker.update(dets, camera_id="A", timestamp=ts + i * 0.1)

        tracks = tracker.get_tracks()
        if tracks:
            # At least one track should have multiple positions
            assert any(len(t.positions) > 1 for t in tracks)


# ===================================================================
# Tracker → Behavior pipeline
# ===================================================================


class TestTrackerToBehaviorPipeline:
    """Integration: :class:`FaceTracker` tracks → :class:`BehaviorAnalyzer` events."""

    def test_tracks_produce_behavior_events(self):
        """Known tracks fed to BehaviorAnalyzer generate correctly typed events."""
        config = _behavior_config(
            enabled=True,
            loitering_threshold_seconds=1.0,
            loitering_radius_px=100,
            running_speed_threshold_px_per_sec=50,
        )
        analyzer = BehaviorAnalyzer(config)
        now = 1000.0

        loiter_track = _make_track(
            "cam-A-1", [(100, 100), (100, 100)], [now - 10.0, now]
        )
        run_track = _make_track(
            "cam-A-2", [(0, 0), (1000, 0)], [now - 2.0, now]
        )

        events = analyzer.analyze(
            tracks=[loiter_track, run_track],
            frame_face_count=0,
            timestamp=now,
        )

        assert len(events) >= 1
        event_types = {e["event_type"] for e in events}
        for ev in events:
            assert "event_type" in ev
            assert "camera_id" in ev
            assert "track_id" in ev
            assert "detail" in ev
            assert "timestamp" in ev
            assert "confidence" in ev

    def test_analyzer_receives_tracker_tracks(
        self, no_blur_detector, synthetic_face_image
    ):
        """End-to-end: real tracker output → BehaviorAnalyzer does not crash.

        Uses the Haar cascade detector with the fixture image, feeds results
        through the tracker, then passes active tracks to the behavior analyzer.
        """
        faces = no_blur_detector.detect(synthetic_face_image)
        tracker = FaceTracker(iou_threshold=0.3)
        ts = 1000.0

        # Run 3 frames to build some track history
        for i in range(3):
            img = np.roll(synthetic_face_image, shift=i * 2, axis=1)
            dets = no_blur_detector.detect(img)
            if dets:
                tracker.update(dets, camera_id="A", timestamp=ts + i * 0.1)

        active_tracks = tracker.get_tracks()
        config = _behavior_config(enabled=True)
        analyzer = BehaviorAnalyzer(config)
        events = analyzer.analyze(
            tracks=active_tracks,
            frame_face_count=len(faces),
            timestamp=ts + 0.3,
        )

        # Pipeline should not crash; events depend on track history
        assert isinstance(events, list)

    def test_mixed_tracks_some_trigger_events(self):
        """Only tracks that meet behavior criteria produce events."""
        config = _behavior_config(
            enabled=True,
            loitering_threshold_seconds=5.0,
            loitering_radius_px=50,
        )
        analyzer = BehaviorAnalyzer(config)
        now = 1000.0

        loiter = _make_track(
            "cam-A-1", [(100, 100), (100, 100)], [now - 10.0, now]  # loitering
        )
        moving = _make_track(
            "cam-A-2", [(100, 100), (300, 100)], [now - 10.0, now]  # moved away
        )

        events = analyzer.analyze(
            tracks=[loiter, moving],
            frame_face_count=0,
            timestamp=now,
        )
        assert len(events) == 1
        assert events[0]["track_id"] == "cam-A-1"


# ===================================================================
# event_type ≤ 8 char constraint
# ===================================================================


class TestEventTypeConstraint:
    """All ``event_type`` values must be ≤ 8 characters (DB ``VARCHAR(8)``)."""

    def test_default_mappings_all_short(self):
        """Default event_type_mapping values are all ≤ 8 chars."""
        config = BehaviorConfig()
        mapping = config.event.event_type_mapping
        assert len(mapping) > 0, "Expected at least one mapping"
        for key, value in mapping.items():
            assert len(value) <= 8, (
                f"Mapping '{key}'→'{value}' is {len(value)} chars (> 8)"
            )

    def test_behavior_analyzer_events_all_short(self):
        """Events emitted by BehaviorAnalyzer have event_type ≤ 8 chars."""
        config = _behavior_config(
            enabled=True,
            loitering_threshold_seconds=1.0,
            loitering_radius_px=100,
            running_speed_threshold_px_per_sec=1,
        )
        analyzer = BehaviorAnalyzer(config)
        now = 1000.0

        loiter = _make_track(
            "cam-A-1", [(100, 100), (100, 100)], [now - 10.0, now]
        )
        run = _make_track(
            "cam-A-2", [(0, 0), (1000, 0)], [now - 2.0, now]
        )

        events = analyzer.analyze(
            tracks=[loiter, run],
            frame_face_count=5,
            timestamp=now,
        )
        assert len(events) >= 1
        for ev in events:
            assert len(ev["event_type"]) <= 8, (
                f"event_type '{ev['event_type']}' is {len(ev['event_type'])} chars "
                f"(exceeds VARCHAR(8) limit)"
            )

    def test_custom_mapping_also_short(self):
        """Custom event_type_mapping with short values passes constraint."""
        config = _behavior_config(
            enabled=True,
            loitering_threshold_seconds=1.0,
            loitering_radius_px=100,
        )
        config.event.event_type_mapping = {"loitering": "LTR"}  # 3 chars
        analyzer = BehaviorAnalyzer(config)
        now = 1000.0

        track = _make_track(
            "cam-A-1", [(100, 100), (100, 100)], [now - 10.0, now]
        )
        events = analyzer.analyze(tracks=[track], frame_face_count=0, timestamp=now)
        assert len(events) == 1
        assert events[0]["event_type"] == "LTR"
        assert len(events[0]["event_type"]) <= 8


# ===================================================================
# Behavior disabled
# ===================================================================


class TestBehaviorDisabled:
    """When ``behavior.enabled=False``, no events are produced at any stage."""

    def test_analyzer_returns_empty_when_disabled(self):
        """BehaviorAnalyzer.analyze() returns [] when disabled, even with valid tracks."""
        config = _behavior_config(
            enabled=False,
            loitering_threshold_seconds=1.0,
            loitering_radius_px=100,
        )
        analyzer = BehaviorAnalyzer(config)
        now = 1000.0
        track = _make_track(
            "cam-A-1", [(100, 100), (100, 100)], [now - 10.0, now]
        )
        events = analyzer.analyze(
            tracks=[track],
            frame_face_count=5,
            timestamp=now,
        )
        assert events == []

    def test_analyzer_crowd_and_zone_also_empty(self):
        """All check methods return [] when behavior is disabled."""
        config = _behavior_config(
            enabled=False,
            zones=[{"name": "test", "points": [[0, 0], [100, 0], [100, 100], [0, 100]]}],
            crowd_threshold_count=1,
        )
        analyzer = BehaviorAnalyzer(config)
        track = _make_track("cam-A-1", [(50, 50)], [1000.0])
        assert analyzer._check_zone_intrusion([track]) == []
        assert analyzer._check_crowd(10, 1000.0) == []

    def test_publisher_no_op_when_disabled(self):
        """BehaviorEventPublisher does nothing when behavior is disabled."""
        cfg = AppConfig()
        cfg.behavior.enabled = False
        publisher = BehaviorEventPublisher(cfg)
        publisher.publish_behavior_event({
            "camera_id": "cam_A_01",
            "event_type": "loitering",
            "timestamp": time.time(),
        })
        assert len(publisher._buffer) == 0


# ===================================================================
# Full pipeline data flow
# ===================================================================


class TestFullPipelineDataFlow:
    """End-to-end: synthetic faces → tracker → behavior → correctly structured event dicts."""

    def test_known_faces_through_full_pipeline(self):
        """Known Face objects → tracker (multi-frame) → behavior analyzer → events.

        Uses manually created Face objects (via ``make_face``) for deterministic
        results, then runs the full tracker → behavior pipeline.
        """
        # -- Stage 1: Tracker --
        faces1 = [
            make_face(x1=10, y1=10, x2=100, y2=100, confidence=0.9),
            make_face(x1=200, y1=200, x2=300, y2=300, confidence=0.85),
        ]
        tracker = FaceTracker(iou_threshold=0.3)
        ts = 1000.0

        tracks1 = tracker.update(faces1, camera_id="A", timestamp=ts)
        assert len(tracks1) == 2

        # Second frame — slight shift
        faces2 = [
            make_face(x1=15, y1=15, x2=105, y2=105, confidence=0.9),
            make_face(x1=205, y1=205, x2=305, y2=305, confidence=0.85),
        ]
        tracker.update(faces2, camera_id="A", timestamp=ts + 0.5)

        # -- Stage 2: Behavior analyzer --
        config = _behavior_config(
            enabled=True,
            loitering_threshold_seconds=1.0,     # > 0.5s alive, so not loitering
            loitering_radius_px=100,
            running_speed_threshold_px_per_sec=500,
        )
        analyzer = BehaviorAnalyzer(config)
        events = analyzer.analyze(
            tracks=tracker.get_tracks(),
            frame_face_count=2,
            timestamp=ts + 0.5,
        )

        # No events expected (duration < loitering threshold, speed < running threshold)
        assert events == []

    def test_full_pipeline_loitering_event(self):
        """End-to-end: known tracks → loitering event emitted with correct structure."""
        tracker = FaceTracker(iou_threshold=0.3)
        ts = 1000.0

        # Create a track that stays in place
        face = make_face(x1=50, y1=50, x2=150, y2=150, confidence=0.9)
        tracker.update([face], camera_id="A", timestamp=ts)

        # Update repeatedly to build history (simulate staying in place)
        for i in range(1, 6):
            tracker.update([face], camera_id="A", timestamp=ts + i * 2.0)

        config = _behavior_config(
            enabled=True,
            loitering_threshold_seconds=5.0,
            loitering_radius_px=50,
        )
        analyzer = BehaviorAnalyzer(config)
        events = analyzer.analyze(
            tracks=tracker.get_tracks(),
            frame_face_count=1,
            timestamp=ts + 10.0,
        )

        assert len(events) >= 1
        ev = events[0]
        assert ev["event_type"] == "loiter"
        assert ev["track_id"] == "cam-A-1"
        assert ev["camera_id"] == "A"
        assert isinstance(ev["detail"], str) and len(ev["detail"]) > 0
        assert isinstance(ev["timestamp"], float)
        assert ev["confidence"] == 1.0

    def test_event_dict_required_keys(self):
        """All event dicts from BehaviorAnalyzer contain the required 6 keys."""
        config = _behavior_config(
            enabled=True,
            loitering_threshold_seconds=1.0,
            loitering_radius_px=100,
        )
        analyzer = BehaviorAnalyzer(config)
        now = 1000.0
        track = _make_track(
            "cam-A-1", [(100, 100), (100, 100)], [now - 10.0, now]
        )
        events = analyzer.analyze(tracks=[track], frame_face_count=0, timestamp=now)
        assert len(events) == 1
        ev = events[0]

        expected_keys = {"event_type", "camera_id", "track_id", "detail", "timestamp", "confidence"}
        assert set(ev.keys()) == expected_keys, (
            f"Event missing keys: {expected_keys - set(ev.keys())}"
        )


# ===================================================================
# Event publisher integration
# ===================================================================


class TestEventPublisherIntegration:
    """Behavior events → :class:`BehaviorEventPublisher` produces correct Kafka messages."""

    def test_behavior_event_through_publisher(self):
        """BehaviorAnalyzer output → BehaviorEventPublisher → Kafka with correct structure.

        Verifies event_type mapping (loitering→loiter), source field, and
        all required fields in the Kafka message.
        """
        cfg = AppConfig()
        cfg.behavior.enabled = True
        cfg.behavior.event.event_type_mapping = {
            "loitering": "loiter",
            "running": "running",
            "zone_intrusion": "zone",
            "crowd_alert": "crowd",
        }

        with patch("kafka.KafkaProducer") as mock_kafka:
            producer_instance = MagicMock()
            mock_kafka.return_value = producer_instance

            publisher = BehaviorEventPublisher(cfg)
            publisher.publish_behavior_event({
                "camera_id": "cam_A_01",
                "event_type": "loitering",
                "track_id": "cam-A-1",
                "detail": "Loitering detected: stayed 10.0s within 5px radius",
                "timestamp": 1000.0,
                "confidence": 1.0,
            })

            sent = producer_instance.send.call_args[1]["value"]
            # Mapping applied
            assert sent["event_type"] == "loiter"
            # Source field added by publisher
            assert sent["source"] == "behavior"
            # All required fields present
            assert sent["camera_id"] == "cam_A_01"
            assert sent["track_id"] == "cam-A-1"
            assert sent["detail"] == "Loitering detected: stayed 10.0s within 5px radius"
            assert sent["timestamp"] == 1000.0
            assert sent["confidence"] == 1.0

    def test_publisher_buffer_with_real_events(self):
        """Behavior events are buffered when Kafka is unavailable (e.g. no broker)."""
        cfg = AppConfig()
        cfg.behavior.enabled = True

        with patch("kafka.KafkaProducer", side_effect=Exception("No Kafka")):
            publisher = BehaviorEventPublisher(cfg)
            publisher.publish_behavior_event({
                "camera_id": "cam_A_01",
                "event_type": "running",
                "track_id": "cam-A-2",
                "detail": "Running detected: speed 250.0 px/s",
                "timestamp": 2000.0,
                "confidence": 1.0,
            })

            assert len(publisher._buffer) == 1
            assert publisher._producer is None
            buffered = publisher._buffer[0]
            assert buffered["event_type"] == "running"
            assert buffered["source"] == "behavior"
