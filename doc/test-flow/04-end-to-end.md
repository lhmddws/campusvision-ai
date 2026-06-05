# 端到端流程

## Purpose

本文档描述 CampusVision AI 的完整端到端测试流程，从 RTSP 摄像头模拟到最终事件入库，覆盖整个感知流水线的验证。

## Prerequisites

- Docker 和 Docker Compose 已安装
- ffmpeg 已安装并位于 `$PATH` 中（`ffmpeg -version` 可执行）
- Python 3.11+ 已安装
- 已安装依赖：`kafka-python`、`opencv-python-headless`、`numpy`
- 测试图片可用：`face-recognition/tests/fixtures/face_test.png`
- 所有服务已构建（`docker compose build`）

## 完整流水线架构

```
RTSP 摄像头 (模拟)
  → Mediamtx (rtsp://localhost:8554/cam_a)
  → stream-gateway (Go, 端口 8080/8081)
  → t_dorm_frame (Kafka, 4 分区, hash by building)
  → face-recognition (Python, ONNX/Haar)
  → t_dorm_event (Kafka, 2 分区)
  → dormitory-service-go (Go, 端口 8083)
  → MariaDB (端口 3306) + Redis (端口 6379)
```

## 步骤一：启动所有服务

### 方式 A：使用 Docker Compose

```bash
cd /path/to/campusvision-ai

# 构建所有镜像（首次或代码变更后）
docker compose build

# 启动所有服务
docker compose up -d

# 等待服务就绪（约 30-60 秒）
sleep 30

# 验证所有容器运行正常
docker compose ps
```

### 方式 B：逐步启动（推荐调试时使用）

```bash
# 1. 基础设施
docker compose up -d kafka redis mariadb

# 2. 等待 Kafka 就绪
sleep 15

# 3. stream-gateway
cd stream-gateway && go run cmd/main.go --config config.yaml &

# 4. face-recognition（先下载模型）
cd ../face-recognition && python -m app.download_models
cd ../face-recognition && python -m app.main --config config.yaml &

# 5. dormitory-service-go
cd ../dormitory-service-go && CONFIG_PATH=config.yaml go run ./cmd/dormitory-service/ &
```

### 验证服务就绪

```bash
# stream-gateway 健康检查
curl http://localhost:8080/health

# dormitory-service-go 健康检查
curl http://localhost:8083/health

# Redis 检查
docker compose exec redis redis-cli ping

# MariaDB 检查
docker compose exec mariadb mysqladmin ping -u sims -psims
```

## 步骤二：模拟 RTSP 摄像头

### 单摄像头模拟

```bash
python scripts/rtsp/simulate_camera.py \
    --image face-recognition/tests/fixtures/face_test.png \
    --rtsp-url rtsp://localhost:8554/cam_a \
    --fps 5
```

此命令会持续推送图片作为循环视频流，Ctrl+C 停止。

### 多摄像头模拟

```bash
# 同时模拟 4 个摄像头（cam_a, cam_b, cam_c, cam_d）
python scripts/rtsp/simulate_camera.py \
    --multi 4 \
    --image face-recognition/tests/fixtures/face_test.png \
    --fps 5
```

### 参数说明

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--image` | 必填（与 --video 互斥） | 测试图片路径（循环推送） |
| `--video` | 必填（与 --image 互斥） | 测试视频路径 |
| `--multi` | 无 | 同时启动 N 个模拟摄像头 |
| `--rtsp-url` | rtsp://localhost:8554/cam_a | RTSP 推送地址 |
| `--fps` | 5 | 帧率 |
| `--base-port` | 8554 | 多摄像头基础端口 |

### Mediamtx 配置

Mediamtx 配置在 `scripts/rtsp/mediamtx.yml`，预定义了 4 个 path：

```yaml
paths:
  cam_a:
    source: publisher
  cam_b:
    source: publisher
  cam_c:
    source: publisher
  cam_d:
    source: publisher
```

## 步骤三：运行一键流水线测试

### 命令

```bash
bash scripts/test/run_pipeline_test.sh
```

### 测试项目

该脚本依次执行以下检查：

| 序号 | 测试项 | 验证内容 |
|------|--------|----------|
| 1 | Kafka 可达 (本机:29092) | 主机可连接 Kafka |
| 2 | Kafka 可达 (容器:9092) | 容器间可连接 Kafka |
| 3 | Redis 可达 | Redis PONG 响应 |
| 4 | MariaDB 可达 | MySQL admin ping 成功 |
| 5 | face-recognition 运行中 | 容器状态 healthy |
| 6 | stream-gateway 运行中 | 容器状态 Up/healthy |
| 7 | dormitory-service-go 运行中 | 容器状态 Up/healthy |
| 8 | 注入测试帧 (5帧) | kafka_inject_frames.py 成功 |
| 9 | face-recognition 处理帧 | 日志中出现 frames_processed |

### 预期输出

```
=========================================
 CampusVision AI — Pipeline 测试套件
