# 测试环境技术设计

> **版本**: v1.0 · **更新**: 2026-05-16  
> **用途**: 在无真实 RTSP 摄像头和学管 API 时，模拟 4 路摄像头画面注入 Kafka，辅助感知层和业务层开发调试。

---

## 1. 架构概览

```
Web Dashboard (浏览器)
        │
        ▼
┌──────────────────┐     POST /api/cameras/{id}/simulate
│  FastAPI Server  │─────────────────────────────────┐
│  (uvicorn)       │                                 │
│  :8082           │── GET /api/cameras/{id}/frame   │
│                  │── GET /api/events               │
│                  │── GET /api/health               │
│                  │── GET /api/scenarios/random      │
└────────┬─────────┘                                 │
         │                                           │
         │ Kafka producer (kafka-python)              │
         ▼                                           ▼
┌──────────────────┐                    ┌──────────────────────┐
│  t_dorm_frame    │                    │  t_dorm_event        │
│  (Stream Gateway │                    │  (Face Recognition   │
│   模拟)          │                    │   模拟)              │
└──────────────────┘                    └──────────────────────┘
         │                                           │
         ▼                                           ▼
    Face Recognition                          Dormitory Service
    (消费帧 → 识别)                           (消费事件 → 查宿)
```

---

## 2. 文件结构

```
test-env/
├── start.sh                    # 一键启动脚本
├── requirements.txt            # Python 依赖
└── server/
    ├── main.py                 # FastAPI 应用 (312行)
    └── static/
        └── index.html          # Web 仪表盘前端
```

---

## 3. 核心功能

### 3.1 4 路摄像头模拟

| Camera ID | Building | 颜色 |
|-----------|----------|------|
| `cam-a`   | A        | `#2980b9` 蓝色 |
| `cam-b`   | B        | `#27ae60` 绿色 |
| `cam-c`   | C        | `#8e44ad` 紫色 |
| `cam-d`   | D        | `#e67e22` 橙色 |

每路摄像头支持三种动作：
- **entry** — 绿色人形从左向右移动（进入宿舍）
- **exit** — 红色人形从右向左移动（离开宿舍）
- **idle** — 静止画面（无人）

### 3.2 测试画面生成

**主方案 (Pillow)**：`PIL.ImageDraw` 生成 640×360 的 JPEG 帧，包含：
- 时间戳叠加
- 摄像头标签
- 模拟门框矩形
- 人形指示（椭圆 + 矩形组合）
- 动作文字标注

**降级方案 (SVG)**：当 Pillow 不可用时，使用内联 SVG 作为 JPEG 兜底。

**测试人员列表**：

```python
TEST_PEOPLE = [
    "张三 (2024001)", "李四 (2024002)", "王五 (2024003)",
    "赵六 (2024004)", "孙七 (2024005)", "周八 (2024006)",
]
```

### 3.3 Kafka 消息注入

调用 `POST /api/cameras/{id}/simulate` 时：

1. 生成模拟 JPEG 帧
2. 构建 `t_dorm_frame` 消息（Base64 JPEG + 元数据）
3. 推送到 Kafka `t_dorm_frame` topic
4. 如为 `entry`/`exit` 动作，同时推 `t_dorm_event`（供下游测试）

---

## 4. API 端点

| 端点 | 方法 | 用途 |
|------|------|------|
| `/api/health` | GET | 服务健康检查，返回 Kafka/Pillow 状态 |
| `/api/cameras/{id}/simulate` | POST | 模拟指定摄像头的进出事件 |
| `/api/cameras/{id}/frame.jpg` | GET | 获取摄像头最新帧（JPEG） |
| `/api/cameras/{id}/status` | GET | 获取摄像头状态 |
| `/api/events` | GET | 查看事件日志（最近 300 条） |
| `/api/scenarios/random` | GET | 生成随机进出场景（批量测试） |
| `/` | GET | Web 仪表盘（静态文件） |

### `/api/cameras/{id}/simulate` 请求体

```json
{
  "action": "entry",        // "entry" | "exit" | "idle"
  "person": "张三 (2024001)"  // 可选，默认按时间选择
}
```

### `/api/cameras/{id}/simulate` 响应

```json
{
  "success": true,
  "camera_id": "cam-a",
  "action": "entry",
  "kafka": true,
  "frame_bytes": 28472
}
```

---

## 5. 启动方式

```bash
# 一键启动（自动检测/启动 infra）
bash test-env/start.sh

# 或手动启动
cd test-env
pip install -r requirements.txt
uvicorn server.main:app --host 0.0.0.0 --port 8082 --reload
```

启动后访问：
- **Web Dashboard**: http://localhost:8082/
- **API 基础路径**: http://localhost:8082/api/
- **事件流日志**: http://localhost:8082/api/events

---

## 6. 使用场景

### 场景 1：手动测试感知层
```
访问 Web Dashboard → 点击"模拟进入"
  → 服务器生成 JPEG 帧 → Kafka t_dorm_frame
  → Face Recognition 消费 → 检测 → 匹配 → t_dorm_event
```

### 场景 2：批量压力测试
```bash
curl "http://localhost:8082/api/scenarios/random?count=10"
# 生成 10 个随机进出事件
```

### 场景 3：配合 Dormitory Service 联调
```
test-env 推送 t_dorm_event → Dormitory Service 消费
  → 查宿统计 → 查询 API 验证
```

---

## 7. 事件日志

服务器维护一个 300 条内存循环队列 `event_log`，通过 `/api/events` 查看：

```json
[
  {
    "time": "14:23:05",
    "camera_id": "cam-b",
    "building": "B",
    "event_type": "entry",
    "detail": "B栋入口 entry [李四 (2024002)]"
  }
]
```

---

## 8. 依赖

| 包 | 用途 | 可选 |
|----|------|------|
| `fastapi` | Web 框架 | 必选 |
| `uvicorn` | ASGI 服务器 | 必选 |
| `kafka-python` | Kafka 生产者 | 必选（无 Kafka 时降级） |
| `Pillow` | 测试帧图像生成 | 可选（降级为 SVG） |
| `httpx` | HTTP 客户端（预留） | 可选 |

---

## 9. 限制

- 生成的帧是模拟画面，非真实摄像头画面
- 学管 API (`/sims/face/match`) 未在 test-env 中实现，需使用 `face-recognition` 的 Redis 缓存降级
- 事件日志存储在内存中，重启丢失
- 测试人员名单固定，如需扩展请修改 `TEST_PEOPLE`
