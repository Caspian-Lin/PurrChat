# PurrChat Backend

PurrChat 后端 API 服务，基于 Go + Gin + PostgreSQL 构建。

## 技术栈

- **语言**: Go 1.24+
- **Web 框架**: Gin 1.9+
- **数据库**: PostgreSQL 15+ (使用 pgx/v5)
- **认证**: JWT (golang-jwt/jwt/v5)
- **测试**: Testcontainers-go
- **日志**: Logrus

## 开发命令

```bash
# 安装依赖
go mod download

# 启动开发服务器
pnpm dev
# 或
go run cmd/server/main.go

# 构建二进制文件
pnpm build
# 或
go build -o bin/server cmd/server/main.go

# 运行测试
pnpm test
# 或
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

# 测试覆盖率
pnpm test:coverage

# 代码检查
pnpm lint
# 或
golangci-lint run --timeout=5m

# 类型检查
pnpm type-check
# 或
go vet ./...

# 清理构建产物
pnpm clean
```

## 项目结构

```
backend/
├── cmd/
│   └── server/           # 主程序入口
│       └── main.go
├── internal/
│   ├── handlers/         # HTTP 处理器
│   ├── models/           # 数据模型
│   ├── repository/       # 数据访问层
│   └── middleware/       # 中间件
├── migrations/           # 数据库迁移文件
├── tests/                # 测试文件
├── logs/                  # 日志目录
├── go.mod                # Go 模块定义
├── go.sum                # 依赖锁定
├── Makefile              # Make 命令
└── Dockerfile            # Docker 构建文件
```

## 环境变量

在 `.env` 文件中配置：

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

# 日志配置
LOG_DIRECTORY=/app/logs
LOG_MAX_FILES=10
LOG_MAX_LINES=10000
```

## Docker 构建

```bash
# 构建镜像
docker build -t purrchat-backend .

# 运行容器
docker run -p 8080:8080 purrchat-backend
```

## 数据库迁移

数据库迁移文件位于 `migrations/` 目录：

- `001_init_schema.sql` - 初始化数据库结构
- `002_update_schema.sql` - 更新数据库结构

使用 Docker Compose 时，迁移会自动执行。

## API 端点

### 认证

- `POST /api/auth/register` - 用户注册
- `POST /api/auth/login` - 用户登录
- `POST /api/auth/logout` - 用户登出
- `GET /api/auth/me` - 获取当前用户信息

### 用户

- `GET /api/users/:id` - 获取用户信息
- `PUT /api/users/:id` - 更新用户信息

### 消息

- `GET /api/messages` - 获取消息列表
- `POST /api/messages` - 发送消息

### 健康检查

- `GET /health` - 健康检查

## 测试

项目使用 Testcontainers-go 进行集成测试：

```bash
# 运行所有测试
go test -v ./...

# 运行测试并生成覆盖率报告
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -html=coverage.out -o coverage.html
```

## 日志

日志文件存储在 `logs/` 目录：

- `{timestamp}-info.log` - 信息日志
- `{timestamp}-error.log` - 错误日志

日志轮转配置：
- 最大文件数: 10
- 每文件最大行数: 10000

## 注意事项

1. 确保 Go 版本 >= 1.24
2. PostgreSQL 数据库需要先启动
3. 生产环境请修改 JWT_SECRET
4. 建议使用 golangci-lint 进行代码检查
