# CampusVision AI — 主进程对接 PRD

> **文档编号**: PRD-004  
> **模块名称**: 主进程对接（Dormitory Service — 核心业务模块）  
> **所属系统**: 学生管理系统（Student Management System）— 宿舍管理 AI 子系统  
> **版本**: v1.0  
> **最后更新**: 2026-05-15  
> **状态**: 初稿  

---

## 目录

1. [模块定位](#1-模块定位)
2. [架构与部署路线](#2-架构与部署路线)
3. [数据流设计](#3-数据流设计)
4. [功能清单](#4-功能清单)
5. [REST API 设计](#5-rest-api-设计)
6. [数据模型](#6-数据模型)
7. [配置清单](#7-配置清单)
8. [与学管系统的 API 契约](#8-与学管系统的-api-契约)
9. [接入主进程路线图](#9-接入主进程路线图)
10. [附录](#10-附录)

---

## 1. 模块定位

### 1.1 一句话定义

本模块是宿舍 AI 子系统的**业务核心**。消费感知层（你团队）推送的 `t_dorm_event` 进出事件，维护每名学生的实时在校状态，每晚 23:00 自动生成查宿报告，并通过 REST API 向上游学管系统和前端页面提供数据。

### 1.2 在整个系统中的位置

```
感知层（你团队）                        业务层（本 PRD —— 搭档）
┌─────────────────────┐               ┌──────────────────────────┐
│ Stream Gateway (Go) │               │  主进程对接 (SpringBoot)  │
│   拉流/解码/抽帧     │               │                          │
└────────┬────────────┘               │  ┌──────────────────┐   │
         │ Kafka t_dorm_frame         │  │ Kafka 事件消费    │   │
         ▼                            │  │ → 实时状态更新    │   │
┌─────────────────────┐               │  └──────────────────┘   │
│ Face Recognition    │               │  ┌──────────────────┐   │
│   人脸检测/匹配/方向  │──────────────►│  │ 每晚查宿统计     │   │
└─────────────────────┘               │  │ (默认 23:00)     │   │
         │ Kafka t_dorm_event         │  └──────────────────┘   │
         │                             │  ┌──────────────────┐   │
         │                             │  │ 告警规则引擎      │   │
         │                             │  └──────────────────┘   │
         │                             │  ┌──────────────────┐   │
         │                             │  │ 学管数据同步      │   │
         │                             │  └──────────────────┘   │
         │                             │  ┌──────────────────┐   │
         │                             │  │ REST API 暴露     │───► 学管/前端
         │                             │  └──────────────────┘   │
         │                             │  ┌──────────────────┐   │
         │                             │  │ 未来:接入主进程   │   │
         │                             │  └──────────────────┘   │
         │                             └──────────────────────────┘
```

### 1.3 核心职责

| 职责 | 说明 | 优先级 |
|------|------|--------|
| **Kafka 事件消费** | 消费 `t_dorm_event`，解析进出事件，按楼栋/学号分类 | P0 |
| **实时状态维护** | Redis 缓存每名学生的 in/out 状态，TTL 次日过期 | P0 |
| **每晚查宿统计** | 23:00 自动触发，按楼栋→房间→人汇总归寝情况 | P0 |
| **学管数据同步** | 定时从学管拉取学生宿舍分配数据（需学管新增接口） | P0 |
| **告警引擎** | 陌生人/晚归/长时间未归/跨楼栋/摄像头离线 | P1 |
| **REST API** | 供前端和学管系统查询查宿数据 | P0 |
| **数据持久化** | 进出事件、查宿报表、告警记录存入 PostgreSQL | P0 |
| **动态配置** | 查宿时间、告警阈值等运行时可修改，无需重启 | P1 |

### 1.4 边界

| 包含 (In Scope) | 不包含 (Out of Scope) |
|---|---|
| Kafka 进出事件消费与处理 | 人脸检测/识别算法（感知层负责） |
| 人员在校状态实时维护 | 摄像头流处理与视频解码（感知层负责） |
| 每晚查宿统计与报表 | 前端页面实现（在学管前端项目中） |
| 学管数据定时同步 | 深度学习模型训练 |
| 陌生人/异常告警 | 摄像头设备管理（见 PRD-005 摄像头功能实现） |
| REST API 对外暴露 | RTSP 拉流与推流 |
| Redis 实时缓存 + PostgreSQL 持久化 | |

---

## 2. 架构与部署路线

### 2.1 技术选型

| 层级 | 技术方案 | 说明 |
|------|---------|------|
| 应用框架 | Spring Boot 3.x | 最终会作为模块嵌入主进程 |
| 开发语言 | Java 17 / 21 | LTS |
| ORM | MyBatis-Plus | 数据库访问 |
| 业务数据库 | PostgreSQL 15+ / MySQL 8+ | 持久化 |
| 缓存 | Redis 7+ | 实时状态 |
| 消息队列 | Kafka 3.x | 事件驱动 |
| 任务调度 | Spring Scheduler | 定时查宿/同步 |
| 配置管理 | 数据库配置表 `dorm_config` | 动态配置 |
| API 文档 | SpringDoc OpenAPI | 自动生成 |

### 2.2 模块内部架构

```
┌─────────────────────────────────────────────────────┐
│               Dormitory Service (Spring Boot)        │
│                                                     │
│  ┌──────────────┐   ┌──────────────────────────┐   │
│  │ KafkaConsumer│   │   REST Controller          │   │
│  │ (事件消费层)  │   │   (/api/dormitory/*)      │   │
│  └──────┬───────┘   └──────────┬───────────────┘   │
│         │                      │                    │
│         ▼                      ▼                    │
│  ┌──────────────────────────────────────────────┐   │
│  │              Service 层 (业务逻辑)             │   │
│  │                                               │   │
│  │  ┌────────────┐  ┌──────────┐  ┌──────────┐  │   │
│  │  │EventService│  │ReportSvc │  │AlertSvc  │  │   │
│  │  │(事件处理)   │  │(查宿统计) │  │(告警)    │  │   │
│  │  └────────────┘  └──────────┘  └──────────┘  │   │
│  │                                               │   │
│  │  ┌────────────┐  ┌──────────┐  ┌──────────┐  │   │
│  │  │SyncService │  │StatsSvc  │  │ConfigSvc │  │   │
│  │  │(学管同步)   │  │(报表)    │  │(配置管理) │  │   │
│  │  └────────────┘  └──────────┘  └──────────┘  │   │
│  └──────────────────────────────────────────────┘   │
│         │                      │                    │
│         ▼                      ▼                    │
│  ┌──────────────────────────────────────────────┐   │
│  │          Repository / DAO 层                  │   │
│  │          MyBatis-Plus Mapper  +  Redis        │   │
│  └────────┬──────────────────────┬─────────────┘   │
│           │                      │                  │
│           ▼                      ▼                  │
│  ┌──────────────┐      ┌────────────────┐          │
│  │ PostgreSQL   │      │    Redis       │          │
│  │ (持久化)      │      │ (实时状态缓存)  │          │
│  └──────────────┘      └────────────────┘          │
└─────────────────────────────────────────────────────┘
```

### 2.3 部署路线：独立 JAR → 接入主进程

**阶段 1 — 独立部署（MVP）**

```
┌──────────────┐     HTTP/REST      ┌──────────────────┐
│  学管前端     │◄──────────────────│  Dormitory        │
│  (Vue)       │                    │  Service (JAR)    │
└──────────────┘                    │  :8080            │
                                    │                   │
┌──────────────┐     Kafka          │  PostgreSQL       │
│  感知层服务    │─────────────────►│  Redis             │
│  (Go+Python)  │   t_dorm_event   └──────────────────┘
└──────────────┘
```

**阶段 2 — 接入主进程（Phase 3）**

```
┌──────────────────────────────────────────────────┐
│              学管主进程 (SpringBoot)               │
│                                                    │
│  ┌──────────────┐  ┌──────────────────────────┐   │
│  │ 认证/鉴权     │  │  Dormitory 模块           │   │
│  │ 网关/路由     │  │  (从独立JAR移入)           │   │
│  └──────────────┘  └──────────────────────────┘   │
│                                                    │
│  统一 API 前缀: /api/sims/dormitory/*              │
└──────────────────────────────────────────────────┘
```

接入方式：
- 将本服务的 `Service` 层和 `Repository` 层作为 Maven 模块引入主进程
- `Controller` 层路径改为 `/api/sims/dormitory/` 统一前缀
- 复用主进程的认证鉴权、数据源、配置中心

---

## 3. 数据流设计

### 3.1 核心数据流

```
t_dorm_event (Kafka)
    │ { building, student_id, student_name, event_type, confidence, timestamp }
    ▼
┌──────────────────────┐
│ 事件消费 & 校验        │
│ • 解析 JSON           │
│ • building 合法性检查  │
│ • event_type 合法     │
│ • 幂等去重 (eventId)  │
└──────────┬───────────┘
           │
     ┌─────▼─────┐
     │ 是否是本楼   │──否──→ 陌生人处理
     │ 住宿学生?   │         → dorm_stranger_record
     └─────┬─────┘         → t_dorm_alert (STRANGER_ENTRY)
           │ 是
           │
     ┌─────▼─────┐
     │ 事件类型判断  │
     └─────┬─────┘
           │
     ┌─────┴─────┐   ┌─────┴─────┐
     │  entry    │   │   exit    │
     │ isInDorm  │   │ isInDorm  │
     │ =true     │   │ =false    │
     │ lastEntry │   │ lastExit  │
     │ =now      │   │ =now      │
     │ todayStat │   └─────┬─────┘
     │ ="in"     │         │
     └─────┬─────┘         │
           │               │
     ┌─────▼───────────────▼─────┐
     │  更新 Redis                │
     │  Key: dorm:student:{id}   │
     │  TTL: 次日 06:00          │
     └─────┬─────────────────────┘
           │
     ┌─────▼─────────────────────┐
     │  异步写入 PostgreSQL       │
     │  dorm_entry_exit_event    │
     │  dorm_student_status      │
     └─────┬─────────────────────┘
           │
           ▼
     ┌────────────────────────┐
     │  告警规则检查            │
     │  • entry > 22:00 → 晚归 │
     │  • 学生状态异常 → 告警   │
     └────────────────────────┘
```

### 3.2 每晚查宿统计流程

```
定时触发 (默认 23:00，可配置)
    │
    ▼
Step 1: 获取应归学生列表
    │ 从 dorm_student_assignment 获取全部 active 住宿学生
    ▼
Step 2: 获取今日进出记录
    │ 从 dorm_entry_exit_event 筛选当日 00:00~现在的 entry 事件
    ▼
Step 3: 逐人判定状态
    │ 有 entry 记录 → present
    │ 有 entry 且晚于 22:00 → late_return
    │ 无 entry 记录 → absent
    ▼
Step 4: 按楼栋→楼层→房间聚合
    │
    ▼
Step 5: 写入 PostgreSQL
    │ dorm_nightly_report (汇总)
    │ dorm_nightly_detail (每人明细)
    ▼
Step 6: 告警检查
    │ 长时间未归 → t_dorm_alert
```

---

## 4. 功能清单

### 4.1 人员状态管理 (F-DS-001) | P0

#### 事件处理规则

| 场景 | 行为 |
|------|------|
| 本楼学生 entry (06:00-22:00) | isInDorm=true, todayStatus=in |
| 本楼学生 entry (22:00-06:00) | isInDorm=true, lateReturn=true, 触发晚归记录 |
| 本楼学生 exit | isInDorm=false |
| 陌生人 entry | 不创建状态，记录陌生人，触发告警 |
| 非本楼学生 entry | 记录跨楼栋事件，触发跨楼栋告警 |
| 重复事件 (1min内同eventId) | 幂等丢弃 |

#### Redis 缓存设计

```
dorm:student:{studentId}:status  →  Hash
  isInDorm      → "true" / "false"
  lastEntryTime → ISO datetime
  lastExitTime  → ISO datetime
  todayStatus   → "in" / "out" / "unknown"
  TTL: 次日 06:00

dorm:building:{building}:students  →  Set (楼栋内所有学生ID)

dorm:building:{building}:status    →  Hash (楼栋级聚合缓存)
  totalStudents
  presentCount
  absentCount
  TTL: 5 分钟
```

#### 幂等方案

```
Kafka 消费时：
  1. 检查 eventId 是否在 Redis 已处理集合中
  2. 若存在 → 跳过
  3. 若不存在 → 处理 → 写入 Redis 已处理集合 (TTL=1h) → commit offset
```

### 4.2 每晚查宿统计 (F-DS-002) | P0

#### 判定规则

| 状态 | 条件 |
|------|------|
| **已归 (present)** | 当日有 ≥1 条 entry 事件 |
| **未归 (absent)** | 当日无 entry 事件 |
| **晚归 (late_return)** | 最后一条 entry 时间 > 晚归阈值 (默认 22:00) |
| **陌生人 (stranger)** | 学管无匹配身份 |
| **无法确定 (unknown)** | 置信度 < 阈值 |

#### 定时策略

| 配置 | 默认值 |
|------|--------|
| 触发时间 | 23:00 (cron 或固定时间) |
| 时区 | Asia/Shanghai |
| 重新统计 | 需手动触发，不支持自动重试 |

#### 统计输出

**楼栋维度**:
```
A 栋 — 2026-05-15
  应归: 120人  已归: 108人 (90.0%)
  未归: 12人   晚归: 5人  陌生人: 2人
```

**房间维度**:
```
A-301 (4人间):
  - 张三    ✓ 已归 (22:15) [晚归]
  - 李四    ✓ 已归 (19:30)
  - 王五    ✗ 未归
```

### 4.3 学管数据同步 (F-DS-003) | P0

> ⚠️ **现状**：学管 OpenAPI 无专门的宿舍学生接口，Student 表有 `dormitory`(int) 但缺 `building`、`room`、`active` 字段。需协调学管新增 `GET /sims/students/dormitory` 接口，或扩展 Student 表字段。

#### 同步策略

| 策略 | 说明 |
|------|------|
| 首次同步 | 全量拉取所有住宿学生 |
| 后续同步 | 基于 `updated_after` 增量拉取（需学管支持） |
| 冲突处理 | 以学管为权威来源，本服务只读不写 |
| 数据删除 | 标记 `active=false`，不物理删除 |
| 降级方案 | 若 `/sims/students/dormitory` 不可用，降级使用 `/sims/student/get-list`（缺 building/room） |

#### 同步触发

| 方式 | 说明 |
|------|------|
| 自动定时 | 默认每 60 分钟（`sync.student.interval_min` 可配） |
| 手动触发 | `POST /api/dormitory/sync/students` |
| 失败重试 | 最多 3 次，间隔 60 秒 |

### 4.4 告警引擎 (F-DS-004) | P1

#### 告警类型

| 类型 | 触发条件 | 级别 | 冷却 |
|------|---------|------|------|
| **陌生人进入** | 学管无匹配身份的人员进楼 | high | 300s |
| **长时间未归** | 离校超 `absent.alert_hours`(默认24h) 未返回 | medium | 600s |
| **跨楼栋串门** | 非本楼学生进入 | low | 300s |
| **晚归** | 22:00-06:00 期间 entry | low | 600s |
| **同步失败** | 连续 N 次学管同步失败 | high | — |
| **摄像头离线** | 某楼栋长时间无事件上报 (需感知层配合) | critical | 300s |

#### 告警去重

| 策略 | 默认值 |
|------|--------|
| 同类型告警冷却 | 300s |
| 同学生告警冷却 | 600s |
| 全局频率限制 | 100条/分钟 |

### 4.5 配置管理 (F-DS-005) | P1

全部配置持久化在 `dorm_config` 表，运行时通过 API 修改立即生效（Redis 缓存主动刷新）。

| 分组 | 示例配置 |
|------|---------|
| 查宿规则 | `nightly_report.trigger_time`, `late_return.threshold` |
| 告警阈值 | `alert.stranger.enabled`, `absent.alert_hours` |
| 同步策略 | `sync.student.interval_min`, `sync.student.api_url` |
| Kafka 消费 | `kafka.consumer.topic`, `kafka.bootstrap.servers` |
| 缓存 | `cache.status.ttl_hours`, `cache.config.ttl_minutes` |

---

## 5. REST API 设计

### 5.1 API 列表

| 方法 | 路径 | 说明 | 优先级 |
|------|------|------|--------|
| **查宿相关** | | | |
| GET | `/api/dormitory/nightly-report/today` | 今日查宿统计概览 | P0 |
| GET | `/api/dormitory/nightly-report/{date}` | 按日期查询 | P0 |
| GET | `/api/dormitory/nightly-report/{date}/building/{building}` | 按楼栋查询明细 | P0 |
| GET | `/api/dormitory/nightly-report/{date}/building/{building}/room/{room}` | 按房间查询明细 | P0 |
| POST | `/api/dormitory/nightly-report/trigger` | 手动触发查宿统计 | P0 |
| **人员状态** | | | |
| GET | `/api/dormitory/students/status` | 所有人员状态（支持 building/room/status 筛选） | P0 |
| GET | `/api/dormitory/students/{studentId}/status` | 单个学生状态 | P0 |
| GET | `/api/dormitory/students/{studentId}/events` | 单个学生进出记录 | P1 |
| **事件查询** | | | |
| GET | `/api/dormitory/events` | 进出事件列表（多维筛选） | P1 |
| **陌生人记录** | | | |
| GET | `/api/dormitory/strangers` | 陌生人记录列表 | P1 |
| **配置管理** | | | |
| GET | `/api/dormitory/config` | 获取全部配置 | P1 |
| PUT | `/api/dormitory/config` | 更新配置 | P1 |
| GET | `/api/dormitory/config/history` | 配置变更历史 | P2 |
| **数据同步** | | | |
| POST | `/api/dormitory/sync/students` | 手动触发学管同步 | P0 |
| GET | `/api/dormitory/sync/status` | 同步状态查询 | P1 |
| **统计报表** | | | |
| GET | `/api/dormitory/stats/overview` | 今日概览 | P1 |
| GET | `/api/dormitory/stats/trend?days=30` | 历史趋势 | P1 |
| GET | `/api/dormitory/stats/export` | 导出 CSV/Excel | P2 |
| **健康检查** | | | |
| GET | `/api/dormitory/health` | 健康检查 | P0 |

### 5.2 响应格式

统一响应格式（对齐学管系统风格）：

```json
{
  "code": 200,
  "text": "success",
  "data": { ... }
}
```

> 学管系统使用 `{ code, text, data }` 格式，code=200 为成功，非 200 为错误。

### 5.3 关键 API 响应示例

**GET /api/dormitory/nightly-report/today**

```json
{
  "date": "2026-05-15",
  "status": "COMPLETED",
  "totalStudents": 440,
  "presentCount": 388,
  "absentCount": 52,
  "lateReturnCount": 18,
  "strangerCount": 3,
  "unknownCount": 2,
  "buildings": [
    {
      "building": "A",
      "totalStudents": 120,
      "presentCount": 108,
      "absentCount": 12,
      "lateReturnCount": 5,
      "strangerCount": 1,
      "occupancyRate": 90.0
    }
  ],
  "summary": {
    "occupancyRate": 88.2,
    "abnormalCount": 75,
    "abnormalRate": 17.0
  },
  "generatedTime": "2026-05-15T23:05:00+08:00"
}
```

**GET /api/dormitory/students/status?building=A&room=A-301**

```json
{
  "total": 4,
  "page": 1,
  "size": 20,
  "records": [
    {
      "studentId": "S2024001",
      "studentName": "张三",
      "building": "A",
      "room": "A-301",
      "class": "计算机2101班",
      "isInDorm": true,
      "lastEntryTime": "2026-05-15T22:15:00+08:00",
      "lastExitTime": "2026-05-15T07:30:00+08:00",
      "todayStatus": "in"
    }
  ]
}
```

**GET /api/dormitory/health**

```json
{
  "status": "UP",
  "timestamp": "2026-05-15T23:00:00+08:00",
  "components": {
    "redis": { "status": "UP", "latencyMs": 2 },
    "postgresql": { "status": "UP", "latencyMs": 5 },
    "kafka": { "status": "UP", "lag": 0, "lastEventTime": "2026-05-15T22:59:00+08:00" },
    "simsApi": { "status": "UP", "lastSync": "2026-05-15T12:00:00+08:00" }
  }
}
```

---

## 6. 数据模型

### 6.1 表结构一览

| 表名 | 说明 | 数据量预估 |
|------|------|-----------|
| `dorm_student_assignment` | 学生宿舍分配（从学管同步） | ~500 行 |
| `dorm_student_status` | 人员在校状态 | ~500 行 |
| `dorm_entry_exit_event` | 进出事件 | ~10,000 行/日 |
| `dorm_nightly_report` | 每晚查宿统计汇总 | 4 行/日 |
| `dorm_nightly_detail` | 查宿明细 | ~500 行/日 |
| `dorm_stranger_record` | 陌生人记录 | ~50 行/日 |
| `dorm_alert_record` | 告警记录 | ~100 行/日 |
| `dorm_config` | 系统配置 | ~30 行 |
| `dorm_sync_log` | 同步日志 | ~20 行/日 |

### 6.2 核心表定义

#### `dorm_student_assignment` — 学生宿舍分配

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | BIGINT | PK | 自增主键 |
| `student_id` | VARCHAR(32) | UNIQUE, NOT NULL | 学号 |
| `student_name` | VARCHAR(64) | NOT NULL | 姓名 |
| `building` | VARCHAR(8) | NOT NULL | 楼栋 A/B/C/D |
| `room` | VARCHAR(16) | NOT NULL | 房间号 |
| `class_name` | VARCHAR(64) | | 班级 |
| `grade` | VARCHAR(32) | | 年级 |
| `gender` | VARCHAR(8) | | 性别 |
| `active` | BOOLEAN | DEFAULT true | 是否在校住宿 |

索引: `idx_building_room`(building, room), `idx_student_id`(student_id)

#### `dorm_entry_exit_event` — 进出事件

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | BIGINT | PK | 自增主键 |
| `event_id` | VARCHAR(64) | UNIQUE, NOT NULL | 事件ID (幂等) |
| `building` | VARCHAR(8) | NOT NULL | 楼栋 |
| `student_id` | VARCHAR(32) | | 学号(可为空=陌生人) |
| `student_name` | VARCHAR(64) | | 姓名 |
| `event_type` | VARCHAR(8) | NOT NULL | entry / exit |
| `confidence` | DECIMAL(5,4) | | 置信度 |
| `face_snapshot_url` | VARCHAR(512) | | 抓拍快照 |
| `is_stranger` | BOOLEAN | DEFAULT false | 是否陌生人 |
| `timestamp` | DATETIME | NOT NULL | 事件时间 |

索引: `idx_building_ts`(building, timestamp), `idx_student_id`(student_id), `idx_event_type`(event_type), `idx_stranger`(is_stranger)

#### `dorm_nightly_report` — 查宿统计汇总

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | BIGINT | PK | 自增主键 |
| `report_date` | DATE | NOT NULL | 统计日期 |
| `building` | VARCHAR(8) | NOT NULL | 楼栋 |
| `total_count` | INT | NOT NULL | 应归 |
| `present_count` | INT | NOT NULL | 已归 |
| `absent_count` | INT | NOT NULL | 未归 |
| `late_return_count` | INT | DEFAULT 0 | 晚归 |
| `stranger_count` | INT | DEFAULT 0 | 陌生人 |
| `status` | VARCHAR(16) | DEFAULT 'COMPLETED' | PENDING/COMPLETED/FAILED |

唯一约束: `uk_date_building`(report_date, building)

---

## 7. 配置清单

所有配置存储在 `dorm_config` 表，运行时通过 `PUT /api/dormitory/config` 动态修改。

| 配置键 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `nightly_report.trigger_time` | string | `"23:00"` | 自动查宿触发时间 |
| `nightly_report.timezone` | string | `"Asia/Shanghai"` | 时区 |
| `late_return.threshold` | string | `"22:00"` | 晚归判定阈值 |
| `absent.alert_hours` | int | `24` | 未归告警阈值(小时) |
| `alert.stranger.enabled` | bool | `true` | 陌生人告警开关 |
| `alert.cross_building.enabled` | bool | `true` | 跨楼栋告警开关 |
| `alert.late_return.enabled` | bool | `true` | 晚归告警开关 |
| `alert.cooldown_seconds` | int | `300` | 同类型告警冷却 |
| `alert.max_per_minute` | int | `100` | 全局告警频率上限 |
| `sync.student.enabled` | bool | `true` | 自动同步开关 |
| `sync.student.interval_min` | int | `60` | 同步间隔(分钟) |
| `sync.student.api_url` | string | `(必填)` | 学管宿舍数据 API 地址 |
| `sync.student.timeout_sec` | int | `30` | 同步超时 |
| `sync.student.retry_max` | int | `3` | 最大重试次数 |
| `kafka.consumer.topic` | string | `"t_dorm_event"` | 事件 Topic |
| `kafka.consumer.group` | string | `"dormitory-service"` | 消费组 ID |
| `kafka.bootstrap.servers` | string | `"localhost:9092"` | Kafka 地址 |
| `cache.status.ttl_hours` | int | `6` | 状态缓存 TTL |
| `stranger.confidence_threshold` | float | `0.6` | 陌生人置信度阈值 |

---

## 8. 与学管系统的 API 契约

### 8.1 本服务调用学管系统

> ⚠️ 以下接口在学管当前 OpenAPI 中不存在，需协调学管开发团队新增。

#### `GET /sims/students/dormitory`（待学管新增）

定时同步学生宿舍分配数据。

```json
// Response 200 (学管统一格式)
{
  "code": 200,
  "text": "success",
  "data": {
    "total": 440,
    "list": [
      {
        "studentNumber": "2024001",
        "name": "张三",
        "myClass": "计算机2101班",
        "building": "A",        // 待新增
        "room": "301",          // 待新增
        "active": true          // 待新增
      }
    ]
  }
}
```

**降级方案**: 若此接口不可用，使用 `GET /sims/student/get-list` 兜底（缺 building/room）。

#### 学管需配合的新增事项

| 事项 | 说明 | 影响 |
|------|------|------|
| `POST /sims/face/match` | 人脸特征向量匹配身份 | Face Recognition 服务调用，本服务不直接依赖 |
| `GET /sims/students/dormitory` | 查询住宿学生宿舍分配 | 本服务同步数据使用 |
| Student 表扩展 | 增加 `building`, `room`, `active` 字段 | 本服务需要的数据源 |

### 8.2 学管系统调用本服务

接入主进程前的独立部署阶段：

| 学管/前端请求 | 本服务响应 |
|-------------|-----------|
| `GET /api/dormitory/nightly-report/today` | 今日查宿概览 |
| `GET /api/dormitory/nightly-report/{date}/building/{b}` | 楼栋查宿明细 |
| `GET /api/dormitory/students/status?building=A` | 楼栋人员状态 |
| `GET /api/dormitory/strangers` | 陌生人列表 |

接入主进程后的统一路径：

```
主进程网关: /api/sims/dormitory/*
转发至:    dormitory-service:8080/api/dormitory/*
```

### 8.3 错误码规范

| HTTP 状态码 | 错误码 | 说明 |
|-------------|--------|------|
| 400 | INVALID_PARAMETER | 请求参数校验失败 |
| 401 | UNAUTHORIZED | 认证失败 |
| 404 | NOT_FOUND | 资源不存在 |
| 409 | CONFLICT | 数据冲突 |
| 500 | INTERNAL_ERROR | 服务器内部错误 |
| 503 | SERVICE_UNAVAILABLE | 服务暂不可用 |

错误响应体：
```json
{
  "code": "INVALID_PARAMETER",
  "message": "楼栋参数不合法，仅支持 A/B/C/D",
  "details": { "field": "building", "rejectedValue": "E" },
  "timestamp": "2026-05-15T23:00:00+08:00"
}
```

---

## 9. 接入主进程路线图

### 9.1 阶段划分

| 阶段 | 目标 | 关键任务 |
|------|------|---------|
| **Phase 1 · 独立部署** | MVP 核心闭环 | 独立 JAR 运行，Kafka 消费→状态→查宿→API |
| **Phase 2 · 体验完善** | 告警、报表、配置 | 告警引擎、历史趋势、配置管理、导出 |
| **Phase 3 · 接入主进程** | 无缝集成 | 作为 Maven 模块引入主进程，复用认证/数据源 |

### 9.2 Phase 3 迁移方案

```
接入前:
                                                           ┌──────────────┐
  学管前端 ──HTTP──► dormitory-service:8080/api/dormitory/*  │ PostgreSQL   │
                  (独立 JAR, 独立端口)                     │ Redis         │
                                                           └──────────────┘

接入后:
                                                           ┌──────────────┐
  学管前端 ──► 主进程网关 /api/sims/dormitory/*              │ PostgreSQL   │
              ▼                                              │ Redis         │
          DormitoryController (在主进程中)                    └──────────────┘
              ▼
          DormitoryService (Maven 模块)
              ▼
          DormitoryMapper + RedisTemplate
```

**迁移具体步骤**:

| 步骤 | 内容 |
|------|------|
| 1 | 抽取 Service + Mapper 为独立 Maven 模块 `dormitory-core` |
| 2 | 主进程引入 `dormitory-core` 依赖 |
| 3 | 主进程新建 `DormitoryController`，路径改为 `/api/sims/dormitory/*` |
| 4 | 复用主进程的 `SecurityContext` 代替独立 Token 认证 |
| 5 | 复用主进程的 `DataSource` 和 `RedisTemplate` 配置 |
| 6 | 独立 JAR 保留作为可选项（备胎 / 测试用） |
| 7 | 下线独立 JAR 的 HTTP 端口 |

### 9.3 需要主进程提供的能力

| 能力 | 说明 | 阶段 |
|------|------|------|
| 统一认证鉴权 | 复用 JWT Token 校验 | Phase 3 |
| 统一路由前缀 | `/api/sims/dormitory/*` 转发到本模块 | Phase 3 |
| 数据源配置 | 复用主进程的 PostgreSQL 连接池 | Phase 3 |
| Redis 配置 | 复用主进程的 Redis 连接 | Phase 3 |
| Kafka 配置 | 复用主进程的 Kafka 消费者工厂 | Phase 2-3 |
| 日志/监控 | 复用主进程的 ELK / Prometheus | Phase 3 |

---

## 10. 附录

### 10.1 与感知层的 Kafka 契约

**消费 Topic**: `t_dorm_event`

| 字段 | 类型 | 说明 |
|------|------|------|
| `event_id` | string | 事件唯一ID（用于幂等） |
| `camera_id` | string | 摄像头ID |
| `building` | string | 楼栋 A/B/C/D |
| `student_id` | string | 学号（空=陌生人） |
| `student_name` | string | 姓名 |
| `event_type` | string | "entry" / "exit" |
| `confidence` | float | 置信度 0-1 |
| `face_snapshot` | string | 抓拍快照(base64) |
| `timestamp_unix_ms` | int64 | 事件时间戳 |
| `is_stranger` | bool | 是否陌生人 |

**生产 Topic**: `t_dorm_alert`

| 字段 | 类型 | 说明 |
|------|------|------|
| `alert_id` | string | 告警ID |
| `type` | string | STRANGER_ENTRY / LONG_ABSENT / CROSS_BUILDING / LATE_RETURN / SYSTEM |
| `building` | string | 相关楼栋 |
| `description` | string | 告警描述 |
| `severity` | string | low / medium / high / critical |
| `timestamp` | datetime | 告警时间 |

### 10.2 性能指标

| 指标 | 目标 | 说明 |
|------|------|------|
| 事件消费延迟 | ≤ 500ms P99 | Kafka → Redis 状态更新 |
| 实时状态查询 | ≤ 100ms P99 | Redis 缓存命中 |
| 查宿报表查询 | ≤ 500ms P99 | PostgreSQL |
| 手动查宿统计 | ≤ 10s | 单栋 ≤ 200人 |
| 每日事件处理 | ≥ 10,000 条 | 4栋楼 |
| 系统可用性 | ≥ 99.9% | |

### 10.3 版本历史

| 版本 | 日期 | 变更说明 |
|------|------|---------|
| v1.0 | 2026-05-15 | 初稿 — 从原 PRD-003 拆分，聚焦主进程对接业务逻辑 |

---

> **本 PRD 对应开发目录**: `dormitory-service/`  
> **依赖感知层**: 消费 `t_dorm_event`（感知层产出）  
> **依赖学管系统**: `GET /sims/students/dormitory`（待学管新增）  
> **最终形态**: 作为 Maven 模块嵌入学管主 SpringBoot 进程
