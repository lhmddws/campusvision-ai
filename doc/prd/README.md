# CampusVision AI — 产品需求文档 (PRD)

> **总版本**: v3.2 · **最后更新**: 2026-05-18 · **状态**: 实施完成  
> **范围说明**: 本 PRD 基于实际需求重新聚焦，定位为**学生管理系统的宿舍管理 AI 子服务**，非通用安防平台。摄像头管理已升级为可扩展管理平台，支持动态注册和加密凭据存储。
>
> **双人协作映射**: 感知层（PRD-001/002）←你→ 业务层（PRD-003/004/005）←搭档→

---

## 文档索引

| 编号 | 模块 | 文件 | 行数 | 核心内容 | 负责人 |
|------|------|------|------|---------|--------|
| PRD-001 | **Stream Gateway** | [01-stream-gateway.md](01-stream-gateway.md) | 840 | RTSP 拉流 → 解码 → 动态抽帧 → Kafka 推送 | **你** |
| PRD-002 | **Face Recognition** | [02-face-recognition.md](02-face-recognition.md) | 1,255 | 人脸检测 → 特征提取 → 学管 API 匹配 → 进出判断 | **你** |
| PRD-003 | **Dormitory Service** | [03-dormitory-service.md](03-dormitory-service.md) | 1,927 | 状态管理 → 每晚查宿统计 → 报表 → API 暴露 | **搭档**（完整版） |
| PRD-004 | **主进程对接** | [04-main-process-integration.md](04-main-process-integration.md) | 815 | 业务核心：事件消费→状态→查宿→API→接入主进程 | **搭档** |
| PRD-005 | **摄像头功能实现** | [05-camera-management.md](05-camera-management.md) | 618 | 摄像头设备管理、状态监控、配置管理、抓拍查看 | **搭档** |
| **合计** | | | **5,455** | | |

> **附加**: 另有技术设计文档 3,754 行（架构/数据库/API/集成/摄像头），测试环境 352 行（模拟服务器 + 启动脚本 + 前端）。

---

## 系统定位

```
学生管理系统 (SpringBoot + Vue)
      │ 主进程
      │
      ├── 学管核心服务 (同步开发中)
      │    ├── 学生管理、班级管理、成绩等
      │    └── API: /sims/face/match, /sims/students/dormitory
      │
      └── Dormitory AI 子服务 ← 本仓库（双人协作）
            │
            ├─── 你（感知层）─────────────── 搭档（业务层）───┐
            │                                              │
            ├ 01 Stream Gateway (Go)         ├ 03 Dormitory Service (Java JAR)
            ├ 02 Face Recognition (Python)   ├ 04 主进程对接（聚焦版）
            │                                ├ 05 摄像头功能实现
            │           Kafka 桥接            │
            │  t_dorm_frame → t_dorm_event ──►  事件消费→状态→查宿→API
```

**核心业务目标**: 每晚自动查宿，统计学生归寝情况，替代辅导员人工查寝。  
**协作模式**: 感知层 Kafka 产出的 `t_dorm_event` 是双方唯一耦合点，两侧独立开发。

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
                    │  • RTSP 拉流 (N路, 可扩展)    │
                    │  • 解码 + 动态抽帧            │
                    │  • JPEG 编码 → Kafka         │
                    │  • 管理 API (摄像头动态注册)   │
                    └──────────┬───────────────────┘
                               │ RTSP
                               │
                    ┌──────────▼───────────────────┐
                    │  宿舍入口摄像头 (可扩展)       │
                    │  通过管理平台动态注册           │
                    └──────────────────────────────┘
```

---

## 核心数据流

```
Camera (N路, 可扩展)
  │ RTSP
  ▼
Stream Gateway (Go)
  │ 解码 → 动态抽帧 → JPEG → Kafka t_dorm_frame
  │ { camera_id, building, timestamp, frame_data }
  ▼
