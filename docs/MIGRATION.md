# PurrChat 数据库迁移指南

## 迁移步骤

### 1. 备份现有数据（如果有）

```bash
# 使用 pg_dump 备份 PostgreSQL 数据库
pg_dump -h localhost -U your_username -d purrchat > backup.sql
```

### 2. 停止服务器

确保后端服务器已停止运行。

### 3. 执行迁移脚本

```bash
# 连接到 PostgreSQL 数据库并执行迁移脚本
psql -h localhost -U your_username -d purrchat -f migrations/001_init_schema.sql
```

或者使用数据库管理工具（如 pgAdmin、DBeaver 等）打开并执行 `migrations/001_init_schema.sql` 文件。

### 4. 验证迁移

连接到数据库并验证表结构：

```sql
-- 检查用户表
\d users

-- 检查会话表
\d conversations

-- 检查消息表
\d messages

-- 检查好友关系表
\d friendships
```

### 5. 重启服务器

```bash
# 在后端目录下
cd purr-chat-server
go run cmd/server/main.go
```

## 数据库结构

### users 表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| uid | INTEGER | 数字UID（自动递增） |
| username | VARCHAR(20) | 用户名（唯一） |
| password_hash | TEXT | 密码哈希 |
| salt | TEXT | 密码盐值 |
| nickname | VARCHAR(20) | 昵称 |
| avatar_url | TEXT | 头像URL |
| email | VARCHAR(100) | 邮箱（可选，唯一） |
| email_verified | BOOLEAN | 邮箱是否验证 |
| phone | VARCHAR(20) | 手机号（可选，唯一） |
| phone_verified | BOOLEAN | 手机号是否验证 |
| created_at | TIMESTAMP | 创建时间 |

### conversations 表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| conversation_type | VARCHAR(20) | 会话类型（friend/stranger） |
| user1_id | UUID | 用户1 ID |
| user2_id | UUID | 用户2 ID |
| has_pending_request | BOOLEAN | 是否有待处理的好友请求 |
| request_status | VARCHAR(20) | 请求状态（none/pending/accepted/rejected） |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

### messages 表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| conversation_id | UUID | 会话ID |
| sender_id | UUID | 发送者ID |
| content | TEXT | 消息内容 |
| msg_type | VARCHAR(20) | 消息类型（text/image） |
| created_at | TIMESTAMP | 创建时间 |

### friendships 表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| user_id | UUID | 用户ID |
| friend_id | UUID | 好友ID |
| status | VARCHAR(20) | 状态（pending/accepted/blocked） |
| created_at | TIMESTAMP | 创建时间 |

## 功能说明

### 1. 用户注册

用户注册时会自动分配一个数字UID，便于记忆和通过UID添加好友。

### 2. 搜索用户

支持通过以下方式搜索用户：
- UID（数字）
- 手机号
- 邮箱

### 3. 陌生人会话

陌生人之间可以发送消息，但在对方没有回复前只能发送3条消息。

### 4. 好友请求

- 发送好友请求会在会话中显示
- 存在未处理的好友请求的会话会置顶
- 接受好友请求后会话类型变为好友会话
- 拒绝好友请求后会话状态更新

### 5. 会话管理

- 每个会话单独管理
- 支持查看会话历史消息
- 支持实时聊天（需要实现WebSocket）

## 故障排除

### 迁移失败

如果迁移脚本执行失败，请检查：

1. PostgreSQL 服务是否正在运行
2. 数据库连接信息是否正确
3. 是否有足够的权限执行DDL语句
4. 是否有其他连接占用数据库

### 服务器启动失败

如果服务器启动失败，请检查：

1. 数据库连接配置（`.env` 文件）
2. 数据库schema是否正确迁移
3. 端口是否被占用
