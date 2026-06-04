# CampusVision AI — 测试流程构建与优化计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) for syntax tracking.

**Goal:** 建立可重复、可验证的端到端测试流程，覆盖 Docker 构建、Kafka 消息流、ONNX 模型推理、RTSP 模拟摄像头，并将调试脚本规范化纳入仓库

**Architecture:**
```
RTSP cameras (simulated) → stream-gateway (Go) → t_dorm_frame (Kafka) → face-recognition (Python)
  → t_dorm_event (Kafka) → dormitory-service-go (Go) → MariaDB + Redis
```

**Tech Stack:** Go 1.26, Python 3.11, Docker Compose, Kafka 7.6, Redis 7, MariaDB 10.11, ONNX Runtime, OpenCV, Mediamtx (RTSP)

---

## 一、背景与现状分析

### 现状总结

经过前期调试已确认：
1. ✅ face-recognition 测试全部通过（41 passed, 2 skipped）
2. ✅ Kafka 端到端帧注入流水线打通（Python kafka-python → localhost:29092 → Docker Kafka）
3. ✅ face-recognition 容器成功消费帧并进入处理管线
4. ❌ ONNX RetinaFace 模型检测返回 0 个人脸（faces_detected=0）
5. ✅ Haar Cascade 回退检测在容器内可正常检测人脸
6. ❌ Docker bind mount 在 Windows 下存在缓存问题（config 更新不生效）
7. ❌ Docker credential 故障导致无法拉取公开镜像
8. ❌ 网络受限（中国防火墙），部分外部资源不可达

### 待解决问题

- ONNX 模型推理返回 0 检测的原因未确认（模型加载耗时导致 exec 中无法测试）
- 无自动化 RTSP 摄像头模拟
- 测试脚本散落在临时目录，未纳入仓库
- Docker 构建流程缺少统一编排（build-args、镜像源、模型下载）
- 缺乏端到端一键测试脚本

---

## 二、文件结构规划

```
campusvision-ai/
├── scripts/
│   ├── test/
│   │   ├── README.md                    # 测试工具说明
│   │   ├── kafka_inject_frames.py       # 向 Kafka 注入测试帧
│   │   ├── kafka_consume_verify.py      # 消费并验证帧格式
│   │   ├── run_pipeline_test.sh         # 一键端到端测试
│   │   └── container_exec_test.py       # 容器内检测诊断
│   ├── rtsp/
│   │   ├── simulate_camera.py           # 用 Mediamtx + FFmpeg 模拟摄像头
│   │   └── mediamtx.yml                 # Mediamtx 配置文件
│   └── docker/
│       ├── rebuild-all.sh               # 一键重建所有服务
│       └── check-health.sh              # 健康检查脚本
├── doc/
│   ├── plans/
│   │   └── test-flow-plan.md            # ← 本文档
│   └── test-flow/
│       ├── 01-infrastructure.md         # 基础设施启动
│       ├── 02-kafka-test.md             # Kafka 消息测试
│       ├── 03-face-detection.md         # 人脸检测验证
│       └── 04-end-to-end.md             # 端到端流程
├── face-recognition/
│   └── tests/
│       └── fixtures/
│           ├── face.jpg                 # 原始测试夹具（200x200）
│           └── face_test.png            # Lena 测试图（重命名，原 face_test.jpg）
```

---

## 三、Phase 1：清理与提交

### Task 1.1：清理无用文件

- [ ] **Step 1: 移除 lfw.tgz（20 字节，下载失败）**

```bash
git rm --cached face-recognition/tests/fixtures/lfw.tgz
del face-recognition\tests\fixtures\lfw.tgz
```

- [ ] **Step 2: 将 face_test.jpg 重命名为 face_test.png（实际为 PNG 格式）**

```bash
ren face-recognition\tests\fixtures\face_test.jpg face_test.png
git add face-recognition/tests/fixtures/face_test.png
```

- [ ] **Step 3: 将 face_test.jpg 添加到 .gitignore（已不存在，确保 git 不追踪旧路径）**

核实 `.gitignore` 中是否包含 `*.onnx` 和 `tests/fixtures/` 规则。

### Task 1.2：提交代码变更

**变更文件：**
- `face-recognition/app/main.py` — FaceMatcher(cfg.match) → FaceMatcher(cfg)
- `face-recognition/app/matcher.py` — self.config → self.config.match.*
- `face-recognition/config.docker.yaml` — 检测阈值调整
- `frontend/pnpm-lock.yaml` — lock 文件更新（无关变更，需确认是否提交）
- `frontend/pnpm-workspace.yaml` — 新增 pnpm 配置

