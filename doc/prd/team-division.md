# CampusVision AI — 双人协作分工与 AI 实现指南

> **版本**: v2.0 · **更新**: 2026-05-16  
> **状态**: 感知层代码已完整实现  
> **场景**: 你（感知层：拉帧 → 解析 → AI 识别 → 发送）  
> **搭档**: 业务层（主服务对接扩展：消费 → 统计 → API → 集成）

---

## 一、总览：整个管道

```
  Camera (4路RTSP)
       │
  ┌────▼──────────────────────────────────────────────────────┐
  │  你 (感知层)                                               │
  │                                                           │
  │  Stream Gateway (Go)          Face Recognition (Python)   │
  │  ┌─────────────┐  Kafka        ┌────────────────────┐     │
  │  │ RTSP拉流     │───────────►  │ 人脸检测            │     │
  │  │ FFmpeg解码   │ t_dorm_frame │ 特征提取            │     │
  │  │ 动态抽帧     │              │ 学管API匹配身份      │     │
  │  │ JPEG编码     │              │ 进出方向判断         │     │
  │  └─────────────┘              │ 异常检测(陌生人等)   │     │
  │                                │ 本地缓存降级         │     │
  │                                └────────┬───────────┘     │
  │                                         │ Kafka            │
  │                                         │ t_dorm_event     │
  └─────────────────────────────────────────┼─────────────────┘
                                            │
  ┌─────────────────────────────────────────▼─────────────────┐
  │  搭档 (业务层)                                             │
  │                                                           │
  │  Dormitory Service (Java Spring Boot JAR)                  │
  │  ┌──────────────┬───────────────┬──────────────┐          │
  │  │ Kafka消费    │ 每晚查宿统计   │ REST API     │          │
  │  │ Redis实时状态│ 告警规则引擎   │ 学管数据同步  │          │
  │  └──────────────┴───────────────┴──────────────┘          │
  │                                                           │
  │  → 未来接入主 SpringBoot 进程                               │
  └───────────────────────────────────────────────────────────┘
```

### 一句话分工

| 角色 | 负责 | 产出的数据 | 交互对象 |
|------|------|-----------|---------|
| **你** | RTSP→帧→AI→Kafka事件 | `t_dorm_event` 进出事件流 | 摄像头、学管API、Kafka |
| **搭档** | 消费事件→状态→统计→API | REST API 的查宿数据 | Kafka、学管API、前端、DB |

---

## 二、你的工作（感知层）详细拆解

### 模块 A：Stream Gateway (Go)

**目录**: `stream-gateway/`  
**语言**: Go (1.22+)  
**唯一依赖**: FFmpeg (解码)、Kafka 客户端库

```
stream-gateway/
├── cmd/
│   └── main.go              # 入口, 启动 4 个 consumer goroutine
├── internal/
│   ├── camera/              # 摄像头管理
│   │   ├── manager.go       # CameraManager: 管理 4 路
│   │   └── stream.go        # CameraStream: 单路 RTSP → 帧通道
│   ├── decoder/             # FFmpeg 解码
│   │   └── ffmpeg.go        # 调用 FFmpeg CGO / exec 解码
│   ├── frame/               # 帧处理
│   │   ├── extractor.go     # 动态抽帧策略 (白天5fps, 夜间1fps)
│   │   └── jpeg.go          # JPEG 编码 + 质量缩放
│   ├── kafka/               # Kafka 推送
│   │   └── producer.go      # 推送 t_dorm_frame
│   ├── health/              # Health API
│   │   └── handler.go       # HTTP /health + /config 端点
│   └── config/              # 配置管理
│       └── config.go        # 从 YAML 加载配置
├── config.yaml              # 默认配置文件
├── go.mod
└── go.sum
```

#### 执行流程（单路 → 4 goroutine 并行）

