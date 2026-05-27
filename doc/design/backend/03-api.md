# Dormitory Service — API 设计

> **文档归属**: 后端开发 → API 设计  
> **对应 PRD**: PRD-004 (主进程对接), PRD-005 (摄像头功能实现)  
> **版本**: v1.0 · **更新**: 2026-05-15  

---

## 目录

1. [通用规范](#1-通用规范)
2. [查宿统计 API](#2-查宿统计-api)
3. [人员状态 API](#3-人员状态-api)
4. [事件查询 API](#4-事件查询-api)
5. [告警 API](#5-告警-api)
6. [配置管理 API](#6-配置管理-api)
7. [数据同步 API](#7-数据同步-api)
8. [统计报表 API](#8-统计报表-api)
9. [摄像头管理 API](#9-摄像头管理-api)
10. [健康检查 API](#10-健康检查-api)
11. [错误码速查](#11-错误码速查)

---

## 1. 通用规范

### 1.1 API 前缀

| 部署阶段 | URL 前缀 |
|---------|---------|
| Phase 1 独立部署 | `/api/dormitory/*` |
| Phase 3 接入主进程 | `/api/sims/dormitory/*` |

控制器层面通过 `server.servlet.context-path` 统一管理，不硬编码。

### 1.2 统一响应格式

```json
// 成功
{
  "code": 200,
  "text": "success",
  "data": { ... },
  "timestamp": "2026-05-15T23:00:00+08:00",
  "requestId": "req_abc123"
}

// 错误
{
  "code": 400,
  "text": "请求参数不合法",
  "data": null,
  "timestamp": "2026-05-15T23:00:00+08:00",
  "requestId": "req_abc123"
}
```

### 1.3 分页规范

```json
// 请求参数
?page=1&size=20

// 响应格式
{
  "code": 200,
  "text": "success",
  "data": {
    "records": [ ... ],
    "total": 440,
    "page": 1,
    "size": 20,
    "pages": 22
  }
}
```

| 参数 | 默认值 | 约束 |
|------|--------|------|
| page | 1 | ≥ 1 |
| size | 20 | 1 ~ 100 (max) |

### 1.4 日期时间格式

- 请求参数: `yyyy-MM-dd` (日期), `yyyy-MM-dd'T'HH:mm:ss` (日期时间)
- 响应字段: `2026-05-15T23:00:00+08:00` (ISO 8601 with timezone)
- 时区: `Asia/Shanghai` (UTC+8)

### 1.5 认证

Phase 1 独立部署阶段使用简单 Token 认证：

```
Authorization: Bearer <token>
```

Phase 3 复用主进程的 Spring Security / JWT。

---

## 2. 查宿统计 API

### 2.1 获取今日查宿概览

```
GET /api/dormitory/nightly-report/today
```

**Response 200:**

```json
{
  "code": 200,
  "text": "success",
  "data": {
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
      },
      {
        "building": "B",
        "totalStudents": 96,
        "presentCount": 88,
        "absentCount": 8,
        "lateReturnCount": 3,
        "strangerCount": 0,
        "occupancyRate": 91.7
      }
    ],
    "summary": {
      "occupancyRate": 88.2,
      "abnormalCount": 75,
      "abnormalRate": 17.0
    },
    "generatedTime": "2026-05-15T23:05:00+08:00"
  }
}
```

### 2.2 按日期查询查宿

```
GET /api/dormitory/nightly-report/{date}
GET /api/dormitory/nightly-report/{date}/building/{building}
GET /api/dormitory/nightly-report/{date}/building/{building}/room/{room}
```

**参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| date | path | 是 | 日期 yyyy-MM-dd |
| building | path | 否 | 楼栋 A/B/C/D |
| room | path | 否 | 房间号 |

**GET .../building/{building} Response:**

```json
{
  "code": 200,
  "text": "success",
  "data": {
    "date": "2026-05-15",
    "building": "A",
    "totalCount": 120,
    "presentCount": 108,
    "absentCount": 12,
    "lateReturnCount": 5,
    "floors": [
      {
        "floor": 3,
        "totalCount": 24,
        "presentCount": 22,
        "absentCount": 2,
        "rooms": [
          {
            "room": "A-301",
            "totalCount": 4,
            "presentCount": 3,
            "absentCount": 1,
            "students": [
              {
                "studentId": "S2024001",
                "studentName": "张三",
                "status": "present",
                "entryTime": "2026-05-15T22:15:00+08:00",
                "isLateReturn": true
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
}
```

### 2.3 手动触发查宿统计

```
POST /api/dormitory/nightly-report/trigger

Request:
{
  "building": "A",          // 可选，不传则全量
  "date": "2026-05-15"      // 可选，默认今日
}

Response 200:
{
  "code": 200,
  "text": "success",
  "data": {
    "message": "查宿统计已触发",
    "taskId": "report_20260515_A",
    "status": "PENDING"
  }
}
```

---

## 3. 人员状态 API

### 3.1 查询所有人员状态

```
GET /api/dormitory/students/status?building=A&room=A-301&status=in&page=1&size=20
```

**参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| building | query | 否 | 楼栋筛选 |
| room | query | 否 | 房间筛选 |
| status | query | 否 | in/out/unknown |
| page | query | 否 | 页码，默认 1 |
| size | query | 否 | 页大小，默认 20 |

**Response 200:**

```json
{
  "code": 200,
  "text": "success",
  "data": {
    "total": 120,
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
}
```

### 3.2 查询单个学生状态

```
GET /api/dormitory/students/{studentId}/status
```

**Response 200:**

```json
{
  "code": 200,
  "text": "success",
  "data": {
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
}
```

### 3.3 查询单个学生进出记录

```
GET /api/dormitory/students/{studentId}/events?startTime=2026-05-01T00:00:00&endTime=2026-05-15T23:59:59&page=1&size=20
```

**Response 200:**

```json
{
  "code": 200,
  "text": "success",
  "data": {
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
      },
      {
        "eventId": "evt_20260515_002",
        "building": "A",
        "eventType": "exit",
        "confidence": 0.95,
        "timestamp": "2026-05-15T07:30:00+08:00"
      }
    ]
  }
}
```

---

## 4. 事件查询 API

### 4.1 进出事件列表

```
GET /api/dormitory/events?building=A&eventType=entry&isStranger=false&startTime=...&endTime=...&page=1&size=20
```

**参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| building | query | 否 | 楼栋 |
| eventType | query | 否 | entry/exit |
| studentId | query | 否 | 学号 |
| isStranger | query | 否 | 是否陌生人 |
| startTime | query | 否 | 开始时间 |
| endTime | query | 否 | 结束时间 |
| page | query | 否 | 页码，默认 1 |
| size | query | 否 | 页大小，默认 20 |

**Response 200:**

```json
{
  "code": 200,
  "text": "success",
  "data": {
    "total": 5840,
    "page": 1,
    "size": 20,
    "records": [
      {
        "eventId": "evt_20260515_001",
        "cameraId": "cam-a",
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
}
```

---

## 5. 告警 API

### 5.1 告警列表

```
GET /api/dormitory/alerts?building=A&severity=high&isResolved=false&page=1&size=20
```

**参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| building | query | 否 | 楼栋 |
| alertType | query | 否 | STRANGER_ENTRY/LONG_ABSENT/... |
| severity | query | 否 | low/medium/high/critical |
| isResolved | query | 否 | 是否已处理 |
| startTime | query | 否 | 开始时间 |
| endTime | query | 否 | 结束时间 |
| page | query | 否 | 页码 |
| size | query | 否 | 页大小 |

**Response 200:**

```json
{
  "code": 200,
  "text": "success",
  "data": {
    "total": 28,
    "page": 1,
    "size": 20,
    "records": [
      {
        "alertId": "alert_001",
        "alertType": "STRANGER_ENTRY",
        "building": "A",
        "severity": "high",
        "description": "陌生人进入 A 栋宿舍楼",
        "faceSnapshotUrl": "https://minio:9000/snapshots/stranger.jpg",
        "isRead": false,
        "isResolved": false,
        "occurredAt": "2026-05-15T14:30:00+08:00"
      }
    ]
  }
}
```

### 5.2 标记告警已处理

```
PUT /api/dormitory/alerts/{alertId}/resolve

Response 200:
{
  "code": 200,
  "text": "success",
  "data": {
    "alertId": "alert_001",
    "resolved": true
  }
}
```

---

## 6. 配置管理 API

### 6.1 获取全部配置

```
GET /api/dormitory/config

Response 200:
{
  "code": 200,
  "text": "success",
  "data": {
    "nightly_report.trigger_time": "23:00",
    "nightly_report.timezone": "Asia/Shanghai",
    "late_return.threshold": "22:00",
    "absent.alert_hours": "24",
    "alert.stranger.enabled": "true",
    "kafka.consumer.topic": "t_dorm_event",
    ...
  }
}
```

### 6.2 更新配置

```
PUT /api/dormitory/config

Request:
{
  "nightly_report.trigger_time": "22:30",
  "late_return.threshold": "22:30"
}

Response 200:
{
  "code": 200,
  "text": "success",
  "data": {
    "message": "配置更新成功",
    "updated": ["nightly_report.trigger_time", "late_return.threshold"],
    "failed": []
  }
}
```

### 6.3 配置变更历史

```
GET /api/dormitory/config/history?page=1&size=20

Response 200:
{
  "code": 200,
  "text": "success",
  "data": {
    "total": 58,
    "page": 1,
    "size": 20,
    "records": [
      {
        "configKey": "nightly_report.trigger_time",
        "oldValue": "23:00",
        "newValue": "22:30",
        "updatedBy": "admin",
        "updatedAt": "2026-05-15T10:00:00+08:00"
      }
    ]
  }
}
```

---

## 7. 数据同步 API

### 7.1 手动触发学管同步

```
POST /api/dormitory/sync/students

Response 200:
{
  "code": 200,
  "text": "success",
  "data": {
    "message": "同步任务已触发",
    "syncId": "sync_20260515_001",
    "status": "IN_PROGRESS"
  }
}
```

### 7.2 同步状态查询

```
GET /api/dormitory/sync/status

Response 200:
{
  "code": 200,
  "text": "success",
  "data": {
    "lastSyncTime": "2026-05-15T12:00:00+08:00",
    "lastSyncStatus": "SUCCESS",
    "lastSyncCount": 440,
    "lastError": null,
    "totalSyncCount": 58,
    "nextSyncTime": "2026-05-15T13:00:00+08:00"
  }
}
```

---

## 8. 统计报表 API

### 8.1 今日概览

```
GET /api/dormitory/stats/overview

Response 200:
{
  "code": 200,
  "text": "success",
  "data": {
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
        "occupancyRate": 90.0
      }
    ],
    "trend": [
      { "date": "2026-05-14", "occupancyRate": 87.5 },
      { "date": "2026-05-13", "occupancyRate": 89.1 }
    ]
  }
}
```

### 8.2 历史趋势

```
GET /api/dormitory/stats/trend?days=30&building=A

Response 200:
{
  "code": 200,
  "text": "success",
  "data": {
    "days": 30,
    "startDate": "2026-04-16",
    "endDate": "2026-05-15",
    "building": "A",
    "series": [
      {
        "date": "2026-05-15",
        "presentCount": 108,
        "absentCount": 12,
        "lateReturnCount": 5,
        "occupancyRate": 90.0
      }
    ],
    "summary": {
      "avgOccupancyRate": 87.3,
      "maxOccupancyRate": 92.1,
      "minOccupancyRate": 82.5,
      "totalAbsentPersonDays": 360
    }
  }
}
```

### 8.3 导出查宿数据

```
GET /api/dormitory/stats/export?date=2026-05-15&building=A&format=csv

Response 200:
  Content-Type: text/csv
  Content-Disposition: attachment; filename="nightly-report-2026-05-15.csv"

楼栋,房间,学号,姓名,班级,状态,进入时间,是否晚归
A,A-301,S2024001,张三,计算机2101班,present,22:15,是
A,A-301,S2024003,王五,计算机2101班,absent,,否
```

---

## 9. 摄像头管理 API

### 9.1 获取所有摄像头

```
GET /api/dormitory/cameras

Response 200:
{
  "code": 200,
  "text": "success",
  "data": {
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
        "enabled": true,
        "config": {
          "fpsDay": 5,
          "fpsNight": 1,
          "jpegQuality": 80,
          "roiLineX": 0.5
        }
      }
    ]
  }
}
```

### 9.2 获取单个摄像头详情

```
GET /api/dormitory/cameras/{cameraId}
```

Response 同上单条。

### 9.3 新增摄像头

```
POST /api/dormitory/cameras

Request:
{
  "cameraId": "cam-a",
  "name": "A栋入口摄像头",
  "building": "A",
  "rtspUrl": "rtsp://admin:password@192.168.1.101:554/stream1",
  "direction": "entry",
  "resolution": "1280x720",
  "remark": ""
}

Response 201:
{
  "code": 200,
  "text": "success",
  "data": {
    "cameraId": "cam-a",
    "status": "unknown"
  }
}
```

### 9.4 更新摄像头

```
PUT /api/dormitory/cameras/{cameraId}

Request:
{
  "rtspUrl": "rtsp://admin:newpassword@192.168.1.101:554/stream1",
  "remark": "已更换摄像头"
}

Response 200:
{
  "code": 200,
  "text": "success",
  "data": {
    "cameraId": "cam-a",
    "updated": true
  }
}
```

### 9.5 删除摄像头

```
DELETE /api/dormitory/cameras/{cameraId}

Response 200:
{
  "code": 200,
  "text": "success",
  "data": null
}
```

### 9.6 摄像头实时状态

```
GET /api/dormitory/cameras/status?building=A

Response 200:
{
  "code": 200,
  "text": "success",
  "data": {
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
      "idle": 0
    }
  }
}
```

### 9.7 按摄像头查询抓拍

```
GET /api/dormitory/cameras/{cameraId}/snapshots?startTime=...&endTime=...&page=1&size=20

Response 200:
{
  "code": 200,
  "text": "success",
  "data": {
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
}
```

---

## 10. 健康检查 API

### 10.1 健康检查

```
GET /api/dormitory/health

Response 200:
{
  "code": 200,
  "text": "success",
  "data": {
    "status": "UP",
    "timestamp": "2026-05-15T23:00:00+08:00",
    "components": {
      "redis": { "status": "UP", "latencyMs": 2 },
      "mariadb": { "status": "UP", "latencyMs": 5 },
      "kafka": {
        "status": "UP",
        "lag": 0,
        "lastEventTime": "2026-05-15T22:59:00+08:00"
      },
      "simsApi": {
        "status": "UP",
        "lastSync": "2026-05-15T12:00:00+08:00"
      },
      "streamGateway": {
        "status": "UP",
        "camerasOnline": 4,
        "lastHeartbeat": "2026-05-15T14:30:00+08:00"
      }
    },
    "metrics": {
      "totalStudents": 440,
      "activeStudents": 440,
      "todayEvents": 5840,
      "todayStrangers": 3
    }
  }
}
```

---

## 11. 错误码速查

| HTTP 状态码 | 业务错误码 | 说明 | 触发场景 |
|-------------|-----------|------|---------|
| 400 | INVALID_PARAMETER | 请求参数校验失败 | building 不是 A/B/C/D |
| 400 | BUILDING_INVALID | 楼栋参数不合法 | 传入了 E 栋 |
| 400 | CAMERA_LIMIT_EXCEEDED | 摄像头数量已达上限 | 超过可配置的最大摄像头数 |
| 401 | UNAUTHORIZED | 认证失败 | Token 无效/过期 |
| 404 | NOT_FOUND | 资源不存在 | studentId 查无此人 |
| 404 | STUDENT_NOT_FOUND | 学生未找到 | 同步数据中无此学号 |
| 409 | CONFLICT | 数据冲突 | 重复统计同一日期 |
| 409 | REPORT_ALREADY_EXISTS | 该日期已存在查宿统计 | 不可重复统计 |
| 409 | SYNC_IN_PROGRESS | 同步任务执行中 | 正在同步时再次触发 |
| 422 | UNPROCESSABLE_ENTITY | 业务规则校验失败 | 更新配置值不合法 |
| 429 | TOO_MANY_REQUESTS | 请求频率超限 | 频繁调用 API |
| 500 | INTERNAL_ERROR | 服务器内部错误 | 未捕获的异常 |
| 503 | SERVICE_UNAVAILABLE | 服务暂不可用 | 数据库/Kafka 离线 |

---

> **本文件属于**: `doc/design/backend/03-api.md`  
> **面向读者**: Java 后端开发 + 前端对接（搭档）  
> **累计端点**: 22 个 REST API