- [ ] **Step 1: 生成语义化提交**

```bash
git add face-recognition/app/main.py
git add face-recognition/app/matcher.py
git commit -m "fix(face-recognition): pass full AppConfig to FaceMatcher"

git add face-recognition/config.docker.yaml
git commit -m "chore(face-recognition): relax detection thresholds for debugging"

git add frontend/pnpm-workspace.yaml
git add frontend/pnpm-lock.yaml
git commit -m "chore(frontend): add pnpm workspace config and update lockfile"

git add face-recognition/tests/fixtures/face_test.png
git commit -m "test(face-recognition): add Lena test fixture for face detection"
```

- [ ] **Step 2: 推送到远程**

```bash
git push origin main
```

---

## 四、Phase 2：测试工具规范化

### Task 2.1：创建 scripts/test/ 目录与入口说明

- [ ] **Step 1: 创建目录结构**

```bash
mkdir scripts\test
mkdir scripts\rtsp
mkdir scripts\docker
```

- [ ] **Step 2: 编写 `scripts/test/README.md`**

```markdown
# CampusVision AI — 测试工具集

## 前置条件
- Python 3.11+ with `kafka-python`, `opencv-python-headless`, `numpy`
- Docker Compose 正常运行
- Kafka 暴露本机端口 29092

## 工具列表

| 脚本 | 用途 | 用法 |
|------|------|------|
| `kafka_inject_frames.py` | 向 t_dorm_frame 注入测试帧 | `python kafka_inject_frames.py [--image path] [--count N]` |
| `kafka_consume_verify.py` | 消费并验证帧格式/检测结果 | `python kafka_consume_verify.py` |
| `run_pipeline_test.sh` | 一键端到端测试 | `bash run_pipeline_test.sh` |
| `container_exec_test.py` | 容器内检测诊断 | `python container_exec_test.py` |

## 快速启动
```bash
# 安装依赖
pip install kafka-python opencv-python-headless numpy

# 注入帧
python scripts/test/kafka_inject_frames.py --image face-recognition/tests/fixtures/face_test.png --count 5

# 验证消费
python scripts/test/kafka_consume_verify.py
```
```

### Task 2.2：规范化 Kafka 帧注入脚本

- [ ] **Step 1: 将 `send_lena_frames.py` 重写为 `scripts/test/kafka_inject_frames.py`**

```python
#!/usr/bin/env python3
"""向 Kafka t_dorm_frame topic 注入测试帧。

用法:
  python scripts/test/kafka_inject_frames.py \\
      --image path/to/test.jpg \\
      --count 20 \\
      --topic t_dorm_frame \\
      --brokers localhost:29092 \\
      --building A \\
      --camera test_cam_001

依赖: kafka-python, opencv-python-headless
"""
import argparse
import base64
import json
import sys
import time

import cv2
import numpy as np
from kafka import KafkaProducer


def encode_frame(image_path: str) -> tuple[str, int, int]:
    """读取图片，编码为 base64 JPEG。返回 (base64_str, width, height)。"""
    img = cv2.imread(image_path, cv2.IMREAD_COLOR)
    if img is None:
        # 尝试以 raw bytes 加载（PNG/其他格式）
        with open(image_path, "rb") as f:
            raw = f.read()
        np_arr = np.frombuffer(raw, dtype=np.uint8)
        img = cv2.imdecode(np_arr, cv2.IMREAD_COLOR)
        if img is None:
            raise ValueError(f"无法解码图片: {image_path}")

    height, width = img.shape[:2]
    _, buffer = cv2.imencode(".jpg", img, [cv2.IMWRITE_JPEG_QUALITY, 95])
    b64 = base64.b64encode(buffer).decode("utf-8")
    return b64, width, height


def main():
    parser = argparse.ArgumentParser(description="向 Kafka 注入测试帧")
    parser.add_argument("--image", required=True, help="测试图片路径")
    parser.add_argument("--count", type=int, default=10, help="发送帧数")
    parser.add_argument("--topic", default="t_dorm_frame", help="Kafka topic")
    parser.add_argument("--brokers", default="localhost:29092", help="Kafka brokers")
    parser.add_argument("--building", default="A", help="楼栋标识")
    parser.add_argument("--camera", default="test_cam_001", help="摄像头 ID")
    parser.add_argument("--interval", type=float, default=0.1, help="帧间隔(秒)")
    parser.add_argument("--key", default=None, help="分区键(默认=building)")
    args = parser.parse_args()

    # 编码图片
    sys.stderr.write(f"加载图片: {args.image}\n")
    frame_data, width, height = encode_frame(args.image)
    sys.stderr.write(f"图片尺寸: {width}x{height}, base64: {len(frame_data)} bytes\n")

    # 创建 producer
    producer = KafkaProducer(
        bootstrap_servers=args.brokers,
        value_serializer=lambda v: json.dumps(v).encode("utf-8"),
        key_serializer=lambda k: k.encode("utf-8"),
        acks=1,
    )

    partition_key = args.key or args.building
    for seq in range(args.count):
        msg = {
            "camera_id": args.camera,
            "building": args.building,
            "frame_sequence": seq,
            "frame_data": frame_data,
            "frame_width": width,
            "frame_height": height,
            "timestamp": int(time.time() * 1000),
        }
        future = producer.send(args.topic, value=msg, key=partition_key)
        result = future.get(timeout=10)
        sys.stderr.write(f"  seq={seq:3d} offset={result.offset}\n")
        time.sleep(args.interval)

    producer.flush()
    producer.close()
    sys.stderr.write(f"完成: 发送 {args.count} 帧到 {args.topic}\n")


if __name__ == "__main__":
    main()
```

