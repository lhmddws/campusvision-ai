# CampusVision AI — Dormitory Service 产品需求文档

> **文档版本:** v1.0  
> **所属项目:** CampusVision AI 校园安防 AI 视觉分析平台 / 学生管理系统 (SMS)  
> **模块名称:** Dormitory Service (宿舍管理 AI 子系统 — 核心业务模块)  
> **更新日期:** 2026-05-15  
> **状态:** 初稿

---

## 目录

1. [模块概述](#1-模块概述)
2. [技术架构](#2-技术架构)
3. [数据流设计](#3-数据流设计)
4. [业务模型](#4-业务模型)
5. [功能清单](#5-功能清单)
   - 5.1 [人员状态管理](#51-人员状态管理)
   - 5.2 [每晚查宿统计](#52-每晚查宿统计核心功能)
   - 5.3 [学管数据同步](#53-学管数据同步)
   - 5.4 [告警与异常](#54-告警与异常)
   - 5.5 [数据统计与报表](#55-数据统计与报表)
   - 5.6 [配置管理](#56-配置管理)
6. [REST API 设计](#6-rest-api-设计)
7. [数据模型](#7-数据模型)
8. [配置清单](#8-配置清单)
9. [与学管系统的 API 契约](#9-与学管系统的-api-契约)
10. [附录](#10-附录)

---

## 1. 模块概述

### 1.1 定位

**Dormitory Service** 是学生管理系统 (Student Management System, SMS) 的子服务，专注宿舍区域的人员管理与每晚查宿统计。它消费 AI Engine 输出的进出事件（通过 Kafka），结合学管系统的宿舍分配数据，维护每栋宿舍楼人员的实时在校状态，并在每晚定时生成查宿报告。

### 1.2 核心职责

| 职责 | 说明 |
|------|------|
| **Kafka 事件消费** | 消费 `t_dorm_event` topic 中的进出事件，解析学生身份与出入方向 |
| **人员状态维护** | 实时更新每栋宿舍楼学生的在校/离校状态 |
| **每晚查宿统计** | 每日固定时间触发统计，生成每栋楼的查宿报告 |
| **数据同步** | 定时从学管系统同步宿舍分配数据（学生→楼栋→房间） |
| **告警处理** | 陌生人进入、夜间归寝、长时间未归等异常告警 |
| **REST API 开放** | 为前端/学管系统提供查宿数据、状态查询等接口 |

### 1.3 边界

| 包含 (In Scope) | 不包含 (Out of Scope) |
|---|---|
| Kafka 进出事件消费与处理 | 人脸检测/识别算法（AI Engine 负责） |
| 人员在校状态实时维护 | 摄像头流处理与视频解码（Stream Gateway 负责） |
| 每晚查宿统计与报表 | 前端页面实现 |
| 学管数据定时同步 | 视频帧的 AI 推理（AI Engine 负责） |
| 陌生人/异常告警 | 深度学习模型训练 |
| REST API 对外暴露 | 摄像头设备管理 |
| Redis 实时缓存 + PostgreSQL 持久化 | RTSP 拉流与推流 |

### 1.4 系统上下文

```
┌─────────────────────────────────────────────────────────────────┐
│                      学生管理系统 (SMS)                           │
│  ┌──────────────────┐    ┌──────────────────────────────────┐   │
│  │  学管主系统        │    │   AI 子系统 (CampusVision)       │   │
│  │  SpringBoot + Vue │    │                                  │   │
│  │                   │    │  ┌──────────┐  ┌──────────────┐ │   │
│  │  学生管理          │◄──►│  │ Stream  │  │   AI Engine  │ │   │
│  │  宿舍分配     ────┼────┼─►│ Gateway  │─►│  (人脸检测)    │ │   │
│  │  班级信息          │    │  └──────────┘  └───────┬──────┘ │   │
│  └──────────────────┘    │                          │        │   │
│                          │                    Kafka │        │   │
│                          │                     t_dorm_event   │   │
│                          │                          │        │   │
│                          │  ┌───────────────────────▼──────┐ │   │
│                          │  │    Dormitory Service          │ │   │
│                          │  │    (本 PRD, standalone JAR)   │ │   │
│                          │  │    Spring Boot 3              │ │   │
│                          │  │    PostgreSQL + Redis         │ │   │
│                          │  └───────────────┬──────────────┘ │   │
│                          └──────────────────┼────────────────┘   │
│                                             │                    │
│                                     REST API│                    │
│                                             ▼                    │
│                                    辅导员/宿管 用户               │
└─────────────────────────────────────────────────────────────────┘
```

### 1.5 部署说明

| 属性 | 说明 |
|------|------|
| **交付物** | 独立 Spring Boot JAR |
| **接入方式** | 先独立部署运行，后续通过依赖注入方式接入主 SpringBoot 进程 |
| **数据库** | 可使用与学管系统相同的 MySQL/PostgreSQL 实例，也可独立部署 |
| **缓存** | Redis（人员实时状态缓存） |
| **消息队列** | Kafka（消费 AI Engine 推送的进出事件） |
| **摄像头规模** | 4 个摄像头（每栋宿舍楼入口 1 个） |

---

## 2. 技术架构

### 2.1 技术选型

| 层级 | 技术方案 | 说明 |
|---|---|---|
| 应用框架 | Gin | HTTP 框架 |
| 开发语言 | Go 1.26 | 编译型高性能语言 |
| ORM | sqlx | 数据库访问层 |
| 业务数据库 | MariaDB 10.11 | 持久化业务数据 |
| 缓存 | Redis 7+ | 实时状态缓存、热点数据 |
| 消息队列 | Kafka 3.x | 事件驱动，消费进出事件 |
| API 文档 | SpringDoc OpenAPI (Swagger) | 自动生成 API 文档 |
| 任务调度 | Spring Scheduler / XXL-Job | 定时查宿统计与同步 |
| 配置中心 | Spring Cloud Config / 数据库配置表 | 动态配置管理 |
| 数据校验 | Jakarta Validation | 请求参数校验 |

### 2.2 模块架构

```
┌──────────────────────────────────────────────────────────────────┐
│                      Dormitory Service                           │
│                                                                  │
│  ┌──────────────────┐    ┌──────────────────────────────┐       │
│  │   Kafka Consumer  │    │       REST Controller        │       │
│  │   (事件消费层)      │    │       (API 暴露层)           │       │
│  │                   │    │                              │       │
│  │   t_dorm_event    │    │  /api/dormitory/*            │       │
│  └────────┬─────────┘    └──────────┬───────────────────┘       │
│           │                        │                            │
│           ▼                        ▼                            │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    Service 层 (业务逻辑)                   │   │
│  │                                                          │   │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────────┐ │   │
│  │  │ StudentStatus│ │ NightlyReport│ │ AlertService     │ │   │
│  │  │ Service      │ │ Service      │ │ (告警处理)        │ │   │
│  │  └──────────────┘ └──────────────┘ └──────────────────┘ │   │
│  │                                                          │   │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────────┐ │   │
│  │  │ SyncService  │ │ StatsService │ │ ConfigService    │ │   │
│  │  │ (学管同步)     │ │ (统计报表)    │ │ (配置管理)        │ │   │
│  │  └──────────────┘ └──────────────┘ └──────────────────┘ │   │
│  └──────────────────────────────────────────────────────────┘   │
│           │                        │                            │
│           ▼                        ▼                            │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                 Repository / DAO 层                       │   │
│  │                                                          │   │
│  │  MyBatis-Plus Mapper    +    Redis Template               │   │
│  └────────┬───────────────────────────┬────────────────────┘   │
│           │                          │                        │
│           ▼                          ▼                        │
│  ┌──────────────┐          ┌──────────────────┐               │
│  │  PostgreSQL   │          │     Redis        │               │
│  │  (持久化数据)   │          │  (实时状态缓存)   │               │
│  └──────────────┘          └──────────────────┘               │
└──────────────────────────────────────────────────────────────────┘
```

### 2.3 依赖关系

| 依赖方向 | 系统 | 协议/媒介 | 说明 |
|---------|------|----------|------|
| 消费 | AI Engine | Kafka `t_dorm_event` | 接收人员进出事件 |
| 调用 | 学管系统 (SIMS) | REST API / HTTP | 获取学生宿舍分配数据 |
| 被调用 | 前端 / 学管系统 | REST API / HTTP | 提供查宿数据与状态查询 |
| 依赖 | PostgreSQL | JDBC | 持久化业务数据 |
| 依赖 | Redis | Redis Protocol | 实时人员状态缓存 |

---

## 3. 数据流设计

### 3.1 全链路数据流

```
                        ┌──────────────┐
                        │  4 栋宿舍楼    │
                        │  (每栋入口1个)  │
                        └──────┬───────┘
                               │ RTSP
                               ▼
                     ┌───────────────────┐
                     │  Stream Gateway   │
                     │  (Go, FFmpeg)     │
                     └───────┬───────────┘
                             │ Frames
                             ▼
                     ┌───────────────────┐
                     │    AI Engine      │
                     │  (Python, YOLO)   │
                     │  人脸检测+识别     │
                     │  进出判定         │
                     └───────┬───────────┘
                             │ Kafka: t_dorm_event
                             ▼
                    ┌────────────────────┐
                    │  Dormitory Service  │  ◄── 本模块
                    │                     │
                    │  ┌───────────────┐  │
                    │  │ 事件消费 & 解析 │  │
                    │  │ student_id +  │  │
                    │  │ event_type    │  │
                    │  └───────┬───────┘  │
                    │          │          │
                    │          ▼          │
                    │  ┌───────────────┐  │
                    │  │ 状态更新       │  │
                    │  │ entry → 在宿舍 │  │
                    │  │ exit  → 离校   │  │
                    │  │ Redis + PG    │  │
                    │  └───────┬───────┘  │
                    │          │          │
                    │          ▼          │
                    │  ┌───────────────┐  │
                    │  │ 定时查宿统计   │  │
                    │  │ (默认 23:00)  │  │
                    │  └───────┬───────┘  │
                    │          │          │
                    │          ▼          │
                    │  ┌───────────────┐  │
                    │  │ 告警处理       │  │
                    │  │ 陌生人/晚归   │  │
                    │  │ 长时间未归    │  │
                    │  └───────────────┘  │
                    └────────────────────┘
                             │
                             ▼
                    ┌────────────────────┐
                    │   REST API 对外     │
                    │  辅导员/学管系统/前端 │
                    └────────────────────┘
```

### 3.2 Kafka Topic 契约

#### t_dorm_event（消费）

| 属性 | 值 |
|------|-----|
| **Topic 名称** | `t_dorm_event` |
| **角色** | Consumer |
| **Producer** | AI Engine |
| **消息格式** | JSON |
| **分区策略** | 按 `building` 哈希分区 |

消息 Schema：

```json
{
  "eventId": "evt_20260515_001",
  "building": "A",                    // 宿舍楼栋号：A / B / C / D
  "studentId": "S2024001",
  "studentName": "张三",
  "eventType": "entry",               // entry | exit
  "confidence": 0.95,                 // 人脸识别置信度
  "faceSnapshot": "https://minio:9000/snapshots/20260515/xxx.jpg",
  "timestamp": "2026-05-15T22:30:00+08:00"
}
```

Consumer 配置：

| 配置项 | 值 |
|--------|-----|
| Consumer Group | `dormitory-service` |
| 消费策略 | `latest` |
| 自动提交 | `false`（手动提交 offset） |
| 最大拉取条数 | 500 / poll |
| 反序列化 | JSON (Jackson) |
| 异常处理 | 解析失败 → DLQ (dead letter queue) |

#### 异常事件（产生，供其他系统消费）

| 属性 | 值 |
|------|-----|
| **Topic 名称** | `t_dorm_alert` |
| **角色** | Producer |
| **消息格式** | JSON |

```json
{
  "alertId": "alert_20260515_001",
  "type": "STRANGER_ENTRY",           // STRANGER_ENTRY | LONG_ABSENT | CROSS_BUILDING | LATE_RETURN | SYSTEM
  "building": "A",
  "studentId": null,
  "description": "陌生人进入 A 栋宿舍楼",
  "faceSnapshotUrl": "https://minio:9000/snapshots/xxx.jpg",
  "timestamp": "2026-05-15T23:05:00+08:00",
  "severity": "high"                  // low | medium | high | critical
}
```

---

## 4. 业务模型

### 4.1 人员状态 (StudentStatus)

维护每名住宿学生的实时在校状态，是查宿统计的核心数据源。

```
StudentStatus {
    studentId:      String        // 学号
    studentName:    String        // 学生姓名
    building:       String        // 所属宿舍楼栋 (A/B/C/D)
    room:           String        // 房间号 (e.g., "A-301")
    gender:         String        // 性别
    class:          String        // 班级
    grade:          String        // 年级

    isInDorm:       Boolean       // 当前是否在宿舍
    lastEntryTime:  DateTime      // 最近一次进入时间
    lastExitTime:   DateTime      // 最近一次离开时间
    todayStatus:    Enum          // 今晚状态:
                                  //   "in"      - 已归（有 entry 记录）
                                  //   "out"     - 未归
                                  //   "unknown" - 无法确定（无记录）

    lastUpdateTime: DateTime      // 最后更新时间
}
```

### 4.2 进出事件 (DormEvent)

AI Engine 检测到人员进出楼栋时产生的事件，是本服务的核心输入。

```
DormEvent {
    eventId:        String        // 事件唯一 ID
    building:       String        // 楼栋 (A/B/C/D)
    studentId:      String        // 学生学号（识别到的）
    studentName:    String        // 学生姓名
    eventType:      Enum          // "entry" | "exit"
    confidence:     Float         // 人脸识别置信度 [0, 1]
    faceSnapshot:   String        // 抓拍快照 URL
    timestamp:      DateTime      // 事件发生时间

    // 以下字段由本服务消费时补充
    isProcessed:    Boolean       // 是否已处理
    isStranger:     Boolean       // 是否为陌生人事件
}
```

### 4.3 每晚查宿统计 (NightlyReport)

每日固定时间生成的查宿报告，是辅导员查寝的核心数据。

```
NightlyReport {
    reportId:       Long          // 报告 ID
    date:           LocalDate     // 统计日期
    building:       String        // 楼栋 (A/B/C/D)

    totalStudents:  Integer       // 应住人数（从学管同步）
    presentCount:   Integer       // 已归人数
    absentCount:    Integer       // 未归人数
    lateReturnCount:Integer       // 晚归人数
    strangerCount:  Integer       // 陌生人/无法识别记录数
    unknownCount:   Integer       // 无法确定状态人数

    createdTime:    DateTime      // 统计生成时间
    triggeredBy:    Enum          // AUTO | MANUAL
    status:         Enum          // PENDING | COMPLETED | FAILED

    details:        [StudentStatus]  // 明细列表（每人一条）
}
```

### 4.4 业务规则矩阵

| 事件 | 状态变更 | 影响统计 | 是否告警 |
|------|---------|---------|---------|
| 本楼学生 entry (06:00-22:00) | isInDorm=true, todayStatus=in | 计入已归 | 否 |
| 本楼学生 entry (22:00-06:00) | isInDorm=true, todayStatus=in | 计入已归 + 晚归 | 是（晚归告警） |
| 本楼学生 exit | isInDorm=false | 当日无变化 | 否 |
| 陌生人 entry | 不创建状态 | 计入 strangerCount | 是（陌生人告警） |
| 非本楼学生 entry | 不创建状态 / 记录 | 计入 strangerCount | 是（跨楼栋告警） |
| 当日 23:00 无 entry 记录 | — | 计入未归 | 取决于 absent.alert_hours |

---

## 5. 功能清单

---

### 5.1 人员状态管理

#### 5.1.1 功能描述

维护每栋宿舍楼住宿学生的实时在校状态（在宿舍/离校），基于 Kafka 进出事件实时更新。状态数据同时存储于 Redis（实时查询）和 PostgreSQL（持久化）。

#### 5.1.2 基本信息

| 项目 | 内容 |
|---|---|
| **功能ID** | `F-DS-001` |
| **优先级** | P0 (核心基础能力) |
| **数据源** | Kafka `t_dorm_event` |
| **存储** | Redis（实时，带 TTL）+ PostgreSQL（持久化） |

#### 5.1.3 事件处理流程

```
收到 Kafka 事件
    │
    ▼
┌─────────────────┐
│ 事件解析 & 校验   │
│ building 是否有效 │
│ eventType 合法   │
└────────┬────────┘
         │
    ┌────▼────┐
    │ 是否是   │──否──→ 标记为 isStranger=true
    │ 本楼学生?│         记录陌生人事件
    └────┬────┘         触发陌生人告警
         │ 是
         │
    ┌────▼────┐
    │ 是否是   │──否──→ 记录跨楼栋事件
    │ 本楼栋   │         触发跨楼栋告警
    │ 住宿的?  │
    └────┬────┘
         │ 是
         │
    ┌────▼──────────────┐
    │ 更新学生状态        │
    │                    │
    │ entry:             │
    │   isInDorm = true  │
    │   lastEntryTime = now│
    │   todayStatus = in │
    │                    │
    │ exit:              │
    │   isInDorm = false │
    │   lastExitTime = now│
    └────┬──────────────┘
         │
    ┌────▼──────────────┐
    │ 更新 Redis 缓存     │
    │ Key: student:{id}  │
    │ TTL: 次日 06:00    │
    └────┬──────────────┘
         │
    ┌────▼──────────────┐
    │ 持久化到 PostgreSQL │
    │ dorm_entry_exit_event│
    │ dorm_student_status│
    └─────────────────────┘
```

#### 5.1.4 处理细节

| 场景 | 行为 |
|------|------|
| 收到 entry 事件，学生已标记为在宿舍 | 仅更新 `lastEntryTime`，状态不变 |
| 收到 exit 事件，学生已标记为离校 | 仅更新 `lastExitTime`，状态不变 |
| 收到 entry 事件，标记晚归 | 若 entry 时间在晚归阈值（默认 22:00）之后，额外触发晚归告警 |
| 相同学生 1 分钟内重复事件 | 去重处理（基于 eventId 幂等） |
| 人脸置信度低于阈值（默认 0.6） | 标记为低置信度事件，由管理员确认 |
| Redis 缓存 TTL 过期 | 下次查询回源 PostgreSQL |

#### 5.1.5 实时状态缓存设计

```
Redis Key 设计:
  dorm:student:{studentId}:status    → Hash
    Fields:
      isInDorm      → "true" / "false"
      lastEntryTime → ISO datetime
      lastExitTime  → ISO datetime
      todayStatus   → "in" / "out" / "unknown"
  TTL: 次日 06:00（通过定时任务或到期评估）

  dorm:building:{building}:students  → Set (楼栋内所有学生ID)

Redis Key 前缀命名空间:
  dorm:*  — 宿舍模块专用 key 空间
```

---

### 5.2 每晚查宿统计（核心功能）

#### 5.2.1 功能描述

每日固定时间触发查宿统计，生成每栋宿舍楼的学生归寝报告。这是整个 Dormitory Service 最核心的功能，直接服务于辅导员查寝场景。

#### 5.2.2 基本信息

| 项目 | 内容 |
|---|---|
| **功能ID** | `F-DS-002` |
| **优先级** | P0 (核心功能) |
| **触发方式** | 定时自动触发（默认 23:00）+ 手动触发 API |
| **统计粒度** | 楼栋 → 楼层 → 房间 |
| **存储** | PostgreSQL（报表表 + 明细表） |

#### 5.2.3 统计流程

```
定时触发 (默认 23:00)
    │
    ▼
┌─────────────────────────────┐
│ Step 1: 获取应归学生列表      │
│ 从 dorm_student_assignment   │
│ 获取所有 active 的住宿学生    │
└──────────┬──────────────────┘
           │
           ▼
┌─────────────────────────────┐
│ Step 2: 获取今日进出记录      │
│ 从 dorm_entry_exit_event     │
│ 筛选今日 00:00 ~ 现在的事件   │
└──────────┬──────────────────┘
           │
           ▼
┌─────────────────────────────┐
│ Step 3: 逐人判定状态          │
│                            │
│ 有 entry 记录 → present     │
│ 无 entry 记录 → absent      │
│ entry > 22:00 → late_return │
│ 陌生人记录   → stranger     │
│ 无法识别人脸 → unknown       │
└──────────┬──────────────────┘
           │
           ▼
┌─────────────────────────────┐
│ Step 4: 按楼栋/楼层/房间聚合  │
│ 生成统计报表                 │
└──────────┬──────────────────┘
           │
           ▼
┌─────────────────────────────┐
│ Step 5: 写入 PostgreSQL      │
│ dorm_nightly_report (汇总)   │
│ dorm_nightly_detail (明细)   │
└──────────┬──────────────────┘
           │
           ▼
┌─────────────────────────────┐
│ Step 6: 触发告警检查          │
│ 长时间未归 → 推送告警         │
└─────────────────────────────┘
```

#### 5.2.4 统计规则

| 状态 | 判定条件 | 说明 |
|------|---------|------|
| **已归 (present)** | 当日有至少 1 条 entry 事件 | 学生出现在宿舍楼入口 |
| **未归 (absent)** | 当日无任何 entry 事件 | 学生未出现在宿舍楼 |
| **晚归 (late_return)** | 当日最后一条 entry 时间 > 晚归阈值 | 阈值默认 22:00，可配置 |
| **陌生人 (stranger)** | event 中 studentId 不在本楼宿舍分配中 | 无法识别身份 |
| **无法确定 (unknown)** | 人脸识别置信度低于阈值或匹配失败 | 需要人工确认 |

#### 5.2.5 统计输出示例

**楼栋维度**：
```
A 栋查宿报告 — 2026-05-15
  应归: 120人
  已归: 108人  (90.0%)
  未归: 12人   (10.0%)
  晚归: 5人
  陌生人: 2人
```

**房间维度**：
```
A-301 (4人间):
  - 张三    ✓ 已归  (22:15进入) [晚归]
  - 李四    ✓ 已归  (19:30进入)
  - 王五    ✗ 未归
  - 赵六    ✓ 已归  (21:00进入)
```

#### 5.2.6 定时策略

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| 触发时间 | 23:00 | cron 表达式或固定时间 |
| 时区 | Asia/Shanghai | 支持时区配置 |
| 重新统计 | 不自动重试 | 失败时需手动触发 |
| 统计窗口 | 当日 00:00 ~ 统计时间 | 不跨日 |

#### 5.2.7 手动触发

提供 REST API 支持手动触发重新统计，用于：
- 定时统计失败后的补救
- 管理员需要提前查看（如 22:00 辅导员突击查寝）
- 变更配置后验证结果

---

### 5.3 学管数据同步

> ⚠️ **实际情况核查**：学管当前 OpenAPI **不存在专门的宿舍学生数据接口**。Student 表仅有 `dormitory`(int) 字段，缺少 `building`(楼栋)、`room`(房间号)。以下 API 需学管同步开发中新增。

#### 5.3.1 功能描述

定时从学管系统 (SIMS) 拉取学生宿舍分配数据，确保本服务中的学生基础信息与学管系统保持一致。同步失败时触发告警并自动重试。

#### 5.3.2 与学管实际数据的映射

学管 Student 现有字段与本服务所需字段的对应关系：

| 本服务需要 | 学管现有字段 | 状态 | 说明 |
|-----------|-------------|------|------|
| student_id | `studentNumber` | ✅ 存在 | 学号 |
| student_name | `name` | ✅ 存在 | 姓名 |
| building | ❌ 无 | **待新增** | 宿舍楼栋标识 (A/B/C/D) |
| room | ❌ 无 | **待新增** | 房间号 |
| class | `myClass` | ✅ 存在 | 班级名 |
| grade | `grader` | ✅ 存在 | 年级 |
| gender | `gender` | ✅ 存在 | 性别 |
| active | ❌ 无 | **待新增** | 是否在校/住宿中 |
| phone | `phone` | ✅ 存在 | 联系电话 |
| dormitory | `dormitory` (int) | ⚠️ 仅有宿舍号 | 现为 int 类型，含义不明 |
| face_photo_url | ❌ 无 | 备选 | 人脸照片（路径 B 需要） |

**结论**: 学管 Student 表需要扩展 `building`(varchar)、`room`(varchar)、`active`(boolean) 三个字段。或新增独立的 `dormitory_assignment` 子表。

#### 5.3.3 基本信息

| 项目 | 内容 |
|---|---|
| **功能ID** | `F-DS-003` |
| **优先级** | P0 |
| **同步方向** | 学管系统 → Dormitory Service（单向拉取） |
| **同步周期** | 默认每 60 分钟（可配置） |
| **通信协议** | REST API（学管系统暴露的 HTTP 接口，待新增） |

#### 5.3.4 同步内容

| 数据字段 | 类型 | 来源 | 说明 |
|---------|------|------|------|
| student_id | String | 学管系统 | 学号，唯一标识 |
| student_name | String | 学管系统 | 学生姓名 |
| building | String | **学管待新增** | 宿舍楼栋 |
| room | String | **学管待新增** | 房间号 |
| class | String | 学管系统 | 班级 (myClass) |
| grade | String | 学管系统 | 年级 (grader) |
| gender | String | 学管系统 | 性别 |
| active | Boolean | **学管待新增** | 是否在校/住宿中 |
| phone | String | 学管系统 | 联系电话 |

#### 5.3.5 同步流程

```
定时触发 (默认 60分钟间隔)
    │
    ▼
┌──────────────────────────┐
│ 调用学管 API              │
│ GET /sims/students/      │   ← 学管路径风格 /sims/
│   dormitory (待新增)      │      此接口需学管同步开发中新增
│                          │
│ Header:                  │
│   Authorization: Bearer  │
│   <token>                │
└──────────┬───────────────┘
           │
    ┌──────▼──────┐
    │ 请求成功?    │──否──→ 降级: 使用 /sims/student/get-list 兜底
    └──────┬──────┘         │ （但缺少 building/room 字段）
           │ 是             │ 仍失败 → 告警
           ▼                ▼
    ┌──────────────┐
    │ 增量/全量更新  │
    │ dorm_student  │
    │ _assignment   │
    │               │
    │ 新增: INSERT  │
    │ 更新: UPDATE  │
    │ 删除: 标记    │
    │ active=false  │
    └──────┬───────┘
           │
           ▼
    ┌──────────────┐
    │ 记录同步日志   │
    │ 写入 sync_log │
    └──────────────┘
```

#### 5.3.6 同步策略

| 策略 | 说明 |
|------|------|
| 首次同步 | 全量拉取所有住宿学生数据 |
| 后续同步 | 基于 last_update 做增量同步（学管需提供时间戳过滤） |
| 冲突处理 | 以学管系统数据为权威来源，本地不做编辑 |
| 数据删除 | 学生退宿/换宿 → 标记 `active=false`，不物理删除 |
| 同步超时 | 默认 30 秒，超时触发重试 |
| **降级方案** | 若 `GET /sims/students/dormitory` 不可用，降级使用 `GET /sims/student/get-list` 获取基础学生信息（仅含 studentNumber/name/class，不含 dormitory/room） |

---

### 5.4 告警与异常

#### 5.4.1 功能描述

基于实时事件和查宿统计结果，自动检测异常场景并生成告警。告警可推送到 Kafka、记录到数据库，供上游系统消费处理。

#### 5.4.2 基本信息

| 项目 | 内容 |
|---|---|
| **功能ID** | `F-DS-004` |
| **优先级** | P1 |
| **告警渠道** | Kafka `t_dorm_alert` + PostgreSQL 持久化 |
| **告警级别** | low / medium / high / critical |

#### 5.4.3 告警类型

| 告警类型 | 触发条件 | 级别 | 说明 |
|---------|---------|------|------|
| **陌生人进入** | 学管系统中无匹配身份的人员进入宿舍楼 | high | 可能有安全风险 |
| **长时间未归** | 学生离校超过 `absent.alert_hours` 小时（默认 24h）仍未返回 | medium | 需要辅导员关注 |
| **跨楼栋串门** | 非本楼栋住宿学生进入该楼栋 | low | 记录行为但非严重问题 |
| **晚归** | 夜间 (22:00~06:00) 进入宿舍楼 | low | 自动记录，可按规则配置是否告警 |
| **同步失败** | 连续 N 次从学管系统同步数据失败 | high | 服务依赖中断 |
| **数据异常** | 某楼栋应归人数突变 > 阈值（如 ±20%） | medium | 学管数据可能异常 |
| **摄像头离线** | 某楼栋入口摄像头长时间无事件上报 | critical | AI 分析链路中断 |

#### 5.4.4 告警处理流程

```
事件/统计触发
    │
    ▼
┌──────────────┐
│ 规则匹配      │
│ 条件检查      │
│ 防重复检查    │──→ 相同告警在冷却期内 → 跳过
└──────┬───────┘
       │ 匹配
       ▼
┌──────────────┐
│ 生成告警记录   │
│ 写入 PostgreSQL│
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ 推送 Kafka    │
│ t_dorm_alert  │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ 通知上游系统   │
│ 学管平台 / 前端 │
└──────────────┘
```

#### 5.4.5 告警去重与防抖

| 策略 | 默认值 | 说明 |
|------|--------|------|
| 同类型告警冷却 | 300 秒 | 同一类型告警最小间隔 |
| 同学生告警冷却 | 600 秒 | 同一学生告警最小间隔 |
| 陌生人去重 | 同一陌生人面容 hash | 基于人脸特征 hash 去重 |
| 告警频率限制 | 100 条/分钟（全局） | 防告警风暴 |

---

### 5.5 数据统计与报表

#### 5.5.1 功能描述

提供查宿数据的多维度统计与可视化查询能力，支持今日概览、历史趋势、明细查询和导出。

#### 5.5.2 基本信息

| 项目 | 内容 |
|---|---|
| **功能ID** | `F-DS-005` |
| **优先级** | P1 |
| **数据源** | PostgreSQL 报表表 |
| **导出格式** | CSV / Excel (Apache POI) |

#### 5.5.3 统计维度

| 维度 | 粒度 | 说明 |
|------|------|------|
| 时间 | 日 / 周 / 月 / 自定义 | 按时间聚合 |
| 楼栋 | A / B / C / D | 按楼栋聚合 |
| 楼层 | 每栋楼的楼层 | 按自然层聚合（房间号前缀） |
| 房间 | 具体房间号 | 最细粒度 |
| 年级 | 按年级分组 | 跨楼栋统计 |
| 班级 | 按班级分组 | 跨楼栋统计 |

#### 5.5.4 统计指标

| 指标 | 计算方式 | 说明 |
|------|---------|------|
| 应住人数 (total) | dorm_student_assignment 中 active=true | 基准值 |
| 入住率 | presentCount / totalCount × 100% | 反映整体归寝情况 |
| 未归率 | absentCount / totalCount × 100% | 反映缺勤情况 |
| 晚归率 | lateReturnCount / totalCount × 100% | 反映纪律情况 |
| 陌生人次数 | 当日陌生人事件计数 | 反映安全隐患 |
| 异常率 | (absentCount + strangerCount) / totalCount × 100% | 综合异常指标 |

#### 5.5.5 统计输出示例

```
=== 今日概览 (2026-05-15) ===

┌─────────┬──────┬──────┬──────┬──────┬──────┬──────┐
│ 楼栋     │ 应住  │ 已归  │ 未归  │ 晚归  │ 入住率 │ 异常  │
├─────────┼──────┼──────┼──────┼──────┼──────┼──────┤
│ A栋      │ 120  │ 108  │ 12   │ 5    │ 90.0%│ 10.0%│
│ B栋      │ 96   │ 88   │ 8    │ 3    │ 91.7%│ 8.3% │
│ C栋      │ 144  │ 120  │ 24   │ 8    │ 83.3%│ 16.7%│
│ D栋      │ 80   │ 72   │ 8    │ 2    │ 90.0%│ 10.0%│
├─────────┼──────┼──────┼──────┼──────┼──────┼──────┤
│ 合计     │ 440  │ 388  │ 52   │ 18   │ 88.2%│ 11.8%│
└─────────┴──────┴──────┴──────┴──────┴──────┴──────┘

=== 未归学生名单 ===
A栋 (12人):
  - A-302 王五     ✗ 未归    班级: 计算机2101班
  - A-105 刘七     ✗ 未归    班级: 软件2102班
  ...

=== 晚归学生名单 ===
A栋 (5人):
  - A-301 张三     晚归 22:15  班级: 计算机2101班
  - A-203 李九     晚归 23:30  班级: 软件2101班
  ...
```

---

### 5.6 配置管理

#### 5.6.1 功能描述

提供统一的系统配置管理能力，支持运行时动态调整配置项，无需重启服务。所有配置项持久化到数据库，变更后实时生效。

#### 5.6.2 基本信息

| 项目 | 内容 |
|---|---|
| **功能ID** | `F-DS-006` |
| **优先级** | P1 |
| **存储** | PostgreSQL `dorm_config` 表 + Redis 缓存 |
| **生效方式** | 配置更新后立即生效（主动刷新缓存） |

#### 5.6.3 配置管理界面（由前端提供，本服务仅提供 API）

| 操作 | API | 说明 |
|------|-----|------|
| 查询全部配置 | GET /api/dormitory/config | 返回键值对列表 |
| 更新单条配置 | PUT /api/dormitory/config | 更新后自动同步 Redis |
| 重置为默认值 | DELETE /api/dormitory/config/{key} | 恢复默认配置 |
| 配置变更历史 | GET /api/dormitory/config/history | 审计日志 |

#### 5.6.4 配置分组

| 分组 | 说明 | 示例配置 |
|------|------|---------|
| 查宿规则 | 查宿触发时间、晚归阈值等 | nightly_report.trigger_time |
| 告警阈值 | 各类告警的判定阈值和开关 | absent.alert_hours |
| 同步策略 | 学管数据同步相关配置 | sync.student.interval_min |
| Kafka 配置 | 消息队列相关 | kafka.consumer.topic |
| 系统参数 | 服务运行参数 | 缓存 TTL、页面大小等 |

---

## 6. REST API 设计

### 6.1 API 概览

| 方法 | 路径 | 说明 | 优先级 |
|------|------|------|--------|
| **查宿相关** | | | |
| GET | `/api/dormitory/nightly-report/today` | 获取今日查宿统计 | P0 |
| GET | `/api/dormitory/nightly-report/{date}` | 按日期查询查宿统计 | P0 |
| GET | `/api/dormitory/nightly-report/{date}/building/{building}` | 按楼栋查询查宿明细 | P0 |
| GET | `/api/dormitory/nightly-report/{date}/building/{building}/room/{room}` | 按房间查询查宿明细 | P0 |
| POST | `/api/dormitory/nightly-report/trigger` | 手动触发查宿统计 | P0 |
| **人员状态** | | | |
| GET | `/api/dormitory/students/status` | 所有人员在校状态 | P0 |
| GET | `/api/dormitory/students/status?building=A` | 按楼栋查询状态 | P0 |
| GET | `/api/dormitory/students/{studentId}/status` | 单个人员状态 | P0 |
| GET | `/api/dormitory/students/{studentId}/events` | 进出记录查询 | P1 |
| **事件查询** | | | |
| GET | `/api/dormitory/events` | 进出事件列表（筛选） | P1 |
| **陌生人记录** | | | |
| GET | `/api/dormitory/strangers` | 陌生人记录列表 | P1 |
| **配置管理** | | | |
| GET | `/api/dormitory/config` | 获取全部配置 | P1 |
| PUT | `/api/dormitory/config` | 更新配置 | P1 |
| GET | `/api/dormitory/config/history` | 配置变更历史 | P2 |
| **数据同步** | | | |
| POST | `/api/dormitory/sync/students` | 手动从学管同步 | P0 |
| GET | `/api/dormitory/sync/status` | 同步状态查询 | P1 |
| **统计报表** | | | |
| GET | `/api/dormitory/stats/overview` | 今日概览 | P1 |
| GET | `/api/dormitory/stats/trend?days=30` | 历史趋势 | P1 |
| GET | `/api/dormitory/stats/export?date={date}` | 导出查宿数据 (CSV/Excel) | P2 |
| **健康检查** | | | |
| GET | `/api/dormitory/health` | 健康检查 | P0 |

### 6.2 API 详细规格

#### 6.2.1 查宿统计

**GET /api/dormitory/nightly-report/today**

获取今日查宿统计概览（所有楼栋汇总）。

```
Response 200:
{
  "date": "2026-05-15",
  "status": "COMPLETED",          // PENDING | COMPLETED | FAILED
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

**GET /api/dormitory/nightly-report/{date}/building/{building}**

按楼栋查询查宿明细。

```
Response 200:
{
  "date": "2026-05-15",
  "building": "A",
  "totalCount": 120,
  "presentCount": 108,
  "absentCount": 12,
  "lateReturnCount": 5,
  "strangerCount": 1,
  "floors": [
    {
      "floor": 3,
      "totalCount": 24,
      "presentCount": 22,
      "absentCount": 2,
      "lateReturnCount": 2,
      "rooms": [
        {
          "room": "A-301",
          "totalCount": 4,
          "presentCount": 3,
          "absentCount": 1,
          "lateReturnCount": 1,
          "students": [
            {
              "studentId": "S2024001",
              "studentName": "张三",
              "status": "present",
              "entryTime": "2026-05-15T22:15:00+08:00",
              "isLateReturn": true
            },
            {
              "studentId": "S2024002",
              "studentName": "李四",
              "status": "present",
              "entryTime": "2026-05-15T19:30:00+08:00",
              "isLateReturn": false
            },
            {
              "studentId": "S2024003",
              "studentName": "王五",
              "status": "absent",
              "entryTime": null,
              "isLateReturn": false
            }
          ]
        }
      ]
    }
  ]
}
```

**POST /api/dormitory/nightly-report/trigger**

手动触发查宿统计。支持按楼栋触发或全量触发。

```
Request:
{
  "building": "A",              // 可选，不传则全量统计
  "date": "2026-05-15"          // 可选，默认今日
}

Response 200:
{
  "message": "查宿统计已触发",
  "taskId": "report_20260515_A",
  "status": "PENDING"
}
```

#### 6.2.2 人员状态

**GET /api/dormitory/students/status**

查询人员在校状态列表。支持楼栋、房间、班级等筛选。

```
Request Params:
  building: String (optional)   // 楼栋筛选
  room: String (optional)       // 房间筛选
  status: String (optional)     // in | out | unknown
  page: Integer (default: 1)
  size: Integer (default: 20)

Response 200:
{
  "total": 440,
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

**GET /api/dormitory/students/{studentId}/status**

查询单个学生的详细在校状态。

```
Response 200:
{
  "studentId": "S2024001",
  "studentName": "张三",
  "building": "A",
  "room": "A-301",
  "class": "计算机2101班",
  "grade": "大三",
  "gender": "男",
  "isInDorm": true,
  "lastEntryTime": "2026-05-15T22:15:00+08:00",
  "lastExitTime": "2026-05-15T07:30:00+08:00",
  "todayStatus": "in",
  "todayEntryCount": 1,
  "todayExitCount": 1,
  "weeklyEntryCount": 7,
  "weeklyAbsentDays": 0
}
```

**GET /api/dormitory/students/{studentId}/events**

查询单个学生的进出事件记录。

```
Request Params:
  startTime: DateTime (optional)
  endTime: DateTime (optional)
  page: Integer (default: 1)
  size: Integer (default: 20)

Response 200:
{
  "studentId": "S2024001",
  "studentName": "张三",
  "total": 128,
  "page": 1,
  "size": 20,
  "records": [
    {
      "eventId": "evt_20260515_001",
      "building": "A",
      "eventType": "entry",
      "confidence": 0.95,
      "timestamp": "2026-05-15T22:15:00+08:00"
    }
  ]
}
```

#### 6.2.3 事件查询

**GET /api/dormitory/events**

进出事件列表查询，支持多维筛选。

```
Request Params:
  building: String (optional)
  eventType: String (optional)     // entry | exit
  studentId: String (optional)
  startTime: DateTime (optional)
  endTime: DateTime (optional)
  isStranger: Boolean (optional)
  page: Integer (default: 1)
  size: Integer (default: 20)

Response 200:
{
  "total": 5840,
  "page": 1,
  "size": 20,
  "records": [
    {
      "eventId": "evt_20260515_001",
      "building": "A",
      "studentId": "S2024001",
      "studentName": "张三",
      "eventType": "entry",
      "confidence": 0.95,
      "faceSnapshotUrl": "https://minio:9000/snapshots/xxx.jpg",
      "isStranger": false,
      "timestamp": "2026-05-15T22:15:00+08:00"
    }
  ]
}
```

#### 6.2.4 陌生人记录

**GET /api/dormitory/strangers**

陌生人/无法识别人员记录列表。

```
Request Params:
  building: String (optional)
  startTime: DateTime (optional)
  endTime: DateTime (optional)
  page: Integer (default: 1)
  size: Integer (default: 20)

Response 200:
{
  "total": 28,
  "page": 1,
  "size": 20,
  "records": [
    {
      "id": 1,
      "building": "A",
      "faceSnapshotUrl": "https://minio:9000/snapshots/stranger_xxx.jpg",
      "confidence": 0.32,
      "detectedTime": "2026-05-15T14:30:00+08:00",
      "eventType": "entry",
      "status": "UNCONFIRMED"       // UNCONFIRMED | CONFIRMED | DISMISSED
    }
  ]
}
```

#### 6.2.5 配置管理

**GET /api/dormitory/config**

获取系统全部配置项。

```
Response 200:
{
  "nightly_report.trigger_time": "23:00",
  "nightly_report.timezone": "Asia/Shanghai",
  "late_return.threshold": "22:00",
  "absent.alert_hours": 24,
  "sync.student.enabled": true,
  "sync.student.interval_min": 60,
  "alert.stranger.enabled": true,
  "alert.cross_building": true,
  ...
}
```

**PUT /api/dormitory/config**

更新配置项。

```
Request:
{
  "nightly_report.trigger_time": "22:30",
  "late_return.threshold": "22:30"
}

Response 200:
{
  "message": "配置更新成功",
  "updated": ["nightly_report.trigger_time", "late_return.threshold"],
  "failed": []
}
```

#### 6.2.6 数据同步

**POST /api/dormitory/sync/students**

手动触发从学管系统同步宿舍数据。

```
Response 200:
{
  "message": "同步任务已触发",
  "syncId": "sync_20260515_001",
  "status": "IN_PROGRESS"
}
```

**GET /api/dormitory/sync/status**

查询同步状态。

```
Response 200:
{
  "lastSyncTime": "2026-05-15T12:00:00+08:00",
  "lastSyncStatus": "SUCCESS",         // SUCCESS | FAILED | IN_PROGRESS
  "lastSyncCount": 440,
  "lastError": null,
  "totalSyncCount": 58,
  "nextSyncTime": "2026-05-15T13:00:00+08:00"
}
```

#### 6.2.7 统计报表

**GET /api/dormitory/stats/overview**

今日概览数据。

```
Response 200:
{
  "date": "2026-05-15",
  "totalStudents": 440,
  "presentCount": 388,
  "absentCount": 52,
  "lateReturnCount": 18,
  "strangerCount": 3,
  "occupancyRate": 88.2,
  "buildings": [
    {
      "building": "A",
      "totalStudents": 120,
      "presentCount": 108,
      "absentCount": 12,
      "lateReturnCount": 5,
      "occupancyRate": 90.0
    }
  ],
  "trend": [
    {
      "date": "2026-05-14",
      "occupancyRate": 87.5
    },
    {
      "date": "2026-05-13",
      "occupancyRate": 89.1
    }
  ]
}
```

**GET /api/dormitory/stats/trend?days=30**

历史趋势数据。

```
Response 200:
{
  "days": 30,
  "startDate": "2026-04-16",
  "endDate": "2026-05-15",
  "series": [
    {
      "date": "2026-05-15",
      "presentCount": 388,
      "absentCount": 52,
      "lateReturnCount": 18,
      "occupancyRate": 88.2
    }
  ],
  "summary": {
    "avgOccupancyRate": 87.3,
    "maxOccupancyRate": 92.1,
    "minOccupancyRate": 82.5,
    "totalAbsentPersonDays": 1560
  }
}
```

**GET /api/dormitory/stats/export?date=2026-05-15&building=A&format=csv**

导出查宿数据。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| date | String | 是 | 导出日期 |
| building | String | 否 | 按楼栋过滤 |
| format | String | 否 | 导出格式：csv（默认）| excel |

```
Response 200:
  Content-Type: text/csv (或 application/vnd.ms-excel)
  Content-Disposition: attachment; filename="nightly-report-2026-05-15.csv"

  楼栋,房间,学号,姓名,班级,状态,进入时间,是否晚归
  A,A-301,S2024001,张三,计算机2101班,present,22:15,是
  A,A-301,S2024002,李四,计算机2101班,present,19:30,否
  ...
```

#### 6.2.8 健康检查

**GET /api/dormitory/health**

```
Response 200:
{
  "status": "UP",
  "timestamp": "2026-05-15T23:00:00+08:00",
  "components": {
    "redis": { "status": "UP", "latencyMs": 2 },
    "mariadb": { "status": "UP", "latencyMs": 5 },
    "kafka": { "status": "UP", "lag": 0 },
    "simsApi": { "status": "UP", "lastSync": "2026-05-15T12:00:00+08:00" }
  },
  "metrics": {
    "totalStudents": 440,
    "activeStudents": 440,
    "todayEvents": 5840,
    "todayStrangers": 3
  }
}
```

---

## 7. 数据模型

### 7.1 核心表结构

#### 7.1.1 学生宿舍分配表 (`dorm_student_assignment`)

来自学管系统的同步数据，记录每名学生的宿舍分配信息。

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | BIGINT | PK, AUTO_INCREMENT | 自增主键 |
| `student_id` | VARCHAR(32) | UNIQUE, NOT NULL | 学号 |
| `student_name` | VARCHAR(64) | NOT NULL | 学生姓名 |
| `building` | VARCHAR(8) | NOT NULL | 宿舍楼栋 (A/B/C/D) |
| `room` | VARCHAR(16) | NOT NULL | 房间号 |
| `class_name` | VARCHAR(64) | | 班级名称 |
| `grade` | VARCHAR(32) | | 年级 |
| `gender` | VARCHAR(8) | | 性别 |
| `phone` | VARCHAR(20) | | 联系电话 |
| `active` | BOOLEAN | DEFAULT true | 是否在校住宿中 |
| `sync_version` | BIGINT | DEFAULT 0 | 同步版本号（乐观锁） |
| `created_at` | DATETIME | NOT NULL | 创建时间 |
| `updated_at` | DATETIME | NOT NULL | 更新时间 |

> **索引**: `idx_building_room` (building, room), `idx_student_id` (student_id)

#### 7.1.2 人员在校状态表 (`dorm_student_status`)

学生的实时在校状态，Redis 为主、PostgreSQL 做持久化备份。

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | BIGINT | PK, AUTO_INCREMENT | 自增主键 |
| `student_id` | VARCHAR(32) | UNIQUE, NOT NULL | 学号 |
| `student_name` | VARCHAR(64) | NOT NULL | 学生姓名 |
| `building` | VARCHAR(8) | NOT NULL | 所属楼栋 |
| `room` | VARCHAR(16) | NOT NULL | 房间号 |
| `is_in_dorm` | BOOLEAN | DEFAULT false | 是否在宿舍 |
| `last_entry_time` | DATETIME | | 最近进入时间 |
| `last_exit_time` | DATETIME | | 最近离开时间 |
| `today_status` | VARCHAR(16) | DEFAULT 'unknown' | 今日状态: in/out/unknown |
| `today_entry_count` | INT | DEFAULT 0 | 今日进入次数 |
| `today_exit_count` | INT | DEFAULT 0 | 今日离开次数 |
| `last_update` | DATETIME | NOT NULL | 最后更新时间 |

> **索引**: `idx_building` (building), `idx_status` (today_status), `idx_student_id` (student_id)

#### 7.1.3 进出事件表 (`dorm_entry_exit_event`)

从 Kafka 消费的原始事件持久化。

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | BIGINT | PK, AUTO_INCREMENT | 自增主键 |
| `event_id` | VARCHAR(64) | UNIQUE, NOT NULL | 事件唯一 ID（幂等） |
| `building` | VARCHAR(8) | NOT NULL | 楼栋 |
| `student_id` | VARCHAR(32) | | 学生学号（可为空=陌生人） |
| `student_name` | VARCHAR(64) | | 学生姓名（可为空） |
| `event_type` | VARCHAR(8) | NOT NULL | entry / exit |
| `confidence` | DECIMAL(5,4) | | 人脸识别置信度 |
| `face_snapshot_url` | VARCHAR(512) | | 抓拍快照 URL |
| `is_stranger` | BOOLEAN | DEFAULT false | 是否为陌生人 |
| `is_processed` | BOOLEAN | DEFAULT false | 是否已被消费处理 |
| `timestamp` | DATETIME | NOT NULL | 事件时间 |
| `created_at` | DATETIME | NOT NULL | 记录创建时间 |

> **索引**: `idx_building_ts` (building, timestamp), `idx_student_id` (student_id), `idx_event_type` (event_type), `idx_timestamp` (timestamp), `idx_stranger` (is_stranger)

#### 7.1.4 每晚查宿统计表 (`dorm_nightly_report`)

查宿统计的汇总数据。

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | BIGINT | PK, AUTO_INCREMENT | 自增主键 |
| `report_date` | DATE | NOT NULL | 统计日期 |
| `building` | VARCHAR(8) | NOT NULL | 楼栋 |
| `total_count` | INT | NOT NULL | 应归人数 |
| `present_count` | INT | NOT NULL | 已归人数 |
| `absent_count` | INT | NOT NULL | 未归人数 |
| `late_return_count` | INT | DEFAULT 0 | 晚归人数 |
| `stranger_count` | INT | DEFAULT 0 | 陌生人记录数 |
| `unknown_count` | INT | DEFAULT 0 | 无法确定人数 |
| `status` | VARCHAR(16) | DEFAULT 'COMPLETED' | PENDING/COMPLETED/FAILED |
| `trigger_type` | VARCHAR(8) | DEFAULT 'AUTO' | AUTO/MANUAL |
| `created_at` | DATETIME | NOT NULL | 创建时间 |

> **唯一约束**: `uk_date_building` (report_date, building)  
> **索引**: `idx_report_date` (report_date), `idx_building` (building)

#### 7.1.5 查宿明细表 (`dorm_nightly_detail`)

每人每条查宿结果明细。

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | BIGINT | PK, AUTO_INCREMENT | 自增主键 |
| `report_id` | BIGINT | FK, NOT NULL | 关联 report 表 |
| `student_id` | VARCHAR(32) | NOT NULL | 学生学号 |
| `student_name` | VARCHAR(64) | NOT NULL | 学生姓名 |
| `building` | VARCHAR(8) | NOT NULL | 楼栋 |
| `room` | VARCHAR(16) | | 房间号 |
| `class_name` | VARCHAR(64) | | 班级 |
| `status` | VARCHAR(16) | NOT NULL | present/absent/late_return/unknown |
| `entry_time` | DATETIME | | 当日最早进入时间 |
| `exit_time` | DATETIME | | 当日最晚离开时间 |
| `is_late_return` | BOOLEAN | DEFAULT false | 是否晚归 |
| `created_at` | DATETIME | NOT NULL | 创建时间 |

> **索引**: `idx_report_id` (report_id), `idx_student_id` (student_id), `idx_status` (status), `idx_building_room` (building, room)

#### 7.1.6 配置表 (`dorm_config`)

系统动态配置项。

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | BIGINT | PK, AUTO_INCREMENT | 自增主键 |
| `config_key` | VARCHAR(128) | UNIQUE, NOT NULL | 配置键 |
| `config_value` | TEXT | NOT NULL | 配置值 |
| `config_type` | VARCHAR(32) | | 配置类型: string/int/bool |
| `description` | VARCHAR(256) | | 配置说明 |
| `default_value` | TEXT | | 默认值 |
| `group_name` | VARCHAR(32) | | 配置分组 |
| `updated_at` | DATETIME | NOT NULL | 更新时间 |

> **索引**: `idx_config_key` (config_key)

#### 7.1.7 陌生人记录表 (`dorm_stranger_record`)

陌生人/无法识别人员的进出记录。

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | BIGINT | PK, AUTO_INCREMENT | 自增主键 |
| `building` | VARCHAR(8) | NOT NULL | 楼栋 |
| `face_snapshot_url` | VARCHAR(512) | | 抓拍快照 URL |
| `confidence` | DECIMAL(5,4) | | 最高置信度 |
| `event_type` | VARCHAR(8) | NOT NULL | entry / exit |
| `detected_time` | DATETIME | NOT NULL | 发现时间 |
| `status` | VARCHAR(16) | DEFAULT 'UNCONFIRMED' | UNCONFIRMED/CONFIRMED/DISMISSED |
| `remark` | VARCHAR(256) | | 备注 |
| `created_at` | DATETIME | NOT NULL | 创建时间 |

> **索引**: `idx_building` (building), `idx_status` (status), `idx_detected_time` (detected_time)

#### 7.1.8 告警记录表 (`dorm_alert_record`)

告警事件的持久化记录。

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | BIGINT | PK, AUTO_INCREMENT | 自增主键 |
| `alert_id` | VARCHAR(64) | UNIQUE, NOT NULL | 告警唯一 ID |
| `alert_type` | VARCHAR(32) | NOT NULL | STRANGER_ENTRY/LONG_ABSENT/CROSS_BUILDING/LATE_RETURN/SYNC_FAILED/SYSTEM |
| `building` | VARCHAR(8) | | 相关楼栋 |
| `student_id` | VARCHAR(32) | | 相关学生（可为空） |
| `severity` | VARCHAR(8) | NOT NULL | low/medium/high/critical |
| `description` | VARCHAR(512) | | 告警描述 |
| `face_snapshot_url` | VARCHAR(512) | | 快照 URL |
| `is_read` | BOOLEAN | DEFAULT false | 是否已读 |
| `is_resolved` | BOOLEAN | DEFAULT false | 是否已处理 |
| `occurred_at` | DATETIME | NOT NULL | 发生时间 |
| `created_at` | DATETIME | NOT NULL | 记录时间 |

> **索引**: `idx_alert_type` (alert_type), `idx_severity` (severity), `idx_occurred_at` (occurred_at), `idx_building` (building)

#### 7.1.9 同步日志表 (`dorm_sync_log`)

学管数据同步的操作日志。

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | BIGINT | PK, AUTO_INCREMENT | 自增主键 |
| `sync_type` | VARCHAR(32) | NOT NULL | STUDENT |
| `sync_status` | VARCHAR(16) | NOT NULL | SUCCESS/FAILED/IN_PROGRESS |
| `total_count` | INT | | 同步总数 |
| `success_count` | INT | | 成功数 |
| `fail_count` | INT | | 失败数 |
| `error_message` | TEXT | | 错误信息 |
| `duration_ms` | BIGINT | | 耗时（毫秒） |
| `started_at` | DATETIME | NOT NULL | 开始时间 |
| `finished_at` | DATETIME | | 结束时间 |

> **索引**: `idx_sync_type` (sync_type), `idx_started_at` (started_at)

### 7.2 ER 关系图

```
┌──────────────────────┐       ┌──────────────────────────┐
│  dorm_student_       │       │  dorm_student_status     │
│  assignment          │       │                          │
│──────────────────────│       │──────────────────────────│
│ PK id                │       │ PK id                    │
│ UNIQUE student_id    │──1:1──│ UNIQUE student_id        │
│ building, room       │       │ is_in_dorm               │
│ class_name, grade    │       │ last_entry_time          │
│ active               │       │ last_exit_time           │
│ created_at, updated  │       │ today_status             │
└──────────────────────┘       └──────────────────────────┘
         │                              │
         │ 1:N                          │
         ▼                              │
┌──────────────────────┐               │
│  dorm_nightly_       │               │
│  detail              │               │
│──────────────────────│               │
│ PK id                │               │
│ FK report_id         │               │
│ student_id           │               │
│ status, room         │               │
│ entry_time           │               │
└──────────┬───────────┘               │
           │ N:1                        │
           ▼                           │
┌──────────────────────┐               │
│  dorm_nightly_       │               │
│  report              │               │
│──────────────────────│               │
│ PK id                │               │
│ report_date + bld    │               │
│ total/present/absent │               │
│ late_return_count    │               │
│ stranger_count       │               │
│ status, trigger_type │               │
└──────────────────────┘               │
                                        │
┌──────────────────────┐               │
│  dorm_entry_exit_    │               │
│  event               │               │
│──────────────────────│               │
│ PK id                │───────────────┘
│ UNIQUE event_id      │  （按 student_id 关联）
│ building, student_id │
│ event_type           │
│ confidence, is_strg  │
│ timestamp            │
└──────────────────────┘

┌──────────────────────┐  ┌──────────────────────┐
│  dorm_stranger_      │  │  dorm_alert_record   │
│  record              │  │                      │
│──────────────────────│  │──────────────────────│
│ PK id                │  │ PK id                │
│ building             │  │ UNIQUE alert_id      │
│ face_snapshot_url    │  │ alert_type, severity │
│ status, remark       │  │ description          │
│ detected_time        │  │ is_resolved          │
└──────────────────────┘  └──────────────────────┘

┌──────────────────────┐  ┌──────────────────────┐
│  dorm_config         │  │  dorm_sync_log       │
│──────────────────────│  │──────────────────────│
│ PK id                │  │ PK id                │
│ UNIQUE config_key    │  │ sync_type, status    │
│ config_value, desc   │  │ total/success/fail   │
│ default_value        │  │ error_message        │
│ group_name           │  │ duration_ms          │
└──────────────────────┘  └──────────────────────┘
```

### 7.3 存储策略

| 数据类型 | 主存储 | 缓存 | 说明 |
|---------|--------|------|------|
| 学生宿舍分配 | PostgreSQL | Redis（可选） | 以学管同步为准，不本地编辑 |
| 实时在校状态 | Redis | — | TTL 次日 06:00 过期 |
| 在校状态持久化 | PostgreSQL | — | 异步写入，为查宿统计提供基础 |
| 进出事件 | PostgreSQL | — | 大量写入，按天分区表（可选） |
| 查宿报表 | PostgreSQL | Redis（当日） | 报表数据，查询频繁 |
| 配置 | PostgreSQL | Redis | 更新时刷新缓存 |
| 配置变更历史 | PostgreSQL | — | 审计日志 |

---

## 8. 配置清单

### 8.1 配置项完整列表

全部配置项存储在 `dorm_config` 表中，支持运行时动态修改，无需重启服务。

| 配置键 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| **查宿规则** | | | |
| `nightly_report.trigger_time` | string | `"23:00"` | 自动查宿每日触发时间 (HH:mm) |
| `nightly_report.timezone` | string | `"Asia/Shanghai"` | 查宿统计使用的时区 |
| `late_return.threshold` | string | `"22:00"` | 晚归判定时间阈值 (HH:mm)，在此时间后的 entry 记为晚归 |
| **告警阈值** | | | |
| `absent.alert_hours` | int | `24` | 未归告警阈值（小时），学生离校超过此时间触发告警 |
| `alert.stranger.enabled` | boolean | `true` | 陌生人进入宿舍告警开关 |
| `alert.cross_building.enabled` | boolean | `true` | 跨楼栋串门检测告警开关 |
| `alert.late_return.enabled` | boolean | `true` | 晚归记录与告警开关 |
| `alert.cooldown_seconds` | int | `300` | 同类型告警最小间隔（秒） |
| `alert.max_per_minute` | int | `100` | 全局告警频率上限（条/分钟） |
| **学管同步** | | | |
| `sync.student.enabled` | boolean | `true` | 自动同步学管宿舍数据开关 |
| `sync.student.interval_min` | int | `60` | 同步间隔（分钟） |
| `sync.student.api_url` | string | `(必填)` | 学管学生宿舍数据 API 地址。**待学管新增**，示例: `http://sims:8080/sims/students/dormitory`。降级: 可用 `http://sims:8080/sims/student/get-list` |
| `sync.student.api_token` | string | `(必填)` | 调用学管 API 的鉴权 Token |
| `sync.student.timeout_sec` | int | `30` | 同步请求超时时间（秒） |
| `sync.student.retry_max` | int | `3` | 同步失败最大重试次数 |
| `sync.student.retry_interval_sec` | int | `60` | 重试间隔（秒） |
| **Kafka 消费** | | | |
| `kafka.consumer.topic` | string | `"t_dorm_event"` | 进出事件 Topic 名称 |
| `kafka.consumer.group` | string | `"dormitory-service"` | 消费者组 ID |
| `kafka.bootstrap.servers` | string | `"localhost:9092"` | Kafka 集群地址（逗号分隔） |
| **缓存** | | | |
| `cache.status.ttl_hours` | int | `6` | 在校状态 Redis 缓存 TTL（小时） |
| `cache.config.ttl_minutes` | int | `5` | 配置信息 Redis 缓存 TTL（分钟） |
| **陌生人检测** | | | |
| `stranger.confidence_threshold` | float | `0.6` | 人脸识别置信度阈值，低于此值标记为陌生人 |
| **批量操作** | | | |
| `page.default_size` | int | `20` | 列表查询默认分页大小 |
| `page.max_size` | int | `100` | 列表查询最大分页大小 |

### 8.2 配置分组管理

| 分组 | 包含配置项 | 说明 |
|------|-----------|------|
| `nightly` | nightly_report.*, late_return.* | 查宿规则相关 |
| `alert` | alert.* | 告警阈值与开关 |
| `sync` | sync.student.* | 学管同步策略 |
| `kafka` | kafka.* | 消息队列配置 |
| `cache` | cache.* | 缓存策略 |
| `stranger` | stranger.* | 陌生人检测 |
| `system` | page.* | 系统参数 |

---

## 9. 与学管系统的 API 契约

### 9.1 本服务调用学管系统

> ⚠️ **实际情况核查**：以下两个接口在学管当前 OpenAPI 中**均不存在**，需在学管同步开发中新增。
>
> **学管路径风格**: 所有业务接口使用 `/sims/` 前缀（如 `/sims/student/get-list`）
> **响应格式**: 学管统一 `{ "code": 200, "text": "success", "data": {...} }`
> **认证方式**: `Authorization: Bearer <token>`

#### 9.1.1 获取住宿学生数据（待学管新增）

本服务定时调用学管 API 同步学生宿舍分配信息。

```
GET /sims/students/dormitory              ← 建议路径，以实际协商为准

Headers:
  Authorization: Bearer <sync.student.api_token>

Query Params:
  updated_after: DateTime (optional)    // 增量同步，返回此时间后有更新的数据
  page: Integer (default: 1)
  size: Integer (default: 200)

Response 200:
{
  "code": 200,                             ← 学管统一响应格式
  "text": "success",
  "data": {
    "total": 440,
    "list": [
      {
        "studentNumber": "2024001",         ← 字段名对齐学管 Student schema
        "name": "张三",
        "myClass": "计算机2101班",
        "grader": 2023,
        "gender": "男",
        "phone": "138xxxxxxxx",
        "building": "A",                    ← 待学管新增字段
        "room": "301",                      ← 待学管新增字段
        "active": true                      ← 待学管新增字段
      }
    ]
  }
}
```

**备选降级**: 若此接口未就绪，可使用 `GET /sims/student/get-list` 替代（缺少 building/room 字段）。

#### 9.1.2 人脸特征匹配（待学管新增，AI 链路上调用）

Note: 此接口由 Face Recognition 模块调用，非本服务直接调用。仅在此列出以保持完整性。

```
POST /sims/face/match                     ← 建议路径，以实际协商为准
Content-Type: application/json
Authorization: Bearer <token>

Request:
{
  "feature_vector": [0.123, 0.456, ...],   // 512-dim float array
  "confidence_threshold": 0.6              // 置信度阈值
}

Response 200:
{
  "code": 200,
  "text": "success",
  "data": {
    "matched": true,
    "student": {
      "studentNumber": "2024001",
      "name": "张三",
      "myClass": "计算机2101班",
      "gender": "男"
    },
    "confidence": 0.89
  }
}
```

**路径 B（备选）**: 若学管不提供 face/match 接口，则由 Face Recognition 服务本地管理人脸特征库（需增加本地向量索引）。

### 9.2 学管系统调用本服务

学管系统通过本服务开放的 REST API 获取查宿数据，前端应用也通过这些接口展示。

#### 9.2.1 查宿概览

```
GET /api/dormitory/status/overview

Response 200:
{
  "date": "2026-05-15",
  "totalStudents": 440,
  "presentCount": 388,
  "absentCount": 52,
  "lateReturnCount": 18,
  "occupancyRate": 88.2,
  "buildings": [
    {
      "building": "A",
      "totalStudents": 120,
      "presentCount": 108,
      "absentCount": 12,
      "occupancyRate": 90.0
    }
  ]
}
```

#### 9.2.2 查宿明细

```
GET /api/dormitory/nightly-report/{date}

GET /api/dormitory/nightly-report/{date}/building/{building}

GET /api/dormitory/nightly-report/{date}/building/{building}/room/{room}
```

详情见 [6.2.1 查宿统计](#621-查宿统计)。

#### 9.2.3 全校查宿统计（接入主进程后统一暴露）

本服务提供的基础 API 在接入主 SpringBoot 进程后，通过主进程的网关层统一暴露给外部系统。

```
主进程网关 URL: /api/sims/dormitory/*
转发至: dormitory-service:8080/api/dormitory/*
```

### 9.3 接口错误码规范

| HTTP 状态码 | 错误码 | 说明 |
|-------------|--------|------|
| 400 | `INVALID_PARAMETER` | 请求参数校验失败 |
| 401 | `UNAUTHORIZED` | 认证失败 / Token 无效 |
| 403 | `FORBIDDEN` | 无权限访问 |
| 404 | `NOT_FOUND` | 请求资源不存在 |
| 409 | `CONFLICT` | 数据冲突（如重复统计） |
| 422 | `UNPROCESSABLE_ENTITY` | 业务规则校验失败 |
| 429 | `TOO_MANY_REQUESTS` | 请求频率超限 |
| 500 | `INTERNAL_ERROR` | 服务器内部错误 |
| 503 | `SERVICE_UNAVAILABLE` | 服务暂不可用（如数据库离线） |

错误响应格式：

```json
{
  "code": "INVALID_PARAMETER",
  "message": "楼栋参数不合法，仅支持 A/B/C/D",
  "details": {
    "field": "building",
    "rejectedValue": "E"
  },
  "timestamp": "2026-05-15T23:00:00+08:00",
  "requestId": "req_xxx"
}
```

---

## 10. 附录

### 10.1 核心业务流程时序

**日常查宿流程：**

```
辅导员/宿管                        Dormitory Service                  学管系统(SIMS)
    │                                    │                              │
    │  ① 日常监控                        │                              │
    │                                    │  ② 定时同步 ─────────────►  │
    │                                    │◄────────── 住宿学生数据 ─── │
    │                                    │                              │
    │                                    │  ③ AI Engine 推送进出事件    │
    │                                    │     Kafka: t_dorm_event      │
    │                                    │◄═══════════════════════════  │
    │                                    │                              │
    │                                    │  ④ 实时更新 Redis 状态       │
    │                                    │                              │
    │  ⑤ 23:00 自动查宿触发             │                              │
    │                                    │                              │
    │  ⑥ 查宿统计完成                    │                              │
    │◄══════ REST API ─════════════════  │                              │
    │                                    │                              │
    │  ⑦ 查看未归名单                    │                              │
    │  ⑧ 关注晚归/陌生人告警              │                              │
    │  ⑨ 导出报表存档                    │                              │
```

**异常告警流程：**

```
陌生人进入
    │
    ▼
Kafka: t_dorm_event
    │
    ▼
Dormitory Service 消费
    │
    ├── 匹配学管数据 → 未匹配到身份
    │
    ├── 标记为陌生人事件
    │
    ├── 写入 dorm_stranger_record
    │
    ├── 推送 Kafka: t_dorm_alert (STRANGER_ENTRY)
    │
    └── 学管/前端系统消费告警
```

### 10.2 性能指标

| 指标 | 目标值 | 说明 |
|------|--------|------|
| Kafka 事件消费延迟 | ≤ 500ms (P99) | 从事件产生到状态更新 |
| 实时状态查询响应 | ≤ 100ms (P99) | Redis 缓存命中 |
| 查宿报表查询响应 | ≤ 500ms (P99) | PostgreSQL 查询 |
| 手动触发查宿统计 | ≤ 10s | 单栋楼 200 人以内 |
| 学管全量同步耗时 | ≤ 30s | 5000 人以内 |
| 陌生人记录查询 | ≤ 200ms (P99) | 带时间范围筛选 |
| 每日事件处理量 | ≥ 10,000 条 | 4 栋楼 × 日夜流量 |
| 系统可用性 | ≥ 99.9% | 核心查宿功能 |

### 10.3 安全与运维

| 类别 | 要求 |
|------|------|
| **认证** | 所有 API 需通过 Token 或 Session 认证 |
| **鉴权** | 查宿数据按角色（辅导员/宿管/院领导）分级授权 |
| **审计** | 配置变更、手动触发统计等操作记录审计日志 |
| **幂等** | Kafka 消费基于 eventId 做幂等处理 |
| **限流** | API 层面做速率限制（按 IP/Token） |
| **数据隐私** | 人脸快照仅用于查宿，不可导出或滥用 |
| **备份** | 每日自动备份 PostgreSQL 查宿数据 |
| **告警** | 服务宕机、Kafka 积压、数据库连接失败 → 系统告警 |

### 10.4 版本历史

| 版本 | 日期 | 修改人 | 变更说明 |
|------|------|--------|---------|
| v1.0 | 2026-05-15 | Dormitory Service Team | 初稿完成 |

### 10.5 术语表

| 术语 | 说明 |
|------|------|
| **SIMS** | Student Information Management System，学生信息管理系统（学管系统） |
| **Dormitory Service** | 宿舍管理 AI 子系统，本 PRD 描述的主体 |
| **Nightly Report** | 每晚查宿统计报表 |
| **Present** | 已归，学生当日有进入宿舍楼记录 |
| **Absent** | 未归，学生当日无进入宿舍楼记录 |
| **Late Return** | 晚归，学生进入时间超过晚归阈值（默认 22:00） |
| **Stranger** | 陌生人，无法与学管系统中的学生身份匹配的人员 |
| **Cross-Building** | 跨楼栋，非本楼住宿学生进入该楼栋 |
| **t_dorm_event** | Kafka Topic，承载人员进出宿舍楼事件 |
| **t_dorm_alert** | Kafka Topic，承载宿舍模块告警事件 |
| **Standalone JAR** | 独立可执行的 Spring Boot JAR 包 |
