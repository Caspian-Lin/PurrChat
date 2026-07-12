# 数据库迁移

PurrChat 的 backend 与 storage 共用 PostgreSQL，但各自拥有独立的迁移目录和版本命名空间：

- `apps/backend/migrations/`，服务名为 `backend`
- `apps/storage/migrations/`，服务名为 `storage`

迁移记录保存于共享表 `purrchat_schema_migrations`，以 `(service, version)` 为主键，记录源文件名、SHA-256 checksum 和执行时间。

## 新增迁移

新迁移文件使用 `NNN_description.sql` 格式，在所属服务目录中分配从当前最大编号递增的未使用编号。例如 backend 当前最新版本是 `011`，下一份应为 `012_description.sql`。

已发布的 SQL 文件、文件名和内容都不可修改、重命名或复用。运行器会拒绝重复版本、已应用文件缺失、文件名变化和 checksum 漂移。

backend 历史上有两份 `006` 文件。为兼容已经部署的文件名，运行器以只读 legacy 映射将它们记录为逻辑版本 `006a` 和 `006b`；不得再新增带字母后缀的迁移。

每份迁移在单独事务中执行，因此不得在迁移 SQL 中使用 PostgreSQL 不允许置于事务内的语句，例如 `CREATE INDEX CONCURRENTLY`。这类变更必须另行设计并在 PR 中说明锁、回滚和部署步骤。

## 执行迁移

```bash
make migrate
```

如果本机只有数据库管理员密码，可显式作为命令参数传入：

```bash
cd apps/backend && go run ./cmd/migrate up --admin-password '管理员密码'
cd apps/storage && go run ./cmd/migrate up --admin-password '管理员密码'
```

管理员账号由 `DB_ADMIN_USER`（默认 `postgres`）指定。连接建立后 migrator 会先
`SET ROLE DB_USER`，因此表、索引和函数仍归应用角色所有；密码不会写入迁移日志。
命令行参数可能出现在本机进程列表中，CI/生产环境仍优先使用 secret 注入的环境变量。

该命令按顺序执行 backend 再执行 storage。每个服务使用 PostgreSQL advisory lock，阻止同一服务的并发迁移实例；已经记录的迁移不会重复执行。

也可单独执行：

```bash
make migrate-backend
make migrate-storage
```

Docker Compose 使用 `backend-migrate` 和 `storage-migrate` one-shot 服务。应用服务只会在相应 migrator 成功退出后启动；不要再向 PostgreSQL 的 `/docker-entrypoint-initdb.d` 挂载业务迁移目录。

## 旧数据库 Baseline

旧机制没有迁移记录。为了避免把历史 SQL 重放到已有数据库，首次运行新机制发现服务的 schema 已存在但没有记录时会失败，并要求明确 baseline。

先确认备份与当前 schema，再分别执行：

```bash
cd apps/backend && go run ./cmd/migrate baseline
cd apps/storage && go run ./cmd/migrate baseline
```

`baseline` 不执行任何 SQL，只写入当前目录中所有迁移的 filename 和 checksum。它要求 backend 已有 `users` 表、storage 已有 `file_metadata` 表，并拒绝覆盖已经存在的该服务迁移记录。完成后运行 `make migrate`，用于校验记录并应用之后新增的迁移。

容器化部署可使用：

```bash
docker compose run --rm backend-migrate /app/migrate baseline
docker compose run --rm storage-migrate /app/migrate baseline
```

## 回滚与故障处理

运行器不提供自动 down migration。迁移失败会回滚正在执行的单份 SQL，并且不会写入其历史记录。对于已在生产应用的 DDL，恢复路径是预先验证过的备份恢复或单独提交的前向修复迁移；不要删除记录、修改已发布 SQL 或通过清库来恢复生产环境。