```
RTSP 地址 → RTSP Dial → Packet → 解码为 YUV → 
  判断是否需要抽帧(动态策略) → 
  YUV → JPEG 编码(指定quality) → 
  Base64 → JSON → Kafka t_dorm_frame
```

#### AI ⚡ 实现指示

**1. 配置文件 (`config.yaml`)**

```yaml
cameras:
  - id: cam-a
    building: A
    rtsp_url: "rtsp://admin:password@192.168.1.101:554/stream1"
    enabled: true
  - id: cam-b
    building: B
    rtsp_url: "rtsp://admin:password@192.168.1.102:554/stream1"
    enabled: true
  - id: cam-c
    building: C
    rtsp_url: "rtsp://admin:password@192.168.1.103:554/stream1"
    enabled: true
  - id: cam-d
    building: D
    rtsp_url: "rtsp://admin:password@192.168.1.104:554/stream1"
    enabled: true

frame:
  fps_day: 5           # 白天 06:00-22:00
  fps_night: 1         # 夜间 22:00-06:00
  jpeg_quality: 80     # 1-100
  width: 1280
  height: 720
  dynamic_extraction: true   # 动态抽帧: 画面无变化则跳帧
  motion_threshold: 0.05     # 动态抽帧灵敏度

kafka:
  brokers: ["kafka:9092"]
  topic: "t_dorm_frame"
  compression: "snappy"
  batch_size: 65536

rtsp:
  reconnect_interval: 5s     # 断线重连间隔
  read_timeout: 10s
  max_reconnect_attempts: 0  # 0=无限重试

log:
  level: "info"
```

**2. CameraManager (`internal/camera/manager.go`)**

关键逻辑：
- 启动时读取配置中的 `cameras` 列表
- 对每个 `enabled: true` 的 camera 启动一个 goroutine 运行 `CameraStream.Run()`
- 所有 camera 共用同一个 Kafka producer 实例
- 提供 `Healthy()` 方法返回各摄像头的连通状态

```go
// CameraManager 伪接口
type CameraManager struct {
    cameras  map[string]*CameraStream
    producer *kafka.Producer
}

func NewManager(cfg *config.Config) *CameraManager
func (m *CameraManager) Start(ctx context.Context)
func (m *CameraManager) Stop()
func (m *CameraManager) Status() map[string]CameraStatus  
// 返回: { cam-a: {connected, fps, last_frame_time, frames_sent} }
```

**3. RTSP 拉流 + 解码 (`internal/camera/stream.go` + `internal/decoder/ffmpeg.go`)**

方案选择（按优先级推荐）：
| 方案 | 优缺点 | 选型建议 |
|------|--------|---------|
| **A. FFmpeg CGO** (github.com/xlab/ffmpeg-go) | 高性能，无进程开销 | ⭐ **推荐**，最稳定 |
| **B. exec ffmpeg pipe** | 简单，但进程管理麻烦 | 备选，调试用 |
| **C. 纯 Go RTSP 库** (bluenviron/mediamtx) | 引入复杂，长期依赖 | 不推荐 |

推荐方案 A 的实现逻辑：

```go
// ffmpeg.go - 核心逻辑
func DecodeFrame(ctx context.Context, rtspURL string) (<-chan []byte, error) {
    // 1. 创建 AVFormatContext, 打开 RTSP 流
    // 2. 找到视频流 index (通常是 h264/h265)
    // 3. 创建 AVCodecContext (h264_cuvid / h264 软件解码)
    // 4. 循环读取 packet:
    //    av_read_frame → avcodec_send_packet → avcodec_receive_frame
    //    → 得到 AVFrame (YUV)
    // 5. 将 YUV 帧发到 channel
    // 6. ctx.Done() 时清理资源
}
```

**关键要点**：
- 解码器优先选硬件加速（`h264_cuvid` / `h264_videotoolbox`），4路软解也够（宿舍入口 720p × 5fps 负载很低）
- macOS 开发可用 `h264_videotoolbox`，生产环境 `h264_cuvid`（NVIDIA）或软解

