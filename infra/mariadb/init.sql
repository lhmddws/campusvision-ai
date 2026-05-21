-- CampusVision AI — MariaDB 初始化脚本
-- 由 docker-compose 自动执行 (maria-init)
-- 数据库 dormitory 由 docker-compose 的 MARIADB_DATABASE 变量自动创建

-- ==================== 核心业务表 ====================

-- 1. 学生宿舍分配表（从学管系统同步）
CREATE TABLE IF NOT EXISTS dorm_student_assignment (
    id              BIGINT          AUTO_INCREMENT PRIMARY KEY,
    student_id      VARCHAR(32)     NOT NULL UNIQUE                 COMMENT '学号',
    student_name    VARCHAR(64)     NOT NULL                        COMMENT '姓名',
    building        VARCHAR(8)      NOT NULL                        COMMENT '宿舍楼栋 A/B/C/D',
    room            VARCHAR(16)     NOT NULL                        COMMENT '房间号',
    class_name      VARCHAR(64)                                     COMMENT '班级',
    grade           VARCHAR(32)                                     COMMENT '年级',
    gender          VARCHAR(8)                                      COMMENT '性别',
    phone           VARCHAR(20)                                     COMMENT '联系电话',
    active          TINYINT(1)      DEFAULT 1                       COMMENT '是否在校住宿',
    sync_version    BIGINT          DEFAULT 0                       COMMENT '同步版本号(乐观锁)',
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

    INDEX idx_asa_building_room (building, room),
    INDEX idx_asa_building (building),
    INDEX idx_asa_active (active)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='学生宿舍分配表';

-- 2. 人员在校状态表
CREATE TABLE IF NOT EXISTS dorm_student_status (
    id                BIGINT        AUTO_INCREMENT PRIMARY KEY,
    student_id        VARCHAR(32)   NOT NULL UNIQUE                 COMMENT '学号',
    student_name      VARCHAR(64)   NOT NULL                        COMMENT '姓名',
    building          VARCHAR(8)    NOT NULL                        COMMENT '所属楼栋',
    room              VARCHAR(16)                                    COMMENT '房间号',
    is_in_dorm        TINYINT(1)    DEFAULT 0                       COMMENT '是否在宿舍',
    last_entry_time   DATETIME                                      COMMENT '最近进入时间',
    last_exit_time    DATETIME                                      COMMENT '最近离开时间',
    today_status      VARCHAR(16)   DEFAULT 'unknown'               COMMENT '今日状态: in/out/unknown',
    today_entry_count INT           DEFAULT 0                       COMMENT '今日进入次数',
    today_exit_count  INT           DEFAULT 0                       COMMENT '今日离开次数',
    last_update       DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '最后更新时间',

    INDEX idx_ss_building (building),
    INDEX idx_ss_today_status (today_status),
    INDEX idx_ss_is_in_dorm (is_in_dorm)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='人员在校状态表';

-- 3. 进出事件表（核心流水）
CREATE TABLE IF NOT EXISTS dorm_entry_exit_event (
    id                  BIGINT        AUTO_INCREMENT PRIMARY KEY,
    event_id            VARCHAR(64)   NOT NULL UNIQUE               COMMENT '事件唯一ID(幂等)',
    camera_id           VARCHAR(32)                                 COMMENT '摄像头ID',
    building            VARCHAR(8)    NOT NULL                       COMMENT '楼栋',
    student_id          VARCHAR(32)                                 COMMENT '学生学号(可为空=陌生人)',
    student_name        VARCHAR(64)                                 COMMENT '学生姓名',
    event_type          VARCHAR(8)    NOT NULL                       COMMENT 'entry/exit',
    confidence          DECIMAL(5,4)                                COMMENT '人脸识别置信度',
    face_snapshot_url   VARCHAR(512)                                COMMENT '抓拍快照URL',
    is_stranger         TINYINT(1)    DEFAULT 0                      COMMENT '是否陌生人',
    is_processed        TINYINT(1)    DEFAULT 1                      COMMENT '是否已被消费处理',
    timestamp           DATETIME      NOT NULL                       COMMENT '事件时间',
    created_at          DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '记录创建时间',

    INDEX idx_eee_building_ts (building, timestamp),
    INDEX idx_eee_student_id (student_id),
    INDEX idx_eee_event_type (event_type),
    INDEX idx_eee_timestamp (timestamp),
    INDEX idx_eee_stranger (is_stranger),
    INDEX idx_eee_camera_ts (camera_id, timestamp)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='进出事件表';

-- 4. 查宿统计汇总表
CREATE TABLE IF NOT EXISTS dorm_nightly_report (
    id                  BIGINT        AUTO_INCREMENT PRIMARY KEY,
    report_date         DATE          NOT NULL                       COMMENT '统计日期',
    building            VARCHAR(8)    NOT NULL                       COMMENT '楼栋',
    total_count         INT           NOT NULL                       COMMENT '应归人数',
    present_count       INT           NOT NULL                       COMMENT '已归人数',
    absent_count        INT           NOT NULL                       COMMENT '未归人数',
    late_return_count   INT           DEFAULT 0                      COMMENT '晚归人数',
    stranger_count      INT           DEFAULT 0                      COMMENT '陌生人记录数',
    unknown_count       INT           DEFAULT 0                      COMMENT '无法确定人数',
    status              VARCHAR(16)   DEFAULT 'COMPLETED'            COMMENT 'PENDING/COMPLETED/FAILED',
    trigger_type        VARCHAR(8)    DEFAULT 'AUTO'                 COMMENT 'AUTO/MANUAL',
    created_at          DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',

    UNIQUE KEY uk_nr_date_building (report_date, building),
    INDEX idx_nr_report_date (report_date),
    INDEX idx_nr_building (building)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='每晚查宿统计表';

-- 5. 查宿明细表
CREATE TABLE IF NOT EXISTS dorm_nightly_detail (
    id                BIGINT        AUTO_INCREMENT PRIMARY KEY,
    report_id         BIGINT        NOT NULL                        COMMENT '关联report表ID',
    student_id        VARCHAR(32)   NOT NULL                        COMMENT '学号',
    student_name      VARCHAR(64)   NOT NULL                        COMMENT '姓名',
    building          VARCHAR(8)    NOT NULL                        COMMENT '楼栋',
    room              VARCHAR(16)                                   COMMENT '房间号',
    class_name        VARCHAR(64)                                   COMMENT '班级',
    status            VARCHAR(16)   NOT NULL                        COMMENT 'present/absent/late_return/unknown',
    entry_time        DATETIME                                      COMMENT '当日最早进入时间',
    exit_time         DATETIME                                      COMMENT '当日最晚离开时间',
    is_late_return    TINYINT(1)    DEFAULT 0                       COMMENT '是否晚归',
    created_at        DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_nd_report_id (report_id),
    INDEX idx_nd_student_id (student_id),
    INDEX idx_nd_status (status),
    INDEX idx_nd_building_room (building, room),
    CONSTRAINT fk_nd_report FOREIGN KEY (report_id)
        REFERENCES dorm_nightly_report(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='查宿明细表';

-- 6. 陌生人记录表
CREATE TABLE IF NOT EXISTS dorm_stranger_record (
    id                  BIGINT        AUTO_INCREMENT PRIMARY KEY,
    building            VARCHAR(8)    NOT NULL                      COMMENT '楼栋',
    face_snapshot_url   VARCHAR(512)                                COMMENT '抓拍快照URL',
    confidence          DECIMAL(5,4)                                COMMENT '最高置信度',
    event_type          VARCHAR(8)    NOT NULL                      COMMENT 'entry/exit',
    detected_time       DATETIME      NOT NULL                      COMMENT '发现时间',
    status              VARCHAR(16)   DEFAULT 'UNCONFIRMED'         COMMENT 'UNCONFIRMED/CONFIRMED/DISMISSED',
    remark              VARCHAR(256)                                COMMENT '备注',
    created_at          DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_sr_building (building),
    INDEX idx_sr_status (status),
    INDEX idx_sr_detected_time (detected_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='陌生人记录表';

-- 7. 告警记录表
CREATE TABLE IF NOT EXISTS dorm_alert_record (
    id                  BIGINT        AUTO_INCREMENT PRIMARY KEY,
    alert_id            VARCHAR(64)   NOT NULL UNIQUE               COMMENT '告警唯一ID',
    alert_type          VARCHAR(32)   NOT NULL                      COMMENT 'STRANGER_ENTRY/LONG_ABSENT/CROSS_BUILDING/LATE_RETURN/SYSTEM',
    building            VARCHAR(8)                                   COMMENT '相关楼栋',
    student_id          VARCHAR(32)                                 COMMENT '相关学生(可为空)',
    severity            VARCHAR(8)    NOT NULL                      COMMENT 'low/medium/high/critical',
    description         VARCHAR(512)                                COMMENT '告警描述',
    face_snapshot_url   VARCHAR(512)                                COMMENT '快照URL',
    is_read             TINYINT(1)    DEFAULT 0                     COMMENT '是否已读',
    is_resolved         TINYINT(1)    DEFAULT 0                     COMMENT '是否已处理',
    occurred_at         DATETIME      NOT NULL                      COMMENT '发生时间',
    created_at          DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_ar_alert_type (alert_type),
    INDEX idx_ar_severity (severity),
    INDEX idx_ar_occurred_at (occurred_at),
    INDEX idx_ar_building (building),
    INDEX idx_ar_is_resolved (is_resolved)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='告警记录表';

-- 8. 配置表
CREATE TABLE IF NOT EXISTS dorm_config (
    id              BIGINT        AUTO_INCREMENT PRIMARY KEY,
    config_key      VARCHAR(128)  NOT NULL UNIQUE                   COMMENT '配置键',
    config_value    TEXT          NOT NULL                          COMMENT '配置值',
    config_type     VARCHAR(32)   DEFAULT 'string'                  COMMENT 'string/int/bool/float',
    description     VARCHAR(256)                                    COMMENT '配置说明',
    default_value   TEXT                                            COMMENT '默认值',
    group_name      VARCHAR(32)                                     COMMENT '配置分组: nightly/alert/sync/kafka/cache/stranger/system',
    created_at      DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_cfg_key (config_key),
    INDEX idx_cfg_group (group_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='系统配置表';

-- 9. 同步日志表
CREATE TABLE IF NOT EXISTS dorm_sync_log (
    id              BIGINT        AUTO_INCREMENT PRIMARY KEY,
    sync_type       VARCHAR(32)   NOT NULL                          COMMENT 'STUDENT',
    sync_status     VARCHAR(16)   NOT NULL                          COMMENT 'SUCCESS/FAILED/IN_PROGRESS',
    total_count     INT                                             COMMENT '同步总数',
    success_count   INT                                             COMMENT '成功数',
    fail_count      INT                                             COMMENT '失败数',
    error_message   TEXT                                            COMMENT '错误信息',
    duration_ms     BIGINT                                          COMMENT '耗时(毫秒)',
    started_at      DATETIME      NOT NULL                          COMMENT '开始时间',
    finished_at     DATETIME                                        COMMENT '结束时间',

    INDEX idx_sl_sync_type (sync_type),
    INDEX idx_sl_started_at (started_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='同步日志表';

-- 10. 摄像头信息表
CREATE TABLE IF NOT EXISTS dorm_camera (
    id                BIGINT        AUTO_INCREMENT PRIMARY KEY,
    camera_id         VARCHAR(32)   NOT NULL UNIQUE                 COMMENT '摄像头唯一ID',
    name              VARCHAR(64)   NOT NULL                        COMMENT '显示名称',
    building          VARCHAR(8)    NOT NULL                        COMMENT '所在楼栋 A/B/C/D',
    rtsp_url          VARCHAR(512)  NOT NULL                        COMMENT 'RTSP拉流地址',
    direction         VARCHAR(16)   DEFAULT 'entry'                 COMMENT '监控方向',
    resolution        VARCHAR(16)   DEFAULT '1280x720'              COMMENT '分辨率',
    status            VARCHAR(16)   DEFAULT 'unknown'               COMMENT 'online/offline/idle/unknown',
    fps_current       DECIMAL(5,2)  DEFAULT 0                       COMMENT '当前帧率',
    total_frames      BIGINT        DEFAULT 0                       COMMENT '累计帧数',
    last_heartbeat    DATETIME                                      COMMENT '最近心跳时间',
    last_health_check DATETIME DEFAULT NULL                          COMMENT '最近健康检查时间',
    last_event_time   DATETIME                                      COMMENT '最近事件时间',
    enabled           TINYINT(1)    DEFAULT 1                       COMMENT '是否启用',
    config_json       TEXT                                          COMMENT '摄像头级配置(JSON)',
    remark            VARCHAR(256)                                  COMMENT '备注',
    created_at        DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_cam_building (building),
    INDEX idx_cam_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='摄像头信息表';

-- 11. 摄像头日志表
CREATE TABLE IF NOT EXISTS dorm_camera_log (
    id              BIGINT        AUTO_INCREMENT PRIMARY KEY,
    camera_id       VARCHAR(32)   NOT NULL                          COMMENT '摄像头ID',
    building        VARCHAR(8)    NOT NULL                          COMMENT '楼栋',
    status_from     VARCHAR(16)                                     COMMENT '变更前状态',
    status_to       VARCHAR(16)   NOT NULL                          COMMENT '变更后状态',
    reason          VARCHAR(128)                                    COMMENT '变更原因',
    fps_at_time     DECIMAL(5,2)                                    COMMENT '变更时帧率',
    created_at      DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_cl_camera_ts (camera_id, created_at),
    INDEX idx_cl_building_ts (building, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='摄像头状态变更日志表';

-- 12. 人脸特征向量表（识别流水线）
CREATE TABLE IF NOT EXISTS face_embedding (
    id          BIGINT AUTO_INCREMENT PRIMARY KEY,
    name        VARCHAR(100) NOT NULL COMMENT '姓名',
    student_id  VARCHAR(50) NOT NULL UNIQUE COMMENT '学号',
    embedding   BLOB COMMENT '512维浮点向量 (2048 bytes)',
    image_path  VARCHAR(500) COMMENT '人脸图片路径',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

    INDEX idx_fe_student_id (student_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='人脸特征向量表';

-- ==================== 默认配置 ====================
INSERT IGNORE INTO dorm_config (config_key, config_value, config_type, description, group_name) VALUES
    ('nightly_report.trigger_time', '23:00', 'string', '自动查宿每日触发时间', 'nightly'),
    ('nightly_report.timezone', 'Asia/Shanghai', 'string', '查宿统计使用的时区', 'nightly'),
    ('late_return.threshold', '22:00', 'string', '晚归判定时间阈值', 'nightly'),
    ('absent.alert_hours', '24', 'int', '未归告警阈值(小时)', 'alert'),
    ('alert.stranger.enabled', 'true', 'bool', '陌生人告警开关', 'alert'),
    ('alert.cooldown_seconds', '300', 'int', '同类型告警最小间隔', 'alert'),
    ('alert.max_per_minute', '100', 'int', '全局告警频率上限', 'alert'),
    ('sync.student.enabled', 'true', 'bool', '自动同步开关', 'sync'),
    ('sync.student.interval_min', '60', 'int', '同步间隔(分钟)', 'sync'),
    ('sync.student.api_url', '', 'string', '学管宿舍数据API地址', 'sync'),
    ('sync.student.timeout_sec', '30', 'int', '同步请求超时(秒)', 'sync'),
    ('kafka.consumer.topic', 't_dorm_event', 'string', '进出事件Topic', 'kafka'),
    ('kafka.bootstrap.servers', 'kafka:9092', 'string', 'Kafka集群地址', 'kafka'),
    ('cache.status.ttl_hours', '6', 'int', '状态缓存TTL(小时)', 'cache'),
    ('stranger.confidence_threshold', '0.6', 'float', '陌生人置信度阈值', 'stranger'),
    ('camera.health_check.interval_sec', '30', 'int', '摄像头健康检查间隔', 'camera'),
    ('camera.offline.alert_threshold', '3', 'int', '连续失败N次触发离线告警', 'camera');

-- ==================== 示例摄像头 ====================
-- 按需通过 REST API 注册，此处仅作为 schema 参考
-- INSERT IGNORE INTO dorm_camera (camera_id, name, building, rtsp_url, direction, resolution, enabled) VALUES
--     ('cam-a', 'A栋入口', 'A', 'rtsp://admin:password@192.168.1.101:554/stream1', 'entry', '1280x720', 1),
--     ('cam-b', 'B栋入口', 'B', 'rtsp://admin:password@192.168.1.102:554/stream1', 'entry', '1280x720', 1),
--     ('cam-c', 'C栋入口', 'C', 'rtsp://admin:password@192.168.1.103:554/stream1', 'entry', '1280x720', 1),
--     ('cam-d', 'D栋入口', 'D', 'rtsp://admin:password@192.168.1.104:554/stream1', 'entry', '1280x720', 1);
