# frontend

CampusVision AI 前端管理界面 — 基于 Vue 3 + TypeScript + Element Plus 的宿舍智能监控 SPA。

## 概述

frontend 是 CampusVision AI 的管理控制台，提供实时监控、摄像头管理、考勤查询、告警处理、人脸记录等功能。基于 [RuoYi-Vue3-ts](https://github.com/zzh948498/RuoYi-Vue3-ts) 框架二次开发。

| 属性 | 值 |
|---|---|
| 框架 | Vue 3.2 + TypeScript |
| UI 库 | Element Plus 2.2 |
| 构建工具 | Vite 3.2 |
| 状态管理 | Pinia |
| 路由 | Vue Router 4 |
| 包管理 | pnpm |
| 开发端口 | 80 |

## 目录结构

```
frontend/
├── src/
│   ├── api/                 # 后端 API 封装
│   │   ├── camera.ts        # 摄像头管理
│   │   ├── events.ts        # 进出事件
│   │   ├── alerts.ts        # 告警
│   │   ├── attendance.ts    # 考勤统计
│   │   ├── face.ts          # 人脸匹配/特征
│   │   ├── dashboard.ts     # 仪表盘数据
│   │   ├── config.ts        # 系统配置
│   │   └── login.ts         # 登录鉴权
│   ├── views/               # 页面组件
│   │   ├── dashboard/       # 仪表盘 (概览统计)
│   │   ├── monitor/         # 实时监控
│   │   ├── camera/          # 摄像头管理
│   │   ├── events/          # 进出事件查询
│   │   ├── alerts/          # 告警管理
│   │   ├── attendance/      # 考勤记录
│   │   ├── face/            # 人脸记录
│   │   ├── config/          # 系统配置
│   │   └── system/          # 系统管理 (用户/角色/菜单/字典)
│   ├── router/              # 路由配置
│   ├── store/               # Pinia 状态管理
│   ├── layout/              # 布局组件
│   ├── components/          # 公共组件
│   ├── plugins/             # 插件
│   ├── hooks/               # 组合式函数
│   ├── locales/             # 国际化 (vue-i18n)
│   └── utils/               # 工具函数
├── public/
├── vite.config.ts
├── tailwind.config.cjs
├── tsconfig.json
├── package.json
└── Dockerfile
```

## 快速开始

### 前置条件

- Node.js 18+
- pnpm (`npm install -g pnpm`)
- dormitory-service-go 运行中 (后端 API :8083)

### 本地开发

```bash
cd frontend

# 安装依赖
pnpm install

# 启动开发服务器
pnpm dev
# → http://localhost:80
```

### 构建

```bash
# 生产构建
pnpm build:prod

# 测试环境构建
pnpm build:stage

# 预览构建产物
pnpm preview
```

### Docker 运行

```bash
docker compose up -d frontend
```

## 环境配置

| 文件 | 用途 |
|---|---|
| `.env.development` | 开发环境 (API → localhost:8083) |
| `.env.staging` | 测试环境 |
| `.env.production` | 生产环境 |

## 功能模块

### 业务功能

| 模块 | 说明 |
|---|---|
| 仪表盘 | 在校人数统计、今日进出趋势、告警概览、摄像头状态 |
| 实时监控 | 宿舍楼栋实时人员状态 |
| 摄像头管理 | 摄像头 CRUD、状态监控、RTSP 配置 |
| 进出事件 | 按楼栋/时间/人员查询进出记录 |
| 告警管理 | 陌生人告警、行为告警，支持处理/忽略 |
| 考勤记录 | 每日考勤统计、缺勤查询 |
| 人脸记录 | 人脸特征库管理 |
| 系统配置 | 系统参数动态配置 |

### 系统功能 (RuoYi 内置)

用户管理、角色权限、菜单管理、字典管理、操作日志、登录日志、定时任务、服务监控。

## 技术栈

| 类别 | 技术 |
|---|---|
| 核心 | Vue 3.2 (Composition API), TypeScript |
| UI | Element Plus, Animate.css, Tailwind CSS |
| 图表 | ECharts 5 |
| 状态 | Pinia |
| 路由 | Vue Router 4 |
| HTTP | Axios |
| 国际化 | vue-i18n |
| 富文本 | WangEditor, Vue Quill |
| 测试 | Vitest, @vue/test-utils |
| 构建 | Vite 3, unplugin-auto-import |

## 测试

```bash
cd frontend

# 运行测试
pnpm test

# 监听模式
pnpm test:watch
```

## 代码规范

- ESLint + Prettier (`.eslintrc.js`, `.prettierrc`)
- TypeScript 严格模式
- EditorConfig (`.editorconfig`)
