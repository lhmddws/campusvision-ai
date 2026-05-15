# CampusVision AI — 摄像头功能实现 PRD

> **文档编号**: PRD-005  
> **模块名称**: 摄像头管理功能（Camera Management）  
> **所属系统**: 学生管理系统（Student Management System）— 宿舍管理 AI 子系统  
> **版本**: v1.0  
> **最后更新**: 2026-05-15  
> **状态**: 初稿  

---

## 目录

1. [模块定位](#1-模块定位)
2. [协作边界](#2-协作边界)
3. [功能清单](#3-功能清单)
4. [REST API 设计](#4-rest-api-设计)
5. [数据模型](#5-数据模型)
6. [配置清单](#6-配置清单)
7. [监控看板指标](#7-监控看板指标)
8. [附录](#8-附录)

---

## 1. 模块定位

### 1.1 一句话定义

摄像头是整个宿舍 AI 系统的**「传感器层」**。本模块负责在业务系统中管理这 4 个摄像头的设备信息、运行状态、配置参数，并提供摄像头层面的事件查询和抓拍回顾能力。它不负责拉流和解码（那是感知层 Stream Gateway 的事），而是负责「摄像头作为设备」的生命周期管理。

### 1.2 部署规模

| 项目 | 规格 |
|------|------|
| 摄像头总数 | **4 路**（物理固定，无扩展计划） |
| 部署位置 | A/B/C/D 栋宿舍楼入口各 1 个 |
| 房型 | 固定枪机，朝向入口通道 |
| 分辨率 | 1080p (1920×1080) / 720p 可配 |
| 协议 | RTSP / ONVIF |
| 供电 | PoE |
| 抓拍内容 | 仅入口进出画面 |

### 1.3 本模块与感知层的分工

```
         感知层（Stream Gateway）                           业务层/主服务
┌─────────────────────────────┐          ┌──────────────────────────┐
│                             │          │                          │
│  RTSP 拉流                  │          │  摄像头 CRUD（增删改查）   │
│  FFmpeg 解码                │          │  摄像头状态监控           │
│  动态抽帧                   │          │  摄像头-楼栋关系绑定       │
│  JPEG 编码                  │          │  摄像头配置下发            │
│  → Kafka t_dorm_frame      │  共享    │  摄像头分组维护            │
│                             │  数据库   │  抓拍历史查询              │
│  自身健康状态报告            │◄────────►│  SNMP/HTTP 检测摄像头连通   │
│  (health API)               │          │  摄像头离线告警           │
│                             │          │  查看当前画面(可配)        │
└─────────────────────────────┘          └──────────────────────────┘
```

**谁做什么**:

| 工作 | 负责人 | 说明 |
|------|--------|------|
| RTSP 拉流 + 解码 + 抽帧 | 感知层 (Go Stream Gateway) | 真正的音视频处理 |
| Kafka 推送 t_dorm_frame | 感知层 (Go Stream Gateway) | 帧数据 |
| 人脸检测/识别/方向 | 感知层 (Python Face Recognition) | AI 推理 |
| Kafka 推送 t_dorm_event | 感知层 (Python Face Recognition) | 事件 |
| **摄像头设备注册** | **本模块 (SpringBoot)** | 录入摄像头信息 |
| **摄像头状态监控** | **本模块 (SpringBoot)** | 定期检测连通性 |
| **摄像头配置管理** | **本模块 (SpringBoot)** | 管理 RTSP 地址等 |
| **摄像头-楼栋绑定** | **本模块 (SpringBoot)** | 管理部署关系 |
| **摄像头离线告警** | **本模块 (SpringBoot)** | 长时间无事件 → 告警 |
| **抓拍历史查询** | **本模块 (SpringBoot)** | 按摄像头/时间检索 |

### 1.4 数据流

```
┌──────────────┐    写入状态      ┌──────────────────────┐
│ Stream       │──────────────►  │  dorm_camera          │
│ Gateway      │  (health API)   │  (摄像头配置/状态表)   │
│ (4个实例)     │                │                        │
└──────────────┘                │  ┌──────────────────┐ │
                                │  │ camera_id        │ │
┌──────────────┐    导出状态      │  │ building         │ │
│ Stream       │──────────────►  │  │ rtsp_url         │ │
│ Gateway      │  (周期性pull)   │  │ status(online/off│ │
│ (本模块拉取)  │                │  │ last_heartbeat   │ │
└──────────────┘                │  │ fps_current      │ │
                                │  │ total_frames     │ │
┌──────────────┐                │  │ config (JSON)    │ │
│ 管理员/前端   │──CRUD─────────►  └──────────────────┘ │
│             │◄──状态/抓拍────  └──────────────────────┘
└──────────────┘
```

---

## 2. 协作边界

### 2.1 你依赖感知层提供的

Stream Gateway 需要暴露一个 HTTP health API，让本模块定时拉取摄像头运行状态：

```
GET http://stream-gateway:8080/health

Response:
{
  "cameras": [
    {
      "camera_id": "cam-a",
      "building": "A",
      "connected": true,
      "fps": 4.8,
      "last_frame_time": "2026-05-15T14:30:00+08:00",
      "frames_sent": 12345,
      "uptime_seconds": 86400
    }
  ]
}
```

> 感知层已实现此接口，本模块只需定时（如每 30 秒）拉取并更新到数据库。

### 2.2 感知层依赖你提供的

本模块维护的 `dorm_camera` 表中的 RTSP 地址，感知层的 `config.yaml` 可以从中同步（可选，目前为本地配置）：

| 方式 | 说明 | 推荐 |
|------|------|------|
| 感知层本地配置 | Stream Gateway 从本地 config.yaml 读取 | ⭐ MVP 阶段推荐 |
| 本模块 API 下发 | Stream Gateway 启动时从本模块拉取 | 后续增强 |

> MVP 阶段感知层使用本地 config.yaml 配置摄像头。本模块的 `dorm_camera` 表作为管理后台的配置视图，暂不下发。

### 2.3 摄像头离线判定

| 判定方式 | 说明 | 由谁判定 |
|---------|------|---------|
| **Stream Gateway 断连** | RTSP 连接断开 | 感知层判定，health API 报告 |
| **超时无事件** | 某摄像头超过 N 分钟无 `t_dorm_event` 产生 | **本模块判定**（消费延迟检测） |
| **Health API 无响应** | Stream Gateway 进程挂了 | **本模块判定**（HTTP 超时） |

---

## 3. 功能清单

### 3.1 摄像头设备 CRUD (F-CAM-001) | P0

| 操作 | 说明 | 约束 |
|------|------|------|
| 新增摄像头 | 录入 ID、名称、楼栋、RTSP 地址等 | 最多 4 路 |
| 编辑摄像头 | 修改 RTSP 地址、分辨率、备注等 | |
| 删除摄像头 | 删除设备记录 | 物理删除或标记删除 |
| 查询摄像头列表 | 支持按楼栋/状态筛选 | |
| 查询单个摄像头详情 | 完整信息 + 实时状态 | |

#### 字段设计

| 字段 | 类型 | 示例 | 说明 |
|------|------|------|------|
| camera_id | string | `cam-a` | 唯一 ID |
| name | string | `A栋入口摄像头` | 显示名称 |
| building | string | `A` | 所在楼栋 |
| rtsp_url | string | `rtsp://admin:pwd@192.168.1.101:554/stream1` | RTSP 地址 |
| status | enum | `online` / `offline` / `unknown` | 当前状态 |
| direction | enum | `entry` | 用途（目前只有入口） |
| resolution | string | `1280x720` | 分辨率 |
| last_heartbeat | datetime | | 最近一次心跳 |
| enabled | bool | true | 是否启用 |
| remark | string | | 备注 |

### 3.2 摄像头状态监控 (F-CAM-002) | P0

#### 监控周期

```
定时任务 (默认每 30 秒)
    │
    ├─ HTTP GET → Stream Gateway health API
    │      │
    │      ├─ 成功 → 更新 dorm_camera 状态：
    │      │     status=online, fps=4.8, last_heartbeat=now
    │      │
    │      └─ 失败(超时/拒绝) → 
    │            status=offline, 连续失败计数++
    │            连续失败 ≥ 3 次 → 触发离线告警
    │
    └─ 检查 Kafka t_dorm_event 消费延迟
           │
           └─ 某摄像头 > 5 分钟无事件 → 
                 标记为 "idle" 状态（有人但画面静止/摄像头卡住）
```

#### 状态定义

| 状态 | 含义 | 判定条件 |
|------|------|---------|
| `online` | 正常运行 | Health API 连通，fps > 0 |
| `offline` | 离线/断连 | Health API 不可达 |
| `idle` | 无事件产生 | Health 连通但 > 5min 无 t_dorm_event |
| `unknown` | 状态未知 | 刚注册或数据缺失 |

### 3.3 摄像头配置管理 (F-CAM-003) | P1

通过本模块管理摄像头级配置，修改后通知感知层（或由感知层轮询）：

| 配置项 | 类型 | 说明 |
|--------|------|------|
| `rtsp_url` | string | RTSP 拉流地址 |
| `fps_day` | int | 白天抽帧帧率 |
| `fps_night` | int | 夜间抽帧帧率 |
| `resolution` | string | 画面分辨率 |
| `jpeg_quality` | int | JPEG 压缩质量 1-100 |
| `roi_line_x` | float | ROI 判定线位置 0-1 |

> MVP 阶段配置变更需手动重启 Stream Gateway，后续可通过配置中心实现热更新。

### 3.4 摄像头分组/楼栋绑定 (F-CAM-004) | P1

系统固定为 4 栋楼 4 个摄像头，但支持：

| 场景 | 说明 |
|------|------|
| 更换摄像头 | 某栋楼摄像头故障，更换新设备 → 更新 RTSP 地址 |
| 调整监控方向 | 入口改为其它朝向 → 调整 ROI 线参数 |
| 临时停用 | 某栋楼维护 → 禁用该摄像头，不影响查宿统计 |

### 3.5 抓拍历史查看 (F-CAM-005) | P2

按摄像头维度查看历史抓拍：

- 从 `dorm_entry_exit_event` 的 `face_snapshot_url` 溯源
- 按摄像头 ID + 时间范围检索
- 按楼栋维度汇总抓拍次数

### 3.6 摄像头离线告警 (F-CAM-006) | P1

| 告警类型 | 条件 | 说明 |
|---------|------|------|
| 摄像头离线 | Health API 连续 3 次不可达 | 网络/电源/设备故障 |
| 摄像头卡顿 | fps < 1 持续 > 30 秒 | 网络带宽不足 |
| 摄像头无事件 | > 5 分钟无 t_dorm_event | 画面静止或系统异常 |
| 磁盘/存储异常 | 抓拍存储不可用 | 见主进程对接告警 |

---

## 4. REST API 设计

### 4.1 API 列表

| 方法 | 路径 | 说明 | 优先级 |
|------|------|------|--------|
| **摄像头管理** | | | |
| GET | `/api/dormitory/cameras` | 获取所有摄像头列表 | P0 |
| GET | `/api/dormitory/cameras/{cameraId}` | 获取单个摄像头详情 | P0 |
| POST | `/api/dormitory/cameras` | 新增摄像头设备 | P0 |
| PUT | `/api/dormitory/cameras/{cameraId}` | 更新摄像头配置 | P0 |
| DELETE | `/api/dormitory/cameras/{cameraId}` | 删除摄像头 | P1 |
| **状态监控** | | | |
| GET | `/api/dormitory/cameras/status` | 所有摄像头实时状态 | P0 |
| GET | `/api/dormitory/cameras/status?building=A` | 按楼栋查询状态 | P0 |
| **抓拍查看** | | | |
| GET | `/api/dormitory/cameras/{cameraId}/snapshots` | 按摄像头查询抓拍 | P2 |
| GET | `/api/dormitory/snapshots` | 全局抓拍检索 | P2 |
| **配置** | | | |
| GET | `/api/dormitory/cameras/{cameraId}/config` | 获取摄像头级配置 | P1 |
| PUT | `/api/dormitory/cameras/{cameraId}/config` | 更新摄像头级配置 | P1 |

### 4.2 API 详细设计

#### GET /api/dormitory/cameras

获取所有摄像头及其状态。

```json
Response 200:
{
  "total": 4,
  "cameras": [
    {
      "cameraId": "cam-a",
      "name": "A栋入口摄像头",
      "building": "A",
      "status": "online",
      "direction": "entry",
      "resolution": "1280x720",
      "rtspUrl": "rtsp://admin:****@192.168.1.101:554/stream1",
      "fpsCurrent": 4.8,
      "totalFrames": 123456,
      "lastHeartbeat": "2026-05-15T14:30:00+08:00",
      "lastEventTime": "2026-05-15T14:29:30+08:00",
      "enabled": true
    }
  ]
}
```

#### GET /api/dormitory/cameras/status

状态看板专用，轻量级。

```json
Response 200:
{
  "buildings": [
    {
      "building": "A",
      "cameraId": "cam-a",
      "status": "online",
      "fps": 4.8,
      "uptimeSeconds": 86400,
      "todayEvents": 1250,
      "todayStrangers": 2,
      "lastEventTime": "2026-05-15T14:29:30+08:00"
    }
  ],
  "summary": {
    "total": 4,
    "online": 3,
    "offline": 1,
    "idle": 0,
    "unknown": 0
  }
}
```

#### POST /api/dormitory/cameras

新增摄像头。

```json
Request:
{
  "cameraId": "cam-e",
  "name": "E栋入口摄像头",
  "building": "E",
  "rtspUrl": "rtsp://admin:password@192.168.1.105:554/stream1",
  "direction": "entry",
  "resolution": "1280x720",
  "remark": "备用"
}

Response 201:
{
  "code": 200,
  "text": "success",
  "data": {
    "cameraId": "cam-e",
    "status": "unknown"
  }
}
```

#### PUT /api/dormitory/cameras/{cameraId}

更新摄像头信息。

```json
Request:
{
  "rtspUrl": "rtsp://admin:newpassword@192.168.1.101:554/stream1",
  "remark": "已更换摄像头"
}
```

#### GET /api/dormitory/cameras/{cameraId}/snapshots

按摄像头查询历史抓拍。

```json
Request Params:
  startTime: DateTime (optional)
  endTime: DateTime   (optional)
  eventType: String   (entry/exit, optional)
  isStranger: Boolean (optional)
  page: Integer       (default 1)
  size: Integer       (default 20)

Response 200:
{
  "cameraId": "cam-a",
  "building": "A",
  "total": 5840,
  "page": 1,
  "size": 20,
  "records": [
    {
      "eventId": "evt_20260515_001",
      "studentId": "S2024001",
      "studentName": "张三",
      "eventType": "entry",
      "faceSnapshotUrl": "https://minio:9000/snapshots/xxx.jpg",
      "confidence": 0.95,
      "timestamp": "2026-05-15T22:15:00+08:00"
    }
  ]
}
```

---

## 5. 数据模型

### 5.1 摄像头信息表 (`dorm_camera`)

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | BIGINT | PK | 自增主键 |
| `camera_id` | VARCHAR(32) | UNIQUE, NOT NULL | 摄像头唯一 ID |
| `name` | VARCHAR(64) | NOT NULL | 显示名称 |
| `building` | VARCHAR(8) | NOT NULL | 所在楼栋 A/B/C/D |
| `rtsp_url` | VARCHAR(512) | NOT NULL | RTSP 拉流地址 |
| `direction` | VARCHAR(16) | DEFAULT 'entry' | 监控方向 |
| `resolution` | VARCHAR(16) | DEFAULT '1280x720' | 分辨率 |
| `status` | VARCHAR(16) | DEFAULT 'unknown' | online/offline/idle/unknown |
| `fps_current` | DECIMAL(5,2) | DEFAULT 0 | 当前帧率 |
| `total_frames` | BIGINT | DEFAULT 0 | 累计帧数 |
| `last_heartbeat` | DATETIME | | 最近心跳时间 |
| `last_event_time` | DATETIME | | 最近事件时间 |
| `enabled` | BOOLEAN | DEFAULT true | 是否启用 |
| `config_json` | TEXT | | 摄像头级配置(JSON) |
| `remark` | VARCHAR(256) | | 备注 |
| `created_at` | DATETIME | NOT NULL | 创建时间 |
| `updated_at` | DATETIME | NOT NULL | 更新时间 |

> **索引**: `idx_building` (building), `idx_status` (status)

### 5.2 摄像头日志表 (`dorm_camera_log`)

记录摄像头状态变更历史。

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | BIGINT | PK | 自增主键 |
| `camera_id` | VARCHAR(32) | NOT NULL | 摄像头 ID |
| `building` | VARCHAR(8) | NOT NULL | 楼栋 |
| `status_from` | VARCHAR(16) | | 变更前状态 |
| `status_to` | VARCHAR(16) | NOT NULL | 变更后状态 |
| `reason` | VARCHAR(128) | | 变更原因 |
| `fps_at_time` | DECIMAL(5,2) | | 变更时帧率 |
| `created_at` | DATETIME | NOT NULL | 记录时间 |

> **索引**: `idx_camera_ts` (camera_id, created_at)

### 5.3 ER 关系

```
dorm_camera (1) ──── (N) dorm_camera_log
    │
    │ building 关联
    ▼
dorm_student_assignment (building)
    │
    │ camera_id 关联（通过 event 表）
    ▼
dorm_entry_exit_event (camera_id)
```

---

## 6. 配置清单

### 6.1 系统配置

**存储位置**: `dorm_config` 表（与主进程对接共用配置表）

| 配置键 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `camera.health_check.interval_sec` | int | `30` | 摄像头健康检查间隔（秒） |
| `camera.health_check.timeout_sec` | int | `5` | HTTP 健康检查超时 |
| `camera.offline.alert_threshold` | int | `3` | 连续检查失败 N 次后触发离线告警 |
| `camera.idle.threshold_min` | int | `5` | 超过 N 分钟无事件标记为 idle |
| `camera.snapshot.retention_days` | int | `30` | 抓拍快照保留天数 |
| `camera.max_count` | int | `4` | 系统最大摄像头数 |

### 6.2 摄像头级配置

**存储位置**: `dorm_camera.config_json` 字段（JSON）

```json
{
  "fps_day": 5,
  "fps_night": 1,
  "jpeg_quality": 80,
  "roi_line_x": 0.5,
  "motion_threshold": 0.05,
  "night_mode": {
    "enabled": true,
    "start_hour": 22,
    "end_hour": 6,
    "clahe_clip_limit": 2.0
  }
}
```

> 此配置对**感知层**生效。MVP 阶段感知层本地配置优先，后续可通过本模块下发。

---

## 7. 监控看板指标

本模块提供给前端展示的摄像头看板数据：

### 7.1 总览卡片

| 指标 | 说明 |
|------|------|
| 总摄像头数 | 4（固定） |
| 在线数 | Health API 可达 |
| 离线数 | Health API 不可达 |
| 今日总事件 | 所有摄像头 t_dorm_event 之和 |
| 今日陌生人 | 所有摄像头陌生人事件之和 |
| 最后心跳时间 | 最近一次任意摄像头心跳 |

### 7.2 单摄像头卡片

```
┌──────────────────────────────────┐
│  A栋入口摄像头     ● 在线          │
│                                  │
│  实时帧率:  4.8 fps              │
│  今日事件:  1,250 次              │
│  今日陌生人: 2 次                  │
│  最后抓拍:  14:29:30 (张三 进入) │
│  运行时长:  24h                   │
│                                  │
│  [查看抓拍]  [编辑配置]  [状态日志] │
└──────────────────────────────────┘
```

### 7.3 异常时间线

```
14:00  ──── cam-a 在线 (4.8fps)
14:05  ──── cam-b 离线告警 ⚠️
14:06  ──── cam-b 自动恢复
14:15  ──── cam-c idle (>5min无事件)
```

---

## 8. 附录

### 8.1 联动告警：摄像头离线

当本模块检测到摄像头离线时，推送到 `t_dorm_alert`：

```json
{
  "alertId": "alert_offline_cam_a",
  "type": "SYSTEM",
  "building": "A",
  "description": "A栋入口摄像头离线 (连续3次健康检查失败)",
  "severity": "critical",
  "details": {
    "cameraId": "cam-a",
    "lastHeartbeat": "2026-05-15T14:00:00+08:00",
    "failCount": 3
  },
  "timestamp": "2026-05-15T14:01:30+08:00"
}
```

### 8.2 感知层与本模块的状态同步流程

```
Stream Gateway                      本模块
     │                                 │
     │  每 30s HTTP GET /health        │
     │◄──────────────────────────────  │  ← 本模块主动拉取
     │  { status: "ok", cameras: [] }  │
     │──────────────────────────────►  │
     │                                 │
     │  更新 dorm_camera 表            │
     │  状态变化时写入 dorm_camera_log  │
     │  离线 → 触发告警                 │
     │                                 │
```

### 8.3 与感知层的接口约定

Stream Gateway 需提供：

| 端点 | 说明 | 必须提供 |
|------|------|---------|
| `GET /health` | 摄像头状态 + 指标 | ✅ **必须** |

Response 格式（感知层已实现）：

```json
{
  "status": "UP",
  "cameras": [
    {
      "camera_id": "cam-a",
      "building": "A",
      "connected": true,
      "fps": 4.8,
      "last_frame_time": "2026-05-15T14:30:00+08:00",
      "frames_sent": 12345,
      "uptime_seconds": 86400
    }
  ]
}
```

### 8.4 版本历史

| 版本 | 日期 | 变更说明 |
|------|------|---------|
| v1.0 | 2026-05-15 | 初稿 — 摄像头设备管理功能 PRD，独立于感知层的流处理逻辑 |

---

> **本 PRD 对应功能**：摄像头设备管理、状态监控、配置管理、抓拍查看  
> **注意**: 本模块的依赖数据来源于感知层。
