"""Tests for ``BehaviorEventPublisher``."""

import time
from unittest.mock import MagicMock, call, patch

import pytest

from app.config import AppConfig
from app.event_publisher import BehaviorEventPublisher


@pytest.fixture
def config():
    """AppConfig with behavior enabled (default mappings, 30 s cooldown)."""
    cfg = AppConfig()
    cfg.behavior.enabled = True
    cfg.behavior.event_cooldown_seconds = 30.0
    return cfg


# ---------------------------------------------------------------------------
# Behavior disabled
# ---------------------------------------------------------------------------


class TestBehaviorDisabled:
    """When ``behavior.enabled`` is ``False`` nothing is published or buffered."""

    def test_no_publish_when_disabled(self, config):
        config.behavior.enabled = False
        publisher = BehaviorEventPublisher(config)
        publisher.publish_behavior_event(
            {
                "camera_id": "cam_01",
                "event_type": "loitering",
                "timestamp": int(time.time() * 1000),
            }
        )
        assert len(publisher._buffer) == 0

    def test_no_kafka_producer_when_disabled(self, config):
        """Assert producer is None when behavior disabled (no connection attempt)."""
        config.behavior.enabled = False
        with patch("kafka.KafkaProducer") as mock_cls:
            publisher = BehaviorEventPublisher(config)
            mock_cls.assert_not_called()
            assert publisher._producer is None


# ---------------------------------------------------------------------------
# event_type ≤ 8 chars assertion
# ---------------------------------------------------------------------------


class TestEventTypeAssertion:
    """event_type must be ≤ 8 characters (DB ``VARCHAR(8)`` constraint)."""

    def test_short_event_type_passes(self, config):
        """Mapped event_type (loitering→loiter=6) does not raise."""
        with patch("kafka.KafkaProducer") as mock_kafka:
            producer_instance = MagicMock()
            mock_kafka.return_value = producer_instance
            publisher = BehaviorEventPublisher(config)
            publisher.publish_behavior_event(
                {
                    "camera_id": "cam_01",
                    "event_type": "loitering",
                    "timestamp": int(time.time() * 1000),
                }
            )
            # Should not raise AssertionError

    def test_exactly_8_chars_passes(self, config):
        """Edge case: exactly 8 characters passes."""
        config.behavior.event.event_type_mapping = {}
        with patch("kafka.KafkaProducer"):
            publisher = BehaviorEventPublisher(config)
            publisher.publish_behavior_event(
                {
                    "camera_id": "cam_01",
                    "event_type": "ABCDEFGH",  # exactly 8
                    "timestamp": int(time.time() * 1000),
                }
            )
        # Should not raise

    def test_long_event_type_raises(self, config):
        """Event_type > 8 chars (and not in mapping) raises AssertionError."""
        config.behavior.event.event_type_mapping = {}
        with patch("kafka.KafkaProducer"):
            publisher = BehaviorEventPublisher(config)
            with pytest.raises(AssertionError, match="exceeds 8 char limit"):
                publisher.publish_behavior_event(
                    {
                        "camera_id": "cam_01",
                        "event_type": "too_long_event_type",
                        "timestamp": int(time.time() * 1000),
                    }
                )

    def test_mapped_to_short_value_respected(self, config):
        """Mapping transforms a long event_type to a short one; assertion passes."""
        with patch("kafka.KafkaProducer") as mock_kafka:
            producer_instance = MagicMock()
            mock_kafka.return_value = producer_instance
            publisher = BehaviorEventPublisher(config)
            # "zone_intrusion" (14 chars) → "zone" (4) via mapping
            publisher.publish_behavior_event(
                {
                    "camera_id": "cam_01",
                    "event_type": "zone_intrusion",
                    "timestamp": int(time.time() * 1000),
                }
            )
            sent = producer_instance.send.call_args[1]["value"]
            assert sent["event_type"] == "zone"


# ---------------------------------------------------------------------------
# Event-type mapping
# ---------------------------------------------------------------------------