**4. 动态抽帧 (`internal/frame/extractor.go`)**

```go
// 抽帧策略接口
type Extractor interface {
    ShouldExtract(frame *AVFrame) bool
}

// 时间策略: 根据当前时间决定 fps
type TimeBasedExtractor struct {
    fpsDay   int  // 5fps
    fpsNight int  // 1fps
}

// 动态策略: 时间策略 + 画面变化检测
type DynamicExtractor struct {
    base             TimeBasedExtractor
    motionThreshold  float64
    lastFrame        *image.YCbCr  // 上一帧用于比较
}
```

动态抽帧：计算当前帧与上一帧的像素差异比例，低于 `motion_threshold` 则跳过该帧（画面静止说明没人经过）。目标是白天平均 ~2fps、夜间 ~0.5fps 的有效帧。

**5. JPEG 编码 (`internal/frame/jpeg.go`)**

YUV → JPEG 转换（使用 Go 标准库 `image/jpeg`）：

```go
func EncodeJPEG(yuvData []byte, width, height int, quality int) ([]byte, error) {
    // 1. YUV → RGBA (使用 swscale 或手动转换)
    // 2. image.NewRGBA → jpeg.Encode 指定 quality
    // 3. 返回 JPEG bytes
}
```

**6. Kafka 推送 (`internal/kafka/producer.go`)**

使用 `github.com/segmentio/kafka-go` 或 `confluent-kafka-go`：

```go
type FrameMessage struct {
    CameraID     string `json:"camera_id"`
    Building     string `json:"building"`
    Timestamp    int64  `json:"timestamp"`
    FrameSequence int64 `json:"frame_sequence"`
    FrameData    string `json:"frame_data"`      // base64
    FrameWidth   int    `json:"frame_width"`
    FrameHeight  int    `json:"frame_height"`
    JpegQuality  int    `json:"jpeg_quality"`
    IsDynamic    bool   `json:"is_dynamic"`
}
```

**7. HTTP Health API**

Stream Gateway 额外暴露一个简单的 HTTP 端口（如 `:8080`）供健康检查和配置热更新：

| 端点 | 方法 | 用途 |
|------|------|------|
| `/health` | GET | 存活检查，返回各摄像头状态 |
| `/config` | GET | 查看当前运行配置 |
| `/config` | POST | 动态更新配置（如调整 fps） |

---

### 模块 B：Face Recognition (Python)

**目录**: `face-recognition/`  
**语言**: Python 3.11+  
**关键依赖**: `insightface`(或 `onnxruntime`), `kafka-python`, `httpx`, `redis`, `Pillow`, `numpy`

```
face-recognition/
├── app/
│   ├── __init__.py
│   ├── main.py                # 入口: Kafka 消费者循环（含事件推送）
│   ├── config.py              # 配置加载 (dataclass)
│   ├── detector.py            # 人脸检测 (FaceDetector)
│   ├── feature.py             # 特征提取 (FeatureExtractor)
│   ├── matcher.py             # 学管 API 调用 + Redis 缓存降级 (FaceMatcher)
│   ├── direction.py           # 进出方向判断 ROI 线穿越 (DirectionDetector)
│   ├── dedup.py               # 10s 去重窗口 (DedupFilter)
│   ├── night_mode.py          # 夜间 CLAHE 增强 (NightModeEnhancer)
│   └── models/                # 模型文件
│       ├── detection.onnx     # 人脸检测模型 (RetinaFace)
│       └── feature.onnx       # 特征提取模型 (ArcFace 512-dim)
├── config.yaml                # 所有可配置项
├── requirements.txt
└── Dockerfile
```

#### 执行流程（单线程消费 Kafka → 批量/逐帧处理）

