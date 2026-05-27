# CampusVision AI — AGENTS.md

Multi-language monorepo (Go + Python) for dormitory AI surveillance.

## Architecture

```
RTSP cameras (A/B/C/D) → stream-gateway (Go) → t_dorm_frame (Kafka)
  → face-recognition (Python) → t_dorm_event (Kafka)
  → dormitory-service-go (Go) → MariaDB + Redis
```

| Module | Language | Entrypoint | Port |
|---|---|---|---|
| `stream-gateway/` | Go (1.26) | `go run cmd/main.go --config config.yaml` | 8080 (health), 8081 (mgmt) |
| `face-recognition/` | Python (3.11) | `python -m app.main --config config.yaml` | — |
| `dormitory-service-go/` | Go (1.26) | `CONFIG_PATH=config.yaml go run ./cmd/dormitory-service/` | 8083 |
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

# Dormitory Service (Go) — on :8083
cd dormitory-service-go && CONFIG_PATH=config.yaml go run ./cmd/dormitory-service/

# Test Environment (Go + Vue frontend)
cd test-env/frontend && npm ci && npm run build  # first time / after frontend changes
cd test-env && go run ./cmd/test-env/             # serves :8082

# Kafka management
docker compose exec kafka kafka-topics --bootstrap-server localhost:9092 --list
docker compose exec kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic t_dorm_event --from-beginning
```

## test-env (Demo Environment)

Go/Gin backend + Vue 3 Vite frontend.

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
| `dormitory-service-go` | kafka, mariadb, redis | Go, port 8083 |
| `test-env` | kafka, redis | Go+Vue, port 8082 |

## Kafka Topics

| Topic | Partitions | Retention | Producer → Consumer |
|---|---|---|---|
| `t_dorm_frame` | 4 | 12h | stream-gateway → face-recognition |
| `t_dorm_event` | 2 | 7d | face-recognition → dormitory-service-go |
| `t_dorm_alert` | 1 | 7d | dormitory-service-go (alert consumer, skeleton) |

- `t_dorm_frame` uses **hash partitioner** (`kafka.Hash{}`) keyed by `building`.
- Compression: Snappy for `t_dorm_frame`.
- Python face-recognition publishes **raw JSON** (no Spring Kafka type headers).
- Go consumer: parallel implementation, same Redis dedup key format.

## Critical Gotchas

### DB Schema Fragmentation
Two main sources of truth that can **disagree**:

1. **`infra/mariadb/init.sql`** (docker-compose executes this) — `dorm_student_assignment`, `dorm_entry_exit_event`, `dorm_alert_record` etc.
2. **Go entities** — hybrid: follows some conventions from `init.sql`, others from the previous Java implementation. Comments document each divergence.

**Always verify table names before writing sqlx queries.**

### Redis Config
- Both face-recognition and dormitory-service-go connect to `127.0.0.1:6379`, same `db=0`.
- Redis-based dedup: Go uses `DefaultDedupTTL=3600` (key: `dedup:{camera_id}:{frame_sequence}`).

### ONNX Model Management
- Models defined in `face-recognition/app/models/model_urls.yaml` — two models with verified SHA256 hashes (no `PLACEHOLDER_UPDATE_ME` sentinel).
- Downloaded at Docker build time or via `python -m app.download_models`.
- Files gitignored (`*.onnx`).

### Face Detector Haar Cascade Fallback
- When ONNX model unavailable (`model_path: ""`), falls back to Haar Cascade.
- **Hardcoded macOS path** at `detector.py:255-256`: `/opt/homebrew/Cellar/opencv/4.13.0_8/share/opencv4/haarcascades/haarcascade_frontalface_default.xml`
- Will **fail on non-macOS or different OpenCV versions**. Tests rely on this fallback.

### Face Recognition External API
- Calls `POST /api/face/match` on dormitory-service-go (port 8083) for identity matching.
- `config.yaml` has `fallback_to_cache: true` — falls back to Redis cache scan on API failure.
- `POST /api/face/embed` endpoint exists but is a **stub** (returns null embedding).
- Docker CMD (`python -m app.main`) omits `--config` flag — relies on bind-mount or default `config.yaml`.

### Stream Gateway Requires ffmpeg
- Decoder spawns `ffmpeg` subprocess to decode RTSP → raw YUV420P.
- Frame size hardcoded: `width * height * 3 / 2` bytes.
- `KAFKA_BROKERS` env var mentioned in comments but **never actually read** — dead code.
- Camera passwords encrypted via `CAMERA_ENCRYPTION_KEY` env var (AES-256-GCM).
- DB polling syncs cameras from `dorm_camera` table every 30s (gated by `database.dsn`).

### CI / Linters / Formatters
- **CI exists** at `.github/workflows/ci.yml` — Go build+test+vet matrix for 3 modules, Python pytest. Runs on `main` branch only.
- **Linter configs exist but are NOT run in CI**: `.golangci.yml` (6 linters), `ruff.toml` (Python 3.11, line-length 120), `.editorconfig` (tabs for Go, 4-space Python/YAML, 2-space JSON/MD).
- **Missing**: Makefile, go.work, pyproject.toml, ESLint, pre-commit hooks, .python-version.
- No version injection via `-ldflags` in CI.

### DB Migrations
- `infra/mariadb/migrations/` contains manual SQL files (no Flyway or automated migration tool).
- Apply migrations manually; track in `migrations/README.md`.

### Code Maturity
- **stream-gateway**: 4 test files (health handler, mgmt handler, camera config, crypto).
- **face-recognition**: 6 tests under `tests/` — use Haar Cascade fallback (no ONNX needed).
- **dormitory-service-go**: 1 test file (`repository/base_test.go` — generic CRUD with go-sqlmock). All other packages untested.
- **test-env**: no tests.

### Cross-Cutting Patterns
- **Dual config**: Every module ships `config.yaml` (local dev) + `config.docker.yaml` (Docker override). Docker Compose bind-mounts the docker variant.
- **Config loading inconsistency**: stream-gateway uses CLI `--config` flag, dormitory-service-go uses `CONFIG_PATH` env var, face-recognition uses argparse `--config`, test-env uses pure env vars with no config file.
- **Go module path inconsistency**: `test-env` uses bare `campusvision/test-env` vs `github.com/sims/campusvision/` prefix for the other two Go modules.
- **Entrypoint nesting**: stream-gateway uses flat `cmd/main.go`, others use `cmd/<name>/main.go`.
- **AES key mismatch**: Dev AES keys in `stream-gateway/internal/crypto/` differ from `dormitory-service-go/internal/util/crypto.go` — cross-module encrypt/decrypt will fail in dev.
- **DB migrations**: `infra/mariadb/migrations/` uses manual serial numbering (`001_*.sql`, `002_*.sql`). No Flyway, no golang-migrate. Apply manually.
- **Kafka topic naming**: `t_dorm_<entity>` convention. `t_dorm_frame` (4p, hash by building, Snappy), `t_dorm_event` (2p), `t_dorm_alert` (1p).
- **DB table naming**: `dorm_` prefix, InnoDB/utf8mb4, BIGINT AUTO_INCREMENT, Chinese column comments.

## Team Division

| Role | Owns | Languages |
|---|---|---|
| **You (perception)** | stream-gateway + face-recognition | Go, Python |
| **Partner (business)** | dormitory-service-go + main-process integration + camera management | Go |

Kafka topic `t_dorm_event` is the only coupling point. Both sides develop independently.

## Python Packages
- face-recognition managed via `pip`/`uv`; prefers `uv` if installed.
- `requirements.txt` uses `opencv-python-headless` (not full `opencv-python`).
- face-recognition has a `.venv/` (Python 3.14) that is not active by default.

## Go Modules
- `stream-gateway`: `github.com/sims/campusvision/stream-gateway`, deps `kafka-go`, `go-sql-driver/mysql`, `yaml.v3`
- `test-env`: `campusvision/test-env`, deps `gin`, `kafka-go`, `golang.org/x/image`
- `dormitory-service-go`: `github.com/sims/campusvision/dormitory-service-go`, deps `gin`, `sqlx`, `kafka-go`, `go-redis`, `viper`, `zap`, `jwt`, `cron`

## API Documentation (OpenAPI 3.0.3)

The project's API surface is documented as three OpenAPI 3.0.3 spec files under `doc/api/`:

| File | Module | Coverage |
|---|---|---|
| `doc/api/stream-gateway-api.json` | stream-gateway (Go) | Health API (port 8080), Management API (port 8081, X-Management-Key auth) — 5 endpoints |
| `doc/api/face-recognition-kafka.json` | face-recognition (Python) | Kafka message schemas (FrameMessage, EntryExitEvent, BehaviorEvent), all 12 config dataclasses, 11-step processing pipeline |
| `doc/api/dormitory-service-api.json` | dormitory-service-go (Go) | 22+ HTTP endpoints: cameras CRUD/status, attendance records/events, alerts, configs, face match/embed — all with standard `{code,message,data}` envelope |

**Validation:**
```bash
# Validate all OpenAPI 3 specs
pip install openapi-spec-validator 2>/dev/null
python3 -m json.tool doc/api/stream-gateway-api.json > /dev/null && echo "✅ stream-gateway"
python3 -m json.tool doc/api/face-recognition-kafka.json > /dev/null && echo "✅ face-recognition-kafka"
python3 -m json.tool doc/api/dormitory-service-api.json > /dev/null && echo "✅ dormitory-service"
```

## References
- `doc/` contains PRDs (5,455 lines) and design docs (3,754 lines). `doc/prd/README.md` for navigation.
- `doc/api/` contains OpenAPI 3.0.3 specs for stream-gateway, face-recognition (Kafka), and dormitory-service-go.
- RTSP URLs are placeholders in both `stream-gateway/config.yaml` and `infra/mariadb/init.sql`.
- Dynamic frame extraction: `fps_day: 5`, `fps_night: 1`, `motion_threshold: 0.05` in stream-gateway config.
- Behavior analysis pipeline is gated by `behavior.enabled: false` — off by default.
