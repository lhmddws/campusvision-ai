# CampusVision AI — AGENTS.md

Multi-language monorepo (Go + Python + Java) for dormitory AI surveillance.

## Architecture

```
RTSP cameras (A/B/C/D) → stream-gateway (Go) → t_dorm_frame (Kafka)
  → face-recognition (Python) → t_dorm_event (Kafka)
  → dormitory-service (Java/Go) → MariaDB + Redis
```

| Module | Language | Entrypoint | Port |
|---|---|---|---|
| `stream-gateway/` | Go (1.26) | `go run cmd/main.go --config config.yaml` | 8080 (health), 8081 (mgmt) |
| `face-recognition/` | Python (3.11) | `python -m app.main --config config.yaml` | — |
| `dormitory-service/` | Java 17 (Spring Boot 3.2.5) | `DormitoryServiceApplication.java` | 8081 |
| `dormitory-service-go/` | Go (1.26) | `go run ./cmd/dormitory-service/` | 8083 |
| `test-env/` | Go (1.26) Gin + Vue 3 | `go run ./cmd/test-env/` | 8082 |

Infra (`docker compose up -d`): Zookeeper (2181), Kafka (9092), Redis (6379), MariaDB (3306), MinIO (9000/9001).

## Commands

```bash
# Infrastructure (minimal for dev)
docker compose up -d kafka redis

# Stream Gateway — requires ffmpeg on $PATH
cd stream-gateway && go run cmd/main.go --config config.yaml

# Face Recognition — requires ONNX models first
cd face-recognition && python -m app.download_models
cd face-recognition && python -m app.main --config config.yaml

# Dormitory Service (Java) — requires system mvn (no wrapper)
cd dormitory-service && mvn compile && mvn spring-boot:run

# Dormitory Service (Go) — runs alongside Java on :8083
cd dormitory-service-go && go run ./cmd/dormitory-service/ --config config.yaml

# Test Environment (Go + Vue frontend)
cd test-env/frontend && npm ci && npm run build  # first time / after frontend changes
cd test-env && go run ./cmd/test-env/             # serves :8082

# Kafka management
docker compose exec kafka kafka-topics --bootstrap-server localhost:9092 --list
docker compose exec kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic t_dorm_event --from-beginning
```

## test-env (Demo Environment)

Go/Gin backend + Vue 3 Vite frontend. Replaces the old Python/FastAPI version.

**Run:**
```bash
# Option A: Docker (recommended)
docker compose up -d --build test-env

# Option B: Native dev mode
cd test-env && go run ./cmd/test-env/
# Frontend dev server with HMR (separate terminal):
cd test-env/frontend && npm run dev
```

**Web UI**: `http://localhost:8082/` — 4 simulated cameras, event log, stats, face enrollment, behavior panel.
**Keyboard shortcuts**: Ctrl+R (random events), Ctrl+T (rush hour), F (toggle stats panel).

## Docker Compose Services

| Service | Depends on | Notes |
|---|---|---|
| `zookeeper` | — | `user: root` (permission fix for macOS) |
| `kafka` | zookeeper | `user: root` (permission fix); topics auto-created by `kafka-init` |
| `kafka-init` | kafka (healthy) | One-shot: creates `t_dorm_frame`(4p), `t_dorm_event`(2p), `t_dorm_alert`(1p) |
| `redis` | — | Healthy check via redis-cli ping |
| `mariadb` | — | Initialized with `infra/mariadb/init.sql` (11 tables) |
| `minio` | — | Unused by any service — `snapshot_path` always `""` |
| `stream-gateway` | kafka, kafka-init, mariadb | Docker override via `config.docker.yaml` |
| `face-recognition` | kafka, redis, stream-gateway | Docker CMD lacks `--config` — relies on bind-mounted `config.docker.yaml` |
| `dormitory-service` | kafka, mariadb, redis | Java, port 8081 |
| `dormitory-service-go` | kafka, mariadb, redis | Go, port 8083 — runs alongside Java |
| `test-env` | kafka, redis | Go+Vue, port 8082 |

## Kafka Topics

| Topic | Partitions | Retention | Producer → Consumer |
|---|---|---|---|
| `t_dorm_frame` | 4 | 12h | stream-gateway → face-recognition |
| `t_dorm_event` | 2 | 7d | face-recognition → dormitory-service (both Java & Go) |
| `t_dorm_alert` | 1 | 7d | dormitory-service → (future) |

- `t_dorm_frame` uses **hash partitioner** (`kafka.Hash{}`) keyed by `building`.
- Compression: Snappy for `t_dorm_frame`.
- Python face-recognition publishes **raw JSON** (no Spring Kafka type headers).
- Java consumer uses `StringDeserializer` + manual `ObjectMapper.readValue` with `@JsonProperty("camera_id")`.
- Java consumer: manual ack, `max.poll.records=500`, concurrency=3.
- Go consumer: parallel implementation, same topics, same Redis dedup key format.

## Critical Gotchas

### DB Schema Fragmentation
Three sources of truth that **disagree**:

1. **`infra/mariadb/init.sql`** (docker-compose executes this) — `dorm_student_assignment`, `dorm_entry_exit_event`, `dorm_alert_record` etc.
2. **Java entities** — use different names: `dorm_event_log` (vs `dorm_entry_exit_event`), `dorm_student` (vs `dorm_student_assignment`), `dorm_alert` (vs `dorm_alert_record`). Reference `dorm_building`/`dorm_room` tables that **don't exist in init.sql**.
3. **Go entities** — hybrid: follows Java convention for some tables, init.sql for others. Comments document each divergence.

