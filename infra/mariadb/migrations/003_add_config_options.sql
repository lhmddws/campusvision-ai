-- Migration 003: Add config_options column for select-type configs
-- Allows defining a JSON array of possible values, e.g. ["Asia/Shanghai","America/New_York"]

ALTER TABLE dorm_config
    ADD COLUMN config_options TEXT DEFAULT NULL COMMENT '可选值列表(JSON数组)' AFTER default_value;
