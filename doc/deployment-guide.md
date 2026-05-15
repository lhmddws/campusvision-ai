# CampusVision AI — 部署指南

> 本文档覆盖 CampusVision AI 的部署架构、环境要求、Docker Compose 配置和生产运维。

---

## 目录

1. [部署架构](#1-部署架构)
2. [硬件要求](#2-硬件要求)
3. [Docker Compose](#3-docker-compose)
4. [配置说明](#4-配置说明)
5. [生产运维](#5-生产运维)
6. [监控告警](#6-监控告警)
7. [灾备与恢复](#7-灾备与恢复)

---

## 1. 部署架构

### 1.1 小型部署（≤ 20 路摄像头）

```
┌───────────────────┐   ┌────────────────────┐
│   GPU 服务器       │   │   业务服务器         │
│                   │   │                    │
│  Stream Gateway   │   │  PostgreSQL        │
│  AI Engine        │   │  Redis             │
│  Platform         │   │  MinIO             │
│  Nginx            │   │  Kafka (单节点)     │
│                   │   │  Web 静态资源       │
└───────────────────┘   └────────────────────┘
       1 台                   1 台
```

### 1.2 中型部署（≤ 100 路摄像头）

```
  ┌──────────────┐    ┌──────────────┐
  │ RTSP 服务器 1  │    │ RTSP 服务器 2  │
  └──────┬───────┘    └──────┬───────┘
         │                  │
         └────────┬─────────┘
                  ▼
          ┌──────────────┐    ┌──────────────┐
          │ GPU 服务器 1   │    │ GPU 服务器 2  │
          │ AI Engine     │    │ AI Engine    │
          └──────┬───────┘    └──────┬───────┘
                 │                  │
                 └────────┬─────────┘
                          ▼
          ┌──────────────────────────────┐
          │         Kafka 集群            │
          └────────┬─────────────────────┘
                   ▼
          ┌──────────────────────────────┐
          │    Java 业务服务器 × 2         │
          │    (负载均衡)                  │
          └────────┬─────────────────────┘
                   ▼
          ┌──────────────────────────────┐
          │  PostgreSQL 主从              │
          │  Redis 集群                   │
          │  Milvus                       │
          │  MinIO (多节点)               │
          └──────────────────────────────┘
```

---

## 2. 硬件要求

### 2.1 GPU 推理节点

| 配置 | 低配（开发） | 中配（小型） | 高配（中型） |
|------|-------------|-------------|-------------|
| GPU | RTX 3060 | RTX 4070 | RTX 4090 / L40S |
| CPU | 8 核 | 16 核 | 32 核 |
| 内存 | 32 GB | 64 GB | 128 GB |
| 存储 | 500 GB SSD | 1 TB NVMe | 2 TB NVMe |
| 网络 | 千兆 | 万兆 | 万兆 |
| 摄像头路数 | ~5 路 | ~20 路 | ~40+ 路 |

### 2.2 业务节点

| 配置 | 小型 | 中型 |
|------|------|------|
| CPU | 8 核 | 16 核 × 2 |
| 内存 | 32 GB | 64 GB × 2 |
| 存储 | 500 GB SSD | 1 TB SSD × 2 |
| 网络 | 千兆 | 万兆 |

### 2.3 摄像头要求

- 分辨率：1080P
- 编码：H.264 / H.265
- 协议：RTSP
- 功能：红外夜视、宽动态
- **不推荐**：低码率模糊摄像头、非 RTSP 协议摄像头

---

## 3. Docker Compose

### 3.1 基础服务

```yaml
version: "3.9"

services:
  nginx:
    image: nginx:1.27-alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./ssl:/etc/nginx/ssl
      - web-static:/usr/share/nginx/html
    restart: always

  kafka:
    image: bitnami/kafka:3.7
    ports:
      - "9092:9092"
    environment:
      - KAFKA_CFG_NODE_ID=1
      - KAFKA_CFG_PROCESS_ROLES=broker,controller
      - KAFKA_CFG_CONTROLLER_QUORUM_VOTERS=1@kafka:9093
      - KAFKA_CFG_LISTENERS=PLAINTEXT://:9092,CONTROLLER://:9093
      - KAFKA_CFG_ADVERTISED_LISTENERS=PLAINTEXT://kafka:9092
    volumes:
      - kafka-data:/bitnami/kafka
    restart: always

  redis:
    image: redis:7.4-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data
    command: redis-server --appendonly yes
    restart: always

  postgres:
    image: postgres:16-alpine
    ports:
      - "5432:5432"
    environment:
      POSTGRES_DB: campusvision
      POSTGRES_USER: campusvision
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres-data:/var/lib/postgresql/data
    restart: always

  milvus:
    image: milvusdb/milvus:v2.4
    ports:
      - "19530:19530"
    environment:
      ETCD_ENDPOINTS: etcd:2379
      MINIO_ADDRESS: minio:9000
    depends_on:
      - etcd
      - minio
    volumes:
      - milvus-data:/var/lib/milvus
    restart: always

  etcd:
    image: bitnami/etcd:3.5
    environment:
      ALLOW_NONE_AUTHENTICATION: "yes"
    volumes:
      - etcd-data:/bitnami/etcd

  minio:
    image: minio/minio:latest
    ports:
      - "9000:9000"
      - "9001:9001"
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD}
    volumes:
      - minio-data:/data
    command: server /data --console-address ":9001"
    restart: always

volumes:
  kafka-data:
  redis-data:
  postgres-data:
  milvus-data:
  etcd-data:
  minio-data:
  web-static:
```

### 3.2 AI Engine

```yaml
services:
  ai-engine:
    build:
      context: ./campusvision-ai-engine
      dockerfile: Dockerfile
    image: campusvision/ai-engine:latest
    ports:
      - "8000:8000"
    environment:
      - KAFKA_BROKERS=kafka:9092
      - REDIS_URL=redis://redis:6379
      - MILVUS_HOST=milvus
      - CUDA_VISIBLE_DEVICES=0
      - LOG_LEVEL=info
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: 1
              capabilities: [gpu]
    volumes:
      - ./weights:/app/weights
      - ./models:/app/models
    depends_on:
      - kafka
      - redis
      - milvus
    restart: unless-stopped
```

### 3.3 Stream Gateway

```yaml
services:
  stream-gateway:
    build:
      context: ./campusvision-stream-gateway
      dockerfile: Dockerfile
    image: campusvision/stream-gateway:latest
    environment:
      - KAFKA_BROKERS=kafka:9092
      - REDIS_URL=redis://redis:6379
      - LOG_LEVEL=info
    depends_on:
      - kafka
      - redis
    restart: unless-stopped
```

### 3.4 Java Platform

```yaml
services:
  campus-platform:
    build:
      context: ./campusvision-platform
      dockerfile: Dockerfile
    image: campusvision/platform:latest
    ports:
      - "8080:8080"
    environment:
      - SPRING_PROFILES_ACTIVE=prod
      - SPRING_DATASOURCE_URL=jdbc:postgresql://postgres:5432/campusvision
      - SPRING_DATASOURCE_USERNAME=campusvision
      - SPRING_DATASOURCE_PASSWORD=${DB_PASSWORD}
      - SPRING_REDIS_HOST=redis
      - KAFKA_BOOTSTRAP_SERVERS=kafka:9092
      - MINIO_ENDPOINT=http://minio:9000
      - MINIO_ACCESS_KEY=${MINIO_ROOT_USER}
      - MINIO_SECRET_KEY=${MINIO_ROOT_PASSWORD}
    depends_on:
      - postgres
      - redis
      - kafka
      - minio
    restart: unless-stopped
```

### 3.5 Web

```yaml
services:
  campus-web:
    build:
      context: ./campusvision-web
      dockerfile: Dockerfile
    image: campusvision/web:latest
    environment:
      - VITE_API_BASE_URL=/api
      - VITE_WS_URL=wss://${DOMAIN}/ws
    depends_on:
      - campus-platform
    restart: unless-stopped
```

---

## 4. 配置说明

### 4.1 环境变量

创建 `.env` 文件：

```bash
# 数据库
DB_PASSWORD=your_secure_password_here

# MinIO
MINIO_ROOT_USER=campusvision_admin
MINIO_ROOT_PASSWORD=your_minio_password_here

# 域名
DOMAIN=campusvision.example.com

# Kafka
KAFKA_CFG_AUTO_CREATE_TOPICS_ENABLE=true
```

### 4.2 Nginx 配置

```nginx
upstream platform {
    server campus-platform:8080;
}

server {
    listen 443 ssl;
    server_name ${DOMAIN};

    ssl_certificate /etc/nginx/ssl/cert.pem;
    ssl_certificate_key /etc/nginx/ssl/key.pem;

    # Web 静态资源
    location / {
        root /usr/share/nginx/html;
        index index.html;
        try_files $uri $uri/ /index.html;
    }

    # API 代理
    location /api/ {
        proxy_pass http://platform;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    # WebSocket
    location /ws/ {
        proxy_pass http://platform;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

### 4.3 模型权重

将模型权重文件放置到 `weights/` 目录：

```text
weights/
├── yolov11n.pt           # YOLO 人体检测
├── yolov11s.pt           # YOLO 高精度模型
├── insightface_model.onnx # 人脸识别
├── reid_model.pth         # ReID
└── ...
```

---

## 5. 生产运维

### 5.1 启动

```bash
# 启动所有服务
docker compose --profile all up -d

# 启动特定服务组
docker compose up -d kafka redis postgres
docker compose up -d ai-engine stream-gateway
```

### 5.2 日志

```bash
# 实时查看
docker compose logs -f ai-engine

# 最近 1 小时
docker compose logs --since=1h ai-engine

# 导出
docker compose logs -t > deployment_logs.txt
```

### 5.3 健康检查

```bash
# Engine API
curl http://localhost:8000/health

# Platform API
curl http://localhost:8080/actuator/health

# Kafka
kcat -b localhost:9092 -L
```

### 5.4 更新

```bash
# 拉取最新代码
git pull

# 重新构建并部署
docker compose build ai-engine
docker compose up -d ai-engine

# 滚动更新（需要多节点）
docker compose up -d --scale ai-engine=2 --no-deps --no-recreate ai-engine
```

### 5.5 数据库迁移

```bash
# Platform 自动执行 Flyway 迁移
# 手动触发：
docker compose exec campus-platform java -jar app.jar --spring.flyway.enabled=true
```

---

## 6. 监控告警

### 6.1 GPU 监控

```bash
# 实时
nvidia-smi -l 1

# 持久监控（推荐 DCGM）
docker run -d --gpus all --rm \
  -p 9400:9400 \
  nvcr.io/nvidia/k8s/dcgm-exporter:latest
```

### 6.2 应用指标

- Prometheus + Grafana 采集各服务指标
- 关键指标：
  - GPU 利用率、显存占用
  - Kafka 消费延迟
  - API 响应时间 P99
  - 摄像头在线率
  - AI 推理 FPS

### 6.3 告警规则

| 指标 | 阈值 | 严重级别 |
|------|------|---------|
| GPU 温度 | > 85°C | 警告 |
| GPU 利用率持续 100% | > 5 min | 警告 |
| Kafka 延迟 | > 10s | 严重 |
| 摄像头断线 | > 5 min | 严重 |
| API 5xx 错误率 | > 1% | 严重 |
| 磁盘使用率 | > 85% | 警告 |

---

## 7. 灾备与恢复

### 7.1 数据备份

```bash
# PostgreSQL
pg_dump -h localhost -U campusvision campusvision > backup_$(date +%Y%m%d).sql

# 定期备份（crontab）
0 3 * * * pg_dump -h localhost -U campusvision campusvision > /backup/db/$(date +\%Y\%m\%d).sql
```

### 7.2 恢复

```bash
# 数据库恢复
psql -h localhost -U campusvision -d campusvision < backup_20260515.sql

# Docker 回滚
docker compose up -d ai-engine=previous
```

### 7.3 高可用策略

| 组件 | 策略 |
|------|------|
| PostgreSQL | 主从复制 + WAL 归档 |
| Redis | 哨兵模式 / 集群 |
| Kafka | 多 broker + 副本因子 ≥ 2 |
| AI Engine | 多实例 + 摄像头分片 |
| Platform | 多实例 + Nginx 负载均衡 |
| MinIO | 多节点纠删码 |

---

## 附录：端口规划

| 服务 | 端口 | 协议 |
|------|------|------|
| Nginx | 80/443 | HTTP/HTTPS |
| Kafka | 9092 | TCP |
| Redis | 6379 | TCP |
| PostgreSQL | 5432 | TCP |
| Milvus | 19530 | gRPC |
| MinIO API | 9000 | HTTP |
| MinIO Console | 9001 | HTTP |
| AI Engine API | 8000 | HTTP |
| Platform API | 8080 | HTTP |

## 附录：Docker 镜像列表

```text
campusvision/ai-engine:latest
campusvision/stream-gateway:latest
campusvision/platform:latest
campusvision/web:latest
```
