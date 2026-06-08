# 人脸检测验证

## Purpose

本文档描述如何在 CampusVision AI 的 face-recognition 服务中验证人脸检测功能，包括 ONNX 模型模式和 Haar Cascade 回退模式的测试方法。

## Prerequisites

- 基础设施已启动（参考 [01-infrastructure.md](./01-infrastructure.md)）
- face-recognition 容器已运行
- 测试图片可用：`face-recognition/tests/fixtures/face_test.png`
- 已安装依赖：`kafka-python`、`opencv-python-headless`、`numpy`

## 检测模式说明

face-recognition 服务支持两种人脸检测模式：

| 模式 | 触发条件 | 精度 | 速度 | 依赖 |
|------|----------|------|------|------|
| ONNX (RetinaFace) | `config.yaml` 中 `model_path` 非空 | 高 | 中等 | `*.onnx` 模型文件 |
| Haar Cascade | `model_path` 为空或模型加载失败 | 中等 | 快 | OpenCV 内置级联分类器 |

### 确认当前检测模式

```bash
# 查看 face-recognition 配置
docker compose exec face-recognition cat /app/config.yaml | grep -A5 detection

# 查看启动日志中的模式信息
docker compose logs face-recognition 2>&1 | grep -i "model\|haar\|onnx\|fallback"
```

## 测试一：容器内检测诊断

### 命令

```bash
python scripts/test/container_exec_test.py \
    --container cv-face-recognition \
    --config config.yaml
```

### 参数说明

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--container` | cv-face-recognition | Docker 容器名 |
| `--config` | config.yaml | 容器内配置文件路径 |

### 工作原理

该脚本执行以下操作：

1. 在宿主机生成诊断 Python 脚本（临时文件）
2. 使用 `docker cp` 将脚本复制到容器内 `/tmp/container_diag.py`
3. 使用 `docker compose exec` 在容器内执行诊断脚本
4. 诊断脚本读取 `t_dorm_frame` topic 的最新 5 条消息
5. 对每条消息执行 `FaceDetector.detect()` 和 `NightModeEnhancer.enhance()`
6. 输出检测结果到 `/tmp/diag_result.txt`
7. 从容器读取结果并打印

### 预期输出

```
OK: results written to /tmp/diag_result.txt

--- Diagnostic Results ---
model_path='models/retinaface.onnx'
min_face_size=40
conf_threshold=0.5
session=ONNX
seq=105 shape=(480, 640, 3) faces=1
  face: (200,150)-(320,290) conf=0.987
seq=106 shape=(480, 640, 3) faces=1
  face: (200,150)-(320,290) conf=0.985
seq=107 shape=(480, 640, 3) faces=0
done (read 3)
```

### Haar Cascade 模式输出

当使用 Haar Cascade 回退时，输出类似：

```
model_path=''
min_face_size=40
conf_threshold=0.5
session=Haar (fallback)
seq=105 shape=(480, 640, 3) faces=1
seq=106 shape=(480, 640, 3) faces=1
seq=107 shape=(480, 640, 3) faces=0
done (read 3)
```

注意：Haar Cascade 模式不输出置信度，因为级联分类器不提供置信度分数。

## 测试二：使用 kafka_consume_verify.py 验证

### 命令

```bash
# 先注入测试帧
python scripts/test/kafka_inject_frames.py \
    --image face-recognition/tests/fixtures/face_test.png \
    --count 5

# 消费并验证
python scripts/test/kafka_consume_verify.py --count 5
```

### 预期输出

```
Topic: t_dorm_frame[0], end_offset=115, seek_to=110
  seq=110 size=(640x480) declared=(640x480) [✓] faces=1 camera=test_cam_001
    face at (200,150) 120x140
  seq=111 size=(640x480) declared=(640x480) [✓] faces=1 camera=test_cam_001
    face at (200,150) 120x140
  ...

验证完成: 检查了 5 帧
```

### 带人脸框显示（桌面环境）

```bash
python scripts/test/kafka_consume_verify.py --count 5 --display
```

`--display` 参数会打开 OpenCV 窗口，在图片上绘制检测到的人脸框。仅在有桌面环境的机器上可用。

## 测试三：ONNX 模型验证

### 确认模型文件存在

```bash
# 进入容器检查
docker compose exec face-recognition ls -la /app/models/

# 预期输出（模型文件名可能不同）：
# retinaface.onnx    (约 2-5 MB)
# arcface.onnx       (约 20-50 MB)
```

### 下载缺失的模型

```bash
# 在宿主机执行
cd face-recognition
python -m app.download_models

# 使用国内镜像（网络受限时）
python -m app.download_models --mirror https://hf-mirror.com

# 使用代理
python -m app.download_models --proxy http://127.0.0.1:7890
```

### 重新构建容器（包含模型）

```bash
docker compose build face-recognition
docker compose up -d face-recognition
```

## 测试四：Haar Cascade 回退验证

当 ONNX 模型不可用时，系统自动回退到 Haar Cascade。

### 触发回退模式

```bash
# 方法 1：修改配置，将 model_path 设为空
docker compose exec face-recognition sed -i 's/model_path: .*/model_path: ""/' /app/config.yaml

# 方法 2：重命名模型文件
docker compose exec face-recognition mv /app/models/retinaface.onnx /app/models/retinaface.onnx.bak

# 重启服务
docker compose restart face-recognition
```

### 验证回退生效

```bash
# 查看启动日志
docker compose logs face-recognition 2>&1 | grep -i "haar\|fallback\|cascade"

# 预期输出：
# Using Haar Cascade fallback (ONNX model not available)
```

### 恢复 ONNX 模式

```bash
# 恢复模型文件
docker compose exec face-recognition mv /app/models/retinaface.onnx.bak /app/models/retinaface.onnx

# 恢复配置
docker compose exec face-recognition sed -i 's/model_path: ""/model_path: models\/retinaface.onnx/' /app/config.yaml

# 重启
docker compose restart face-recognition
```

## 常见问题

### faces_detected=0

**可能原因：**

1. 测试图片中无人脸或人脸太小
2. ONNX 模型未加载，Haar Cascade 参数不当
3. 图片解码失败

**排查：**

```bash
# 确认图片可解码
python -c "import cv2; img=cv2.imread('face-recognition/tests/fixtures/face_test.png'); print(img.shape if img is not None else 'FAILED')"

# 运行容器内诊断
python scripts/test/container_exec_test.py --container cv-face-recognition

# 检查 face-recognition 日志中的错误
docker compose logs --tail 50 face-recognition | grep -i "error\|exception\|warn"
```

### 容器 exec 失败

```
Error: No such container: cv-face-recognition
```

**排查：**

```bash
# 确认容器名
docker compose ps

# 如果容器名为其他名称，使用 --container 参数指定
python scripts/test/container_exec_test.py --container <实际容器名>
```

### ONNX 模型加载失败

```
onnxruntime.capi.onnxruntime_pybind11_state.InvalidGraph: ...
```

**原因：** 模型文件损坏或版本不兼容。

**解决：**

```bash
# 删除并重新下载
docker compose exec face-recognition rm -f /app/models/*.onnx
cd face-recognition && python -m app.download_models
docker compose restart face-recognition
```

## 后续步骤

人脸检测验证通过后，继续：

1. [端到端流程](./04-end-to-end.md)
