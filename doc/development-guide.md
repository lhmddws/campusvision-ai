# CampusVision AI — 开发指南

> 本文档面向参与 CampusVision AI 项目的开发者，涵盖环境搭建、编码规范、Git 工作流、测试与 CI/CD。

---

## 目录

1. [开发环境](#1-开发环境)
2. [本地搭建](#2-本地搭建)
3. [编码规范](#3-编码规范)
4. [Git 工作流](#4-git-工作流)
5. [测试](#5-测试)
6. [CI/CD](#6-cicd)
7. [常见问题](#7-常见问题)

---

## 1. 开发环境

### 1.1 推荐工具

| 工具 | 版本要求 | 用途 |
|------|---------|------|
| Python | ≥ 3.11 | AI Engine |
| Go | ≥ 1.22 | Stream Gateway |
| Node.js | ≥ 20 | Web 前端 |
| Docker | ≥ 24 | 本地基础设施 |
| NVIDIA Driver | ≥ 545 | GPU 推理 |
| CUDA | ≥ 12.4 | GPU 加速 |
| VSCode / IDEA | - | IDE |

### 1.2 检查清单

```bash
# 确认各工具已安装
python3 --version      # ≥ 3.11
go version             # ≥ 1.22
node --version         # ≥ 20
docker --version       # ≥ 24
nvidia-smi             # GPU 驱动正常
nvcc --version         # CUDA ≥ 12.4
```

---

## 2. 本地搭建

### 2.1 克隆仓库

```bash
git clone http://192.168.113.82/lhmddws/campusvision-ai.git
git clone http://192.168.113.82/lhmddws/campusvision-ai-engine.git
git clone http://192.168.113.82/lhmddws/campusvision-stream-gateway.git
git clone http://192.168.113.82/lhmddws/campusvision-web.git
```

### 2.2 启动基础设施

项目依赖 Kafka、Redis、MariaDB、MinIO。使用 Docker Compose 一键启动：

```bash
# 基础服务
docker compose up -d

# 确认所有服务正常运行
docker compose ps

# 查看日志
docker compose logs -f
```

### 2.3 AI Engine

```bash
cd campusvision-ai-engine

# Python 虚拟环境
python3 -m venv .venv
source .venv/bin/activate

# 安装依赖
pip install -r requirements.txt

# 下载模型权重
# 将 YOLO、InsightFace 等权重放入 weights/ 目录

# 启动开发服务
python app/main.py --config config.dev.yaml
```

### 2.4 Stream Gateway

```bash
cd campusvision-stream-gateway

# 安装依赖
go mod download

# 启动
go run cmd/gateway/main.go --config config.dev.yaml

# 热重载（安装 air）
go install github.com/air-verse/air@latest
air --config .air.toml
```

### 2.5 Web 前端

```bash
cd campusvision-web

npm install
npm run dev     # 开发服务器，默认 http://localhost:5173
```

### 2.7 环境变量参考

每个子仓库的 `config.dev.yaml` 或 `application-dev.yml` 中包含完整配置项。关键变量：

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `KAFKA_BROKERS` | Kafka 地址 | `localhost:9092` |
| `REDIS_URL` | Redis 地址 | `redis://localhost:6379` |
| `MARIADB_DSN` | 数据库连接 | `mariadb://localhost:3306/campusvision` |
| `MINIO_ENDPOINT` | MinIO 地址 | `http://localhost:9000` |
| `CUDA_VISIBLE_DEVICES` | GPU 设备号 | `0` |

---

## 3. 编码规范

### 3.1 Python (AI Engine)

- **格式化**: Black (`black .`)
- **Lint**: Ruff (`ruff check .`)
- **类型注解**: 所有函数签名必须包含类型注解
- **Pydantic**: 所有配置使用 Pydantic 模型
- **异步**: IO 操作使用 `asyncio`
- **命名**: `snake_case` 变量/函数, `PascalCase` 类

```python
# ✅ 正确示例
async def detect_persons(
    frame: np.ndarray,
    confidence_threshold: float = 0.5,
) -> list[Detection]:
    """对单帧进行人体检测。"""
    ...

# ❌ 避免
def do_stuff(img, thr):
    pass
```

### 3.2 Go (Stream Gateway)

- **格式化**: `go fmt`
- **Lint**: `golangci-lint`
- **错误处理**: 每个错误必须检查，不吞 error
- **日志**: 使用结构化日志 (slog / zap)
- **命名**: `camelCase` 不导出, `PascalCase` 导出

```go
// ✅ 正确示例
func NewGateway(cfg *Config) (*Gateway, error) {
    if cfg == nil {
        return nil, errors.New("config is required")
    }
    // ...
}

// ❌ 避免
func NewGateway(cfg *Config) *Gateway {
    return &Gateway{config: cfg}
}
```

### 3.3 TypeScript / Vue (Web)

- **TypeScript**: 严格模式，禁止 `any`
- **组件**: Composition API + `<script setup>`
- **状态管理**: Pinia
- **命名**: 组件 `PascalCase`, 文件 `kebab-case`
- **CSS**: 使用 CSS Modules 或 Scoped

```typescript
// ✅ 正确示例
const cameraList = ref<Camera[]>([])

async function fetchCameras() {
  cameraList.value = await cameraApi.list(queryParams)
}
```

### 3.5 通用规范

- 禁止提交 `.env`、密钥、密码到仓库
- 所有配置通过环境变量或配置中心注入
- 日志输出使用对应语言的日志框架，禁止 `print` / `console.log`（调试除外）
- API 返回统一格式：`{ code, message, data }`
- 所有新增 API 必须有操作日志注解

---

## 4. Git 工作流

### 4.1 分支策略

```
main        ← 生产就绪代码
release/    ← 预发布分支
dev         ← 开发集成分支
feature/*   ← 功能开发 （从 dev 切出）
hotfix/*    ← 紧急修复 （从 main 切出）
```

### 4.2 工作流程

```bash
# 1. 从 dev 创建 feature 分支
git checkout dev
git pull
git checkout -b feature/xxx

# 2. 开发 & 提交
git add .
git commit -m "feat: 添加摄像头注册功能"

# 3. 提交信息格式
# <type>: <简短描述>
# type: feat / fix / refactor / docs / test / chore

# 4. 合并到 dev
git checkout dev
git merge feature/xxx

# 5. 删除 feature 分支
git branch -d feature/xxx
```

### 4.3 Commit 规范

```
feat:      新功能
fix:       Bug 修复
refactor:  重构
docs:      文档
test:      测试
chore:     构建/工具
style:     格式调整（不影响逻辑）

示例:
  feat: 添加 RTSP 拉流断线重连
  fix: 修复人脸识别批次内存泄漏
  refactor: 提取帧处理为独立 pipeline
  docs: 更新 API 文档
  test: 添加摄像头注册单元测试
```

### 4.4 合并请求 (MR)

1. 确保分支通过 CI
2. 至少 1 人 Code Review
3. 解决所有 Review 意见后方可合并
4. 合并方式：**Squash merge**（保持 main 历史干净）

---

## 5. 测试

### 5.1 测试要求

| 层级 | 覆盖率要求 | 框架 |
|------|-----------|------|
| AI Engine 单元测试 | ≥ 70% | pytest |
| Stream Gateway 单元测试 | ≥ 60% | go test |
| Web 组件测试 | - | Vitest |
| E2E | 核心流程 | Playwright |

### 5.2 运行测试

```bash
# AI Engine
cd campusvision-ai-engine
pytest --cov=app --cov-report=term

# Stream Gateway
cd campusvision-stream-gateway
go test ./... -cover

# Web
cd campusvision-web
npm run test
```

### 5.3 测试数据

- 单元测试使用 Mock，不依赖真实数据库/摄像头
- 集成测试使用 `testcontainers` (Java) 或 Docker Compose
- 测试用的 RTSP 视频流使用本地模拟推流

---

## 6. CI/CD

### 6.1 CI Pipeline

每个 MR 自动触发：

```
Lint → Unit Test → Build → Image Scan
```

### 6.2 CD Pipeline

合并到 `main` 后自动部署：

```
Build Image → Push Registry → Deploy Staging → E2E Test → Deploy Production
```

### 6.3 Docker 镜像规范

```
# 命名
registry.internal/campusvision/{service-name}:{version}

# 标签
{version}    # 语义化版本
latest       # 最新稳定版
git-{sha}    # 具体提交（回滚用）
```

---

## 7. 常见问题

### 7.1 GPU 不可用

```bash
# 检查驱动
nvidia-smi

# 检查 Docker GPU 支持
docker run --rm --gpus all nvidia/cuda:12.4-base nvidia-smi

# 检查 NVIDIA Container Toolkit
which nvidia-container-runtime
```

### 7.2 RTSP 拉流失败

- 确认摄像头 RTSP 地址可达
- 检查 FFmpeg 是否正确解码：`ffprobe rtsp://...`
- 查看 Gateway 日志中的错误信息

### 7.3 Kafka 连接失败

```bash
# 测试连接
kcat -b localhost:9092 -L

# 确认配置
echo $KAFKA_BROKERS
```

### 7.4 数据库迁移

参考 `infra/mariadb/migrations/` 目录下的 SQL 文件，按编号顺序手动执行。
