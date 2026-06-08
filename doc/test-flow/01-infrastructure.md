# 基础设施启动步骤

## Purpose

本文档描述 CampusVision AI 测试所需的基础设施启动流程，包括 Kafka、Redis、MariaDB 等核心依赖服务的启动和健康验证。

## Prerequisites

- Docker 和 Docker Compose 已安装
- 项目根目录下存在 `docker-compose.yml`
- 端口 2181、9092、29092、6379、3306 未被占用
- 磁盘空间充足（至少 5GB 用于镜像和数据卷）

## 启动步骤

### 1. 启动核心基础设施

```bash
# 在项目根目录执行
docker compose up -d kafka redis mariadb
```

该命令会按依赖顺序启动：

| 服务 | 容器名 | 端口 | 说明 |
|------|--------|------|------|
| zookeeper | cv-zookeeper | 2181 | Kafka 依赖，必须首先启动 |
| kafka | cv-kafka | 9092（容器内）/ 29092（主机） | 消息队列核心 |
| kafka-init | cv-kafka-init | — | 一次性任务，创建 3 个 topic |
| redis | cv-redis | 6379 | 人脸缓存和去重 |
| mariadb | cv-mariadb | 3306 | 业务数据存储 |

### 2. 验证服务状态

```bash
# 查看所有容器运行状态
docker compose ps

# 预期输出示例：
# NAME                 STATUS
# cv-zookeeper         Up (healthy)
# cv-kafka             Up (healthy)
# cv-kafka-init        Exited (0)
# cv-redis             Up (healthy)
# cv-mariadb           Up (healthy)
```

`kafka-init` 容器是 one-shot 任务，创建 topic 后会正常退出（状态为 `Exited (0)`），这是预期行为。

### 3. 验证各服务健康状态

#### Kafka

```bash
# 列出 topic，确认 3 个 topic 已创建
docker compose exec kafka kafka-topics --bootstrap-server localhost:9092 --list

# 预期输出：
# t_dorm_event
# t_dorm_frame
```

```bash
# 查看 topic 详情
docker compose exec kafka kafka-topics --bootstrap-server localhost:9092 --describe --topic t_dorm_frame

# 预期输出：
# Topic: t_dorm_frame  TopicId: xxx  PartitionCount: 4  ReplicationFactor: 1
# Configs: compression.type=producer,cleanup.policy=delete,...
```

#### Redis

```bash
# Redis 健康检查
docker compose exec redis redis-cli ping
# 预期输出: PONG

# 查看 Redis 信息
docker compose exec redis redis-cli info server | grep redis_version
```

#### MariaDB

```bash
# MariaDB 健康检查
docker compose exec mariadb mysqladmin ping -u sims -psims
# 预期输出: mysqld is alive

# 查看数据库表
docker compose exec mariadb mysql -u sims -psims dormitory -e "SHOW TABLES;"
# 预期输出: 11 张 dorm_ 前缀的表
```

## 配置说明

### Kafka 监听器配置

Kafka 配置了两个监听器，分别用于容器间通信和主机访问：

| 监听器 | 地址 | 用途 |
|--------|------|------|
| PLAINTEXT | kafka:9092 | 容器间通信（face-recognition、stream-gateway 使用此地址） |
| PLAINTEXT_HOST | localhost:29092 | 主机访问（测试脚本使用此地址） |

### Kafka Topic 配置

| Topic | 分区数 | 保留时间 | 压缩 | 最大消息 |
|-------|--------|----------|------|----------|
| t_dorm_frame | 4 | 12 小时 | Snappy | 5MB |
| t_dorm_event | 2 | 7 天 | Producer | 1MB |
| t_dorm_alert | 1 | 7 天 | — | — |

### Redis 配置

- 无密码认证（开发环境）
- db=0（所有服务共用）
- 数据持久化到 `redis-data` 卷

### MariaDB 配置

- 数据库名: `dormitory`
- 用户名: `sims`
- 密码: `sims`（可通过环境变量 `MARIADB_PASSWORD` 覆盖）
- Root 密码: `root_dev`（可通过环境变量 `MARIADB_ROOT_PASSWORD` 覆盖）

## 常见问题

### Kafka 无法启动

```bash
# 查看 Kafka 日志
docker compose logs kafka

# 常见原因:
# 1. 端口 9092/29092 被占用 → lsof -i :29092
# 2. Zookeeper 未就绪 → docker compose ps zookeeper
# 3. 磁盘空间不足 → df -h
```

### kafka-init 未创建 topic

```bash
# 查看初始化日志
docker compose logs kafka-init

# 手动创建 topic（如果自动创建失败）
docker compose exec kafka kafka-topics --bootstrap-server localhost:9092 \
  --create --topic t_dorm_frame --partitions 4 --replication-factor 1
```

### Redis 连接失败

```bash
# 确认容器运行
docker compose ps redis

# 测试连通性
docker compose exec redis redis-cli ping
```

### MariaDB 初始化失败

```bash
# 查看初始化日志
docker compose logs mariadb

# 检查 init.sql 挂载
docker compose exec mariadb ls /docker-entrypoint-initdb.d/

# 重新初始化（会丢失数据）
docker compose down -v
docker compose up -d mariadb
```

## 停止基础设施

```bash
# 停止所有服务（保留数据卷）
docker compose down

# 停止并清除数据卷（完全重置）
docker compose down -v
```

## 后续步骤

基础设施就绪后，可继续：

1. [Kafka 消息测试](./02-kafka-test.md)
2. [人脸检测验证](./03-face-detection.md)
3. [端到端流程](./04-end-to-end.md)
