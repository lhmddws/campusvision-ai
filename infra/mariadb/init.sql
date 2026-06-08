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
    type              VARCHAR(16)   DEFAULT 'RTSP'                  COMMENT '摄像头类型: RTSP/SIMULATED/USB',
    rtsp_url          VARCHAR(512)  NOT NULL                        COMMENT 'RTSP拉流地址',
    protocol          VARCHAR(8)    DEFAULT 'rtsp'                  COMMENT '拉流协议',
    host              VARCHAR(128)                                   COMMENT '摄像头主机',
    port              INT           DEFAULT 554                     COMMENT '端口',
    path              VARCHAR(256)                                   COMMENT 'RTSP路径',
    username          VARCHAR(64)                                    COMMENT '认证用户名',
    password_enc      TEXT                                           COMMENT 'AES-256-GCM加密密码: base64(nonce|ciphertext)',
    nonce             VARCHAR(32)                                    COMMENT '加密随机数(base64)',
    key_id            VARCHAR(16)   DEFAULT 'v1'                    COMMENT '密钥版本',
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

-- 13. 宿舍楼宇表
CREATE TABLE IF NOT EXISTS dorm_building (
    id              BIGINT AUTO_INCREMENT PRIMARY KEY,
    code            VARCHAR(8) NOT NULL UNIQUE COMMENT '楼宇编号 A/B/C/D',
    name            VARCHAR(64) NOT NULL COMMENT '楼宇名称',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='宿舍楼宇';

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

-- Set config_options for select-type configs (must run after INSERT IGNORE)
UPDATE dorm_config
SET config_options = '["Asia/Shanghai","Asia/Hong_Kong","Asia/Tokyo","Asia/Singapore","America/New_York","America/Los_Angeles","Europe/London","Europe/Paris","Europe/Berlin","Australia/Sydney","UTC"]'
WHERE config_key = 'nightly_report.timezone';

UPDATE dorm_config SET config_options = '["15","30","60","120","360","720","1440"]' WHERE config_key = 'sync.student.interval_min';
UPDATE dorm_config SET config_options = '["10","15","30","60","120"]' WHERE config_key = 'sync.student.timeout_sec';
UPDATE dorm_config SET config_options = '["t_dorm_frame","t_dorm_event","t_dorm_alert"]' WHERE config_key = 'kafka.consumer.topic';
UPDATE dorm_config SET config_options = '["1","2","6","12","24","48"]' WHERE config_key = 'cache.status.ttl_hours';
UPDATE dorm_config SET config_options = '["0.3","0.4","0.5","0.6","0.7","0.8","0.9"]' WHERE config_key = 'stranger.confidence_threshold';
UPDATE dorm_config SET config_options = '["10","15","30","60","120","300"]' WHERE config_key = 'camera.health_check.interval_sec';
UPDATE dorm_config SET config_options = '["1","2","3","5","10"]' WHERE config_key = 'camera.offline.alert_threshold';
UPDATE dorm_config SET config_options = '["60","120","300","600","1800"]' WHERE config_key = 'alert.cooldown_seconds';
UPDATE dorm_config SET config_options = '["20","50","100","200","500"]' WHERE config_key = 'alert.max_per_minute';

-- ==================== 测试数据 ====================

-- 13. 宿舍楼宇
INSERT IGNORE INTO dorm_building (code, name) VALUES
    ('A', 'A栋-学思楼'),
    ('B', 'B栋-致远楼'),
    ('C', 'C栋-明德楼'),
    ('D', 'D栋-博雅楼');

-- 14. 学生宿舍分配 (20人, 散布 A/B/C/D)
INSERT IGNORE INTO dorm_student_assignment (student_id, student_name, building, room, class_name, grade, gender, phone) VALUES
    ('2024001', '张三',   'A', 'A-101', '计算机1班', '2024', '男', '13800001001'),
    ('2024002', '李四',   'A', 'A-101', '计算机1班', '2024', '男', '13800001002'),
    ('2024003', '王五',   'A', 'A-101', '计算机1班', '2024', '男', '13800001003'),
    ('2024004', '赵六',   'A', 'A-102', '计算机2班', '2024', '男', '13800001004'),
    ('2024005', '孙七',   'A', 'A-102', '计算机2班', '2024', '男', '13800001005'),
    ('2024006', '周八',   'A', 'A-201', '计算机1班', '2024', '男', '13800001006'),
    ('2024007', '吴九',   'A', 'A-201', '计算机1班', '2024', '男', '13800001007'),
    ('2024008', '郑十',   'A', 'A-201', '计算机2班', '2024', '男', '13800001008'),
    ('2024009', '钱一',   'B', 'B-101', '软件1班',  '2024', '男', '13800001009'),
    ('2024010', '陈二',   'B', 'B-101', '软件1班',  '2024', '男', '13800001010'),
    ('2024011', '朱三',   'B', 'B-101', '软件2班',  '2024', '男', '13800001011'),
    ('2024012', '刘四',   'B', 'B-102', '软件2班',  '2024', '男', '13800001012'),
    ('2024013', '黄五',   'B', 'B-102', '软件1班',  '2024', '男', '13800001013'),
    ('2024014', '林一',   'C', 'C-101', '网络1班',  '2024', '女', '13800001014'),
    ('2024015', '何二',   'C', 'C-101', '网络1班',  '2024', '女', '13800001015'),
    ('2024016', '罗三',   'C', 'C-101', '网络2班',  '2024', '女', '13800001016'),
    ('2024017', '谢四',   'C', 'C-102', '网络2班',  '2024', '女', '13800001017'),
    ('2024018', '唐五',   'C', 'C-102', '网络1班',  '2024', '女', '13800001018'),
    ('2024019', '韩一',   'D', 'D-101', '大数据1班', '2024', '女', '13800001019'),
    ('2024020', '冯二',   'D', 'D-101', '大数据1班', '2024', '女', '13800001020'),
    ('2024021', '董三',   'D', 'D-102', '大数据2班', '2024', '女', '13800001021'),
    ('2024022', '魏四',   'D', 'D-102', '大数据2班', '2024', '女', '13800001022');

-- 15. 人员在校状态 (基于今日事件模拟: 大部分已进入, 少量缺勤/已离开)
INSERT INTO dorm_student_status (student_id, student_name, building, room, is_in_dorm, last_entry_time, last_exit_time, today_status, today_entry_count, today_exit_count)
SELECT s.student_id, s.student_name, s.building, s.room,
       CASE s.student_id
         WHEN '2024009' THEN 0  -- 钱一 全天未归(缺勤)
         WHEN '2024017' THEN 0  -- 谢四 全天未归(缺勤)
         WHEN '2024006' THEN 0  -- 周八 下午离开未归
         WHEN '2024021' THEN 0  -- 董三 下午离开未归
         ELSE 1
       END,
       CASE s.student_id
         WHEN '2024009' THEN NULL
         WHEN '2024017' THEN NULL
         ELSE CONCAT(CURDATE(), ' ', LPAD(FLOOR(7 + RAND() * 2), 2, '0'), ':', LPAD(FLOOR(0 + RAND() * 60), 2, '0'), ':00')
       END,
       CASE s.student_id
         WHEN '2024006' THEN CONCAT(CURDATE(), ' 14:30:00')
         WHEN '2024021' THEN CONCAT(CURDATE(), ' 15:10:00')
         WHEN '2024004' THEN CONCAT(CURDATE(), ' 12:15:00')
         WHEN '2024012' THEN CONCAT(CURDATE(), ' 16:45:00')
         ELSE NULL
       END,
       CASE s.student_id
         WHEN '2024009' THEN 'out'
         WHEN '2024017' THEN 'out'
         ELSE 'in'
       END,
       CASE WHEN s.student_id IN ('2024009','2024017') THEN 0 ELSE 1 END,
       CASE s.student_id
         WHEN '2024006' THEN 1
         WHEN '2024021' THEN 1
         WHEN '2024004' THEN 1
         WHEN '2024012' THEN 1
         ELSE 0
       END
FROM dorm_student_assignment s
ON DUPLICATE KEY UPDATE
    is_in_dorm    = VALUES(is_in_dorm),
    today_status  = VALUES(today_status),
    last_entry_time = VALUES(last_entry_time),
    last_exit_time  = VALUES(last_exit_time),
    today_entry_count = VALUES(today_entry_count),
    today_exit_count  = VALUES(today_exit_count);

-- 16. 摄像头 (每栋楼 1 个入口摄像头)
-- NOTE: 如果 dorm_camera 表已有旧 schema（缺少 type/protocol 列），执行前先 ALTER TABLE 或删表重建
INSERT IGNORE INTO dorm_camera (camera_id, name, building, rtsp_url, direction, status, enabled) VALUES
    ('cam-a-entry', 'A栋入口', 'A', 'rtsp://admin:PLACEHOLDER@192.168.1.101:554/stream1', 'entry', 'online',  1),
    ('cam-b-entry', 'B栋入口', 'B', 'rtsp://admin:PLACEHOLDER@192.168.1.102:554/stream1', 'entry', 'online',  1),
    ('cam-c-entry', 'C栋入口', 'C', 'rtsp://admin:PLACEHOLDER@192.168.1.103:554/stream1', 'entry', 'online',  1),
    ('cam-d-entry', 'D栋入口', 'D', 'rtsp://admin:PLACEHOLDER@192.168.1.104:554/stream1', 'entry', 'idle',    1);

-- 17. 摄像头日志
INSERT IGNORE INTO dorm_camera_log (camera_id, building, status_from, status_to, reason, created_at) VALUES
    ('cam-a-entry', 'A', NULL, 'online',    '系统初始化', NOW() - INTERVAL 2 HOUR),
    ('cam-b-entry', 'B', NULL, 'online',    '系统初始化', NOW() - INTERVAL 2 HOUR),
    ('cam-c-entry', 'C', NULL, 'online',    '系统初始化', NOW() - INTERVAL 2 HOUR),
    ('cam-d-entry', 'D', NULL, 'idle',      '系统初始化(无事件)', NOW() - INTERVAL 2 HOUR);

-- 18. 今日进出事件 (共 26 条：20 条 entry + 4 条 exit + 2 条陌生人 entry)
INSERT IGNORE INTO dorm_entry_exit_event (event_id, camera_id, building, student_id, student_name, event_type, confidence, is_stranger, timestamp) VALUES
    -- A栋 07:30-08:00 早高峰进入
    ('evt-a-001', 'cam-a-entry', 'A', '2024001', '张三', 'entry', 0.9512, 0, CONCAT(CURDATE(), ' 07:35:00')),
    ('evt-a-002', 'cam-a-entry', 'A', '2024002', '李四', 'entry', 0.9334, 0, CONCAT(CURDATE(), ' 07:38:00')),
    ('evt-a-003', 'cam-a-entry', 'A', '2024003', '王五', 'entry', 0.9656, 0, CONCAT(CURDATE(), ' 07:42:00')),
    ('evt-a-004', 'cam-a-entry', 'A', '2024004', '赵六', 'entry', 0.8910, 0, CONCAT(CURDATE(), ' 07:50:00')),
    ('evt-a-005', 'cam-a-entry', 'A', '2024005', '孙七', 'entry', 0.9123, 0, CONCAT(CURDATE(), ' 07:55:00')),
    ('evt-a-006', 'cam-a-entry', 'A', '2024007', '吴九', 'entry', 0.9478, 0, CONCAT(CURDATE(), ' 08:05:00')),
    ('evt-a-007', 'cam-a-entry', 'A', '2024008', '郑十', 'entry', 0.9234, 0, CONCAT(CURDATE(), ' 08:10:00')),
    -- B栋 07:45-08:15
    ('evt-b-001', 'cam-b-entry', 'B', '2024010', '陈二', 'entry', 0.9432, 0, CONCAT(CURDATE(), ' 07:45:00')),
    ('evt-b-002', 'cam-b-entry', 'B', '2024011', '朱三', 'entry', 0.9687, 0, CONCAT(CURDATE(), ' 07:48:00')),
    ('evt-b-003', 'cam-b-entry', 'B', '2024012', '刘四', 'entry', 0.9056, 0, CONCAT(CURDATE(), ' 07:52:00')),
    ('evt-b-004', 'cam-b-entry', 'B', '2024013', '黄五', 'entry', 0.9234, 0, CONCAT(CURDATE(), ' 08:12:00')),
    -- C栋 07:30-07:55
    ('evt-c-001', 'cam-c-entry', 'C', '2024014', '林一', 'entry', 0.9765, 0, CONCAT(CURDATE(), ' 07:30:00')),
    ('evt-c-002', 'cam-c-entry', 'C', '2024015', '何二', 'entry', 0.9543, 0, CONCAT(CURDATE(), ' 07:33:00')),
    ('evt-c-003', 'cam-c-entry', 'C', '2024016', '罗三', 'entry', 0.9456, 0, CONCAT(CURDATE(), ' 07:40:00')),
    ('evt-c-004', 'cam-c-entry', 'C', '2024018', '唐五', 'entry', 0.9122, 0, CONCAT(CURDATE(), ' 07:52:00')),
    -- D栋 07:35-08:00
    ('evt-d-001', 'cam-d-entry', 'D', '2024019', '韩一', 'entry', 0.9634, 0, CONCAT(CURDATE(), ' 07:36:00')),
    ('evt-d-002', 'cam-d-entry', 'D', '2024020', '冯二', 'entry', 0.9356, 0, CONCAT(CURDATE(), ' 07:44:00')),
    ('evt-d-003', 'cam-d-entry', 'D', '2024022', '魏四', 'entry', 0.9567, 0, CONCAT(CURDATE(), ' 07:58:00')),
    -- 下午外出事件
    ('evt-a-101', 'cam-a-entry', 'A', '2024004', '赵六', 'exit',  0.8845, 0, CONCAT(CURDATE(), ' 12:15:00')),
    ('evt-a-102', 'cam-a-entry', 'A', '2024006', '周八', 'exit',  0.9122, 0, CONCAT(CURDATE(), ' 14:30:00')),
    ('evt-b-101', 'cam-b-entry', 'B', '2024012', '刘四', 'exit',  0.8956, 0, CONCAT(CURDATE(), ' 16:45:00')),
    ('evt-d-101', 'cam-d-entry', 'D', '2024021', '董三', 'exit',  0.8765, 0, CONCAT(CURDATE(), ' 15:10:00')),
    -- 陌生人记录
    ('evt-str-01', 'cam-a-entry', 'A', NULL, NULL, 'entry', 0.4234, 1, CONCAT(CURDATE(), ' 09:20:00')),
    ('evt-str-02', 'cam-b-entry', 'B', NULL, NULL, 'entry', 0.3654, 1, CONCAT(CURDATE(), ' 10:05:00'));

-- 19. 陌生人记录
INSERT IGNORE INTO dorm_stranger_record (building, confidence, event_type, detected_time) VALUES
    ('A', 0.4234, 'entry', CONCAT(CURDATE(), ' 09:20:00')),
    ('B', 0.3654, 'entry', CONCAT(CURDATE(), ' 10:05:00'));

-- 20. 昨日查寝报告 (昨日正常执行)
INSERT IGNORE INTO dorm_nightly_report (report_date, building, total_count, present_count, absent_count, late_return_count, stranger_count, unknown_count, status, trigger_type)
SELECT CURDATE() - INTERVAL 1 DAY, b.code,
       -- 各楼栋应归人数
       (SELECT COUNT(*) FROM dorm_student_assignment WHERE building = b.code),
       -- present_count
       CASE b.code
         WHEN 'A' THEN 6   -- 8人中6人已归(周八离开未归视为absent, 赵六下午出去但已归)
         WHEN 'B' THEN 3   -- 5人中3人已归(钱一缺勤, 刘四下午离开但已归)
         WHEN 'C' THEN 3   -- 5人中3人已归(谢四缺勤, 唐五已归)
         WHEN 'D' THEN 3   -- 4人中3人已归(董三下午离开但已归)
       END,
       -- absent_count
       CASE b.code
         WHEN 'A' THEN 1   -- 周八(未归)
         WHEN 'B' THEN 1   -- 钱一(缺勤)
         WHEN 'C' THEN 1   -- 谢四(缺勤)
         WHEN 'D' THEN 0
       END,
       -- late_return_count
       CASE b.code
         WHEN 'A' THEN 1   -- 赵六(晚归)
         WHEN 'B' THEN 1   -- 刘四(晚归)
         WHEN 'D' THEN 1   -- 董三(晚归)
         ELSE 0
       END,
       -- stranger_count
       CASE b.code
         WHEN 'A' THEN 1
         WHEN 'B' THEN 1
         ELSE 0
       END,
       -- unknown_count
       0,
       'COMPLETED', 'AUTO'
FROM dorm_building b;

-- 21. 昨日查寝明细 (用关联的 report_id + student 数据)
INSERT IGNORE INTO dorm_nightly_detail (report_id, student_id, student_name, building, room, class_name, status, entry_time, exit_time, is_late_return)
SELECT
    r.id,
    s.student_id,
    s.student_name,
    s.building,
    s.room,
    s.class_name,
    CASE s.student_id
      WHEN '2024009' THEN 'absent'       -- 钱一 缺勤
      WHEN '2024017' THEN 'absent'       -- 谢四 缺勤
      WHEN '2024006' THEN 'absent'       -- 周八 未归
      WHEN '2024004' THEN 'late_return'  -- 赵六 晚归
      WHEN '2024012' THEN 'late_return'  -- 刘四 晚归
      WHEN '2024021' THEN 'late_return'  -- 董三 晚归
      ELSE 'present'
    END,
    CASE
      WHEN s.student_id IN ('2024009','2024017') THEN NULL
      ELSE CONCAT(CURDATE() - INTERVAL 1 DAY, ' ', LPAD(FLOOR(7 + RAND() * 2), 2, '0'), ':', LPAD(FLOOR(0 + RAND() * 60), 2, '0'), ':00')
    END,
    CASE s.student_id
      WHEN '2024004' THEN CONCAT(CURDATE() - INTERVAL 1 DAY, ' 22:30:00')
      WHEN '2024012' THEN CONCAT(CURDATE() - INTERVAL 1 DAY, ' 22:45:00')
      WHEN '2024021' THEN CONCAT(CURDATE() - INTERVAL 1 DAY, ' 23:10:00')
      ELSE NULL
    END,
    CASE WHEN s.student_id IN ('2024004','2024012','2024021') THEN 1 ELSE 0 END
FROM dorm_student_assignment s
JOIN dorm_nightly_report r ON r.building = s.building AND r.report_date = CURDATE() - INTERVAL 1 DAY;
