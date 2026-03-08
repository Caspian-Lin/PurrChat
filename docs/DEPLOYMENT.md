# PurrChat Server CI/CD 部署指南

本文档提供了 PurrChat 后端服务器的完整 CI/CD 部署步骤和配置说明。

## 目录

1. [概述](#概述)
2. [前置要求](#前置要求)
3. [本地开发环境搭建](#本地开发环境搭建)
4. [Docker 部署](#docker-部署)
5. [GitHub Actions CI/CD](#github-actions-cicd)
6. [GitLab CI/CD](#gitlab-cicd)
7. [生产环境部署](#生产环境部署)
8. [监控和维护](#监控和维护)
9. [故障排查](#故障排查)

## 概述

PurrChat 后端服务器使用 Go 语言开发，采用以下技术栈：

- **Web 框架**: Gin
- **数据库**: PostgreSQL 15
- **容器化**: Docker & Docker Compose
- **CI/CD**: GitHub Actions / GitLab CI
- **测试框架**: Go Testing + Testify

### CI/CD 流程

```
代码提交 → 代码检查 → 单元测试 → 构建镜像 → 部署到开发/生产环境
```

## 前置要求

### 本地开发环境

- Go 1.24 或更高版本
- Docker 20.10 或更高版本
- Docker Compose 2.0 或更高版本
- PostgreSQL 15 (可选，可使用 Docker)
- Git

### 服务器环境

- Linux 服务器 (Ubuntu 24.04+ / CentOS 8+ / Alpine Linux)
- Docker 20.10+
- Docker Compose 2.0+
- 至少 2GB RAM
- 至少 10GB 磁盘空间

## 本地开发环境搭建

### 1. 克隆项目

```bash
git clone <repository-url>
cd purr-chat-server
```

### 2. 配置环境变量

复制 `.env.example` 文件并创建 `.env` 文件：

```bash
cp .env.example .env
```

编辑 `.env` 文件，配置以下变量：

```env
# 服务器配置
PORT=8080
GIN_MODE=debug

# 数据库配置
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=purrchat

# JWT配置
JWT_SECRET=your_jwt_secret_key_here_change_in_production
JWT_EXPIRATION=24h
```

### 3. 启动数据库

使用 Docker Compose 启动 PostgreSQL：

```bash
docker-compose up -d postgres
```

### 4. 运行数据库迁移

```bash
# 方式1: 使用 psql 命令
psql -h localhost -U postgres -d purrchat -f migrations/001_init_schema.sql
psql -h localhost -U postgres -d purrchat -f migrations/002_update_schema.sql

# 方式2: 使用 docker exec
docker-compose exec -T postgres psql -U postgres -d purrchat -f /docker-entrypoint-initdb.d/001_init_schema.sql
docker-compose exec -T postgres psql -U postgres -d purrchat -f /docker-entrypoint-initdb.d/002_update_schema.sql
```

### 5. 安装依赖

```bash
go mod download
```

### 6. 运行服务器

```bash
go run cmd/server/main.go
```

服务器将在 `http://localhost:8080` 启动。

### 7. 运行测试

```bash
# 运行所有测试
go test -v ./...

# 运行测试并生成覆盖率报告
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

# 查看覆盖率报告
go tool cover -html=coverage.out
```

## Docker 部署

### 1. 构建镜像

```bash
docker build -t purr-chat-server:latest .
```

### 2. 运行容器

```bash
docker run -d \
  --name purrchat-backend \
  -p 8080:8080 \
  -e DB_HOST=postgres \
  -e DB_PORT=5432 \
  -e DB_USER=postgres \
  -e DB_PASSWORD=postgres \
  -e DB_NAME=purrchat \
  -e JWT_SECRET=your_jwt_secret \
  purr-chat-server:latest
```

### 3. 使用 Docker Compose

```bash
# 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f backend

# 停止服务
docker-compose down

# 停止服务并删除数据卷
docker-compose down -v
```

### 4. 健康检查

```bash
curl http://localhost:8080/health
```

预期响应：

```json
{
  "status": "ok",
  "message": "PurrChat Server is running"
}
```

## GitHub Actions CI/CD

### 配置 GitHub Secrets

在 GitHub 仓库设置中配置以下 Secrets：

#### Docker Hub 认证
- `DOCKER_USERNAME`: Docker Hub 用户名
- `DOCKER_PASSWORD`: Docker Hub 密码或访问令牌

#### 开发服务器配置
- `DEV_SERVER_HOST`: 开发服务器主机名或 IP
- `DEV_SERVER_USER`: 开发服务器用户名
- `DEV_SERVER_SSH_KEY`: 开发服务器 SSH 私钥

#### 生产服务器配置
- `PROD_SERVER_HOST`: 生产服务器主机名或 IP
- `PROD_SERVER_USER`: 生产服务器用户名
- `PROD_SERVER_SSH_KEY`: 生产服务器 SSH 私钥

#### 通知配置
- `SLACK_WEBHOOK`: Slack Webhook URL (可选)

### CI/CD 流程

#### 1. 代码检查 (Lint)

- 使用 golangci-lint 进行代码质量检查
- 检查代码风格、潜在问题、安全漏洞等

#### 2. 测试 (Test)

- 运行所有单元测试
- 生成代码覆盖率报告
- 上传覆盖率到 Codecov

#### 3. 构建 (Build)

- 构建 Docker 镜像
- 推送到 Docker Hub
- 标记版本标签

#### 4. 部署 (Deploy)

- **开发环境**: 自动部署到 `develop` 分支
- **生产环境**: 手动触发部署到 `main` 分支

### 手动触发部署

1. 进入 GitHub 仓库的 "Actions" 标签页
2. 选择 "CI/CD Pipeline" 工作流
3. 点击 "Run workflow"
4. 选择分支并点击 "Run workflow"

## GitLab CI/CD

### 配置 GitLab Variables

在 GitLab 项目设置中配置以下 Variables：

#### CI/CD 变量
- `CI_REGISTRY_USER`: 容器注册表用户名
- `CI_REGISTRY_PASSWORD`: 容器注册表密码
- `DEV_SERVER_HOST`: 开发服务器主机名
- `DEV_SERVER_USER`: 开发服务器用户名
- `DEV_SSH_PRIVATE_KEY`: 开发服务器 SSH 私钥
- `PROD_SERVER_HOST`: 生产服务器主机名
- `PROD_SERVER_USER`: 生产服务器用户名
- `PROD_SSH_PRIVATE_KEY`: 生产服务器 SSH 私钥

### CI/CD 流程

#### 1. 代码检查 (Lint)

- 运行 golangci-lint
- 检查代码质量

#### 2. 测试 (Test)

- 运行单元测试
- 生成覆盖率报告
- 上传测试报告

#### 3. 构建 (Build)

- 构建 Docker 镜像
- 推送到 GitLab Container Registry

#### 4. 部署 (Deploy)

- **开发环境**: 手动触发
- **生产环境**: 手动触发

### 手动触发部署

1. 进入 GitLab 项目的 "CI/CD" > "Pipelines"
2. 点击 "Run pipeline"
3. 选择分支并点击 "Run pipeline"

## 生产环境部署

### 服务器准备

#### 1. 安装 Docker 和 Docker Compose

```bash
# Ubuntu/Debian
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER

# 安装 Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
```

#### 2. 配置防火墙

```bash
# 开放必要端口
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 80/tcp    # HTTP
sudo ufw allow 443/tcp   # HTTPS
sudo ufw allow 8080/tcp  # 应用端口
sudo ufw enable
```

#### 3. 创建部署目录

```bash
sudo mkdir -p /opt/purrchat
sudo chown $USER:$USER /opt/purrchat
cd /opt/purrchat
```

#### 4. 准备配置文件

```bash
# 从仓库复制 docker-compose.yml
scp docker-compose.yml user@server:/opt/purrchat/

# 创建 .env 文件
cat > .env << 'EOF'
PORT=8080
GIN_MODE=release
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_secure_password
DB_NAME=purrchat
JWT_SECRET=your_very_secure_jwt_secret
JWT_EXPIRATION=24h
LOG_DIRECTORY=/app/logs
LOG_MAX_FILES=10
LOG_MAX_LINES=10000
EOF
```

### 部署步骤

#### 1. 登录容器注册表

```bash
docker login -u <username> -p <password>
```

#### 2. 拉取最新镜像

```bash
cd /opt/purrchat
docker-compose pull backend
```

#### 3. 启动服务

```bash
docker-compose up -d
```

#### 4. 运行数据库迁移

```bash
docker-compose exec -T postgres psql -U postgres -d purrchat -f /docker-entrypoint-initdb.d/001_init_schema.sql
docker-compose exec -T postgres psql -U postgres -d purrchat -f /docker-entrypoint-initdb.d/002_update_schema.sql
```

#### 5. 验证部署

```bash
# 检查容器状态
docker-compose ps

# 查看日志
docker-compose logs -f backend

# 健康检查
curl http://localhost:8080/health
```

### 配置 Nginx 反向代理 (可选)

```nginx
server {
    listen 80;
    server_name api.purrchat.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

server {
    listen 443 ssl http2;
    server_name api.purrchat.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## 监控和维护

### 日志管理

```bash
# 查看实时日志
docker-compose logs -f backend

# 查看最近100行日志
docker-compose logs --tail=100 backend

# 导出日志
docker-compose logs backend > backend.log
```

### 数据库备份

```bash
# 创建备份
docker-compose exec postgres pg_dump -U postgres purrchat > backup_$(date +%Y%m%d_%H%M%S).sql

# 恢复备份
docker-compose exec -T postgres psql -U postgres purrchat < backup_20240101_120000.sql
```

### 更新应用

```bash
cd /opt/purrchat

# 拉取新镜像
docker-compose pull backend

# 重启服务
docker-compose up -d backend

# 清理旧镜像
docker image prune -f
```

### 监控指标

建议配置以下监控：

- **应用监控**: Prometheus + Grafana
- **日志聚合**: ELK Stack 或 Loki
- **错误追踪**: Sentry
- **性能监控**: New Relic 或 Datadog

## 故障排查

### 常见问题

#### 1. 容器无法启动

```bash
# 查看容器日志
docker-compose logs backend

# 检查容器状态
docker-compose ps

# 检查资源使用
docker stats
```

#### 2. 数据库连接失败

```bash
# 检查数据库容器
docker-compose ps postgres

# 测试数据库连接
docker-compose exec postgres psql -U postgres -d purrchat -c "SELECT 1"

# 检查网络
docker network inspect purrchat_purrchat-network
```

#### 3. 端口冲突

```bash
# 检查端口占用
sudo lsof -i :8080

# 修改端口
# 编辑 .env 文件中的 PORT 变量
```

#### 4. 内存不足

```bash
# 检查内存使用
free -h

# 增加 Docker 内存限制
# 编辑 docker-compose.yml 添加:
# deploy:
#   resources:
#     limits:
#       memory: 2G
```

### 调试模式

启用调试模式以获取详细日志：

```env
GIN_MODE=debug
```

### 性能优化

1. **数据库优化**
   - 创建适当的索引
   - 定期清理旧数据
   - 使用连接池

2. **应用优化**
   - 启用 Gzip 压缩
   - 使用缓存
   - 优化查询

3. **容器优化**
   - 限制资源使用
   - 使用多阶段构建
   - 最小化镜像大小

## 安全建议

1. **定期更新**: 保持 Docker 镜像和依赖项最新
2. **使用强密码**: 为数据库和 JWT 使用强密码
3. **限制访问**: 使用防火墙和网络安全组
4. **启用 HTTPS**: 使用 SSL/TLS 加密通信
5. **备份策略**: 定期备份数据库和配置
6. **监控日志**: 监控异常活动和安全事件

## 支持

如有问题，请联系：

- 技术支持: support@purrchat.com
- 文档: https://docs.purrchat.com
- 问题追踪: https://github.com/purrchat/purr-chat-server/issues
