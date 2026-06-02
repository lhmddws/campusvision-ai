# dormitory-service-go

Dormitory management business service — consumes Kafka face events, handles attendance records, alerts, camera management, and face matching, exposing an HTTP API for the frontend.

## Overview

dormitory-service-go is the business layer of CampusVision AI, bridging the perception pipeline and the frontend management UI:

```
t_dorm_event (Kafka) → EventConsumer → Attendance records + student status updates
t_dorm_alert (Kafka) → AlertConsumer → (skeleton, not yet implemented)

Frontend SPA → Gin HTTP API (:8083) → MariaDB + Redis
```

| Property | Value |
|---|---|
| Language | Go 1.26 |
| Port | 8083 |
| Entrypoint | `cmd/dormitory-service/main.go` |
| Framework | Gin + sqlx + go-redis + Viper |
| Consumes | `t_dorm_event`, `t_dorm_alert` |

## Directory Structure

```
dormitory-service-go/
├── cmd/dormitory-service/
│   └── main.go              # Entrypoint: DI, route registration, Kafka consumer startup
├── internal/
│   ├── client/              # PushClient → stream-gateway notifications
│   ├── config/              # Viper config (YAML + env vars + Spring Boot compat)
│   ├── consumer/            # Kafka consumers (EventConsumer, AlertConsumer)
│   ├── handler/             # Gin HTTP handlers (camera, record, alert, config, face)
│   ├── middleware/           # JWT auth + CORS
│   ├── model/
│   │   ├── dto/             # Request/response types
│   │   ├── entity/          # DB entities (12 structs, db tag mapping)
│   │   └── enums/           # Domain enums (EventType, AlertType, ...)
│   ├── redis/               # go-redis wrapper + event dedup
│   ├── repository/          # sqlx repositories + generic BaseRepository[T]
│   ├── scheduler/           # Cron jobs (robfig/cron: nightly reports + health checks)
│   ├── service/             # Business logic (camera, record, alert, config, report)
│   └── util/                # AES-256-GCM password encryption
├── config.yaml              # Local dev config
├── config.docker.yaml       # Docker Compose config override
├── Dockerfile
└── go.mod
```

## Quick Start

### Prerequisites

- Go 1.26+
- MariaDB running (database `dormitory`, tables initialized by `infra/mariadb/init.sql`)
- Redis running
- Kafka running

### Local Development

```bash
cd dormitory-service-go
CONFIG_PATH=config.yaml go run ./cmd/dormitory-service/
```

### Docker

```bash
docker compose up -d dormitory-service-go
```

## Configuration

### config.yaml

```yaml
server:
  port: 8083

database:
  dsn: "root:root_dev@tcp(localhost:3306)/dormitory?charset=utf8mb4&parseTime=True&loc=Asia%2FShanghai"
  driver: mysql
  max_open_conns: 25
  max_idle_conns: 10

redis:
  host: "127.0.0.1"
  port: 6379
  db: 0
  password: ""

kafka:
  brokers: ["localhost:9092"]
  event_topic: "t_dorm_event"
  alert_topic: "t_dorm_alert"
  group_id: "dormitory-service-group"

jwt:
  secret: "${JWT_SECRET:your-256-bit-secret}"
  expiration_hours: 24

log:
  level: "info"
```

### Environment Variables

| Variable | Description |
|---|---|
| `CONFIG_PATH` | Config file path (default `config.yaml`) |
| `JWT_SECRET` | JWT signing key — must be set in production, must match main backend |
| `CAMERA_ENCRYPTION_KEY` | 32-byte AES-256-GCM key — must match stream-gateway |

> Config loading priority: defaults < config.yaml < environment variables. Supports Spring Boot-style env vars (`SPRING_DATASOURCE_URL`, `KAFKA_BOOTSTRAP_SERVERS`, etc.).

## API Overview

All endpoints return a unified `{code, message, data}` envelope.

### Camera Management

| Method | Path | Description |
|---|---|---|
| GET | `/api/cameras` | List cameras |
| POST | `/api/cameras` | Register camera |
| PUT | `/api/cameras/:id` | Update camera |
| DELETE | `/api/cameras/:id` | Delete camera |
| GET | `/api/cameras/:id/status` | Camera status |

### Attendance & Events

| Method | Path | Description |
|---|---|---|
| GET | `/api/records` | Entry/exit records |
| GET | `/api/events` | Event list |
| GET | `/api/attendance` | Attendance statistics |

### Alerts

| Method | Path | Description |
|---|---|---|
| GET | `/api/alerts` | Alert list |
| PUT | `/api/alerts/:id/handle` | Handle alert |

### Face

| Method | Path | Description |
|---|---|---|
| POST | `/api/face/match` | Face feature matching (cosine similarity) |
| POST | `/api/face/embed` | Face feature extraction (skeleton, returns null) |

### System Config

| Method | Path | Description |
|---|---|---|
| GET | `/api/configs` | System config list |
| PUT | `/api/configs/:key` | Update config item |

Full API spec: [`doc/api/dormitory-service-api.json`](../doc/api/dormitory-service-api.json).

## Core Mechanisms

### Event Consumption

EventConsumer reads from `t_dorm_event` and calls the repository layer directly (bypasses service):

1. Parse JSON event message
2. Redis dedup check (key: `dedup:{camera_id}:{frame_sequence}`, TTL 3600s)
3. Update student on-campus status (`dorm_student_status`)
4. Write entry/exit event record (`dorm_entry_exit_event`)
5. Stranger detection → create alert (`dorm_alert`)

### Scheduled Tasks

| Task | Schedule | Description |
|---|---|---|
| Nightly attendance report | Daily 23:00 | Generate daily attendance summary (skeleton) |
| Camera health check | Every 5 min | Poll stream-gateway health endpoint |

### Generic Repository

`BaseRepository[T]` provides generic CRUD operations using Go generics + reflection. Entities must use `db:"column"` tags for database column mapping.

## Testing

```bash
cd dormitory-service-go && go test ./...
```

Currently only `repository/base_test.go` — covers generic CRUD operations (go-sqlmock + testify).

## Caveats

- **CONFIG_PATH**: Uses environment variable for config loading, not CLI flag (differs from stream-gateway)
- **JWT dev key**: Default `your-256-bit-secret` — must set `JWT_SECRET` in production
- **AES key sync**: `CAMERA_ENCRYPTION_KEY` must match stream-gateway
- **AlertConsumer skeleton**: Currently only logs and commits offset — no actual alert handling
- **FaceMatch performance**: O(n) full table scan + cosine similarity — needs optimization at scale
- **Camera limit**: Hardcoded at 50, checked via `FindAll()` count (non-atomic operation)