class TestEventTypeMapping:
    """Event_type_mapping transforms long names to short codes before publish."""

    def test_mapping_applied_on_send(self, config):
        with patch("kafka.KafkaProducer") as mock_kafka:
            producer_instance = MagicMock()
            mock_kafka.return_value = producer_instance
            publisher = BehaviorEventPublisher(config)

            publisher.publish_behavior_event(
                {
                    "camera_id": "cam_01",
                    "event_type": "loitering",
                    "timestamp": int(time.time() * 1000),
                }
            )

            sent = producer_instance.send.call_args[1]["value"]
            assert sent["event_type"] == "loiter"

    def test_mapping_applied_on_buffer(self, config):
        """Mapping is applied even when event goes to buffer (Kafka unavailable)."""
        with patch("kafka.KafkaProducer", side_effect=Exception("No Kafka")):
            publisher = BehaviorEventPublisher(config)
            publisher.publish_behavior_event(
                {
                    "camera_id": "cam_01",
                    "event_type": "crowd_alert",
                    "timestamp": int(time.time() * 1000),
                }
            )
            assert len(publisher._buffer) == 1
            assert publisher._buffer[0]["event_type"] == "crowd"

    def test_unmapped_event_type_used_as_is(self, config):
        """Event_type not in mapping is passed through unchanged."""
        config.behavior.event.event_type_mapping = {}
        with patch("kafka.KafkaProducer") as mock_kafka:
            producer_instance = MagicMock()
            mock_kafka.return_value = producer_instance
            publisher = BehaviorEventPublisher(config)

            publisher.publish_behavior_event(
                {
                    "camera_id": "cam_01",
                    "event_type": "running",
                    "timestamp": int(time.time() * 1000),
                }
            )

            sent = producer_instance.send.call_args[1]["value"]
            assert sent["event_type"] == "running"


# ---------------------------------------------------------------------------
# Kafka send / buffer
# ---------------------------------------------------------------------------


class TestKafkaPublish:
    """Successful Kafka send path."""

    def test_sends_to_event_topic(self, config):
        with patch("kafka.KafkaProducer") as mock_kafka:
            producer_instance = MagicMock()
            mock_kafka.return_value = producer_instance
            publisher = BehaviorEventPublisher(config)

            publisher.publish_behavior_event(
                {
                    "camera_id": "cam_01",
                    "event_type": "running",
                    "timestamp": int(time.time() * 1000),
                }
            )

            topic_arg = producer_instance.send.call_args[0][0]
            assert topic_arg == "t_dorm_event"

    def test_source_field_present(self, config):
        with patch("kafka.KafkaProducer") as mock_kafka:
            producer_instance = MagicMock()
            mock_kafka.return_value = producer_instance
            publisher = BehaviorEventPublisher(config)

            publisher.publish_behavior_event(
                {
                    "camera_id": "cam_01",
                    "event_type": "running",
                    "timestamp": int(time.time() * 1000),
                }
            )

            sent = producer_instance.send.call_args[1]["value"]
            assert sent["source"] == "behavior"

    def test_confidence_defaults_to_1_dot_0(self, config):
        with patch("kafka.KafkaProducer") as mock_kafka:
            producer_instance = MagicMock()
            mock_kafka.return_value = producer_instance
            publisher = BehaviorEventPublisher(config)

            publisher.publish_behavior_event(
                {
                    "camera_id": "cam_01",
                    "event_type": "running",
                    "timestamp": int(time.time() * 1000),
                }
            )

            sent = producer_instance.send.call_args[1]["value"]
            assert sent["confidence"] == 1.0

    def test_custom_confidence_passed_through(self, config):
        with patch("kafka.KafkaProducer") as mock_kafka:
            producer_instance = MagicMock()
            mock_kafka.return_value = producer_instance
            publisher = BehaviorEventPublisher(config)

            publisher.publish_behavior_event(
                {
                    "camera_id": "cam_01",
                    "event_type": "running",
                    "confidence": 0.87,
                    "timestamp": int(time.time() * 1000),
                }
            )

            sent = producer_instance.send.call_args[1]["value"]
            assert sent["confidence"] == 0.87

    def test_track_id_and_detail_passed_through(self, config):
        with patch("kafka.KafkaProducer") as mock_kafka:
            producer_instance = MagicMock()
            mock_kafka.return_value = producer_instance
            publisher = BehaviorEventPublisher(config)

            publisher.publish_behavior_event(
                {
                    "camera_id": "cam_01",
                    "event_type": "running",
                    "track_id": "track_42",
                    "detail": "Person running in hallway",
                    "timestamp": int(time.time() * 1000),
                }
            )

            sent = producer_instance.send.call_args[1]["value"]
            assert sent["track_id"] == "track_42"
            assert sent["detail"] == "Person running in hallway"


