
# PurrChat
---
<div align="center">

<img src="docs/image.png" alt="PurrChat">




A space where you compose AI characters with real behavior, memory and agency — then talk to them inside a quiet, breathing chat.

[![CI](https://github.com/Caspian-Lin/PurrChat/actions/workflows/ci.yml/badge.svg?branch=dev)](https://github.com/Caspian-Lin/PurrChat/actions/workflows/ci.yml?query=branch%3Adev)

> Active development: [Caspian-Lin/PurrChat](https://github.com/Caspian-Lin/PurrChat) — this repository is the public stable mirror.

[English](#english) &middot; [简体中文](#简体中文)

<!-- Demo GIF — replace with a real recording when available -->
<!-- <img src="docs/demo/bot-studio.gif" alt="Bot Studio demo" width="720"> -->

</div>

---

## English

### What Is PurrChat?

PurrChat is a full-stack real-time chat platform built around one idea: **an AI character should be something you author, not something you prompt.**

At its heart is **Bot Studio** — a visual, event-driven orchestration system where you compose an agent's *personality, memory, decision flow and tools* as a drag-and-drop workflow. The agent then lives inside your conversations: it listens for the right moments, recalls context, branches on conditions, calls LLMs or external services, and replies as a character with continuity.

PurrChat follows **Soft Architecture**: it is not a stack of utility panels, but a quiet, refined, breathing space where conversations with people and AI can flow naturally.

### Compose an Agent, Not a Prompt

Most chat tools treat the AI as a single text-in/text-out function. PurrChat treats it as a **runtime actor** with state, memory and a behavior graph you can see and edit.

| Concern            | How PurrChat models it                                              |
| ------------------ | ------------------------------------------------------------------- |
| **Personality**    | Authored through reply nodes, LLM prompts and persona config        |
| **Memory**         | History nodes pull conversation context into every decision         |
| **Behavior flow**  | A DAG workflow: trigger → condition → action, with loop / switch    |
| **Tools**          | LLM, HTTP tools, Dify, n8n — wired as external nodes                |
| **Agency**         | Each conversation spins up an isolated actor with its own context   |
| **Trust**          | A capability model declares, grants and enforces what an agent can do |

### Highlights

#### Bot Studio — Visual Agent Orchestration

The DAG editor is the core creative surface. Compose agent behavior with nodes instead of code.

- **Event-driven workflows**: trigger → condition branch → action, with **switch / loop / merge** control-flow nodes.
- **Node contract**: trigger, reply, template, LLM, HTTP tool, Dify, n8n, and **history** (context memory) nodes — each with a typed port and validated contract.
- **Per-mechanism workflows**: every trigger mechanism owns its own draft and published version history.
- **Publish & install**: publish an immutable workflow version; install the agent into a DM or group chat with a capability review.
- **Debug trace**: watch the agent's real decision path, variable values and node timings as messages flow.
- **Bot discovery**: search and install community-authored agents.

#### Agent Runtime — Actor-per-Conversation

The workflow engine (`@purrchat/workflow-engine`, TypeScript) compiles each published workflow into a state machine. Every conversation where an agent is installed gets an isolated **actor** with its own context, so state never leaks between chats and crash recovery is built in.

#### Capability Permission Model

An agent can only do what it declares and what the installer grants.

| Capability               | Meaning                                     |
| ------------------------ | ------------------------------------------- |
| `messages:read_trigger`  | Read the message that woke the agent        |
| `messages:read_history`  | Pull prior conversation context             |
| `messages:send`          | Send replies                                |
| `members:read`           | Read conversation members                   |
| `network:external`       | Call LLM / tools / Dify / n8n               |
| `secrets:use`            | Use owner-configured secrets (encrypted)    |

Capabilities are derived from the published workflow, reviewed at install time, and enforced on every action at runtime.

#### External Agents — OneBot Universal WebSocket

PurrChat is also a **host for external agents**. Any program speaking the OneBot protocol can connect over a Universal WebSocket (`/api/bot/v1/ws`), receive message / member / installation events, and call back actions (`send_message`, `get_message_history`, …) with Bearer-credential auth, reliable event delivery (outbox + ACK + resume) and a developer capability matrix.

#### Soft Architecture Design System

- **Living Geometry**: six-level radius system with no hard corners.
- **Breathing Space**: more generous spacing than typical chat tools.
- **Material Honesty**: sage green accent, low-saturation mineral color, blue-toned dark mode.

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
├── Bot Engine        TypeScript · XState actor model · workflow runtime
└── Storage           Go · Cloudflare R2 / MinIO (two-step upload)
```

| Layer      | Technology              | Why                                                                |
| ---------- | ----------------------- | ------------------------------------------------------------------ |
| Frontend   | Vue 3 (Composition API) | Reactive model fits event-driven UI well                           |
| Build      | Vite + Turborepo        | Fast HMR and incremental monorepo builds                           |
| Backend    | Go                      | High-concurrency WebSocket management and single-binary deployment |
| Bot Engine | TypeScript + XState     | Actor-per-conversation, typed node contracts, crash recovery       |
| Database   | PostgreSQL              | JSONB for workflow docs; per-mechanism versioning                  |
| Realtime   | Native WebSocket        | Multi-client identity, heartbeat, and message caching              |
| Storage    | R2 / MinIO              | Presigned upload flow with low server bandwidth                    |
| Desktop    | Tauri 2                 | Native performance with reusable Rust-side capabilities             |
| Mobile     | Tauri 2 Mobile          | Shared frontend with native Android integration                    |

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
| Bot Engine  | http://localhost:3001 |
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
│   │       ├── botengine/       # Bot runtime coordination & installation
│   │       ├── botws/           # OneBot Universal WebSocket transport
│   │       ├── onebot/          # OneBot protocol registry & codec
│   │       ├── handlers/        # HTTP handlers
│   │       └── websocket/       # WebSocket connection management
│   └── storage/                 # Go file storage service
│       └── internal/providers/  # R2 / MinIO providers
├── packages/
│   ├── workflow-types/          # Node / port / document / capability types
│   └── workflow-engine/         # TS runtime: compiler, actor, trace, validator
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

- Create workflow bots and external (OneBot) bots.
- Visual DAG editor: trigger → condition → action with switch / loop / merge.
- Node library: reply, template, LLM, HTTP tool, Dify, n8n, history (memory).
- Per-mechanism draft + published version history with rollback.
- Publish immutable versions; install into DM or group with capability review.
- Real-time debug trace of the agent decision path.
- External agent hosting via OneBot Universal WebSocket (events, actions, ACK/resume).
- Bot discovery and sharing.

### Documentation

| Document                                                 | Description                       |
| -------------------------------------------------------- | --------------------------------- |
| [Design System](docs/DESIGN_SYSTEM.md)                   | Soft Architecture design rules    |
| [Bot Engine Architecture](docs/bot-engine/ARCHITECTURE.md) | Workflow engine & actor model  |
| [Bot App Model](docs/bot-engine/BOT_APP_MODEL.md)        | App / identity / installation ADR |
| [Universal WebSocket](docs/bot-api/UNIVERSAL_WEBSOCKET.md) | OneBot external agent protocol  |
| [Deployment Guide](docs/DEPLOYMENT.md)                   | Docker deployment and Nginx setup |
| [Roadmap](docs/ROADMAP.md)                               | Feature planning and progress     |
| [Commit Convention](docs/COMMIT_CONVENTION.md)           | Git commit message rules          |
| [WebSocket Architecture](docs/WEBSOCKET_ARCHITECTURE.md) | WebSocket event management        |

### Common Commands

```bash
pnpm dev              # Start frontend, backend, bot-engine and storage
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
- [ ] Internationalization.
- [ ] Rich media messages: voice, video, stickers.
- [ ] Desktop enhancements: tray and auto-start.

See [Roadmap](docs/ROADMAP.md) for details.

### Contributing

Contributions are welcome. Please read [Commit Convention](docs/COMMIT_CONVENTION.md) before opening a pull request.

---

## 简体中文

### PurrChat 是什么？

PurrChat 是一个全栈实时聊天平台,围绕一个核心理念构建:**AI 角色应该是被你「创作」出来的,而不是被你「提示」出来的。**

它的核心是 **Bot Studio** —— 一个可视化、事件驱动的编排系统。你用拖拽节点的方式,把一个 agent 的*人格、记忆、决策流和工具*组合成一张工作流图。这个 agent 随后活在你的对话里:它捕捉恰当的时机,回忆上下文,按条件分支,调用 LLM 或外部服务,像一个有连续性的角色一样回应。

PurrChat 采用 **Soft Architecture(柔软建筑)** 设计理念:它不是功能面板的堆砌,而是一个安静、精致、有呼吸感的空间,对话在其中自然流淌,无论对方是人还是 AI。

### 创作一个 Agent,而不是写一段 Prompt

多数聊天工具把 AI 当成一个「文本进、文本出」的函数。PurrChat 把它当成一个**运行时 Actor**:有状态、有记忆、有一张你能看见和编辑的行为图。

| 关注点   | PurrChat 如何建模                                            |
| -------- | ------------------------------------------------------------ |
| **人格** | 通过回复节点、LLM prompt 和人设配置共同塑造                  |
| **记忆** | History 节点把对话上下文拉入每一次决策                        |
| **行为流** | DAG 工作流:触发器 → 条件 → 动作,含 switch / loop / merge |
| **工具** | LLM、HTTP 工具、Dify、n8n 作为外部节点接入                   |
| **能动性** | 每个会话启动一个隔离的 actor,拥有独立上下文                 |
| **信任** | capability 模型声明、授权并强制约束 agent 能做什么           |

### 核心亮点

#### Bot Studio —— 可视化 Agent 编排

DAG 编辑器是核心创作界面。用节点而非代码编排 agent 行为。

- **事件驱动工作流**:触发器 → 条件分支 → 动作,含 **switch / loop / merge** 控制流节点。
- **节点契约**:trigger、reply、template、LLM、HTTP tool、Dify、n8n 和 **history**(上下文记忆)节点,每个节点有类型化端口和校验契约。
- **mechanism 级工作流**:每个触发机制拥有独立的草稿与发布版本历史。
- **发布与安装**:发布不可变工作流版本;将 agent 安装到私聊或群聊,并经过能力审查。
- **调试 Trace**:实时观察 agent 的决策路径、变量取值与节点耗时。
- **Bot 发现**:搜索并安装社区创作的 agent。

#### Agent 运行时 —— 每会话一个 Actor

工作流引擎(`@purrchat/workflow-engine`,TypeScript)将每个已发布工作流编译成状态机。每个安装了 agent 的会话获得一个隔离的 **actor**,拥有独立上下文,状态不会跨会话泄漏,且内建崩溃恢复。

#### Capability 权限模型

agent 只能做它声明且安装者授权的事。

| Capability               | 含义                                |
| ------------------------ | ----------------------------------- |
| `messages:read_trigger`  | 读取唤醒 agent 的消息               |
| `messages:read_history`  | 拉取之前的对话上下文                |
| `messages:send`          | 发送回复                            |
| `members:read`           | 读取会话成员                        |
| `network:external`       | 调用 LLM / 工具 / Dify / n8n        |
| `secrets:use`            | 使用 owner 配置的 secret(加密存储) |

能力由已发布工作流推导,在安装时审查,并在运行时每次动作上强制校验。

#### 外部 Agent —— OneBot Universal WebSocket

PurrChat 同时是**外部 agent 的宿主**。任何讲 OneBot 协议的程序都能通过 Universal WebSocket(`/api/bot/v1/ws`)接入,接收消息/成员/安装事件,并回调 action(`send_message`、`get_message_history`…),带 Bearer 凭证鉴权、可靠事件投递(outbox + ACK + 断线恢复)和开发者能力矩阵。

#### Soft Architecture 设计系统

- **Living Geometry**:6 级统一圆角系统,消灭所有直角。
- **Breathing Space**:比一般聊天应用更慷慨的间距。
- **Material Honesty**:Sage 绿强调色、低饱和矿物感、蓝调深灰暗色模式。

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
├── Bot Engine        TypeScript · XState actor 模型 · 工作流运行时
└── Storage           Go · Cloudflare R2 / MinIO(两阶段上传)
```

| 层面       | 技术选型                | 理由                                      |
| ---------- | ----------------------- | ----------------------------------------- |
| 前端框架   | Vue 3 (Composition API) | 响应式系统与事件驱动模型天然契合          |
| 构建工具   | Vite + Turborepo        | 极速 HMR,Monorepo 增量构建               |
| 后端语言   | Go                      | 高并发 WebSocket 连接管理,单二进制部署   |
| Bot 引擎   | TypeScript + XState     | 每会话一个 actor,类型化节点契约,崩溃恢复 |
| 数据库     | PostgreSQL              | JSONB 存工作流文档;mechanism 级版本管理  |
| 实时通信   | 原生 WebSocket          | 多端身份识别、连接心跳、消息缓存          |
| 文件存储   | R2 / MinIO              | 预签名 URL 两阶段上传,降低服务端带宽压力 |
| 桌面端     | Tauri 2                 | 原生性能,Rust 侧能力可复用               |
| 移动端     | Tauri 2 Mobile          | 共享前端代码,接入 Android 原生能力       |

### 快速开始

#### 环境要求

- Node.js >= 20
- pnpm >= 9
- Go >= 1.24
- Docker & Docker Compose(用于 PostgreSQL)

Android 构建可选要求:

- JDK 17+
- Android SDK API 35+
- Android NDK r27+
- Rust Android targets:`rustup target add aarch64-linux-android x86_64-linux-android`

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
| Bot Engine  | http://localhost:3001 |
| Storage     | http://localhost:8081 |

#### 环境配置

复制 `.env.example` 到 `.env`,再参考各服务示例配置:

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
│   │       ├── botengine/       # Bot 运行时协调与安装
│   │       ├── botws/           # OneBot Universal WebSocket 传输层
│   │       ├── onebot/          # OneBot 协议注册表与编解码
│   │       ├── handlers/        # HTTP 处理器
│   │       └── websocket/       # WebSocket 连接管理
│   └── storage/                 # Go 文件存储服务
│       └── internal/providers/  # R2 / MinIO 提供者
├── packages/
│   ├── workflow-types/          # 节点 / 端口 / 文档 / 能力类型
│   └── workflow-engine/         # TS 运行时:编译器、actor、trace、校验器
├── docs/                        # 项目文档
├── docker-compose.yml
├── turbo.json
└── Makefile
```

### 功能概览

#### 聊天核心

- 用户注册/登录(JWT 认证)。
- 好友系统(请求、接受、拒绝、黑名单)。
- 一对一实时聊天(WebSocket)。
- 群聊创建、管理与成员角色。
- 头像和背景图上传(两阶段预签名)。
- 消息缓存与增量加载(IndexedDB)。
- Web/Tauri 多客户端同时在线。

#### Bot Studio

- 创建工作流 Bot 与外部(OneBot)Bot。
- DAG 可视化编辑器:触发器 → 条件 → 动作,含 switch / loop / merge。
- 节点库:reply、template、LLM、HTTP tool、Dify、n8n、history(记忆)。
- mechanism 级草稿 + 发布版本历史,支持回滚。
- 发布不可变版本;安装到私聊或群聊并经能力审查。
- 实时调试 Trace,追踪 agent 决策路径。
- 通过 OneBot Universal WebSocket 托管外部 agent(事件、action、ACK/断线恢复)。
- Bot 发现与分享。

### 文档

| 文档                                                   | 说明                          |
| ------------------------------------------------------ | ----------------------------- |
| [设计系统](docs/DESIGN_SYSTEM.md)                      | Soft Architecture 设计规范    |
| [Bot 引擎架构](docs/bot-engine/ARCHITECTURE.md)        | 工作流引擎与 actor 模型       |
| [Bot App 模型](docs/bot-engine/BOT_APP_MODEL.md)       | App / 身份 / 安装 ADR         |
| [Universal WebSocket](docs/bot-api/UNIVERSAL_WEBSOCKET.md) | OneBot 外部 agent 协议    |
| [部署指南](docs/DEPLOYMENT.md)                         | Docker 部署、Nginx 配置       |
| [开发路线图](docs/ROADMAP.md)                          | 功能规划与开发进度            |
| [提交规范](docs/COMMIT_CONVENTION.md)                  | Git Commit Message 规范       |
| [WebSocket 架构](docs/WEBSOCKET_ARCHITECTURE.md)       | WebSocket 事件管理            |

### 常用命令

```bash
pnpm dev              # 启动开发模式:前端 + 后端 + bot-engine + 存储服务
pnpm build            # 构建生产版本
pnpm test             # 运行测试
pnpm lint             # 代码检查
make docker-up        # Docker 一键启动全栈
make docker-down      # Docker 停止
```

#### Android 构建

```bash
make android-dev            # 开发模式,需连接设备或模拟器
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
- [ ] 国际化(i18n)。
- [ ] 消息多媒体(语音、视频、表情包)。
- [ ] 桌面端增强(系统托盘、开机自启)。

详见 [开发路线图](docs/ROADMAP.md)。

### 贡献

欢迎贡献。请阅读 [提交规范](docs/COMMIT_CONVENTION.md) 后提交 Pull Request。

---

<div align="center">

**Built with care by [Caspian-Lin](https://github.com/Caspian-Lin)**

</div>
