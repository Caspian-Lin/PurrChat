# PurrChat Server - 测试和部署文档

本文档提供了 PurrChat 后端服务器的测试和部署指南。

## 📁 项目结构

```
purr-chat-server/
├── .github/
│   └── workflows/
│       └── ci-cd.yml              # GitHub Actions CI/CD 配置
├── .gitlab-ci.yml                    # GitLab CI/CD 配置
├── Dockerfile                        # Docker 镜像构建文件
├── docker-compose.yml                 # Docker Compose 配置
├── .dockerignore                    # Docker 构建忽略文件
├── Makefile                         # 开发命令快捷方式
├── DEPLOYMENT.md                    # 详细部署指南
├── scripts/
│   └── run-tests.sh                 # 测试运行脚本
└── tests/
    ├── setup_test.go                 # 测试设置
    ├── auth_test.go                 # 认证功能测试
    ├── user_test.go                 # 用户功能测试
    ├── conversation_test.go          # 会话功能测试
    └── message_and_friend_test.go   # 消息和好友功能测试
```

## 🧪 测试

### 测试覆盖范围

测试文件覆盖了所有 API 端点：

#### 1. 认证功能测试 ([`auth_test.go`](tests/auth_test.go))

- ✅ 用户注册
  - 成功注册
  - 用户名已存在
  - 邮箱已存在
  - 用户名太短
  - 密码太短
  - 邮箱格式错误

- ✅ 用户登录
  - 成功登录
  - 用户名不存在
  - 密码错误
  - 缺少邮箱
  - 缺少密码

- ✅ 获取当前用户信息
  - 成功获取用户信息
  - 未提供 token
  - 无效的 token

- ✅ 更新个人资料
  - 成功更新邮箱
  - 成功更新手机号
  - 邮箱格式错误
  - 未提供 token

#### 2. 用户功能测试 ([`user_test.go`](tests/user_test.go))

- ✅ 搜索用户
  - 成功搜索用户
  - 通过邮箱搜索
  - 搜索结果为空
  - 缺少查询参数
  - 未提供 token

- ✅ 根据 ID 获取用户信息
  - 成功获取用户信息
  - 用户不存在
  - 无效的用户 ID 格式
  - 未提供 token

- ✅ 完整认证流程
  - 注册 → 登录 → 获取用户信息

#### 3. 会话功能测试 ([`conversation_test.go`](tests/conversation_test.go))

- ✅ 获取会话列表
  - 成功获取会话列表
  - 未提供 token
  - 无效的 token

- ✅ 创建会话
  - 成功创建会话
  - 创建已存在的会话
  - 不能与自己创建会话
  - 目标用户不存在
  - 无效的用户 ID 格式
  - 未提供 token

- ✅ 会话工作流
  - 创建会话并获取

#### 4. 消息和好友功能测试 ([`message_and_friend_test.go`](tests/message_and_friend_test.go))

- ✅ 获取消息列表
  - 成功获取消息列表
  - 缺少 conversation_id
  - 无效的 conversation_id
  - 未提供 token

- ✅ 发送消息
  - 成功发送文本消息
  - 成功发送图片消息
  - 会话不存在
  - 内容为空
  - 未提供 token

- ✅ 发送好友请求
  - 成功发送好友请求
  - 不能向自己发送好友请求
  - 目标用户不存在
  - 无效的用户 ID 格式
  - 未提供 token

- ✅ 处理好友请求
  - 成功接受好友请求
  - 成功拒绝好友请求
  - 无效的操作
  - 会话不存在
  - 未提供 token

- ✅ 获取好友列表
  - 成功获取好友列表
  - 未提供 token
  - 无效的 token

- ✅ 完整好友工作流
  - 发送好友请求 → 接受好友请求 → 获取好友列表

### 运行测试

#### 方式 1: 使用 Makefile

```bash
# 运行所有测试
make test

# 运行测试并查看覆盖率
make test-coverage

# 运行所有检查（lint + vet + test）
make check
```

#### 方式 2: 使用测试脚本

```bash
# 运行测试脚本
./scripts/run-tests.sh
```

#### 方式 3: 直接使用 Go 命令

```bash
# 运行所有测试
go test -v ./...

# 运行测试并生成覆盖率报告
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

# 查看覆盖率报告
go tool cover -html=coverage.out
```

### 测试覆盖率要求

- 最低覆盖率: 70%
- 目标覆盖率: 80%+

## 🐳 Docker 部署

### 本地开发环境

#### 1. 启动所有服务

```bash
# 使用 Docker Compose
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

#### 2. 仅启动数据库

```bash
docker-compose up -d postgres
```

#### 3. 运行数据库迁移

```bash
make migrate-up
```

### 生产环境部署

#### 1. 构建镜像

```bash
make docker-build
```

#### 2. 推送到镜像仓库

```bash
# 登录 Docker Hub
docker login

# 推送镜像
docker push purr-chat-server:latest
```

#### 3. 在服务器上部署

```bash
# 拉取最新镜像
docker pull purr-chat-server:latest

