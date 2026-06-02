# face-recognition

人脸识别服务 — 从 Kafka 帧流中检测人脸、提取特征、匹配身份、分析行为，将事件发布到下游 Kafka topic。

## 概述

face-recognition 是 CampusVision AI 感知管线的核心处理节点：

```
t_dorm_frame (Kafka)
  → Base64 解码 → JPEG → BGR
  → 夜间增强 (CLAHE)
  → 人脸检测 (RetinaFace ONNX / Haar Cascade 回退)
  → 特征提取 (ArcFace ONNX)
  → 身份匹配 (dormitory-service-go API + Redis 缓存)
  → 方向判定 (ROI 线穿越 → 进/出)
  → 事件去重 (Redis)
  → 行为分析 (徘徊/奔跑/区域入侵/聚集, 默认关闭)
  → t_dorm_event (Kafka, raw JSON)
```

| 属性 | 值 |
|---|---|
| 语言 | Python 3.11 |
| 端口 | 无 HTTP (纯 Kafka consumer/producer) |
| 入口 | `python -m app.main --config config.yaml` |
| 消费 Topic | `t_dorm_frame` |
| 生产 Topic | `t_dorm_event` |

## 目录结构

```
face-recognition/
├── app/
│   ├── main.py              # 入口: Kafka consumer/producer 主循环
│   ├── config.py            # 12 个 dataclass 配置定义
│   ├── detector.py          # 人脸检测: ONNX RetinaFace + Haar Cascade 回退
│   ├── feature.py           # 特征提取: ONNX ArcFace + 零向量回退
│   ├── matcher.py           # 身份匹配: 外部 API + Redis 缓存回退
│   ├── direction.py         # 方向判定: ROI 线穿越 → 进/出
│   ├── dedup.py             # Redis 事件去重
│   ├── tracker.py           # 人脸跟踪 (IoU, 行为分析启用时初始化)
│   ├── behavior.py          # 行为分析: 徘徊/奔跑/区域入侵/聚集
│   ├── event_publisher.py   # 行为事件 Kafka 发布
│   ├── night_mode.py        # 夜间 CLAHE 增强
│   ├── download_models.py   # ONNX 模型下载器 (SHA256 校验)
│   └── models/              # ONNX 模型文件 (gitignored)
│       └── model_urls.yaml  # 模型 URL + SHA256 哈希
├── tests/                   # pytest 测试 (使用 Haar Cascade 回退, 无需 ONNX)
├── config.yaml              # 本地开发配置
├── config.docker.yaml       # Docker Compose 配置覆盖
├── requirements.txt
├── Dockerfile
└── ruff.toml
```

## 快速开始

### 前置条件

- Python 3.11+
- Kafka 运行中
- Redis 运行中
- dormitory-service-go 运行中 (身份匹配 API)

### 安装依赖

```bash
cd face-recognition
pip install -r requirements.txt
# 或使用 uv
uv pip install -r requirements.txt
```

### 下载 ONNX 模型

```bash
# 自动下载 (优先 hf-mirror.com → huggingface.co)
python -m app.download_models

# 自定义镜像 + 代理
python -m app.download_models --mirror https://hf-mirror.com --proxy http://127.0.0.1:7890

# 查看已配置的镜像
python -m app.download_models --list-mirrors
```

### 本地运行

```bash
python -m app.main --config config.yaml
```

### Docker 运行

```bash
# 标准构建
docker compose up -d face-recognition

# 带镜像加速构建
docker compose build \
  --build-arg "HF_ENDPOINT=https://hf-mirror.com" \
  --build-arg "APT_MIRROR=https://mirrors.tuna.tsinghua.edu.cn/debian" \
  --build-arg "PIP_INDEX_URL=https://pypi.tuna.tsinghua.edu.cn/simple" \
  face-recognition
```

## 配置

### config.yaml 核心配置

```yaml
kafka:
  brokers: ["localhost:9092"]
  frame_topic: "t_dorm_frame"
  event_topic: "t_dorm_event"
  group_id: "face-recognition-group"

detection:
  model_path: "app/models/retinaface-R50.onnx"  # 空字符串 → Haar Cascade 回退
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
  fallback_to_cache: true    # API 失败时回退到 Redis 缓存扫描

redis:
  host: "localhost"
  port: 6379
  db: 0

behavior:
  enabled: false             # 行为分析默认关闭
```

完整配置项见 `config.yaml` 和 `app/config.py` 中的 12 个 dataclass 定义。

## 核心机制

### 双检测策略

| 模式 | 模型 | 触发条件 |
|---|---|---|
| 主检测 | RetinaFace ONNX | `detection.model_path` 指向有效 `.onnx` 文件 |
| 回退检测 | Haar Cascade (OpenCV) | `model_path` 为空或文件不存在 |

### 身份匹配流程

1. 调用 `POST /api/face/match` (dormitory-service-go:8083) 进行余弦相似度匹配
2. API 失败时 (超时/连接错误)，回退到 Redis 缓存扫描 (`fallback_to_cache: true`)
3. 匹配阈值: `match_threshold` (默认 0.65)

### 事件去重

Redis key 格式: `dedup:{student_id}:{direction}:{camera_id}`，TTL 由 `dedup.window_seconds` 控制 (默认 10s)。

### 行为分析 (默认关闭)

设置 `behavior.enabled: true` 启用，支持：

| 行为 | 阈值配置 |
|---|---|
| 徘徊 (loitering) | `loitering_threshold_seconds`, `loitering_radius_px` |
| 奔跑 (running) | `running_speed_threshold_px_per_sec` |
| 区域入侵 (zone intrusion) | `zones[]` 定义多边形区域 |
| 人群聚集 (crowd) | `crowd_threshold_count`, `crowd_debounce_frames` |

## 测试

```bash
cd face-recognition && pytest tests/
```

测试使用 Haar Cascade 回退 (无需 ONNX 模型)：

| 测试文件 | 覆盖范围 |
|---|---|
| `test_detector.py` | Haar 回退检测、模糊过滤 |
| `test_behavior.py` | 徘徊/奔跑/区域入侵/聚集检测 |
| `test_tracker.py` | IoU 跟踪、轨迹过期 |
| `test_event_publisher.py` | 行为事件 Kafka 发布 |
| `test_integration.py` | 端到端管线 (mock Kafka) |

## Kafka 消息格式

### 消费: t_dorm_frame

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

### 生产: t_dorm_event (raw JSON, 无 Spring Kafka 头)

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

## 注意事项

- **ONNX 模型 gitignored**: `*.onnx` 文件不入库，需通过 `download_models` 或 Docker 构建下载
- **Docker CMD**: Dockerfile 已包含 `--config` 参数，使用 bind-mounted 的 `config.docker.yaml`
- **Raw JSON**: Kafka 消息为纯 JSON (无 `__TypeId__` 头)，Go 消费端需适配
- **外部 API 依赖**: 身份匹配依赖 dormitory-service-go 的 `/api/face/match` 接口
