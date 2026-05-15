# CampusVision AI — Stream Gateway 产品需求文档

> **文档编号**: PRD-001  
> **模块名称**: Stream Gateway（流网关服务）  
> **所属系统**: 学生管理系统（Student Management System）— 宿舍管理 AI 子系统  
> **版本**: v2.0  
> **最后更新**: 2026-05-15  
> **状态**: 初稿  

---

## 目录

1. [模块定位](#1-模块定位)
2. [技术选型与约束](#2-技术选型与约束)
3. [整体数据流](#3-整体数据流)
4. [功能清单](#4-功能清单)
5. [配置清单](#5-配置清单)
6. [接口定义](#6-接口定义)
7. [错误处理](#7-错误处理)
8. [性能指标与容量](#8-性能指标与容量)
9. [附录：Kafka 消息协议](#9-附录kafka-消息协议)

---

## 1. 模块定位

### 1.1 系统归属

Stream Gateway 是**学生管理系统（SMS）** 下属的宿舍管理 AI 子系统的流处理入口模块。本服务**不是**通用安防视频分析平台，而是专注校园宿舍场景的轻量化辅助系统。

**一句话定义**：从 4 路宿舍入口 RTSP 摄像头拉流 → FFmpeg 解码 → 抽帧 → Kafka 推送的实时流处理管道。

### 1.2 业务背景

宿舍查寝是高校学生管理的刚性需求。传统查寝依赖学生干部逐间敲门/扫码/签到，流程繁琐、数据滞后、难以追溯。本系统通过宿舍楼入口摄像头自动采集进出人员帧数据，后续由 AI 子系统完成人脸识别和轨迹判断，最终产出每晚宿舍在寝/离寝统计，**将查寝流程简化为「打开系统 → 查看报表」两个步骤**。

### 1.3 部署规模

| 项目 | 规格 |
|------|------|
| 宿舍楼栋 | 4 栋（A/B/C/D 栋） |
| 每栋摄像头 | 1 路（入口处，仅判断进出） |
| 摄像头总数 | **4 路**（固定，无扩展计划） |
| 人员轨迹 | 仅判断两种状态：**进楼 / 出楼** |
| 核心产出 | 每晚宿舍在寝统计报表 |

### 1.4 核心职责

| 职责 | 说明 |
|------|------|
| **拉流** | 同时管理 4 路宿舍入口 RTSP 摄像头流，支持断线自动重连 |
| **解码** | 将 H.264 编码流解码为原始帧 |
| **抽帧** | 按可配置帧率从视频流中选取有效帧，控制下游负载。支持**动态抽帧**（有人进入画面时才抽帧，降低 70%+ 无效帧） |
| **推送** | 将帧数据以 JPEG 编码推送到 Kafka topic `t_dorm_frame`，供 AI 推理服务消费 |
| **健康上报** | 4 路摄像头在线状态、推流 FPS、内存/CPU 监控上报对接学管平台 |

### 1.5 非职责（本模块绝不包含）

- ❌ 不做人脸检测/识别（由下游 AI Engine 处理）
- ❌ 不判断人员进出/在寝状态（由下游 AI Engine + 学管平台处理）
- ❌ 不涉及业务逻辑（查寝统计、报表等）
- ❌ 不存储视频或帧数据（纯实时流转发管道）
- ❌ 不含 Web 前端页面
- ❌ 不管理摄像头设备注册（4 路静态配置）
- ❌ 不做录像/回放

### 1.6 与上下游的协作

```
学管系统 (SpringBoot + Vue)
  │
  ├── 提供人脸底库 API（按需查询身份）
  │   └── Stream Gateway 不直接调用，下游 AI Engine 调用
  │
  ├── 接收摄像头健康状态
  │   └── Stream Gateway → Kafka → 学管系统
  │
  └── 配置管理（摄像头 RTSP 地址等）
      └── Stream Gateway 读取本地配置文件
```

---

## 2. 技术选型与约束

### 2.1 技术栈

| 组件 | 技术 | 版本要求 | 用途 |
|------|------|---------|------|
| 服务语言 | **Go** | ≥ 1.22 | 主服务框架，高并发协程模型，适合流处理场景 |
| RTSP 拉流解码 | **FFmpeg** | ≥ 6.0 | libavformat/libavcodec 拉流 + 软件解码 |
| IPC 管道 | FFmpeg pipe / gocv | — | Go 与 FFmpeg 进程间帧数据传递 |
| 消息队列 | **Kafka** | ≥ 3.7 | 帧数据推送（单 topic `t_dorm_frame`） |
| JPEG 编码 | Go `image/jpeg` 标准库 / libjpeg | — | 帧数据压缩编码 |
| 配置管理 | YAML 文件 | — | 4 路摄像头静态配置 |
| 可观测 | Prometheus + 简单 HTTP | — | 健康检查与基础指标 |
| 日志 | slog (Go 标准库) | Go 1.22+ | 结构化 JSON 日志 |

### 2.2 为什么选择 Go

| 因素 | 说明 |
|------|------|
| **并发模型** | Goroutine 天然适合管理多路独立摄像头协程，每路一个 goroutine + 共享 Kafka producer |
| **部署轻量** | 单一二进制，无运行时依赖，适合作为独立 JAR 包旁的服务进程 |
| **FFmpeg 集成** | 通过 exec pipe 或 gocv binding 与 FFmpeg 进程通信，Go 负责协程编排和资源管理 |
| **生态匹配** | 与 Kafka 集成成熟（confluent-kafka-go / sarama），`image/jpeg` 标准库免额外依赖 |

### 2.3 环境约束

| 约束项 | 规格 |
|--------|------|
| 操作系统 | Linux (Ubuntu 22.04+) / macOS（开发） |
| 部署方式 | 独立二进制进程，与学管系统 JAR 包分离部署 |
| CPU | **不需要 GPU**。纯软件解码（4 路 × 720p × 5fps 负载极低） |
| 内存 | ≤ 128MB 基准内存 |
| 摄像头协议 | 仅 RTSP (TCP)，H.264 编码 |
| 摄像头分辨率 | 720p ~ 1080p（推荐 720p） |
| 摄像头特性 | 需支持红外夜视（宿舍夜间场景核心需求） |

### 2.4 为什么不需要 GPU

```
4 路 × 720p × 5fps 解码 ≈ 20 帧/秒
单核软件解码能力       ≈ 30-60 帧/秒 (720p H.264)

⇒ 不需要 GPU，纯 CPU 完全可承载
⇒ 降低部署成本和运维复杂度
```

---

## 3. 整体数据流

### 3.1 端到端数据链路

```
┌─────────────────────────────────────────────────────────────┐
│                       学生宿舍区域                            │
│                                                             │
│  ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐ │
│  │ A栋入口   │   │ B栋入口   │   │ C栋入口   │   │ D栋入口   │ │
│  │ 摄像头    │   │ 摄像头    │   │ 摄像头    │   │ 摄像头    │ │
│  │ RTSP     │   │ RTSP     │   │ RTSP     │   │ RTSP     │ │
│  └────┬─────┘   └────┬─────┘   └────┬─────┘   └────┬─────┘ │
│       │              │              │              │        │
└───────┼──────────────┼──────────────┼──────────────┼────────┘
        │ RTSP (TCP)   │              │              │
        ▼              ▼              ▼              ▼
┌─────────────────────────────────────────────────────────────┐
│              Stream Gateway (Go)                             │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐     │
│  │                   拉流管理器                          │     │
│  │                                                     │     │
│  │  goroutine-A栋  goroutine-B栋  goroutine-C栋  ...   │     │
│  │     │               │              │                 │     │
│  │     ▼               ▼              ▼                 │     │
│  │  FFmpeg pipe    FFmpeg pipe    FFmpeg pipe           │     │
│  │  (H.264→YUV)    (H.264→YUV)    (H.264→YUV)          │     │
│  └─────────────────────────────────────────────────────┘     │
│                              │                                │
│                              ▼                                │
│  ┌─────────────────────────────────────────────────────┐     │
│  │                   抽帧引擎                           │     │
│  │                                                     │     │
│  │  ① 按 5fps 固定抽帧                                  │     │
│  │  ② 动态抽帧（画面变化检测，有人才输出帧）               │     │
│  │  ③ 缩放至 1280×720（可配）                            │     │
│  │  ④ JPEG 编码（quality=80）                            │     │
│  │  ⑤ 丢弃无效帧（夜间全黑帧过滤）                        │     │
│  └─────────────────────────────────────────────────────┘     │
│                              │                                │
│                              ▼                                │
│  ┌─────────────────────────────────────────────────────┐     │
│  │                   Kafka 推送器                       │     │
│  │                                                     │     │
│  │  Topic: t_dorm_frame                                │     │
│  │  Key:   building (A/B/C/D) → 保证楼栋有序            │     │
│  │  Value: { camera_id, building, timestamp,            │     │
│  │           frame_data(JPEG) }                         │     │
│  └─────────────────────────────────────────────────────┘     │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────────────────┐
│                     Kafka                                     │
│                    t_dorm_frame                                │
│  ┌──────────┬──────────┬──────────┬──────────┬──────────┐    │
│  │Partition0│Partition1│Partition2│Partition3│...        │    │
│  │  A栋      │  B栋      │  C栋      │  D栋      │          │    │
│  └──────────┴──────────┴──────────┴──────────┴──────────┘    │
└──────────────────────┬────────────────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────────────────┐
│  Face Recognition Service (Python)                            │
│  - 消费 t_dorm_frame 帧数据                                    │
│  - 调用学管系统 API 获取人脸底库                                 │
│  - 人脸检测 → 人脸识别 → 进出判断                                │
│  - 产出：谁、什么时候、进了/出了哪栋楼                            │
│  → 最终：每晚宿舍在寝统计报表                                    │
└──────────────────────────────────────────────────────────────┘
```

### 3.2 消息流向

| 方向 | Topic | 数据内容 | 生产者 | 消费者 |
|------|-------|---------|--------|--------|
| 帧数据下发 | `t_dorm_frame` | JPEG 编码帧 + 元数据 | Stream Gateway | Face Recognition Service |

> **注意**：Stream Gateway 只产出 1 个 Kafka Topic，不存在状态上报、事件通知、配置下发等额外 Topic。摄像头状态通过 HTTP 健康检查接口暴露给学管系统轮询或拉取。

### 3.3 与上下游的边界

| 交互对象 | 通信方式 | 传输内容 | 说明 |
|----------|---------|---------|------|
| 摄像头设备 | RTSP (TCP) | H.264 视频流 | 只读拉流，不反向控制摄像头 |
| Face Recognition Service | Kafka `t_dorm_frame` | JPEG 帧 + 元数据 | 单向推送，Gateway 不关心消费结果 |
| 学管系统 | HTTP `/health` | 服务健康状态 | 学管系统定期轮询/拉取健康状态 |

---

## 4. 功能清单

### 4.1 RTSP 拉流管理

#### F1.1 固定 4 路 RTSP 并发拉流

| 属性 | 值 |
|------|-----|
| **功能描述** | 从 4 栋宿舍楼入口的固定摄像头拉取 RTSP 视频流。每路摄像头由独立 goroutine 管理，互不影响。配置硬编码在 YAML 配置文件中，**不支持运行时动态增删**。 |
| **优先级** | P0 |
| **输入** | 配置文件中 `cameras[4]` 数组，每项含：building（楼栋标识）, rtsp_url（RTSP 地址） |
| **输出** | H.264 编码数据包 → 送入解码管道 |
| **配置项** | `cameras[].rtsp_url`, `cameras[].building`, `cameras[].enabled` |

#### F1.2 拉流协议参数配置

| 属性 | 值 |
|------|-----|
| **功能描述** | RTSP 拉流参数可配置：传输协议固定 TCP（可靠性优于 UDP）、连接超时时间、读超时时间。支持按摄像头粒度覆盖。 |
| **优先级** | P1 |
| **输入** | 摄像头级参数覆盖 |
| **输出** | 对应参数生效后的 RTSP 连接 |
| **配置项** | `rtsp.transport`（固定 tcp）、`rtsp.timeout_ms`、`rtsp.read_timeout_ms` |

#### F1.3 断线自动重连（指数退避）

| 属性 | 值 |
|------|-----|
| **功能描述** | 当 RTSP 拉流因网络波动或摄像头重启断线时自动重连。重连间隔指数退避：1s → 2s → 4s → 8s → … → max_interval。达到最大重试次数后标记摄像头离线，不再自动重连，等待人工介入。 |
| **优先级** | P0 |
| **输入** | 断线事件（网络错误、超时、EOF） |
| **输出** | 重连尝试 → 成功恢复 或 标记离线（日志告警） |
| **配置项** | `reconnect.max_retries`、`reconnect.interval_ms`、`reconnect.max_interval_ms` |

#### F1.4 摄像头启动/停止控制

| 属性 | 值 |
|------|-----|
| **功能描述** | 支持通过 HTTP 接口手动控制指定楼栋摄像头的拉流启动和停止。用于维护场景（如摄像头检修时停止拉流，避免大量重连日志）。 |
| **优先级** | P1 |
| **输入** | HTTP POST `/api/camera/{building}/start` 或 `/api/camera/{building}/stop` |
| **输出** | 摄像头 goroutine 启动/停止 + 状态切换 |
| **配置项** | `control.api_enabled`（是否启用控制接口） |

---

### 4.2 视频解码与抽帧

#### F2.1 FFmpeg 软件解码

| 属性 | 值 |
|------|-----|
| **功能描述** | 使用 FFmpeg `libavcodec` 对 H.264 编码流进行软件解码。通过创建 FFmpeg 子进程（exec pipe）或嵌入式调用，将输出帧通过管道传递到 Go 进程。**纯 CPU 解码，不依赖 GPU**。 |
| **优先级** | P0 |
| **输入** | H.264 编码数据包 |
| **输出** | YUV420P 格式原始帧 |
| **配置项** | `decoder.type`（固定 software）、`decoder.output_format`（固定 yuv420p） |

#### F2.2 帧率控制（默认 5fps）

| 属性 | 值 |
|------|-----|
| **功能描述** | 控制输出帧率。宿舍入口场景人员通行速度慢，5fps 即可完整捕捉进出动作，不会漏检。支持按摄像头粒度配置（如夜间可降至 2fps 降低负载）。 |
| **优先级** | P0 |
| **输入** | 解码后的全帧序列 |
| **输出** | 按目标帧率筛选后的帧子集 |
| **配置项** | `cameras[].fps`（默认 5） |

**帧率选取依据**：

```
宿舍入口人员正常步行速度 ≈ 1.0-1.5 m/s
摄像头到入口距离      ≈ 3-5 米
通过时间              ≈ 2-5 秒

5fps → 通过期间可捕获 10-25 帧
→ 足够下游 AI 做准确的人脸检测和识别
→ 相比 30fps 降低 83% 的数据量
```

#### F2.3 分辨率控制（默认 720p）

| 属性 | 值 |
|------|-----|
| **功能描述** | 统一输出分辨率为 1280×720（720p）。宿舍场景无需超高分辨率，720p 足够人脸检测。减少帧体积，降低 Kafka 网络带宽和下游 AI 处理压力。 |
| **优先级** | P1 |
| **输入** | 原始分辨率帧（可能为 1080p 或更高） |
| **输出** | 统一 720p 分辨率帧 |
| **配置项** | `cameras[].resolution`（默认 "1280x720"） |

#### F2.4 夜间低光照适配

| 属性 | 值 |
|------|-----|
| **功能描述** | 宿舍夜间场景的摄像头画面普遍存在低光照、高噪点、红外照明不均匀等问题。本模块需兼容以下输入质量下降情况：<br>① 不因画面过暗而断开 RTSP 连接<br>② 容忍高噪点输入（解码不报错）<br>③ 全黑画面（无人员通过时）仍然正常解码，但抽帧引擎会丢弃纯黑帧<br>④ 遇到全黑/纯色帧时静默丢弃，不触发异常告警 |
| **优先级** | P1 |
| **输入** | 低光照/高噪点/红外模式下解码后的帧 |
| **输出** | 正常帧继续后续处理，无效帧（全黑/纯色）静默丢弃 |
| **配置项** | `frame_filter.min_brightness`（亮度阈值，低于此值视为无效帧）、`frame_filter.black_frame_ratio`（黑色像素占比阈值） |

**低光照场景说明**：

```
宿舍入口典型场景：
- 白天：正常光照，画面清晰
- 晚间22:00后：仅保留门厅照明 + 摄像头红外补光
- 凌晨0:00-6:00：仅红外模式，画面黑白，噪点明显
- 06:00-08:00：自然光逐渐恢复

摄像头选型建议：
- 支持红外夜视（至少 20m 有效距离）
- 宽动态范围（WDR）≥ 120dB
- 最低照度 ≤ 0.01 Lux（彩色）/ 0 Lux（红外）
```

#### F2.5 动态抽帧——有人才抽帧

| 属性 | 值 |
|------|-----|
| **功能描述** | 仅在画面中存在运动物体（有人进入画面）时进行抽帧，画面静止时跳过解码后的帧不推送。通过帧间差异检测实现：计算连续帧的像素变化率，超过阈值时判定为"有人"。<br><br>**效果**：宿舍夜间场景，大部分时间无人进出，动态抽帧可减少 70-90% 的无效帧推送，大幅降低 Kafka 负载和 AI 引擎处理压力。 |
| **优先级** | P0 |
| **输入** | 连续解码帧序列 |
| **输出** | 动态判定 → 有人时正常抽帧推送 | 无人时丢弃帧（不推送） |
| **配置项** | `dynamic_frame.enabled`（默认 true）、`dynamic_frame.motion_threshold`（运动检测灵敏度 0.0-1.0） |

**动态抽帧 vs 固定抽帧对比**：

| 指标 | 固定 5fps | 动态抽帧 + 5fps |
|------|-----------|-----------------|
| 日均帧数（单路） | ~432,000 | ~20,000-80,000（取决于通行量） |
| 4 路日均帧总数 | ~1,728,000 | ~80,000-320,000 |
| 日均 Kafka 流量 | ~173 GB | ~8-32 GB |
| AI Engine 负载 | 高（持续处理） | 低（峰值处理） |

#### F2.6 JPEG 编码与质量控制

| 属性 | 值 |
|------|-----|
| **功能描述** | 抽帧输出的原始 YUV 帧经 JPEG 编码后推送。JPEG 质量可配置（默认 80%），平衡图像质量与传输带宽。编码后的帧数据为 `t_dorm_frame` 消息的 `frame_data` 字段。 |
| **优先级** | P0 |
| **输入** | YUV 原始帧 |
| **输出** | JPEG 编码字节数组 |
| **配置项** | `jpeg_quality`（默认 80，范围 1-100） |

---

### 4.3 帧推送（Kafka）

#### F3.1 帧数据推送

| 属性 | 值 |
|------|-----|
| **功能描述** | 将 JPEG 编码后的帧数据推送至 Kafka topic `t_dorm_frame`。每条消息包含：<br>- `camera_id`：摄像头 ID（如 cam-a）<br>- `building`：楼栋标识（A/B/C/D）<br>- `timestamp`：捕获时间戳（Unix 毫秒）<br>- `frame_data`：JPEG 编码的帧图像数据 |
| **优先级** | P0 |
| **输入** | JPEG 编码帧 + 元数据 |
| **输出** | Kafka `t_dorm_frame` 消息 |
| **配置项** | `kafka.topic`（默认 "t_dorm_frame"）、`kafka.brokers` |

#### F3.2 消息分区（按楼栋）

| 属性 | 值 |
|------|-----|
| **功能描述** | 使用 `building`（楼栋标识）作为 Kafka 消息 Key，确保同一栋楼的帧路由到同一 Partition。4 栋楼 → 4 个 Key → Kafka 自动分配 Partition。保证 AI Engine 消费时同一栋楼的帧有序。 |
| **优先级** | P0 |
| **输入** | 帧消息 + building |
| **输出** | 已分区的 Kafka 消息 |
| **配置项** | `kafka.partition_key`（默认 "building"，固定值） |

#### F3.3 批量发送与压缩

| 属性 | 值 |
|------|-----|
| **功能描述** | Kafka Producer 配置批量发送参数减少网络往返：<br>① `batch_size`：单批次最大字节数<br>② `linger.ms`：批次最大等待时间（低帧率场景下避免无限等待）<br>③ `compression`：启用 snappy 压缩减少网络带宽 |
| **优先级** | P1 |
| **输入** | 帧消息 + Producer 配置 |
| **输出** | 批量 Kafka 请求 |
| **配置项** | `kafka.batch_size`、`kafka.linger_ms`、`kafka.compression` |

#### F3.4 单帧大小控制

| 属性 | 值 |
|------|-----|
| **功能描述** | 通过 JPEG 质量参数控制单帧大小。质量 80 时，720p JPEG 帧约 30-80KB。可在配置中调整以适配网络带宽。 |
| **优先级** | P1 |
| **输入** | 帧尺寸 + JPEG 质量参数 |
| **输出** | 符合大小预期的 JPEG 帧 |
| **配置项** | `jpeg_quality`（默认 80） |

**单帧大小估算**：

```
720p (1280×720) JPEG quality=80：
  白天正常光照：约 50-80 KB/帧
  低光照/红外模式：约 30-50 KB/帧（噪点被压缩，实际体积较小）
  纯黑帧（已过滤）：0（不推送）

单路 5fps 带宽：约 250-400 KB/s
4 路总带宽：约 1-1.6 MB/s
日总量：约 84-138 GB（未压缩）
Kafka snappy 压缩后：约 50-80 GB
```

---

### 4.4 健康监测

#### F4.1 摄像头在线状态

| 属性 | 值 |
|------|-----|
| **功能描述** | 实时追踪每路摄像头的拉流状态。状态包括：`running`（正常拉流）、`connecting`（正在连接）、`reconnecting`（重连中）、`offline`（离线）。通过 HTTP 健康检查接口暴露。 |
| **优先级** | P0 |
| **输入** | 拉流协程内部状态 |
| **输出** | HTTP `/health` 响应中的摄像头状态段 |
| **配置项** | 无（基于拉流协程状态自动汇报） |

#### F4.2 推流 FPS 统计

| 属性 | 值 |
|------|-----|
| **功能描述** | 统计每路摄像头的实时推流帧率（实际推送到 Kafka 的帧率，非解码帧率）。通过 `/health` 接口暴露。当实际帧率持续低于配置帧率的 50% 超过 30 秒时，输出 WARN 日志。 |
| **优先级** | P1 |
| **输入** | 推送计数器 |
| **输出** | 帧率统计（暴露于 `/health`） |
| **配置项** | `monitor.fps_warn_threshold`（告警阈值比例，默认 0.5） |

#### F4.3 进程资源监控

| 属性 | 值 |
|------|-----|
| **功能描述** | 采集 Go 进程基础资源指标：内存使用（RSS）、CPU 使用率、goroutine 数量。通过 `/health` 接口暴露。用于学管系统统一监控面板展示。 |
| **优先级** | P1 |
| **输入** | Go runtime 指标 + OS 统计 |
| **输出** | 资源指标（暴露于 `/health`） |
| **配置项** | 无 |

---

## 5. 配置清单

### 5.1 配置项总表

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| **摄像头配置** | | | |
| `cameras[].building` | string | - | 楼栋标识（A/B/C/D） |
| `cameras[].rtsp_url` | string | - | 摄像头 RTSP 地址 |
| `cameras[].enabled` | bool | true | 是否启用此路拉流 |
| `cameras[].fps` | int | 5 | 抽帧帧率 |
| `cameras[].resolution` | string | "1280x720" | 解码输出分辨率 |
| **RTSP 拉流** | | | |
| `rtsp.transport` | string | "tcp" | 传输协议（仅支持 tcp） |
| `rtsp.timeout_ms` | int | 10000 | 连接超时（毫秒） |
| `rtsp.read_timeout_ms` | int | 5000 | 读取超时（毫秒） |
| **重连策略** | | | |
| `reconnect.max_retries` | int | 10 | 最大重连次数，超过标记离线 |
| `reconnect.interval_ms` | int | 5000 | 首次重连间隔（毫秒） |
| `reconnect.max_interval_ms` | int | 120000 | 最大重连间隔（毫秒，2分钟） |
| `reconnect.multiplier` | float | 2.0 | 退避倍数 |
| **帧处理** | | | |
| `jpeg_quality` | int | 80 | JPEG 压缩质量（1-100） |
| `frame_filter.min_brightness` | int | 10 | 最小平均亮度（0-255），低于此丢弃 |
| `frame_filter.black_frame_ratio` | float | 0.95 | 黑色像素占比阈值 |
| **动态抽帧** | | | |
| `dynamic_frame.enabled` | bool | true | 是否启用动态抽帧 |
| `dynamic_frame.motion_threshold` | float | 0.05 | 运动检测灵敏度（像素变化率） |
| `dynamic_frame.check_interval` | int | 3 | 运动检测间隔帧数（每 N 帧检测一次） |
| **Kafka** | | | |
| `kafka.brokers` | []string | ["localhost:9092"] | Kafka Broker 列表 |
| `kafka.topic` | string | "t_dorm_frame" | 帧数据 Topic |
| `kafka.partitions` | int | 4 | Topic 分区数 |
| `kafka.replication_factor` | int | 1 | 副本因子 |
| `kafka.batch_size` | int | 65536 | Producer 批量发送字节数 |
| `kafka.linger_ms` | int | 100 | 批次最大等待时间 |
| `kafka.compression` | string | "snappy" | 压缩算法（none/snappy/lz4/zstd） |
| `kafka.max_message_bytes` | int | 1048576 | 最大消息字节数（1MB） |
| `kafka.retries` | int | 3 | 发送重试次数 |
| **HTTP 服务** | | | |
| `http.port` | int | 9100 | HTTP 服务监听端口 |
| `http.health_path` | string | "/health" | 健康检查路径 |
| **日志** | | | |
| `log.level` | string | "info" | 日志级别（debug/info/warn/error） |
| `log.format` | string | "json" | 输出格式（json/text） |

### 5.2 完整 YAML 配置示例

```yaml
# stream-gateway.yaml
#
# 注意：本服务仅用于宿舍管理子系统，固定 4 路摄像头
#       不需要 GPU，纯 CPU 解码

cameras:
  - building: "A"
    rtsp_url: "rtsp://admin:pass@192.168.1.10:554/stream1"
    enabled: true
    fps: 5
    resolution: "1280x720"
  - building: "B"
    rtsp_url: "rtsp://admin:pass@192.168.1.11:554/stream1"
    enabled: true
    fps: 5
    resolution: "1280x720"
  - building: "C"
    rtsp_url: "rtsp://admin:pass@192.168.1.12:554/stream1"
    enabled: true
    fps: 5
    resolution: "1280x720"
  - building: "D"
    rtsp_url: "rtsp://admin:pass@192.168.1.13:554/stream1"
    enabled: true
    fps: 5
    resolution: "1280x720"

rtsp:
  transport: "tcp"           # 宿舍内网，TCP 稳定性更好
  timeout_ms: 10000          # 10s 连接超时
  read_timeout_ms: 5000      # 5s 读取超时

reconnect:
  max_retries: 10            # 重连 10 次后标记离线
  interval_ms: 5000          # 首次 5s
  max_interval_ms: 120000    # 最多 2min
  multiplier: 2.0            # 指数退避

dynamic_frame:
  enabled: true              # 默认启用动态抽帧
  motion_threshold: 0.05     # 5% 像素变化即触发
  check_interval: 3          # 每 3 帧检测一次

jpeg_quality: 80             # 平衡质量与带宽

frame_filter:
  min_brightness: 10         # 丢弃全黑帧
  black_frame_ratio: 0.95

kafka:
  brokers:
    - "192.168.100.10:9092"
    - "192.168.100.11:9092"
    - "192.168.100.12:9092"
  topic: "t_dorm_frame"
  partitions: 4              # 4 栋楼，4 个分区
  replication_factor: 1
  batch_size: 65536
  linger_ms: 100
  compression: "snappy"
  max_message_bytes: 1048576
  retries: 3

http:
  port: 9100
  health_path: "/health"

log:
  level: "info"
  format: "json"
```

### 5.3 配置优先级

```
环境变量 > YAML 配置文件 > 程序默认值
```

环境变量命名规则：大写 + 下划线，层级字段用 `__` 分隔。

示例：
| 环境变量 | 对应配置项 |
|----------|-----------|
| `KAFKA_BROKERS` | `kafka.brokers` |
| `CAMERAS__0__RTSP_URL` | `cameras[0].rtsp_url` |
| `JPEG_QUALITY` | `jpeg_quality` |
| `LOG_LEVEL` | `log.level` |

---

## 6. 接口定义

### 6.1 内部 HTTP 接口（仅运维用）

#### 6.1.1 健康检查

```
GET /health
```

**Response：**
```json
{
  "service": "campusvision-stream-gateway",
  "version": "2.0.0",
  "timestamp": "2026-05-15T22:30:00+08:00",
  "uptime_seconds": 86400,
  "status": "ok",
  "cameras": {
    "total": 4,
    "online": 4,
    "offline": 0,
    "details": [
      {
        "building": "A",
        "status": "running",
        "fps": 4.8,
        "uptime_seconds": 86400,
        "total_frames": 414720
      },
      {
        "building": "B",
        "status": "running",
        "fps": 5.1,
        "uptime_seconds": 86400,
        "total_frames": 432000
      },
      {
        "building": "C",
        "status": "reconnecting",
        "fps": 0,
        "uptime_seconds": 0,
        "total_frames": 345600,
        "reconnect_count": 3
      },
      {
        "building": "D",
        "status": "running",
        "fps": 4.9,
        "uptime_seconds": 86400,
        "total_frames": 423360
      }
    ]
  },
  "resources": {
    "memory_mb": 45.2,
    "cpu_percent": 2.3,
    "goroutines": 12
  },
  "kafka": {
    "status": "ok",
    "producer_latency_ms": 3
  }
}
```

**HTTP 状态码：**

| 场景 | 状态码 |
|------|--------|
| 全部摄像头正常推送 | 200 |
| 部分摄像头离线（≥1 路故障但仍可运行） | 200（在线信息在 body 中） |
| 服务内部错误（Kafka 不可用等） | 503 |

#### 6.1.2 摄像头控制（可选）

```
POST /api/camera/{building}/start
POST /api/camera/{building}/stop
```

**Response：**
```json
{
  "building": "A",
  "action": "start",
  "status": "connecting"
}
```

### 6.2 与学管系统的协作接口

Stream Gateway **不直接**对接学管系统 API。下游 Face Recognition Service（Python）负责：

1. 消费 Kafka `t_dorm_frame` 获取帧数据
2. 调用学管系统人脸查询 API 获取底库
3. 完成人脸检测 → 识别 → 进出判断

Stream Gateway 仅通过 `/health` 接口向学管系统暴露运行状态，学管系统定期拉取。

---

## 7. 错误处理

### 7.1 异常场景与处理策略

| 异常场景 | 影响 | 检测方式 | 处理策略 | 恢复方式 |
|----------|------|---------|---------|---------|
| **RTSP 连接超时** | 某路摄像头无数据 | FFmpeg 返回超时错误 | 记录日志，触发指数退避重连 | 重连成功恢复 |
| **RTSP 认证失败（401）** | 某路摄像头不可用 | RTSP 返回认证错误 | 记录 ERROR 日志，标记摄像头 error，**不自动重连** | 人工修正配置后重启 |
| **网络闪断** | 多路同时断流 | 多个协程检测到 EOF | 逐路独立重连（加随机抖动避免重连风暴） | 逐路独立恢复 |
| **FFmpeg 进程异常退出** | 某路解码中断 | 子进程 exit 信号 | 记录 ERROR，重启 FFmpeg 子进程 | 子进程重启成功恢复 |
| **Kafka Broker 不可用** | 全局帧推送阻塞 | Producer 返回错误 | ① 队列积压限速 ② 丢帧保活 ③ Broker恢复自动恢复推送 | Broker 恢复后继续 |
| **消息过大** | 单条消息发送失败 | `max.message.bytes` 超限 | 降低 JPEG 质量或分辨率，重试 | 持续失败则丢弃并告警 |
| **持续全黑帧** | 推送大量无效帧 | 帧亮度检测 | 静默丢弃，不告警（宿舍夜间正常现象） | 亮度恢复后正常推送 |
| **收到 SIGTERM/SIGINT** | 服务关闭 | OS 信号 | 优雅关闭：停止拉流 → 刷新 Kafka Producer → 退出 | 预期行为 |

### 7.2 错误分级

| 严重级别 | 触发场景 | 响应方式 |
|----------|---------|---------|
| **ERROR** | 摄像头认证失败、FFmpeg 进程崩溃、Kafka 不可用超过 30s | 日志 ERROR，服务标记 degraded |
| **WARN** | 摄像头断线重连、帧率低于阈值、单条消息发送失败 | 日志 WARN |
| **INFO** | 摄像头上线/离线、重连成功、服务启动/停止 | 日志 INFO |

---

## 8. 性能指标与容量

### 8.1 目标性能指标

| 指标 | 目标值 | 说明 |
|------|--------|------|
| 最大拉流路数 | 4 路（固定） | 无扩展计划 |
| 端到端延迟（摄像头→Kafka） | ≤ 1000ms | 含解码 + 抽帧 + JPEG 编码 + Kafka 推送 |
| 帧率稳定性 | ≥ 配置值的 90% | 目标 5fps 时实测 ≥ 4.5fps |
| 丢帧率（正常工况） | ≤ 0.1% | 只在 Kafka 不可用时丢帧 |
| 重连恢复时间（网络恢复后） | ≤ 10s + 退避等待 | 含连接建立时间 |
| CPU 使用率（4 路满负载） | ≤ 15%（单核） | 纯软件解码 |
| 内存使用（基准） | ≤ 128 MB | 不含帧缓冲峰值 |
| 优雅关闭超时 | ≤ 15s | SIGTERM → 进程退出 |

### 8.2 容量模型

| 场景 | CPU | 内存 | 网络出向带宽 | Kafka 吞吐 |
|------|-----|------|-------------|-----------|
| 4 路 × 720p × 5fps × jpeg80 | ≤ 15% 单核 | ≤ 128 MB | ≤ 2 MB/s | ~20 msg/s |
| 4 路 × 720p × 5fps × 动态抽帧 | ≤ 5% 单核 | ≤ 128 MB | ≤ 0.5 MB/s | ~5 msg/s |

---

## 9. 附录：Kafka 消息协议

### 9.1 Topic 定义

| 属性 | 值 |
|------|-----|
| Topic 名称 | `t_dorm_frame` |
| 分区数 | 4（按楼栋，建议 = 楼栋数） |
| 副本因子 | 1（开发环境）/ 2（生产建议） |
| 保留策略 | delete（48h 自动清理） |
| 压缩类型 | producer |

### 9.2 消息格式

**Key**：`building`（string，A/B/C/D）

**Value**（JSON）：

```json
{
  "camera_id": "cam-a",
  "building": "A",
  "timestamp": 1747305000000,
  "frame_sequence": 12345,
  "frame_data": "<base64_encoded_jpeg>",
  "frame_width": 1280,
  "frame_height": 720,
  "jpeg_quality": 80,
  "is_dynamic": true
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `camera_id` | string | 摄像头 ID（cam-a / cam-b / cam-c / cam-d） |
| `building` | string | 楼栋标识（A/B/C/D），同时也是消息 Key |
| `timestamp` | int64 | 帧捕获时间（Unix 毫秒，UTC+8） |
| `frame_sequence` | int64 | 帧序号（本摄像头独立递增） |
| `frame_data` | string | Base64 编码的 JPEG 帧数据 |
| `frame_width` | int | 帧宽度 |
| `frame_height` | int | 帧高度 |
| `jpeg_quality` | int | 编码质量（1-100） |
| `is_dynamic` | bool | 是否由动态抽帧触发 |

> **为什么用 JSON 而非 Protobuf**？
> - 4 路 × 5fps 的数据量使用 JSON 完全可承载
> - 降低开发和调试成本（人类可读，无需编译 proto）
> - 若后续数据量大幅增长可切换为 Protobuf（兼容协议，仅改序列化层）

### 9.3 消息大小估算

| 场景 | 单帧大小 | 单路日量 | 4 路日总量 |
|------|---------|---------|-----------|
| 白天正常帧 | ~60 KB | ~26 GB | ~104 GB |
| 夜间红外帧 | ~40 KB | ~17 GB | ~69 GB |
| 动态抽帧（日均） | ~50 KB avg | ~4-16 GB | ~16-64 GB |
| Base64 膨胀（+33%） | 以上值 × 1.33 | — | — |

> **建议**：在 Kafka Producer 端启用 `snappy` 压缩，可减少 30-50% 网络传输量。

### 9.4 消费者建议

Face Recognition Service（Python）消费 `t_dorm_frame` 时的建议配置：

```python
# 消费者配置示例
consumer_config = {
    'bootstrap.servers': 'kafka:9092',
    'group.id': 'face-recognition-group',
    'auto.offset.reset': 'latest',
    'enable.auto.commit': True,
    'auto.commit.interval.ms': 5000,
    'max.poll.interval.ms': 300000,
    'fetch.min.bytes': 1024,
    'fetch.max.wait.ms': 500,
}
```

---

> **文档结束**

---

**修订记录**：

| 版本 | 日期 | 修改内容 | 作者 |
|------|------|---------|------|
| v2.0 | 2026-05-15 | 重构为宿舍管理子系统定位。移除通用安防能力，精简为 4 路固定配置、纯 CPU 解码、单 Kafka Topic、动态抽帧、低光照适配。简化配置清单和接口定义。 | — |
