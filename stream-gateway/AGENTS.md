# stream-gateway — AGENTS.md
RTSP video ingestion service that decodes camera streams via ffmpeg and publishes frames to Kafka.

## Architecture

```
RTSP cameras → internal/decoder (ffmpeg subprocess) → YUV420P frames
  → internal/frame (motion detection) → internal/kafka.Producer → t_dorm_frame
```

**9 internal packages, strict unidirectional deps, zero interfaces:**
- `cmd/main.go` → `camera` → `decoder` + `frame` + `kafka`
- `camera` manages per-camera goroutines with dual cancellation (`ctx` + `stopCh`)
- `decoder/ffmpeg.go` spawns ffmpeg, reads raw YUV420P from stdout
- `frame/extractor.go` downsamples Y-plane to 160×90 for motion diff
- `kafka/producer.go` wraps kafka-go Writer with hash balancer (keyed by `building`)

## Entry Point

`cmd/main.go` (lines 25-108):
- Config loaded via CLI `--config` flag (not env var like other modules)
- Graceful shutdown: signal → `camManager.Stop()` → server `Shutdown(ctx)` → `cancel()`
- **Dual cancellation paths**: `dbPollLoop` watches `ctx.Done()`, camera streams watch `stopCh`
- Domain logic mixed with wiring: `dbPollLoop`, `syncCamerasFromDB` live here (lines 110-166)

## Key Packages

| Package | Purpose | Gotchas |
|---------|---------|---------|
| `internal/decoder` | ffmpeg subprocess management | Frame size hardcoded `width*height*3/2` (decoder.go:46); no retry on crash |
| `internal/kafka` | Kafka producer wrapper | No backpressure/retry; `KAFKA_BROKERS` mentioned in config.yaml comments but never read in code |
| `internal/health` | Health check HTTP handler | `ServeHTTP()` dead code (handler.go:35, never called); status throttled to every 30 frames or 5s |
| `internal/frame` | Motion detection + dynamic extraction | 160×90 downsampled Y-plane diff (extractor.go:48); threshold from config |
| `internal/camera` | Camera config + DB sync | `DiffAndSync` set-diff has race window during sync (manager.go:134-166) |
| `internal/crypto` | AES-256-GCM encryption | Dev keys differ from dormitory-service-go — cross-module decrypt fails |
| `internal/config` | YAML config structs | `LogConfig.Level` defined but unused (all stdlib `log.Printf`) |

## Critical Gotchas

1. **ffmpeg dependency**: Decoder spawns `ffmpeg` subprocess — must be on $PATH. Frame size hardcoded, no validation. (decoder/ffmpeg.go:46)
2. **KAFKA_BROKERS is a lie**: Comment says env var overrides brokers, but code never reads it. (config.yaml:20-24)
3. **Dead code**: `ServeHTTP()` in health handler is never wired up. (health/handler.go:35)
4. **Dual shutdown paths**: Some goroutines watch `ctx.Done()`, others watch `stopCh` — inconsistent. (cmd/main.go:99-108)
5. **Race in DiffAndSync**: Set-diff algorithm has window where camera can be deleted mid-sync. (camera/manager.go:134-166)
6. **No producer backpressure**: If Kafka is slow, frames queue unbounded in memory. (kafka/producer.go:52-63)
7. **Log level ignored**: `config.Log.Level` exists but all logging uses stdlib `log.Printf` with no level filtering. (config/config.go:117-119)

## Testing

**4 test files** (white-box, same package):
- `internal/health/handler_test.go` — health handler
- `internal/management/handler_test.go` — mgmt handler with X-Management-Key auth
- `internal/config/camera_config_test.go` — camera config parsing
- `internal/crypto/service_test.go` — AES-256-GCM encrypt/decrypt

**Coverage gaps**: `decoder`, `kafka`, `frame` have zero tests.
**Run**: `cd stream-gateway && go test ./...`
**No table-driven tests**, no coverage tooling, no `testify` except base_test.go.

## Configuration

**config.yaml fields:**
- `cameras[]`: static camera list (empty by default, DB polling preferred)
- `kafka.brokers`: `["localhost:9092"]` (Docker: `kafka:9092`)
- `kafka.topic`: `t_dorm_frame`
- `database.dsn`: MySQL connection string for dynamic camera sync
- `health.port`: 8080, `management.port`: 8081
- `log.level`: defined but unused

**Env vars:**
- `CAMERA_ENCRYPTION_KEY`: 32-byte AES-256-GCM key for RTSP password decryption
- `KAFKA_BROKERS`: documented in config.yaml comments but NOT implemented

**Docker override**: `config.docker.yaml` bind-mounted, overrides ports/hosts for container networking.

**Dynamic frame extraction** (config.yaml:10-17):
- `fps_day: 5`, `fps_night: 1`
- `motion_threshold: 0.05` (160×90 Y-plane mean abs diff)
- `dynamic_extraction: true` (gates motion-based capture)

**See root AGENTS.md** for: Kafka topics table, Docker Compose services, team division, cross-cutting patterns (dual config, AES key mismatch, module path inconsistencies).