```
消费 t_dorm_frame (Kafka)
  → Base64 解码 → JPEG → numpy array (RGB)
  → 人脸检测 (RetinaFace/yolov8-face)
  → 质量过滤 (模糊度 < 阈值, 人脸大小 > 80px, 角度 < 30°)
  → 特征提取 (ArcFace → 512-dim float32 vector)
  → 去重窗口: 10s 内同一 building 是否已处理过此人
  → 身份匹配:
       A) 优先调学管 API POST /sims/face/match (推荐)
       B) 降级: 本地 SQLite/Redis 缓存特征库
  → 进出方向判断 (ROI 线穿越)
  → 陌生人检测 (匹配置信度 < 阈值)
  → 推送 t_dorm_event (Kafka)
```

#### AI ⚡ 详细实现指示

**1. 配置 (`config.yaml`)**

```yaml
kafka:
  brokers: ["kafka:9092"]
  frame_topic: "t_dorm_frame"
  event_topic: "t_dorm_event"
  group_id: "face-recognition-group"
  max_poll_records: 10

detection:
  model_path: "app/models/detection.onnx"
  confidence_threshold: 0.6     # 检测置信度阈值
  input_size: [640, 640]        # 检测模型输入尺寸
  min_face_size: 80             # 最小人脸像素 (过滤远处小人)

feature:
  model_path: "app/models/feature.onnx"
  embedding_size: 512

match:
  method: "sims_api"             # sims_api | local_cache | fallback
  sims_api_url: "http://sims-backend:8080/sims/face/match"
  sims_api_timeout: 3.0          # 秒
  auth_token: ""                 # Bearer token
  cache_ttl: 3600                # 秒, 本地缓存特征有效时间
  match_threshold: 0.65          # 余弦相似度阈值 (低于此=陌生人)
  fallback_to_cache: true        # API 失败时降级本地缓存

direction:
  method: "roi_line"             # ROI 线穿越法
  roi_line_x: 0.5                # 虚拟线在画面中的水平比例 (0-1)
  min_track_points: 3            # 最少跟踪点数才能判断方向

dedup:
  window_seconds: 10             # 同人同楼栋 10s 内不重复推事件
  max_cache_size: 1000           # 去重缓存最大容量

stranger:
  enabled: true
  alert_threshold: 0.45          # 匹配分低于此 = 陌生人

night_mode:
  enabled: true
  start_hour: 22                 # 22:00 进入夜间模式
  end_hour: 6                    # 06:00 退出
  clahe_clip_limit: 2.0          # 夜间增强参数

log:
  level: "INFO"
```

**2. 人脸检测 (`app/detector.py`)**

推荐模型（按优先级）：
| 模型 | 框架 | 速度 | 精度 | 选型建议 |
|------|------|------|------|---------|
| **RetinaFace (InsightFace)** | ONNX | ★★★★ | ★★★★★ | ⭐ **推荐** 内置80点检测 |
| **YOLOv8n-face** | ONNX | ★★★★★ | ★★★★ | 更轻量, 精度稍低 |
| **MTCNN** | PyTorch | ★★★ | ★★★ | 不推荐, 无关键点 |

```python
# detector.py - 核心逻辑
class FaceDetector:
    def __init__(self, config):
        # 加载 ONNX 模型
        self.session = ort.InferenceSession(model_path)
        self.conf_threshold = config.detection.confidence_threshold
        self.min_face_size = config.detection.min_face_size
        
    def detect(self, image: np.ndarray) -> List[Face]:
        """返回 [Face(x1,y1,x2,y2,confidence,landmarks), ...]"""
        # 1. 预处理: resize → normalize → HWC→CHW → 加 batch
        # 2. 推理: session.run()
        # 3. 后处理: NMS → 坐标映射回原图
        # 4. 过滤 < threshold 和 < min_face_size
        # 5. 返回列表 (一张图可能多人, 但单个宿舍入口通常 1-2 人)
```

**3. 质量过滤（在 `detect()` 后调用）**

