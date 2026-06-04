# CampusVision AI — 测试工具集

## 前置条件
- Python 3.11+ with `kafka-python`, `opencv-python-headless`, `numpy`
- Docker Compose 正常运行（`docker compose up -d kafka redis`）
- Kafka 暴露本机端口 29092

## 安装依赖

```bash
pip install kafka-python opencv-python-headless numpy
```

## 工具列表

| 脚本 | 用途 | 用法 |
|------|------|------|
| `kafka_inject_frames.py` | 向 t_dorm_frame 注入测试帧 | `python kafka_inject_frames.py --image path --count 5` |
| `kafka_consume_verify.py` | 消费并验证帧格式 / 检测结果 | `python kafka_consume_verify.py --count 3` |
| `container_exec_test.py` | 在 Docker 容器内执行检测诊断 | `python container_exec_test.py --container cv-face-recognition` |
| `run_pipeline_test.sh` | 端到端流水线一键测试 | `bash run_pipeline_test.sh` |

## 快速入门

```bash
# 1. 安装依赖
pip install kafka-python opencv-python-headless numpy

# 2. 注入 5 帧测试
python scripts/test/kafka_inject_frames.py \
    --image face-recognition/tests/fixtures/face_test.png \
    --count 5

# 3. 验证消费
python scripts/test/kafka_consume_verify.py --count 3

# 4. 一键运行全部测试
bash scripts/test/run_pipeline_test.sh
```

## 故障排除

- `KafkaProducer` 连接超时 → 确认 `docker compose ps kafka` 正常，检查 `KAFKA_ADVERTISED_LISTENERS`
- OpenCV 无法解码图片 → 确认图片文件存在且格式正确（`file path/to/image` 或 `python -c "import cv2; print(cv2.imread('path'))"`）
- Docker exec 失败 → 确认容器名正确（`docker compose ps`）
