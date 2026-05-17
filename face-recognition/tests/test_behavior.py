"""Tests for BehaviorAnalyzer."""

import pytest

from app.behavior import BehaviorAnalyzer
from app.config import BehaviorConfig
from app.tracker import Track


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


def _config(**overrides) -> BehaviorConfig:
    """Create a BehaviorConfig with overrides applied."""
    return BehaviorConfig(**overrides)


# ---------------------------------------------------------------------------
# Tests: disabled
# ---------------------------------------------------------------------------


class TestDisabled:
    def test_disabled_returns_empty_list(self):
        """When ``behavior.enabled=False``, ``analyze()`` returns ``[]``."""
        config = _config(enabled=False)
        analyzer = BehaviorAnalyzer(config)
        assert analyzer.analyze(tracks=[], frame_face_count=0) == []

    def test_disabled_with_valid_tracks(self):
        """Even with tracks that would trigger events, disabled returns []."""
        config = _config(
            enabled=False,
            loitering_threshold_seconds=1.0,
            loitering_radius_px=100,
        )
        analyzer = BehaviorAnalyzer(config)
        now = 1000.0
        track = _make_track(
            track_id="cam-1-1",
            positions=[(100, 100), (100, 100)],
            timestamps=[now - 10.0, now],
        )
        assert analyzer.analyze(tracks=[track], frame_face_count=0, timestamp=now) == []


# ---------------------------------------------------------------------------
# Tests: loitering
# ---------------------------------------------------------------------------


class TestLoitering:
    def test_triggers_event_when_in_place_long_enough(self):
        """Track that stays within radius beyond threshold emits loiter event."""
        config = _config(
            enabled=True,
            loitering_threshold_seconds=5.0,
            loitering_radius_px=50,
        )
        analyzer = BehaviorAnalyzer(config)
        now = 1000.0
        track = _make_track(
            track_id="cam-1-42",
            positions=[(100, 100), (100, 100)],
            timestamps=[now - 10.0, now],
        )
        events = analyzer._check_loitering([track], now)
        assert len(events) == 1
        ev = events[0]
        assert ev["event_type"] == "loiter"
        assert ev["track_id"] == "cam-1-42"
        assert ev["camera_id"] == "1"
        assert ev["confidence"] == 1.0
        assert "Loitering" in ev["detail"]

    def test_not_triggered_when_moved_beyond_radius(self):
        """Track that moved far away does not trigger loiter."""
        config = _config(
            enabled=True,
            loitering_threshold_seconds=5.0,
            loitering_radius_px=50,
        )
        analyzer = BehaviorAnalyzer(config)
        now = 1000.0
        track = _make_track(
            track_id="cam-1-42",
            positions=[(100, 100), (300, 100)],
            timestamps=[now - 10.0, now],
        )
        assert analyzer._check_loitering([track], now) == []

    def test_not_triggered_when_duration_too_short(self):
        """Track with short duration does not trigger loiter."""
        config = _config(
            enabled=True,
            loitering_threshold_seconds=30.0,
        )
        analyzer = BehaviorAnalyzer(config)
        now = 1000.0
        track = _make_track(
            track_id="cam-1-42",
            positions=[(100, 100), (105, 105)],
            timestamps=[now - 5.0, now],
        )
        assert analyzer._check_loitering([track], now) == []

    def test_empty_positions(self):
        """Track with empty positions/timestamps is handled gracefully."""
        config = _config(enabled=True)
        analyzer = BehaviorAnalyzer(config)
        track = Track(
            track_id="cam-1-42",
            bbox=(0, 0, 10, 10),
            center=(5, 5),
            timestamps=[],
            positions=[],
            last_seen=0.0,
            frames_alive=1,
        )
        assert analyzer._check_loitering([track], 1000.0) == []

    def test_single_position(self):
        """Track with only one timestamp is handled gracefully."""
        config = _config(enabled=True)
        analyzer = BehaviorAnalyzer(config)
        track = _make_track(
            track_id="cam-1-42",
            positions=[(100, 100)],
            timestamps=[1000.0],
        )
        assert analyzer._check_loitering([track], 1000.0) == []

    def test_position_on_boundary(self):
        """Track at exactly the radius boundary (equal to radius) triggers loiter."""
        config = _config(
            enabled=True,
            loitering_threshold_seconds=1.0,
            loitering_radius_px=100.0,
        )
        analyzer = BehaviorAnalyzer(config)
        now = 1000.0
        track = _make_track(
            track_id="cam-1-42",
            positions=[(100, 100), (200, 100)],
            timestamps=[now - 10.0, now],
        )
        # distance = 100, radius = 100 → within boundary
        events = analyzer._check_loitering([track], now)
        assert len(events) == 1

    def test_multiple_tracks_only_loitering_one(self):
        """Only the loitering track generates an event."""
        config = _config(
            enabled=True,
            loitering_threshold_seconds=5.0,
            loitering_radius_px=50,
        )
        analyzer = BehaviorAnalyzer(config)
        now = 1000.0
        loiter_track = _make_track(
            track_id="cam-1-1",
            positions=[(100, 100), (100, 100)],
            timestamps=[now - 10.0, now],
        )
        moving_track = _make_track(
            track_id="cam-1-2",
            positions=[(100, 100), (300, 100)],
            timestamps=[now - 10.0, now],
        )
        events = analyzer._check_loitering([loiter_track, moving_track], now)
        assert len(events) == 1
        assert events[0]["track_id"] == "cam-1-1"