```python
def quality_check(face: Face, image: np.ndarray) -> bool:
    """过滤掉模糊、太暗、角度不好的脸"""
    # 1. 拉普拉斯方差 < 阈值 → 模糊
    # 2. 人脸区域平均亮度 < 40 → 太暗
    # 3. 左右眼 y 坐标差 > 30% face_height → 侧脸太大
    # 4. 返回 True/False
```

**4. 特征提取 (`app/feature.py`)**

使用 ArcFace 模型（输出 512-dim L2 归一化向量）：

```python
class FeatureExtractor:
    def __init__(self, config):
        # 加载 feature.onnx (ArcFace)
        self.session = ort.InferenceSession(model_path)
        self.input_size = (112, 112)  # ArcFace 标准输入
        
    def extract(self, image: np.ndarray, face: Face) -> np.ndarray:
        """返回 512-dim float32 特征向量"""
        # 1. 根据 landmarks 做人脸对齐 (仿射变换)
        # 2. 裁剪 112×112 → 归一化
        # 3. 推理 → L2 归一化
        # 4. 返回 shape=(512,) float32
```

**5. 身份匹配 (`app/matcher.py`)**

实际实现 `FaceMatcher`，支持 SIMS API 主路径 + Redis 缓存降级：

```python
class FaceMatcher:
    def __init__(self, config):
        self.api_url = config.sims_api_url
        self.timeout = config.sims_api_timeout
        self.threshold = config.match_threshold
        self._redis = None
        self._init_redis()

    def match(self, embedding: np.ndarray) -> Optional[dict]:
        """返回 { student_id, name, confidence, from_cache } 或 None"""
        # 路径 A: SIMS API
        try:
            result = self._call_sims_api(embedding)
            if result is not None:
                self._cache_result(result["student_id"], embedding, result["name"])
                result["from_cache"] = False
                return result
        except Exception:
            pass  # fall through to cache

        # 路径 B: Redis 缓存降级
        if self.config.fallback_to_cache:
            cached = self._search_cache(embedding)
            if cached is not None:
                cached["from_cache"] = True
                return cached
        return None  # → stranger

    def _call_sims_api(self, embedding: np.ndarray) -> Optional[dict]:
        """POST /sims/face/match
        Request:  { embedding: float[512] }
        Response: { match: bool, student_id, name, confidence }
        """
        resp = httpx.post(
            self.api_url,
            json={"embedding": embedding.tolist()},
            headers={"Authorization": f"Bearer {self.config.auth_token}"},
            timeout=self.config.sims_api_timeout,
        )
        data = resp.json()
        if data.get("match") and data.get("confidence", 0) >= self.config.match_threshold:
            return {"student_id": data["student_id"], "name": data.get("name", ""), "confidence": data["confidence"]}
        return None
```

**6. 进出方向判断 (`app/direction.py`) — ROI 线穿越法**

```
画面视角：摄像头从宿舍门内向外拍摄

        ┌──────────────────────┐
        │                      │
        │      走廊/大厅        │
        │                      │
        ├──────────────────────┤ ←——— ROI 虚拟线 (垂直, x=0.5)
        │                      │
        │      入口区域          │
        │                      │
        └──────────────────────┘

穿越方向判定:
  上一帧人脸中心: (x1, y), 当前帧: (x2, y)
  如果 x1 < ROI_x ≤ x2 → 从外到内 → entry (进楼)
  如果 x2 ≤ ROI_x < x1 → 从内到外 → exit (出楼)
```

