# Dormitory Service — 数据库设计

> **文档归属**: 后端开发 → 数据库设计  
> **对应 PRD**: PRD-004 (主进程对接), PRD-005 (摄像头功能实现)  
> **版本**: v1.0 · **更新**: 2026-05-15  

---

## 目录

1. [ER 关系图](#1-er-关系图)
2. [表定义总览](#2-表定义总览)
3. [DDL 完整定义](#3-ddl-完整定义)
4. [索引设计](#4-索引设计)
5. [Flyway 迁移](#5-flyway-迁移)
6. [Redis Key 设计](#6-redis-key-设计)
7. [数据量估算](#7-数据量估算)
8. [存储与清理策略](#8-存储与清理策略)

---

## 1. ER 关系图

```
┌──────────────────────┐       ┌──────────────────────────┐
│  dorm_camera         │       │  dorm_student_assignment  │
│──────────────────────│       │──────────────────────────│
│ PK  id               │       │ PK  id                    │
│ UNIQUE camera_id     │       │ UNIQUE student_id         │
│     building         │ 1:1   │     building              │
│     rtsp_url         │──────►│     room                  │ ← 学管同步
│     status           │       │     class_name            │
│     fps_current      │       │     active                │
│     last_heartbeat   │       └──────────┬────────────────┘
└──────────┬───────────┘                  │
           │                              │ 1:1
           │                              │
           │ 1:N                 ┌────────▼────────────────┐
           │                     │  dorm_student_status     │
           ▼                     │─────────────────────────│
┌──────────────────────┐        │ PK  id                   │
│  dorm_camera_log     │        │ UNIQUE student_id        │
│──────────────────────│        │     is_in_dorm           │
│ PK  id               │        │     today_status         │
│     camera_id        │        │     last_entry_time      │
│     status_from      │        │     ...                  │
│     status_to        │        └─────────────────────────┘
│     reason           │
└──────────────────────┘                  │
                                          │ (关联 student_id)
┌─────────────────────────────────────────┴──────────────────────┐
│  dorm_entry_exit_event                                         │
│───────────────────────────────────────────────────────────────│
│ PK  id                                                          │
│ UNIQUE event_id         ← Kafka 幂等关键                        │
│     building              │                                      │
│     student_id            │                                      │
│     event_type (entry/exit)│                                     │
│     confidence             │                                      │
│     is_stranger            │                                      │
│     timestamp              │                                      │
│     camera_id              │                                      │
│     face_snapshot_url      │                                      │
└────────────────────────────────────────────────────────────────┘
         │                              │
         │                              │
         ▼                              ▼
┌──────────────────┐     ┌──────────────────────────┐
│ dorm_nightly_    │     │  dorm_stranger_record     │
│ report           │     │──────────────────────────│
│──────────────────│     │ PK  id                   │
│ PK  id           │     │     building              │
│ UNIQUE (date,    │     │     face_snapshot_url     │
│         building)│     │     confidence            │
│     present_     │     │     status                │
│     count        │     └──────────────────────────┘
│     absent_count │
│     ...          │           ┌──────────────────────┐
└────────┬─────────┘           │  dorm_alert_record   │
         │ 1:N                 │──────────────────────│
         ▼                     │ PK  id               │
┌──────────────────────┐       │ UNIQUE alert_id      │
│  dorm_nightly_detail │       │     alert_type        │
│──────────────────────│       │     severity          │
│ PK  id               │       │     description       │
│ FK  report_id        │       │     building          │
│     student_id       │       └──────────────────────┘
│     status           │
│     is_late_return   │       ┌──────────────────────┐
│     entry_time       │       │  dorm_config          │
└──────────────────────┘       │──────────────────────│
                               │ PK  id               │
┌──────────────────────┐       │ UNIQUE config_key    │
│  dorm_sync_log       │       │     config_value      │
│──────────────────────│       │     group_name        │
│ PK  id               │       │     description       │
│     sync_type        │       └──────────────────────┘
│     sync_status      │
│     total_count      │
└──────────────────────┘
```

---

## 2. 表定义总览

| # | 表名 | 说明 | 归属模块 | 数据量 |
|---|------|------|---------|--------|
| 1 | `dorm_student_assignment` | 学生宿舍分配（学管同步） | 主进程对接 | ~500 行 |
| 2 | `dorm_student_status` | 人员实时在校状态 | 主进程对接 | ~500 行 |
| 3 | `dorm_entry_exit_event` | 进出事件明细 | 主进程对接 | ~10,000 行/日 |
| 4 | `dorm_nightly_report` | 每晚查宿统计汇总 | 主进程对接 | 4 行/日 |
| 5 | `dorm_nightly_detail` | 查宿每人明细 | 主进程对接 | ~500 行/日 |
| 6 | `dorm_stranger_record` | 陌生人记录 | 主进程对接 | ~50 行/日 |
| 7 | `dorm_alert_record` | 告警记录 | 主进程对接 | ~100 行/日 |
| 8 | `dorm_config` | 系统动态配置 | 主进程对接 | ~30 行 |
| 9 | `dorm_sync_log` | 学管同步日志 | 主进程对接 | ~20 行/日 |
| 10 | `dorm_camera` | 摄像头设备信息 | 摄像头功能实现 | 4 行 |
| 11 | `dorm_camera_log` | 摄像头状态变更日志 | 摄像头功能实现 | ~50 行/日 |

---

## 3. DDL 完整定义

### 3.1 学生宿舍分配表 `dorm_student_assignment`

```sql
-- 从学管系统同步的宿舍数据，本服务只读
CREATE TABLE dorm_student_assignment (
    id              BIGINT          AUTO_INCREMENT PRIMARY KEY,
    student_id      VARCHAR(32)     NOT NULL UNIQUE        COMMENT '学号',
    student_name    VARCHAR(64)     NOT NULL               COMMENT '姓名',
    building        VARCHAR(8)      NOT NULL               COMMENT '宿舍楼栋 A/B/C/D',
    room            VARCHAR(16)     NOT NULL               COMMENT '房间号',
    class_name      VARCHAR(64)                             COMMENT '班级',
    grade           VARCHAR(32)                             COMMENT '年级',
    gender          VARCHAR(8)                              COMMENT '性别',
    phone           VARCHAR(20)                             COMMENT '联系电话',
    active          BOOLEAN         DEFAULT TRUE            COMMENT '是否在校住宿',
    sync_version    BIGINT          DEFAULT 0               COMMENT '同步版本号(乐观锁)',
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

    INDEX idx_building_room (building, room),
    INDEX idx_building (building),
    INDEX idx_active (active)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='学生宿舍分配表';
```

### 3.2 人员在校状态表 `dorm_student_status`

```sql
-- Redis 为主存储，MariaDB 做持久化备份
CREATE TABLE dorm_student_status (
    id                BIGINT        AUTO_INCREMENT PRIMARY KEY,
    student_id        VARCHAR(32)   NOT NULL UNIQUE          COMMENT '学号',
    student_name      VARCHAR(64)   NOT NULL                 COMMENT '姓名',
    building          VARCHAR(8)    NOT NULL                 COMMENT '所属楼栋',
    room              VARCHAR(16)   NOT NULL                 COMMENT '房间号',
    is_in_dorm        BOOLEAN       DEFAULT FALSE            COMMENT '是否在宿舍',
    last_entry_time   DATETIME                               COMMENT '最近进入时间',
    last_exit_time    DATETIME                               COMMENT '最近离开时间',
    today_status      VARCHAR(16)   DEFAULT 'unknown'        COMMENT '今日状态: in/out/unknown',
    today_entry_count INT           DEFAULT 0                COMMENT '今日进入次数',
    today_exit_count  INT           DEFAULT 0                COMMENT '今日离开次数',
    last_update       DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '最后更新时间',

    INDEX idx_building (building),
    INDEX idx_today_status (today_status),
    INDEX idx_is_in_dorm (is_in_dorm)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='人员在校状态表';
```

### 3.3 进出事件表 `dorm_entry_exit_event`

```sql
-- 核心流水表，大量写入，建议按日分区
CREATE TABLE dorm_entry_exit_event (
    id                  BIGINT        AUTO_INCREMENT PRIMARY KEY,
    event_id            VARCHAR(64)   NOT NULL UNIQUE        COMMENT '事件唯一ID(幂等)',
    camera_id           VARCHAR(32)                         COMMENT '摄像头ID',
    building            VARCHAR(8)    NOT NULL               COMMENT '楼栋',
    student_id          VARCHAR(32)                         COMMENT '学生学号(可为空=陌生人)',
    student_name        VARCHAR(64)                         COMMENT '学生姓名',
    event_type          VARCHAR(8)    NOT NULL               COMMENT 'entry/exit',
    confidence          DECIMAL(5,4)                        COMMENT '人脸识别置信度',
    face_snapshot_url   VARCHAR(512)                        COMMENT '抓拍快照URL',
    is_stranger         BOOLEAN       DEFAULT FALSE          COMMENT '是否陌生人',
    is_processed        BOOLEAN       DEFAULT TRUE           COMMENT '是否已被消费处理',
    timestamp           DATETIME      NOT NULL               COMMENT '事件时间',
    created_at          DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '记录创建时间',

    INDEX idx_building_ts (building, timestamp),
    INDEX idx_student_id (student_id),
    INDEX idx_event_type (event_type),
    INDEX idx_timestamp (timestamp),
    INDEX idx_stranger (is_stranger),
    INDEX idx_camera_ts (camera_id, timestamp)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='进出事件表';
```

#### 分区建议（事件表大表优化）

```sql
-- 按月分区，保留 6 个月
ALTER TABLE dorm_entry_exit_event
PARTITION BY RANGE (TO_DAYS(timestamp)) (
    PARTITION p202605 VALUES LESS THAN (TO_DAYS('2026-06-01')),
    PARTITION p202606 VALUES LESS THAN (TO_DAYS('2026-07-01')),
    PARTITION p202607 VALUES LESS THAN (TO_DAYS('2026-08-01')),
    PARTITION p202608 VALUES LESS THAN (TO_DAYS('2026-09-01')),
    PARTITION p202609 VALUES LESS THAN (TO_DAYS('2026-10-01')),
    PARTITION p202610 VALUES LESS THAN (TO_DAYS('2026-11-01')),
    PARTITION p_future VALUES LESS THAN MAXVALUE
);
```

### 3.4 每晚查宿统计表 `dorm_nightly_report`

```sql
CREATE TABLE dorm_nightly_report (
    id                  BIGINT        AUTO_INCREMENT PRIMARY KEY,
    report_date         DATE          NOT NULL               COMMENT '统计日期',
    building            VARCHAR(8)    NOT NULL               COMMENT '楼栋',
    total_count         INT           NOT NULL               COMMENT '应归人数',
    present_count       INT           NOT NULL               COMMENT '已归人数',
    absent_count        INT           NOT NULL               COMMENT '未归人数',
    late_return_count   INT           DEFAULT 0              COMMENT '晚归人数',
    stranger_count      INT           DEFAULT 0              COMMENT '陌生人记录数',
    unknown_count       INT           DEFAULT 0              COMMENT '无法确定人数',
    status              VARCHAR(16)   DEFAULT 'COMPLETED'    COMMENT 'PENDING/COMPLETED/FAILED',
    trigger_type        VARCHAR(8)    DEFAULT 'AUTO'         COMMENT 'AUTO/MANUAL',
    created_at          DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',

    UNIQUE KEY uk_date_building (report_date, building),
    INDEX idx_report_date (report_date),
    INDEX idx_building (building)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='每晚查宿统计表';
```

### 3.5 查宿明细表 `dorm_nightly_detail`

```sql
CREATE TABLE dorm_nightly_detail (
    id                BIGINT        AUTO_INCREMENT PRIMARY KEY,
    report_id         BIGINT        NOT NULL                COMMENT '关联report表ID',
    student_id        VARCHAR(32)   NOT NULL                COMMENT '学号',
    student_name      VARCHAR(64)   NOT NULL                COMMENT '姓名',
    building          VARCHAR(8)    NOT NULL                COMMENT '楼栋',
    room              VARCHAR(16)                           COMMENT '房间号',
    class_name        VARCHAR(64)                           COMMENT '班级',
    status            VARCHAR(16)   NOT NULL                COMMENT 'present/absent/late_return/unknown',
    entry_time        DATETIME                              COMMENT '当日最早进入时间',
    exit_time         DATETIME                              COMMENT '当日最晚离开时间',
    is_late_return    BOOLEAN       DEFAULT FALSE           COMMENT '是否晚归',
    created_at        DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_report_id (report_id),
    INDEX idx_student_id (student_id),
    INDEX idx_status (status),
    INDEX idx_building_room (building, room),
    CONSTRAINT fk_report FOREIGN KEY (report_id)
        REFERENCES dorm_nightly_report(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='查宿明细表';
```

### 3.6 陌生人记录表 `dorm_stranger_record`

```sql
CREATE TABLE dorm_stranger_record (
    id                  BIGINT        AUTO_INCREMENT PRIMARY KEY,
    building            VARCHAR(8)    NOT NULL               COMMENT '楼栋',
    face_snapshot_url   VARCHAR(512)                        COMMENT '抓拍快照URL',
    confidence          DECIMAL(5,4)                        COMMENT '最高置信度',
    event_type          VARCHAR(8)    NOT NULL               COMMENT 'entry/exit',
    detected_time       DATETIME      NOT NULL               COMMENT '发现时间',
    status              VARCHAR(16)   DEFAULT 'UNCONFIRMED'  COMMENT 'UNCONFIRMED/CONFIRMED/DISMISSED',
    remark              VARCHAR(256)                        COMMENT '备注',
    created_at          DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_building (building),
    INDEX idx_status (status),
    INDEX idx_detected_time (detected_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='陌生人记录表';
```

### 3.7 告警记录表 `dorm_alert_record`

```sql
CREATE TABLE dorm_alert_record (
    id                  BIGINT        AUTO_INCREMENT PRIMARY KEY,
    alert_id            VARCHAR(64)   NOT NULL UNIQUE        COMMENT '告警唯一ID',
    alert_type          VARCHAR(32)   NOT NULL               COMMENT 'STRANGER_ENTRY/LONG_ABSENT/CROSS_BUILDING/LATE_RETURN/SYSTEM',
    building            VARCHAR(8)                           COMMENT '相关楼栋',
    student_id          VARCHAR(32)                          COMMENT '相关学生(可为空)',
    severity            VARCHAR(8)    NOT NULL               COMMENT 'low/medium/high/critical',
    description         VARCHAR(512)                        COMMENT '告警描述',
    face_snapshot_url   VARCHAR(512)                        COMMENT '快照URL',
    is_read             BOOLEAN       DEFAULT FALSE          COMMENT '是否已读',
    is_resolved         BOOLEAN       DEFAULT FALSE          COMMENT '是否已处理',
    occurred_at         DATETIME      NOT NULL               COMMENT '发生时间',
    created_at          DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_alert_type (alert_type),
    INDEX idx_severity (severity),
    INDEX idx_occurred_at (occurred_at),
    INDEX idx_building (building),
    INDEX idx_is_resolved (is_resolved)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='告警记录表';
```

### 3.8 配置表 `dorm_config`

```sql
CREATE TABLE dorm_config (
    id              BIGINT        AUTO_INCREMENT PRIMARY KEY,
    config_key      VARCHAR(128)  NOT NULL UNIQUE            COMMENT '配置键',
    config_value    TEXT          NOT NULL                   COMMENT '配置值',
    config_type     VARCHAR(32)   DEFAULT 'string'           COMMENT 'string/int/bool/float',
    description     VARCHAR(256)                            COMMENT '配置说明',
    default_value   TEXT                                     COMMENT '默认值',
    group_name      VARCHAR(32)                              COMMENT '配置分组: nightly/alert/sync/kafka/cache/stranger/system',
    created_at      DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_config_key (config_key),
    INDEX idx_group (group_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统配置表';
```

### 3.9 同步日志表 `dorm_sync_log`

```sql
CREATE TABLE dorm_sync_log (
    id              BIGINT        AUTO_INCREMENT PRIMARY KEY,
    sync_type       VARCHAR(32)   NOT NULL               COMMENT 'STUDENT',
    sync_status     VARCHAR(16)   NOT NULL               COMMENT 'SUCCESS/FAILED/IN_PROGRESS',
    total_count     INT                                  COMMENT '同步总数',
    success_count   INT                                  COMMENT '成功数',
    fail_count      INT                                  COMMENT '失败数',
    error_message   TEXT                                 COMMENT '错误信息',
    duration_ms     BIGINT                               COMMENT '耗时(毫秒)',
    started_at      DATETIME      NOT NULL               COMMENT '开始时间',
    finished_at     DATETIME                             COMMENT '结束时间',

    INDEX idx_sync_type (sync_type),
    INDEX idx_started_at (started_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='同步日志表';
```

### 3.10 摄像头信息表 `dorm_camera`

```sql
CREATE TABLE dorm_camera (
    id                BIGINT        AUTO_INCREMENT PRIMARY KEY,
    camera_id         VARCHAR(32)   NOT NULL UNIQUE        COMMENT '摄像头唯一ID',
    name              VARCHAR(64)   NOT NULL               COMMENT '显示名称',
    building          VARCHAR(8)    NOT NULL               COMMENT '所在楼栋 A/B/C/D',
    rtsp_url          VARCHAR(512)  NOT NULL               COMMENT 'RTSP拉流地址',
    direction         VARCHAR(16)   DEFAULT 'entry'        COMMENT '监控方向',
    resolution        VARCHAR(16)   DEFAULT '1280x720'     COMMENT '分辨率',
    status            VARCHAR(16)   DEFAULT 'unknown'      COMMENT 'online/offline/idle/unknown',
    fps_current       DECIMAL(5,2)  DEFAULT 0              COMMENT '当前帧率',
    total_frames      BIGINT        DEFAULT 0              COMMENT '累计帧数',
    last_heartbeat    DATETIME                             COMMENT '最近心跳时间',
    last_event_time   DATETIME                             COMMENT '最近事件时间',
    enabled           BOOLEAN       DEFAULT TRUE           COMMENT '是否启用',
    config_json       TEXT                                 COMMENT '摄像头级配置(JSON)',
    remark            VARCHAR(256)                         COMMENT '备注',
    created_at        DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_building (building),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='摄像头信息表';
```

### 3.11 摄像头日志表 `dorm_camera_log`

```sql
CREATE TABLE dorm_camera_log (
    id              BIGINT        AUTO_INCREMENT PRIMARY KEY,
    camera_id       VARCHAR(32)   NOT NULL               COMMENT '摄像头ID',
    building        VARCHAR(8)    NOT NULL               COMMENT '楼栋',
    status_from     VARCHAR(16)                          COMMENT '变更前状态',
    status_to       VARCHAR(16)   NOT NULL               COMMENT '变更后状态',
    reason          VARCHAR(128)                         COMMENT '变更原因',
    fps_at_time     DECIMAL(5,2)                        COMMENT '变更时帧率',
    created_at      DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_camera_ts (camera_id, created_at),
    INDEX idx_building_ts (building, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='摄像头状态变更日志表';
```

---

## 4. 索引设计

### 4.1 索引矩阵

| 表 | 索引 | 列 | 理由 |
|----|------|----|------|
| `dorm_entry_exit_event` | `idx_building_ts` | (building, timestamp) | 按楼栋查时间段事件 |
| `dorm_entry_exit_event` | `idx_student_id` | (student_id) | 查单个学生进出记录 |
| `dorm_entry_exit_event` | `idx_camera_ts` | (camera_id, timestamp) | 按摄像头查抓拍 |
| `dorm_entry_exit_event` | `idx_stranger` | (is_stranger) | 陌生人筛选 |
| `dorm_nightly_report` | `uk_date_building` | (report_date, building) | 唯一约束+快速查询 |
| `dorm_student_assignment` | `idx_building_room` | (building, room) | 按楼栋/房间查住宿分配 |
| `dorm_student_assignment` | `idx_active` | (active) | 只在住宿的学生 |
| `dorm_camera` | `idx_building` | (building) | 按楼栋查摄像头 |
| `dorm_camera_log` | `idx_camera_ts` | (camera_id, created_at) | 摄像头状态变化历史 |
| `dorm_config` | `idx_group` | (group_name) | 按分组查询配置 |

### 4.2 索引定义汇总

```sql
-- 事件流水表（核心查询）
ALTER TABLE dorm_entry_exit_event ADD INDEX idx_building_ts (building, timestamp);
ALTER TABLE dorm_entry_exit_event ADD INDEX idx_student_id (student_id);
ALTER TABLE dorm_entry_exit_event ADD INDEX idx_event_type (event_type);
ALTER TABLE dorm_entry_exit_event ADD INDEX idx_timestamp (timestamp);
ALTER TABLE dorm_entry_exit_event ADD INDEX idx_stranger (is_stranger);
ALTER TABLE dorm_entry_exit_event ADD INDEX idx_camera_ts (camera_id, timestamp);

-- 查宿统计
ALTER TABLE dorm_nightly_detail ADD INDEX idx_report_id (report_id);
ALTER TABLE dorm_nightly_detail ADD INDEX idx_status (status);
ALTER TABLE dorm_nightly_detail ADD INDEX idx_building_room (building, room);

-- 告警
ALTER TABLE dorm_alert_record ADD INDEX idx_alert_type (alert_type);
ALTER TABLE dorm_alert_record ADD INDEX idx_severity (severity);
ALTER TABLE dorm_alert_record ADD INDEX idx_occurred_at (occurred_at);
ALTER TABLE dorm_alert_record ADD INDEX idx_is_resolved (is_resolved);

-- 摄像头
ALTER TABLE dorm_camera ADD INDEX idx_status (status);
ALTER TABLE dorm_camera_log ADD INDEX idx_camera_ts (camera_id, created_at);
```

---

## 5. Flyway 迁移

### 5.1 迁移脚本命名规范

```
src/main/resources/db/migration/
├── V1__init_schema.sql           # 初始建表
├── V2__seed_config.sql           # 插入默认配置
├── V3__add_camera_tables.sql     # 摄像头相关表
└── V4__add_indexes.sql           # 性能优化索引
```

### 5.2 V1__init_schema.sql 概要

```sql
-- V1: 初始化核心业务表
-- 按顺序执行: assignment → status → event → report → detail → stranger → alert → config → sync_log

-- 创建数据库
-- CREATE DATABASE IF NOT EXISTS dormitory DEFAULT CHARSET utf8mb4;
-- USE dormitory;

-- 1. 学生宿舍分配表
CREATE TABLE dorm_student_assignment ( ... );
-- 2. 人员在校状态表
CREATE TABLE dorm_student_status ( ... );
-- 3. 进出事件表
CREATE TABLE dorm_entry_exit_event ( ... );
-- 4. 查宿统计表
CREATE TABLE dorm_nightly_report ( ... );
-- 5. 查宿明细表
CREATE TABLE dorm_nightly_detail ( ... );
-- 6. 陌生人记录表
CREATE TABLE dorm_stranger_record ( ... );
-- 7. 告警记录表
CREATE TABLE dorm_alert_record ( ... );
-- 8. 配置表
CREATE TABLE dorm_config ( ... );
-- 9. 同步日志表
CREATE TABLE dorm_sync_log ( ... );
```

### 5.3 V2__seed_config.sql

```sql
-- V2: 插入默认配置
INSERT INTO dorm_config (config_key, config_value, config_type, description, default_value, group_name) VALUES

-- 查宿规则
('nightly_report.trigger_time', '23:00', 'string', '自动查宿每日触发时间', '23:00', 'nightly'),
('nightly_report.timezone', 'Asia/Shanghai', 'string', '查宿统计使用的时区', 'Asia/Shanghai', 'nightly'),
('late_return.threshold', '22:00', 'string', '晚归判定时间阈值', '22:00', 'nightly'),

-- 告警阈值
('absent.alert_hours', '24', 'int', '未归告警阈值(小时)', '24', 'alert'),
('alert.stranger.enabled', 'true', 'bool', '陌生人告警开关', 'true', 'alert'),
('alert.cross_building.enabled', 'true', 'bool', '跨楼栋告警开关', 'true', 'alert'),
('alert.late_return.enabled', 'true', 'bool', '晚归记录开关', 'true', 'alert'),
('alert.cooldown_seconds', '300', 'int', '同类型告警最小间隔', '300', 'alert'),
('alert.max_per_minute', '100', 'int', '全局告警频率上限', '100', 'alert'),

-- 学管同步
('sync.student.enabled', 'true', 'bool', '自动同步开关', 'true', 'sync'),
('sync.student.interval_min', '60', 'int', '同步间隔(分钟)', '60', 'sync'),
('sync.student.api_url', '', 'string', '学管宿舍数据API地址(必填)', '', 'sync'),
('sync.student.timeout_sec', '30', 'int', '同步请求超时(秒)', '30', 'sync'),
('sync.student.retry_max', '3', 'int', '同步失败最大重试次数', '3', 'sync'),

-- Kafka
('kafka.consumer.topic', 't_dorm_event', 'string', '进出事件Topic', 't_dorm_event', 'kafka'),
('kafka.consumer.group', 'dormitory-service', 'string', '消费者组ID', 'dormitory-service', 'kafka'),
('kafka.bootstrap.servers', 'localhost:9092', 'string', 'Kafka集群地址', 'localhost:9092', 'kafka'),

-- 缓存
('cache.status.ttl_hours', '6', 'int', '状态缓存TTL(小时)', '6', 'cache'),
('cache.config.ttl_minutes', '5', 'int', '配置缓存TTL(分钟)', '5', 'cache'),

-- 陌生人
('stranger.confidence_threshold', '0.6', 'float', '陌生人置信度阈值', '0.6', 'stranger'),

-- 摄像头
('camera.health_check.interval_sec', '30', 'int', '摄像头健康检查间隔(秒)', '30', 'camera'),
('camera.offline.alert_threshold', '3', 'int', '连续失败N次触发离线告警', '3', 'camera'),
('camera.idle.threshold_min', '5', 'int', 'N分钟无事件标记idle', '5', 'camera');
```

---

## 6. Redis Key 设计

### 6.1 Key 完整列表

| Key 模式 | 类型 | 用途 | TTL |
|----------|------|------|-----|
| `dorm:student:{studentId}:status` | Hash | 实时在校状态 | 次日06:00 |
| `dorm:building:{building}:students` | Set | 楼栋内所有学生ID | 永久（随同步更新） |
| `dorm:building:{building}:status` | Hash | 楼栋聚合状态（总人数/在/离） | 5min |
| `dorm:event:processed:{eventId}` | String | 事件幂等去重 | 1h |
| `dorm:config` | Hash | 配置缓存 | 配置变更时刷新 |
| `dorm:report:today:{building}` | String | 今日查宿缓存 | 次日06:00 |
| `dorm:alert:cooldown:{type}` | String | 告警冷却 | 300s |

### 6.2 Redis 操作示例

```java
// 查询学生状态
String key = RedisKeys.studentStatus(studentId);
Map<Object, Object> status = redisTemplate.opsForHash().entries(key);

// 批量查询楼栋学生
String setKey = RedisKeys.buildingStudents(building);
Set<String> studentIds = redisTemplate.opsForSet().members(setKey);

// 幂等检查
String dedupKey = RedisKeys.eventProcessed(eventId);
if (Boolean.TRUE.equals(redisTemplate.hasKey(dedupKey))) {
    log.debug("Deduplicated event: {}", eventId);
    return;
}
```

---

## 7. 数据量估算

### 7.1 业务数据量

| 表 | 每行大小 | 日增量 | 月增量 | 年增量 |
|----|---------|--------|--------|--------|
| `dorm_entry_exit_event` | ~300 bytes | ~3 MB | ~90 MB | ~1.1 GB |
| `dorm_nightly_report` | ~200 bytes | ~800 B | ~24 KB | ~288 KB |
| `dorm_nightly_detail` | ~250 bytes | ~125 KB | ~3.75 MB | ~45 MB |
| `dorm_stranger_record` | ~300 bytes | ~15 KB | ~450 KB | ~5.4 MB |
| `dorm_alert_record` | ~400 bytes | ~40 KB | ~1.2 MB | ~14.4 MB |
| `dorm_camera_log` | ~200 bytes | ~10 KB | ~300 KB | ~3.6 MB |
| **合计** | | **~3.2 MB/日** | **~96 MB/月** | **~1.2 GB/年** |

### 7.2 Redis 内存估算

| Key 类型 | 数量 | 单条大小 | 总内存 |
|----------|------|---------|--------|
| 学生状态 | 500 | ~200 bytes | ~100 KB |
| 楼栋状态 | 4 | ~100 bytes | ~400 B |
| 事件去重 | 主动过期 | ~50 bytes | 可控 |
| 配置缓存 | 30 | ~200 bytes | ~6 KB |
| **合计** | | | **~200 KB 常驻** |

---

## 8. 存储与清理策略

### 8.1 数据生命周期

```
事件数据:
  产生 → 实时查询(Redis) → 最近7天(热数据) → 最近30天(温数据) → >30天(归档/清理)

查宿报表:
  产生 → 实时查询 → 长期保存（业务需求，不可删除）
```

### 8.2 清理策略

| 数据 | 保留策略 | 清理方式 |
|------|---------|---------|
| 进出事件 (entry_exit) | 保留 90 天 | 定时任务 DELETE > 90d 或 DROP 历史分区 |
| 陌生人记录 | 保留 30 天 | 定时任务 DELETE > 30d |
| 告警记录 | 保留 90 天 | 定时任务 DELETE > 90d |
| 摄像头日志 | 保留 30 天 | 定时任务 DELETE > 30d |
| 查宿报表 | **永久保留** | 不做清理 |
| 同步日志 | 保留 30 天 | 定时任务 DELETE > 30d |
| Redis 状态缓存 | TTL 次日 06:00 | Redis 自动过期 |
| Redis 去重缓存 | TTL 1 小时 | Redis 自动过期 |

### 8.3 清理定时任务示例

```java
@Component
@Slf4j
public class DataCleanupTask {

    @Autowired
    private JdbcTemplate jdbcTemplate;

    /**
     * 每天凌晨 3:00 执行数据清理
     */
    @Scheduled(cron = "0 0 3 * * ?")
    public void cleanupOldData() {
        log.info("开始清理过期数据...");

        // 删除 > 90 天的进出事件
        int deletedEvents = jdbcTemplate.update(
            "DELETE FROM dorm_entry_exit_event WHERE timestamp < DATE_SUB(NOW(), INTERVAL 90 DAY)");
        log.info("清理事件: {} 条", deletedEvents);

        // 删除 > 30 天的陌生人记录
        int deletedStrangers = jdbcTemplate.update(
            "DELETE FROM dorm_stranger_record WHERE detected_time < DATE_SUB(NOW(), INTERVAL 30 DAY)");
        log.info("清理陌生人记录: {} 条", deletedStrangers);

        // 删除 > 90 天的告警记录
        int deletedAlerts = jdbcTemplate.update(
            "DELETE FROM dorm_alert_record WHERE occurred_at < DATE_SUB(NOW(), INTERVAL 90 DAY)");
        log.info("清理告警记录: {} 条", deletedAlerts);
    }
}
```

---

> **本文件属于**: `doc/design/backend/02-database.md`  
> **面向读者**: Java 后端开发（搭档）  
> **迁移工具**: Flyway  
> **数据库**: MariaDB 10.11+