=========================================

[TEST] Kafka 可达 (本机:29092) ... ✅ PASS
[TEST] Kafka 可达 (容器:9092) ... ✅ PASS
[TEST] Redis 可达 ... ✅ PASS
[TEST] MariaDB 可达 ... ✅ PASS
[TEST] face-recognition 运行中 ... ✅ PASS
[TEST] stream-gateway 运行中 ... ✅ PASS
[TEST] dormitory-service-go 运行中 ... ✅ PASS
[TEST] 注入测试帧 (5帧) ... ✅ PASS
[TEST] face-recognition 处理帧 ... ✅ PASS

--- 人脸检测验证 ---
  Latest stats: frames_processed=5 faces_detected=3 events_published=2

=========================================
 结果: 9 通过, 0 失败
=========================================
```

### 退出码

- 全部通过：退出码 `0`
- 有失败项：退出码 = 失败数量

## 步骤四：验证最终结果

### 检查 t_dorm_event 消息

```bash
# 消费 face-recognition 产生的事件
docker compose exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic t_dorm_event \
  --from-beginning \
  --max-messages 5 \
  --timeout-ms 15000
```

### 检查 dormitory-service-go 日志

```bash
docker compose logs --tail 30 dormitory-service-go

# 预期包含事件处理日志：
# Processing face event: camera_id=test_cam_001, building=A
# Event deduplicated: key=dedup:test_cam_001:0
# Attendance record created for student_id=xxx
```

### 检查数据库记录

```bash
# 查看事件表
docker compose exec mariadb mysql -u sims -psims dormitory -e \
  "SELECT id, camera_id, building, event_type, created_at FROM dorm_event ORDER BY id DESC LIMIT 10;"

# 查看考勤记录
docker compose exec mariadb mysql -u sims -psims dormitory -e \
  "SELECT id, student_id, camera_id, check_time FROM dorm_attendance ORDER BY id DESC LIMIT 10;"
```

### 检查 Redis 去重键

```bash
# 查看去重键
docker compose exec redis redis-cli keys "dedup:*"

# 查看键的过期时间
docker compose exec redis redis-cli ttl "dedup:test_cam_001:0"
```

## 步骤五：使用 dormitory-service-go API 验证

### 查询摄像头状态

```bash
curl http://localhost:8083/api/cameras/status
```

### 查询考勤记录

```bash
curl http://localhost:8083/api/attendance/records?building=A
```

### 查询告警列表

```bash
curl http://localhost:8083/api/alerts
```

## 故障排除

### stream-gateway 无法连接 RTSP

```bash
# 检查 Mediamtx 是否运行
docker compose ps mediamtx

# 检查 RTSP 流是否可达
ffprobe rtsp://localhost:8554/cam_a -show_streams 2>&1 | head -20

# 查看 stream-gateway 日志
docker compose logs --tail 20 stream-gateway
```

### face-recognition 未消费帧

```bash
# 确认 t_dorm_frame 有消息
python scripts/test/kafka_consume_verify.py --count 1

# 检查 face-recognition 连接配置
docker compose exec face-recognition cat /app/config.yaml | grep kafka

# 查看 face-recognition 错误日志
docker compose logs --tail 50 face-recognition | grep -i "error\|exception\|fail"
```

### dormitory-service-go 未处理事件

```bash
# 确认 t_dorm_event 有消息
docker compose exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic t_dorm_event \
  --max-messages 1 \
  --timeout-ms 10000

# 查看 dormitory-service-go 日志
docker compose logs --tail 50 dormitory-service-go

# 检查 Redis 去重（可能事件被去重了）
docker compose exec redis redis-cli keys "dedup:*"
```

### 模拟摄像头无法推送

```bash
# 确认 ffmpeg 可用
ffmpeg -version

# 确认 Mediamtx 端口可用
lsof -i :8554

# 手动测试 Mediamtx
curl http://localhost:8554/
```

### 完整重置

如果测试环境状态混乱，可以完全重置：

```bash
# 停止并清除所有数据
docker compose down -v

# 重新构建并启动
docker compose build
docker compose up -d

# 等待就绪
sleep 30

# 重新运行测试
bash scripts/test/run_pipeline_test.sh
```

## 后续阅读

- [基础设施启动步骤](./01-infrastructure.md)
- [Kafka 消息测试](./02-kafka-test.md)
- [人脸检测验证](./03-face-detection.md)
- 测试工具说明：`scripts/test/README.md`