# ---------------------------------------------------------------------------
# Tests: running
# ---------------------------------------------------------------------------


class TestRunning:
    def test_triggers_event_when_fast(self):
        """Fast-moving track emits running event."""
        config = _config(
            enabled=True,
            running_speed_threshold_px_per_sec=100,
        )
        analyzer = BehaviorAnalyzer(config)
        now = 1000.0
        track = _make_track(
            track_id="cam-2-17",
            positions=[(0, 0), (500, 0)],
            timestamps=[now - 2.0, now],
        )
        events = analyzer._check_running([track], now)
        assert len(events) == 1
        ev = events[0]
        assert ev["event_type"] == "running"
        assert ev["track_id"] == "cam-2-17"
        assert ev["camera_id"] == "2"
        assert "Running" in ev["detail"]

    def test_not_triggered_when_slow(self):
        """Slow-moving track does not emit running event."""
        config = _config(
            enabled=True,
            running_speed_threshold_px_per_sec=200,
        )
        analyzer = BehaviorAnalyzer(config)
        now = 1000.0
        track = _make_track(
            track_id="cam-1-42",
            positions=[(0, 0), (100, 0)],
            timestamps=[now - 10.0, now],
        )
        assert analyzer._check_running([track], now) == []

    def test_zero_duration(self):
        """Track with zero elapsed time is handled gracefully."""
        config = _config(enabled=True)
        analyzer = BehaviorAnalyzer(config)
        now = 1000.0
        track = _make_track(
            track_id="cam-1-42",
            positions=[(0, 0), (500, 0)],
            timestamps=[now, now],
        )
        assert analyzer._check_running([track], now) == []

    def test_single_timestamp(self):
        """Track with only one timestamp is handled gracefully."""
        config = _config(enabled=True)
        analyzer = BehaviorAnalyzer(config)
        track = _make_track(
            track_id="cam-1-42",
            positions=[(100, 100)],
            timestamps=[1000.0],
        )
        assert analyzer._check_running([track], 1000.0) == []

    def test_empty_positions(self):
        """Track with empty positions is handled gracefully."""
        config = _config(enabled=True)
        analyzer = BehaviorAnalyzer(config)
        track = Track(
            track_id="cam-1-42",
            bbox=(0, 0, 10, 10),
            center=(5, 5),
            timestamps=[],
            positions=[],
            last_seen=0.0,
            frames_alive=1,
        )
        assert analyzer._check_running([track], 1000.0) == []

    def test_diagonal_movement(self):
        """Diagonal movement is correctly measured with Euclidean distance."""
        config = _config(
            enabled=True,
            running_speed_threshold_px_per_sec=50,
        )
        analyzer = BehaviorAnalyzer(config)
        now = 1000.0
        # Diagonal: (0,0) → (300,400), distance = 500px, 5s → 100 px/s
        track = _make_track(
            track_id="cam-1-1",
            positions=[(0, 0), (300, 400)],
            timestamps=[now - 5.0, now],
        )
        events = analyzer._check_running([track], now)
        assert len(events) == 1


