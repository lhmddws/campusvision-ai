# CampusVision AI

AI-powered dormitory surveillance system with multi-camera RTSP streaming, real-time face recognition, and automated attendance & alerting.

```
RTSP cameras (A/B/C/D) → stream-gateway (Go) → t_dorm_frame (Kafka)
  → face-recognition (Python) → t_dorm_event (Kafka)
  → dormitory-service (Java) → MariaDB + Redis
```

## Architecture

**3-tier perception pipeline:**

| Layer | Language | Service | Role |
|---|---|---|---|
| Stream ingest | Go | `stream-gateway` | RTSP capture → Kafka frame producer |
| Recognition | Python | `face-recognition` | Face detection/recognition → event producer |
| Business | Java | `dormitory-service` | Event processing, alerts, reports, API |
| Test env | Python | `test-env` | Simulated 4-camera setup for dev/testing |

**Infrastructure** (Docker Compose): Kafka, Redis, MariaDB, MinIO

## Modules

| Directory | Language | Entrypoint | Port |
|---|---|---|---|
| `stream-gateway/` | Go | `cmd/main.go --config config.yaml` | 8080 |
| `face-recognition/` | Python | `python -m app.main --config config.yaml` | — |
| `dormitory-service/` | Java (Spring Boot) | `DormitoryServiceApplication.java` | 8081 |
| `test-env/` | Python (FastAPI) | `bash start.sh` | 8082 |
| `infra/` | — | `docker-compose.yml` | — |

## Quick Start

```bash
# 1. Start infrastructure
docker compose up -d

# 2. Stream gateway (requires ffmpeg)
cd stream-gateway && go run cmd/main.go --config config.yaml

# 3. Face recognition (requires ONNX models)
cd face-recognition && python -m app.download_models
cd face-recognition && python -m app.main --config config.yaml

# 4. Dormitory service (requires JDK 17+, Maven)
cd dormitory-service && mvn compile
mvn spring-boot:run

# 5. Test environment (simulates 4 cameras)
bash test-env/start.sh
```

## Kafka Topics

| Topic | Partitions | Retention | Producer → Consumer |
|---|---|---|---|
| `t_dorm_frame` | 4 | 12h | stream-gateway → face-recognition |
| `t_dorm_event` | 2 | 7d | face-recognition → dormitory-service |
| `t_dorm_alert` | 1 | 7d | dormitory-service → (future) |

## Key Features

- **Multi-camera RTSP ingest** with configurable FPS (day/night) and motion-based dynamic extraction
- **Face detection** (RetinaFace ONNX) with Haar Cascade fallback
- **Face recognition** (ArcFace ONNX) with Redis-based identity cache
- **Behavior analysis** pipeline (tracking, direction detection, quality filtering)
- **Automated nightly reports** and attendance tracking
- **Stranger detection** and real-time alerting
- **Event deduplication** via Redis (3600s TTL)

## Development

See [AGENTS.md](AGENTS.md) for detailed architecture docs, gotchas, and team division.
