# dormitory-service-go

宿舍管理业务服务 — 消费 Kafka 人脸事件，处理考勤记录、告警、摄像头管理、人脸匹配，对外提供 HTTP API。

## 概述

dormitory-service-go 是 CampusVision AI 的业务层，连接感知管线与前端管理界面：

```
t_dorm_event (Kafka) → EventConsumer → 考勤记录 + 学生状态更新
t_dorm_alert (Kafka) → AlertConsumer → (骨架, 待实现)

前端 SPA → Gin HTTP API (:8083) → MariaDB + Redis
```

| 属性 | 值 |
|---|---|
| 语言 | Go 1.26 |
| 端口 | 8083 |
| 入口 | `cmd/dormitory-service/main.go` |
| 框架 | Gin + sqlx + go-redis + Viper |
| 消费 Topic | `t_dorm_event`, `t_dorm_alert` |

## 目录结构

```
dormitory-service-go/
├── cmd/dormitory-service/
│   └── main.go              # 入口: 依赖注入、路由注册、Kafka 消费者启动
├── internal/
│   ├── client/              # PushClient → stream-gateway 通知
│   ├── config/              # Viper 配置 (YAML + 环境变量 + Spring Boot 兼容)
│   ├── consumer/            # Kafka 消费者 (EventConsumer, AlertConsumer)
│   ├── handler/             # Gin HTTP handlers (camera, record, alert, config, face)
│   ├── middleware/           # JWT 鉴权 + CORS
│   ├── model/
│   │   ├── dto/             # 请求/响应类型
│   │   ├── entity/          # DB 实体 (12 个, db tag 映射)
│   │   └── enums/           # 领域枚举 (EventType, AlertType, ...)
│   ├── redis/               # go-redis 封装 + 事件去重
│   ├── repository/          # sqlx 仓储 + 泛型 BaseRepository[T]
│   ├── scheduler/           # 定时任务 (robfig/cron: 每晚报告 + 健康检查)
│   ├── service/             # 业务逻辑 (camera, record, alert, config, report)
│   └── util/                # AES-256-GCM 密码加密
├── config.yaml              # 本地开发配置
├── config.docker.yaml       # Docker Compose 配置覆盖
├── Dockerfile
└── go.mod
```

## 快速开始

### 前置条件

- Go 1.26+
- MariaDB 运行中 (数据库 `dormitory`，表由 `infra/mariadb/init.sql` 初始化)
- Redis 运行中
- Kafka 运行中

### 本地运行

```bash
cd dormitory-service-go
CONFIG_PATH=config.yaml go run ./cmd/dormitory-service/
```

### Docker 运行

```bash
docker compose up -d dormitory-service-go
```

## 配置

### config.yaml

```yaml
server:
  port: 8083

database:
  dsn: "root:root_dev@tcp(localhost:3306)/dormitory?charset=utf8mb4&parseTime=True&loc=Asia%2FShanghai"
  driver: mysql
  max_open_conns: 25
  max_idle_conns: 10

redis:
  host: "127.0.0.1"
  port: 6379
  db: 0
  password: ""

kafka:
  brokers: ["localhost:9092"]
  event_topic: "t_dorm_event"
  alert_topic: "t_dorm_alert"
  group_id: "dormitory-service-group"

jwt:
  secret: "${JWT_SECRET:your-256-bit-secret}"
  expiration_hours: 24

log:
  level: "info"
```

### 环境变量

| 变量 | 说明 |
|---|---|
| `CONFIG_PATH` | 配置文件路径 (默认 `config.yaml`) |
| `JWT_SECRET` | JWT 签名密钥，生产环境必须设置，需与主后端一致 |
| `CAMERA_ENCRYPTION_KEY` | 32 字节 AES-256-GCM 密钥，需与 stream-gateway 保持一致 |

> 配置加载优先级: 默认值 < config.yaml < 环境变量。支持 Spring Boot 风格环境变量 (`SPRING_DATASOURCE_URL`, `KAFKA_BOOTSTRAP_SERVERS` 等)。

## API 概览

所有接口返回统一 `{code, message, data}` 信封格式。

### 摄像头管理

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/cameras` | 摄像头列表 |
| POST | `/api/cameras` | 注册摄像头 |
| PUT | `/api/cameras/:id` | 更新摄像头 |
| DELETE | `/api/cameras/:id` | 删除摄像头 |
| GET | `/api/cameras/:id/status` | 摄像头状态 |

### 考勤与事件

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/records` | 进出记录查询 |
| GET | `/api/events` | 事件列表 |
| GET | `/api/attendance` | 考勤统计 |

### 告警

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/alerts` | 告警列表 |
| PUT | `/api/alerts/:id/handle` | 处理告警 |

### 人脸

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/face/match` | 人脸特征匹配 (余弦相似度) |
| POST | `/api/face/embed` | 人脸特征提取 (骨架, 返回 null) |

### 系统配置

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/configs` | 系统配置列表 |
| PUT | `/api/configs/:key` | 更新配置项 |

详细接口定义见 [`doc/api/dormitory-service-api.json`](../doc/api/dormitory-service-api.json)。

## 核心机制

### 事件消费

EventConsumer 从 `t_dorm_event` 消费人脸事件，直接调用 repository 层 (跳过 service)：

1. 解析 JSON 事件消息
2. Redis 去重检查 (key: `dedup:{camera_id}:{frame_sequence}`, TTL 3600s)
3. 更新学生在校状态 (`dorm_student_status`)
4. 写入进出事件记录 (`dorm_entry_exit_event`)
5. 陌生人检测 → 创建告警 (`dorm_alert`)

### 定时任务

| 任务 | 调度 | 说明 |
|---|---|---|
| 每晚考勤报告 | 每天 23:00 | 生成当日考勤汇总 (骨架实现) |
| 摄像头健康检查 | 每 5 分钟 | 轮询 stream-gateway 健康端点 |

### 泛型仓储

`BaseRepository[T]` 基于 Go 泛型 + 反射，提供通用 CRUD 操作。实体必须使用 `db:"column"` tag 映射数据库列。

## 测试

```bash
cd dormitory-service-go && go test ./...
```

目前仅 `repository/base_test.go` 一个测试文件，覆盖泛型 CRUD 操作 (go-sqlmock + testify)。

## 注意事项

- **CONFIG_PATH**: 使用环境变量加载配置，不是 CLI flag (与 stream-gateway 不同)
- **JWT 开发密钥**: 默认 `your-256-bit-secret`，生产环境必须通过 `JWT_SECRET` 设置
- **AES 密钥同步**: `CAMERA_ENCRYPTION_KEY` 需与 stream-gateway 保持一致
- **AlertConsumer 骨架**: 当前仅记录日志并提交 offset，无实际告警处理逻辑
- **FaceMatch 性能**: 使用 O(n) 全表扫描 + 余弦相似度，大规模场景需优化
- **摄像头上限**: 硬编码 50 台，通过 `FindAll()` 计数检查 (非原子操作)
