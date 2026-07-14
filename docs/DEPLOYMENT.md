# PurrChat 部署指南

## 目录

- [项目概述](#项目概述)
- [前置要求](#前置要求)
- [本地开发](#本地开发)
- [Docker 部署](#docker-部署)
- [生产环境部署](#生产环境部署)
- [Nginx 反向代理](#nginx-反向代理)
- [多客户端构建](#多客户端构建)
- [数据库迁移](#数据库迁移)
- [CI/CD](#cicd)
- [监控与维护](#监控与维护)
- [故障排查](#故障排查)

## 项目概述

PurrChat 使用 monorepo 架构，由以下服务组成：

| 服务     | 技术栈                      | 端口                   |
| -------- | --------------------------- | ---------------------- |
| Frontend | Vue 3 + Vite + Tailwind CSS | 5173 (dev) / 80 (prod) |
| Backend  | Go + Gin + PostgreSQL       | 8080                   |
| Storage  | Go + MinIO / Cloudflare R2  | 8081                   |

## 前置要求

- Node.js >= 18, pnpm >= 9
- Go >= 1.24
- Docker & Docker Compose (可选)

## 本地开发

```bash
# 安装依赖
pnpm install

# 启动所有服务 (推荐通过 Docker Compose 启动数据库)
docker compose up -d postgres
pnpm dev
```

服务地址:

- 前端: http://localhost:5173
- 后端 API: http://localhost:8080
- 存储服务: http://localhost:8081

### 环境变量

复制 `.env.example` 到 `.env` 并配置:

```env
# 后端
PORT=8080
GIN_MODE=debug
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=purrchat
JWT_SECRET=your_jwt_secret_key_here
JWT_EXPIRATION=24h

# 存储服务
STORAGE_PROVIDER=minio
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin

# 前端
VITE_API_BASE_URL=http://localhost:8080
```

## Docker 部署

```bash
# 一键启动所有服务
docker compose up -d

# 查看日志
docker compose logs -f

# 停止
docker compose down
```

## 生产环境部署

### 服务器准备

```bash
# 安装 Docker
curl -fsSL https://get.docker.com | sudo sh

# 克隆项目
git clone <repo-url> && cd PurrChat

# 配置环境变量
cp .env.example .env
# 编辑 .env，设置强密码和 JWT_SECRET
```

### 部署步骤

```bash
# 构建并启动
docker compose up -d --build

# 验证
curl http://localhost:8080/health
```

### 推荐服务器配置

- CPU: 2 核以上
- 内存: 4GB 以上
- 存储: 20GB 以上 SSD
- 系统: Ubuntu 22.04+

## Nginx 反向代理

```nginx
server {
    listen 80;
    server_name your-server.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name your-server.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    # 前端静态文件
    location / {
        root /var/www/purrchat;
        try_files $uri $uri/ /index.html;
    }

    # 后端 API
    location /api/ {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # WebSocket
    location /api/ws {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_read_timeout 86400;
    }

    # 存储服务 API
    location /storage/api/ {
        proxy_pass http://localhost:8081/api/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## 多客户端构建

### 网页端

```bash
cd apps/frontend
VITE_API_BASE_URL=/ pnpm run build
# 将 dist/ 部署到 Nginx
```

### Tauri 桌面端

```bash
cd apps/frontend
# 编辑 .env.tauri 设置服务器地址
pnpm tauri:build:win64
```

### 环境变量说明

| 变量                | 说明       | Web            | Tauri                     |
| ------------------- | ---------- | -------------- | ------------------------- |
| `VITE_API_BASE_URL` | API 地址   | `/` (相对路径) | `https://your-server.com` |
| `VITE_APP_ENV`      | 环境标识   | `production`   | `production`              |
| `VITE_APP_CLIENT`   | 客户端类型 | `web`          | `tauri`                   |

## 数据库迁移

迁移分别位于 `apps/backend/migrations/` 与 `apps/storage/migrations/`。Docker Compose 会先运行对应的 one-shot migrator，只有成功后才启动应用服务。手动执行、已有数据库 baseline、回滚边界和新迁移编号规则见 [数据库迁移指南](MIGRATIONS.md)。

## CI/CD

### GitHub Actions

配置 Secrets:

- `DOCKER_USERNAME`, `DOCKER_PASSWORD`
- `DEV_SERVER_HOST`, `DEV_SERVER_USER`, `DEV_SERVER_SSH_KEY`
- `PROD_SERVER_HOST`, `PROD_SERVER_USER`, `PROD_SERVER_SSH_KEY`

### GitLab CI

配置 Variables (同上，加上 `CI_REGISTRY_USER`/`CI_REGISTRY_PASSWORD`)

## 监控与维护

```bash
# 查看日志
docker compose logs -f backend

# 数据库备份
docker compose exec postgres pg_dump -U postgres purrchat > backup_$(date +%Y%m%d).sql

# 数据库恢复
docker compose exec -T postgres psql -U postgres purrchat < backup.sql
```

## 故障排查

| 问题           | 排查命令                                                                  |
| -------------- | ------------------------------------------------------------------------- |
| 容器无法启动   | `docker compose logs backend`                                             |
| 数据库连接失败 | `docker compose exec postgres psql -U postgres -d purrchat -c "SELECT 1"` |
| 端口冲突       | `sudo lsof -i :8080`                                                      |
| WebSocket 断连 | 检查 Nginx `proxy_read_timeout` 和 `Upgrade` 头配置                       |
