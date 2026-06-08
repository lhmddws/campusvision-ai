# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
