# CampusVision AI — 宿舍管理 AI 子服务

> **学生管理系统的宿舍 AI 子服务** — 通过 4 路宿舍入口摄像头，自动识别人员进出，生成每晚查宿统计，替代辅导员人工查寝。

CampusVision AI 是学生管理系统的子服务，覆盖从 RTSP 拉流、人脸识别到查宿统计的全链路。后端独立部署，后期接入主 SpringBoot 进程。前端页面在学管前端项目中统一管理，不存放于本仓库。

---

## 系统架构

```
                    ┌──────────────────────────────┐
                    │    学生管理系统 (学管)         │
                    │  SpringBoot + Vue             │
                    │  API: 人脸匹配 / 学生数据      │
                    └──────────┬───────────────────┘
                               │ REST API
                               │
                    ┌──────────▼───────────────────┐
                    │  Dormitory Service (JAR)     │  ← Java Spring Boot
                    │  • 人员状态管理               │     独立部署，后接入主进程
                    │  • 每晚查宿统计               │
                    │  • 报表/告警                  │
                    │  • 全量配置可动态调整          │
                    └──────────┬───────────────────┘
                               │ Kafka t_dorm_event
                               │
                    ┌──────────▼───────────────────┐
                    │  Face Recognition (Python)   │  ← 独立服务
                    │  • 人脸检测 / 特征提取         │
                    │  • 调学管 API 匹配身份         │
                    │  • 进出方向判断               │
                    │  • 本地缓存降级               │
                    └──────────┬───────────────────┘
                               │ Kafka t_dorm_frame
                               │
                    ┌──────────▼───────────────────┐
                    │  Stream Gateway (Go)         │  ← 独立服务
                    │  • RTSP 拉流 (4路)            │
                    │  • 解码 + 动态抽帧            │
                    │  • JPEG 编码 → Kafka         │
                    └──────────┬───────────────────┘
                               │ RTSP
                               │
                    ┌──────────▼───────────────────┐
                    │  宿舍入口摄像头 × 4           │
                    │  A/B/C/D 栋各 1 路            │
                    └──────────────────────────────┘
```

## 核心能力

### 🏠 自动查宿
- 每晚定时（默认 23:00）自动统计归寝情况
- 按楼栋/楼层/房间多级汇总
- 已归 / 未归 / 晚归 / 陌生人明细
- 替代辅导员上门查寝

### 🚶 进出追踪
- 4 路摄像头覆盖 4 栋宿舍楼入口
- 实时判断人员进出（entry / exit）
- 人脸识别 + 学管 API 身份匹配
- 跨楼栋串门检测

### ⚠️ 异常告警
- 陌生人进入宿舍楼
- 长时间未归（超时可配）
- 跨楼栋非本楼人员进入
- 摄像头离线检测

---

## 技术栈

| 子系统 | 技术 | 部署方式 |
|--------|------|---------|
| **Stream Gateway** | Go, FFmpeg | 独立容器 |
| **Face Recognition** | Python, ONNX/TensorRT, FastAPI | 独立容器 |
| **Dormitory Service** | Spring Boot 3, Java 17/21, MyBatis Plus | 独立 JAR → 接入主进程 |
| **消息队列** | Kafka | 基础服务 |
| **缓存** | Redis | 基础服务 |
| **数据库** | PostgreSQL / MySQL | 基础服务 |

---

## 服务模块

| 模块 | 职责 | 语言 |
|------|------|------|
| `stream-gateway` | 4 路 RTSP 拉流、解码、动态抽帧、Kafka 推送 | Go |
| `face-recognition` | 人脸检测、特征提取、学管 API 身份匹配、进出判断 | Python |
| `dormitory-service` | 状态管理、每晚查宿统计、报表、API 暴露、配置管理 | Java |

---

## 快速开始

### 前置要求

- Docker & Docker Compose
- 4 路 RTSP 摄像头（宿舍入口）

### 启动基础服务

```bash
docker compose up -d kafka redis postgres
```

### 启动 Stream Gateway

```bash
cd stream-gateway
go run cmd/main.go --config config.yaml
```

### 启动 Face Recognition

```bash
cd face-recognition
python app/main.py --config config.yaml
```

### 启动 Dormitory Service

```bash
cd dormitory-service
java -jar target/dormitory-service.jar --spring.profiles.active=prod
```

---

## 开发阶段

| 阶段 | 目标 | 完成内容 |
|------|------|---------|
| **Phase 1 (MVP)** | 核心查宿闭环 | 4路拉流 → 人脸识别 → 进出判断 → 每晚查宿统计 |
| **Phase 2** | 体验完善 | 动态抽帧、本地缓存降级、陌生人告警、历史趋势 |
| **Phase 3** | 系统集成 | 接入主 SpringBoot 进程、统一认证 |

---

## 文档

| 文档 | 说明 |
|------|------|
| [架构设计](doc/main.md) | 系统架构、模块划分、技术选型 |
| [产品需求（PRD）](doc/prd/README.md) | 完整 PRD 文档集（3 个模块） |
| [开发指南](doc/development-guide.md) | 环境搭建、编码规范、Git 工作流 |
| [部署指南](doc/deployment-guide.md) | 硬件要求、Docker Compose、配置说明 |

---

## 与学管系统的关系

| 交互 | 方向 | 说明 |
|------|------|------|
| 人脸匹配 | Face Recognition → 学管 | 检测到人脸后调学管 API 获取身份 |
| 学生数据同步 | Dormitory Service → 学管 | 定时拉取住宿学生信息 |
| 查宿数据 | 学管 → Dormitory Service | 学管前端调本服务 API 展示查宿结果 |

---

## License

MIT
