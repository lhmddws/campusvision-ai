-- ============================================================================
-- Migration: 004_sanitize_rtsp
-- Description: Idempotent credential column guard for dorm_camera table.
--
-- Ensures all camera credential columns exist (password_enc, nonce, key_id)
-- regardless of whether they were already created by migration 001 on older
-- DB instances, or if the DB was seeded from scratch via init.sql.
--
-- Also backfills host/port/path/username columns for the same robustness.
--
-- Idempotent: uses MariaDB's IF NOT EXISTS column syntax (MariaDB 10.5+).
-- rtsp_url column is preserved (not dropped or altered) — only masked in API.
-- ============================================================================

-- Credential columns (added by 001, idempotent guard)
ALTER TABLE dorm_camera
  ADD COLUMN IF NOT EXISTS password_enc      TEXT
    COMMENT 'AES-256-GCM加密密码: base64(nonce|ciphertext)'
    AFTER username;

ALTER TABLE dorm_camera
  ADD COLUMN IF NOT EXISTS nonce             VARCHAR(32)
    COMMENT '加密随机数(base64)'
    AFTER password_enc;

ALTER TABLE dorm_camera
  ADD COLUMN IF NOT EXISTS key_id            VARCHAR(16) DEFAULT 'v1'
    COMMENT '密钥版本'
    AFTER nonce;

-- Platform columns (added by 001, idempotent guard for completeness)
ALTER TABLE dorm_camera
  ADD COLUMN IF NOT EXISTS type              VARCHAR(16) DEFAULT 'RTSP'
    COMMENT '摄像头类型: RTSP/SIMULATED/USB'
    AFTER building;

ALTER TABLE dorm_camera
  ADD COLUMN IF NOT EXISTS protocol          VARCHAR(8)  DEFAULT 'rtsp'
    COMMENT '拉流协议'
    AFTER rtsp_url;

ALTER TABLE dorm_camera
  ADD COLUMN IF NOT EXISTS host              VARCHAR(128)
    COMMENT '摄像头主机'
    AFTER protocol;

ALTER TABLE dorm_camera
  ADD COLUMN IF NOT EXISTS port              INT         DEFAULT 554
    COMMENT '端口'
    AFTER host;

ALTER TABLE dorm_camera
  ADD COLUMN IF NOT EXISTS path              VARCHAR(256)
    COMMENT 'RTSP路径'
    AFTER port;

ALTER TABLE dorm_camera
  ADD COLUMN IF NOT EXISTS username          VARCHAR(64)
    COMMENT '认证用户名'
    AFTER path;
