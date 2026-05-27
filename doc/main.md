# CampusVision AI

## 仓库命名建议

推荐主仓库名：

```text
campusvision-ai
```

含义：

* Campus：校园
* Vision：视觉分析
* AI：智能分析

适合：

* 学校监控
* 人员轨迹
* 人脸识别
* 视频结构化
* AI安防

---

# 一、推荐仓库拆分

推荐采用多仓库模式。

## 1. AI 推理服务

```text
campusvision-ai-engine
```

职责：

* YOLO人体检测
* ByteTrack目标跟踪
* InsightFace人脸识别
* ReID特征提取
* TensorRT推理
* CUDA调度

推荐语言：

```text
Python
```

---

## 2. RTSP 流处理服务

```text
campusvision-stream-gateway
```

职责：

* RTSP拉流
* FFmpeg解码
* 摄像头重连
* 抽帧
* Kafka推送

推荐语言：

```text
Go
```

---

## 3. 业务服务 (Go)

```text
dormitory-service-go
```

职责：

* 事件消费处理
* 学生状态管理
* 每晚查宿统计
* 告警管理
* 报表 API
* 摄像头管理
* 数据统计

推荐技术：

```text
Gin
Vue
MariaDB
Redis
```

---

## 4. 前端系统

```text
campusvision-web
```

职责：

* 实时监控页面
* AI告警页面
* 轨迹查询
* 摄像头地图
* 管理后台

推荐技术：

```text
Vue3
TypeScript
Vite
```

---

# 二、项目整体架构

```text
RTSP Cameras
        ↓
Stream Gateway
        ↓
Kafka / Redis Stream
        ↓
AI Engine
        ↓
Redis / MariaDB
        ↓
Dormitory Service (Go)
        ↓
WebSocket / REST API
        ↓
Vue Web
```

---

# 三、系统核心能力

## 已规划功能

### 人员识别

* 人脸识别
* 人体检测
* ReID追踪
* 陌生人识别
* 黑名单告警

---

### 视频分析

* 实时RTSP分析
* 多摄像头支持
* 抽帧分析
* GPU推理
* 视频结构化

---

### 安防能力

* 区域入侵
* 越界检测
* 聚集检测
* 徘徊检测
* 行为分析

---

### 轨迹系统

* 人员轨迹回放
* 跨摄像头追踪
* 时间轴分析
* 区域热力图

---

# 四、推荐技术栈

## AI Engine

| 模块    | 技术          |
| ----- | ----------- |
| 推理框架  | TensorRT    |
| 人体检测  | YOLOv11     |
| 多目标跟踪 | ByteTrack   |
| 人脸识别  | InsightFace |
| ReID  | FastReID    |
| GPU   | CUDA        |
| 服务框架  | FastAPI     |

---

## Stream Gateway

| 模块   | 技术        |
| ---- | --------- |
| 拉流   | FFmpeg    |
| RTSP | GStreamer |
| 服务语言 | Go        |
| 消息队列 | Kafka     |

---

## Dormitory Service (Go)

| 模块   | 技术        |
| ---- | --------- |
| 后端   | Gin        |
| 数据库  | MariaDB   |
| 缓存   | Redis      |
| 消息   | Kafka      |
| 文件存储 | MinIO     |

---

## Web

| 模块   | 技术           |
| ---- | ------------ |
| 前端框架 | Vue3         |
| UI   | Element Plus |
| 状态管理 | Pinia        |
| 实时通信 | WebSocket    |
| 地图   | OpenLayers   |

---

# 五、目录结构建议

## AI Engine

```text
campusvision-ai-engine/
├── app/
│   ├── api/
│   ├── models/
│   ├── services/
│   ├── trackers/
│   ├── reid/
│   ├── face/
│   ├── stream/
│   └── utils/
├── weights/
├── scripts/
├── docker/
└── requirements.txt
```

---

## Dormitory Service (Go)

```text
dormitory-service-go/
├── cmd/
├── internal/
├── pkg/
└── config.yaml
```

---

# 六、推荐部署架构

## 小型学校（20路摄像头）

```text
1台GPU服务器
1台业务服务器
```

---

## 中型学校（100路摄像头）

```text
RTSP服务器 × 2
GPU推理服务器 × 2
Go业务服务器 × 2
Kafka集群
Redis集群
MariaDB主从
```

---

# 七、Docker Compose 基础规划

```yaml
services:
  nginx:

  kafka:

  redis:

  mariadb:

  minio:

  ai-engine:

  stream-gateway:

  dormitory-service:
```

---

# 八、数据库核心表设计

## 用户表

```text
sys_user
```

---

## 摄像头表

```text
camera_device
```

字段：

* camera_id
* rtsp_url
* building
* floor
* status

---

## 人脸特征表

```text
face_embedding
```

字段：

* user_id
* embedding
* created_at

---

## 轨迹记录表

```text
person_track
```

字段：

* track_id
* camera_id
* timestamp
* image_path
* reid_vector

---

## 告警表

```text
security_alarm
```

字段：

* alarm_type
* level
* camera_id
* snapshot
* created_at

---

# 九、接口设计建议

## AI服务接口

### 人脸注册

```http
POST /api/face/register
```

---

### 人脸识别

```http
POST /api/face/recognize
```

---

### 行人ReID

```http
POST /api/reid/search
```

---

### 摄像头分析

```http
POST /api/stream/analyze
```

---

# 十、开发阶段建议

## 第一阶段

目标：

```text
基础AI识别能力
```

完成：

* RTSP拉流
* YOLO人体检测
* 人脸识别
* Java接入

---

## 第二阶段

目标：

```text
轨迹分析
```

完成：

* ByteTrack
* ReID
* 轨迹查询
* 跨摄像头追踪

---

## 第三阶段

目标：

```text
智能安防
```

完成：

* 聚集检测
* 越界检测
* 陌生人告警
* 黑名单

---

# 十一、推荐编码规范

## Python

* Black
* Ruff
* Pydantic
* AsyncIO

---

## Java

* Go 1.26
* MapStruct
* Lombok

---

## Git Flow

推荐分支：

```text
main
release
dev
feature/*
hotfix/*
```

---

# 十二、推荐硬件配置

## GPU服务器

推荐：

| GPU      | 推荐摄像头数量 |
| -------- | ------- |
| RTX 4070 | 20 路左右  |
| RTX 4090 | 40 路左右  |
| L40S     | 60+ 路   |
| A100     | 80+ 路   |

---

## 摄像头建议

推荐：

* 1080P
* H.264
* RTSP
* 红外夜视
* 宽动态

不建议：

* 低码率模糊摄像头
* 非RTSP协议

---

# 十三、安全建议

必须：

* HTTPS
* 摄像头账号隔离
* 内网部署
* MinIO权限控制
* GPU服务器隔离
* Kafka ACL

---

# 十四、未来扩展方向

后期可扩展：

* 车辆识别
* OCR
* 工牌识别
* 姿态识别
* 课堂行为分析
* 边缘计算盒子
* 多GPU调度
* Kubernetes

---

# 十五、最终推荐

当前阶段推荐优先完成：

```text
RTSP拉流
+ YOLO人体检测
+ 人脸识别
+ Java联动
+ WebSocket实时展示
```

先形成完整闭环，再逐步增加：

```text
ReID
轨迹系统
智能告警
```

