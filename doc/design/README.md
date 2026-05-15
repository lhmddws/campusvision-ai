# CampusVision AI — 技术设计文档

> **版本**: v1.0 · **更新**: 2026-05-15  
> **说明**: 本目录存放后端技术设计文档，按角色/领域分文件夹组织。

---

## 文档结构

```
design/
├── README.md                          ← 本索引
│
├── backend/                           ← Java 后端开发（搭档）
│   ├── 01-architecture.md             ← Java 架构设计：项目骨架、分层、配置、Kafka/Redis/调度
│   ├── 02-database.md                 ← 数据库设计：DDL、索引、分区、Flyway、Redis Key
│   └── 03-api.md                      ← API 设计：22个端点完整请求/响应规格、错误码
│
├── integration/                       ← 系统集成（搭档）
│   └── 01-main-process.md            ← 主进程对接设计：Maven模块提取、Controller迁移、认证合并、回退方案
│
└── camera/                            ← 摄像头功能实现（搭档）
    └── 01-camera-feature.md          ← 摄像头技术设计：健康检查、离线检测、CRUD、与感知层接口约定
```

---

## 角色与文档映射

| 角色 | 负责内容 | 文档 |
|------|---------|------|
| **Java 后端开发** | 实现所有业务逻辑、数据库、API | `backend/01-architecture.md` + `02-database.md` + `03-api.md` |
| **系统集成工程师** | 将 dormitory-service 接入主 SpringBoot 进程 | `integration/01-main-process.md` |
| **摄像头功能开发者** | 实现摄像头设备管理、状态监控 | `camera/01-camera-feature.md` |

---

## 文档引用关系

```
PRD-004 主进程对接
    ↓ 细化
backend/01-architecture.md  →  Spring Boot 配置、分层、消费者
backend/02-database.md      →  11 张表 DDL、索引、迁移脚本
backend/03-api.md           →  22 个 REST 端点、错误码
    ↓
integration/01-main-process.md  →  独立 JAR → 主进程模块的 8 步迁移

PRD-005 摄像头功能实现
    ↓ 细化
camera/01-camera-feature.md →  健康检查 Task、离线检测、CRUD 实现
```

---

## 快速入口

| 你需要做什么 | 先读这个 |
|------------|---------|
| 启动开发环境、跑通项目 | `backend/01-architecture.md` → 配置节 |
| 建表、写 Mapper | `backend/02-database.md` → DDL + Flyway |
| 对接前端 API | `backend/03-api.md` → 响应格式 + 端点 |
| 把模块接进主进程 | `integration/01-main-process.md` → 8步迁移清单 |
| 实现摄像头状态监控 | `camera/01-camera-feature.md` → 健康检查 + 离线检测 |
