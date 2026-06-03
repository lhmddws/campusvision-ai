# stream-gateway

RTSP video stream gateway — captures RTSP streams from dormitory cameras, decodes frames via ffmpeg, and publishes to Kafka after motion detection.

## Overview

stream-gateway is the first hop in the CampusVision AI perception pipeline:

```
RTSP Cameras (A/B/C/D)
  → ffmpeg decode (YUV420P)
  → Motion detection (160×90 Y-plane differential)
  → Kafka Producer → t_dorm_frame (hash by building, Snappy)
```

| Property    | Value                                                                 |
| ----------- | --------------------------------------------------------------------- |
| Language    | Go 1.26                                                               |
| Ports       | 8080 (health), 8081 (management API)                                  |
| Entrypoint  | `cmd/main.go`                                                         |
| Kafka Topic | `t_dorm_frame` (4 partitions, hash by `building`, Snappy compression) |

## Directory Structure

```
stream-gateway/
├── cmd/main.go              # Entrypoint: config loading, signal handling, DB polling, HTTP servers
├── internal/
│   ├── camera/              # Camera management: goroutine lifecycle, DB sync
│   ├── config/              # YAML config structs
│   ├── crypto/              # AES-256-GCM RTSP password encryption/decryption
│   ├── decoder/             # ffmpeg subprocess: RTSP → raw YUV420P frames
│   ├── frame/               # Motion detection: Y-plane downsampled differential
│   ├── health/              # Health check HTTP handler
│   ├── kafka/               # Kafka Producer wrapper (hash balancer)
│   └── management/          # Management API (X-Management-Key auth)
├── config.yaml              # Local dev config
├── config.docker.yaml       # Docker Compose config override
├── Dockerfile
└── go.mod
```

## Quick Start

### Prerequisites

- Go 1.26+
- ffmpeg installed and available in `$PATH`
- Kafka running (`docker compose up -d kafka`)
- MariaDB running (optional, for dynamic camera sync)

### Local Development

```bash
cd stream-gateway
go run cmd/main.go --config config.yaml
```

### Docker

```bash
docker compose up -d stream-gateway
```

## Configuration

### config.yaml

```yaml
frame:
  fps_day: 5 # Daytime frame rate
  fps_night: 1 # Nighttime frame rate
  jpeg_quality: 80
  width: 1280
  height: 720
  dynamic_extraction: true # Enable motion-based dynamic frame extraction
  motion_threshold: 0.05 # Motion threshold (160×90 Y-plane mean absolute differential)

kafka:
  brokers: ["localhost:9092"]
  topic: "t_dorm_frame"
  compression: "snappy"
  batch_size: 65536

rtsp:
  reconnect_interval: 5s
  read_timeout: 10s
  max_reconnect_attempts: 0 # 0 = infinite retry

health:
  port: 8080

management:
  port: 8081
  bind_address: "127.0.0.1"
  management_key: "" # Empty = no auth

database:
  dsn: "root:root@tcp(localhost:3306)/dormitory"
  driver: "mysql"
  poll_interval: 30s # DB polling interval
```

### Environment Variables

| Variable                | Description                                                                                   |
| ----------------------- | --------------------------------------------------------------------------------------------- |
| `CAMERA_ENCRYPTION_KEY` | 32-byte AES-256-GCM key for RTSP password decryption. Falls back to built-in dev key if unset |

> **Note**: `CAMERA_ENCRYPTION_KEY` must match the one in `dormitory-service-go`, otherwise cross-module password decryption will fail.

## Core Mechanisms

### Dynamic Frame Extraction

Instead of fixed-rate frame capture, stream-gateway dynamically decides based on scene motion:

1. ffmpeg decodes RTSP to raw YUV420P frames
2. Y-plane downsampled to 160×90 grayscale
3. Mean absolute differential (MAD) computed against previous frame
4. Frame published to Kafka only when MAD exceeds `motion_threshold`

Daytime caps at `fps_day` (5fps), nighttime drops to `fps_night` (1fps).

### Camera Management

- **DB Polling**: Syncs camera list from `dorm_camera` table every 30s (requires `database.dsn`)
- **Management API**: Add/remove cameras dynamically via port 8081 (requires `X-Management-Key`)
- **Lifecycle**: Each camera runs in its own goroutine with start/stop/reconnect support

### Graceful Shutdown

Signal → stop all camera goroutines → HTTP server Shutdown → context cancel

## Testing

```bash
cd stream-gateway && go test ./...
```

| Test File                               | Coverage                          |
| --------------------------------------- | --------------------------------- |
| `internal/health/handler_test.go`       | Health check handler              |
| `internal/management/handler_test.go`   | Management API handler            |
| `internal/config/camera_config_test.go` | Config parsing                    |
| `internal/crypto/service_test.go`       | AES-256-GCM encryption/decryption |

## API

### Health Check (port 8080)

| Method | Path      | Description           |
| ------ | --------- | --------------------- |
| GET    | `/health` | Service health status |

### Management API (port 8081)

| Method | Path           | Description            |
| ------ | -------------- | ---------------------- |
| GET    | `/cameras`     | List all cameras       |
| POST   | `/cameras`     | Add camera             |
| DELETE | `/cameras/:id` | Remove camera          |
| GET    | `/status`      | Gateway runtime status |

> Management API requires `management.management_key` configured and `X-Management-Key` header in requests.

Full API spec: [`doc/api/stream-gateway-api.json`](../doc/api/stream-gateway-api.json).

## Caveats

- **ffmpeg dependency**: Decoder spawns ffmpeg subprocess — must be available in `$PATH`
- **Hardcoded frame size**: `width × height × 3 / 2` bytes (YUV420P), no validation
- **No producer backpressure**: Frames queue indefinitely in memory when Kafka is slow
- **Log level unimplemented**: `config.Log.Level` is defined but code uses stdlib `log.Printf`
