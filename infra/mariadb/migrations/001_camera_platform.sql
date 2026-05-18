-- ============================================================================
-- Migration: 001_camera_platform
-- Description: Add camera platform expansion columns to dorm_camera table
-- 
-- Adds credential management and device metadata columns to support
-- multiple camera types (RTSP, SIMULATED, USB) and centralized credential
-- storage with AES-256-GCM encryption.
--
-- Columns added: 10
--   - type             Camera type classification
--   - protocol         RTSP protocol variant
--   - host             Camera host/IP
--   - port             Network port
--   - path             RTSP URL path component
--   - username         Authentication username
--   - password_enc     AES-256-GCM encrypted password
--   - nonce            Encryption nonce (base64)
--   - key_id           Encryption key version
--   - last_health_check Last health check timestamp
--
-- Backward compatible: rtsp_url column is preserved (not dropped or altered).
-- ============================================================================

ALTER TABLE dorm_camera
  ADD COLUMN type              VARCHAR(16)  DEFAULT 'RTSP'
    COMMENT '摄像头类型: RTSP/SIMULATED/USB'
    AFTER building,

  ADD COLUMN protocol          VARCHAR(8)   DEFAULT 'rtsp'
    COMMENT '拉流协议'
    AFTER rtsp_url,

  ADD COLUMN host              VARCHAR(128)
    COMMENT '摄像头主机'
    AFTER protocol,

  ADD COLUMN port              INT          DEFAULT 554
    COMMENT '端口'
    AFTER host,

  ADD COLUMN path              VARCHAR(256)
    COMMENT 'RTSP路径'
    AFTER port,

  ADD COLUMN username          VARCHAR(64)
    COMMENT '认证用户名'
    AFTER path,

  ADD COLUMN password_enc      TEXT
    COMMENT 'AES-256-GCM加密密码: base64(nonce|ciphertext)'
    AFTER username,

  ADD COLUMN nonce             VARCHAR(32)
    COMMENT '加密随机数(base64)'
    AFTER password_enc,

  ADD COLUMN key_id            VARCHAR(16)  DEFAULT 'v1'
    COMMENT '密钥版本'
    AFTER nonce,

  ADD COLUMN last_health_check DATETIME
    COMMENT '上次健康检查时间'
    AFTER last_event_time;