```python
class DirectionDetector:
    """ROI 线穿越法。实际实现使用线程安全 tracks + cleanup 机制。"""
    def __init__(self, config):
        self.roi_x_ratio = config.roi_line_x
        self.min_track_points = config.min_track_points
        self._tracks: Dict[str, List[Tuple[float, float, float]]] = {}
        self._lock = threading.Lock()
    
    def determine(self, face_id: str, face_center_x: float, 
                  face_center_y: float, frame_width: int) -> Optional[str]:
        """判断方向: 'entry' | 'exit' | None"""
        roi_x = frame_width * self.roi_x_ratio
        with self._lock:
            if face_id not in self._tracks:
                self._tracks[face_id] = []
            self._tracks[face_id].append((face_center_x, face_center_y, time.time()))
            track = self._tracks[face_id]
            if len(track) < self.min_track_points:
                return None
            first_x = track[0][0]
            last_x = track[-1][0]
            if first_x < roi_x <= last_x:
                return "entry"
            if first_x >= roi_x > last_x:
                return "exit"
        return None
    
    def cleanup(self):
        """移除 5s 无更新的 track"""
        # 线程安全清理逻辑
```

**7. 去重 (`app/dedup.py`)**

```python
class DedupFilter:
    """10s 窗口: (student_id, direction) 去重。线程安全 + LRU 淘汰。"""
    def __init__(self, config):
        self.window = config.window_seconds
        self.max_cache_size = config.max_cache_size
        self._seen: OrderedDict[Tuple[str, str], float] = OrderedDict()
        self._lock = threading.Lock()
    
    def is_duplicate(self, student_id: str, direction: str) -> bool:
        key = (student_id, direction)
        with self._lock:
            ts = self._seen.get(key)
            if ts is not None and time.time() - ts < self.window:
                return True
        return False
    
    def mark_seen(self, student_id: str, direction: str):
        """记录已发送事件（LRU 淘汰）"""
        key = (student_id, direction)
        with self._lock:
            self._seen[key] = time.time()
            self._seen.move_to_end(key)
            while len(self._seen) > self.max_cache_size:
                self._seen.popitem(last=False)
```

**8. Kafka 事件推送 (内联在 `app/main.py`)**

无独立 `producer.py` — producer 逻辑在 `main.py` 中直接内联，通过 `KafkaProducer(value_serializer=json.dumps)` 推送。消息格式（匹配 PRD 协议）：

```python
event = {
    "camera_id": msg["camera_id"],
    "building": msg["building"],
    "event_type": direction_result,      # "entry" | "exit"
    "student_id": match_result["student_id"] if match_result else None,
    "name": match_result["name"] if match_result else None,
    "confidence": match_result["confidence"] if match_result else 0.0,
    "timestamp": int(time.time() * 1000),
    "frame_sequence": msg["frame_sequence"],
    "is_stranger": match_result is None,
    "snapshot_path": "",                  # MinIO 路径
    "direction_method": "roi_line",
}
producer.send(cfg.kafka.event_topic, value=event)
```

**9. 主循环 (`app/main.py`) — 实际实现**

```python
def main():
    cfg = load_config("config.yaml")
    detector = FaceDetector(...)
    extractor = FeatureExtractor(...)
    matcher = FaceMatcher(cfg.match)
    direction = DirectionDetector(cfg.direction)
    dedup = DedupFilter(cfg.dedup)
    enhancer = NightModeEnhancer(cfg.night_mode)  # 夜间增强

    consumer = KafkaConsumer(cfg.kafka.frame_topic, ...)
    producer = KafkaProducer(...)

    # 信号处理: SIGINT/SIGTERM 优雅关闭
    running = True

    while running:
        msg_pack = consumer.poll(timeout_ms=1000)
        for raw_msg in messages:
            frame_data = raw_msg.value
            # base64 → JPEG → numpy (BGR)
            frame = cv2.imdecode(np.frombuffer(base64.b64decode(frame_data["frame_data"]), dtype=np.uint8), ...)
            frame = enhancer.enhance(frame)       # 夜间 CLAHE 增强

            faces = detector.detect(frame)
            for face in faces:
                embedding = extractor.extract(frame, face)
                match_result = matcher.match(embedding)

                face_center_x = (face.x1 + face.x2) / 2.0
                direction_result = direction.determine(face_id, face_center_x, ...)
                if direction_result is None:
                    continue

                if dedup.is_duplicate(student_id, direction_result):
                    continue
                dedup.mark_seen(student_id, direction_result)

                # 构建事件 → Kafka t_dorm_event
                event = { ... }
                producer.send(cfg.kafka.event_topic, value=event)

        # 定期清理 track / dedup 缓存
        direction.cleanup()
        dedup.cleanup()

        # 每 60s 输出统计日志
```

