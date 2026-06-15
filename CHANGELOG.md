# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-06-15

### Added

#### Frontend — Live Monitoring & Config System
- **Live monitoring page**: Real-time camera frame display with `FrameCanvas` Vue component and WebSocket-based live streaming (`frontend/src/views/live/`) — 4 new test files with 436 assertions
- **Config system optimization**: Dynamic `<el-select>` dropdowns driven by `config_options` from the database (migration 003), lazy-loaded ECharts (340KB gzip) / wangeditor (306KB) / xlsx (231KB) via dynamic `import()` — reducing initial bundle size
- **Test data seed**: 22 students, 4 cameras, 24 entry events, 4 nightly reports, 2 stranger records — idempotent `INSERT IGNORE` in `init.sql`
- **83 Vitest tests** across 11 test files covering dashboard, config, camera, attendance, alerts, events, face, inspection, layout, login, and smoke

#### Backend — WebSocket, Real-time Events & Frame Processing
- **Event Hub** (`dormitory-service-go/internal/handler/event_hub.go`): WebSocket-based real-time event broadcasting — pushes face recognition results to connected frontend clients in real time
- **Frame Consumer** (`dormitory-service-go/internal/consumer/frame_consumer.go`): Kafka consumer that processes `t_dorm_frame` frames directly — enables the dormitory service to receive and forward frames to the live monitoring page
- **DLQ Consumer** (`dormitory-service-go/internal/consumer/dlq_consumer.go`): Dead Letter Queue consumer for failed events — prevents message loss when processing errors occur
- **Live streaming handler** (`dormitory-service-go/internal/handler/live_handler.go`): HTTP + WebSocket endpoints for camera live feed — `/api/live/stream` with SSE/WS upgrade, includes comprehensive unit + integration tests (607 lines)
- **bbox fields**: `FaceEventMessage` and Kafka event schema now include `bbox` (bounding box) — face detection results carry precise face position (x, y, width, height)

