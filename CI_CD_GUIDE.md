# CI/CD 指南

本文档详细说明 PurrChat 项目的 CI/CD 流程配置和使用方法。

## 目录

- [概述](#概述)
- [分支策略](#分支策略)
- [GitHub Actions](#github-actions)
- [本地 CI/CD](#本地-cicd)
- [环境变量配置](#环境变量配置)
- [部署流程](#部署流程)
- [故障排查](#故障排查)

## 概述

PurrChat 项目使用 GitHub Actions 作为 CI/CD 平台：

- **GitHub Actions** - 使用 [`.github/workflows/ci.yml`](.github/workflows/ci.yml:1)

## 分支策略

项目采用三分支模型：

```
main (生产环境)
  ↑
test (预发布环境)
  ↑
dev (开发环境)
```

### 分支职责

| 分支 | 用途 | CI/CD | 部署环境 |
|------|------|-------|----------|
| [`main`](main:1) | 生产环境，只接受从 [`test`](main:1) 分支的合并 | GitHub Actions | 生产环境 (purrchat.com) |
| [`test`](main:1) | 预发布环境，运行完整 CI/CD 验证 | GitHub Actions | 测试环境 (test.purrchat.com) |
| [`dev`](main:1) | 开发环境，日常开发分支 | 本地 CI/CD | 开发环境 |

### 工作流程

1. **开发阶段**
   - 在 [`dev`](main:1) 分支进行日常开发
   - 提交代码到 [`dev`](main:1) 分支

2. **合并到 test**
   - 从 [`dev`](main:1) 分支合并到 [`test`](main:1) 分支
   - 在 [`test`](main:1) 分支上运行本地 CI/CD
   - 本地 CI/CD 通过后，推送到远程 [`test`](main:1) 分支
   - GitHub Actions 自动运行完整 CI/CD 流程

3. **合并到 main**
   - GitHub Actions CI/CD 通过后，从 [`test`](main:1) 合并到 [`main`](main:1)
   - GitHub Actions 自动运行生产环境 CI/CD
   - 自动部署到生产环境

### 快捷命令

```bash
# 从 dev 合并到 test（自动运行本地 CI/CD）
bash scripts/merge-dev-to-test.sh

# 手动运行本地 CI/CD
bash scripts/run-local-ci.sh
```

## GitHub Actions

### 流水线阶段

GitLab CI 流水线分为 4 个阶段：

```
lint → test → build → deploy
```

#### 1. Lint 阶段

**lint:frontend** - 前端代码检查

```yaml
lint:frontend:
  stage: lint
  image: node:20
  script:
    - pnpm lint
```

**lint:backend** - 后端代码检查

```yaml
lint:backend:
  stage: lint
  image: golang:1.24
  script:
    - golangci-lint run --timeout=5m
```

#### 2. Test 阶段

**test:frontend** - 前端测试

```yaml
test:frontend:
  stage: test
  image: node:20
  script:
    - pnpm test:coverage
  coverage: '/Lines\s+:\s+\d+\.?\d*%/'
```

**test:backend** - 后端测试

```yaml
test:backend:
  stage: test
  image: golang:1.24
  services:
    - postgres:15-alpine
  script:
    - go test -v -race -coverprofile=coverage.out ./...
  coverage: '/coverage: \d+\.\d+% of statements/'
```

#### 3. Build 阶段

**build:frontend** - 构建前端 Docker 镜像

```yaml
build:frontend:
  stage: build
  image: docker:24.0.5
  services:
    - docker:24.0.5-dind
  script:
    - docker buildx build --push
      --tag $CI_REGISTRY_IMAGE/frontend:$CI_COMMIT_SHORT_SHA
      -f apps/frontend/Dockerfile
      apps/frontend
```

**build:backend** - 构建后端 Docker 镜像

```yaml
build:backend:
  stage: build
  image: docker:24.0.5
  services:
    - docker:24.0.5-dind
  script:
    - docker buildx build --push
      --tag $CI_REGISTRY_IMAGE/backend:$CI_COMMIT_SHORT_SHA
      -f apps/backend/Dockerfile
      apps/backend
```

#### 4. Deploy 阶段

**deploy:dev** - 部署到开发环境

```yaml
deploy:dev:
  stage: deploy
  environment:
    name: development
    url: https://dev.purrchat.com
  only:
    - develop
  when: manual
```

**deploy:prod** - 部署到生产环境

```yaml
deploy:prod:
  stage: deploy
  environment:
    name: production
    url: https://purrchat.com
  only:
    - main
  when: manual
```

### 触发条件

| 任务 | 分支 | 触发方式 |
|------|------|----------|
| lint | main, develop, MR | 自动 |
| test | main, develop, MR | 自动 |
| build | main, develop | 自动 |
| deploy:dev | develop | 手动 |
| deploy:prod | main | 手动 |

### 配置步骤

1. **启用 GitLab CI**

在 GitLab 项目设置中，确保 CI/CD 功能已启用。

2. **配置环境变量**

在 `Settings > CI/CD > Variables` 中添加以下变量：

| 变量名 | 说明 | 类型 | 保护 | 遮蔽 |
|--------|------|------|------|------|
| `CI_REGISTRY` | 镜像仓库地址 | Variable | 否 | 否 |
| `CI_REGISTRY_USER` | 镜像仓库用户名 | Variable | 否 | 否 |
| `CI_REGISTRY_PASSWORD` | 镜像仓库密码 | Variable | 是 | 是 |
| `DEV_SERVER_HOST` | 开发服务器地址 | Variable | 否 | 否 |
| `DEV_SERVER_USER` | 开发服务器用户 | Variable | 否 | 否 |
| `DEV_SSH_PRIVATE_KEY` | 开发服务器 SSH 密钥 | File | 是 | 是 |
| `PROD_SERVER_HOST` | 生产服务器地址 | Variable | 是 | 否 |
| `PROD_SERVER_USER` | 生产服务器用户 | Variable | 是 | 否 |
| `PROD_SSH_PRIVATE_KEY` | 生产服务器 SSH 密钥 | File | 是 | 是 |

3. **生成 SSH 密钥**

```bash
# 生成 SSH 密钥对
ssh-keygen -t ed25519 -C "gitlab-ci" -f ~/.ssh/gitlab_ci_key

# 将公钥添加到服务器
ssh-copy-id -i ~/.ssh/gitlab_ci_key.pub user@server

# 将私钥内容复制到 GitLab CI/CD 变量
cat ~/.ssh/gitlab_ci_key
```

## GitHub Actions

### 工作流结构

GitHub Actions 工作流包含以下 Jobs：

```
lint-frontend → test-frontend → build-frontend → deploy-test/prod
lint-backend  → test-backend  → build-backend → deploy-test/prod
```

### Jobs 说明

#### Lint Jobs

```yaml
lint-frontend:
  name: Lint Frontend
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-node@v4
    - run: pnpm lint
```

```yaml
lint-backend:
  name: Lint Backend
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
    - run: golangci-lint run
```

#### Test Jobs

```yaml
test-frontend:
  name: Test Frontend
  runs-on: ubuntu-latest
  needs: lint-frontend
  steps:
    - uses: actions/checkout@v4
    - run: pnpm test:coverage
    - uses: codecov/codecov-action@v4
```

```yaml
test-backend:
  name: Test Backend
  runs-on: ubuntu-latest
  needs: lint-backend
  services:
    postgres:
      image: postgres:15-alpine
  steps:
    - uses: actions/checkout@v4
    - run: go test -v -race -coverprofile=coverage.out ./...
    - uses: codecov/codecov-action@v4
```

#### Build Jobs

```yaml
build-frontend:
  name: Build Frontend
  runs-on: ubuntu-latest
  needs: test-frontend
  permissions:
    contents: read
    packages: write
  steps:
    - uses: actions/checkout@v4
    - uses: docker/login-action@v3
    - uses: docker/build-push-action@v5
```

#### Deploy Jobs

```yaml
deploy-dev:
  name: Deploy to Development
  runs-on: ubuntu-latest
  needs: [build-frontend, build-backend]
  environment:
    name: development
    url: https://dev.purrchat.com
  steps:
    - uses: actions/checkout@v4
    - uses: appleboy/ssh-action@v1.0.3
```

### 触发条件

| 事件 | 分支 | 触发 |
|------|------|------|
| Push | main, dev, test | ✅ |
| Pull Request | main, dev, test | ✅ |
| Build & Test | test, main | ✅ |
| Deploy | test | 自动 |
| Deploy | main | 自动 |

### 配置步骤

1. **启用 GitHub Actions**

   在 GitHub 仓库设置中，确保 Actions 功能已启用。

2. **配置 Secrets**

   在 `Settings > Secrets and variables > Actions` 中添加以下 Secrets：

   | Secret 名称 | 说明 |
   |-------------|------|
   | `TEST_SERVER_HOST` | 测试服务器地址 |
   | `TEST_SERVER_USER` | 测试服务器用户 |
   | `TEST_SSH_PRIVATE_KEY` | 测试服务器 SSH 私钥 |
   | `PROD_SERVER_HOST` | 生产服务器地址 |
   | `PROD_SERVER_USER` | 生产服务器用户 |
   | `PROD_SSH_PRIVATE_KEY` | 生产服务器 SSH 私钥 |

3. **配置 Environments**

   在 `Settings > Environments` 中创建环境：

   - **Test**
     - Name: `test`
     - URL: `https://test.purrchat.com`
     - Protection rules: 无

   - **Production**
     - Name: `production`
     - URL: `https://purrchat.com`
     - Protection rules: 需要审批

4. **生成 SSH 密钥**

   ```bash
   # 生成 SSH 密钥对
   ssh-keygen -t ed25519 -C "github-actions" -f ~/.ssh/github_actions_key

   # 将公钥添加到服务器
   ssh-copy-id -i ~/.ssh/github_actions_key.pub user@server

   # 将私钥内容复制到 GitHub Secrets
   cat ~/.ssh/github_actions_key
   ```

## 本地 CI/CD

项目提供了本地 CI/CD 脚本，用于在从 [`dev`](main:1) 合并到 [`test`](main:1) 分支前进行验证。

### 脚本说明

#### 1. 运行本地 CI/CD

[`scripts/run-local-ci.sh`](scripts/run-local-ci.sh:1) - 本地运行完整的 CI/CD 流程

```bash
bash scripts/run-local-ci.sh
```

该脚本会执行以下步骤：
- 检查当前分支是否为 [`test`](main:1)
- 检查工作目录是否干净
- 安装前端依赖
- 运行前端和后端 Lint
- 运行前端和后端测试
- 构建前端和后端

#### 2. 从 dev 合并到 test

[`scripts/merge-dev-to-test.sh`](scripts/merge-dev-to-test.sh:1) - 自动化从 [`dev`](main:1) 合并到 [`test`](main:1) 的流程

```bash
bash scripts/merge-dev-to-test.sh
```

该脚本会执行以下步骤：
- 检查当前分支是否为 [`dev`](main:1)
- 拉取最新代码
- 切换到 [`test`](main:1) 分支
- 合并 [`dev`](main:1) 分支
- 运行本地 CI/CD
- 推送到远程 [`test`](main:1) 分支

### 使用建议

1. **日常开发**
   - 在 [`dev`](main:1) 分支进行开发
   - 提交代码到 [`dev`](main:1) 分支

2. **准备发布**
   - 运行 `bash scripts/merge-dev-to-test.sh`
   - 等待本地 CI/CD 通过
   - 等待 GitHub Actions CI/CD 通过

3. **发布到生产**
   - 从 [`test`](main:1) 合并到 [`main`](main:1)
   - GitHub Actions 自动部署到生产环境

## 环境变量配置

### 通用环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `GO_VERSION` | Go 版本 | `1.24` |
| `NODE_VERSION` | Node.js 版本 | `20` |
| `DB_HOST` | 数据库主机 | `postgres` |
| `DB_PORT` | 数据库端口 | `5432` |
| `DB_NAME` | 数据库名称 | `testdb` |
| `DB_USER` | 数据库用户 | `testuser` |
| `DB_PASSWORD` | 数据库密码 | `testpass` |

### 前端环境变量

| 变量名 | 说明 |
|--------|------|
| `VITE_API_BASE_URL` | API 基础 URL |

### 后端环境变量

| 变量名 | 说明 |
|--------|------|
| `PORT` | 服务器端口 |
| `GIN_MODE` | Gin 运行模式 |
| `JWT_SECRET` | JWT 密钥 |
| `JWT_EXPIRATION` | JWT 过期时间 |
| `LOG_DIRECTORY` | 日志目录 |
| `LOG_MAX_FILES` | 最大日志文件数 |
| `LOG_MAX_LINES` | 每文件最大行数 |

## 部署流程

### 测试环境部署

测试环境部署是自动的，当 [`test`](main:1) 分支的 CI/CD 通过后自动触发。

1. **触发条件**

    - 推送到 [`test`](main:1) 分支
    - GitHub Actions CI/CD 全部通过

2. **部署步骤**

    ```bash
    # 1. 拉取最新镜像
    docker-compose pull backend frontend

    # 2. 启动服务
    docker-compose up -d backend frontend

    # 3. 清理旧镜像
    docker image prune -f
    ```

3. **健康检查**

    ```bash
    curl -f https://test.purrchat.com/health
    curl -f https://test.purrchat.com/
    ```

### 生产环境部署

生产环境部署是自动的，当 [`main`](main:1) 分支的 CI/CD 通过后自动触发。

1. **触发条件**

    - 从 [`test`](main:1) 合并到 [`main`](main:1)
    - GitHub Actions CI/CD 全部通过

2. **部署步骤**

    ```bash
    # 1. 备份数据库
    docker-compose exec -T postgres pg_dump -U postgres purrchat > backup_$(date +%Y%m%d_%H%M%S).sql

    # 2. 拉取最新镜像
    docker-compose pull backend frontend

    # 3. 启动服务
    docker-compose up -d backend frontend

    # 4. 清理旧镜像
    docker image prune -f
    ```

3. **健康检查**

    ```bash
    curl -f https://api.purrchat.com/health
    curl -f https://purrchat.com/
    ```

### 回滚流程

如果部署失败，可以快速回滚：

```bash
# 1. 停止当前服务
docker-compose down

# 2. 恢复数据库备份
docker-compose up -d postgres
docker-compose exec -T postgres psql -U postgres purrchat < backup_20240101_120000.sql

# 3. 拉取上一个版本的镜像
docker pull registry.example.com/purrchat/backend:previous-tag
docker pull registry.example.com/purrchat/frontend:previous-tag

# 4. 启动服务
docker-compose up -d backend frontend
```

## 故障排查

### 常见问题

#### 1. Lint 失败

**问题**: 代码检查失败

**解决方案**:

```bash
# 本地运行 lint
pnpm lint

# 自动修复
pnpm format
```

#### 2. 测试失败

**问题**: 测试用例失败

**解决方案**:

```bash
# 本地运行测试
pnpm test

# 查看详细日志
pnpm test --reporter=verbose
```

#### 3. 构建失败

**问题**: Docker 镜像构建失败

**解决方案**:

```bash
# 本地构建测试
docker build -t test-image -f apps/frontend/Dockerfile apps/frontend

# 查看构建日志
docker build --progress=plain -t test-image -f apps/frontend/Dockerfile apps/frontend
```

#### 4. 部署失败

**问题**: 部署到服务器失败

**解决方案**:

```bash
# 检查 SSH 连接
ssh user@server

# 检查 Docker 状态
docker-compose ps

# 查看服务日志
docker-compose logs backend
docker-compose logs frontend
```

#### 5. 健康检查失败

**问题**: 服务启动后健康检查失败

**解决方案**:

```bash
# 检查服务状态
curl http://localhost:8080/health
curl http://localhost:80/

# 查看服务日志
docker-compose logs --tail=100 backend
docker-compose logs --tail=100 frontend

# 重启服务
docker-compose restart backend
docker-compose restart frontend
```

### 调试技巧

#### 查看流水线日志

- GitLab: 在 CI/CD > Pipelines 页面点击具体任务查看日志
- GitHub: 在 Actions 页面点击具体 workflow run 查看日志

#### 本地复现 CI 环境

```bash
# 使用相同的 Docker 镜像
docker run -it --rm node:20 bash
docker run -it --rm golang:1.24 bash

# 使用相同的环境变量
export GO_VERSION=1.24
export NODE_VERSION=20
```

#### 使用 SSH 调试

在 CI 配置中添加调试步骤：

```yaml
debug:
  stage: debug
  script:
    - echo "Debugging..."
  when: on_failure
```

## 参考资料

- [GitLab CI 文档](https://docs.gitlab.com/ee/ci/)
- [GitHub Actions 文档](https://docs.github.com/en/actions)
- [Docker 文档](https://docs.docker.com/)
- [Turborepo 文档](https://turbo.build/repo/docs)
