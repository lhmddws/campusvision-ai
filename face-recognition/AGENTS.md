# face-recognition — AGENTS.md
Python service for real-time face detection, recognition, and behavior analysis from Kafka frame streams.

## Architecture
```
t_dorm_frame (Kafka) → frame decode → face detect → embed → match → behavior analyze → t_dorm_event (Kafka)
```

**11-step pipeline:**
1. Kafka consumer polls `t_dorm_frame` (partition keyed by `building`)
2. Base64 decode `frame_data` → JPEG → BGR numpy array
3. Night-mode enhancement (CLAHE) when `night_mode.enabled` and hour in range
4. Face detection: ONNX RetinaFace → Haar Cascade fallback
5. Batch embedding extraction for all detected faces
6. Identity matching via `POST /api/face/match` (dormitory-service-go:8083)
7. Direction detection (ROI line crossing) → entry/exit determination
8. Deduplication via Redis (key: `dedup:{student_id}:{direction}:{camera_id}`)
9. Behavior analysis (loitering, running, zone intrusion, crowd) — only when enabled
10. Build event JSON with student info, confidence, snapshot path
11. Kafka produce to `t_dorm_event` (raw JSON, no Spring headers)

**Dual detection:** ONNX RetinaFace (primary) → `cv2.data.haarcascades` Haar Cascade (fallback when `model_path=""`)

**Dual embedding:** ONNX ArcFace (primary) → zero-vector fallback (returns null embedding when model missing)

## Entry Points
- `python -m app.main --config config.yaml` — main service (Kafka consumer/producer loop, no HTTP port)
- `python -m app.download_models` — downloads ONNX models from `app/models/model_urls.yaml` with SHA256 verification
- **Docker CMD gap:** `CMD ["python", "-m", "app.main"]` (line 24) omits `--config` flag — relies on bind-mounted `config.docker.yaml`

## Key Modules
| Module | Purpose | Gotchas |
|--------|---------|---------|
| `app/main.py` | Service entry point, Kafka consumer/producer loop | Argparse `--config` flag (default: `config.yaml`); behavior components (tracker, analyzer, publisher) only initialized when `cfg.behavior.enabled` |
| `app/detector.py` | Face detection (ONNX + Haar fallback) | Haar uses `cv2.data.haarcascades` — fails on non-macOS or different OpenCV versions; tests rely on this fallback |
| `app/embedder.py` | Face embedding (ONNX + zero-vector fallback) | Returns null embedding when model missing; batch extraction for all faces in frame |
| `app/matcher.py` | Identity matching via external API + Redis cache | Calls `POST /api/face/match` on dormitory-service-go:8083; falls back to Redis cache scan when `fallback_to_cache: true` and API fails |
| `app/behavior.py` | Behavior analysis (loitering, running, zone intrusion, crowd) | Gated by `behavior.enabled: false` in config — tracker, analyzer, publisher all inert when disabled |
| `app/download_models.py` | ONNX model downloader with SHA256 verification | `PLACEHOLDER_SHA256` sentinel (line 30) is dead code — models in `model_urls.yaml` have real hashes |
| `app/config.py` | 12 dataclass config definitions | All fields have defaults; YAML overrides; nested `BehaviorEventConfig` in `BehaviorConfig`; loaded via `load_config()` |
| `app/tracker.py` | Face tracking across frames (IoU-based) | Only initialized when behavior analysis enabled; generates `track_id` for behavior events |
| `app/dedup.py` | Redis-based event deduplication | Key format: `dedup:{student_id}:{direction}:{camera_id}`; TTL from `dedup.window_seconds` |
| `app/direction.py` | Entry/exit determination via ROI line crossing | Uses `roi_line_x` (default 0.5 = center vertical line); tracks face centroid crossing |

