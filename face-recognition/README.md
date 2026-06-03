# face-recognition

Face recognition service — detects faces from Kafka frame stream, extracts embeddings, matches identities, analyzes behavior, and publishes events to downstream Kafka topics.

## Overview

face-recognition is the core processing node in the CampusVision AI perception pipeline:

```
t_dorm_frame (Kafka)
  → Base64 decode → JPEG → BGR
  → Night enhancement (CLAHE)
  → Face detection (RetinaFace ONNX / Haar Cascade fallback)
  → Feature extraction (ArcFace ONNX)
  → Identity matching (dormitory-service-go API + Redis cache)
  → Direction determination (ROI line crossing → in/out)
  → Event deduplication (Redis)
  → Behavior analysis (loitering/running/zone intrusion/crowd, disabled by default)
  → t_dorm_event (Kafka, raw JSON)
```

| Property   | Value                                     |
| ---------- | ----------------------------------------- |
| Language   | Python 3.11                               |
| Port       | None (pure Kafka consumer/producer)       |
| Entrypoint | `python -m app.main --config config.yaml` |
| Consumes   | `t_dorm_frame`                            |
| Produces   | `t_dorm_event`                            |

## Directory Structure

```
face-recognition/
├── app/
│   ├── main.py              # Entrypoint: Kafka consumer/producer main loop
│   ├── config.py            # 12 dataclass config definitions
│   ├── detector.py          # Face detection: ONNX RetinaFace + Haar Cascade fallback
│   ├── feature.py           # Feature extraction: ONNX ArcFace + zero-vector fallback
│   ├── matcher.py           # Identity matching: external API + Redis cache fallback
│   ├── direction.py         # Direction: ROI line crossing → in/out
│   ├── dedup.py             # Redis event deduplication
│   ├── tracker.py           # Face tracking (IoU, initialized when behavior enabled)
│   ├── behavior.py          # Behavior analysis: loitering/running/zone intrusion/crowd
│   ├── event_publisher.py   # Behavior event Kafka publisher
│   ├── night_mode.py        # Nighttime CLAHE enhancement
│   ├── download_models.py   # ONNX model downloader (SHA256 verification)
│   └── models/              # ONNX model files (gitignored)
│       └── model_urls.yaml  # Model URLs + SHA256 hashes
├── tests/                   # pytest tests (use Haar Cascade fallback, no ONNX needed)
├── config.yaml              # Local dev config
├── config.docker.yaml       # Docker Compose config override
├── requirements.txt
├── Dockerfile
└── ruff.toml
```

## Quick Start

### Prerequisites

- Python 3.11+
- Kafka running
- Redis running
- dormitory-service-go running (identity matching API)

### Install Dependencies

```bash
cd face-recognition
pip install -r requirements.txt
# Or using uv
uv pip install -r requirements.txt
```

### Download ONNX Models

```bash
# Auto-download (prefers hf-mirror.com → huggingface.co)
python -m app.download_models

# Custom mirror + proxy
python -m app.download_models --mirror https://hf-mirror.com --proxy http://127.0.0.1:7890

# List configured mirrors
python -m app.download_models --list-mirrors
```

### Local Development

```bash
python -m app.main --config config.yaml
```

### Docker

```bash
# Standard build
docker compose up -d face-recognition

# Build with mirror acceleration
docker compose build \
  --build-arg "HF_ENDPOINT=https://hf-mirror.com" \
  --build-arg "APT_MIRROR=https://mirrors.tuna.tsinghua.edu.cn/debian" \
  --build-arg "PIP_INDEX_URL=https://pypi.tuna.tsinghua.edu.cn/simple" \
  face-recognition
```

## Configuration

### config.yaml Core Settings

```yaml
kafka:
  brokers: ["localhost:9092"]
  frame_topic: "t_dorm_frame"
  event_topic: "t_dorm_event"
  group_id: "face-recognition-group"

detection:
  model_path: "app/models/retinaface-R50.onnx" # Empty string → Haar Cascade fallback
  confidence_threshold: 0.6
  input_size: [640, 640]
  min_face_size: 80
  blur_threshold: 100.0

feature:
  model_path: "app/models/arcface-resnet100.onnx"
  embedding_size: 512

match:
  method: "sims_api"
  sims_api_url: "http://localhost:8083/api/face/match"
  sims_api_timeout: 3.0
  match_threshold: 0.65
  fallback_to_cache: true # Fall back to Redis cache scan on API failure

redis:
  host: "localhost"
  port: 6379
  db: 0

behavior:
  enabled: false # Behavior analysis disabled by default
```

Full configuration reference: `config.yaml` and the 12 dataclasses in `app/config.py`.

## Core Mechanisms

### Dual Detection Strategy

| Mode     | Model                 | Trigger                                             |
| -------- | --------------------- | --------------------------------------------------- |
| Primary  | RetinaFace ONNX       | `detection.model_path` points to valid `.onnx` file |
| Fallback | Haar Cascade (OpenCV) | `model_path` empty or file not found                |

### Identity Matching Flow

1. Call `POST /api/face/match` (dormitory-service-go:8083) for cosine similarity matching
2. On API failure (timeout/connection error), fall back to Redis cache scan (`fallback_to_cache: true`)
3. Match threshold: `match_threshold` (default 0.65)

### Event Deduplication

Redis key format: `dedup:{student_id}:{direction}:{camera_id}`, TTL controlled by `dedup.window_seconds` (default 10s).

### Behavior Analysis (disabled by default)

Set `behavior.enabled: true` to enable. Supported behaviors:

| Behavior        | Threshold Config                                     |
| --------------- | ---------------------------------------------------- |
| Loitering       | `loitering_threshold_seconds`, `loitering_radius_px` |
| Running         | `running_speed_threshold_px_per_sec`                 |
| Zone Intrusion  | `zones[]` polygon definitions                        |
| Crowd Gathering | `crowd_threshold_count`, `crowd_debounce_frames`     |

## Testing

```bash
cd face-recognition && pytest tests/
```

Tests use Haar Cascade fallback (no ONNX models required):

| Test File                 | Coverage                                         |
| ------------------------- | ------------------------------------------------ |
| `test_detector.py`        | Haar fallback detection, blur filtering          |
| `test_behavior.py`        | Loitering/running/zone intrusion/crowd detection |
| `test_tracker.py`         | IoU tracking, trajectory expiration              |
| `test_event_publisher.py` | Behavior event Kafka publishing                  |
| `test_integration.py`     | End-to-end pipeline (mock Kafka)                 |

## Kafka Message Formats

### Consumed: t_dorm_frame

```json
{
  "camera_id": "cam-a1",
  "building": "A",
  "frame_sequence": 12345,
  "timestamp": "2025-01-15T10:30:00Z",
  "frame_data": "<base64 JPEG>",
  "width": 1280,
  "height": 720
}
```

### Produced: t_dorm_event (raw JSON, no Spring Kafka headers)

```json
{
  "event_type": "entry",
  "camera_id": "cam-a1",
  "building": "A",
  "student_id": "2024001",
  "student_name": "张三",
  "confidence": 0.87,
  "direction": "in",
  "timestamp": "2025-01-15T10:30:00Z",
  "snapshot_path": ""
}
```

## Caveats

- **ONNX models gitignored**: `*.onnx` files not committed — download via `download_models` or Docker build
- **Docker CMD**: Dockerfile includes `--config` flag, uses bind-mounted `config.docker.yaml`
- **Raw JSON**: Kafka messages are plain JSON (no `__TypeId__` headers) — Go consumer must handle accordingly
- **External API dependency**: Identity matching depends on dormitory-service-go's `/api/face/match` endpoint
