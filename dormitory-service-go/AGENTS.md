# dormitory-service-go — AGENTS.md

Business logic service: Kafka event consumption, camera management, attendance, alerts, face matching. Serves HTTP API on :8083.

## Architecture

```
cmd/dormitory-service/main.go (wiring + Gin router)
internal/
├── client/        PushClient → stream-gateway notifications
├── config/        Viper config (YAML + env vars + Spring Boot compat)
├── consumer/      Kafka consumers (EventConsumer, AlertConsumer, Manager)
├── handler/       Gin HTTP handlers (camera, record, alert, config, face)
├── middleware/     JWT auth + CORS
├── model/
│   ├── dto/       Request/response types (7 files)
│   ├── entity/    DB entity structs with `db` tags (12 files)
│   └── enums/     Typed string enums (EventType, AlertType, etc.)
├── redis/         go-redis wrapper + dedup logic
├── repository/    sqlx repositories + generic BaseRepository[T]
├── scheduler/     Cron jobs (nightly report, health check)
├── service/       Business logic (camera, record, alert, config, report)
└── util/          AES-256-GCM password encryption
```

**Dependency flow**: handler → service → repository → sqlx/DB. Consumer layer bypasses services and calls repositories directly.

## Entry Point

`cmd/dormitory-service/main.go` — single `main()` that wires everything:
1. Load config via `CONFIG_PATH` env var (default `config.yaml`)
2. Connect MariaDB (sqlx) + Redis (go-redis)
3. Initialize 8 repositories → 5 services → 5 handlers
4. Start EventConsumer (t_dorm_event) + AlertConsumer (t_dorm_alert)
5. Start cron scheduler (nightly report @23:00, health check every 5min)
6. Serve Gin HTTP on `:8083` with JWT-protected routes

## Key Packages

| Package | Purpose | Key Types |
|---|---|---|
| `consumer` | Kafka event processing pipeline | `EventConsumer`, `AlertConsumer`, `Manager`, `Consumer` interface |
| `repository` | Generic CRUD + domain queries | `BaseRepository[T]` (generics), 8 domain repos |
| `service` | Business logic layer | `CameraService`, `RecordService`, `AlertService`, `ConfigService`, `ReportService` |
| `handler` | HTTP request handling | 5 handler structs + shared `Handler` (holds `*sqlx.DB` for face endpoints) |
| `scheduler` | Cron job management | `Manager` (robfig/cron), `NightlyReportJob`, `HealthCheckJob` |
| `redis` | Dedup + caching | `Client` wrapper, `CheckAndSetDedup` (key: `dedup:{camera_id}:{frame_sequence}`) |
| `model/entity` | DB row mappings | 12 structs with `db:"column"` tags, `sql.Null*` for nullable columns |
| `model/enums` | Domain constants | `EventType`, `AlertType`, `CameraStatus`, `StudentStatus`, etc. |

## Critical Gotchas

### AlertConsumer Is Inert
`alert_consumer.go:84-88` — logs every message and commits offset. No action routing, no notification dispatch. Skeleton for future implementation.

### FaceEmbed Is a Stub
`handler/face.go:35` — `POST /api/face/embed` returns `{success: true, embedding: null}`. No ONNX model integration. FaceMatch (`handler/face.go:68`) works but does **O(n) full table scan** of `face_embedding` with cosine similarity — will not scale.

### ReportService Is a Skeleton
`service/report_service.go:26` — `GenerateNightlyReport` has a TODO, creates a placeholder report with zero aggregation.

### AES Key Mismatch with stream-gateway
`util/crypto.go:15` — dev key `"01234567890123456789012345678901"` differs from stream-gateway's dev key. Cross-module encrypt/decrypt **will fail** unless `CAMERA_ENCRYPTION_KEY` env var is set to the same value.

### Config Loading: CONFIG_PATH (Not CLI Flag)
`cmd/dormitory-service/main.go:33` — uses `CONFIG_PATH` env var, not `--config` CLI flag like stream-gateway. Config also accepts Spring Boot env vars (`SPRING_DATASOURCE_URL`, `KAFKA_BOOTSTRAP_SERVERS`, etc.) via `config.go:107-144`.

### Camera Limit Hardcoded
`service/camera_service.go:49` — max 50 cameras, checked via `FindAll()` count (not atomic). Race condition possible under concurrent registration.

### HealthCheck Pings Hardcoded URL
`service/camera_service.go:272` — `http://localhost:8080/health` is hardcoded. Won't work in Docker or non-default stream-gateway deployments.

### JWT Dev Secret
`config.yaml:25` — default secret `"your-256-bit-secret"`. `main.go:50` warns but does not block startup. Must match main backend's signing key in production via `JWT_SECRET` env var.

### BaseRepository Uses Reflection
`repository/base.go:88-112` — `getDBColumns[T]()` extracts columns from `db` struct tags via reflection. Entities without `db` tags produce zero columns → Create fails with "no columns found". ORDER BY is whitelist-validated (`base.go:47-61`) to prevent SQL injection.

### Redis Dedup Matches Java Format
`redis/client.go:83-85` — key `dedup:{camera_id}:{frame_sequence}`, TTL 3600s. Must stay compatible with any Java service rewrites.

### EventConsumer Bypasses Service Layer
`consumer/event_consumer.go` — calls repositories directly (not services) for event processing. Business logic in consumer is duplicated from service layer (student status update, stranger handling).

## Testing

**1 test file**: `repository/base_test.go` — tests generic `BaseRepository[T]` CRUD with `go-sqlmock` + `testify`. Uses external test package (`repository_test`).

**Zero tests for**: services, handlers, consumers, middleware, scheduler, client, redis, util, config.

## Configuration

| File | Usage |
|---|---|
| `config.yaml` | Local dev (port 8083, localhost Kafka/Redis/MariaDB) |
| `config.docker.yaml` | Docker Compose override (service hostnames) |

**Config precedence**: defaults < config.yaml < env vars. Viper-based with `AutomaticEnv()` and Spring Boot env var mapping.
