# 校园宿舍管理 AI 子系统 — Face Recognition 产品需求文档

> **文档版本:** v1.0  
> **所属项目:** 校园宿舍管理 AI 子系统（Student Management System — Dormitory AI）  
> **模块名称:** Face Recognition（人脸识别模块）  
> **更新日期:** 2026-05-15  
> **状态:** 初稿  

---

## 目录

1. [模块定位](#1-模块定位)
2. [技术架构](#2-技术架构)
3. [数据流设计](#3-数据流设计)
4. [功能清单](#4-功能清单)
   - 4.1 [人脸检测](#41-人脸检测)
   - 4.2 [特征提取](#42-特征提取)
   - 4.3 [学管 API 对接](#43-学管-api-对接)
   - 4.4 [进出方向判断](#44-进出方向判断)
   - 4.5 [事件推送](#45-事件推送)
   - 4.6 [陌生人检测](#46-陌生人检测)
   - 4.7 [夜间模式](#47-夜间模式)
5. [接口设计](#5-接口设计)
6. [配置清单](#6-配置清单)
7. [错误处理与降级策略](#7-错误处理与降级策略)
8. [性能指标](#8-性能指标)
9. [附录：Kafka 消息协议](#9-附录kafka-消息协议)

---

## 1. 模块定位

### 1.1 系统背景

本模块是**学生管理系统（Student Management System, SMS）**的宿舍管理子服务，专注宿舍楼出入口的人员识别与进出事件采集。与通用的安防 AI 平台不同，本系统面向**极小规模部署**（仅 4 路摄像头，每栋宿舍楼入口 1 个），核心产出是**每晚宿舍就寝统计**，即传统"查宿"流程的自动化替代方案。

系统整体拓扑：

```
                        ┌──────────────────┐
                        │  学管主系统        │
                        │  SpringBoot + Vue │
                        │  (学生数据/业务)   │
                        └────────┬─────────┘
                                 │ HTTP REST (身份匹配 API)
                    ┌────────────┴────────────┐
                    │  Face Recognition 服务   │
                    │  Python · FastAPI        │
                    │  人脸检测 → 特征提取     │
                    │  → 身份匹配 → 进出判定    │
                    └────────────┬────────────┘
                                 │ Kafka
              ┌──────────────────┼──────────────────┐
              │                  │                  │
     ┌────────▼──────┐  ┌───────▼───────┐  ┌───────▼───────┐
     │  摄像头 1     │  │  摄像头 2     │  │  摄像头 3/4   │
     │  宿舍A 入口    │  │  宿舍B 入口    │  │  宿舍C/D 入口  │
     └───────────────┘  └───────────────┘  └───────────────┘
```

### 1.2 模块职责

| 职责 | 说明 |
|------|------|
| **帧消费** | 从 Kafka `t_dorm_frame` Topic 消费 JPEG 帧数据 |
| **人脸检测** | 在帧中检测人脸区域，支持多人同时检测 |
| **特征提取** | 将检测到的人脸对齐后提取 512 维特征向量 |
| **身份匹配** | 调用学管系统 API 匹配学生身份，本地缓存做降级 |
| **进出判断** | 基于人脸检测框轨迹判断进入/离开方向 |
| **事件推送** | 将进出事件推送到 Kafka `t_dorm_event` |
| **陌生人标记** | 学管 API 匹配失败或置信度不足时标记陌生人 |

### 1.3 非职责（本模块不包含）

- ❌ 宿舍就寝统计、查宿报表等业务逻辑（由学管主系统处理）
- ❌ Web 前端页面展示
- ❌ 学生人脸数据的持久化存储与库管理（由学管系统管理）
- ❌ 视频流拉流与解码（由 Stream Gateway / 摄像头端处理）
- ❌ 人员跟踪（ByteTrack / ReID）—— 宿舍场景仅 4 路入口，无跨摄像头追踪需求
- ❌ 智能规则引擎（区域入侵、徘徊等安防规则）

### 1.4 边界与上下游

| 交互对象 | 通信方式 | 数据内容 | 说明 |
|----------|---------|---------|------|
| **Kafka (t_dorm_frame)** | Consumer | JPEG 帧 + 元数据 | 消费帧数据，来源为 Stream Gateway 或摄像头直接推送 |
| **学管系统 API** | HTTP REST | 特征向量 ↔ 学生身份 | 按需调用身份匹配，接口可自定义 |
| **Kafka (t_dorm_event)** | Producer | 进出事件 JSON | 推送判断结果，由学管系统消费 |
| **学管系统** | 定时批量拉取或学管推送 | 学生人脸特征库 | 本地缓存预加载，减少实时调用延迟 |

---

## 2. 技术架构

### 2.1 技术选型

| 层级 | 技术方案 | 说明 |
|------|---------|------|
| 编程语言 | **Python 3.11+** | AI 生态成熟，推理框架绑定友好 |
| Web 框架 | **FastAPI** | 轻量异步 HTTP 服务，用于健康检查和运维接口 |
| 人脸检测 | **RetinaFace** / MTCNN | RetinaFace 精度高；MTCNN 轻量级备选 |
| 特征提取 | **ArcFace (InsightFace)** | 输出 512 维 float32 特征向量 |
| 推理框架 | **ONNX Runtime** / TensorRT | ONNX Runtime 通用性佳；TensorRT GPU 加速 |
| 消息队列 | **Kafka** (confluent-kafka-python) | 帧消费与事件产出 |
| HTTP 客户端 | **httpx / aiohttp** | 异步调用学管 API |
| 数值计算 | **NumPy + OpenCV** | 图像预处理、特征归一化 |
| GPU 加速 | **CUDA 12.x** | 推理加速（可选，CPU 也可运行） |
| 进程管理 | **Supervisor / systemd** | 守护进程管理 |

### 2.2 为什么选择 Python + 独立部署

| 决策 | 理由 |
|------|------|
| **Python 做人脸识别** | AI 推理生态最成熟（InsightFace、ONNX Runtime、PyTorch 原生支持），训练与推理链统一 |
| **学管主系统 Java SpringBoot** | 主系统负责业务编排，人脸识别独立部署，语言分离降低耦合 |
| **Kafka 异步通信** | 帧消费与事件产出完全异步，Face Recognition 服务故障不影响主系统 |
| **FastAPI 轻量 HTTP** | 仅用于健康检查和运维指令，不做业务路由 |

### 2.3 模块架构

```
┌────────────────────────────────────────────────────────────┐
│                 Face Recognition Service                     │
│                                                              │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │  Frame        │    │  Face         │    │  Direction    │  │
│  │  Consumer     │───▶│  Pipeline     │───▶│  Judger      │  │
│  │  (Kafka)      │    │  Detect+Embed │    │  进出判断     │  │
│  └──────────────┘    └──────┬───────┘    └──────┬───────┘  │
│                              │                    │          │
│                     ┌────────▼───────┐    ┌──────▼───────┐  │
│                     │  Student API   │    │  Event        │  │
│                     │  Client        │    │  Producer     │  │
│                     │  学管身份匹配    │    │  (Kafka)      │  │
│                     └──────┬────────┘    └──────────────┘  │
│                            │                                 │
│                     ┌──────▼────────┐                        │
│                     │  Face Cache   │                        │
│                     │  本地特征缓存   │                        │
│                     └───────────────┘                        │
│                                                              │
│  ┌──────────────┐    ┌──────────────┐                        │
│  │  Health      │    │  Config      │                        │
│  │  Check       │    │  Manager     │                        │
│  │  GET /health │    │  配置热加载   │                        │
│  └──────────────┘    └──────────────┘                        │
└────────────────────────────────────────────────────────────┘
```

### 2.4 推理 Pipeline 详细流程

```
Kafka Frame ──→ 解码 JPEG → 图像预处理 ←┬─ 白天模式参数
                                         │  或
                                         ├─ 夜间模式参数（低光照增强）
                                         │
                    ┌─────────────────────┘
                    ▼
          ┌──────────────────┐
          │ ROI 区域过滤      │ 只处理配置中的检测区域
          └────────┬─────────┘
                   ▼
          ┌──────────────────┐
          │ 人脸检测          │ RetinaFace / MTCNN
          │ (多人同时检测)     │
          └────────┬─────────┘
                   ▼
          ┌──────────────────┐
          │ 人脸质量过滤      │ 模糊度、角度、大小阈值
          └────────┬─────────┘
                   ▼
          ┌──────────────────┐
          │ 人脸对齐          │ Landmark → 仿射变换
          └────────┬─────────┘
                   ▼
          ┌──────────────────┐
          │ 特征提取          │ ArcFace → 512-dim
          │ L2 归一化         │
          └────────┬─────────┘
                   ▼
          ┌──────────────────┐
          │ 身份匹配          │ ① 本地缓存查询（快速）
          │                   │ ② 学管 API 调用（兜底）
          └────────┬─────────┘
                   ▼
          ┌──────────────────┐
          │ 进出方向判断      │ 基于轨迹/ROI穿越
          └────────┬─────────┘
                   ▼
          ┌──────────────────┐
          │ 去重判断          │ 防重复推送窗口
          └────────┬─────────┘
                   ▼
          Kafka t_dorm_event
```

---

## 3. 数据流设计

### 3.1 端到端数据流

```
                                            ┌──────────────┐
                               ┌───────────▶│  学管主系统    │
                               │            │  消费事件      │
                               │            │  生成查宿统计  │
                               │            └──────────────┘
                               │
┌──────────┐   Kafka       ┌──┴─────────────┐   Kafka
│ 摄像头     │  t_dorm_frame │  Face           │  t_dorm_event
│ 或         │─────────────▶│  Recognition    │──────────────▶ 学管业务
│ 推流服务   │  JPEG 帧      │  Service        │  进出事件      后端 →
└──────────┘               │                 │               查宿统计
                            └────────┬────────┘
                                     │ HTTP POST
                                     │ /api/face/match
                                     ▼
                            ┌──────────────────┐
                            │  学管系统 API     │
                            │  身份匹配服务      │
                            └──────────────────┘
```

### 3.2 Kafka Topic 契约

| Topic | 角色 | 消息格式 | 说明 |
|-------|------|---------|------|
| `t_dorm_frame` | Consumer | JSON (帧消息) | 摄像头推送的 JPEG 帧数据 |
| `t_dorm_event` | Producer | JSON (事件消息) | 进出事件、陌生人事件 |

### 3.3 数据依赖

| 数据 | 来源 | 用途 | 存储 |
|------|------|------|------|
| 视频帧 (JPEG) | Kafka `t_dorm_frame` | 推理输入 | 内存（处理后丢弃） |
| 人脸特征向量 | 本地提取 | 身份匹配 | 内存（临时） |
| 学生人脸特征库 | 学管 API 预加载 | 本地缓存匹配 | 内存缓存（TTL 可配） |
| 匹配结果 | 学管 API | 学生身份确认 | 随事件推送 |
| 进出事件 | 本模块产出 | 学管系统消费 | Kafka `t_dorm_event` |

### 3.4 Frame 消息格式 (t_dorm_frame)

```json
{
  "frame_id": "uuid-xxxx",
  "camera_id": "dorm_a_entrance",
  "building": "宿舍A栋",
  "timestamp_unix_ms": 1747305000000,
  "image_data": "<base64_jpeg>",
  "image_width": 1920,
  "image_height": 1080,
  "sequence_num": 1523
}
```

---

## 4. 功能清单

---

### 4.1 人脸检测

#### 4.1.1 功能描述

从 Kafka 消费的视频帧中检测人脸区域，为后续特征提取提供输入。支持多人同时检测（宿舍入口可能同时有多人进出）。

#### 4.1.2 基本信息

| 项目 | 内容 |
|------|------|
| **功能ID** | `F-FD-001` |
| **优先级** | P0 |
| **算法** | RetinaFace（主选）/ MTCNN（轻量备选） |
| **推理框架** | ONNX Runtime / TensorRT |
| **输入** | RGB/BGR 图像 (H×W×3) |
| **输出** | 人脸检测框 `[x1, y1, x2, y2, confidence]` + 5 个关键点 `[left_eye, right_eye, nose, left_mouth, right_mouth]` |

#### 4.1.3 多人检测支持

- 单帧最大检测人数：可配置（默认 10 人）
- 宿舍场景下入口并发量通常 ≤ 5 人
- NMS 后处理去重，保留置信度最高的检测框

#### 4.1.4 人脸质量过滤

检测出的人脸框经过三重质量过滤，任一条件不满足则丢弃：

| 过滤项 | 说明 | 默认阈值 |
|--------|------|---------|
| **模糊度** | 基于 Laplacian 方差判断 | < 100 视为模糊 |
| **角度** | 人脸偏转角度（yaw/pitch/roll） | |yaw| > 45° 丢弃 |
| **大小** | 人脸区域像素尺寸 | 最小 80×80 px |

#### 4.1.5 检测区域配置 (ROI)

支持按摄像头配置只检测画面中的特定区域，非入口区域（如画面边缘、窗外区域）忽略。

```yaml
# 配置示例：只检测画面下半部分（入口区域）
camera_roi:
  dorm_a_entrance: [0, 0.4, 1.0, 1.0]    # [x1_norm, y1_norm, x2_norm, y2_norm]
  dorm_b_entrance: [0, 0.3, 1.0, 1.0]
```

- 坐标使用归一化值 (0~1)，相对图像宽高
- ROI 外的人脸直接丢弃，不计入后续处理

#### 4.1.6 性能目标

| 指标 | 目标值 |
|------|--------|
| 单帧检测延迟 (RetinaFace, GPU) | ≤ 30ms |
| 单帧检测延迟 (RetinaFace, CPU) | ≤ 150ms |
| 单帧检测延迟 (MTCNN, CPU) | ≤ 80ms |
| 多人检测能力（单帧） | ≥ 10 人 |
| 检测精度 (WIDER Face) | ≥ 90% Easy Set |

---

### 4.2 特征提取

#### 4.2.1 功能描述

将检测到的人脸区域对齐后提取特征向量，用于后续身份匹配。特征向量为 512 维 float32，经 L2 归一化后输出。

#### 4.2.2 基本信息

| 项目 | 内容 |
|------|------|
| **功能ID** | `F-FE-001` |
| **优先级** | P0 |
| **算法** | ArcFace (InsightFace) |
| **Backbone** | ResNet50 / ResNet100 |
| **特征维度** | 512 维 float32 |
| **推理框架** | ONNX Runtime / TensorRT |
| **对齐方式** | Landmark-based 仿射变换 (112×112) |

#### 4.2.3 Pipeline 流程

```
人脸裁切区域
    │
    ▼
┌──────────────────┐
│ 人脸关键点检测     │ RetinaFace 输出的 5 landmarks
│ (5 points)        │
└────────┬─────────┘
         ▼
┌──────────────────┐
│ 仿射变换 (Align)  │ 基于 landmarks → 112×112 标准脸
└────────┬─────────┘
         ▼
┌──────────────────┐
│ 归一化            │ 像素值归一化至 [-1, 1]
└────────┬─────────┘
         ▼
┌──────────────────┐
│ ArcFace 推理      │ ONNX Runtime / TensorRT
└────────┬─────────┘
         ▼
┌──────────────────┐
│ L2 归一化         │ 使特征向量模长为 1
│ Output: 512-dim  │ → 用于余弦相似度计算
└──────────────────┘
```

#### 4.2.4 GPU 加速

| 加速方式 | 说明 | 加速比 (vs CPU) |
|----------|------|----------------|
| ONNX Runtime + CUDA | 使用 CUDAExecutionProvider | ~5x |
| TensorRT FP16 | TensorRT 优化 + FP16 推理 | ~8x |
| TensorRT INT8 | TensorRT 量化推理 | ~10x |

#### 4.2.5 性能目标

| 指标 | 目标值 |
|------|--------|
| 单张人脸特征提取 (GPU, TensorRT FP16) | ≤ 15ms |
| 单张人脸特征提取 (GPU, ONNX CUDA) | ≤ 25ms |
| 单张人脸特征提取 (CPU) | ≤ 80ms |
| 批量提取（4 张, GPU） | ≤ 30ms |
| 特征向量维度 | 512 |
| 归一化方式 | L2 Norm |

---

### 4.3 学管 API 对接

> ⚠️ **重要前提**：以下接口在学管系统当前 OpenAPI 中**不存在**，需在学管同步开发中新增。当前学管无任何人脸相关的 API 或数据存储。本节定义的 API 契约是"建议方案"，具体需要与学管开发团队协商确定。

#### 4.3.1 功能描述

封装学管系统的身份匹配 API 调用，将提取的 512 维特征向量发送至学管系统，获取匹配的学生身份信息。

**由于学管目前不存储人脸数据，两种实现路径：**

| 路径 | 方案 | 推荐度 |
|------|------|--------|
| **A — 学管新增 face/match 接口** | 学管存储人脸特征向量，本服务通过 REST API 调用匹配 | ⭐ 推荐（数据统一） |
| **B — 本服务本地管理人脸特征** | 学管提供学生照片下载 API，本服务本地提取特征并存储 | 备选（学管改造成本低） |

以下按路径 A 设计，若选路径 B 则需在本服务中增加特征存储（本地向量索引）。

#### 4.3.2 设计原则

- 学管 API **按需调用**：检测到人脸 + 提取特征后才调用
- 接口格式**可自定义**：学管系统为 SpringBoot + Vue，可根据双方约定调整
- **本地缓存优先**：减少对学管系统的实时调用压力
- **失败降级**：学管不可用时使用本地缓存做兜底匹配

#### 4.3.3 API 协议建议

##### 身份匹配接口（待学管新增）

> **实际状态**: 学管 OpenAPI 中 **不存在此接口**  
> **学管路径风格**: 使用 `/sims/` 前缀（如 `/sims/student/get-list`）  
> **建议路径**: `POST /sims/face/match`  
> **接口格式可自定义** — 以下为建议方案，以双方协商为准

```http
POST /sims/face/match
Content-Type: application/json
Authorization: Bearer <token>

Request:
{
  "feature_vector": [0.123, -0.456, ..., 0.789],    // 512 维 L2 归一化向量
  "confidence_threshold": 0.65,                       // 置信度阈值
  "camera_id": "dorm_a_entrance",                     // 来源摄像头
  "timestamp": 1747305000000                          // 检测时间戳
}

Response (200 OK):
{
  "code": 200,
  "text": "success",
  "data": {
    "matched": true,
    "student": {
      "studentNumber": "2024001",
      "name": "张三",
      "myClass": "计算机2101班",
      "gender": "男",
      "dormitory": 301
    },
    "confidence": 0.89
  }
}
```

> **响应格式对齐**: 学管统一使用 `{ code, text, data }` 格式，而非 `{ success, matched }`。本服务调用时需适配解析。

#### 4.3.4 调用参数

| 参数 | 类型 | 说明 |
|------|------|------|
| `feature_vector` | float[] | 512 维 L2 归一化特征向量 |
| `confidence_threshold` | float | 置信度阈值，低于此值视为未匹配 |
| `camera_id` | string | 来源摄像头 ID，用于学管端日志 |
| `timestamp` | int64 | 检测时间戳 (Unix ms) |

#### 4.3.5 重试与超时策略

| 参数 | 默认值 | 说明 |
|------|--------|------|
| 超时时间 | 3000 ms | 单个 API 请求超时 |
| 最大重试次数 | 2 次 | 失败后重试，重试间隔 500ms |
| 重试条件 | 网络超时 / 5xx 错误 | 4xx 错误不重试 |
| 熔断阈值 | 连续 5 次失败 | 触发熔断，直接使用本地缓存 |
| 熔断恢复 | 30s 后半开尝试 | 成功后关闭熔断 |

#### 4.3.6 本地人脸特征缓存

由于仅 4 个摄像头，场景固定，本地缓存是减少延迟的关键手段。

| 缓存策略 | 说明 |
|----------|------|
| **预加载** | 服务启动时从学管系统批量拉取全体住宿学生的人脸特征 |
| **按需缓存** | 每次 API 匹配成功的特征向量缓存到本地（key: student_id, value: feature_vector） |
| **缓存淘汰** | TTL 过期淘汰，默认 60 分钟 |
| **缓存容量** | 5000 人（宿舍规模上限） |
| **存储方式** | 内存字典 + NumPy 数组（用于快速余弦相似度计算） |

**本地匹配流程：**

```
收到特征向量 v
   │
   ├──▶ 本地缓存中查找：计算 v 与所有缓存向量的余弦相似度
   │     └── 最高相似度 > 阈值 (0.65) → 直接返回学生身份 (O(1) ~ O(n))
   │
   └──▶ 本地未命中或置信度不足 → 调用学管 API
         ├── 成功 → 更新本地缓存 → 返回结果
         └── 失败/超时 → 使用本地缓存中置信度最高的结果（降级）
                            └── 仍无结果 → 标记为陌生人
```

---

### 4.4 进出方向判断

#### 4.4.1 功能描述

基于人脸检测框在画面中的位置变化趋势或 ROI 穿越方向，判断人员是进入宿舍楼还是离开宿舍楼。

#### 4.4.2 基本信息

| 项目 | 内容 |
|------|------|
| **功能ID** | `F-DJ-001` |
| **优先级** | P0 |
| **判断方法** | ROI 线穿越（默认）/ 轨迹方向 |
| **输入** | 同一人脸的连续多帧检测框 + 时间戳 |
| **输出** | `entry` / `exit` / `unknown` |

#### 4.4.3 判断方法一：ROI 线穿越（推荐）

在画面中配置一条虚拟分割线（ROI 线），通过检测框中心点穿越方向判断进出。

```
画面示意图（宿舍入口）：

     ┌──────────────────────────────────┐
     │         宿舍 外 部                │
     │                                  │
     │   ○ ← 检测框轨迹 (进入: 上→下)   │
     │                                  │
     ├────────── ROI 分割线 ────────────┤  ← y = 0.6 (归一化坐标)
     │                                  │
     │   ○ ← 检测框轨迹 (离开: 下→上)   │
     │         宿舍 内 部                │
     └──────────────────────────────────┘
```

| 穿越方向 | 行为 | 判定 |
|----------|------|------|
| 上 → 下（外部 → 内部） | 检测框从 ROI 线上方穿越到下方 | **进入 (entry)** |
| 下 → 上（内部 → 外部） | 检测框从 ROI 线下方穿越到上方 | **离开 (exit)** |

#### 4.4.4 判断方法二：轨迹方向

不依赖 ROI 线，通过检测框中心点的连续帧位移趋势判断。

| 趋势 | 判定 |
|------|------|
| Y 坐标从大 → 小（框向上移动） | 离开（走向室外） |
| Y 坐标从小 → 大（框向下移动） | 进入（走向室内） |

#### 4.4.5 判断参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `direction.method` | `roi_cross` | 判断方式：`roi_cross` / `trajectory` |
| `direction.roi_line_y` | `0.6` | ROI 分割线 Y 坐标 (归一化 0~1) |
| `direction.judge_frames` | `3` | 判定所需连续帧数（防止抖动误判） |
| `direction.trajectory_threshold` | `20` | 轨迹方向判断的最小位移 (像素) |

#### 4.4.6 防抖逻辑

- 单人需要连续 `judge_frames` 帧（默认 3 帧）检测框方向一致，才触发进出事件
- 中间出现方向不一致 → 计数器重置
- 人员在 ROI 线附近来回移动 → 不触发事件，直到方向稳定

---

### 4.5 事件推送

#### 4.5.1 功能描述

将进出判断结果以标准化事件格式推送到 Kafka `t_dorm_event` Topic，供学管系统消费。

#### 4.5.2 基本信息

| 项目 | 内容 |
|------|------|
| **功能ID** | `F-EP-001` |
| **优先级** | P0 |
| **目标 Topic** | `t_dorm_event` |
| **序列化** | JSON |
| **推送时机** | 进出方向判定完成 + 通过去重检查 |

#### 4.5.3 事件格式

```json
{
  "event_id": "evt_20260515_a1b2c3d4",
  "camera_id": "dorm_a_entrance",
  "building": "宿舍A栋",
  "student_id": "2024001",
  "student_name": "张三",
  "event_type": "entry",
  "confidence": 0.89,
  "face_snapshot": "/9j/4AAQ...base64_jpeg_data...",
  "timestamp_unix_ms": 1747305000000,
  "is_stranger": false,
  "extra": {
    "class": "计算机科学 2024-1 班",
    "dorm_room": "A-301"
  }
}
```

**字段说明：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `event_id` | string | ✅ | 全局唯一事件 ID (UUID / 时间戳+随机) |
| `camera_id` | string | ✅ | 摄像头标识 |
| `building` | string | ✅ | 宿舍楼名称 |
| `student_id` | string | ✅ | 学生学号（陌生人时为 `"stranger"`） |
| `student_name` | string | ✅ | 学生姓名（陌生人时为 `"未知"`） |
| `event_type` | string | ✅ | `"entry"` / `"exit"` |
| `confidence` | float | ✅ | 身份匹配置信度 |
| `face_snapshot` | string | ✅ | 人脸裁切区域 JPEG 的 Base64 编码（小图，~5-15KB） |
| `timestamp_unix_ms` | int64 | ✅ | 事件产生时间 (Unix 毫秒) |
| `is_stranger` | bool | ✅ | 是否为陌生人标记 |
| `extra` | object | ❌ | 额外信息（学号、班级等，从学管 API 透传） |

#### 4.5.4 防重复推送（去重）

同一人在短时间内反复进出或检测到 → 合并为一条事件。

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `dedup.window_sec` | 10s | 同一人防重复时间窗口 |
| `dedup.key` | `camera_id + student_id + event_type` | 去重 Key |
| 窗口期内重复事件 | 丢弃 | 只推送窗口期内的第一条 |

**去重示例：**

```
t=0s   张三 进入宿舍A栋    → 推送 entry 事件
t=3s   张三 再次检测到进入  → 去重窗口内，丢弃
t=12s  张三 离开宿舍A栋     → 推送 exit 事件（不同 event_type，新窗口）
t=15s  张三 再次检测到离开  → 去重窗口内，丢弃
```

#### 4.5.5 陌生人事件

陌生人事件与正常事件共享同一 Topic，通过 `is_stranger: true` 和 `student_id: "stranger"` 区分。

```json
{
  "event_id": "evt_20260515_stranger_xxxx",
  "camera_id": "dorm_b_entrance",
  "building": "宿舍B栋",
  "student_id": "stranger",
  "student_name": "未知",
  "event_type": "entry",
  "confidence": 0.32,
  "face_snapshot": "/9j/4AAQ...",
  "timestamp_unix_ms": 1747305000000,
  "is_stranger": true,
  "extra": {
    "reason": "学管API匹配失败",
    "top_match_confidence": 0.32
  }
}
```

---

### 4.6 陌生人检测

#### 4.6.1 功能描述

当学管 API 匹配失败或匹配置信度低于阈值时，将人员标记为陌生人。陌生人事件单独标记，供学管系统后续处理。

#### 4.6.2 陌生人判定条件

满足以下任一条件即判定为陌生人：

| 条件 | 说明 |
|------|------|
| **API 无匹配** | 学管 API 返回 `matched: false` |
| **置信度不足** | API 返回的最高置信度 < `recognition.confidence_threshold` |
| **API 不可用** | 学管 API 超时/熔断，且本地缓存也无匹配 |
| **本地匹配不足** | 本地缓存中最高相似度 < `recognition.confidence_threshold` |

#### 4.6.3 陌生人事件特别处理

| 处理项 | 说明 |
|--------|------|
| `is_stranger` | 标记为 `true` |
| `student_id` | 固定为 `"stranger"` |
| `student_name` | 固定为 `"未知"` |
| `confidence` | 保留实际相似度值（用于学管端参考） |
| `extra.reason` | 记录陌生人原因：`"学管API匹配失败"` / `"置信度不足(0.32)"` / `"学管API不可用"` |
| `extra.top_match_confidence` | 最高匹配度（即使低于阈值） |
| 人脸快照 | 始终保留，方便人工核验 |

#### 4.6.4 陌生人阈值配置

| 配置项 | 说明 |
|--------|------|
| `recognition.confidence_threshold` | 身份认定置信度（高于此值认定为具体学生） |
| `recognition.stranger_threshold` | 陌生人判定下限（低于此值视为陌生人） |

示例值：`confidence_threshold=0.65`, `stranger_threshold=0.0`。即相似度 > 0.65 认定为对应学生，≤ 0.65 标记为陌生人。

---

### 4.7 夜间模式

#### 4.7.1 功能描述

宿舍夜间（22:00 ~ 06:00）是查宿关键时段，针对低光照、红外摄像头场景进行检测参数优化。

#### 4.7.2 基本信息

| 项目 | 内容 |
|------|------|
| **功能ID** | `F-NM-001` |
| **优先级** | P1 |
| **触发方式** | 基于系统时钟自动切换（可配置时间段） |

#### 4.7.3 夜间模式参数自动切换

| 参数 | 白天 | 夜间 | 说明 |
|------|------|------|------|
| `detection.confidence` | 0.7 | 0.5 | 夜间降低检测阈值（红外图像质量较低） |
| `detection.min_face_size` | 80 | 60 | 夜间允许检测更小的人脸（红外细节少） |
| `recognition.confidence_threshold` | 0.65 | 0.55 | 夜间降低身份匹配阈值 |
| 图像预处理 | 标准 | CLAHE 增强 | 自适应直方图均衡化提升红外图像对比度 |

#### 4.7.4 配置示例

```yaml
night_mode:
  enabled: true
  start_hour: 22          # 22:00 进入夜间模式
  end_hour: 6             # 06:00 退出夜间模式
  detection:
    confidence: 0.5
    min_face_size: 60
  recognition:
    confidence_threshold: 0.55
```

---

## 5. 接口设计

### 5.1 与学管系统的 API 契约

#### 5.1.1 身份匹配接口

```http
POST /api/face/match
```

**Request:**
```json
{
  "feature_vector": [0.123, -0.456, ..., 0.789],
  "confidence_threshold": 0.65,
  "camera_id": "dorm_a_entrance",
  "timestamp": 1747305000000
}
```

**Response (200):**
```json
{
  "success": true,
  "matched": true,
  "student": {
    "student_id": "2024001",
    "student_name": "张三",
    "class": "计算机科学 2024-1 班",
    "dorm_building": "宿舍A栋",
    "dorm_room": "A-301",
    "gender": "male",
    "confidence": 0.89
  },
  "match_cost_ms": 15.3
}
```

**Response (未匹配):**
```json
{
  "success": true,
  "matched": false,
  "student": null,
  "confidence": 0.0,
  "match_cost_ms": 5.2
}
```

#### 5.1.2 批量特征预加载接口（可选）

用于服务启动时的本地缓存预热，建议学管系统提供此接口以减少冷启动延迟。

```http
GET /api/face/features?building=宿舍A栋&page=1&page_size=500
```

**Response:**
```json
{
  "success": true,
  "total": 1200,
  "page": 1,
  "students": [
    {
      "student_id": "2024001",
      "student_name": "张三",
      "feature_vector": [0.123, -0.456, ...],
      "dorm_room": "A-301"
    }
  ]
}
```

### 5.2 本地健康检查

```http
GET /health
```

**Response (正常):**
```json
{
  "status": "ok",
  "service": "face-recognition",
  "version": "1.0.0",
  "uptime_seconds": 86400,
  "timestamp": "2026-05-15T10:30:00Z",
  "checks": {
    "kafka_consumer": { "status": "ok", "lag": 0 },
    "kafka_producer": { "status": "ok" },
    "student_api": { "status": "ok", "latency_ms": 5 },
    "model_loaded": { "status": "ok", "models": ["retinaface", "arcface"] },
    "cache_status": { "total_cached": 3500, "hit_rate": 0.87 }
  }
}
```

**Response (降级):**
```json
{
  "status": "degraded",
  "service": "face-recognition",
  "checks": {
    "student_api": { "status": "error", "message": "连续超时，已切换本地缓存降级" },
    "kafka_producer": { "status": "ok" },
    "model_loaded": { "status": "ok" }
  }
}
```

### 5.3 配置热加载接口（运维用）

```http
POST /api/v1/config/reload
```

重新加载配置文件，无需重启服务。

**Response:**
```json
{
  "success": true,
  "message": "配置已重新加载",
  "changed_keys": ["detection.confidence", "night_mode.enabled"]
}
```

### 5.4 Prometheus 指标接口

```http
GET /metrics
```

暴露以下 Prometheus 指标：

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `face_detection_total` | Counter | `camera_id`, `status` | 人脸检测总数 |
| `face_detection_latency_ms` | Histogram | `model` | 检测延迟分布 |
| `face_recognition_total` | Counter | `camera_id`, `matched` | 识别尝试总数 |
| `face_recognition_latency_ms` | Histogram | `source` | 识别延迟分布 (api/cache) |
| `face_cache_hit_total` | Counter | - | 本地缓存命中数 |
| `face_cache_miss_total` | Counter | - | 本地缓存未命中数 |
| `student_api_call_total` | Counter | `status` | 学管 API 调用计数 |
| `student_api_latency_ms` | Histogram | - | 学管 API 延迟分布 |
| `event_produced_total` | Counter | `camera_id`, `event_type`, `is_stranger` | 事件产出计数 |
| `event_dedup_dropped_total` | Counter | - | 去重丢弃事件数 |
| `night_mode_active` | Gauge | - | 夜间模式是否激活 (0/1) |

---

## 6. 配置清单

### 6.1 完整配置项

所有配置项均通过 YAML 配置文件管理，支持环境变量覆盖（大写 + 下划线格式）。

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `service.name` | string | `"face-recognition"` | 服务名称 |
| `service.log_level` | string | `"info"` | 日志级别 (debug/info/warn/error) |
| `service.profile` | string | `"production"` | 运行环境 |
| | | | |
| **人脸检测** | | | |
| `detection.model` | string | `"retinaface"` | 检测模型 (retinaface/mtcnn) |
| `detection.model_path` | string | `"/models/retinaface.onnx"` | 检测模型文件路径 |
| `detection.confidence` | float | `0.7` | 检测置信度阈值 |
| `detection.nms_iou_threshold` | float | `0.45` | NMS IoU 阈值 |
| `detection.min_face_size` | int | `80` | 最小人脸像素 (px) |
| `detection.max_face_size` | int | `1024` | 最大人脸像素 (px) |
| `detection.max_faces_per_frame` | int | `10` | 单帧最大检测人数 |
| `detection.roi` | json | `null` | 检测区域 [x1,y1,x2,y2]，null=全图 |
| `detection.quality.blur_threshold` | float | `100.0` | 模糊度阈值 (Laplacian 方差) |
| `detection.quality.max_yaw` | float | `45.0` | 最大偏转角 (度) |
| | | | |
| **人脸识别** | | | |
| `recognition.model` | string | `"arcface"` | 识别模型 |
| `recognition.model_path` | string | `"/models/arcface.onnx"` | 识别模型文件路径 |
| `recognition.vector_dim` | int | `512` | 特征向量维度 |
| `recognition.input_size` | int | `112` | 识别输入尺寸 (112×112) |
| `recognition.gpu_device` | int | `0` | GPU 设备号 (-1=CPU) |
| `recognition.inference_backend` | string | `"onnxruntime"` | 推理后端 (onnxruntime/tensorrt) |
| `recognition.confidence_threshold` | float | `0.65` | 身份认定置信度阈值 |
| | | | |
| **学管 API** | | | |
| `student_api.url` | string | `"http://sms-platform:8080"` | 学管系统 API 基地址 |
| `student_api.match_path` | string | `"/sims/face/match"` | 身份匹配接口路径。**此接口学管当前不存在，待新增** |
| `student_api.timeout_ms` | int | `3000` | API 调用超时 (ms) |
| `student_api.retry_count` | int | `2` | 失败重试次数 |
| `student_api.retry_interval_ms` | int | `500` | 重试间隔 (ms) |
| `student_api.circuit_breaker_threshold` | int | `5` | 熔断阈值 (连续失败数) |
| `student_api.circuit_breaker_recovery_s` | int | `30` | 熔断恢复时间 (秒) |
| `student_api.cache_ttl_min` | int | `60` | 本地特征缓存时间 (分钟) |
| `student_api.cache_max_size` | int | `5000` | 本地缓存最大容量 (人数) |
| `student_api.batch_preload_enabled` | bool | `true` | 是否启用批量预加载 |
| `student_api.batch_preload_url` | string | `"/api/face/features"` | 预加载接口路径 |
| | | | |
| **进出方向** | | | |
| `direction.method` | string | `"roi_cross"` | 进出判断方式 (roi_cross/trajectory) |
| `direction.roi_line_y` | float | `0.6` | ROI 分割线 Y 坐标 (归一化) |
| `direction.judge_frames` | int | `3` | 判定所需连续帧数 |
| `direction.trajectory_threshold_px` | int | `20` | 轨迹方向判断位移阈值 (像素) |
| | | | |
| **防重复** | | | |
| `dedup.window_sec` | int | `10` | 同人防重复时间窗口 (秒) |
| `dedup.enabled` | bool | `true` | 去重开关 |
| | | | |
| **夜间模式** | | | |
| `night_mode.enabled` | bool | `true` | 夜间模式开关 |
| `night_mode.start_hour` | int | `22` | 夜间开始时间 (24h) |
| `night_mode.end_hour` | int | `6` | 夜间结束时间 (24h) |
| `night_mode.detection.confidence` | float | `0.5` | 夜间检测置信度阈值覆盖 |
| `night_mode.detection.min_face_size` | int | `60` | 夜间最小人脸覆盖 |
| `night_mode.recognition.threshold` | float | `0.55` | 夜间身份认定阈值覆盖 |
| | | | |
| **Kafka 消费者** | | | |
| `kafka.bootstrap_servers` | string[] | `["localhost:9092"]` | Kafka 集群地址 |
| `kafka.consumer.topic` | string | `"t_dorm_frame"` | 消费帧 Topic |
| `kafka.consumer.group_id` | string | `"face-recognition-group"` | Consumer Group ID |
| `kafka.consumer.auto_offset_reset` | string | `"latest"` | 偏移重置策略 |
| `kafka.consumer.enable_auto_commit` | bool | `false` | 自动提交 |
| `kafka.consumer.max_poll_records` | int | `100` | 单次最大拉取条数 |
| `kafka.consumer.session_timeout_ms` | int | `30000` | Session 超时 |
| | | | |
| **Kafka 生产者** | | | |
| `kafka.producer.topic` | string | `"t_dorm_event"` | 产出事件 Topic |
| `kafka.producer.compression` | string | `"snappy"` | 压缩算法 |
| `kafka.producer.linger_ms` | int | `10` | 批量等待时间 (ms) |
| `kafka.producer.retries` | int | `3` | 发送重试次数 |

### 6.2 配置示例 (config.yaml)

```yaml
service:
  name: "face-recognition"
  log_level: "info"
  profile: "production"

detection:
  model: "retinaface"
  model_path: "/models/retinaface.onnx"
  confidence: 0.7
  min_face_size: 80
  max_faces_per_frame: 10
  roi: null
  quality:
    blur_threshold: 100.0
    max_yaw: 45.0

recognition:
  model: "arcface"
  model_path: "/models/arcface.onnx"
  vector_dim: 512
  gpu_device: 0
  inference_backend: "onnxruntime"
  confidence_threshold: 0.65

student_api:
  url: "http://sms-platform:8080"
  match_path: "/api/face/match"
  timeout_ms: 3000
  retry_count: 2
  circuit_breaker_threshold: 5
  circuit_breaker_recovery_s: 30
  cache_ttl_min: 60
  cache_max_size: 5000
  batch_preload_enabled: true

direction:
  method: "roi_cross"
  roi_line_y: 0.6
  judge_frames: 3

dedup:
  window_sec: 10
  enabled: true

night_mode:
  enabled: true
  start_hour: 22
  end_hour: 6
  detection:
    confidence: 0.5
    min_face_size: 60
  recognition:
    threshold: 0.55

kafka:
  bootstrap_servers: ["localhost:9092"]
  consumer:
    topic: "t_dorm_frame"
    group_id: "face-recognition-group"
    auto_offset_reset: "latest"
    enable_auto_commit: false
    max_poll_records: 100
  producer:
    topic: "t_dorm_event"
    compression: "snappy"
    linger_ms: 10
    retries: 3
```

### 6.3 配置优先级

```
环境变量 > 配置文件 > 代码默认值
```

环境变量使用 `__` 作为层级分隔符：
- `DETECTION_CONFIDENCE=0.8` 覆盖 `detection.confidence`
- `STUDENT_API_URL=http://new-host:8080` 覆盖 `student_api.url`

---

## 7. 错误处理与降级策略

### 7.1 异常场景矩阵

| 异常场景 | 影响 | 检测方式 | 处理策略 | 恢复方式 |
|----------|------|---------|---------|---------|
| **学管 API 超时** | 身份匹配延迟 | HTTP 请求超时 | 重试 2 次，间隔 500ms；失败后使用本地缓存匹配 | 请求成功后恢复正常 |
| **学管 API 连续失败** | 身份匹配不可用 | 熔断器阈值触发 | 切换到本地缓存匹配模式，记录 WARN 日志 | 30s 后半开重试 |
| **学管 API 返回无匹配** | 该人标记为陌生人 | API 返回 matched=false | 标记陌生人，记录人脸快照 | 学管端补充人脸数据后恢复 |
| **本地缓存未命中** | 必须调学管 API | 本地查找无结果 | 正常调用学管 API | 成功后写入缓存 |
| **本地缓存过期** | 身份可能过时 | TTL 到期 | 删除旧条目，下次请求触发 API 调用 | 重新缓存 |
| **人脸检测漏检** | 人员未被识别 | 连续多帧无检测框 | 降低检测阈值重试，记录漏检指标 | 配置适配后恢复 |
| **低质量人脸频繁** | 大量丢弃 | 质量过滤计数高 | 调整质量阈值，记录指标告警 | 人工确认后调整配置 |
| **Kafka 消费积压** | 事件处理延迟 | Consumer Lag 监控 | 降帧处理，暂时跳帧 | 消费恢复正常后自动恢复 |
| **Kafka Broker 不可用** | 无法消费/生产 | 连接超时 | 消费者：等待重连；生产者：本地缓存事件队列（有限） | Broker 恢复后自动恢复 |
| **GPU OOM** | 推理失败 | CUDA 内存分配失败 | 自动回退到 CPU 推理，记录 CRITICAL 告警 | 人工介入或重启服务 |
| **夜间模式误切** | 参数不匹配 | 时钟同步偏差 | 基于 UTC+8 固定时区，跨日逻辑正确处理 | 无需恢复 |

### 7.2 降级策略分级

| 级别 | 触发条件 | 行为 | 用户感知 |
|------|---------|------|---------|
| **L0 — 正常** | 所有组件健康 | 全功能运行 | 无 |
| **L1 — 轻降级** | 学管 API 熔断 | 仅使用本地缓存匹配，学管匹配不可用 | 新学生（未缓存）被标记为陌生人 |
| **L2 — 中降级** | Kafka 不可用 > 30s | 停止消费帧，事件写入本地队列 | 事件延迟推送 |
| **L3 — 严重降级** | GPU 故障 | 回退 CPU 推理，检测帧率下降 | 处理延迟增加 |
| **L4 — 停服** | 模型加载失败 / 关键配置错误 | 服务拒绝启动，返回 503 | 服务不可用 |

### 7.3 日志规范

| 字段 | 说明 | 示例 |
|------|------|------|
| `timestamp` | ISO 8601 时间戳 | `2026-05-15T10:30:00.123Z` |
| `level` | 日志级别 | `INFO` / `WARN` / `ERROR` |
| `module` | 模块名 | `detection` / `recognition` / `kafka` |
| `camera_id` | 摄像头 ID | `dorm_a_entrance` |
| `message` | 日志内容 | 结构化文本 |
| `latency_ms` | 耗时（可选） | `23.5` |
| `extra` | 附加数据（可选） | `{"student_id": "2024001"}` |

---

## 8. 性能指标

### 8.1 延迟目标

| 处理阶段 | GPU (TensorRT FP16) | GPU (ONNX CUDA) | CPU (ONNX) |
|----------|---------------------|-----------------|------------|
| JPEG 解码 + 预处理 | ≤ 5ms | ≤ 5ms | ≤ 15ms |
| 人脸检测 (RetinaFace) | ≤ 15ms | ≤ 30ms | ≤ 150ms |
| 人脸对齐 + 特征提取 | ≤ 10ms | ≤ 20ms | ≤ 80ms |
| 本地缓存匹配 (5000 人) | ≤ 2ms | ≤ 2ms | ≤ 5ms |
| 学管 API 调用 (含网络) | ≤ 50ms | ≤ 50ms | ≤ 50ms |
| **端到端 (本地缓存命中)** | **≤ 35ms** | **≤ 60ms** | **≤ 250ms** |
| **端到端 (调学管 API)** | **≤ 80ms** | **≤ 105ms** | **≤ 295ms** |

### 8.2 吞吐目标

| 指标 | 目标值 | 说明 |
|------|--------|------|
| 单摄像头处理帧率 | ≥ 2 fps | 宿舍场景低帧率足够 |
| 4 路并发处理能力 | ≥ 8 fps | 每路 2fps |
| 峰值人脸处理量 | ≥ 20 人脸/秒 | 多人同时通过时 |
| 事件产出延迟 | ≤ 3 秒 | 从人脸出现在画面到事件推送 |
| Kafka 推送延迟 (P99) | ≤ 100ms | 从事件产生到 Kafka ACK |

### 8.3 资源占用

| 组件 | GPU 显存 | 内存 | 说明 |
|------|---------|------|------|
| RetinaFace 模型 (FP16) | ~400 MB | — | ONNX/TensorRT |
| ArcFace 模型 (FP16) | ~300 MB | — | ONNX/TensorRT |
| 本地特征缓存 (5000 人) | — | ~50 MB | 512-dim float32 × 5000 |
| 运行时开销 | — | ~200 MB | 帧缓冲、Kafka 缓冲等 |
| **合计** | **~700 MB** | **~250 MB** | 低功耗，旧 GPU 也可运行 |

### 8.4 精度指标

| 指标 | 目标值 | 说明 |
|------|--------|------|
| 人脸检测召回率 | ≥ 95% | 正脸、光线正常 |
| 人脸检测召回率 (红外/夜间) | ≥ 85% | 夜间红外模式 |
| 身份识别 Top-1 准确率 | ≥ 95% | 注册人脸质量良好 |
| 陌生人误识率 | ≤ 1% | 非本宿舍人员被识别为本宿舍人员 |
| 进出方向判断准确率 | ≥ 98% | ROI 线穿越法 |
| 去重正确率 | ≥ 99% | 同人 10s 窗口内仅推送一次 |

---

## 9. 附录：Kafka 消息协议

### 9.1 Topic 清单

| Topic 名称 | 分区数 | 保留策略 | 说明 |
|------------|--------|---------|------|
| `t_dorm_frame` | 4 | 12h 或 1GB | 帧数据，每个摄像头 1 分区 |
| `t_dorm_event` | 2 | 7d | 进出事件，学管系统消费 |

### 9.2 t_dorm_frame 消息协议

**Producer:** 摄像头推流服务 / Stream Gateway  
**Consumer:** 本 Face Recognition 服务  
**序列化:** JSON  

```json
{
  "frame_id": "uuid-xxxx",
  "camera_id": "dorm_a_entrance",
  "building": "宿舍A栋",
  "timestamp_unix_ms": 1747305000000,
  "image_data": "<base64_jpeg>",
  "image_width": 1920,
  "image_height": 1080,
  "sequence_num": 1523
}
```

### 9.3 t_dorm_event 消息协议

**Producer:** 本 Face Recognition 服务  
**Consumer:** 学管系统（业务后端）  
**序列化:** JSON  
**分区 Key:** `camera_id` + `event_type`  

```json
{
  "event_id": "evt_20260515_a1b2c3d4",
  "camera_id": "dorm_a_entrance",
  "building": "宿舍A栋",
  "student_id": "2024001",
  "student_name": "张三",
  "event_type": "entry",
  "confidence": 0.89,
  "face_snapshot": "/9j/4AAQ...",
  "timestamp_unix_ms": 1747305000000,
  "is_stranger": false,
  "extra": {
    "class": "计算机科学 2024-1 班",
    "dorm_room": "A-301"
  }
}
```

### 9.4 Topic 配置参数建议

```yaml
topics:
  t_dorm_frame:
    partitions: 4
    replication_factor: 1
    configs:
      cleanup.policy: "delete"
      retention.ms: 43200000        # 12 小时
      max.message.bytes: 5242880    # 5 MB (JPEG 帧)
      compression.type: "producer"

  t_dorm_event:
    partitions: 2
    replication_factor: 1
    configs:
      cleanup.policy: "delete"
      retention.ms: 604800000       # 7 天
      max.message.bytes: 1048576    # 1 MB
      compression.type: "producer"
```

---

## 附录 A：版本历史

| 版本 | 日期 | 修改人 | 变更说明 |
|------|------|--------|---------|
| v1.0 | 2026-05-15 | — | 初稿完成 |

## 附录 B：术语表

| 术语 | 说明 |
|------|------|
| **SMS** | Student Management System，学生管理系统 |
| **学管系统** | 学生管理系统的简称，SpringBoot + Vue |
| **Face Recognition** | 人脸识别模块，本 PRD 定义的服务 |
| **RetinaFace** | 基于 CNN 的人脸检测算法，支持关键点输出 |
| **MTCNN** | Multi-task Cascaded Convolutional Networks，轻量人脸检测 |
| **ArcFace** | 基于 Additive Angular Margin Loss 的人脸识别模型 |
| **ONNX Runtime** | 跨平台深度学习推理引擎 |
| **TensorRT** | NVIDIA 深度学习推理优化引擎 |
| **ROI** | Region of Interest，感兴趣区域 |
| **ROI 穿越** | 基于虚拟分割线穿越方向的进出判断方法 |
| **L2 归一化** | 将向量缩放至单位长度（模长为 1） |
| **CLAHE** | Contrast Limited Adaptive Histogram Equalization，对比度受限自适应直方图均衡化 |
| **Feature Vector** | 特征向量，512 维 float32，用于身份匹配 |
| **Embedding** | 嵌入向量，特征向量的同义术语 |
| **Cosine Similarity** | 余弦相似度，向量间相似度度量 |
| **熔断器** | Circuit Breaker，防止级联故障的保护机制 |

---

> **文档结束**