# ---------------------------------------------------------------------------
# Tests: cooldown
# ---------------------------------------------------------------------------


class TestCooldown:
    def test_suppresses_duplicate(self):
        """Same event_type+track_id within cooldown is suppressed."""
        config = _config(
            enabled=True,
            loitering_threshold_seconds=1.0,
            loitering_radius_px=100,
            event_cooldown_seconds=30.0,
        )
        analyzer = BehaviorAnalyzer(config)
        now = 1000.0
        track = _make_track(
            track_id="cam-1-42",
            positions=[(100, 100), (100, 100)],
            timestamps=[now - 10.0, now],
        )
        # First call — should emit
        assert len(analyzer._check_loitering([track], now)) == 1
        # Second call within cooldown — suppressed
        assert len(analyzer._check_loitering([track], now + 5.0)) == 0

    def test_expires_after_cooldown_period(self):
        """Same event re-emitted after cooldown expires."""
        config = _config(
            enabled=True,
            loitering_threshold_seconds=1.0,
            loitering_radius_px=100,
            event_cooldown_seconds=10.0,
        )
        analyzer = BehaviorAnalyzer(config)
        now = 1000.0
        track = _make_track(
            track_id="cam-1-42",
            positions=[(100, 100), (100, 100)],
            timestamps=[now - 10.0, now],
        )
        assert len(analyzer._check_loitering([track], now)) == 1
        # After cooldown expires
        assert len(analyzer._check_loitering([track], now + 15.0)) == 1

    def test_different_event_types_not_suppressed(self):
        """Different event types for same track are independent."""
        config = _config(
            enabled=True,
            loitering_threshold_seconds=1.0,
            loitering_radius_px=100,
            running_speed_threshold_px_per_sec=1,
            event_cooldown_seconds=30.0,
        )
        analyzer = BehaviorAnalyzer(config)
        now = 1000.0
        track = _make_track(
            track_id="cam-1-42",
            positions=[(100, 100), (100, 100)],
            timestamps=[now - 10.0, now],
        )
        # Emit loiter
        assert len(analyzer._check_loitering([track], now)) == 1
        # Running with same track — different event type, not suppressed
        # Need a fast track for running
        fast_track = _make_track(
            track_id="cam-1-42",
            positions=[(0, 0), (1000, 0)],
            timestamps=[now - 2.0, now],
        )
        assert len(analyzer._check_running([fast_track], now)) == 1


# ---------------------------------------------------------------------------
# Tests: integrated analyze()
# ---------------------------------------------------------------------------


class TestAnalyze:
    def test_integrates_all_checks(self):
        """analyze() calls all check methods and returns combined results."""
        config = _config(
            enabled=True,
            loitering_threshold_seconds=1.0,
            loitering_radius_px=100,
            running_speed_threshold_px_per_sec=1,
        )
        analyzer = BehaviorAnalyzer(config)
        now = 1000.0

        loiter_track = _make_track(
            track_id="cam-1-10",
            positions=[(100, 100), (100, 100)],
            timestamps=[now - 10.0, now],
        )
        run_track = _make_track(
            track_id="cam-2-20",
            positions=[(0, 0), (1000, 0)],
            timestamps=[now - 2.0, now],
        )
        events = analyzer.analyze(
            tracks=[loiter_track, run_track],
            frame_face_count=0,
            timestamp=now,
        )
        assert len(events) == 2
        event_types = {e["event_type"] for e in events}
        assert event_types == {"loiter", "running"}

    def test_empty_tracks(self):
        """analyze() with empty tracks returns []."""
        config = _config(enabled=True)
        analyzer = BehaviorAnalyzer(config)
        assert analyzer.analyze(tracks=[], frame_face_count=0, timestamp=1000.0) == []