### Task 2.3：规范化 Kafka 消费验证脚本

- [ ] **Step 1: 创建 `scripts/test/kafka_consume_verify.py`**

```python
#!/usr/bin/env python3
"""消费 Kafka t_dorm_frame topic 并验证帧格式与人脸检测。

用法:
  python scripts/test/kafka_consume_verify.py \\
      --count 5 \\
      --brokers localhost:29092 \\
      --topic t_dorm_frame

依赖: kafka-python, opencv-python-headless, numpy
"""
import argparse
import base64
import json
import sys

import cv2
import numpy as np
from kafka import KafkaConsumer, TopicPartition


def main():
    parser = argparse.ArgumentParser(description="消费并验证 Kafka 帧")
    parser.add_argument("--count", type=int, default=3, help="验证帧数")
    parser.add_argument("--brokers", default="localhost:29092", help="Kafka brokers")
    parser.add_argument("--topic", default="t_dorm_frame", help="Kafka topic")
    parser.add_argument("--partition", type=int, default=0, help="分区编号")
    parser.add_argument("--seek-latest", type=int, default=5, help="从末尾前移 N 帧")
    args = parser.parse_args()

    consumer = KafkaConsumer(
        bootstrap_servers=args.brokers,
        value_deserializer=lambda m: json.loads(m.decode()),
        consumer_timeout_ms=10000,
        group_id=None,  # 无 consumer group，只读不提交
    )

    tp = TopicPartition(args.topic, args.partition)
    consumer.assign([tp])

    end_offset = consumer.end_offsets([tp])[tp]
    seek_to = max(0, end_offset - args.seek_latest)
    consumer.seek(tp, seek_to)
    print(f"Topic: {args.topic}[{args.partition}], end_offset={end_offset}, seek_to={seek_to}")

    # Haar Cascade 检测器
    cascade = cv2.CascadeClassifier(
        cv2.data.haarcascades + "haarcascade_frontalface_default.xml"
    )

    count = 0
    for msg in consumer:
        val = msg.value
        is_valid = True

        # 验证必要字段
        required = ["camera_id", "building", "frame_sequence", "frame_data",
                     "frame_width", "frame_height", "timestamp"]
        missing = [k for k in required if k not in val]
        if missing:
            print(f"  [FAIL] seq={val.get('frame_sequence', '?')} 缺少字段: {missing}")
            continue

        # 解码帧
        try:
            frame_bytes = base64.b64decode(val["frame_data"])
            np_arr = np.frombuffer(frame_bytes, dtype=np.uint8)
            frame = cv2.imdecode(np_arr, cv2.IMREAD_COLOR)
            if frame is None:
                print(f"  [FAIL] seq={val['frame_sequence']} 帧解码失败")
                continue
        except Exception as e:
            print(f"  [FAIL] seq={val['frame_sequence']} 解码异常: {e}")
            continue

        # 人脸检测
        gray = cv2.cvtColor(frame, cv2.COLOR_BGR2GRAY)
        rects = cascade.detectMultiScale(gray, 1.1, 5, minSize=(40, 40))

        actual_h, actual_w = frame.shape[:2]
        declared_w, declared_h = val["frame_width"], val["frame_height"]
        match = "✓" if actual_w == declared_w and actual_h == declared_h else "✗"

        print(f"  seq={val['frame_sequence']:3d} "
              f"size=({actual_w}x{actual_h}) declared=({declared_w}x{declared_h}) [{match}] "
              f"faces={len(rects)} camera={val['camera_id']}")

        for x, y, w, h in rects:
            print(f"    face at ({x},{y}) {w}x{h}")

        count += 1
        if count >= args.count:
            break

    consumer.close()
    print(f"\n验证完成: 检查了 {count} 帧")


if __name__ == "__main__":
    main()
```

