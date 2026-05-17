"""Kafka consumer pipeline for face recognition.

Consumes frames from ``t_dorm_frame``, runs detection → feature extraction →
SIMS API matching → direction detection → and produces events to
``t_dorm_event``.
"""

import base64
import json
import logging
import signal
import time
from datetime import datetime

import cv2
import numpy as np
import structlog
from kafka import KafkaConsumer, KafkaProducer

from app.config import load_config
from app.behavior import BehaviorAnalyzer
from app.dedup import DedupFilter
from app.detector import FaceDetector
from app.direction import DirectionDetector
from app.event_publisher import BehaviorEventPublisher
from app.feature import FeatureExtractor
from app.matcher import FaceMatcher
from app.night_mode import NightModeEnhancer
from app.tracker import FaceTracker


def main():
    import argparse
    # ------------------------------------------------------------------
    # Load config
    # ------------------------------------------------------------------
    parser = argparse.ArgumentParser(description="Face Recognition Service")
    parser.add_argument("--config", default="config.yaml", help="Path to config file")
    args = parser.parse_args()
    cfg = load_config(args.config)

    # ------------------------------------------------------------------
    # Logging (structlog)
    # ------------------------------------------------------------------
    structlog.configure(
        wrapper_class=structlog.make_filtering_bound_logger(
            getattr(logging, cfg.log.level.upper(), logging.INFO)
        ),
        processors=[
            structlog.stdlib.add_log_level,
            structlog.dev.ConsoleRenderer(),
        ],
    )
    log = structlog.get_logger()

    # ------------------------------------------------------------------
    # Initialise components
    # ------------------------------------------------------------------
    detector = FaceDetector(
        model_path=cfg.detection.model_path,
        conf_threshold=cfg.detection.confidence_threshold,
        input_size=tuple(cfg.detection.input_size),
        min_face_size=cfg.detection.min_face_size,
        blur_threshold=cfg.detection.blur_threshold,
        nms_iou_threshold=cfg.detection.nms_iou_threshold,
    )

    extractor = FeatureExtractor(
        model_path=cfg.feature.model_path,
        embedding_size=cfg.feature.embedding_size,
    )

    matcher = FaceMatcher(cfg.match)
    direction = DirectionDetector(cfg.direction)
    dedup = DedupFilter(cfg.dedup)
    enhancer = NightModeEnhancer(cfg.night_mode)

    # Behaviour components (only when enabled)
    tracker = (
        FaceTracker(iou_threshold=0.3, track_ttl=5.0)
        if cfg.behavior.enabled
        else None
    )
    behavior_analyzer = (
        BehaviorAnalyzer(cfg.behavior)
        if cfg.behavior.enabled
        else None
    )
    event_publisher = (
        BehaviorEventPublisher(cfg)
        if cfg.behavior.enabled
        else None
    )

    # ------------------------------------------------------------------
    # Kafka consumer & producer
    # ------------------------------------------------------------------
    consumer = KafkaConsumer(
        cfg.kafka.frame_topic,
        bootstrap_servers=cfg.kafka.brokers,
        group_id=cfg.kafka.group_id,
        value_deserializer=lambda m: json.loads(m.decode()),
        max_poll_records=cfg.kafka.max_poll_records,
        auto_offset_reset="latest",
    )

    producer = KafkaProducer(
        bootstrap_servers=cfg.kafka.brokers,
        value_serializer=lambda v: json.dumps(v).encode(),
    )

    log.info(
        "face_recognition_started",
        frame_topic=cfg.kafka.frame_topic,
        event_topic=cfg.kafka.event_topic,
        brokers=cfg.kafka.brokers,
    )

    # ------------------------------------------------------------------
    # Graceful shutdown
    # ------------------------------------------------------------------
    running = True

    def _signal_handler(signum, _frame):
        nonlocal running
        log.info("shutdown_signal_received", signal=signum)
        running = False

    signal.signal(signal.SIGINT, _signal_handler)
    signal.signal(signal.SIGTERM, _signal_handler)

    # ------------------------------------------------------------------
    # Stats accumulator
    # ------------------------------------------------------------------
    stats = {
        "frames_processed": 0,
        "faces_detected": 0,
        "events_produced": 0,
        "matches_found": 0,
        "strangers": 0,
        "behavior_events": 0,
        "last_log_time": time.time(),
    }

    # ------------------------------------------------------------------
    # Per-frame processing
    # ------------------------------------------------------------------
    def process_frame(msg: dict):
        nonlocal stats, tracker, behavior_analyzer, event_publisher

        # Decode base64 JPEG -> numpy array (BGR)
        frame_bytes = base64.b64decode(msg["frame_data"])
        np_arr = np.frombuffer(frame_bytes, dtype=np.uint8)
        frame = cv2.imdecode(np_arr, cv2.IMREAD_COLOR)
        if frame is None:
            return

        # Night-mode enhancement
        frame = enhancer.enhance(frame)

        # Face detection
        faces = detector.detect(frame)
        if not faces:
            return

        stats["faces_detected"] += len(faces)

        # Batch-extract embeddings for all faces
        embeddings = []
        for face in faces:
            emb = extractor.extract(frame, face)
            embeddings.append(emb)

        camera_id = msg["camera_id"]
        timestamp = time.time()

        # --- BEHAVIOUR PIPELINE (only when enabled) ---
        if tracker and behavior_analyzer and event_publisher:
            tracks = tracker.update(faces, embeddings, camera_id, timestamp)
            behavior_events = behavior_analyzer.analyze(
                tracks, len(faces), timestamp
            )
            for be in behavior_events:
                be["camera_id"] = camera_id
                event_publisher.publish_behavior_event(be)
                stats["behavior_events"] += 1

        # --- EXISTING ENTRY/EXIT PIPELINE (always runs) ---
        for i, face in enumerate(faces):
            embedding = embeddings[i]

            # Identity matching
            match_result = matcher.match(embedding)
            if match_result:
                stats["matches_found"] += 1

            # Use tracker-generated track_id when tracker active
            face_center_x = (face.x1 + face.x2) / 2.0
            face_center_y = (face.y1 + face.y2) / 2.0
            if tracker:
                active_tracks = tracker.get_tracks()
                if i < len(active_tracks):
                    face_id = active_tracks[i].track_id
                else:
                    face_id = camera_id
            else:
                face_id = camera_id

            # Direction determination (ROI line crossing)
            direction_result = direction.determine(
                face_id, face_center_x, face_center_y, msg["frame_width"]
            )

            if direction_result is None:
                continue

            # Deduplication
            student_id = (
                match_result["student_id"]
                if match_result
                else f"stranger_{msg['building']}"
            )
            if dedup.is_duplicate(student_id, direction_result):
                continue
            dedup.mark_seen(student_id, direction_result)

            if match_result is None:
                stats["strangers"] += 1

            # Build & produce event
            event = {
                "camera_id": msg["camera_id"],
                "building": msg["building"],
                "event_type": direction_result,
                "student_id": match_result["student_id"] if match_result else None,
                "name": match_result["name"] if match_result else None,
                "confidence": match_result["confidence"] if match_result else 0.0,
                "timestamp": int(time.time() * 1000),
                "frame_sequence": msg["frame_sequence"],
                "is_stranger": match_result is None,
                "snapshot_path": "",
                "direction_method": "roi_line",
            }

            producer.send(cfg.kafka.event_topic, value=event)
            stats["events_produced"] += 1

    # ------------------------------------------------------------------
    # Main loop
    # ------------------------------------------------------------------
    try:
        while running:
            msg_pack = consumer.poll(timeout_ms=1000)

            for _tp, messages in msg_pack.items():
                for raw_msg in messages:
                    process_frame(raw_msg.value)
                    stats["frames_processed"] += 1

            # Periodic maintenance
            direction.cleanup()
            dedup.cleanup()
            if tracker:
                tracker.cleanup()

            # Stats logging every 60 seconds
            now = time.time()
            if now - stats["last_log_time"] >= 60:
                elapsed = int(now - stats["last_log_time"])
                log.info(
                    "processing_stats",
                    frames_processed=stats["frames_processed"],
                    faces_detected=stats["faces_detected"],
                    events_produced=stats["events_produced"],
                    matches_found=stats["matches_found"],
                    strangers=stats["strangers"],
                    behavior_events=stats["behavior_events"],
                    elapsed_seconds=elapsed,
                )
                stats["last_log_time"] = now

    except Exception:
        log.error("unexpected_error", exc_info=True)
        raise
    finally:
        log.info("shutting_down")
        consumer.close()
        producer.close()
        if event_publisher:
            event_publisher.close()
        log.info("shutdown_complete")


if __name__ == "__main__":
    main()