---

## 三、另一位的工作（业务层）拆解 — 仅边界参考

### 模块 C：Dormitory Service (Java Spring Boot JAR) — 已实现

**你的感知层知道这些就够了**：

```
消费 t_dorm_event (Kafka)
  → EventConsumer 更新 Redis 实时状态
     Key: dorm:building:{building}:room:{room} → student status
  → 每晚23:00 NightlyReportTask 统计:
     已归 / 未归 / 晚归 / 陌生人
  → 告警 (DormitoryAlertService):
     陌生人进楼、长时间未归、跨楼栋串门
  → REST API (3 个 Controller):
     CameraController     # 摄像头 CRUD + 状态
     DormitoryRecordController  # 查宿报表 + 明细
     DormitoryAlertController   # 告警列表 + 处理
  → 定时任务:
     NightlyReportTask   # 23:00 查宿
     DataCleanupTask     # 历史数据清理
     HealthCheckTask     # 摄像头健康检查
```

**你需要给他的约定（你的产出边界）**：

| 约定项 | 说明 |
|--------|------|
| **Kafka Topic** | `t_dorm_event`，JSON 格式，分区 Key 为 building |
| **消息字段** | 必须包含 event_id, building, student_id, student_name, event_type, confidence, timestamp_unix_ms |
| **缺失处理** | 当学管API不可用 → event 中 student_id="unknown", is_stranger=true |
| **事件量级** | 白天 ~10 events/min, 夜间 ~2 events/min (4栋楼合计) |
| **异常约定** | 摄像头离线 → Stream Gateway 发 offline 心跳事件 |

---

## 四、双人接口协作时间线

```
你初始化:
   1. 确定 Kafka / Redis / DB 地址 (dev 环境)
   2. 确认 RTSP 地址 (4路)
   3. 确认学管 API 地址 + Token
   4. 测试环境: test-env/server/main.py (模拟4路摄像头+事件注入)
   → 先跑通单路 → 再 4 路并行

你验证通过:
   Stream Gateway 输出 t_dorm_frame ✓
   Face Recognition 输出 t_dorm_event ✓
   → 通知搭档可以开始消费

搭档初始化:
   1. 确认 Kafka / Redis / DB 地址
   2. Consumer t_dorm_event
   3. 确认学管 API (宿舍数据)
   → 跑通事件消费 → 输出 REST API

联调:
   你推真实事件 → 搭档消费 → 前端展示
   使用 test-env Web 面板 http://localhost:8082/ 手动模拟事件
```

---

## 五、感知层完成清单

