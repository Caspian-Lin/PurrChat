# PurrChat Backend

Go + Gin + PostgreSQL 后端 API 服务。

## 技术栈

- **语言**: Go 1.24+
- **Web 框架**: Gin
- **数据库**: PostgreSQL 15+ (pgx/v5)
- **认证**: JWT (golang-jwt/jwt/v5)
- **测试**: Testcontainers-go

## 开发命令

```bash
go run cmd/server/main.go        # 启动开发服务器
go test -v ./...                  # 运行测试
golangci-lint run                 # 代码检查
```

## API 端点

### 认证 (无需 Token)

| 方法 | 路径            | 说明     |
| ---- | --------------- | -------- |
| POST | `/api/register` | 用户注册 |
| POST | `/api/login`    | 用户登录 |

### 需要认证 (Bearer Token)

| 方法   | 路径                         | 说明              |
| ------ | ---------------------------- | ----------------- |
| GET    | `/api/me`                    | 获取当前用户信息  |
| PUT    | `/api/profile`               | 更新个人资料      |
| GET    | `/api/users/search?query=`   | 搜索用户          |
| GET    | `/api/users/:id`             | 获取用户信息      |
| GET    | `/api/users/uid/:uid`        | 通过 UID 获取用户 |
| GET    | `/api/conversations`         | 获取会话列表      |
| POST   | `/api/conversations`         | 创建私聊会话      |
| POST   | `/api/conversations/group`   | 创建群聊          |
| GET    | `/api/conversations/members` | 获取会话成员      |
| POST   | `/api/conversations/members` | 添加成员          |
| DELETE | `/api/conversations/members` | 移除成员          |
| GET    | `/api/messages`              | 获取消息列表      |
| POST   | `/api/messages`              | 发送消息          |
| GET    | `/api/messages/export`       | 导出消息          |
| GET    | `/api/messages/incremental`  | 增量获取消息      |
| GET    | `/api/friends`               | 好友列表          |
| GET    | `/api/friends/pending`       | 待处理好友请求    |
| GET    | `/api/friends/requests`      | 所有好友请求      |
| POST   | `/api/friends/request`       | 发送好友请求      |
| POST   | `/api/friends/handle`        | 处理好友请求      |
| GET    | `/api/ws?token=`             | WebSocket 连接    |

### 存储 (Storage 服务, 端口 8081)

| 方法   | 路径                               | 说明                      |
| ------ | ---------------------------------- | ------------------------- |
| POST   | `/api/files/upload/request`        | 申请上传 (获取预签名 URL) |
| POST   | `/api/files/upload/confirm`        | 确认上传完成              |
| GET    | `/api/files/download/url?file_id=` | 获取下载链接              |
| DELETE | `/api/files/:file_id`              | 删除文件                  |

## 项目结构

```
backend/
├── cmd/server/           # 主程序入口
├── internal/
│   ├── handlers/         # HTTP 处理器
│   ├── models/           # 数据模型
│   ├── repository/       # 数据访问层
│   ├── services/         # 业务逻辑层
│   └── websocket/        # WebSocket 管理
├── pkg/
│   ├── config/           # 配置加载
│   ├── database/         # 数据库连接
│   ├── jwt/              # JWT 工具
│   └── logger/           # 日志
├── migrations/           # 数据库迁移
└── tests/                # 集成测试
```

## 环境变量

```env
PORT=8080
GIN_MODE=release
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=purrchat
JWT_SECRET=your_jwt_secret_key_here
JWT_EXPIRATION=24h
```
