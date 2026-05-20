-- ============================================================================
-- Migration: V002__face_embedding
-- Description: Create face_embedding table for face recognition pipeline
--
-- This table stores face feature vectors (512-dim float embeddings as BLOB)
-- and associated metadata for the face matching pipeline.
--
-- Columns:
--   id           Auto-increment primary key
--   name         Student display name
--   student_id   Unique student ID (used as lookup key for face matching)
--   embedding    512-dim float vector packed as BLOB (2048 bytes)
--   image_path   Path to the source face image
--   created_at   Record creation timestamp
--   updated_at   Record update timestamp (auto-updated)
--
-- Backward compatible: No existing tables are modified.
-- ============================================================================

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
