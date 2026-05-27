# 数据库初始化设计

> **版本**: v1.0 · **更新**: 2026-05-16  
> **用途**: 定义 MariaDB 建表 DDL。

---

## 1. 设计原则

- **自动初始化**: MariaDB 通过 Docker `docker-entrypoint-initdb.d` 机制自动执行
- **生产建议**: 与本仓库保持一致，统一使用 MariaDB

---

## 2. 表结构总览（11 张表）

```
dorm_building          # 楼栋基础信息
dorm_room              # 房间基础信息
dorm_student           # 住宿学生
dorm_camera            # 摄像头设备
dorm_event_log         # 进出事件日志
dorm_attendance_record # 考勤记录
dorm_nightly_report    # 每晚查宿报告
dorm_daily_summary     # 每日汇总
dorm_stranger_record   # 陌生人记录
dorm_alert             # 告警记录
dorm_config            # 系统配置
```

---

## 3. 核心表设计

### dorm_building（楼栋）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT PK | 自增主键 |
| name | VARCHAR(50) | 楼栋名称（A/B/C/D） |
| code | VARCHAR(20) UNIQUE | 楼栋编码 |
| floor_count | INT | 楼层数 |
| room_count | INT | 房间数 |
| camera_id | VARCHAR(50) | 关联摄像头 |
| status | TINYINT | 状态：1启用 0禁用 |

### dorm_student（住宿学生）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT PK | 自增主键 |
| student_id | VARCHAR(50) UNIQUE | 学号 |
| name | VARCHAR(100) | 姓名 |
| gender | TINYINT | 性别：1男 2女 |
| building_id | BIGINT FK | 所属楼栋 |
| room_id | BIGINT FK | 所属房间 |
| class_name | VARCHAR(100) | 班级 |
| status | TINYINT | 状态：1在校 0离校 |

### dorm_event_log（事件日志）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT PK | 自增主键 |
| event_id | VARCHAR(100) | 事件唯一 ID |
| camera_id | VARCHAR(50) | 摄像头 ID |
| building | VARCHAR(20) | 楼栋 |
| student_id | VARCHAR(50) | 学号（陌生人=null） |
| event_type | VARCHAR(20) | entry/exit/unknown |
| confidence | DECIMAL(5,4) | 置信度 |
| is_stranger | TINYINT | 是否陌生人 |
| snapshot_path | VARCHAR(500) | 抓拍图 MinIO 路径 |
| occurred_at | DATETIME | 事件时间 |

**索引**: `(building, occurred_at)`, `(student_id, occurred_at)`, `(event_id)`

### dorm_nightly_report（查宿报告）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT PK | 自增主键 |
| report_date | DATE | 报告日期 |
| building_id | BIGINT FK | 楼栋 |
| total_students | INT | 应到人数 |
| present_count | INT | 已归人数 |
| absent_count | INT | 未归人数 |
| late_count | INT | 晚归人数 |
| stranger_count | INT | 陌生人次数 |
| report_time | DATETIME | 生成时间 |

**索引**: `(report_date, building_id)` 唯一索引

### dorm_alert（告警记录）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT PK | 自增主键 |
| alert_type | VARCHAR(50) | 告警类型 |
| building_id | BIGINT FK | 楼栋 |
| student_id | VARCHAR(50) | 相关学号 |
| description | TEXT | 告警描述 |
| status | TINYINT | 0未处理 1已处理 |
| created_at | DATETIME | 创建时间 |
| handled_at | DATETIME | 处理时间 |

### dorm_config（系统配置）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT PK | 自增主键 |
| config_key | VARCHAR(100) UNIQUE | 配置键 |
| config_value | TEXT | 配置值 |
| description | VARCHAR(500) | 描述 |
| updated_at | DATETIME | 更新时间 |

---

## 4. MariaDB DDL 要点

```sql
-- 使用 InnoDB 引擎，utf8mb4 字符集
ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 时间戳字段使用 CURRENT_TIMESTAMP 默认值
created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
```

---

## 6. 数据一致性

- 外键约束仅在建表时定义逻辑关系，实际生产按需启用
- `building_id` 和 `room_id` 为逻辑外键，允许为 NULL
- 事件日志表按 `occurred_at` 做分区（按月或按周）
- `dorm_config` 支持运行时动态配置，修改无需重启

---

## 7. 初始化数据

`init.sql` 在创建表结构后插入初始数据：

```sql
-- 4 栋宿舍楼
INSERT INTO dorm_building (name, code, floor_count, room_count, status) VALUES
('A栋', 'A', 6, 120, 1),
('B栋', 'B', 6, 120, 1),
('C栋', 'C', 6, 120, 1),
('D栋', 'D', 6, 120, 1);

-- 默认配置
INSERT INTO dorm_config (config_key, config_value, description) VALUES
('nightly_report_time', '23:00', '每晚查宿触发时间'),
('late_threshold_minutes', '120', '晚归阈值（分钟）'),
('stranger_alert_enabled', 'true', '陌生人告警开关');
```
