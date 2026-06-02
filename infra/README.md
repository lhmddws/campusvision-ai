# infra

CampusVision AI infrastructure — database initialization scripts and migration files.

## Overview

`infra/` contains MariaDB database initialization DDL and incremental migration files, auto-loaded by Docker Compose.

```
infra/
└── mariadb/
    ├── init.sql             # Init script (auto-executed on first docker-compose startup)
    └── migrations/          # Incremental migrations (applied manually)
        ├── 001_camera_platform.sql
        ├── 002_face_embedding.sql
        └── README.md
```

## Database

| Property | Value |
|---|---|
| Engine | MariaDB |
| Database | `dormitory` |
| Charset | utf8mb4 / utf8mb4_unicode_ci |
| Port | 3306 |

## Initialization Script (init.sql)

Auto-executed by the `mariadb` Docker Compose service on first startup. Creates 11 business tables:

### Core Business Tables

| Table | Description |
|---|---|
| `dorm_student_assignment` | Student dormitory assignments (synced from student management system) |
| `dorm_student_status` | Real-time on-campus student status |
| `dorm_entry_exit_event` | Entry/exit event log (core) |
| `dorm_camera` | Camera registration and configuration |
| `dorm_face_embedding` | Face feature vector storage |
| `dorm_alert` | Alert records |
| `dorm_attendance_record` | Daily attendance records |
| `dorm_attendance_report` | Attendance summary reports |

### System Configuration Tables

| Table | Description |
|---|---|
| `dorm_system_config` | System parameter configuration |
| `dorm_behavior_event` | Behavior analysis events |
| `dorm_stranger_record` | Stranger records |

### Design Conventions

- Table names prefixed with `dorm_`
- Engine: InnoDB
- Primary key: `BIGINT AUTO_INCREMENT`
- Timestamps: `DATETIME` + `CURRENT_TIMESTAMP`
- Column comments: Chinese COMMENT
- Index naming: `idx_{table_abbrev}_{column_name}`

## Migration Files

Migration files are in `mariadb/migrations/`, sequentially numbered:

```
001_camera_platform.sql    # Camera platform field extensions
002_face_embedding.sql     # Face feature table adjustments
```

### Applying Migrations

Migrations must be **applied manually** (no Flyway or golang-migrate):

```bash
# Apply a single migration
docker compose exec -T mariadb mysql -uroot -proot dormitory < infra/mariadb/migrations/001_camera_platform.sql

# Apply all migrations in order
for f in infra/mariadb/migrations/[0-9]*.sql; do
  echo "Applying $f..."
  docker compose exec -T mariadb mysql -uroot -proot dormitory < "$f"
done
```

> Migration execution records are maintained in `mariadb/migrations/README.md`.

## Docker Compose Integration

MariaDB service configuration in `docker-compose.yml`:

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

## Other Infrastructure

The full infrastructure stack managed by Docker Compose:

| Service | Port | Description |
|---|---|---|
| Zookeeper | 2181 | Kafka dependency |
| Kafka | 9092 | Message broker (3 topics) |
| Redis | 6379 | Event dedup + caching |
| MariaDB | 3306 | Business data storage |
| MinIO | 9000/9001 | Object storage (currently unused) |

Quick start minimal dev environment:

```bash
docker compose up -d kafka redis
```