### Task 2.4：创建容器内检测诊断脚本

- [ ] **Step 1: 创建 `scripts/test/container_exec_test.py`**

```python
#!/usr/bin/env python3
"""在 Docker face-recognition 容器内执行检测诊断。

用法:
  python scripts/test/container_exec_test.py \\
      --container cv-face-recognition \\
      --config config.yaml        # 容器内配置路径
"""
import argparse
import subprocess
import sys
import tempfile


DIAG_SCRIPT = r'''"""Diagnostic: verify face detection inside container."""
import sys, json, base64, os
os.chdir("/app")
sys.path.insert(0, "/app")

import numpy as np
import cv2
from kafka import KafkaConsumer, TopicPartition
from app.detector import FaceDetector
from app.night_mode import NightModeEnhancer
from app.config import load_config

cfg = load_config("config.yaml")
detector = FaceDetector(
    model_path=cfg.detection.model_path,
    conf_threshold=cfg.detection.confidence_threshold,
    input_size=tuple(cfg.detection.input_size),
    min_face_size=cfg.detection.min_face_size,
    blur_threshold=cfg.detection.blur_threshold,
    nms_iou_threshold=cfg.detection.nms_iou_threshold,
)
enhancer = NightModeEnhancer(cfg.night_mode)

results = []
results.append(f"model_path={cfg.detection.model_path!r}")
results.append(f"min_face_size={detector.min_face_size}")
results.append(f"conf_threshold={detector.conf_threshold}")
results.append(f"session={'ONNX' if detector.session else 'Haar (fallback)'}")

# Read latest frames from Kafka
consumer = KafkaConsumer(
    bootstrap_servers="kafka:9092",
    value_deserializer=lambda m: json.loads(m.decode()),
    consumer_timeout_ms=15000,
    group_id=None,
)
tp = TopicPartition("t_dorm_frame", 0)
consumer.assign([tp])
end = consumer.end_offsets([tp])[tp]
consumer.seek(tp, max(0, end - 5))

count = 0
for msg in consumer:
    v = msg.value
    d = base64.b64decode(v["frame_data"])
    f = cv2.imdecode(np.frombuffer(d, np.uint8), 1)
    if f is None:
        results.append(f"seq={v['frame_sequence']} DECODE FAILED")
    else:
        f2 = enhancer.enhance(f)
        faces = detector.detect(f2)
        results.append(f"seq={v['frame_sequence']} shape={f.shape} faces={len(faces)}")
        for face in faces:
            results.append(f"  face: ({face.x1:.0f},{face.y1:.0f})-({face.x2:.0f},{face.y2:.0f}) conf={face.confidence:.3f}")
    count += 1
    if count >= 3:
        break

results.append(f"done (read {count})")
with open("/tmp/diag_result.txt", "w") as out:
    out.write("\n".join(results))
print("OK: results written to /tmp/diag_result.txt")
'''


def main():
    parser = argparse.ArgumentParser(description="容器内检测诊断")
    parser.add_argument("--container", default="cv-face-recognition", help="容器名")
    parser.add_argument("--config", default="config.yaml", help="配置路径")
    args = parser.parse_args()

    # 写入诊断脚本
    with tempfile.NamedTemporaryFile(
        mode="w", suffix=".py", delete=False, prefix="diag_"
    ) as f:
        f.write(DIAG_SCRIPT.replace('config.yaml', args.config))
        script_path = f.name

    # 复制到容器
    subprocess.run(
        ["docker", "cp", script_path,
         f"{args.container}:/tmp/container_diag.py"],
        check=True
    )

    # 执行
    result = subprocess.run(
        ["docker", "compose", "exec", "-T", args.container,
         "timeout", "30", "python", "/tmp/container_diag.py"],
        capture_output=True, text=True
    )
    print(result.stdout)
    if result.stderr:
        print(f"STDERR: {result.stderr[:500]}", file=sys.stderr)

    # 读取结果
    result2 = subprocess.run(
        ["docker", "compose", "exec", args.container, "cat", "/tmp/diag_result.txt"],
        capture_output=True, text=True
    )
    if result2.stdout:
        print("\n--- Diagnostic Results ---")
        print(result2.stdout)


if __name__ == "__main__":
    main()
```

