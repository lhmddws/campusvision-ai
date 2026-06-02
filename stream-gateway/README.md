# stream-gateway

RTSP 视频流网关 — 从宿舍摄像头采集 RTSP 流，通过 ffmpeg 解码为原始帧，经运动检测后发布到 Kafka。

## 概述

stream-gateway 是 CampusVision AI 感知管线的第一跳：

```
RTSP 摄像头 (A/B/C/D)
  → ffmpeg 解码 (YUV420P)
  → 运动检测 (160×90 Y 平面差分)
  → Kafka Producer → t_dorm_frame (hash by building, Snappy)
```

| 属性 | 值 |
|---|---|
| 语言 | Go 1.26 |
| 端口 | 8080 (健康检查), 8081 (管理 API) |
| 入口 | `cmd/main.go` |
| Kafka Topic | `t_dorm_frame` (4 分区, hash by `building`, Snappy 压缩) |

## 目录结构

```
stream-gateway/
├── cmd/main.go              # 入口: 配置加载、信号处理、DB 轮询、HTTP 服务
├── internal/
│   ├── camera/              # 摄像头管理: goroutine 生命周期、DB 同步
│   ├── config/              # YAML 配置结构体
│   ├── crypto/              # AES-256-GCM RTSP 密码加解密
│   ├── decoder/             # ffmpeg 子进程: RTSP → 原始 YUV420P 帧
│   ├── frame/               # 运动检测: Y 平面降采样差分
│   ├── health/              # 健康检查 HTTP handler
│   ├── kafka/               # Kafka Producer 封装 (hash balancer)
│   └── management/          # 管理 API (X-Management-Key 鉴权)
├── config.yaml              # 本地开发配置
├── config.docker.yaml       # Docker Compose 配置覆盖
├── Dockerfile
└── go.mod
```

## 快速开始

### 前置条件

- Go 1.26+
- ffmpeg 已安装并在 `$PATH` 中
- Kafka 运行中 (`docker compose up -d kafka`)
- MariaDB 运行中 (可选, 用于动态摄像头同步)

### 本地运行

```bash
cd stream-gateway
go run cmd/main.go --config config.yaml
```

### Docker 运行

```bash
docker compose up -d stream-gateway
```

## 配置

### config.yaml

```yaml
frame:
  fps_day: 5              # 白天帧率
  fps_night: 1            # 夜间帧率
  jpeg_quality: 80
  width: 1280
  height: 720
  dynamic_extraction: true # 启用运动检测动态抽帧
  motion_threshold: 0.05   # 运动阈值 (160×90 Y 平面均值绝对差分)

kafka:
  brokers: ["localhost:9092"]
  topic: "t_dorm_frame"
  compression: "snappy"
  batch_size: 65536

rtsp:
  reconnect_interval: 5s
  read_timeout: 10s
  max_reconnect_attempts: 0  # 0 = 无限重试

health:
  port: 8080

management:
  port: 8081
  bind_address: "127.0.0.1"
  management_key: ""         # 空 = 不启用鉴权

database:
  dsn: "root:root@tcp(localhost:3306)/dormitory"
  driver: "mysql"
  poll_interval: 30s         # DB 轮询间隔
```

### 环境变量

| 变量 | 说明 |
|---|---|
| `CAMERA_ENCRYPTION_KEY` | 32 字节 AES-256-GCM 密钥, 用于 RTSP 密码解密。未设置时使用内置开发密钥 |

> **注意**: `CAMERA_ENCRYPTION_KEY` 必须与 `dormitory-service-go` 保持一致，否则跨模块密码解密会失败。

## 核心机制

### 动态抽帧

stream-gateway 不是固定帧率抽帧，而是根据画面运动量动态决定：

1. ffmpeg 解码输出 YUV420P 原始帧
2. Y 平面降采样到 160×90 灰度图
3. 与上一帧计算均值绝对差分 (MAD)
4. 超过 `motion_threshold` 时才发布帧到 Kafka

白天最高 `fps_day` (5fps)，夜间降至 `fps_night` (1fps)。

### 摄像头管理

- **DB 轮询**: 每 30s 从 `dorm_camera` 表同步摄像头列表 (需配置 `database.dsn`)
- **管理 API**: 通过 8081 端口动态增删摄像头 (需 `X-Management-Key` 鉴权)
- **生命周期**: 每个摄像头独立 goroutine，支持启停和重连

### 优雅关闭

信号 → 停止所有摄像头 goroutine → HTTP 服务 Shutdown → context cancel

## 测试

```bash
cd stream-gateway && go test ./...
```

| 测试文件 | 覆盖范围 |
|---|---|
| `internal/health/handler_test.go` | 健康检查 handler |
| `internal/management/handler_test.go` | 管理 API handler |
| `internal/config/camera_config_test.go` | 配置解析 |
| `internal/crypto/service_test.go` | AES-256-GCM 加解密 |

## API

### 健康检查 (port 8080)

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/health` | 服务健康状态 |

### 管理 API (port 8081)

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/cameras` | 列出所有摄像头 |
| POST | `/cameras` | 添加摄像头 |
| DELETE | `/cameras/:id` | 删除摄像头 |
| GET | `/status` | 网关运行状态 |

> 管理 API 需设置 `management.management_key` 并在请求头中携带 `X-Management-Key`。

详细接口定义见 [`doc/api/stream-gateway-api.json`](../doc/api/stream-gateway-api.json)。

## 注意事项

- **ffmpeg 依赖**: decoder 通过子进程调用 ffmpeg，必须在 `$PATH` 中可用
- **帧大小硬编码**: `width × height × 3 / 2` 字节 (YUV420P)，无校验
- **无 Producer 背压**: Kafka 慢时帧会在内存中无限排队
- **日志级别未实现**: `config.Log.Level` 已定义但代码使用 stdlib `log.Printf`
