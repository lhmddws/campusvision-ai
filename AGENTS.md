# CampusVision AI — AGENTS.md

Multi-language monorepo (Go + Python + Java) for dormitory AI surveillance.
4 RTSP cameras → Stream Gateway → Kafka → Face Recognition → Kafka → Dormitory Service.

**Perception layer** (Go, Python) produces `t_dorm_event` → **Business layer** (Java) consumes it.

---

## Architecture (3-tier pipeline)

```
RTSP cameras (A/B/C/D) → stream-gateway (Go) → t_dorm_frame (Kafka)
  → face-recognition (Python) → t_dorm_event (Kafka)
  → dormitory-service (Java) → MariaDB + Redis
```

| Dir | Language | Entrypoint | Port |
|---|---|---|---|
| `stream-gateway/` | Go | `cmd/main.go` (via `go run cmd/main.go --config config.yaml`) | 8080 (health) |
| `face-recognition/` | Python | `app/main.py` (via `python -m app.main --config config.yaml`) | — |
| `dormitory-service/` | Java | `DormitoryServiceApplication.java` | 8081 |
| `test-env/` | Python | `server/main.py` (FastAPI, via `bash test-env/start.sh`) | 8082 |

Infra (`docker compose up -d`): Kafka (9092), Redis (6379), MariaDB (3306), MinIO (9000).

---

## Commands

```bash
# Infrastructure
docker compose up -d                           # start all: Kafka + Redis + MariaDB + MinIO
docker compose up -d kafka redis                # minimal for dev (no DB/MinIO)

# Stream Gateway (Go) — requires ffmpeg on $PATH
cd stream-gateway && go run cmd/main.go --config config.yaml

# Face Recognition (Python) — requires ONNX models first
cd face-recognition && python -m app.main --config config.yaml

# Model download (build-time in Docker, also runnable standalone)
cd face-recognition && python -m app.download_models

# Dormitory Service (Java) — requires mvn on $PATH (no .mvn wrapper)
cd dormitory-service && mvn compile
mvn package -DskipTests

# Test environment (simulates 4 cameras)
bash test-env/start.sh                          # Auto-detects Kafka/Redis, starts uvicorn :8082

# Kafka management
docker compose exec kafka kafka-topics --bootstrap-server localhost:9092 --list
docker compose exec kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic t_dorm_event --from-beginning
```

---

## Kafka topics (auto-created by docker-compose kafka-init)

| Topic | Partitions | Message fields | Retention | Max bytes | Producer → Consumer |
|---|---|---|---|---|---|
| `t_dorm_frame` | 4 | `camera_id, building, frame_data(jpeg base64)` | 12h | 5MB | stream-gateway → face-recognition |
| `t_dorm_event` | 2 | `camera_id, event_type, student_id, timestamp` | 7d | 1MB | face-recognition → dormitory-service |
| `t_dorm_alert` | 1 | action commands | 7d | — | dormitory-service → (future) |

- `t_dorm_frame` uses **hash partitioner** (`kafka.Hash{}`) keyed by `building` — same building always goes to same partition.
- Compression: `compression.type=producer` (Go uses Snappy for `t_dorm_frame`).

---

## Critical gotchas

### Python → Java Kafka Integration
- Python face-recognition publishes **raw JSON** (no Spring Kafka type headers).
- Java consumer **must** use `value-deserializer: StringDeserializer` + manual `ObjectMapper.readValue`.
- JSON uses **snake_case** → `FaceEventMessage.java` maps via `@JsonProperty("camera_id")` etc.
- Java producer uses `JsonSerializer`; consumer is `StringDeserializer` — asymmetric by design.
- Consumer uses **manual ack mode** (`ack-mode: manual`) with batch consumption (`max.poll.records=500`, concurrency=3).
- Java also does Redis-based dedup (TTL 3600s via `DEDUP_TTL_SECONDS`) keyed on `camera_id + frame_sequence`.

