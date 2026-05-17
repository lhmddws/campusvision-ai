# 基础架构设计

> **版本**: v1.0 · **更新**: 2026-05-16  
> **用途**: 定义 CampusVision AI 所有基础服务的 Docker Compose 编排、配置和健康检查。

---

## 1. 服务总览

```yaml
# docker-compose.yml 位于项目根目录
services:
  zookeeper:  confluentinc/cp-zookeeper:7.6.1  # Kafka 依赖
  kafka:      confluentinc/cp-kafka:7.6.1       # 消息队列
  kafka-init:                                    # Topic 初始化
  redis:      redis:7-alpine                     # 缓存
  mariadb:    mariadb:10.11                      # 持久化数据库
  minio:      minio/minio:latest                 # 对象存储
```

---

## 2. 端口映射

| 服务 | 内部端口 | 外部端口 | 用途 |
|------|---------|---------|------|
| ZooKeeper | 2181 | 2181 | Kafka 协调 |
| Kafka | 9092 | 9092 | 消息队列（外部访问） |
| Kafka | 9101 | 9101 | JMX 监控 |
| Redis | 6379 | 6379 | 缓存 |
| MariaDB | 3306 | 3306 | 数据库 |
| MinIO API | 9000 | 9000 | 对象存储 API |
| MinIO Console | 9001 | 9001 | 管理控制台 |

---

## 3. 网络

所有服务接入 `campusvision` 桥接网络，容器间通过 hostname 互通：

```
campusvision (bridge)
  ├── zookeeper :2181
  ├── kafka     :9092  (advertised: kafka:9092 / localhost:9092)
  ├── redis     :6379
  ├── mariadb   :3306
  └── minio     :9000
```

---

## 4. Kafka 配置

### 关键配置

```yaml
KAFKA_MESSAGE_MAX_BYTES: 5242880        # 5MB，容纳 JPEG 帧
KAFKA_REPLICA_FETCH_MAX_BYTES: 5242880
KAFKA_AUTO_CREATE_TOPICS_ENABLE: "true" # 允许自动创建
KAFKA_LOG_RETENTION_HOURS: 48           # 帧数据保留 48h
```

### Topic 初始化 (kafka-init)

| Topic | 分区 | 保留时间 | 最大消息 | 压缩 | 用途 |
|-------|------|---------|---------|------|------|
| `t_dorm_frame` | 4 | 12h | 5MB | producer | 帧数据 |
| `t_dorm_event` | 2 | 7d | 1MB | producer | 事件数据 |
| `t_dorm_alert` | 1 | 7d | 1MB | — | 告警数据 |

### 分区策略

- `t_dorm_frame` 4 分区：每路摄像头 1 分区，保证同 camera 顺序消费
- `t_dorm_event` 2 分区：按 building hash 分区
- `t_dorm_alert` 1 分区：告警量小，单分区足够

---

## 5. Redis 配置

```yaml
redis:7-alpine
  ports:
    - "6379:6379"
  volumes:
    - redis-data:/data
  healthcheck:
    test: ["CMD", "redis-cli", "ping"]
```

### Key 设计（由 Dormitory Service 管理）

| Key 模式 | 用途 | TTL |
|----------|------|-----|
| `dorm:student:{id}` | 学生实时状态 | 无（永久） |
| `dorm:building:{bld}:status` | 楼栋概览 | 无 |
| `face:match:{id}` | 人脸特征缓存 | 3600s |

---

## 6. MariaDB 配置

```yaml
mariadb:10.11
  environment:
    MARIADB_ROOT_PASSWORD: root_dev
    MARIADB_DATABASE: dormitory
    MARIADB_USER: dormitory
    MARIADB_PASSWORD: dormitory_dev
  volumes:
    - mariadb-data:/var/lib/mysql
    - ./infra/mariadb/init.sql:/docker-entrypoint-initdb.d/init.sql
```

初始化 SQL 自动挂载到容器的 `/docker-entrypoint-initdb.d/`，首次启动时执行建表和初始数据。

---

## 7. MinIO 配置

```yaml
minio/minio:latest
  environment:
    MINIO_ROOT_USER: minioadmin
    MINIO_ROOT_PASSWORD: minioadmin123
  command: server /data --console-address ":9001"
```

### 存储策略

| Bucket | 用途 | 文件有效期 |
|--------|------|-----------|
| `face-snapshots` | 人脸抓拍 JPEG | 7 天 |
| `event-evidence` | 事件证据图 | 30 天 |

---

## 8. 健康检查

每个服务配置了 Docker healthcheck：

| 服务 | 检查命令 | 间隔 |
|------|---------|------|
| Kafka | `kafka-topics --list` | 10s |
| Redis | `redis-cli ping` | 5s |
| MariaDB | `mysqladmin ping` | 5s |
| MinIO | `curl /minio/health/live` | 10s |

---

## 9. 启动/停止

```bash
# 启动所有基础服务
docker compose up -d

# 仅启动特定服务
docker compose up -d kafka redis

# 查看状态
docker compose ps

# 查看日志
docker compose logs -f kafka

# 停止
docker compose down

# 停止并清除数据
docker compose down -v
```

### 服务依赖顺序

```
zookeeper → kafka → kafka-init
                 → redis
                 → mariadb
                 → minio
```

所有基础服务启动后，即可启动业务服务：
1. `stream-gateway`（Go）
2. `face-recognition`（Python）
3. `dormitory-service`（Java JAR）

---

## 10. 注意事项

- Kafka `advertised.listeners` 配置为双地址：容器内用 `kafka:9092`，宿主机用 `localhost:9092`
- MariaDB 数据卷持久化，`docker compose down -v` 会清除数据
- MinIO 默认不自动创建 bucket，需通过 API 或 Console 手动创建
- 开发环境密码均为弱密码，生产环境需替换