# ---------------------------------------------------------------------------
# Tests: zone intrusion
# ---------------------------------------------------------------------------


class TestZoneIntrusion:
    """Zone intrusion detection using cv2.pointPolygonTest."""

    def test_track_inside_polygon_triggers_event(self):
        """Track center inside defined zone polygon -> zone event emitted."""
        config = _config(
            enabled=True,
            zones=[{"name": "entrance", "points": [[0, 0], [100, 0], [100, 100], [0, 100]]}],
        )
        analyzer = BehaviorAnalyzer(config)
        track = _make_track(
            track_id="cam-1-1",
            positions=[(50, 50), (50, 50)],
            timestamps=[1000.0, 1000.1],
        )
        events = analyzer._check_zone_intrusion([track])
        assert len(events) == 1
        ev = events[0]
        assert ev["event_type"] == "zone"
        assert ev["track_id"] == "cam-1-1"
        assert ev["camera_id"] == "1"
        assert "entrance" in ev["detail"]
        assert "zone" in ev["detail"].lower()

    def test_track_outside_polygon_no_event(self):
        """Track center outside all zones -> no event."""
        config = _config(
            enabled=True,
            zones=[{"name": "entrance", "points": [[0, 0], [100, 0], [100, 100], [0, 100]]}],
        )
        analyzer = BehaviorAnalyzer(config)
        track = _make_track(
            track_id="cam-1-1",
            positions=[(200, 200), (200, 200)],
            timestamps=[1000.0, 1000.1],
        )
        assert analyzer._check_zone_intrusion([track]) == []

    def test_empty_zones_list_no_event(self):
        """zones=[] -> no zone events."""
        config = _config(enabled=True, zones=[])
        analyzer = BehaviorAnalyzer(config)
        track = _make_track(
            track_id="cam-1-1",
            positions=[(50, 50)],
            timestamps=[1000.0],
        )
        assert analyzer._check_zone_intrusion([track]) == []

    def test_zone_with_invalid_points_skipped(self):
        """Zone with <3 points is skipped (no crash)."""
        config = _config(
            enabled=True,
            zones=[
                {"name": "bad", "points": [[0, 0], [100, 0]]},
                {"name": "good", "points": [[0, 0], [100, 0], [100, 100], [0, 100]]},
            ],
        )
        analyzer = BehaviorAnalyzer(config)
        track = _make_track(
            track_id="cam-1-1",
            positions=[(50, 50), (50, 50)],
            timestamps=[1000.0, 1000.1],
        )
        events = analyzer._check_zone_intrusion([track])
        assert len(events) == 1  # only 'good' zone fires
        assert "good" in events[0]["detail"]

    def test_zone_intrusion_cooldown(self):
        """Same track+zone event suppressed within cooldown."""
        config = _config(
            enabled=True,
            zones=[{"name": "entrance", "points": [[0, 0], [100, 0], [100, 100], [0, 100]]}],
            event_cooldown_seconds=30.0,
        )
        analyzer = BehaviorAnalyzer(config)
        now = 1000.0

        track1 = _make_track(
            track_id="cam-1-1",
            positions=[(50, 50)],
            timestamps=[now],
        )
        assert len(analyzer._check_zone_intrusion([track1])) == 1

        # Within cooldown
        track2 = _make_track(
            track_id="cam-1-1",
            positions=[(50, 50)],
            timestamps=[now + 5.0],
        )
        assert len(analyzer._check_zone_intrusion([track2])) == 0

    def test_zone_intrusion_after_cooldown(self):
        """Same track+zone event emitted after cooldown expires."""
        config = _config(
            enabled=True,
            zones=[{"name": "entrance", "points": [[0, 0], [100, 0], [100, 100], [0, 100]]}],
            event_cooldown_seconds=10.0,
        )
        analyzer = BehaviorAnalyzer(config)
        now = 1000.0

        track1 = _make_track(
            track_id="cam-1-1",
            positions=[(50, 50)],
            timestamps=[now],
        )
        assert len(analyzer._check_zone_intrusion([track1])) == 1

        # After cooldown expires
        track2 = _make_track(
            track_id="cam-1-1",
            positions=[(50, 50)],
            timestamps=[now + 15.0],
        )
        assert len(analyzer._check_zone_intrusion([track2])) == 1

    def test_disabled_config_returns_empty(self):
        """When behavior.enabled=False, zone intrusion returns []."""
        config = _config(
            enabled=False,
            zones=[{"name": "entrance", "points": [[0, 0], [100, 0], [100, 100], [0, 100]]}],
        )
        analyzer = BehaviorAnalyzer(config)
        track = _make_track(
            track_id="cam-1-1",
            positions=[(50, 50)],
            timestamps=[1000.0],
        )
        assert analyzer._check_zone_intrusion([track]) == []

    def test_track_at_polygon_edge_triggers_event(self):
        """Track center exactly on polygon edge (dist=0) triggers event."""
        config = _config(
            enabled=True,
            zones=[{"name": "edge", "points": [[0, 0], [100, 0], [100, 100], [0, 100]]}],
        )
        analyzer = BehaviorAnalyzer(config)
        # On the top edge of the square
        track = _make_track(
            track_id="cam-1-1",
            positions=[(50, 0), (50, 0)],
            timestamps=[1000.0, 1000.1],
        )
        events = analyzer._check_zone_intrusion([track])
        assert len(events) == 1