# ---------------------------------------------------------------------------
# Buffer on failure
# ---------------------------------------------------------------------------


class TestBufferOnFailure:
    """Events are buffered when Kafka is unavailable or send fails."""

    def test_kafka_unavailable_buffers(self, config):
        """KafkaProducer init fails → producer is None → events go to buffer."""
        with patch("kafka.KafkaProducer", side_effect=Exception("Connection refused")):
            publisher = BehaviorEventPublisher(config)

            publisher.publish_behavior_event(
                {
                    "camera_id": "cam_01",
                    "event_type": "running",
                    "timestamp": int(time.time() * 1000),
                }
            )

            assert len(publisher._buffer) == 1
            assert publisher._producer is None

    def test_kafka_send_failure_buffers(self, config):
        """KafkaProducer.send() raises → event is buffered."""
        with patch("kafka.KafkaProducer") as mock_kafka:
            producer_instance = MagicMock()
            producer_instance.send.side_effect = Exception("Kafka timeout")
            mock_kafka.return_value = producer_instance
            publisher = BehaviorEventPublisher(config)

            publisher.publish_behavior_event(
                {
                    "camera_id": "cam_01",
                    "event_type": "running",
                    "timestamp": int(time.time() * 1000),
                }
            )

            assert len(publisher._buffer) == 1

    def test_buffer_overflow_evicts_oldest(self, config):
        """Buffer > 1000 items → oldest (FIFO) is evicted."""
        with patch("kafka.KafkaProducer", side_effect=Exception("No Kafka")):
            publisher = BehaviorEventPublisher(config)

            for i in range(1000):
                publisher.publish_behavior_event(
                    {
                        "camera_id": f"cam_{i % 4}",
                        "event_type": f"ev_{i}",
                        "timestamp": i,
                    }
                )

            assert len(publisher._buffer) == 1000

            publisher.publish_behavior_event(
                {
                    "camera_id": "cam_new",
                    "event_type": "ev_1000",
                    "timestamp": 9999,
                }
            )

            assert len(publisher._buffer) == 1000
            timestamps = [e["timestamp"] for e in publisher._buffer]
            assert min(timestamps) == 1  # timestamp 0 evicted
            assert max(timestamps) == 9999  # newest event present


# ---------------------------------------------------------------------------
# Flush buffer on successful send
# ---------------------------------------------------------------------------


class TestFlushBuffer:
    """Buffered events are flushed on the next successful publish."""

    def test_flush_on_successful_send(self, config):
        with patch("kafka.KafkaProducer") as mock_kafka:
            producer_instance = MagicMock()
            # First send fails → event gets buffered
            producer_instance.send.side_effect = [Exception("Fail"), None]
            mock_kafka.return_value = producer_instance
            publisher = BehaviorEventPublisher(config)

            publisher.publish_behavior_event(
                {
                    "camera_id": "cam_01",
                    "event_type": "running",
                    "timestamp": 100,
                }
            )
            assert len(publisher._buffer) == 1

            # Second event: send should succeed and flush buffer
            producer_instance.send.side_effect = None
            publisher.publish_behavior_event(
                {
                    "camera_id": "cam_02",
                    "event_type": "crowd",
                    "timestamp": 200,
                }
            )

            assert len(publisher._buffer) == 0

    def test_close_flushes_buffer(self, config):
        """close() tries to flush buffered events then closes producer."""
        with patch("kafka.KafkaProducer") as mock_kafka:
            producer_instance = MagicMock()
            mock_kafka.return_value = producer_instance
            publisher = BehaviorEventPublisher(config)

            publisher._buffer = [
                {"event_type": "loiter", "timestamp": 100},
                {"event_type": "running", "timestamp": 200},
            ]

            publisher.close()

            # Flush should have been attempted (send called for each buffered)
            assert producer_instance.send.call_count >= 2
            producer_instance.flush.assert_called_once()
            producer_instance.close.assert_called_once()
            assert publisher._producer is None
            assert len(publisher._buffer) == 0

    def test_close_idempotent(self, config):
        """Calling close() twice does not crash."""
        with patch("kafka.KafkaProducer") as mock_kafka:
            producer_instance = MagicMock()
            mock_kafka.return_value = producer_instance
            publisher = BehaviorEventPublisher(config)

            publisher.close()
            publisher.close()  # second call
            producer_instance.close.assert_called_once()