---

## 五、Phase 3：Docker 构建流程梳理

### Task 3.1：检查 face-recognition Dockerfile

**当前 `face-recognition/Dockerfile` 问题：**

| 问题 | 影响 | 修复方案 |
|------|------|---------|
| 使用 `pip` 而不是 `uv` | 构建速度慢 | 可选改用 `uv`，保留 pip 作为 fallback |
| config 文件名混淆（`config.docker.yaml` COPY 为 `config.yaml`） | 调试时配置查找路径不直观 | 保持当前做法（docker-compose 挂载），增加文档说明 |
| ONNX 模型下载策略复杂（3 种策略） | 构建逻辑不清晰 | 简化策略：运行时下载 + Docker volume 持久化 |
| 无 `--config` 参数的 CMD | 依赖默认 config.yaml | **已修复**：CMD 含 `--config config.yaml` |

- [ ] **Step 1: 优化 Dockerfile，使用 uv 加速安装**

```dockerfile
FROM python:3.11-slim

ARG APT_MIRROR
ARG PIP_INDEX_URL
ARG PIP_TRUSTED_HOST
ARG HF_ENDPOINT=https://huggingface.co

ENV HF_ENDPOINT=${HF_ENDPOINT}

# 系统依赖
RUN if [ -n "$APT_MIRROR" ]; then \
        rm -f /etc/apt/sources.list /etc/apt/sources.list.d/*.sources 2>/dev/null; \
        echo "deb $APT_MIRROR bookworm main" > /etc/apt/sources.list; \
    fi && \
    apt-get update && apt-get install -y --no-install-recommends \
    libgl1 libglib2.0-0 curl \
    && rm -rf /var/lib/apt/lists/*

# 安装 uv（提速 10x）
COPY --from=ghcr.io/astral-sh/uv:latest /uv /uvx /bin/

WORKDIR /app

COPY pyproject.toml uv.lock* requirements.txt ./
RUN uv pip install --system \
    $( [ -n "$PIP_INDEX_URL" ] && echo "--index-url $PIP_INDEX_URL" ) \
    -r requirements.txt

COPY config.docker.yaml config.yaml
COPY app/ app/

# ONNX 模型：构建时下载（可配置）
ARG BUILD_MODELS=0
RUN if [ "$BUILD_MODELS" = "1" ]; then \
        python -m app.download_models; \
    fi

USER appuser
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
  CMD python -c "import app.main; exit(0)" 2>/dev/null || exit 1

CMD ["python", "-m", "app.main", "--config", "config.yaml"]
```

- [ ] **Step 2: 添加 pyproject.toml（可选，用于 uv 约束）**

```toml
[project]
name = "face-recognition"
version = "0.1.0"
requires-python = ">=3.11"
dependencies = [
    "opencv-python-headless",
    "onnxruntime",
    "onnxruntime-silicon; sys_platform == 'darwin'",
    "numpy",
    "kafka-python",
    "redis",
    "httpx",
    "structlog",
    "pyyaml",
]
```

### Task 3.2：检查 stream-gateway Dockerfile

- [ ] **Step 1: 审查并优化 Go 构建**

```dockerfile
FROM golang:1.26-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /build/stream-gateway ./cmd/main.go

FROM alpine:3.20
RUN apk add --no-cache ffmpeg ca-certificates tzdata
WORKDIR /app
COPY --from=builder /build/stream-gateway .
COPY config.docker.yaml /app/config.yaml
EXPOSE 8080 8081
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s \
  CMD wget -qO- http://localhost:8080/health || exit 1
CMD ["./stream-gateway", "--config", "config.yaml"]
```

**关键检查点：**
- ✅ 多阶段构建（builder → runtime）
- ✅ `go mod download` 单独层（缓存依赖）
- ✅ `CGO_ENABLED=0` 静态编译
- ✅ `-ldflags="-s -w"` 减小体积
- ❌ 缺少 `ffmpeg` 版本锁定

### Task 3.3：检查 dormitory-service-go Dockerfile

- [ ] **Step 1: 审查 Go 构建**

```dockerfile
FROM golang:1.26-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /build/dormitory-service ./cmd/dormitory-service/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /build/dormitory-service .
COPY config.docker.yaml /app/config.yaml
EXPOSE 8083
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s \
  CMD wget -qO- http://localhost:8083/health || exit 1
CMD ["./dormitory-service"]
```

### Task 3.4：统一构建脚本

- [ ] **Step 1: 创建 `scripts/docker/rebuild-all.sh`**

