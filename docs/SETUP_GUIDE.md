# PurrChat 设置指南

## 前置要求

- Node.js >= 18
- pnpm >= 9
- Go >= 1.24
- Docker (可选，用于容器化部署)

## 安装依赖

### 安装 Node.js 依赖

```bash
pnpm install
```

### 安装 Go 工具

```bash
cd apps/backend
go mod download
```

### 安装 golangci-lint (可选但推荐)

golangci-lint 是 Go 项目的代码检查工具，可以提供更全面的代码质量检查。

```bash
# 安装 golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 验证安装
golangci-lint --version
```

如果未安装 golangci-lint，lint 命令会自动降级使用 `go vet`。

## 开发命令

所有命令都从项目根目录执行。

### 使用 Makefile (推荐)

```bash
make help         # 查看所有可用命令
make install      # 安装所有依赖
make dev          # 启动开发模式
make build        # 构建所有应用
make test         # 运行所有测试
make lint         # 运行代码检查
make lint-fix     # 自动修复代码问题
make format       # 格式化代码
make type-check   # 类型检查
make clean        # 清理构建产物和依赖
```

### 使用 pnpm

```bash
pnpm run build        # 构建所有应用
pnpm run dev          # 启动开发模式
pnpm run lint         # 运行代码检查
pnpm run lint:fix     # 自动修复代码问题
pnpm run test         # 运行所有测试
pnpm run format       # 格式化代码
pnpm run type-check   # 类型检查
pnpm run clean        # 清理构建产物和依赖
```

### Docker 命令

```bash
make docker-up        # 启动 Docker 容器
make docker-down      # 停止 Docker 容器
make docker-logs      # 查看 Docker 日志
make docker-build     # 构建 Docker 镜像
```

## 项目结构

```
PurrChat/
├── apps/
│   ├── backend/      # Go 后端服务
│   └── frontend/     # Vue.js 前端应用
├── Makefile          # 统一的构建入口
├── package.json      # 根 package.json (pnpm workspace)
├── turbo.json        # Turborepo 配置
└── .gitlab-ci.yml    # CI/CD 配置
```

## Turborepo 说明

本项目使用 Turborepo 来管理 monorepo，提供以下优势：

- **并行执行**: 同时运行多个子项目的任务
- **智能缓存**: 基于输入的缓存机制，避免重复构建
- **任务依赖**: 自动处理任务间的依赖关系
- **统一入口**: 通过根目录统一管理所有子项目

## CI/CD

项目使用 GitLab CI/CD 进行持续集成和部署，配置文件位于根目录的 `.gitlab-ci.yml`。

CI/CD 流程包括：

1. **Lint**: 代码质量检查
2. **Test**: 运行测试并生成覆盖率报告
3. **Build**: 构建 Docker 镜像
4. **Deploy**: 部署到开发/生产环境

## 故障排除

### lint 命令失败

如果 `make lint` 失败，请确保：

1. 已安装所有依赖：`pnpm install`
2. 对于 backend，已安装 golangci-lint（可选）：`go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`

### 测试失败

如果测试失败，请确保：

1. 数据库服务正在运行（如果需要）
2. 已安装所有依赖
3. 环境变量已正确配置

### 构建失败

如果构建失败，请确保：

1. 已安装所有依赖
2. Node.js 和 Go 版本符合要求
3. 磁盘空间充足

## 更多信息

- [Turborepo 文档](https://turbo.build/repo/docs)
- [pnpm 文档](https://pnpm.io/)
- [Go 文档](https://golang.org/doc/)
- [Vue.js 文档](https://vuejs.org/)