# ---------------------------------------------------------------------------
# Cooldown
# ---------------------------------------------------------------------------


class TestCooldown:
    """Cooldown suppresses duplicate event_types within the time window."""

    def test_suppresses_duplicate_within_cooldown(self, config):
        with patch("kafka.KafkaProducer") as mock_kafka:
            producer_instance = MagicMock()
            mock_kafka.return_value = producer_instance
            publisher = BehaviorEventPublisher(config)

            publisher.publish_behavior_event(
                {
                    "camera_id": "cam_01",
                    "event_type": "running",
                    "timestamp": 100,
                }
            )
            assert producer_instance.send.call_count == 1

            publisher.publish_behavior_event(
                {
                    "camera_id": "cam_01",
                    "event_type": "running",
                    "timestamp": 200,
                }
            )
            assert producer_instance.send.call_count == 1  # suppressed

    def test_cooldown_expires_allows_publish(self, config):
        with patch("kafka.KafkaProducer") as mock_kafka:
            producer_instance = MagicMock()
            mock_kafka.return_value = producer_instance
            publisher = BehaviorEventPublisher(config)

            publisher.publish_behavior_event(
                {
                    "camera_id": "cam_01",
                    "event_type": "running",
                    "timestamp": 100,
                }
            )
            assert producer_instance.send.call_count == 1

            # Simulate cooldown expiry (> 30 s)
            publisher._last_publish_time["running"] = time.time() - 60

            publisher.publish_behavior_event(
                {
                    "camera_id": "cam_01",
                    "event_type": "running",
                    "timestamp": 200,
                }
            )
            assert producer_instance.send.call_count == 2  # published again

    def test_different_event_types_not_suppressed(self, config):
        with patch("kafka.KafkaProducer") as mock_kafka:
            producer_instance = MagicMock()
            mock_kafka.return_value = producer_instance
            publisher = BehaviorEventPublisher(config)

            publisher.publish_behavior_event(
                {
                    "camera_id": "cam_01",
                    "event_type": "running",
                    "timestamp": 100,
                }
            )
            publisher.publish_behavior_event(
                {
                    "camera_id": "cam_01",
                    "event_type": "crowd",  # different type → no suppression
                    "timestamp": 200,
                }
            )
            assert producer_instance.send.call_count == 2

    def test_cooldown_not_updated_on_failure(self, config):
        """_last_publish_time is NOT updated when Kafka send fails."""
        with patch("kafka.KafkaProducer") as mock_kafka:
            producer_instance = MagicMock()
            mock_kafka.return_value = producer_instance
            publisher = BehaviorEventPublisher(config)

            # Kafka send fails
            producer_instance.send.side_effect = Exception("Kafka down")
            publisher.publish_behavior_event(
                {
                    "camera_id": "cam_01",
                    "event_type": "running",
                    "timestamp": 100,
                }
            )

            # _last_publish_time should NOT have been updated
            assert "running" not in publisher._last_publish_time

    def test_cooldown_uses_mapped_event_type(self, config):
        """Cooldown key is the mapped (short) event_type, not the original."""
        with patch("kafka.KafkaProducer") as mock_kafka:
            producer_instance = MagicMock()
            mock_kafka.return_value = producer_instance
            publisher = BehaviorEventPublisher(config)

            # "loitering" → mapped to "loiter"
            publisher.publish_behavior_event(
                {
                    "camera_id": "cam_01",
                    "event_type": "loitering",
                    "timestamp": 100,
                }
            )

            # Same mapped type "loiter" should be suppressed
            publisher.publish_behavior_event(
                {
                    "camera_id": "cam_01",
                    "event_type": "loitering",
                    "timestamp": 200,
                }
            )

            assert producer_instance.send.call_count == 1
