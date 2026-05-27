# CampusVision AI

AI-powered dormitory surveillance system with multi-camera RTSP streaming, real-time face recognition, automated attendance & alerting, and a Vue 3 SPA dashboard.

```
RTSP cameras (A/B/C/D)
  → stream-gateway (Go) ── t_dorm_frame (Kafka) ──→
  → face-recognition (Python) ── t_dorm_event (Kafka) ──→
  → dormitory-service-go (Go) ──→ MariaDB + Redis
                                    ↑
                            frontend (Vue 3 SPA)
```

## Architecture

**Perception pipeline:**

| Layer | Language | Service | Role |
|---|---|---|---|
| Stream ingest | Go | `stream-gateway` | RTSP capture → Kafka frame producer (hash-partitioned by building) |
| Recognition | Python | `face-recognition` | Face detection/recognition → event producer |
| Business API | Go | `dormitory-service-go` | Event processing, alerts, attendance, reports, HTTP API |
| Frontend | Vue 3 | `frontend/` | SPA dashboard built on RuoYi-Vue3-ts (Element Plus + Vite) |

**Infrastructure** (Docker Compose): Zookeeper, Kafka (3 topics), Redis, MariaDB, MinIO

## Modules

| Directory | Language | Entrypoint | Port |
|---|---|---|---|
| `stream-gateway/` | Go 1.26 | `go run cmd/main.go --config config.yaml` | 8080 (health), 8081 (mgmt) |
| `face-recognition/` | Python 3.11 | `python -m app.main --config config.yaml` | — |
| `dormitory-service-go/` | Go 1.26 | `CONFIG_PATH=config.yaml go run ./cmd/dormitory-service/` | 8083 |
| `frontend/` | Vue 3.2 + TS | `pnpm dev` | 80 (dev) |
| `infra/` | — | `docker compose up -d` | — |

## Quick Start

```bash
# 1. Start infrastructure (minimal for dev)
docker compose up -d kafka redis

# 2. Stream gateway (requires ffmpeg on $PATH)
cd stream-gateway && go run cmd/main.go --config config.yaml

# 3. Face recognition — download models first (see below)
cd face-recognition && python -m app.download_models
cd face-recognition && python -m app.main --config config.yaml

# 4. Dormitory service (Go) — API on :8083
cd dormitory-service-go && CONFIG_PATH=config.yaml go run ./cmd/dormitory-service/

# 5. Frontend SPA (requires pnpm)
cd frontend && pnpm dev
```

### Model Download (China / Restricted Networks)

The face-recognition module supports mirror/proxy fallback for restricted networks:

```bash
# Auto-download (tries hf-mirror.com → huggingface.co)
cd face-recognition && python -m app.download_models

# Custom mirror + proxy
python -m app.download_models --mirror https://hf-mirror.com --proxy http://127.0.0.1:7890

# List configured mirrors
python -m app.download_models --list-mirrors

# List models & exit
python -m app.download_models --retries 5
```

**Docker build with mirror support:**

```bash
docker compose build \
  --build-arg "HF_ENDPOINT=https://hf-mirror.com" \
  --build-arg "APT_MIRROR=https://mirrors.tuna.tsinghua.edu.cn/debian" \
  --build-arg "PIP_INDEX_URL=https://pypi.tuna.tsinghua.edu.cn/simple" \
  face-recognition
```

## Kafka Topics

| Topic | Partitions | Retention | Producer → Consumer |
|---|---|---|---|
| `t_dorm_frame` | 4 | 12h | stream-gateway → face-recognition |
| `t_dorm_event` | 2 | 7d | face-recognition → dormitory-service-go |
| `t_dorm_alert` | 1 | 7d | dormitory-service-go → (future) |

- `t_dorm_frame` uses **hash partitioner** (`kafka.Hash{}`) keyed by `building`.
- Compression: **Snappy** for `t_dorm_frame`.
- Python consumer publishes **raw JSON** (no Spring Kafka type headers).

## Docker Compose Services

| Service | Depends on | Notes |
|---|---|---|
| `zookeeper` | — | `user: root` (permission fix) |
| `kafka` | zookeeper | `user: root`; topics auto-created by `kafka-init` |
| `kafka-init` | kafka (healthy) | One-shot: creates `t_dorm_frame`(4p), `t_dorm_event`(2p), `t_dorm_alert`(1p) |
| `redis` | — | db=0, shared by face-recognition and dormitory-service-go |
| `mariadb` | — | Initialized with `infra/mariadb/init.sql` (11 tables) |
| `minio` | — | Snapshot storage (currently unused — snapshot_path is always `""`) |
| `stream-gateway` | kafka, kafka-init, mariadb | Docker override via `config.docker.yaml` |
| `face-recognition` | kafka, redis, stream-gateway | ONNX models: preload, build-time, or runtime download |
| `dormitory-service-go` | kafka, mariadb, redis | Go HTTP API on port 8083 |

## Key Features

- **Multi-camera RTSP ingest** with configurable FPS (day/night 5/1) and motion-based dynamic extraction
- **Face detection** (RetinaFace ONNX) with Haar Cascade fallback (OpenCV)
- **Face recognition** (ArcFace ONNX) with Redis-based identity cache
- **Behavior analysis** pipeline (tracking, direction detection, quality filtering) — *disabled by default*
- **Automated attendance** tracking with nightly reports
- **Stranger detection** and real-time alerting
- **Event deduplication** via Redis (3600s TTL)
- **Vue 3 SPA dashboard** — monitoring, camera management, events, alerts, attendance, face records, system config

## Development

See [AGENTS.md](AGENTS.md) for detailed architecture docs, cross-module gotchas, team division, and CI setup.

### Quick Reference

```bash
# Kafka management
docker compose exec kafka kafka-topics --bootstrap-server localhost:9092 --list
docker compose exec kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic t_dorm_event --from-beginning

# Run tests
cd dormitory-service-go && go test ./...
cd face-recognition && python -m pytest
cd frontend && npx vitest run
```

### Known Gotchas

- **DB schema fragmentation**: `infra/mariadb/init.sql` and Go entities can diverge — always verify table names before writing sqlx queries.
- **AES key mismatch**: Dev keys in `stream-gateway/internal/crypto/` differ from `dormitory-service-go/internal/util/crypto.go` — cross-module encrypt/decrypt will fail in dev.
- **Haar Cascade path**: Hardcoded macOS path at `face-recognition/app/detector/detector.py:255-256` — fails on non-macOS or different OpenCV versions.
- **Config loading**: Each module uses a different mechanism (CLI `--config`, `CONFIG_PATH` env var, argparse). See AGENTS.md for details.
- **ONNX models**: Model files are gitignored (`*.onnx`). Download via `python -m app.download_models` or Docker build.
- **The `POST /api/face/embed` endpoint is a stub** (returns null embedding).