#### API Extensions
- **Dynamic routes**: `/getRouters` endpoint for frontend route generation (#24)
- **Alert stats extension**: `AlertStats` response now includes trend data and severity breakdown (#24)
- **Camera field sanitization**: RTSP URL password masking — `/api/cameras` returns `sanitizeRtspURL` to hide credentials (`rtsp://admin:***@host:554/path`) (#24)
- **Camera DTO layer**: `ToDTO()` transforms entity → response with automatic field filtering, preventing password leaks in API responses

#### Stream Gateway Management API
- **Management handlers**: New HTTP endpoints on port 8081 (`X-Management-Key` auth) for camera CRUD operations — `GET/POST /api/cameras`, `PUT/DELETE /api/cameras/:id`, `GET /api/cameras/status` (stream-gateway/internal/management/)
- **Camera config sanitization**: Dedicated `camera/sanitize.go` for URL credential masking
- **Connection manager enhancements**: Thread-safe camera lifecycle tracking via `sync.Map`

#### Infrastructure
- **Health check scheduler**: Periodic camera health monitoring every 30 seconds in `dormitory-service-go` — auto-detects offline cameras and updates status
- **Frame topic config**: `frame_topic` field in stream-gateway config for customizable Kafka topic mapping
- **DB migrations**: `003_add_config_options.sql` (dropdown options), `004_sanitize_rtsp.sql` (camera URL sanitization)
- **Start/stop scripts**: `scripts/start-all.sh` and `scripts/stop-all.sh` — one-command full stack lifecycle management

### Changed

- **Dependencies**: Added `gorilla/websocket` for WebSocket support, `frame_topic` config field in stream-gateway configuration
- **Config schema**: `dormitory-service-go` config now includes `websocket.addr` and `websocket.allowed_origins` sections
- **Entity layer**: All 13 domain entities (`DormCamera`, `DormEventLog`, `DormAlert`, `DormStudent`, `DormFaceEmbedding`, `DormStrangerRecord`, etc.) updated with consistent `jsontype.Null*` fields and `db:` / `json:` tags
- **Docker Compose**: Added health checks for Zookeeper and Kafka services, `depends_on` with condition for `kafka-init`

### Fixed

- **JSON serialization**: Custom `jsontype.NullString` / `NullInt64` / `NullFloat64` / `NullBool` / `NullTime` types replace `sql.Null*` — fixes `null` value JSON marshaling across all API responses (`dormitory-service-go/internal/model/jsontype/null_types.go`)
- **Codebase Hardening**: Resolved all 9 Brooks-Lint findings across 3 modules:
  - `stream-gateway`: golangci-lint error-level issues in decoder, crypto, and camera packages
  - `face-recognition`: ruff error-level issues in detector, tracker, behavior, direction modules
  - `dormitory-service-go`: All linter errors in handlers, services, and entities — verified with `go vet` and `golangci-lint`
- **Frontend**: Fixed stale theme test assertions, resolved TypeScript deprecation warnings
- **Face matching**: `AppConfig` now correctly passed to `FaceMatcher` constructor, relaxed detection thresholds for debug mode
- **DLQ routing**: Failed Kafka messages are routed to DLQ topic instead of being silently dropped

### Performance

- **Frontend lazy loading**: Heavy dependencies split into on-demand chunks via Vite `rollupOptions` — reduces initial page load by ~850KB (uncompressed)
- **Stream gateway**: Configurable FPS adjustment via `management PUT /api/cameras/:id` — allows runtime frame rate tuning per camera

### Documentation

- **Project planning**: Comprehensive `doc/README.md` covering system architecture, tech stack rationale, recommended deployment topology, hardware specs (RTX 4070–A100), security guidelines, and future roadmap
- **OpenAPI specs**: Updated all 3 spec files (`stream-gateway-api.json`, `face-recognition-kafka.json`, `dormitory-service-api.json`) — new endpoints documented with request/response schemas

### Chores

- **.gitignore**: Added `skills-lock.json` to ignore list
- **uv.lock**: Python dependency lockfile for reproducible builds (507 lines)

## [0.4.0] - 2026-06-08

### Added

- **infra**: Mediamtx RTSP server service for camera streaming simulation — enables end-to-end testing with synthetic RTSP feeds in Docker Compose (#18)
- **face-recognition**: Switch to RetinaFace-MobileNetV2 (lighter, faster) with corrected image preprocessing (BGR→RGB conversion, proper letterboxing) — improved detection accuracy and throughput (#19)
- **frontend**: Update branding title, fix sidebar route navigation, add login page wireframe (#20, #22)
- **deploy**: SSH deployment script and production environment config (`deploy.sh`) — one-command deploy to remote host
- **ci**: GitHub Actions CI pipeline (Go build+test, Python lint+test, frontend lint+build), Dependabot config for automated dependency updates, PR template (#18)

### Performance

- **face-recognition**: Migrate Dockerfile from `pip` to `uv` — ~10x faster dependency installation in Docker builds
- **stream-gateway**: Add `-ldflags -s -w` to Go build — reduces binary size by ~30%
- **dormitory-service-go**: Add `-ldflags -s -w` to Go build — reduces binary size by ~30%

### Fixed

- **stream-gateway**: Resolve all `golangci-lint` error-level issues, verify build + test pass
- **face-recognition**: Resolve all `ruff` error-level issues, pass full `AppConfig` to `FaceMatcher`, relax detection thresholds for debug mode, add `pyproject.toml` — clean linting and configurable sensitivity
- **dormitory-service-go**: Resolve all linter errors, add `AlertHandler`/`FaceHandler`/`RecordHandler` tests — verified build + test pass
- **frontend**: Update stale theme test assertions, resolve TypeScript deprecation warnings

### Tests

- **dormitory-service-go**: Add comprehensive handler tests — `AlertHandler` (GetAlerts, AcknowledgeAlert), `FaceHandler` (Create, List), `RecordHandler` (GetAttendanceStats, GetEvents)
- **test**: Add test scripts, pipeline plan, Lena test fixture image
- **test**: Fix checkmark encoding in kafka verify script (#21)
- **e2e**: Add end-to-end test documentation (test-flow docs)

### Documentation

- **CHANGELOG, CONTRIBUTING, SECURITY**: Add standard project governance docs
- **README**: Add GitHub badges (Go, Python, Vue, Docker, License, PRs welcome) and repo links
- **test-flow**: Document end-to-end test scenarios and verification procedures

### Chores

- **monorepo**: Add Go workspace (`go.work`) for multi-module development
- **infrastructure**: Root Makefile, tooling scripts, project license
- **frontend**: Update pnpm lockfile with workspace configuration

## [0.1.0] - 2025-05-28

### Added

- **stream-gateway**: RTSP stream ingestion with motion-based frame extraction — captures video from up to 4 cameras (A/B/C/D), decodes via ffmpeg subprocess, extracts frames at configurable FPS (day: 5, night: 1), applies motion threshold gating (0.05), and produces hash-partitioned frames to `t_dorm_frame` Kafka topic with Snappy compression
- **face-recognition**: Face detection (RetinaFace ONNX) with Haar Cascade fallback, ArcFace ONNX embedding extraction, identity matching via external API with Redis cache fallback — consumes `t_dorm_frame`, publishes entry/exit events and behavior alerts to `t_dorm_event`
- **dormitory-service-go**: Event processing, attendance tracking, alert management, and face matching HTTP API (Gin framework on port 8083) — 22+ endpoints including camera CRUD, attendance records, alerts, system configuration, and nightly report generation
- **frontend**: Vue 3.2 + TypeScript SPA dashboard (Element Plus + Vite 3) — real-time monitoring, camera management, entry/exit event queries, alert handling, attendance statistics, face records library, and system administration (users, roles, menus)
- **Infrastructure**: Docker Compose with Zookeeper (2181), Kafka (9092, 3 topics auto-created), Redis (6379), MariaDB (3306, 11 tables via init.sql), MinIO (9000/9001) — single-command infrastructure bootstrap
- **Dual config system**: Every service ships `config.yaml` (local dev) and `config.docker.yaml` (Docker Compose override) — consistent configuration across environments
- **Redis-based event deduplication**: Both face-recognition (Python) and dormitory-service-go (Go) share Redis db=0 with matching dedup key format — prevents duplicate event processing with configurable TTL (default 3600s)
- **Dynamic frame extraction**: Day/night FPS profiles (`fps_day: 5`, `fps_night: 1`) controlled by configurable night mode hour range — motion_threshold (0.05) gates frame production during low-activity periods
- **Behavior analysis pipeline**: Loitering detection (configurable threshold seconds + radius), running speed detection, zone intrusion (polygon definition), crowd gathering detection — gated by `behavior.enabled: false` (inert by default)
- **ONNX model management**: Two verified models (retinaface-R50, arcface-resnet100) with SHA256 hashes in `model_urls.yaml` — auto-download with mirror/proxy support for restricted networks (hf-mirror.com → huggingface.co fallback)
- **Night mode enhancement**: CLAHE (Contrast Limited Adaptive Histogram Equalization) preprocessing for low-light frames — configurable clip limit and active hours
- **GitHub repository ecosystem**: `.golangci.yml` linter configuration, `ruff.toml` Python linter, `.editorconfig` cross-editor settings, `frontend/.eslintrc.js` + `frontend/.prettierrc` for frontend code quality

### Fixed

- **Haar Cascade path**: Uses `cv2.data.haarcascades` (OpenCV-bundled, portable) instead of hardcoded macOS Homebrew path — fixes face detection fallback on non-macOS platforms (commit f6a24e0)
- **AES encryption key sync**: Stream-gateway and dormitory-service-go now read `CAMERA_ENCRYPTION_KEY` from the same env var — camera passwords encrypted/decrypted with matching AES-256-GCM
- **DB schema alignment**: `infra/mariadb/init.sql` and Go entity structs reconciled — table names, column types, and constraints verified across both sources via migration 001

### Known Issues

- `AlertConsumer` in dormitory-service-go is a skeleton — `t_dorm_alert` topic consumed but messages are only logged, not persisted or dispatched
- `POST /api/face/embed` endpoint in dormitory-service-go returns null embedding — not currently consumed by face-recognition (embedding is done locally via ONNX), but the endpoint stub is non-functional
- `ReportService.GenerateNightlyReport` is a placeholder — nightly attendance reports log a message but produce no output file or database record
- `POST /api/face/match` uses O(n) full table scan against `dorm_face_feature` — no vector index, degrades linearly with face record count
- No CI pipeline configured — `.github/workflows/` directory does not exist; linter configs are present but not executed automatically
