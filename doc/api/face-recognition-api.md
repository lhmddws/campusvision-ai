# Face Recognition Module — API Documentation

> **模块位置**: `face-recognition/`
> **入口**: `python -m app.main --config config.yaml`
> **语言**: Python 3.11+

---

## 目录

1. [Class API](#1-class-api)
2. [Usage Examples](#2-usage-examples)
3. [Kafka Message API](#3-kafka-message-api)
4. [Planned Components](#4-planned-components)
5. [JSON Spec](#5-json-spec)

---

## 1. Class API

### 1.1 `Face` — 检测结果数据类

`app/detector.py`

| 字段 | 类型 | 说明 |
|------|------|------|
| `x1` | `float` | 边界框左 |
| `y1` | `float` | 边界框上 |
| `x2` | `float` | 边界框右 |
| `y2` | `float` | 边界框下 |
| `confidence` | `float` | 检测置信度 [0, 1] |
| `landmarks` | `list\|None` | 5 个关键点 `[(x,y), ...]` 或 None |

关键点顺序: 左眼 → 右眼 → 鼻尖 → 左嘴角 → 右嘴角

**使用**:
```python
face = Face(100, 100, 200, 200, 0.95,
            landmarks=[(150,150), (180,150), (165,180), (140,210), (190,210)])
print(face.x1, face.y1, face.x2, face.y2)  # 100 100 200 200
```

---

### 1.2 `FaceDetector` — 人脸检测器

`app/detector.py`

```python
detector = FaceDetector(
    model_path="app/models/retinaface-R50.onnx",  # None → Haar Cascade
    conf_threshold=0.6,                           # 置信度阈值
    input_size=(640, 640),                        # 模型输入尺寸
    min_face_size=80,                             # 最小人脸(像素)
)

faces = detector.detect(image: np.ndarray) -> list[Face]
```

**流程**:
- ONNX session 存在 → RetinaFace 推理 + NMS + 质量过滤
- 无 ONNX → Haar Cascade 回退 (`haarcascade_frontalface_default.xml`)

**使用**:
```python
import cv2
from app.detector import FaceDetector

d = FaceDetector("app/models/retinaface-R50.onnx", 0.6, (640, 640), 80)
img = cv2.imread("photo.jpg")
faces = d.detect(img)
for f in faces:
    print(f"({f.x1:.0f},{f.y1:.0f})-({f.x2:.0f},{f.y2:.0f}) conf={f.confidence:.2f}")
```

---

### 1.3 `FeatureExtractor` — 特征提取器

`app/feature.py`

```python
extractor = FeatureExtractor(
    model_path="app/models/arcface-resnet100.onnx",  # None → flatten 回退
    embedding_size=512,
)

embedding = extractor.extract(image: np.ndarray, face: Face) -> np.ndarray  # shape (512,)
```

**流程**:
1. 人脸对齐: 有 landmarks → 5 点仿射变换 (112×112 ArcFace 目标); 无 landmarks → crop+resize
2. ArcFace ONNX 推理 → 512-dim 向量
3. L2 归一化

**ArcFace 对齐目标关键点**:
```python
LEFT_EYE   = (38.2946, 51.6963)
RIGHT_EYE  = (73.5318, 51.5014)
NOSE       = (56.0252, 71.7366)
LEFT_MOUTH = (41.5493, 92.3655)
RIGHT_MOUTH= (70.7299, 92.2041)
```

**使用**:
```python
from app.feature import FeatureExtractor
e = FeatureExtractor("app/models/arcface-resnet100.onnx", 512)
emb = e.extract(img, faces[0])
print(emb.shape)          # (512,)
print(np.linalg.norm(emb))  # ≈ 1.0
```

---

### 1.4 `FaceMatcher` — 身份匹配

`app/matcher.py`

```python
matcher = FaceMatcher(config: MatchConfig)

result = matcher.match(embedding: np.ndarray) -> dict | None
```

**返回** (匹配时):
```python
{
    "student_id": "2024001",
    "name": "张三",
    "confidence": 0.92,
    "from_cache": False,   # True = Redis 缓存命中
}
```

**流程**:
1. POST embedding 到 SIMS API (`/sims/face/match`)
2. 成功 → Redis 缓存, 返回结果
3. API 故障 → Redis 缓存余弦相似度扫描
4. 均失败 → None

**SIMS API 请求格式**:
```http
POST /sims/face/match
Authorization: Bearer {auth_token}
Content-Type: application/json

{"embedding": [0.01, 0.02, ...]}  # 512-dim

# 响应:
{"match": true, "student_id": "2024001", "name": "张三", "confidence": 0.92}
```

**使用**:
```python
from app.matcher import FaceMatcher
from app.config import MatchConfig

cfg = MatchConfig(sims_api_url="http://192.168.113.114:8080/sims/face/match",
                  match_threshold=0.65)
m = FaceMatcher(cfg)
result = m.match(emb)
if result:
    print(f"{result['name']} ({result['student_id']}) conf={result['confidence']}")
```

**工具函数**:
```python
from app.matcher import cosine_similarity
sim = cosine_similarity(emb1, emb2)  # → float in [-1, 1]
```

---

### 1.5 `DirectionDetector` — 进出方向判定

`app/direction.py`

```python
detector = DirectionDetector(config: DirectionConfig)

result = detector.determine(
    face_id: str,          # 轨迹 ID
    face_center_x: float,  # 人脸中心 X
    face_center_y: float,  # 人脸中心 Y
    frame_width: int,      # 帧宽度
) -> str | None            # "entry" | "exit" | None

detector.cleanup()
```

**算法**: ROI 线 (位于 `frame_width * roi_line_x`) 穿越判定:
- 轨迹从左→右穿线 → `"entry"` (进楼)
- 轨迹从右→左穿线 → `"exit"` (出楼)
- 轨迹点 < `min_track_points` → None

**使用**:
```python
from app.direction import DirectionDetector
from app.config import DirectionConfig

d = DirectionDetector(DirectionConfig(roi_line_x=0.5, min_track_points=3))
result = d.determine("cam-A-1", 320.0, 240.0, 640)
# → "entry" 或 "exit" 或 None
```

---

### 1.6 `DedupFilter` — 事件去重

`app/dedup.py`

```python
dedup = DedupFilter(config: DedupConfig)

dedup.is_duplicate(student_id: str, direction: str) -> bool
dedup.mark_seen(student_id: str, direction: str)
dedup.cleanup()
```

去重键: `(student_id, direction)`, 窗口 `window_seconds` 秒内抑制重复。

**使用**:
```python
from app.dedup import DedupFilter
d = DedupFilter(DedupConfig(window_seconds=10, max_cache_size=1000))

if not d.is_duplicate("2024001", "entry"):
    d.mark_seen("2024001", "entry")
    # emit event
```

---

### 1.7 `NightModeEnhancer` — 低光照增强

`app/night_mode.py`

```python
enhancer = NightModeEnhancer(config: NightModeConfig)

frame = enhancer.enhance(frame: np.ndarray) -> np.ndarray  # BGR → BGR
enhancer.is_night() -> bool
```

**算法**: 仅在夜间窗口 (`[start_hour, end_hour)`) 且 enabled 时:
- BGR → LAB → L 通道 CLAHE → LAB → BGR

**使用**:
```python
from app.night_mode import NightModeEnhancer
e = NightModeEnhancer(NightModeConfig(enabled=True, start_hour=22, end_hour=6))
enhanced = e.enhance(frame)
```

---

### 1.8 Kafka Pipeline (`app/main.py`)

```bash
python -m app.main --config config.yaml
```

**处理流程**:
```
t_dorm_frame (Kafka) → decode JPEG → enhance → detect faces
  → [per face] extract → match → determine direction → dedup
  → t_dorm_event (Kafka)
```

**组件初始化顺序**:
```python
cfg = load_config("config.yaml")

detector   = FaceDetector(cfg.detection.model_path, ...)
extractor  = FeatureExtractor(cfg.feature.model_path, ...)
matcher    = FaceMatcher(cfg.match)
direction  = DirectionDetector(cfg.direction)
dedup      = DedupFilter(cfg.dedup)
enhancer   = NightModeEnhancer(cfg.night_mode)

consumer   = KafkaConsumer(cfg.kafka.frame_topic, ...)
producer   = KafkaProducer(bootstrap_servers=cfg.kafka.brokers, ...)
```

---

## 2. Usage Examples

### 2.1 完整独立检测链路 (无 Kafka)

```python
"""Detect → extract → match. No Kafka dependency."""
import cv2
from app.config import load_config
from app.detector import FaceDetector
from app.feature import FeatureExtractor
from app.matcher import FaceMatcher

cfg = load_config("config.yaml")
detector  = FaceDetector(cfg.detection.model_path, cfg.detection.confidence_threshold,
                         tuple(cfg.detection.input_size), cfg.detection.min_face_size)
extractor = FeatureExtractor(cfg.feature.model_path, cfg.feature.embedding_size)
matcher   = FaceMatcher(cfg.match)

img = cv2.imread("test.jpg")
for face in detector.detect(img):
    emb = extractor.extract(img, face)
    result = matcher.match(emb)
    print(f"{result['name']}" if result else "Unknown")
```

### 2.2 ROI 线方向跟踪模拟

```python
"""Simulate a person crossing the ROI line."""
from app.direction import DirectionDetector
from app.config import DirectionConfig

d = DirectionDetector(DirectionConfig(roi_line_x=0.5, min_track_points=3))
for x in [200, 300, 400, 500]:
    result = d.determine("t1", x, 240, 640)
    print(f"x={x}: {result}")  # x=200/300 → None, x=400 → "entry"
```

### 2.3 Kafka 消息查看

```bash
# 查看输入帧
docker compose exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 --topic t_dorm_frame \
  --max-messages 1 --timeout-ms 3000 | python -m json.tool

# 查看输出事件
docker compose exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 --topic t_dorm_event \
  --from-beginning --max-messages 5 | python -m json.tool
```

---

## 3. Kafka Message API

### 3.1 输入: `t_dorm_frame`

| 字段 | 类型 | 必须 | 说明 |
|------|------|------|------|
| `camera_id` | `string` | ✅ | 摄像头: `A`/`B`/`C`/`D` |
| `building` | `string` | ✅ | 楼栋 |
| `frame_sequence` | `int` | ✅ | 递增序号 |
| `frame_data` | `string` | ✅ | JPEG base64 |
| `frame_width` | `int` | ✅ | 原始宽 |
| `frame_height` | `int` | ✅ | 原始高 |
| `timestamp` | `int` | ✅ | Unix ms |

### 3.2 输出: `t_dorm_event` — Entry/Exit

| 字段 | 类型 | 必须 | 说明 |
|------|------|------|------|
| `camera_id` | `string` | ✅ | |
| `building` | `string` | ✅ | |
| `event_type` | `string` | ✅ | `"entry"` / `"exit"` |
| `student_id` | `string\|null` | ✅ | 匹配学号 / null |
| `name` | `string\|null` | ❌ | 姓名 |
| `confidence` | `float` | ✅ | [0, 1] |
| `timestamp` | `int` | ✅ | Unix ms |
| `frame_sequence` | `int` | ❌ | |
| `is_stranger` | `bool` | ❌ | |
| `snapshot_path` | `string` | ❌ | 预留 |
| `direction_method` | `string` | ❌ | `"roi_line"` |

### 3.3 输出: `t_dorm_event` — Behavior (计划)

| 字段 | 类型 | 必须 | 说明 |
|------|------|------|------|
| `camera_id` | `string` | ✅ | |
| `event_type` | `string` | ✅ | `"loiter"`/`"running"`/`"zone"`/`"crowd"` |
| `track_id` | `string` | ✅ | `cam-{id}-{seq}` |
| `detail` | `string` | ❌ | 描述 |
| `confidence` | `float` | ✅ | 恒为 1.0 |
| `timestamp` | `float` | ✅ | Unix s |
| `source` | `string` | ✅ | 固定 `"behavior"` |

**event_type 映射**:
| 行为 | 映射值 | 原长 | 映射后 |
|------|--------|------|--------|
| Loitering | `"loiter"` | 9 | 6 ✅ |
| Running | `"running"` | 7 | 7 ✅ |
| Zone Intrusion | `"zone"` | 15 | 4 ✅ |
| Crowd Alert | `"crowd"` | 11 | 5 ✅ |

> DB `event_type VARCHAR(8)` 约束全部满足。

---

## 4. Planned Components

### 4.1 `Track` — 轨迹数据类 (计划)

`app/tracker.py`

| 字段 | 类型 | 说明 |
|------|------|------|
| `track_id` | `str` | `cam-{camera_id}-{seq}` |
| `bbox` | `tuple` | (x1,y1,x2,y2) |
| `center` | `tuple` | (cx,cy) |
| `timestamps` | `list[float]` | 历史时间戳 |
| `positions` | `list[tuple]` | 历史位置 |
| `embedding` | `np.ndarray\|None` | 特征向量 |
| `face` | `Face\|None` | 最新检测 |
| `last_seen` | `float` | 最后时间 |
| `frames_alive` | `int` | 存活帧数 |

### 4.2 `FaceTracker` (计划)

`app/tracker.py`

```python
tracker = FaceTracker(iou_threshold=0.3, max_tracks=100, track_ttl=5.0)

tracks = tracker.update(
    faces: list[Face],
    embeddings: list[np.ndarray],
    camera_id: str,
    timestamp: float,
) -> list[Track]

tracker.get_tracks() -> list[Track]
tracker.cleanup()
```

**算法**: IoU 矩阵 → 贪心匹配 (置信度排序, 最佳 IoU 分配) → 更新/创建/移除

### 4.3 `BehaviorAnalyzer` (计划)

`app/behavior.py`

```python
analyzer = BehaviorAnalyzer(config: BehaviorConfig)

events = analyzer.analyze(
    tracks: list[Track],
    frame_face_count: int,
    timestamp: float,
) -> list[dict]
# [{"event_type", "camera_id", "track_id", "detail", "timestamp", "confidence"}]
```

**检测项**:
| 方法 | 触发条件 | event_type |
|------|---------|-----------|
| `_check_loitering` | 存活 > 阈值 AND 位移 < 半径 | `"loiter"` |
| `_check_running` | 速度 > 阈值 | `"running"` |
| `_check_zone_intrusion` | 中心在多边形内 | `"zone"` |
| `_check_crowd` | 人脸数 > 阈值 持续 N 帧 | `"crowd"` |

### 4.4 `BehaviorEventPublisher` (计划)

`app/event_publisher.py`

```python
publisher = BehaviorEventPublisher(config: AppConfig)

publisher.publish_behavior_event(event: dict)
# 自动添加 source="behavior", 断言 event_type ≤ 8 字符
# Kafka 不可用 → 缓冲 (max 1000)

publisher.close()
```

### 4.5 模型下载 (计划)

```bash
python -m app.download_models
```

从 `app/models/model_urls.yaml` 下载 + SHA256 验证到 `app/models/`.

---

## 5. JSON Spec

完整 API JSON Spec 见: [`face-recognition-api.json`](face-recognition-api.json)

包含:
- 全部 12 个类的 API 签名
- 方法参数、返回值、字段定义
- Kafka 消息合约
- 可提取用于代码生成或文档渲染

---

> **See also**:
> - [Parameter reference](../../.sisyphus/drafts/face-recognition-reference.md) — config.yaml + dataclass fields
> - [Implementation plan](../../.sisyphus/plans/face-behavior-recognition.md)
> - [JSON API spec](face-recognition-api.json)
