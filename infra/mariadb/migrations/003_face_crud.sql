-- Migration: 003_face_crud
-- Description: Add room_number column to face_embedding for face CRUD management

ALTER TABLE face_embedding
    ADD COLUMN room_number VARCHAR(16) COMMENT '房间号' AFTER student_id;