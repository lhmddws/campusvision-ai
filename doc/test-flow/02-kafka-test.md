# Kafka 消息测试

## Purpose

本文档描述如何测试 CampusVision AI 的 Kafka 消息流，包括向 `t_dorm_frame` topic 注入测试帧、消费验证消息格式，以及排查消息处理问题。

## Prerequisites

- 基础设施已启动（参考 [01-infrastructure.md](./01-infrastructure.md)）
- Python 3.11+ 已安装
- 已安装依赖：`kafka-python`、`opencv-python-headless`、`numpy`

```bash
pip install kafka-python opencv-python-headless numpy
```

- 测试图片可用：`face-recognition/tests/fixtures/face_test.png`

## 测试一：注入测试帧

### 命令

```bash
cd /path/to/campusvision-ai

python scripts/test/kafka_inject_frames.py \
    --image face-recognition/tests/fixtures/face_test.png \
    --count 10 \
    --building A \
    --camera test_cam_001 \
    --interval 0.1
```

### 参数说明

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--image` | 必填 | 测试图片路径 |
| `--count` | 10 | 发送的帧数量 |
| `--topic` | t_dorm_frame | 目标 Kafka topic |
| `--brokers` | localhost:29092 | Kafka broker 地址（主机访问） |
| `--building` | A | 楼栋标识（用作分区键） |
| `--camera` | test_cam_001 | 摄像头 ID |
| `--interval` | 0.1 | 帧间隔（秒） |
| `--key` | building 值 | 自定义分区键 |

### 预期输出

```
加载图片: face-recognition/tests/fixtures/face_test.png
图片尺寸: 640x480, base64: 45230 bytes
  seq=  0 offset=100
  seq=  1 offset=101
  seq=  2 offset=102
  ...
  seq=  9 offset=109
完成: 发送 10 帧到 t_dorm_frame
```

### 消息格式

注入的每条消息为 JSON，包含以下字段：

```json
{
  "camera_id": "test_cam_001",
  "building": "A",
  "frame_sequence": 0,
  "frame_data": "<base64-encoded JPEG>",
  "frame_width": 640,
  "frame_height": 480,
  "timestamp": 1717500000000
}
```

## 测试二：消费并验证帧

### 命令

```bash
python scripts/test/kafka_consume_verify.py \
    --count 5 \
    --partition 0
```

### 参数说明

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--count` | 3 | 要验证的帧数量 |
| `--brokers` | localhost:29092 | Kafka broker 地址 |
| `--topic` | t_dorm_frame | 目标 topic |
| `--partition` | 0 | 分区编号 |
| `--seek-latest` | 5 | 从末尾前移 N 帧开始读取 |
| `--display` | false | 显示人脸检测框（需要桌面环境） |

### 预期输出

```
Topic: t_dorm_frame[0], end_offset=110, seek_to=105
  seq=105 size=(640x480) declared=(640x480) [✓] faces=1 camera=test_cam_001
    face at (200,150) 120x140
  seq=106 size=(640x480) declared=(640x480) [✓] faces=1 camera=test_cam_001
    face at (200,150) 120x140
  seq=107 size=(640x480) declared=(640x480) [✓] faces=1 camera=test_cam_001
    face at (200,150) 120x140

验证完成: 检查了 5 帧
```

### 验证内容

消费验证脚本执行以下检查：

1. **字段完整性**：验证 7 个必要字段是否存在
2. **帧解码**：验证 base64 数据可正确解码为图片
3. **尺寸一致性**：对比声明的尺寸与实际解码后的尺寸
4. **人脸检测**：使用 Haar Cascade 检测人脸并输出位置

## 测试三：使用 Kafka 命令行工具

### 查看 topic 列表

```bash
docker compose exec kafka kafka-topics --bootstrap-server localhost:9092 --list
```

### 查看 topic 详情

```bash
# t_dorm_frame 分区信息
docker compose exec kafka kafka-topics --bootstrap-server localhost:9092 \
  --describe --topic t_dorm_frame

# t_dorm_event 分区信息
docker compose exec kafka kafka-topics --bootstrap-server localhost:9092 \
  --describe --topic t_dorm_event
```

### 消费原始消息

```bash
# 消费 t_dorm_frame（从最新开始）
docker compose exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic t_dorm_frame \
  --from-beginning \
  --max-messages 3 \
  --timeout-ms 10000

# 消费 t_dorm_event（face-recognition 的处理结果）
docker compose exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic t_dorm_event \
  --from-beginning \
  --max-messages 5 \
  --timeout-ms 15000
```

### 查看 topic 水位（offset）

```bash
docker compose exec kafka kafka-run-class kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 \
  --topic t_dorm_frame
```

## 测试四：验证 face-recognition 消费处理

### 查看 face-recognition 日志

```bash
# 查看最近的处理日志
docker compose logs --tail 20 face-recognition

# 持续跟踪日志
docker compose logs -f face-recognition
```

### 预期日志内容

face-recognition 处理帧后，日志中应包含：

```
frames_processed=5 faces_detected=3 events_published=2
processing_stats: latency_ms=45.2, faces_per_frame=0.6
```

## 常见问题

### KafkaProducer 连接超时

```
kafka.errors.KafkaTimeoutError: KafkaTimeoutError: Failed to update metadata after 60.0 secs.
```

**排查步骤：**

```bash
# 1. 确认 Kafka 容器运行
docker compose ps kafka

# 2. 确认端口映射正确
docker compose exec kafka kafka-topics --bootstrap-server localhost:9092 --list

# 3. 检查 advertised.listeners 配置
docker compose exec kafka cat /etc/confluent/docker/kafka.properties | grep advertised
```

### 无法解码图片

```
ValueError: 无法解码图片: path/to/image.png
```

**排查步骤：**

```bash
# 确认文件存在
file face-recognition/tests/fixtures/face_test.png

# 使用 OpenCV 测试
python -c "import cv2; img=cv2.imread('face-recognition/tests/fixtures/face_test.png'); print(img.shape if img is not None else 'DECODE FAILED')"
```

### 消费不到消息

```bash
# 确认消息已写入
python scripts/test/kafka_consume_verify.py --count 1 --seek-latest 100

# 查看所有分区的 offset
docker compose exec kafka kafka-run-class kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 --topic t_dorm_frame

# 检查 face-recognition 是否正在消费
docker compose logs --tail 50 face-recognition | grep -i "error\|exception\|fail"
```

### 分区数据不均衡

`t_dorm_frame` 使用 hash 分区器，以 `building` 为分区键。如果只注入一个楼栋的数据，所有消息会落入同一分区。

```bash
# 注入多个楼栋的数据以测试分区分布
python scripts/test/kafka_inject_frames.py --image path/to/img.png --building A --count 5
python scripts/test/kafka_inject_frames.py --image path/to/img.png --building B --count 5
python scripts/test/kafka_inject_frames.py --image path/to/img.png --building C --count 5
```

## 后续步骤

消息流验证通过后，继续：

1. [人脸检测验证](./03-face-detection.md)
2. [端到端流程](./04-end-to-end.md)