**Always verify table names before writing MyBatis Plus or sqlx queries.** No XML mappers exist — all queries rely on `BaseMapper<T>` (Java) or generic Go repository with `db` struct tags.

### Java JDK / Lombok Compatibility
- `pom.xml` targets JDK 17, dev machines run JDK 26.
- Lombok 1.18.30 **crashes** on JDK 26 — fix: `<lombok.version>1.18.38</lombok.version>` (already applied).
- Surefire: `-Dnet.bytebuddy.experimental=true` for Mockito (already in pom.xml).
- `CameraServiceImplTest.java` is **excluded** from compilation (`<testExclude>` in pom.xml).
- No `.mvn/wrapper` — Maven must be installed system-wide.

### Production Credentials in application.yml
- `dormitory-service/src/main/resources/application.yml` contains hardcoded production credentials.
- Overridable via `SPRING_DATASOURCE_*` env vars.
- **Do not commit real credentials.**

### Redis Config
- Both face-recognition and dormitory-service connect to `127.0.0.1:6379`, same `db=0`.
- Redis-based dedup: Java uses `DEDUP_TTL_SECONDS=3600` (key: `camera_id + frame_sequence`); Go uses `DefaultDedupTTL=3600` (key: `dedup:{camera_id}:{frame_sequence}`).

### ONNX Model Management
- Models defined in `face-recognition/app/models/model_urls.yaml` — two models with verified SHA256 hashes (no `PLACEHOLDER_UPDATE_ME` sentinel).
- Downloaded at Docker build time or via `python -m app.download_models`.
- Files gitignored (`*.onnx`).

### Face Detector Haar Cascade Fallback
- When ONNX model unavailable (`model_path: ""`), falls back to Haar Cascade.
- **Hardcoded macOS path** at `detector.py:255-256`: `/opt/homebrew/Cellar/opencv/4.13.0_8/share/opencv4/haarcascades/haarcascade_frontalface_default.xml`
- Will **fail on non-macOS or different OpenCV versions**. Tests rely on this fallback.

### Face Recognition Missing API
- `POST /sims/face/match` endpoint may not exist on main backend.
- `config.yaml` has `fallback_to_cache: true` — falls back to Redis cache scan.
- Docker CMD (`python -m app.main`) omits `--config` flag — relies on bind-mount or default `config.yaml`.

### Stream Gateway Requires ffmpeg
- Decoder spawns `ffmpeg` subprocess to decode RTSP → raw YUV420P.
- Frame size hardcoded: `width * height * 3 / 2` bytes.
- `KAFKA_BROKERS` env var overrides Kafka brokers.
- Camera passwords encrypted via `CAMERA_ENCRYPTION_KEY` env var (AES-256-GCM).
- DB polling syncs cameras from `dorm_camera` table every 30s (gated by `database.dsn`).

### No CI / Linters / Formatters
- `.github/workflows/` doesn't exist. `doc/development-guide.md` describes an aspirational pipeline — nothing configured.
- Zero linter/formatter configs across all languages (no `.golangci.yml`, `.eslintrc`, `.flake8`, `ruff.toml`, `.editorconfig`).
- No pre-commit, no Makefile. All workflows are ad-hoc.

### DB Migrations
- `infra/mariadb/migrations/` contains manual SQL files (no Flyway or automated migration tool).
- Apply migrations manually; track in `migrations/README.md`.

### Code Maturity
- **stream-gateway**: 4 test files (health handler, mgmt handler, camera config, crypto).
- **face-recognition**: 6 tests under `tests/` — use Haar Cascade fallback (no ONNX needed).
- **dormitory-service (Java)**: test source structure exists, 10+ test classes, `CameraServiceImplTest` is excluded.
- **dormitory-service-go**: **zero tests** — all implementation, no `*_test.go`.
- **test-env**: no tests.

## Team Division

| Role | Owns | Languages |
|---|---|---|
| **You (perception)** | stream-gateway + face-recognition | Go, Python |
| **Partner (business)** | dormitory-service (Java+Go) + main-process integration + camera management | Java, Go |

Kafka topic `t_dorm_event` is the only coupling point. Both sides develop independently. The Go dormitory service runs alongside Java on port 8083 — core event pipeline is functional, but reporting/aggregation features are stubs.

## Python Packages
- face-recognition managed via `pip`/`uv`; prefers `uv` if installed.
- `requirements.txt` uses `opencv-python-headless` (not full `opencv-python`).
- face-recognition has a `.venv/` (Python 3.14) that is not active by default.

## Go Modules
- `stream-gateway`: `github.com/sims/campusvision/stream-gateway`, deps `kafka-go`, `go-sql-driver/mysql`, `yaml.v3`
- `test-env`: `campusvision/test-env`, deps `gin`, `kafka-go`, `golang.org/x/image`
- `dormitory-service-go`: `github.com/sims/campusvision/dormitory-service-go`, deps `gin`, `sqlx`, `kafka-go`, `go-redis`, `viper`, `zap`, `jwt`, `cron`

## References
- `doc/` contains PRDs (5,455 lines) and design docs (3,754 lines). `doc/prd/README.md` for navigation.
- RTSP URLs are placeholders in both `stream-gateway/config.yaml` and `infra/mariadb/init.sql`.
- Dynamic frame extraction: `fps_day: 5`, `fps_night: 1`, `motion_threshold: 0.05` in stream-gateway config.
- Behavior analysis pipeline is gated by `behavior.enabled: false` — off by default.
