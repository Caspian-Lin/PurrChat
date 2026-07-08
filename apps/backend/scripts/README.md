# 数据库清理脚本

本目录包含用于清理和重置 PurrChat 数据库的脚本。

## 警告

⚠️ **重要提示**：这些脚本将删除整个数据库，请谨慎使用！

## 脚本说明

### 1. cleanup_database.sql

纯 SQL 脚本，用于删除并重建整个 PurrChat 数据库。

#### 功能

- 终止所有连接到 purrchat 数据库的会话
- 删除整个 purrchat 数据库
- 创建新的 purrchat 数据库
- 创建必要的扩展（如 uuid-ossp）
- 显示重建后的数据库状态

#### 使用方法

```bash
# 直接使用 psql 执行（需要连接到 postgres 数据库）
psql -U postgres -d postgres -f scripts/cleanup_database.sql

# 或者指定主机和端口
psql -h localhost -p 5432 -U postgres -d postgres -f scripts/cleanup_database.sql
```

#### 安全特性

- 执行前显示 5 秒倒计时，允许取消操作
- 自动终止所有连接，避免删除失败
- 详细的日志输出，显示每个操作步骤
- 不依赖于现有的数据库结构

### 2. cleanup_database.sh

Shell 包装脚本，提供更友好的用户界面和额外功能。

#### 功能

- ✅ 自动备份数据库（可选）
- ✅ 交互式确认提示
- ✅ 彩色输出，易于阅读
- ✅ 支持环境变量配置
- ✅ 详细的错误处理
- ✅ 自动连接到 postgres 数据库以删除 purrchat 数据库

#### 使用方法

```bash
# 基本用法（会提示确认）
./scripts/cleanup_database.sh

# 自动备份并跳过确认
./scripts/cleanup_database.sh --backup --yes

# 使用环境变量配置数据库连接
DB_HOST=localhost \
DB_PORT=5432 \
DB_NAME=postgres \
DB_USER=postgres \
DB_PASSWORD=mypassword \
./scripts/cleanup_database.sh --backup --yes

# 显示帮助信息
./scripts/cleanup_database.sh --help
```

#### 命令行选项

| 选项           | 说明               |
| -------------- | ------------------ |
| `-b, --backup` | 在清理前备份数据库 |
| `-y, --yes`    | 跳过确认提示       |
| `-h, --help`   | 显示帮助信息       |

#### 环境变量

| 变量              | 说明                                              | 默认值        |
| ----------------- | ------------------------------------------------- | ------------- |
| `DB_HOST`         | 数据库主机                                        | `localhost`   |
| `DB_PORT`         | 数据库端口                                        | `5432`        |
| `DB_NAME`         | 管理连接数据库                                    | `postgres`    |
| `DB_USER`         | 管理数据库用户，需要 `CREATEDB`/`CREATEROLE` 权限 | `postgres`    |
| `DB_PASSWORD`     | 管理数据库用户密码                                | （空）        |
| `APP_DB_NAME`     | 应用数据库名称                                    | `purrchat`    |
| `APP_DB_USER`     | 应用迁移/运行用户                                 | `purrchat`    |
| `APP_DB_PASSWORD` | 应用数据库用户密码                                | `purrchat_pw` |

## 使用场景

### 场景 1：开发环境重置

```bash
# 备份并清理开发数据库。脚本会创建 APP_DB_USER，并授予 public schema 的迁移权限。
./scripts/cleanup_database.sh --backup --yes

# 重新初始化数据库
make migrate
```

### 场景 2：生产环境迁移

```bash
# 1. 备份生产数据库
./scripts/cleanup_database.sh --backup

# 2. 确认备份文件
ls -lh scripts/backups/

# 3. 执行清理
./scripts/cleanup_database.sh

# 4. 运行新的迁移
make migrate

# 5. 验证数据库
make test
```

### 场景 3：测试环境清理

```bash
# 快速清理测试数据库（不需要备份）
./scripts/cleanup_database.sh --yes

# 重新初始化
make migrate
```

## 清理后的操作

清理数据库后，需要运行新的迁移脚本来初始化数据库结构：

```bash
# 方法 1：使用 psql 先修复应用用户权限，再执行迁移
psql -U postgres -d purrchat -c "ALTER DATABASE purrchat OWNER TO purrchat; GRANT ALL PRIVILEGES ON DATABASE purrchat TO purrchat; GRANT USAGE, CREATE ON SCHEMA public TO purrchat; ALTER SCHEMA public OWNER TO purrchat;"

# 方法 2：使用 Makefile（推荐）
make migrate

# 方法 3：直接运行 Go 程序
cd apps/backend && go run cmd/server/main.go migrate
```

## 迁移脚本自动执行

新的迁移系统会自动执行 `migrations` 目录下的所有 SQL 文件，按文件名排序执行（如 001、002、003...）。添加新的迁移文件时，只需在 `migrations` 目录下添加新的 `.sql` 文件即可，无需修改代码。

## 备份恢复

如果需要恢复备份的数据库：

```bash
# 恢复最新的备份
psql -U postgres -d purrchat < scripts/backups/purrchat_backup_YYYYMMDD_HHMMSS.sql

# 或者使用 pg_restore（如果使用自定义格式）
pg_restore -U postgres -d purrchat scripts/backups/purrchat_backup.dump
```

## 故障排除

### 问题：权限错误

```bash
# 错误：permission denied: './scripts/cleanup_database.sh'
# 解决：添加执行权限
chmod +x scripts/cleanup_database.sh
```

### 问题：连接数据库失败

```bash
# 检查 PostgreSQL 服务是否运行
sudo systemctl status postgresql

# 检查数据库连接
psql -h localhost -U postgres -d purrchat
```

### 问题：备份失败

```bash
# 确保 pg_dump 已安装
which pg_dump

# 检查磁盘空间
df -h

# 检查备份目录权限
ls -ld scripts/backups/
```

## 注意事项

1. **生产环境**：在生产环境使用前，务必先备份数据库
2. **数据丢失**：此操作不可逆，请确保有备份
3. **停机时间**：清理过程可能导致短暂的服务中断
4. **依赖关系**：确保清理后立即运行新的迁移脚本
5. **测试验证**：清理后运行测试验证数据库结构
6. **数据库连接**：清理脚本需要连接到 `postgres` 数据库，而不是 `purrchat` 数据库

## 相关文档

- [数据库迁移指南](../../docs/MIGRATION.md)
- [部署指南](../../docs/DEPLOYMENT.md)
- [数据库架构](../../migrations/001_init_schema.sql)

## 支持

如果遇到问题，请：

1. 检查日志输出中的错误信息
2. 验证数据库连接配置
3. 确认 PostgreSQL 服务状态
4. 查阅故障排除部分