| 状态 | 模块 | 任务 |
|------|------|------|
| ✅ | **基础架构** | Kafka + Redis + MariaDB + MinIO Docker Compose 编排 |
| ✅ | **基础架构** | Kafka topic 初始化 (t_dorm_frame / t_dorm_event / t_dorm_alert) |
| ✅ | **基础架构** | 数据库初始化 DDL (MariaDB + PostgreSQL 双版本) |
| ✅ | **Stream Gateway** | Go 项目骨架 (cobra 入口 + YAML 配置) |
| ✅ | **Stream Gateway** | CameraManager: 4 路摄像机配置管理与 goroutine 生命周期 |
| ✅ | **Stream Gateway** | CameraStream: 单路 RTSP 拉流 + FFmpeg 解码 |
| ✅ | **Stream Gateway** | 动态抽帧 (时间策略 + 画面变化检测) |
| ✅ | **Stream Gateway** | YUV → JPEG 编码 |
| ✅ | **Stream Gateway** | Kafka 推送 t_dorm_frame (4 路复用一个 producer) |
| ✅ | **Stream Gateway** | HTTP Health API (/health, /config) |
| ✅ | **Face Recognition** | Python 项目骨架 (dataclass 配置 + structlog) |
| ✅ | **Face Recognition** | RetinaFace ONNX 人脸检测 (含 Haar Cascade 降级) |
| ✅ | **Face Recognition** | ArcFace ONNX 特征提取 (512-dim, L2 归一化) |
| ✅ | **Face Recognition** | SIMS API 身份匹配 (FaceMatcher + Redis 缓存降级) |
| ✅ | **Face Recognition** | ROI 线穿越进出方向判断 (DirectionDetector) |
| ✅ | **Face Recognition** | 10s 去重窗口 (DedupFilter, 线程安全 + LRU) |
| ✅ | **Face Recognition** | 夜间 CLAHE 增强 (NightModeEnhancer) |
| ✅ | **Face Recognition** | Kafka 消费 t_dorm_frame + 推送 t_dorm_event |
| ✅ | **Face Recognition** | 优雅关闭 (SIGINT/SIGTERM) + 60s 统计日志 |
| ✅ | **测试环境** | FastAPI 模拟服务器: 4 路摄像头仿真 + Kafka 注入 |
| ✅ | **测试环境** | Web 测试面板 + 随机场景生成 |
| ✅ | **Dormitory Service** | Spring Boot 完整项目: 11 entity → 9 repository → 5 service → 3 controller |
| ✅ | **Dormitory Service** | Kafka 事件消费 → Redis 实时状态 |
| ✅ | **Dormitory Service** | 每晚 23:00 查宿统计 + 数据清理定时任务 |
| ✅ | **Dormitory Service** | 22 个 REST API 端点 |
| ✅ | **Dormitory Service** | 告警机制 (陌生人/长时间未归) |
| ➡️ | **文档** | PRD + 架构 + 设计 + 部署指南 完整编写 |

---

## 六、关键决策清单

| 决策点 | 你的选择 | 理由 |
|--------|---------|------|
| Go RTSP 方案 | FFmpeg CGO (ffmpeg-go) | 最稳定，无进程管理开销 |
| 人脸检测模型 | RetinaFace (ONNX) + Haar 降级 | 精度高，无模型时自动降级 OpenCV Haar |
| 特征模型 | ArcFace (ONNX, 512-dim) + flatten 降级 | 业界标准，无模型时降级像素特征 |
| 方向判断 | ROI 垂直穿越线 + 线程安全 track | 最简单可靠，支持 cleanup 防内存泄漏 |
| 身份匹配 | SIMS API 优先 + Redis 缓存降级 | 数据不重复维护，匹配分缓存扫描 |
| 去重策略 | 10s 时间窗口 (per student+direction) | 线程安全 + LRU 淘汰，防止 OOM |
| 消息格式 | JSON (frame → base64, event → JSON) | 调试友好，量级够用 |
| 配置管理 | YAML 文件 + dataclass 类型校验 | 静态类型安全，可扩展 |
| 日志 | structlog | 结构化日志，适合生产排查 |
| 数据库 | MariaDB (MySQL 兼容) + PostgreSQL 双版本 | `infra/` 提供两套 init SQL，按需选择 |
| 抓拍图存储 | MinIO 对象存储 | 不通过 Kafka 传输大图 |

---

> **你的 AI 工作范围**：Pipeline RTSP → frame → face → feature → match → direction → event  
> **AI 产出的最终数据**：Kafka `t_dorm_event` 中的 `{building, student_id, student_name, event_type, timestamp}`
> 
> **提供给搭档的约定**：只要 `t_dorm_event` topic 格式不变，你内部随便重构。
