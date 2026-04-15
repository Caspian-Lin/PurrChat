# PurrChat

<div align="center">

**PurrChat - 现代化聊天应用**

[![CI/CD](https://github.com/Caspian-Lin/PurrChat/workflows/CI%2FCD%20Pipeline/badge.svg)](https://github.com/Caspian-Lin/PurrChat/actions)

</div>

## 项目概述

PurrChat 是一个基于 Turborepo monorepo 的全栈实时聊天应用。

| 服务 | 技术栈 | 说明 |
|------|--------|------|
| Frontend | Vue 3 + Vite + Tailwind CSS + Tauri | Web 端 + 桌面端 |
| Backend | Go + Gin + PostgreSQL | REST API + WebSocket |
| Storage | Go + MinIO / Cloudflare R2 | 文件上传/下载 |

## 快速开始

```bash
# 安装依赖
pnpm install

# 启动数据库
docker compose up -d postgres

# 启动开发服务器
pnpm dev
```

- 前端: http://localhost:5173
- 后端: http://localhost:8080

## 项目结构

```
PurrChat/
├── apps/
│   ├── frontend/           # Vue 3 前端 (Web + Tauri)
│   ├── backend/            # Go 后端 API
│   └── storage/            # Go 文件存储服务
├── docs/                   # 项目文档
├── docker-compose.yml
├── turbo.json
├── Makefile
└── CLAUDE.md               # AI 辅助开发指令
```

## 常用命令

```bash
pnpm dev              # 启动开发模式
pnpm build            # 构建生产版本
pnpm test             # 运行测试
pnpm lint             # 代码检查
make docker-up        # Docker 一键启动
make docker-down      # Docker 停止
```

## 环境变量

复制 `.env.example` 到 `.env`，主要配置项:

```env
# 数据库
DB_HOST=localhost
DB_PASSWORD=your_password

# JWT
JWT_SECRET=your_jwt_secret_key_here

# 前端
VITE_API_BASE_URL=http://localhost:8080
```

## 文档

| 文档 | 说明 |
|------|------|
| [部署指南](docs/DEPLOYMENT.md) | Docker 部署、Nginx 配置、多客户端构建 |
| [安全修复列表](docs/SECURITY_ISSUES.md) | 已发现的安全漏洞及修复进度 |
| [开发路线图](docs/ROADMAP.md) | 待实现功能列表及开发计划 |
| [提交规范](docs/COMMIT_CONVENTION.md) | Git commit message 规范 |
| [WebSocket 架构](docs/WEBSOCKET_ARCHITECTURE.md) | 前端 WebSocket 事件管理架构 |
| [WebSocket 调试](docs/WEBSOCKET_DEBUG_GUIDE.md) | WebSocket 问题排查指南 |

## 已实现功能

- 用户注册/登录 (JWT 认证)
- 好友系统 (请求/接受/拒绝)
- 一对一实时聊天 (WebSocket)
- 群聊 (创建群聊/添加成员)
- 头像和背景图上传 (R2/MinIO)
- 消息缓存 (IndexedDB)
- 多客户端支持 (Web/Tauri)

## 提交规范

项目使用 Conventional Commits:

```
<type>(<scope>): <描述>
```

类型: `feat` / `fix` / `refactor` / `docs` / `style` / `perf` / `test` / `chore` / `ci`

详见 [提交规范](docs/COMMIT_CONVENTION.md)。

## 贡献指南

1. Fork 并创建特性分支
2. 提交更改 (遵循提交规范)
3. 推送并开启 Pull Request

---

<div align="center">

**Made with PurrChat Team**

</div>
