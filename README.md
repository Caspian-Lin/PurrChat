<div align="center">

<img src="docs/logo.svg" alt="PurrChat" width="120" height="120">

# PurrChat

**Intimate &middot; Refined &middot; Alive**

Exploring emotional connection and creative boundaries between humans and AI.

[![CI](https://github.com/Caspian-Lin/PurrChat/actions/workflows/ci.yml/badge.svg?branch=dev)](https://github.com/Caspian-Lin/PurrChat/actions/workflows/ci.yml?query=branch%3Adev)

[English](#english) &middot; [简体中文](#简体中文)

</div>

---

## English

### What Is PurrChat?

PurrChat is a full-stack real-time chat platform with **Bot Studio**, a visual event-driven bot orchestration system for building AI characters with complex behavior directly inside chat.

PurrChat follows **Soft Architecture**: it is not just a tool, but a quiet, refined, breathing space where conversations with people and AI can flow naturally.

<!-- Add project screenshots here when available. -->
<!-- ![Chat](docs/screenshots/chat.png "PurrChat chat interface") -->
<!-- ![Bot Studio](docs/screenshots/bot-studio.png "Bot Studio DAG editor") -->
<!-- ![Debug Panel](docs/screenshots/debug.png "Bot debug panel") -->

### Highlights

#### Bot Studio - Visual Event Chain Engine

Bot Studio is the core creative surface in PurrChat. The DAG editor lets users compose bot behavior with drag-and-drop nodes instead of writing code.

- **Event-driven workflows**: Trigger -> condition branch -> action.
- **Built-in actions**: reply, delayed message, group nickname updates, user tag matching, and more.
- **External integrations**: LLM, HTTP tools, Dify, n8n.
- **Debug panel**: trace event execution and inspect variables in real time.
- **Bot discovery**: search and deploy community-created bots.

#### Soft Architecture Design System

PurrChat is designed as a spatial experience rather than a stack of utility panels.

- **Living Geometry**: six-level radius system with no hard corners.
- **Breathing Space**: more generous spacing than typical chat tools.
- **Material Honesty**: sage green accent, low-saturation mineral color, and blue-toned dark mode.

#### Multi-Client Architecture

| Client  | Stack                            | Status         |
| ------- | -------------------------------- | -------------- |
| Web     | Vue 3 + Vite + Tailwind CSS      | Supported      |
| Desktop | Tauri 2 (Rust)                   | Supported      |
| Mobile  | Tauri 2 (Rust + Android WebView) | In development |

### Tech Stack

```text
Monorepo (Turborepo + pnpm)
├── Frontend          Vue 3 · Vite · Tailwind CSS · Pinia · IndexedDB
├── Backend           Go · Gin · PostgreSQL · WebSocket
└── Storage           Go · Cloudflare R2 / MinIO (two-step upload)
```

| Layer    | Technology              | Why                                                                |
| -------- | ----------------------- | ------------------------------------------------------------------ |
| Frontend | Vue 3 (Composition API) | Reactive model fits event-driven UI well                           |
| Build    | Vite + Turborepo        | Fast HMR and incremental monorepo builds                           |
| Backend  | Go                      | High-concurrency WebSocket management and single-binary deployment |
| Database | PostgreSQL              | JSONB support for flexible bot configuration                       |
| Realtime | Native WebSocket        | Multi-client identity, heartbeat, and message caching              |
| Storage  | R2 / MinIO              | Presigned upload flow with low server bandwidth usage              |
| Desktop  | Tauri 2                 | Native performance with reusable Rust-side capabilities            |
| Mobile   | Tauri 2 Mobile          | Shared frontend with native Android integration                    |

### Quick Start

#### Requirements

- Node.js >= 20
- pnpm >= 9
- Go >= 1.24
- Docker & Docker Compose for PostgreSQL

Optional Android build requirements:

- JDK 17+
- Android SDK API 35+
- Android NDK r27+
- Rust Android targets: `rustup target add aarch64-linux-android x86_64-linux-android`

#### Run Locally

```bash
git clone https://github.com/Caspian-Lin/PurrChat.git
cd PurrChat

pnpm install
docker compose up -d postgres
pnpm dev
```

| Service     | URL                   |
| ----------- | --------------------- |
| Frontend    | http://localhost:5173 |
| Backend API | http://localhost:8080 |
| Storage     | http://localhost:8081 |

#### Environment

Copy `.env.example` to `.env`, then review the service examples:

- [`apps/backend/.env.example`](apps/backend/.env.example) - backend API settings
- [`apps/storage/.env.example`](apps/storage/.env.example) - storage service settings
- [`apps/frontend/.env.example`](apps/frontend/.env.example) - frontend settings

### Repository Structure

```text
PurrChat/
├── apps/
│   ├── frontend/               # Vue 3 frontend (Web + Tauri Desktop/Mobile)
│   │   ├── src-tau/             # Tauri Rust backend + Android/iOS projects
│   │   └── src/
│   │       ├── components/      # UI components
│   │       │   └── home/panel/bots/   # Bot Studio components
│   │       ├── stores/          # Pinia stores
│   │       └── utils/           # Utilities and event-chain helpers
│   ├── backend/                 # Go backend API
│   │   └── internal/
│   │       ├── botengine/       # Bot event-chain engine
│   │       ├── handlers/        # HTTP handlers
│   │       └── websocket/       # WebSocket connection management
│   └── storage/                 # Go file storage service
│       └── internal/providers/  # R2 / MinIO providers
├── docs/                        # Project documentation
├── docker-compose.yml
├── turbo.json
└── Makefile
```

### Feature Overview

#### Chat Core

- User registration and login with JWT authentication.
- Friend requests, accept/reject flow, and blocklist.
- One-to-one realtime chat over WebSocket.
- Group chat creation, management, and member roles.
- Avatar and background image uploads with presigned URLs.
- Message cache and incremental loading with IndexedDB.
- Multi-client online sessions across Web and Tauri.

#### Bot Studio

- Bot CRUD: create, edit, enable, and disable bots.
- Deploy bots to direct messages or group chats.
- Trigger and reply system with keywords, regular expressions, and user tags.
- Workflow mode with event chains and external integrations (LLM, Dify, n8n).
- Visual DAG editor for drag-and-drop orchestration.
- Debug panel for event-chain tracing.
- Bot discovery and sharing.

### Documentation

| Document                                                 | Description                       |
| -------------------------------------------------------- | --------------------------------- |
| [Design System](docs/DESIGN_SYSTEM.md)                   | Soft Architecture design rules    |
| [Deployment Guide](docs/DEPLOYMENT.md)                   | Docker deployment and Nginx setup |
| [Roadmap](docs/ROADMAP.md)                               | Feature planning and progress     |
| [Commit Convention](docs/COMMIT_CONVENTION.md)           | Git commit message rules          |
| [WebSocket Architecture](docs/WEBSOCKET_ARCHITECTURE.md) | WebSocket event management        |
| [WebSocket Debug Guide](docs/WEBSOCKET_DEBUG_GUIDE.md)   | WebSocket troubleshooting         |

### Common Commands

```bash
pnpm dev              # Start frontend, backend, and storage in development mode
pnpm build            # Build production artifacts
pnpm test             # Run tests
pnpm lint             # Run lint checks
make docker-up        # Start the full stack with Docker
make docker-down      # Stop Docker services
```

#### Android Build

```bash
make android-dev            # Development mode with a device or emulator
make android-build-debug    # Build debug APK
make android-build-release  # Build release APKs split by CPU architecture
make android-build-apk      # Build universal release APK
make android-keystore       # Generate release signing keystore
make android-clean          # Clean build artifacts
```

Set `ANDROID_HOME`, `NDK_HOME`, and `JAVA_HOME` in `~/.bashrc` before Android builds.

### Roadmap

- [ ] Typing indicator, message recall, and read receipts.
- [ ] Offline message sync and complete file-message support.
- [ ] External bot integration with the OneBot 11 protocol.
- [ ] Internationalization.
- [ ] Rich media messages: voice, video, stickers.
- [ ] Desktop enhancements: tray and auto-start.

See [Roadmap](docs/ROADMAP.md) for details.

### Contributing

Contributions are welcome. Please read [Commit Convention](docs/COMMIT_CONVENTION.md) before opening a pull request.

---

## 简体中文

### PurrChat 是什么？

PurrChat 是一个全栈实时聊天平台，内置 **Bot Studio**：一个可视化、事件驱动的 Bot 编排系统，让你可以在聊天中构建具有复杂行为的 AI 角色。

PurrChat 采用 **Soft Architecture（柔软建筑）** 设计理念：它不是一个工具，而是一个安静、精致、有呼吸感的空间，对话在其中自然流淌，无论对方是人还是 AI。

<!-- 后续可在这里补充项目截图。 -->
<!-- ![主界面](docs/screenshots/chat.png "PurrChat 主聊天界面") -->
<!-- ![Bot Studio](docs/screenshots/bot-studio.png "Bot Studio DAG 编辑器") -->
<!-- ![调试面板](docs/screenshots/debug.png "Bot 调试面板") -->

### 核心亮点

#### Bot Studio - 可视化事件链引擎

Bot Studio 是 PurrChat 的核心创作界面。通过 **DAG（有向无环图）编辑器**，你可以用拖拽方式编排 Bot 的行为逻辑，无需编写代码。

- **事件驱动工作流**：触发器 -> 条件分支 -> 动作。
- **内置动作库**：回复消息、延迟发送、修改群名片、用户标签匹配等。
- **Python 沙箱**：支持在事件链中嵌入自定义 Python 逻辑。
- **调试面板**：实时追踪事件流执行过程并查看变量状态。
- **Bot 发现**：搜索和部署社区创建的 Bot。

#### Soft Architecture 设计系统

PurrChat 的 UI 不是功能面板的堆砌，而是经过完整设计系统规范的空间体验。

- **Living Geometry**：6 级统一圆角系统，消灭所有直角。
- **Breathing Space**：比一般聊天应用更慷慨的间距。
- **Material Honesty**：Sage 绿强调色、低饱和矿物感、蓝调深灰暗色模式。

#### 多端架构

| 客户端  | 技术                             | 状态   |
| ------- | -------------------------------- | ------ |
| Web     | Vue 3 + Vite + Tailwind CSS      | 已支持 |
| Desktop | Tauri 2 (Rust)                   | 已支持 |
| Mobile  | Tauri 2 (Rust + Android WebView) | 开发中 |

### 技术栈

```text
Monorepo (Turborepo + pnpm)
├── Frontend          Vue 3 · Vite · Tailwind CSS · Pinia · IndexedDB
├── Backend           Go · Gin · PostgreSQL · WebSocket
└── Storage           Go · Cloudflare R2 / MinIO (两阶段上传)
```

| 层面     | 技术选型                | 理由                                      |
| -------- | ----------------------- | ----------------------------------------- |
| 前端框架 | Vue 3 (Composition API) | 响应式系统与事件驱动模型天然契合          |
| 构建工具 | Vite + Turborepo        | 极速 HMR，Monorepo 增量构建               |
| 后端语言 | Go                      | 高并发 WebSocket 连接管理，单二进制部署   |
| 数据库   | PostgreSQL              | JSONB 支持灵活的 Bot 配置存储             |
| 实时通信 | 原生 WebSocket          | 多端身份识别、连接心跳、消息缓存          |
| 文件存储 | R2 / MinIO              | 预签名 URL 两阶段上传，降低服务端带宽压力 |
| 桌面端   | Tauri 2                 | 原生性能，Rust 侧能力可复用               |
| 移动端   | Tauri 2 Mobile          | 共享前端代码，接入 Android 原生能力       |

### 快速开始

#### 环境要求

- Node.js >= 20
- pnpm >= 9
- Go >= 1.24
- Docker & Docker Compose（用于 PostgreSQL）

Android 构建可选要求：

- JDK 17+
- Android SDK API 35+
- Android NDK r27+
- Rust Android targets：`rustup target add aarch64-linux-android x86_64-linux-android`

#### 本地启动

```bash
git clone https://github.com/Caspian-Lin/PurrChat.git
cd PurrChat

pnpm install
docker compose up -d postgres
pnpm dev
```

| 服务        | 地址                  |
| ----------- | --------------------- |
| Frontend    | http://localhost:5173 |
| Backend API | http://localhost:8080 |
| Storage     | http://localhost:8081 |

#### 环境配置

复制 `.env.example` 到 `.env`，再参考各服务示例配置：

- [`apps/backend/.env.example`](apps/backend/.env.example) - 后端 API 配置
- [`apps/storage/.env.example`](apps/storage/.env.example) - 存储服务配置
- [`apps/frontend/.env.example`](apps/frontend/.env.example) - 前端配置

### 项目结构

```text
PurrChat/
├── apps/
│   ├── frontend/               # Vue 3 前端 (Web + Tauri Desktop/Mobile)
│   │   ├── src-tau/             # Tauri Rust 后端 + Android/iOS 原生项目
│   │   └── src/
│   │       ├── components/      # UI 组件
│   │       │   └── home/panel/bots/   # Bot Studio 组件
│   │       ├── stores/          # Pinia 状态管理
│   │       └── utils/           # 工具函数和事件链辅助逻辑
│   ├── backend/                 # Go 后端 API
│   │   └── internal/
│   │       ├── botengine/       # Bot 事件链引擎
│   │       ├── handlers/        # HTTP 处理器
│   │       └── websocket/       # WebSocket 连接管理
│   └── storage/                 # Go 文件存储服务
│       └── internal/providers/  # R2 / MinIO 提供者
├── docs/                        # 项目文档
├── docker-compose.yml
├── turbo.json
└── Makefile
```

### 功能概览

#### 聊天核心

- 用户注册/登录（JWT 认证）。
- 好友系统（请求、接受、拒绝、黑名单）。
- 一对一实时聊天（WebSocket）。
- 群聊创建、管理与成员角色。
- 头像和背景图上传（两阶段预签名）。
- 消息缓存与增量加载（IndexedDB）。
- Web/Tauri 多客户端同时在线。

#### Bot Studio

- Bot CRUD：创建、编辑、启用、禁用 Bot。
- 部署到会话：一键将 Bot 部署到私聊或群聊。
- 触发与回复系统：关键词、正则、用户标签触发。
- 工作流模式：事件链引擎 + Python 沙箱。
- DAG 可视化编辑器：拖拽编排事件流。
- 调试面板：实时追踪事件链执行。
- Bot 发现与分享。

### 文档

| 文档                                             | 说明                       |
| ------------------------------------------------ | -------------------------- |
| [设计系统](docs/DESIGN_SYSTEM.md)                | Soft Architecture 设计规范 |
| [部署指南](docs/DEPLOYMENT.md)                   | Docker 部署、Nginx 配置    |
| [开发路线图](docs/ROADMAP.md)                    | 功能规划与开发进度         |
| [提交规范](docs/COMMIT_CONVENTION.md)            | Git Commit Message 规范    |
| [WebSocket 架构](docs/WEBSOCKET_ARCHITECTURE.md) | WebSocket 事件管理         |
| [WebSocket 调试](docs/WEBSOCKET_DEBUG_GUIDE.md)  | WebSocket 问题排查         |

### 常用命令

```bash
pnpm dev              # 启动开发模式：前端 + 后端 + 存储服务
pnpm build            # 构建生产版本
pnpm test             # 运行测试
pnpm lint             # 代码检查
make docker-up        # Docker 一键启动全栈
make docker-down      # Docker 停止
```

#### Android 构建

```bash
make android-dev            # 开发模式，需连接设备或模拟器
make android-build-debug    # 构建 debug APK
make android-build-release  # 构建按 CPU 架构拆分的 release APK
make android-build-apk      # 构建 release 通用 APK
make android-keystore       # 生成 release 签名 keystore
make android-clean          # 清理构建产物
```

Android 构建前需要在 `~/.bashrc` 中配置 `ANDROID_HOME`、`NDK_HOME`、`JAVA_HOME`。

### 路线图

- [ ] 输入状态广播、消息撤回、已读回执。
- [ ] 离线消息推送、文件消息完整支持。
- [ ] 外部 Bot 接入（OneBot 11 协议）。
- [ ] 国际化（i18n）。
- [ ] 消息多媒体（语音、视频、表情包）。
- [ ] 桌面端增强（系统托盘、开机自启）。

详见 [开发路线图](docs/ROADMAP.md)。

### 贡献

欢迎贡献。请阅读 [提交规范](docs/COMMIT_CONVENTION.md) 后提交 Pull Request。

---

<div align="center">

**Built with care by [Caspian-Lin](https://github.com/Caspian-Lin)**

</div>