## Critical Gotchas
1. **Haar Cascade path**: `detector.py:255-257` uses `cv2.data.haarcascades + "haarcascade_frontalface_default.xml"` — OpenCV-bundled path, not hardcoded Homebrew; tests rely on this fallback when ONNX unavailable.
2. **Docker CMD missing --config**: Dockerfile line 24 lacks `--config` flag — relies on bind-mounted `config.docker.yaml` (copied at build time) or default `config.yaml`.
3. **External API dependency**: `matcher.py:80-97` calls `POST /api/face/match` on dormitory-service-go:8083. Falls back to Redis cache scan when `fallback_to_cache: true` and API fails (timeout, connection error).
4. **Stub embed endpoint**: `POST /api/face/embed` exists in dormitory-service-go but returns null embedding — not used by face-recognition (embedding done locally via ONNX).
5. **Behavior analysis disabled**: `behavior.enabled: false` by default in config (`config.py:99`) — tracker, analyzer, publisher only initialized when enabled; pipeline runs entry/exit only.
6. **Dead sentinel code**: `download_models.py:30` has `PLACEHOLDER_SHA256` — never triggered (models in `model_urls.yaml` lines 4, 10 have real SHA256 hashes).
7. **Raw JSON Kafka messages**: Producer publishes raw JSON (no Spring Kafka type headers) — Go consumer in dormitory-service-go must handle this format (no `__TypeId__` header).
8. **Matcher API URL**: `match.sims_api_url` defaults to empty string — must be set in config or via env; uses `httpx` with `match.sims_api_timeout` (default 3.0s).

## Testing
**7 test files** under `tests/`:
- `conftest.py` — session-scoped fixtures (`face_detector`, `synthetic_image`, `blurry_image`)
- `test_detector.py` — Haar fallback detection, blur filtering
- `test_behavior.py` — loitering, running, zone intrusion, crowd detection
- `test_tracker.py` — IoU-based tracking, track expiry
- `test_event_publisher.py` — behavior event Kafka publishing
- `test_integration.py` — end-to-end pipeline (mocked Kafka)
- `manual_qa.py` — not pytest; manual verification script

**Run:** `cd face-recognition && pytest tests/`

**Key:** Tests use Haar Cascade fallback (no ONNX models needed) — `conftest.py:9-15` creates `FaceDetector(model_path="")`.

**Mocking:** `unittest.mock.patch` for Kafka consumer/producer; `pytest` fixtures for detectors and images.

**Coverage:** All `app/` modules have tests. `manual_qa.py` exists but is not pytest — manual verification script for QA testing.

## Configuration
**`config.yaml` fields:**
- `kafka.brokers`, `kafka.frame_topic`, `kafka.event_topic`, `kafka.group_id`, `kafka.max_poll_records`
- `redis.host`, `redis.port`, `redis.db`, `redis.socket_timeout`
- `detection.model_path`, `detection.confidence_threshold`, `detection.input_size`, `detection.min_face_size`, `detection.blur_threshold`, `detection.nms_iou_threshold`
- `feature.model_path`, `feature.embedding_size`
- `match.method`, `match.sims_api_url`, `match.sims_api_timeout`, `match.auth_token`, `match.cache_ttl`, `match.match_threshold`, `match.fallback_to_cache`
- `direction.method`, `direction.roi_line_x`, `direction.min_track_points`
- `dedup.window_seconds`, `dedup.max_cache_size`
- `stranger.enabled`, `stranger.alert_threshold`
- `night_mode.enabled`, `night_mode.start_hour`, `night_mode.end_hour`, `night_mode.clahe_clip_limit`
- `behavior.enabled`, `behavior.loitering_threshold_seconds`, `behavior.loitering_radius_px`, `behavior.running_speed_threshold_px_per_sec`, `behavior.crowd_threshold_count`, `behavior.zones`
- `log.level`

**12 dataclasses** in `app/config.py`: `KafkaConfig`, `DetectionConfig`, `FeatureConfig`, `MatchConfig`, `DirectionConfig`, `DedupConfig`, `StrangerConfig`, `NightModeConfig`, `RedisConfig`, `LogConfig`, `BehaviorEventConfig`, `BehaviorConfig` → all aggregated in `AppConfig`

**Docker:** `config.docker.yaml` bind-mounted at build time (Dockerfile line 14), overrides hosts/ports for container networking (kafka:9092, redis:6379)

**ONNX models:** Defined in `app/models/model_urls.yaml` (retinaface-R50, arcface-resnet100), downloaded at Docker build time (line 18) or via `python -m app.download_models`, files gitignored (`*.onnx`)

**See root AGENTS.md for:** Kafka topics table, Docker Compose services, team division, cross-cutting patterns, ONNX model management overview
