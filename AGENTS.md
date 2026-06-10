# CampusVision AI — AGENTS.md

Multi-language monorepo (Go + Python) for dormitory AI surveillance.

## Architecture

```
RTSP cameras (A/B/C/D) → stream-gateway (Go) → t_dorm_frame (Kafka)
  → face-recognition (Python) → t_dorm_event (Kafka)
  → dormitory-service-go (Go) → MariaDB + Redis
```

| Module                  | Language      | Entrypoint                                                | Port                       |
| ----------------------- | ------------- | --------------------------------------------------------- | -------------------------- |
| `stream-gateway/`       | Go (1.26)     | `go run cmd/main.go --config config.yaml`                 | 8080 (health), 8081 (mgmt) |
| `face-recognition/`     | Python (3.11) | `python -m app.main --config config.yaml`                 | —                          |
| `dormitory-service-go/` | Go (1.26)     | `CONFIG_PATH=config.yaml go run ./cmd/dormitory-service/` | 8083                       |

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

# Frontend (Vue 3)
cd frontend && pnpm install && pnpm dev       # dev server on :3000
cd frontend && pnpm build:prod                 # production build (chunked)
cd frontend && pnpm test                       # 83 tests with Vitest

# Apply test data seed to running DB
docker compose exec mariadb mysql -uroot -proot_dev dormitory < infra/mariadb/init.sql

