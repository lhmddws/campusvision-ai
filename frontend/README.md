# frontend

CampusVision AI frontend management console — a Vue 3 + TypeScript + Element Plus SPA for dormitory intelligent monitoring.

## Overview

The frontend is the management console for CampusVision AI, providing real-time monitoring, camera management, attendance queries, alert handling, and face records. Built on top of [RuoYi-Vue3-ts](https://github.com/zzh948498/RuoYi-Vue3-ts).

| Property | Value |
|---|---|
| Framework | Vue 3.2 + TypeScript |
| UI Library | Element Plus 2.2 |
| Build Tool | Vite 3.2 |
| State Management | Pinia |
| Router | Vue Router 4 |
| Package Manager | pnpm |
| Dev Port | 80 |

## Directory Structure

```
frontend/
├── src/
│   ├── api/                 # Backend API wrappers
│   │   ├── camera.ts        # Camera management
│   │   ├── events.ts        # Entry/exit events
│   │   ├── alerts.ts        # Alerts
│   │   ├── attendance.ts    # Attendance statistics
│   │   ├── face.ts          # Face matching/features
│   │   ├── dashboard.ts     # Dashboard data
│   │   ├── config.ts        # System config
│   │   └── login.ts         # Login auth
│   ├── views/               # Page components
│   │   ├── dashboard/       # Dashboard (overview stats)
│   │   ├── monitor/         # Real-time monitoring
│   │   ├── camera/          # Camera management
│   │   ├── events/          # Entry/exit event queries
│   │   ├── alerts/          # Alert management
│   │   ├── attendance/      # Attendance records
│   │   ├── face/            # Face records
│   │   ├── config/          # System configuration
│   │   └── system/          # System admin (users/roles/menus/dicts)
│   ├── router/              # Route configuration
│   ├── store/               # Pinia state management
│   ├── layout/              # Layout components
│   ├── components/          # Shared components
│   ├── plugins/             # Plugins
│   ├── hooks/               # Composables
│   ├── locales/             # i18n (vue-i18n)
│   └── utils/               # Utility functions
├── public/
├── vite.config.ts
├── tailwind.config.cjs
├── tsconfig.json
├── package.json
└── Dockerfile
```

## Quick Start

### Prerequisites

- Node.js 18+
- pnpm (`npm install -g pnpm`)
- dormitory-service-go running (backend API :8083)

### Local Development

```bash
cd frontend

# Install dependencies
pnpm install

# Start dev server
pnpm dev
# → http://localhost:80
```

### Build

```bash
# Production build
pnpm build:prod

# Staging build
pnpm build:stage

# Preview build output
pnpm preview
```

### Docker

```bash
docker compose up -d frontend
```

## Environment Configuration

| File | Purpose |
|---|---|
| `.env.development` | Development (API → localhost:8083) |
| `.env.staging` | Staging |
| `.env.production` | Production |

## Features

### Business Modules

| Module | Description |
|---|---|
| Dashboard | On-campus headcount, daily entry/exit trends, alert overview, camera status |
| Real-time Monitor | Live personnel status per dormitory building |
| Camera Management | Camera CRUD, status monitoring, RTSP configuration |
| Entry/Exit Events | Query entry/exit records by building/time/person |
| Alert Management | Stranger alerts, behavior alerts — handle or dismiss |
| Attendance | Daily attendance statistics, absence queries |
| Face Records | Face feature library management |
| System Config | Dynamic system parameter configuration |

### System Modules (RuoYi built-in)

User management, role permissions, menu management, dictionary management, operation logs, login logs, scheduled tasks, service monitoring.

## Tech Stack

| Category | Technology |
|---|---|
| Core | Vue 3.2 (Composition API), TypeScript |
| UI | Element Plus, Animate.css, Tailwind CSS |
| Charts | ECharts 5 |
| State | Pinia |
| Router | Vue Router 4 |
| HTTP | Axios |
| i18n | vue-i18n |
| Rich Text | WangEditor, Vue Quill |
| Testing | Vitest, @vue/test-utils |
| Build | Vite 3, unplugin-auto-import |

## Testing

```bash
cd frontend

# Run tests
pnpm test

# Watch mode
pnpm test:watch
```

## Code Standards

- ESLint + Prettier (`.eslintrc.js`, `.prettierrc`)
- TypeScript strict mode
- EditorConfig (`.editorconfig`)