### JDK / Lombok Compatibility
- `pom.xml` targets JDK 17, dev machines run JDK 26.
- Lombok 1.18.30 (Spring Boot 3.2.5 default) **crashes** on JDK 26 with `TypeTag::UNKNOWN` — **Fix** (already applied): `<lombok.version>1.18.38</lombok.version>`.
- Surefire needs `-Dnet.bytebuddy.experimental=true` for Mockito on JDK 26 (already in pom.xml).
- No `.mvn/wrapper` — **Maven must be installed system-wide**.

### DB Schema Fragmentation (CRITICAL)
Three sources of truth that **disagree**:

1. **`infra/mariadb/init.sql`** (source of truth — docker-compose executes it) — 11 tables: `dorm_student_assignment`, `dorm_student_status`, `dorm_entry_exit_event`, `dorm_nightly_report`, `dorm_nightly_detail`, `dorm_stranger_record`, `dorm_alert_record`, `dorm_config`, `dorm_sync_log`, `dorm_camera`, `dorm_camera_log`.

2. **Java entity classes** — different table names and FKs:
   - `dorm_event_log` (entity) vs `dorm_entry_exit_event` (init.sql) — entity uses `building_id` FK, init.sql uses `building` VARCHAR
   - `dorm_student` (entity) vs `dorm_student_assignment` (init.sql) — entity references `building_id`/`room_id` FKs
   - `dorm_alert` (entity) vs `dorm_alert_record` (init.sql)
   - Entity schemas reference `dorm_building` and `dorm_room` tables that **don't exist in init.sql**

3. **JPA-style annotations**: Entities use `@TableName("dorm_*")` matching entity conventions, not init.sql.

**Always verify before using MyBatis Plus queries.** The `mapper/` directory under `src/main/resources` is **empty** — no XML mappers. All queries rely on `BaseMapper<T>` or `@Select` annotations with `map-underscore-to-camel-case: true`. `PaginationInnerInterceptor` uses `DbType.MARIADB`.

### Production Credentials in application.yml
- `dormitory-service/src/main/resources/application.yml` contains hardcoded production credentials (`jdbc:mariadb://192.168.113.114:3306/SIMS_CD250626`).
- Overridable via `SPRING_DATASOURCE_*` env vars.
- Local dev config (commented out) uses `mariadb:3306/dormitory`. Switch by hand.
- **Do not commit real credentials.**

### Redis Config
- Both face-recognition (`config.yaml`) and dormitory-service (`application.yml`) connect to `127.0.0.1:6379`, potentially same `db=0`. Separate DB index recommended for production.

### ONNX Model Management
- Models defined in `face-recognition/app/models/model_urls.yaml` — two models:
  - `retinaface-R50.onnx`: RetinaFace-ResNet50 face detection (from HivisionIDPhotos on HuggingFace)
  - `arcface-resnet100.onnx`: ArcFace-ResNet100 feature extraction (from onnxmodelzoo on HuggingFace)
- Downloaded at Docker build time via `python -m app.download_models` or standalone.
- SHA256 verification with a `PLACEHOLDER_UPDATE_ME` sentinel for unverified hashes.
- Models stored in `app/models/` — gitignored via `*.onnx`.

### Face Detector Fallback
- When no ONNX model is available (`model_path: ""`), falls back to **Haar Cascade** classifier.
- **Hardcoded macOS path** at `detector.py:256`: `/opt/homebrew/Cellar/opencv/4.13.0_8/share/opencv4/haarcascades/haarcascade_frontalface_default.xml`
- This **will fail on non-macOS or different OpenCV versions**. The tests rely on this fallback.
- Also applies Laplacian blur filter and aspect-ratio heuristics in `_quality_filter`.

### Face Recognition Missing API
- `POST /sims/face/match` endpoint **does not exist** on the main backend yet.
- `config.yaml` has `fallback_to_cache: true` — falls back to Redis cache scan when API is unavailable.
- Other usable but limited APIs: `/sims/student/get-list`, `/sims/class/studentMessage`.