```bash
#!/bin/bash
# CampusVision AI — 一键重建所有服务
# 用法: bash scripts/docker/rebuild-all.sh [--with-models] [--no-cache]

set -euo pipefail

WITH_MODELS=false
NO_CACHE=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --with-models) WITH_MODELS=true ;;
        --no-cache) NO_CACHE="--no-cache" ;;
        *) echo "未知参数: $1"; exit 1 ;;
    esac
    shift
done

BUILD_ARGS=""
if [ -n "${APT_MIRROR:-}" ]; then
    BUILD_ARGS="$BUILD_ARGS --build-arg APT_MIRROR=$APT_MIRROR"
fi
if [ -n "${PIP_INDEX_URL:-}" ]; then
    BUILD_ARGS="$BUILD_ARGS --build-arg PIP_INDEX_URL=$PIP_INDEX_URL"
fi
if [ -n "${HF_ENDPOINT:-}" ]; then
    BUILD_ARGS="$BUILD_ARGS --build-arg HF_ENDPOINT=$HF_ENDPOINT"
fi

if [ "$WITH_MODELS" = true ]; then
    BUILD_ARGS="$BUILD_ARGS --build-arg BUILD_MODELS=1"
fi

echo "=== 构建 stream-gateway ==="
docker compose build $NO_CACHE $BUILD_ARGS stream-gateway

echo "=== 构建 face-recognition ==="
docker compose build $NO_CACHE $BUILD_ARGS face-recognition

echo "=== 构建 dormitory-service-go ==="
docker compose build $NO_CACHE $BUILD_ARGS dormitory-service-go

echo "=== 构建 frontend ==="
docker compose build $NO_CACHE $BUILD_ARGS frontend

echo "=== 完成 ==="
echo "全部服务构建完成。使用 'docker compose up -d' 启动。"
```

- [ ] **Step 2: 创建 `scripts/docker/check-health.sh`**

```bash
#!/bin/bash
# CampusVision AI — 健康检查
set -euo pipefail

echo "=== 容器状态 ==="
docker compose ps --format "table {{.Name}}\t{{.Status}}"

echo ""
echo "=== Kafka Topics ==="
docker compose exec kafka kafka-topics --bootstrap-server localhost:9092 --list 2>/dev/null || echo "Kafka 不可用"

echo ""
echo "=== Redis ==="
docker compose exec redis redis-cli ping 2>/dev/null || echo "Redis 不可用"

echo ""
echo "=== MariaDB ==="
docker compose exec mariadb mysqladmin ping -u sims -psims 2>/dev/null || echo "MariaDB 不可用"
```

---

## 六、Phase 4：RTSP 摄像头模拟

### Task 4.1：Mediamtx 集成

- [ ] **Step 1: 在 docker-compose.yml 中添加 RTSP 模拟服务**

Mediamtx（原 RTSP Simple Server）提供 RTSP 服务，配合 FFmpeg 发布测试视频流。

```yaml
# docker-compose.yml 新增服务
  mediamtx:
    image: bluenviron/mediamtx:latest
    container_name: cv-mediamtx
    networks: [campusvision]
    restart: unless-stopped
    ports:
      - "8554:8554"     # RTSP
      - "1935:1935"     # RTMP
      - "8888:8888"     # HLS
      - "9996:9996"     # WebRTC
    volumes:
      - ./scripts/rtsp/mediamtx.yml:/mediamtx.yml
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:9996/"]
      interval: 10s
      timeout: 3s
```

- [ ] **Step 2: 创建 Mediamtx 配置 `scripts/rtsp/mediamtx.yml`**

```yaml
# Mediamtx 配置 - 用于开发测试
paths:
  cam_a:
    source: publisher
    sourceOnDemand: false
  cam_b:
    source: publisher
    sourceOnDemand: false
  cam_c:
    source: publisher
    sourceOnDemand: false
  cam_d:
    source: publisher
    sourceOnDemand: false

# 允许任何 RTSP 客户端发布
rtspAuthMethods: []

# 日志级别
logLevel: info
```

- [ ] **Step 3: 创建模拟摄像头脚本 `scripts/rtsp/simulate_camera.py`**