# 启动容器
docker run -d \
  --name purrchat-backend \
  -p 8080:8080 \
  -e DB_HOST=postgres \
  -e DB_PORT=5432 \
  -e DB_USER=postgres \
  -e DB_PASSWORD=your_password \
  -e DB_NAME=purrchat \
  -e JWT_SECRET=your_jwt_secret \
  purr-chat-server:latest
```

## 🚀 CI/CD 部署

### GitHub Actions

#### 配置 Secrets

在 GitHub 仓库设置中配置以下 Secrets：

- `DOCKER_USERNAME`: Docker Hub 用户名
- `DOCKER_PASSWORD`: Docker Hub 密码
- `DEV_SERVER_HOST`: 开发服务器主机名
- `DEV_SERVER_USER`: 开发服务器用户名
- `DEV_SERVER_SSH_KEY`: 开发服务器 SSH 私钥
- `PROD_SERVER_HOST`: 生产服务器主机名
- `PROD_SERVER_USER`: 生产服务器用户名
- `PROD_SERVER_SSH_KEY`: 生产服务器 SSH 私钥
- `SLACK_WEBHOOK`: Slack Webhook URL (可选)

#### CI/CD 流程

```
代码推送
  ↓
代码检查 (Lint)
  ↓
运行测试 (Test)
  ↓
构建镜像 (Build)
  ↓
部署到开发环境 (Deploy Dev) [自动]
部署到生产环境 (Deploy Prod) [手动]
```

### GitLab CI/CD

#### 配置 Variables

在 GitLab 项目设置中配置以下 Variables：

- `CI_REGISTRY_USER`: 容器注册表用户名
- `CI_REGISTRY_PASSWORD`: 容器注册表密码
- `DEV_SERVER_HOST`: 开发服务器主机名
- `DEV_SERVER_USER`: 开发服务器用户名
- `DEV_SSH_PRIVATE_KEY`: 开发服务器 SSH 私钥
- `PROD_SERVER_HOST`: 生产服务器主机名
- `PROD_SERVER_USER`: 生产服务器用户名
- `PROD_SSH_PRIVATE_KEY`: 生产服务器 SSH 私钥

#### CI/CD 流程

```
代码推送
  ↓
代码检查 (Lint)
  ↓
运行测试 (Test)
  ↓
构建镜像 (Build)
  ↓
部署到开发环境 (Deploy Dev) [手动]
部署到生产环境 (Deploy Prod) [手动]
```

## 📋 Makefile 命令

| 命令 | 描述 |
|------|------|
| `make help` | 显示帮助信息 |
| `make build` | 构建应用 |
| `make test` | 运行测试 |
| `make test-coverage` | 显示测试覆盖率 |
| `make lint` | 运行代码检查 |
| `make clean` | 清理构建文件 |
| `make docker-build` | 构建 Docker 镜像 |
| `make docker-run` | 运行 Docker 容器 |
| `make docker-stop` | 停止 Docker 容器 |
| `make docker-logs` | 查看 Docker 日志 |
| `make docker-clean` | 清理 Docker 资源 |
| `make migrate-up` | 运行数据库迁移 |
| `make migrate-down` | 回滚数据库迁移 |
| `make setup` | 初始化开发环境 |
| `make dev` | 启动开发环境 |
| `make prod` | 启动生产环境 |
| `make db-shell` | 进入数据库 Shell |
| `make db-backup` | 备份数据库 |
| `make db-restore` | 恢复数据库 |
| `make install-tools` | 安装开发工具 |
| `make fmt` | 格式化代码 |
| `make vet` | 运行 go vet |
| `make deps` | 更新依赖 |
| `make run` | 运行应用 |
| `make check` | 运行所有检查 |

## 🔍 监控和维护

### 日志管理

```bash
# 查看实时日志
docker-compose logs -f backend

# 查看最近 100 行日志
docker-compose logs --tail=100 backend

# 导出日志
docker-compose logs backend > backend.log
```

### 数据库备份

```bash
# 创建备份
make db-backup

# 恢复备份
make db-restore
```

### 健康检查

```bash
# 检查应用健康状态
curl http://localhost:8080/health
```

## 📚 相关文档

- [详细部署指南](DEPLOYMENT.md)
- [数据库迁移](MIGRATION.md)
- [更新日志](UPDATE_SUMMARY.md)
- [主 README](README.md)

## 🆘 故障排查

### 常见问题

1. **容器无法启动**
   ```bash
   docker-compose logs backend
   docker-compose ps
   ```

2. **数据库连接失败**
   ```bash
   docker-compose ps postgres
   docker-compose exec postgres psql -U postgres -d purrchat -c "SELECT 1"
   ```

3. **端口冲突**
   ```bash
   sudo lsof -i :8080
   ```

4. **测试失败**
   ```bash
   # 检查依赖
   go mod download
   
   # 清理缓存
   go clean -cache
   ```

## 📞 支持

如有问题，请联系：

- 技术支持: support@purrchat.com
- 文档: https://docs.purrchat.com
- 问题追踪: https://github.com/purrchat/purr-chat-server/issues

## 📝 许可证

本项目采用 MIT 许可证。详见 LICENSE 文件。