# Kafka management
docker compose exec kafka kafka-topics --bootstrap-server localhost:9092 --list
docker compose exec kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic t_dorm_event --from-beginning
```

## Docker Compose Services

| Service                | Depends on                   | Notes                                                                        |
| ---------------------- | ---------------------------- | ---------------------------------------------------------------------------- |
| `zookeeper`            | —                            | `user: root` (permission fix for macOS)                                      |
| `kafka`                | zookeeper                    | `user: root` (permission fix); topics auto-created by `kafka-init`           |
| `kafka-init`           | kafka (healthy)              | One-shot: creates `t_dorm_frame`(4p), `t_dorm_event`(2p), `t_dorm_alert`(1p, skeleton) |
| `redis`                | —                            | Healthy check via redis-cli ping                                             |
| `mariadb`              | —                            | Initialized with `infra/mariadb/init.sql` (13 tables + test data seed)       |
| `minio`                | —                            | Unused by any service — `snapshot_path` always `""`                          |
| `stream-gateway`       | kafka, kafka-init, mariadb   | Docker override via `config.docker.yaml`                                     |
| `face-recognition`     | kafka, redis, stream-gateway | Docker CMD lacks `--config` — relies on bind-mounted `config.docker.yaml`    |
| `dormitory-service-go` | kafka, mariadb, redis        | Go, port 8083                                                                |

## Kafka Topics

| Topic          | Partitions | Retention | Producer → Consumer                             |
| -------------- | ---------- | --------- | ----------------------------------------------- |
| `t_dorm_frame` | 4          | 12h       | stream-gateway → face-recognition               |
| `t_dorm_event` | 2          | 7d        | face-recognition → dormitory-service-go         |
| `t_dorm_alert` | 1          | 7d        | dormitory-service-go (alert consumer, skeleton) |

- `t_dorm_frame` uses **hash partitioner** (`kafka.Hash{}`) keyed by `building`.
- Compression: Snappy for `t_dorm_frame`.
- Python face-recognition publishes **raw JSON** (no Spring Kafka type headers).
- Go consumer: parallel implementation, same Redis dedup key format.

## Critical Gotchas

### DB Schema Fragmentation

Fixed in migration 001 and entity updates — `infra/mariadb/init.sql` and Go entities now match. Table names verified across both sources.

- `dorm_config` has `config_options TEXT` column (migration 003) — used by frontend to render `<el-select>` dropdowns. `init.sql` includes UPDATE statements to populate option lists.
- `dorm_building` table exists in `init.sql` but was missing from some DB instances — run `CREATE TABLE IF NOT EXISTS` if missing.
- `dorm_camera` table may be missing `type`/`protocol` columns on older DB instances — the `init.sql` INSERT uses a compatible column list (`camera_id, name, building, rtsp_url, direction, status, enabled`).

### Redis Config

- Both face-recognition and dormitory-service-go connect to `127.0.0.1:6379`, same `db=0`.
- Redis-based dedup: Go uses `DefaultDedupTTL=3600` (key: `dedup:{camera_id}:{frame_sequence}`).

### ONNX Model Management

- Models defined in `face-recognition/app/models/model_urls.yaml` — two models with verified SHA256 hashes (no `PLACEHOLDER_UPDATE_ME` sentinel).
- Downloaded at Docker build time or via `python -m app.download_models`.
- Files gitignored (`*.onnx`).

### Face Detector Haar Cascade Fallback

- When ONNX model unavailable (`model_path: ""`), falls back to Haar Cascade.
- Fixed in f6a24e0 — uses `cv2.data.haarcascades` (portable) instead of hardcoded macOS path.
- Tests rely on this fallback.

### Face Recognition External API

- Calls `POST /api/face/match` on dormitory-service-go (port 8083) for identity matching.
- `config.yaml` has `fallback_to_cache: true` — falls back to Redis cache scan on API failure.
- Docker CMD (`python -m app.main`) omits `--config` flag — relies on bind-mount or default `config.yaml`.

### Stream Gateway Requires ffmpeg

- Decoder spawns `ffmpeg` subprocess to decode RTSP → raw YUV420P.
- Frame size hardcoded: `width * height * 3 / 2` bytes.
- `KAFKA_BROKERS` env var mentioned in comments but **never actually read** — dead code.
- Camera passwords encrypted via `CAMERA_ENCRYPTION_KEY` env var (AES-256-GCM).
- DB polling syncs cameras from `dorm_camera` table every 30s (gated by `database.dsn`).

### CI / Linters / Formatters

- **CI**: Not yet set up — no `.github/workflows/ci.yml` file exists yet. Linter configs (`.golangci.yml`, `ruff.toml`, `.editorconfig`) are present in the project root but are not run in any pipeline.
- **Missing**: Makefile, go.work, pyproject.toml, ESLint, pre-commit hooks, .python-version.
- No version injection via `-ldflags` in CI.

### DB Migrations

- `infra/mariadb/migrations/` contains manual SQL files (no Flyway or automated migration tool).
- Apply migrations manually; track in `migrations/README.md`.

### Code Maturity

- **stream-gateway**: 4 test files (health handler, mgmt handler, camera config, crypto).
- **face-recognition**: 6 tests under `tests/` — use Haar Cascade fallback (no ONNX needed).
- **dormitory-service-go**: 1 test file (`repository/base_test.go` — generic CRUD with go-sqlmock). All other packages untested.
- **frontend**: 83 Vitest tests across 11 test files (dashboard, config, camera, attendance, alerts, events, face, inspection, layout, login, smoke).

### Cross-Cutting Patterns

- **Dual config**: Every module ships `config.yaml` (local dev) + `config.docker.yaml` (Docker override). Docker Compose bind-mounts the docker variant.
- **Entrypoint nesting**: stream-gateway uses flat `cmd/main.go`, dormitory-service-go uses `cmd/<name>/main.go`.
- **DB migrations**: `infra/mariadb/migrations/` uses manual serial numbering (`001_*.sql`, `002_*.sql`). No Flyway, no golang-migrate. Apply manually.
- **Kafka topic naming**: `t_dorm_<entity>` convention. `t_dorm_frame` (4p, hash by building, Snappy), `t_dorm_event` (2p), `t_dorm_alert` (1p, skeleton).
- **DB table naming**: `dorm_` prefix, InnoDB/utf8mb4, BIGINT AUTO_INCREMENT, Chinese column comments.
- **Frontend lazy loading**: Heavy libs split into on-demand chunks: `echarts` (340KB gzip), `wangeditor` (306KB), `xlsx+jspdf` (231KB). Routes use dynamic `import()` already.
- **Test data seed**: `init.sql` contains 22 students, 4 cameras, 24 entry events, 4 nightly reports, and 2 stranger records — all using `INSERT IGNORE` for idempotent re-runs.

## Team Division

| Role                   | Owns                                                                | Languages  |
| ---------------------- | ------------------------------------------------------------------- | ---------- |
| **You (perception)**   | stream-gateway + face-recognition                                   | Go, Python |
| **Partner (business)** | dormitory-service-go + main-process integration + camera management | Go         |

Kafka topic `t_dorm_event` is the only coupling point. Both sides develop independently.

## Python Packages

- face-recognition managed via `pip`/`uv`; prefers `uv` if installed.
- `requirements.txt` uses `opencv-python-headless` (not full `opencv-python`).
- face-recognition has a `.venv/` (Python 3.14) that is not active by default.

## Go Modules

- `stream-gateway`: `github.com/sims/campusvision/stream-gateway`, deps `kafka-go`, `go-sql-driver/mysql`, `yaml.v3`
- `dormitory-service-go`: `github.com/sims/campusvision/dormitory-service-go`, deps `gin`, `sqlx`, `kafka-go`, `go-redis`, `viper`, `zap`, `jwt`, `cron`

## API Documentation (OpenAPI 3.0.3)

The project's API surface is documented as three OpenAPI 3.0.3 spec files under `doc/api/`:

| File                                  | Module                    | Coverage                                                                                                                                                 |
| ------------------------------------- | ------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `doc/api/stream-gateway-api.json`     | stream-gateway (Go)       | Health API (port 8080), Management API (port 8081, X-Management-Key auth) — 5 endpoints                                                                  |
| `doc/api/face-recognition-kafka.json` | face-recognition (Python) | Kafka message schemas (FrameMessage, EntryExitEvent, BehaviorEvent), all 12 config dataclasses, 11-step processing pipeline                              |
| `doc/api/dormitory-service-api.json`  | dormitory-service-go (Go) | 22+ HTTP endpoints: cameras CRUD/status, attendance records/events, alerts, configs, face match/embed — all with standard `{code,message,data}` envelope |

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
