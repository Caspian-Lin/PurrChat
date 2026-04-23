<div align="center">

<img src="docs/logo.svg" alt="PurrChat" width="120" height="120">

# PurrChat

**Intimate &middot; Refined &middot; Alive**

探索人与 AI 的情感连接和灵感边界

[![CI](https://github.com/Caspian-Lin/PurrChat/actions/workflows/dev-to-beta.yml/badge.svg)](https://github.com/Caspian-Lin/PurrChat/actions)
[![License: CC BY-NC-SA 4.0](https://img.shields.io/badge/License-CC%20BY--NC--SA%204.0-lightgrey.svg)](https://creativecommons.org/licenses/by-nc-sa/4.0/)

[English](#) &middot; 简体中文

</div>

---

## PurrChat 是什么？

PurrChat 是一个全栈实时聊天平台，内置 **Bot Studio** — 一个可视化的事件驱动 Bot 编排系统，让你可以在聊天中构建具有复杂行为的 AI 角色。

不同于传统的聊天应用，PurrChat 引入了 **Soft Architecture（柔软建筑）** 设计理念：它不是一个工具，而是一个空间 — 一个安静、精致、有呼吸感的空间，对话在其中自然流淌，无论对方是人还是 AI。

<!-- 在这里添加项目截图：主聊天界面 -->
<!-- ![主界面](docs/screenshots/chat.png "PurrChat 主聊天界面") -->

<!-- 在这里添加项目截图：Bot Studio 界面 -->
<!-- ![Bot Studio](docs/screenshots/bot-studio.png "Bot Studio 事件链编辑器") -->

<!-- 在这里添加项目截图：Bot 调试面板 -->
<!-- ![调试面板](docs/screenshots/debug.png "Bot 调试面板") -->

---

## 核心亮点

### Bot Studio — 可视化事件链引擎

PurrChat 的核心创新。通过 **DAG（有向无环图）编辑器**，你可以用拖拽的方式编排 Bot 的行为逻辑，无需编写代码。

- **事件驱动架构**：触发器 (Trigger) → 条件分支 (If/Else) → 动作 (Action)，构建复杂对话流
- **内置动作库**：回复消息、延迟发送、修改群名片、用户标签匹配等
- **Python 沙箱**：支持在事件链中嵌入 Python 脚本，实现自定义逻辑
- **实时调试面板**：逐步追踪事件流执行过程，实时查看变量和状态
- **Bot 发现市场**：搜索和部署社区创建的 Bot

### Soft Architecture 设计系统

PurrChat 的 UI 不是"功能面板的堆砌"，而是经过完整设计系统规范的空间体验：

- **Living Geometry** — 6 级统一圆角系统，消灭所有直角
- **Breathing Space** — 比一般聊天应用更慷慨的间距
- **Material Honesty** — Sage 绿强调色，低饱和矿物感，暗色模式采用蓝调深灰

### 多端架构

| 客户端 | 技术 | 状态 |
|--------|------|------|
| Web | Vue 3 + Vite + Tailwind CSS | 已支持 |
| Desktop | Tauri (Rust) | 已支持 |
| Mobile | Capacitor | 规划中 |

---

## 技术栈

```
Monorepo (Turborepo + pnpm)
├── Frontend          Vue 3 · Vite · Tailwind CSS · Pinia · IndexedDB
├── Backend           Go · Gin · PostgreSQL · WebSocket
└── Storage           Go · Cloudflare R2 / MinIO (两阶段上传)
```

| 层面 | 技术选型 | 理由 |
|------|---------|------|
| 前端框架 | Vue 3 (Composition API) | 响应式系统与事件驱动模型天然契合 |
| 构建工具 | Vite + Turborepo | 极速 HMR，Monorepo 增量构建 |
| 后端语言 | Go | 高并发 WebSocket 连接管理，单二进制部署 |
| 数据库 | PostgreSQL | JSONB 支持灵活的 Bot 配置存储 |
| 实时通信 | 原生 WebSocket | 多端身份识别，连接心跳，消息缓存 |
| 文件存储 | R2 / MinIO | 预签名 URL 两阶段上传，零服务器带宽消耗 |
| 桌面端 | Tauri | 原生性能，WASM 前端复用 |

---

## 快速开始

### 环境要求

- Node.js >= 20
- pnpm >= 9
- Go >= 1.24
- Docker & Docker Compose（用于数据库）

### 启动

```bash
# 克隆仓库
git clone https://github.com/Caspian-Lin/PurrChat.git
cd PurrChat

# 安装前端依赖
pnpm install

# 启动 PostgreSQL
docker compose up -d postgres

# 启动开发服务（前端 + 后端 + 存储服务）
pnpm dev
```

| 服务 | 地址 |
|------|------|
| Frontend | http://localhost:5173 |
| Backend API | http://localhost:8080 |
| Storage | http://localhost:8081 |

### 环境配置

复制 `.env.example` 到 `.env`，参考各服务的示例配置文件：

- [`apps/backend/.env.example`](apps/backend/.env.example) — 后端 API 配置
- [`apps/storage/.env.example`](apps/storage/.env.example) — 存储服务配置
- [`apps/frontend/.env.example`](apps/frontend/.env.example) — 前端配置

---

## 项目结构

```
PurrChat/
├── apps/
│   ├── frontend/               # Vue 3 前端 (Web + Tauri)
│   │   └── src/
│   │       ├── components/     # UI 组件
│   │       │   └── home/panel/bots/   # Bot Studio 组件
│   │       ├── stores/         # Pinia 状态管理
│   │       └── utils/          # 工具函数 (事件流引擎等)
│   ├── backend/                # Go 后端 API
│   │   └── internal/
│   │       ├── botengine/      # Bot 事件链引擎
│   │       ├── handlers/       # HTTP 处理器
│   │       └── websocket/      # WebSocket 连接管理
│   └── storage/                # Go 文件存储服务
│       └── internal/providers/ # R2 / MinIO 提供者
├── docs/                       # 项目文档
├── docker-compose.yml
├── turbo.json
└── Makefile
```

---

## 功能概览

### 聊天核心
- 用户注册/登录 (JWT 认证)
- 好友系统 (请求/接受/拒绝/黑名单)
- 一对一实时聊天 (WebSocket)
- 群聊 (创建/管理/成员角色)
- 头像和背景图上传 (两阶段预签名)
- 消息缓存与增量加载 (IndexedDB)
- 多客户端同时在线 (Web/Tauri)

### Bot Studio
- Bot CRUD — 创建、编辑、启用/禁用 Bot
- 部署到会话 — 一键将 Bot 部署到私聊或群聊
- 触发与回复系统 — 关键词/正则/用户标签触发
- 特殊模式 (Agent) — 事件链引擎 + Python 沙箱
- DAG 可视化编辑器 — 拖拽编排事件流
- 调试面板 — 实时追踪事件链执行
- Bot 发现与分享 — 搜索、部署社区 Bot

---

## 文档

| 文档 | 说明 |
|------|------|
| [设计系统](docs/DESIGN_SYSTEM.md) | Soft Architecture 设计规范 |
| [部署指南](docs/DEPLOYMENT.md) | Docker 部署、Nginx 配置 |
| [开发路线图](docs/ROADMAP.md) | 功能规划与开发进度 |
| [提交规范](docs/COMMIT_CONVENTION.md) | Git Commit Message 规范 |
| [WebSocket 架构](docs/WEBSOCKET_ARCHITECTURE.md) | WebSocket 事件管理 |
| [WebSocket 调试](docs/WEBSOCKET_DEBUG_GUIDE.md) | WebSocket 问题排查 |

---

## 常用命令

```bash
pnpm dev              # 启动开发模式 (前端 + 后端 + 存储服务)
pnpm build            # 构建生产版本
pnpm test             # 运行测试
pnpm lint             # 代码检查
make docker-up        # Docker 一键启动全栈
make docker-down      # Docker 停止
```

---

## 路线图

- [ ] 输入状态广播 / 消息撤回 / 已读回执
- [ ] 离线消息推送 / 文件消息完整支持
- [ ] 外部 Bot 接入 (OneBot 11 协议)
- [ ] 国际化 (i18n)
- [ ] 消息多媒体 (语音/视频/表情包)
- [ ] 桌面端增强 (系统托盘/开机自启)

详见 [开发路线图](docs/ROADMAP.md)。

---

## 贡献

欢迎贡献！请阅读 [提交规范](docs/COMMIT_CONVENTION.md) 后提交 Pull Request。

本项目采用 [CC BY-NC-SA 4.0](LICENSE) 许可证 — 非商业用途，转载请注明出处。

---

<div align="center">

**Built with care by [Caspian-Lin](https://github.com/Caspian-Lin)**

</div>