```python
#!/usr/bin/env python3
"""使用 Mediamtx + FFmpeg 模拟 RTSP 摄像头推送。

依赖:
  - mediamtx 容器正在运行（docker compose up -d mediamtx）
  - ffmpeg 在 $PATH 中
  - 测试图片（如 face-recognition/tests/fixtures/face_test.png）

用法:
  # 推送单张图片为循环视频流
  python scripts/rtsp/simulate_camera.py \\
      --image face-recognition/tests/fixtures/face_test.png \\
      --rtsp-url rtsp://localhost:8554/cam_a \\
      --fps 5

  # 推送视频文件
  python scripts/rtsp/simulate_camera.py \\
      --video /path/to/test_video.mp4 \\
      --rtsp-url rtsp://localhost:8554/cam_b

  # 推送多摄像头
  python scripts/rtsp/simulate_camera.py --multi 4 \\
      --image face-recognition/tests/fixtures/face_test.png
"""
import argparse
import subprocess
import sys
import shutil


def find_ffmpeg() -> str:
    ffmpeg = shutil.which("ffmpeg")
    if not ffmpeg:
        raise RuntimeError("ffmpeg 未安装，请安装 ffmpeg 并确保在 $PATH 中")
    return ffmpeg


def publish_image(ffmpeg: str, image: str, rtsp_url: str, fps: int):
    """使用图片循环推送 RTSP 流。"""
    cmd = [
        ffmpeg,
        "-re",
        "-loop", "1",
        "-i", image,
        "-c:v", "libx264",
        "-preset", "ultrafast",
        "-tune", "stillimage",
        "-r", str(fps),
        "-pix_fmt", "yuv420p",
        "-b:v", "500k",
        "-maxrate", "500k",
        "-bufsize", "1000k",
        "-f", "rtsp",
        "-rtsp_transport", "tcp",
        rtsp_url,
    ]
    sys.stderr.write(f"推送图片流: {image} → {rtsp_url} @ {fps}fps\n")
    subprocess.run(cmd)


def publish_video(ffmpeg: str, video: str, rtsp_url: str):
    """推送视频文件为 RTSP 流。"""
    cmd = [
        ffmpeg,
        "-re",
        "-i", video,
        "-c:v", "libx264",
        "-preset", "ultrafast",
        "-pix_fmt", "yuv420p",
        "-b:v", "1000k",
        "-maxrate", "1000k",
        "-bufsize", "2000k",
        "-f", "rtsp",
        "-rtsp_transport", "tcp",
        rtsp_url,
    ]
    sys.stderr.write(f"推送视频: {video} → {rtsp_url}\n")
    subprocess.run(cmd)


def main():
    parser = argparse.ArgumentParser(description="RTSP 摄像头模拟")
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument("--image", help="测试图片路径（循环推送）")
    group.add_argument("--video", help="测试视频路径")
    group.add_argument("--multi", type=int, metavar="N",
                       help="同时启动 N 个模拟摄像头")

    parser.add_argument("--rtsp-url", default="rtsp://localhost:8554/cam_a",
                        help="RTSP 推送地址")
    parser.add_argument("--fps", type=int, default=5, help="帧率")
    parser.add_argument("--base-port", type=int, default=8554,
                        help="多摄像头基础端口")

    args = parser.parse_args()
    ffmpeg = find_ffmpeg()

    if args.multi:
        # 多摄像头模式：cam_a, cam_b, cam_c, cam_d
        cameras = [chr(ord("a") + i) for i in range(args.multi)]
        for cam in cameras:
            process = subprocess.Popen([
                sys.executable, __file__,
                "--image", args.image,
                "--rtsp-url", f"rtsp://localhost:{args.base_port}/cam_{cam}",
                "--fps", str(args.fps),
            ])
            sys.stderr.write(f"摄像头 cam_{cam}: PID {process.pid}\n")
    elif args.image:
        publish_image(ffmpeg, args.image, args.rtsp_url, args.fps)
    elif args.video:
        publish_video(ffmpeg, args.video, args.rtsp_url)


if __name__ == "__main__":
    main()
```

- [ ] **Step 4: 验证 RTSP 流可用性**

```bash
# 1. 启动 Mediamtx
docker compose up -d mediamtx

# 2. 推送测试流
python scripts/rtsp/simulate_camera.py \
    --image face-recognition/tests/fixtures/face_test.png \
    --rtsp-url rtsp://localhost:8554/cam_a

# 3. 用 ffplay 验证
ffplay rtsp://localhost:8554/cam_a

# 4. 确认 stream-gateway 可消费
# 检查 stream-gateway 日志中 camera_a 的连接状态
docker compose logs stream-gateway | grep -i "camera\|rtsp\|connected"
```

---

## 七、Phase 5：端到端一键测试

### Task 5.1：创建 `scripts/test/run_pipeline_test.sh`