# ---------------------------------------------------------------------------
# Tests: crowd
# ---------------------------------------------------------------------------


class TestCrowd:
    """Crowd detection with debounce logic."""

    def test_crowd_triggered_after_debounce(self):
        """face_count >= threshold for debounce_frames -> event."""
        config = _config(
            enabled=True,
            crowd_threshold_count=3,
            crowd_debounce_frames=2,
        )
        analyzer = BehaviorAnalyzer(config)
        now = 1000.0

        # Frame 1: below debounce
        assert len(analyzer._check_crowd(5, now)) == 0
        # Frame 2: reaches debounce -> event
        events = analyzer._check_crowd(5, now + 0.1)
        assert len(events) == 1
        ev = events[0]
        assert ev["event_type"] == "crowd"
        assert ev["track_id"] == ""
        assert "Crowd" in ev["detail"]
        assert "5" in ev["detail"]
        assert ev["confidence"] == 1.0

    def test_crowd_not_triggered_below_threshold(self):
        """face_count < threshold -> no event."""
        config = _config(
            enabled=True,
            crowd_threshold_count=5,
        )
        analyzer = BehaviorAnalyzer(config)
        assert analyzer._check_crowd(3, 1000.0) == []

    def test_crowd_not_triggered_before_debounce(self):
        """Not enough consecutive frames -> no event."""
        config = _config(
            enabled=True,
            crowd_threshold_count=3,
            crowd_debounce_frames=5,
        )
        analyzer = BehaviorAnalyzer(config)
        now = 1000.0
        for i in range(4):
            assert len(analyzer._check_crowd(5, now + i * 0.1)) == 0

    def test_crowd_cooldown(self):
        """Crowd event suppressed within cooldown."""
        config = _config(
            enabled=True,
            crowd_threshold_count=3,
            crowd_debounce_frames=2,
            event_cooldown_seconds=10.0,
        )
        analyzer = BehaviorAnalyzer(config)
        now = 1000.0

        # Build up and trigger
        assert len(analyzer._check_crowd(5, now)) == 0       # frame 1
        assert len(analyzer._check_crowd(5, now + 0.1)) == 1 # frame 2 -> event

        # Reset by dropping below threshold
        assert len(analyzer._check_crowd(0, now + 1.0)) == 0

        # Within cooldown, try to build up again
        assert len(analyzer._check_crowd(5, now + 2.0)) == 0  # frame 1
        assert len(analyzer._check_crowd(5, now + 2.1)) == 0  # frame 2, but cooldown active

    def test_crowd_cooldown_expires(self):
        """Crowd event emitted after cooldown expires."""
        config = _config(
            enabled=True,
            crowd_threshold_count=3,
            crowd_debounce_frames=2,
            event_cooldown_seconds=5.0,
        )
        analyzer = BehaviorAnalyzer(config)
        now = 1000.0

        # Build up and trigger
        assert len(analyzer._check_crowd(5, now)) == 0       # frame 1
        assert len(analyzer._check_crowd(5, now + 0.1)) == 1 # frame 2 -> event

        # Reset
        assert len(analyzer._check_crowd(0, now + 1.0)) == 0

        # After cooldown expires
        assert len(analyzer._check_crowd(5, now + 6.0)) == 0  # frame 1
        assert len(analyzer._check_crowd(5, now + 6.1)) == 1  # frame 2 -> event (cooldown expired)

    def test_crowd_resets_on_drop(self):
        """When face count drops below threshold, counter resets."""
        config = _config(
            enabled=True,
            crowd_threshold_count=3,
            crowd_debounce_frames=3,
        )
        analyzer = BehaviorAnalyzer(config)
        now = 1000.0

        # Build up 2 frames
        assert len(analyzer._check_crowd(5, now)) == 0
        assert len(analyzer._check_crowd(5, now + 0.1)) == 0

        # Drop below threshold -> resets
        assert len(analyzer._check_crowd(0, now + 0.2)) == 0

        # After reset, needs full debounce again
        assert len(analyzer._check_crowd(5, now + 0.3)) == 0  # frame 1
        assert len(analyzer._check_crowd(5, now + 0.4)) == 0  # frame 2
        assert len(analyzer._check_crowd(5, now + 0.5)) == 1  # frame 3 -> event

    def test_disabled_config_returns_empty(self):
        """When behavior.enabled=False, crowd returns []."""
        config = _config(
            enabled=False,
            crowd_threshold_count=1,
        )
        analyzer = BehaviorAnalyzer(config)
        assert analyzer._check_crowd(10, 1000.0) == []


# ---------------------------------------------------------------------------
# Tests: stubs
# ---------------------------------------------------------------------------


class TestStubs:
    def test_zone_intrusion_stub(self):
        """_check_zone_intrusion returns empty list (stub)."""
        config = _config(enabled=True)
        analyzer = BehaviorAnalyzer(config)
        assert analyzer._check_zone_intrusion([]) == []

    def test_crowd_stub(self):
        """_check_crowd returns empty list (stub)."""
        config = _config(enabled=True)
        analyzer = BehaviorAnalyzer(config)
        assert analyzer._check_crowd(0, 1000.0) == []


# ---------------------------------------------------------------------------
# Tests: default config values
# ---------------------------------------------------------------------------


class TestDefaultConfig:
    def test_default_enabled_is_false(self):
        """Default BehaviorConfig has enabled=False."""
        config = BehaviorConfig()
        assert config.enabled is False

    def test_default_thresholds(self):
        """Default thresholds match plan specification."""
        config = BehaviorConfig()
        assert config.loitering_threshold_seconds == 30.0
        assert config.loitering_radius_px == 50.0
        assert config.running_speed_threshold_px_per_sec == 200.0
        assert config.event_cooldown_seconds == 30.0