Face Recognition (Python)
  │ 人脸检测 → 质量过滤 → 特征提取
  │ → 调学管 API: POST /api/sims/face/match
  │ → 返回 student_id, student_name, dorm_info
  │ → 进出方向判断（ROI 线穿越法）
  │ → Kafka t_dorm_event
  │ { event_id, building, student_id, student_name,
  │   event_type: "entry"|"exit", confidence, timestamp }
  ▼
Dormitory Service (Java JAR)
  │ 消费事件 → 更新 Redis 实时状态
  │ 每日 23:00 触发查宿统计
  │ 按楼栋/房间汇总 → 存 PostgreSQL
  │ 提供 REST API 供前端/学管调用
  ▼
前端页面 (不存本仓库)
  │ 查宿概览 → 明细 → 趋势
  │ 辅导员查看今日未归学生
```

---

## 关键设计决策

| 决策 | 选择 | 理由 |
|------|------|------|
| 摄像头数量 | **可扩展（动态注册）** | 支持通过管理平台动态增删摄像头，按需注册 |
| 帧率 | **5 fps** | 宿舍入口步行速度足够，降低负载 |
| 进出判断 | **ROI 线穿越法** | 入口单一方向，准确率 ≥ 98% |
| 身份匹配 | **学管 API 按需查询** | 学管维护人脸库，本服务不做存储 |
| 模块间通信 | **Kafka** | 异步解耦，独立扩缩容，语言无关 |
| 实时状态 | **Redis** | 进出事件频繁，Redis 适合实时状态 |
| 查宿数据持久化 | **MariaDB/PostgreSQL** | 与学管数据库一致，统一管理；`infra/` 提供双版本 init SQL |
| Java JAR 部署 | **独立 jar → 接入主进程** | 先独立运行稳定再集成 |
| 配置管理 | **全部可动态调整** | 查宿时间、阈值等无需重启即可修改 |
| 面部抓拍存储 | **MinIO** | 人脸抓拍图存对象存储，不在 Kafka 中传输大图 |
| 前端 | **不存本仓库** | 前端页面在学管前端项目中统一管理 |

---

## 功能优先级

### Phase 1 (MVP) — 核心闭环

| 模块 | 功能 | 优先级 |
|------|------|--------|
| Stream Gateway | 多路 RTSP 拉流 + 解码 + 抽帧推送（可扩展） | P0 |
| Face Recognition | 人脸检测 + 学管 API 匹配 + 进出判断 | P0 |
| Dormitory Service | 事件消费 + 状态管理 + 每晚查宿统计 | P0 |
| Dormitory Service | 查宿报表 API + 异常告警 | P0 |
| Infrastructure | Kafka + Redis + 数据库部署 | P0 |

### Phase 2 — 体验完善

| 模块 | 功能 | 优先级 |
|------|------|--------|
| Stream Gateway | 动态抽帧优化、夜间模式 | P1 |
| Face Recognition | 本地特征缓存降级、陌生人告警 | P1 |
| Dormitory Service | 历史趋势统计、跨楼栋告警 | P1 |
| Dormitory Service | 导出报表 CSV/Excel | P1 |

### Phase 3 — 增强

| 模块 | 功能 | 优先级 |
|------|------|--------|
| Dormitory Service | 接入主进程、统一认证 | P2 |
| All | 性能优化、监控完善 | P2 |

---

## 各模块间契约

### Kafka Topic

| Topic | Producer | Consumer | 消息体 |
|-------|----------|----------|--------|
| `t_dorm_frame` | Stream Gateway | Face Recognition | `{ camera_id, building, timestamp, frame_data(jpeg) }` |
| `t_dorm_event` | Face Recognition | Dormitory Service | `{ event_id, building, student_id, student_name, event_type, confidence, face_snapshot, timestamp }` |
| `t_dorm_alert` | Dormitory Service | — | 告警消息（陌生人进楼、长时间未归等） |

### 学管 API 调用

> ⚠️ **实际情况核查**：以下接口在学管 OpenAPI 中**不存在**，需在学管同步开发中新增。  
> 开发/测试阶段可使用 `test-env` 模拟服务器。

```
Face Recognition ──POST /sims/face/match (待新增)──→ 学管系统
  Request:  { embedding: float[512] }
  Response: { match: bool, student_id, name, confidence }
  ↑ 测试替代: test-env 未实现此端点，需配合本地 Redis 缓存降级