```bash
#!/bin/bash
# CampusVision AI — 端到端流水线测试
# 验证: Kafka 消息流 → 人脸检测 → 事件产生
set -euo pipefail

PASS=0
FAIL=0

check() {
    local desc="$1"
    local cmd="$2"
    echo -n "[TEST] $desc ... "
    if eval "$cmd" 2>/dev/null; then
        echo "✅ PASS"
        ((PASS++))
    else
        echo "❌ FAIL"
        ((FAIL++))
    fi
}

echo "========================================="
echo " CampusVision AI — Pipeline 测试套件"
echo "========================================="
echo ""

# 1. 基础设施检查
check "Kafka 可达 (本机)" \
    "python -c \"from kafka import KafkaProducer; p=KafkaProducer(bootstrap_servers='localhost:29092'); p.close()\""

check "Redis 可达" \
    "docker compose exec redis redis-cli ping | grep PONG"

check "MariaDB 可达" \
    "docker compose exec mariadb mysqladmin ping -u sims -psims"

# 2. 容器健康检查
check "face-recognition 运行中" \
    "docker compose ps face-recognition --format '{{.Status}}' | grep -q healthy"

check "stream-gateway 运行中" \
    "docker compose ps stream-gateway --format '{{.Status}}' | grep -q healthy"

check "dormitory-service-go 运行中" \
    "docker compose ps dormitory-service-go --format '{{.Status}}' | grep -q healthy"

# 3. Kafka 生产-消费测试
check "注入测试帧" \
    "python scripts/test/kafka_inject_frames.py --image face-recognition/tests/fixtures/face_test.png --count 5 --interval 0.05"

check "face-recognition 处理帧" \
    "sleep 5 && docker compose logs --tail 5 face-recognition 2>&1 | grep -q frames_processed"

# 4. 人脸检测验证
echo ""
echo "--- 人脸检测验证 ---"
# 检查 processing_stats 中 faces_detected > 0
STATS=$(docker compose logs --tail 20 face-recognition 2>&1 | grep processing_stats | tail -1)
echo "  Latest stats: $STATS"

# 5. 结果汇总
echo ""
echo "========================================="
echo " 结果: $PASS 通过, $FAIL 失败"
echo "========================================="

exit $FAIL
```

### Task 5.2：添加 pytest 到 face-recognition 测试套件

- [ ] **Step 1: 验证现有测试**

```bash
cd face-recognition
python -m pytest tests/ -v --tb=short
```

测试覆盖：
- `test_detector.py` — Haar Cascade 检测、模糊过滤
- `test_behavior.py` — 行为分析（逗留、奔跑、区域入侵、人群）
- `test_tracker.py` — 追踪器
- `test_event_publisher.py` — 事件发布
- `test_integration.py` — 集成测试

---

## 八、依赖关系与执行顺序

```
Phase 1: 清理与提交
  └── Task 1.1 清理无用文件
  └── Task 1.2 提交代码变更
         ↓
Phase 2: 测试工具规范化
  └── Task 2.1 创建目录结构
  └── Task 2.2 Kafka注入脚本
  └── Task 2.3 Kafka验证脚本
  └── Task 2.4 容器诊断脚本
         ↓
Phase 3: Docker 构建流程
  └── Task 3.1 face-recognition Dockerfile
  └── Task 3.2 stream-gateway Dockerfile
  └── Task 3.3 dormitory-service Dockerfile
  └── Task 3.4 统一构建脚本
         ↓
Phase 4: RTSP 模拟
  └── Task 4.1 Mediamtx 集成
  └── Task 4.2 模拟摄像头脚本
         ↓
Phase 5: 端到端测试
  └── Task 5.1 一键测试脚本
  └── Task 5.2 pytest 验证
```

---

## 九、未解决问题与后续工作

### 已知阻塞

| 问题 | 影响 | 缓解措施 |
|------|------|---------|
| Docker credential 故障 | 无法 pull 基础镜像 | 使用 `docker login` 或禁用 `credsStore` |
| Windows bind mount 缓存 | 配置文件更新不即时生效 | 改用 `docker compose down && up` 而非 restart |
| 网络受限 | 无法访问 huggingface/LFW | 配置镜像源 `HF_ENDPOINT=https://hf-mirror.com` |
| ONNX 模型检测返回 0 | 人脸检测不可用 | 临时使用 Haar Cascade fallback；待定位根因 |

### 后续优化方向

1. **ONNX 模型诊断**：在 Dockerfile 中添加构建时模型验证步骤
2. **性能基准测试**：测量每帧处理延迟、FPS
3. **golang-migrate 集成**：替代手动 SQL 迁移
4. **CI/CD 集成**：在 GitHub Actions 中运行端到端测试
5. **Mock RTSP 源**：用合成视频测试 stream-gateway 解码
