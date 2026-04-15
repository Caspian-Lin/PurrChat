# PurrChat

<div align="center">

**PurrChat - 现代化聊天应用**

[![CI/CD](https://github.com/Caspian-Lin/PurrChat/workflows/CI%2FCD%20Pipeline/badge.svg)](https://github.com/Caspian-Lin/PurrChat/actions)

一个基于 Turborepo 的全栈聊天应用，包含前端和后端服务。

</div>

## 项目概述

PurrChat 是一个现代化的聊天应用，采用前后端分离架构，使用 Turborepo 进行 monorepo 管理。

- **前端**: Vue 3 + Vite + Tauri
- **后端**: Go + Gin + PostgreSQL
- **容器化**: Docker + Docker Compose
- **CI/CD**: GitHub Actions

## 已知问题和路线图

### 当前问题
- alert 和提示使用不统一，有些使用了浏览器的 alert
- 重复 alert
- 缺乏多用户测试，不知道 ws 能否成功
- friendslist 缺滚动条
- 添加群聊按钮因为色彩显得比较大，需要添加补偿
- 前端部分功能日志过多

### 预计实现功能
- 上传头像、发送图片
- 服务器端并发限制、分流、排队处理等
- 桌面端、存储重构
- 接入 beh

### 文件存储安全路线图

当前文件存储基于 Cloudflare R2（S3 兼容），按资源类型采用不同访问策略：

| 资源类型 | 当前方案 | 目标方案 |
|---------|---------|---------|
| 头像 | 公开 URL（R2.dev 子域名） | 同左 |
| 聊天图片 | 预签名 URL | 短有效期（30s）预签名 URL |
| 聊天文件 | 预签名 URL | 短有效期（30s）预签名 URL |

#### 阶段一：公开资源（头像）— 当前阶段
- R2 存储桶开启公开访问，头像通过公开 URL 直接访问
- 优势：零后端负载、访问速度快、无需鉴权
- 适用场景：所有用户可见的公开资源

#### 阶段二：受保护资源（聊天文件）— 待实现
- 使用短有效期预签名 URL（30-60 秒）
- 每次前端请求文件时，后端验证用户身份和访问权限后生成签名 URL
- 签名 URL 过期后自动失效，降低泄露风险

#### 阶段三：细粒度权限控制（可选进阶）
- 通过 Cloudflare Workers 在边缘节点执行鉴权逻辑
- 实现聊天级别的访问控制（仅聊天双方可查看文件）
- 后端零负载，鉴权在 CDN 边缘完成

## 项目结构

```
PurrChat/
├── apps/
│   ├── frontend/           # 前端应用 (Vue 3)
│   │   ├── src/
│   │   ├── public/
│   │   ├── src-tau/        # Tauri 源码
│   │   ├── Dockerfile
│   │   └── package.json
│   ├── backend/            # 后端应用 (Go)
│   │   ├── cmd/
│   │   ├── internal/
│   │   ├── migrations/
│   │   ├── tests/
│   │   ├── Dockerfile
│   │   ├── go.mod
│   │   └── package.json
│   └── storage/            # 存储服务 (Go)
│       ├── cmd/
│       ├── internal/
│       ├── migrations/
│       ├── Dockerfile
│       ├── go.mod
│       └── package.json
├── .github/
│   └── workflows/
│       └── ci.yml          # GitHub Actions CI/CD
├── .gitlab-ci.yml          # GitLab CI/CD
├── docker-compose.yml      # Docker Compose 配置
├── turbo.json              # Turborepo 配置
├── package.json            # 根 package.json
├── Makefile                # Make 命令
└── README.md
```

## 快速开始

### 前置要求

- Node.js >= 18
- pnpm >= 9
- Go >= 1.25
- Docker (可选，用于容器化部署)

### 安装依赖

```bash
# 安装所有依赖
pnpm install
```

### 启动开发环境

```bash
# 启动所有应用的开发服务器
pnpm dev

# 或使用 Makefile
make dev
```

这将启动：
- 前端开发服务器: http://localhost:5173
- 后端 API 服务器: http://localhost:8080

### 构建生产版本

```bash
# 构建所有应用
pnpm build

# 或使用 Makefile
make build
```

### 运行测试

```bash
# 运行所有测试
pnpm test

# 或使用 Makefile
make test
```

### Docker 部署

```bash
# 构建并启动所有服务
make docker-up

# 查看日志
make docker-logs

# 停止服务
make docker-down
```

## 开发指南

### 环境变量配置

复制 `.env.example` 到 `.env` 并配置：

```bash
cp .env.example .env
```

主要配置项：

```env
# 服务器配置
PORT=8080
GIN_MODE=release

# 数据库配置
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=purrchat

# JWT配置
JWT_SECRET=your_jwt_secret_key_here_change_in_production
JWT_EXPIRATION=24h

# 前端配置
FRONTEND_PORT=80
VITE_API_BASE_URL=http://localhost:8080
```

### 常用命令

```bash
# 开发
pnpm dev              # 启动开发模式
pnpm build            # 构建生产版本
pnpm test             # 运行测试
pnpm lint             # 代码检查
pnpm format           # 格式化代码
pnpm type-check       # 类型检查

# Docker
make docker-up        # 启动 Docker 容器
make docker-down      # 停止 Docker 容器
make docker-logs      # 查看 Docker 日志

# 清理
pnpm clean            # 清理构建产物和依赖
make clean            # 同上
```

### 单独运行前端

```bash
cd apps/frontend
pnpm dev
```

### 单独运行后端

```bash
cd apps/backend
go run cmd/server/main.go
```

## Turborepo 配置说明

### turbo.json 配置

[`turbo.json`](turbo.json:1) 是 Turborepo 的核心配置文件，定义了任务的行为和依赖关系。

#### 配置结构

```json
{
  "$schema": "https://turborepo.dev/schema.json",
  "ui": "tui",
  "globalEnv": ["环境变量列表"],
  "tasks": {
    "任务名": {
      "dependsOn": ["依赖任务"],
      "inputs": ["输入文件"],
      "outputs": ["输出文件"],
      "cache": "缓存配置",
      "persistent": "持久化配置"
    }
  }
}
```

#### 任务说明

| 任务 | 说明 | 依赖 |
|------|------|------|
| `build` | 构建应用 | `^build` (所有依赖的 build) |
| `dev` | 开发模式 | `^build` |
| `lint` | 代码检查 | `^lint` |
| `test` | 运行测试 | `^build` |
| `type-check` | 类型检查 | `^type-check` |
| `clean` | 清理构建产物 | 无 |

#### 关键配置项

- **`dependsOn`**: 定义任务依赖关系，`^` 表示所有依赖包
- **`inputs`**: 指定任务输入文件，用于缓存失效判断
- **`outputs`**: 指定任务输出文件，用于缓存
- **`cache`**: 设置是否缓存，`false` 表示禁用缓存
- **`persistent`**: 设置是否持久化运行，用于开发服务器

### package.json 配置

根目录的 [`package.json`](package.json:1) 定义了 monorepo 的脚本和工作区。

#### 工作区配置

```json
{
  "workspaces": [
    "apps/*",
    "packages/*"
  ]
}
```

这告诉 Turborepo 在 `apps/` 和 `packages/` 目录中查找工作区。

#### 脚本命令

所有脚本都使用 `turbo run` 来执行：

```json
{
  "scripts": {
    "build": "turbo run build",
    "dev": "turbo run dev",
    "lint": "turbo run lint",
    "test": "turbo run test"
  }
}
```

### 缓存机制

Turborepo 使用智能缓存来加速构建：

1. **任务缓存**: 每个任务的输出都会被缓存
2. **远程缓存**: 可配置远程缓存（如 Vercel、自建缓存服务器）
3. **缓存失效**: 当输入文件变化时，缓存自动失效

启用远程缓存：

```bash
turbo login
turbo link
```

## CI/CD 指南

### GitLab CI

项目包含完整的 GitLab CI 配置 ([`.gitlab-ci.yml`](.gitlab-ci.yml:1))。

#### 流水线阶段

1. **lint**: 代码质量检查
   - 前端 ESLint 检查
   - 后端 golangci-lint 检查

2. **test**: 运行测试
   - 前端单元测试 + 覆盖率
   - 后端集成测试 + 覆盖率

3. **build**: 构建 Docker 镜像
   - 前端镜像构建并推送到镜像仓库
   - 后端镜像构建并推送到镜像仓库

4. **deploy**: 部署到环境
   - 开发环境部署 (develop 分支)
   - 生产环境部署 (main 分支)

#### 配置环境变量

在 GitLab 项目设置中配置以下变量：

| 变量名 | 说明 | 示例 |
|--------|------|------|
| `CI_REGISTRY` | 镜像仓库地址 | `registry.gitlab.com` |
| `CI_REGISTRY_USER` | 镜像仓库用户名 | `gitlab-ci-token` |
| `CI_REGISTRY_PASSWORD` | 镜像仓库密码 | `${CI_JOB_TOKEN}` |
| `DEV_SERVER_HOST` | 开发服务器地址 | `dev.example.com` |
| `DEV_SERVER_USER` | 开发服务器用户 | `deploy` |
| `DEV_SSH_PRIVATE_KEY` | 开发服务器 SSH 密钥 | - |
| `PROD_SERVER_HOST` | 生产服务器地址 | `prod.example.com` |
| `PROD_SERVER_USER` | 生产服务器用户 | `deploy` |
| `PROD_SSH_PRIVATE_KEY` | 生产服务器 SSH 密钥 | - |

#### 触发部署

部署任务设置为手动触发：

```bash
# 在 GitLab UI 中点击 "Play" 按钮触发部署
```

### GitHub Actions

项目也包含 GitHub Actions 配置 ([`.github/workflows/ci.yml`](.github/workflows/ci.yml:1))。

#### 配置 Secrets

在 GitHub 仓库设置中配置以下 Secrets：

| Secret 名称 | 说明 |
|-------------|------|
| `DEV_SERVER_HOST` | 开发服务器地址 |
| `DEV_SERVER_USER` | 开发服务器用户 |
| `DEV_SSH_PRIVATE_KEY` | 开发服务器 SSH 密钥 |
| `PROD_SERVER_HOST` | 生产服务器地址 |
| `PROD_SERVER_USER` | 生产服务器用户 |
| `PROD_SSH_PRIVATE_KEY` | 生产服务器 SSH 密钥 |

#### 工作流触发

- **Push**: 推送到 `main` 或 `develop` 分支
- **Pull Request**: 针对 `main` 或 `develop` 分支的 PR

### 本地 CI 测试

在提交前本地运行 CI 检查：

```bash
# 运行所有 CI 检查
pnpm lint
pnpm test
pnpm build
```

## 提交规范

项目使用 Conventional Commits 规范进行提交。

### 提交格式

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Type 类型

| Type | 说明 |
|------|------|
| `feat` | 新功能 |
| `fix` | 修复 Bug |
| `docs` | 文档更新 |
| `style` | 代码格式调整（不影响功能） |
| `refactor` | 重构（既不是新功能也不是修复） |
| `perf` | 性能优化 |
| `test` | 测试相关 |
| `chore` | 构建过程或辅助工具的变动 |
| `ci` | CI/CD 配置变更 |
| `revert` | 回退提交 |

### Scope 范围

| Scope | 说明 |
|-------|------|
| `frontend` | 前端相关 |
| `backend` | 后端相关 |
| `ci` | CI/CD 相关 |
| `docker` | Docker 相关 |
| `docs` | 文档相关 |

### 示例

```bash
# 新功能
git commit -m "feat(frontend): add user profile page"

# 修复 Bug
git commit -m "fix(backend): resolve authentication token expiration issue"

# 文档更新
git commit -m "docs: update deployment guide"

# 重构
git commit -m "refactor(backend): simplify user repository logic"

# CI/CD 变更
git commit -m "ci: add GitHub Actions workflow"

# 破坏性变更
git commit -m "feat(frontend)!: redesign authentication flow

BREAKING CHANGE: The authentication API has been completely redesigned.
Please update your client applications accordingly."
```

### 提交前检查

项目配置了 Husky 和 lint-staged，提交前自动运行：

```bash
# 安装 Git hooks
pnpm install

# 提交时自动运行
pnpm commit
```

## 部署

### Docker 部署

使用 Docker Compose 一键部署：

```bash
# 启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

### 生产环境部署

1. **准备环境变量**

```bash
cp .env.example .env
# 编辑 .env 文件，配置生产环境参数
```

2. **构建镜像**

```bash
docker-compose build
```

3. **启动服务**

```bash
docker-compose up -d
```

4. **健康检查**

```bash
# 检查后端健康状态
curl https://api.purrchat.com/health

# 检查前端
curl https://purrchat.com/
```

### 服务器配置

推荐的服务器配置：

- **CPU**: 2 核心以上
- **内存**: 4GB 以上
- **存储**: 20GB 以上 SSD
- **系统**: Ubuntu 20.04+ / CentOS 8+

### 数据库备份

定期备份数据库：

```bash
# 备份数据库
docker-compose exec postgres pg_dump -U postgres purrchat > backup_$(date +%Y%m%d_%H%M%S).sql

# 恢复数据库
docker-compose exec -T postgres psql -U postgres purrchat < backup_20240101_120000.sql
```

## 贡献指南

欢迎贡献代码！请遵循以下步骤：

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'feat: add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

### 代码规范

- 前端遵循 ESLint 和 Prettier 配置
- 后端遵循 golangci-lint 配置
- 提交信息遵循 Conventional Commits 规范

### Pull Request 要求

- 通过所有 CI 检查
- 代码覆盖率不降低
- 更新相关文档
- 添加必要的测试

---

<div align="center">

**Made with ❤️ by PurrChat Team**

</div>