Dormitory Service ──GET /sims/students/dormitory (待新增)──→ 学管系统
  Response: [{ student_id, student_name, building, room, class }]
```

### 学管已存在可复用的 API

| 学管现有接口 | 方法 | 对本服务的用途 |
|-------------|------|--------------|
| `/sims/student/get-list` | GET | 获取学生列表，可作为宿舍数据同步的兜底（但缺少楼栋/房间字段） |
| `/sims/class/studentMessage` | GET | 根据班级获取学生学号+姓名 |
| `/sims/class/get-all` | GET | 获取所有班级列表 |
| `/sims/grader/info/all` | GET | 获取所有年级列表 |
| `/sims/personage/inquire` | GET | 根据工号/学号查个人信息 |

### 学管待新增接口（学管同步开发中需补充）

| 接口 | 用途 | 涉及的新字段 |
|------|------|-------------|
| `POST /sims/face/match` | 人脸特征向量匹配学生身份 | 学管需存储人脸特征或调用第三方算法 |
| `GET /sims/students/dormitory` | 获取住宿学生宿舍分配数据 | `building`(楼栋)、`room`(房间号)、`active`(是否在校) |
| Student 扩展 | 宿舍管理所需 | `dorm_building`, `dorm_room`, `face_photo_url` |

---

## 配置总览

| 模块 | 可配置项 | 配置说明 |
|------|----------|---------|
| Stream Gateway | RTSP 地址、帧率、分辨率、重连策略、JPEG 质量 | 见 01-stream-gateway.md §5 |
| Face Recognition | 检测阈值、最小人脸、ROI 区域、学管 API 地址、进出判断参数 | 见 02-face-recognition.md §6 |
| Dormitory Service | 查宿触发时间、晚归阈值、同步间隔、告警开关 | 见 03-dormitory-service.md §8 |

所有配置项均可通过 API 动态调整，无需重启服务。

---

## 测试环境

| 组件 | 说明 |
|------|------|
| **Simulation Server** (`test-env/server/main.py`) | FastAPI 服务，模拟多路摄像头画面（默认4路）注入 Kafka，提供 Web 测试面板 |
| **启动脚本** (`test-env/start.sh`) | 自动检测基础设施 → 安装依赖 → 启动测试服务器 |
| **Web Dashboard** | `http://localhost:8082/` 可视化控制台，可手动模拟进出事件 |
| **模拟 API** | `POST /api/cameras/{id}/simulate` 注入事件，`GET /api/events` 查看日志 |

详见 [测试环境技术设计](../design/test-env/01-test-environment.md)。

## 基础架构部署

| 服务 | 用途 | 端口 |
|------|------|------|
| **Kafka** (`confluentinc/cp-kafka:7.6.1`) | 模块间异步消息 | `9092` |
| **ZooKeeper** | Kafka 依赖 | `2181` |
| **Redis** (`redis:7-alpine`) | 实时状态缓存、人脸特征缓存 | `6379` |
| **MariaDB** (`mariadb:10.11`) | 查宿数据持久化（MySQL 兼容） | `3306` |
| **MinIO** | 人脸抓拍图存储 | `9000`(API) `9001`(Console) |

详见 [部署指南](../deployment-guide.md)。

## 相关文档

| 文档 | 说明 | 位置 |
|------|------|------|
| 架构设计 | 系统架构、模块划分、技术选型 | [doc/main.md](../main.md) |
| 开发指南 | 环境搭建、编码规范、Git 工作流 | [doc/development-guide.md](../development-guide.md) |
| 部署指南 | 硬件要求、Docker Compose、配置说明 | [doc/deployment-guide.md](../deployment-guide.md) |
| 技术设计文档 | 后端架构/数据库/API/集成/摄像头/测试环境 | [doc/design/README.md](../design/README.md) |
| **双人分工指南** | 感知层 vs 业务层详细分工、接口契约、AI 实现指示 | [team-division.md](team-division.md) |
