# infra

CampusVision AI 基础设施 — 数据库初始化脚本与迁移文件。

## 概述

`infra/` 包含 MariaDB 数据库的初始化 DDL 和增量迁移文件，由 Docker Compose 自动加载。

```
infra/
└── mariadb/
    ├── init.sql             # 初始化脚本 (docker-compose 首次启动自动执行)
    └── migrations/          # 增量迁移 (手动执行)
        ├── 001_camera_platform.sql
        ├── 002_face_embedding.sql
        └── README.md
```

## 数据库

| 属性 | 值 |
|---|---|
| 引擎 | MariaDB |
| 数据库名 | `dormitory` |
| 字符集 | utf8mb4 / utf8mb4_unicode_ci |
| 端口 | 3306 |

## 初始化脚本 (init.sql)

由 Docker Compose 的 `mariadb` 服务在首次启动时自动执行，创建 11 张业务表：

### 核心业务表

| 表名 | 说明 |
|---|---|
| `dorm_student_assignment` | 学生宿舍分配 (从学管系统同步) |
| `dorm_student_status` | 人员在校实时状态 |
| `dorm_entry_exit_event` | 进出事件流水 (核心) |
| `dorm_camera` | 摄像头注册与配置 |
| `dorm_face_embedding` | 人脸特征向量存储 |
| `dorm_alert` | 告警记录 |
| `dorm_attendance_record` | 每日考勤记录 |
| `dorm_attendance_report` | 考勤汇总报告 |

### 系统配置表

| 表名 | 说明 |
|---|---|
| `dorm_system_config` | 系统参数配置 |
| `dorm_behavior_event` | 行为分析事件 |
| `dorm_stranger_record` | 陌生人记录 |

### 设计规范

- 表名统一 `dorm_` 前缀
- 引擎: InnoDB
- 主键: `BIGINT AUTO_INCREMENT`
- 时间列: `DATETIME` + `CURRENT_TIMESTAMP`
- 列注释: 中文 COMMENT
- 索引命名: `idx_{表缩写}_{列名}`

## 迁移文件

迁移文件位于 `mariadb/migrations/`，按序号递增命名：

```
001_camera_platform.sql    # 摄像头平台字段扩展
002_face_embedding.sql     # 人脸特征表调整
```

### 执行迁移

迁移需要**手动执行**（项目未使用 Flyway 或 golang-migrate）：

```bash
# 执行单个迁移
docker compose exec -T mariadb mysql -uroot -proot dormitory < infra/mariadb/migrations/001_camera_platform.sql

# 按顺序执行所有迁移
for f in infra/mariadb/migrations/[0-9]*.sql; do
  echo "Applying $f..."
  docker compose exec -T mariadb mysql -uroot -proot dormitory < "$f"
done
```

> 迁移执行记录维护在 `mariadb/migrations/README.md` 中。

## Docker Compose 集成

`docker-compose.yml` 中 MariaDB 服务配置：

```yaml
mariadb:
  image: mariadb:11
  environment:
    MARIADB_ROOT_PASSWORD: root
    MARIADB_DATABASE: dormitory
  volumes:
    - ./infra/mariadb/init.sql:/docker-entrypoint-initdb.d/init.sql
  ports:
    - "3306:3306"
```

## 其他基础设施

项目通过 Docker Compose 管理的完整基础设施栈：

| 服务 | 端口 | 说明 |
|---|---|---|
| Zookeeper | 2181 | Kafka 依赖 |
| Kafka | 9092 | 消息队列 (3 topics) |
| Redis | 6379 | 事件去重 + 缓存 |
| MariaDB | 3306 | 业务数据存储 |
| MinIO | 9000/9001 | 对象存储 (当前未使用) |

快速启动最小开发环境：

```bash
docker compose up -d kafka redis
```