### Stream Gateway Requires ffmpeg
- The Go decoder spawns an **ffmpeg subprocess** to decode RTSP → raw YUV420P frames.
- Alpine runtime Dockerfile installs `ffmpeg` via `apk add`.
- `ffmpeg` must be on `$PATH` in development too.
- Frame size is hardcoded: `width * height * 3 / 2` (YUV420P).
- `KAFKA_BROKERS` env var overrides Kafka brokers (default `localhost:9092`).

### Test Environment Webcam Workaround
- `test-env` server uses **ffmpeg subprocess** (not OpenCV `VideoCapture`) for webcam capture on macOS.
- Reason: macOS AVFoundation + uvicorn's `fork()`-based worker spawning causes crashes. ffmpeg runs as a separate process piping MJPEG frames.
- Required: `ffmpeg` on `$PATH`, `Pillow` for frame generation fallback.
- Config is fully adjustable via `/api/config` endpoints.

---

## Team Division

| Role | Owns | Languages |
|---|---|---|
| **You (perception)** | stream-gateway + face-recognition | Go, Python |
| **Partner (business)** | dormitory-service + main-process integration + camera management | Java |

Kafka topic `t_dorm_event` is the only coupling point. Both sides develop independently.

---

## Notable structure details

### Code maturity
- No CI/CD (`.github/workflows` doesn't exist). `doc/development-guide.md` describes a pipeline but nothing is configured — it's aspirational. Mentions Postgres/Milvus/web frontend that don't exist in this repo.
- No Go tests (`*_test.go` = 0 files).
- Face-recognition has Python tests under `tests/` (`test_behavior.py`, `test_detector.py`, `test_event_publisher.py`, `test_integration.py`, `test_tracker.py`). Tests use Haar Cascade fallback (no ONNX model needed).
- `face-recognition/app/` has backup files (`detector.py.backup`, `detector.py.bak`) — should be cleaned up.
- Dormitory-service has test sources under `src/test/java/com/sims/dormitory/` (structure exists, actual tests TBD).

### Python packages
- Managed via `pip`/`uv` — both test-env and face-recognition prefer `uv` if installed (`command -v uv` in `start.sh`).
- `requirements.txt` uses `opencv-python-headless` (not full `opencv-python`).

### Go module
- `github.com/sims/campusvision/stream-gateway`, `go 1.26.2`, deps: `segmentio/kafka-go`, `gopkg.in/yaml.v3`.
- Uses `kafka.Hash{}` partition balancer — frames from same building land in same partition.

### Dormitory Service specifics
- Spring Boot 3.2.5, MyBatis Plus 3.5.7, Lombok 1.18.38 (bumped for JDK 26).
- Uses JJWT 0.9.1 for shared auth with main backend (requires `jaxb-api` on JDK 17+).
- `@MapperScan("com.sims.dormitory.repository")` in the application class.
- `db/migration/` directory exists but is **empty** — no Flyway migrations yet.
- Nightly report cron is hardcoded at `0 0 23 * * ?` (`@Scheduled`) — config-driven override via `dorm_config` table not wired.
- `DormBuilding` entity references `dorm_building` table (doesn't exist in init.sql) and `DormRoom` references `dorm_room` (also doesn't exist).
- `cameraService.updateLastEventTime()` is the camera management coupling point.

### MinIO
- Provisioned in infra (ports 9000 API, 9001 console) but no service writes to it — `snapshot_path` is always `""`.

### Dynamic frame extraction
- `stream-gateway/config.yaml`: `fps_day: 5`, `fps_night: 1`, `dynamic_extraction: true`, `motion_threshold: 0.05`.
- Behavior analysis pipeline (`behavior.py`, `tracker.py`) is gated by `behavior.enabled: false` — off by default.

### Duplicate instruction files
- `.agents/skills/repo.md` and `.openhands/microagents/repo.md` exist but are stale clones of an earlier version of this file. They are NOT authoritative — delete or ignore them.
- `config.toml` is for old OpenHands tooling (not OpenCode).

### References
- `doc/` contains extensive PRDs (5,455 lines) and design docs (3,754 lines). Read `doc/prd/README.md` for navigation.
- RTSP URLs are placeholders in both `stream-gateway/config.yaml` and `infra/mariadb/init.sql`.
