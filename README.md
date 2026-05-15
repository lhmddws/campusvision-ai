# CampusVision AI

> 校园视觉 AI 分析平台 — 基于深度学习的实时视频监控与智能分析系统

CampusVision AI 是一套面向校园安防场景的 AI 视觉分析平台，覆盖视频流接入、AI 推理分析、数据存储、业务管理和前端可视化全链路。提供人员识别、轨迹追踪、智能告警等核心能力。

---

## 系统架构

```
┌──────────────┐   RTSP
│  摄像头设备    │──────────────┐
│  (1080P H.264)│              │
└──────────────┘              ▼
                    ┌──────────────────┐
                    │  Stream Gateway  │  ← Go / FFmpeg
                    │  RTSP 拉流       │
                    │  抽帧 · 重连     │
                    └──────┬───────────┘
                           │ Frames
                    ┌──────▼───────────┐
                    │   Kafka / Redis  │
                    │   Stream         │
                    └──────┬───────────┘
                           │
                    ┌──────▼───────────┐
                    │   AI Engine      │  ← Python / TensorRT
                    │   YOLO · Face    │
                    │   ReID · Track   │
                    └──────┬───────────┘
                           │ Features / Vectors
                    ┌──────▼───────────┐
                    │   Data Layer     │
                    │  PostgreSQL      │
                    │  Redis · Milvus  │
                    │  MinIO           │
                    └──────┬───────────┘
                           │
                    ┌──────▼───────────┐
                    │   Java Platform  │  ← Spring Boot
                    │   业务 · 告警    │
                    │   WebSocket      │
                    └──────┬───────────┘
                           │ REST / WS
                    ┌──────▼───────────┐
                    │   Vue3 Frontend  │  ← Web 管理端
                    │   实时监控 · 轨迹 │
                    │   告警 · 管理    │
                    └──────────────────┘
```

## 核心能力

### 👤 人员识别
- 人脸识别 (InsightFace)
- 人体检测 (YOLOv11)
- ReID 跨摄像头追踪
- 陌生人识别 & 黑名单告警

### 🎥 视频分析
- 多路 RTSP 实时分析
- GPU 硬件加速推理
- 智能抽帧与结构化

### 🚨 智能安防
- 区域入侵检测
- 越界检测
- 人员聚集检测
- 徘徊检测

### 🗺️ 轨迹系统
- 人员轨迹回放
- 跨摄像头连续追踪
- 时间轴分析
- 区域热力图

---

## 技术栈

| 子系统 | 技术 |
|--------|------|
| **AI Engine** | Python, TensorRT, YOLOv11, ByteTrack, InsightFace, FastReID, FastAPI |
| **Stream Gateway** | Go, FFmpeg, GStreamer, Kafka |
| **Platform** | Spring Boot 3, Java 21, MyBatis Plus, Redis, PostgreSQL, Milvus, MinIO |
| **Web** | Vue 3, TypeScript, Vite, Element Plus, Pinia, WebSocket, OpenLayers |

---

## 子仓库

| 仓库 | 职责 |
|------|------|
| `campusvision-ai-engine` | AI 推理服务：YOLO 检测、人脸识别、ReID、跟踪 |
| `campusvision-stream-gateway` | RTSP 流处理：拉流、FFmpeg 解码、抽帧推送 |
| `campusvision-platform` | Java 业务系统：用户、摄像头、告警、轨迹 |
| `campusvision-web` | 前端系统：监控页面、管理后台、地图 |

---

## 快速开始

### 前置要求

- Docker & Docker Compose
- NVIDIA GPU + CUDA (推理节点)
- NVIDIA Container Toolkit (Docker GPU 支持)

### 基础依赖

```bash
# 基础服务
docker compose up -d kafka redis postgres milvus minio nginx
```

### 启动 Stream Gateway

```bash
# campusvision-stream-gateway
go run cmd/gateway/main.go --config config.yaml
```

### 启动 AI Engine

```bash
# campusvision-ai-engine
python app/main.py --config config.yaml
```

### 启动 Platform

```bash
# campusvision-platform
java -jar target/campusvision-platform.jar
```

### 启动 Web

```bash
# campusvision-web
npm install && npm run dev
```

---

## 硬件建议

| GPU | 推荐摄像头数量 |
|-----|---------------|
| RTX 4070 | ~20 路 |
| RTX 4090 | ~40 路 |
| L40S | ~60+ 路 |
| A100 | ~80+ 路 |

> 摄像头：1080P / H.264 / RTSP / 红外夜视

---

## 开发阶段

| 阶段 | 目标 | 完成内容 |
|------|------|---------|
| **Phase 1** | 基础 AI 识别 | RTSP 拉流 → YOLO 检测 → 人脸识别 → Java 接入 |
| **Phase 2** | 轨迹分析 | ByteTrack → ReID → 轨迹查询 → 跨摄像头追踪 |
| **Phase 3** | 智能安防 | 聚集检测 → 越界检测 → 陌生人告警 → 黑名单 |

---

## 文档

| 文档 | 说明 |
|------|------|
| [架构设计](doc/main.md) | 系统架构、模块划分、技术选型 |
| [开发指南](doc/development-guide.md) | 环境搭建、编码规范、Git 工作流 |
| [部署指南](doc/deployment-guide.md) | 硬件要求、Docker Compose、配置说明 |

---

## 安全

- 全链路 HTTPS
- 摄像头账号隔离
- 内网部署
- MinIO 权限控制
- GPU 服务器网络隔离
- Kafka ACL

## License

MIT
