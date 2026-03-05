.PHONY: help install dev build test lint lint-fix clean format type-check docker-up docker-down docker-logs docker-build migrate

# 默认目标
help:
	@echo "PurrChat Turborepo - 可用命令:"
	@echo ""
	@echo "开发命令:"
	@echo "  make install      - 安装所有依赖"
	@echo "  make dev          - 启动开发模式"
	@echo "  make build        - 构建所有应用"
	@echo "  make test         - 运行所有测试"
	@echo "  make lint         - 运行代码检查"
	@echo "  make lint-fix     - 自动修复代码问题"
	@echo "  make format       - 格式化代码"
	@echo "  make type-check   - 类型检查"
	@echo ""
	@echo "Docker 命令:"
	@echo "  make docker-up    - 启动 Docker 容器"
	@echo "  make docker-down  - 停止 Docker 容器"
	@echo "  make docker-logs  - 查看 Docker 日志"
	@echo "  make docker-build - 构建 Docker 镜像"
	@echo ""
	@echo "清理命令:"
	@echo "  make clean        - 清理构建产物和依赖"
	@echo ""
	@echo "数据库迁移:"
	@echo "  make migrate      - 执行所有数据库迁移"

ifneq (,$(wildcard ./apps/backend/.env))
    include ./apps/backend/.env
    export
endif
# 安装依赖
install:
	pnpm install

# 开发模式
dev:
	pnpm run dev

# 构建所有应用
build:
	pnpm run build

# 运行测试
test:
	pnpm run test

# 代码检查
lint:
	pnpm run lint

# 自动修复代码问题
lint-fix:
	pnpm run lint:fix

# 格式化代码
format:
	pnpm run format

# 类型检查
type-check:
	pnpm run type-check

# 清理
clean:
	pnpm run clean

# 数据库迁移
migrate:
	@echo "执行数据库迁移..."
	cd apps/backend && go run cmd/server/main.go migrate

# Docker 启动
docker-up:
	docker-compose up -d

# Docker 停止
docker-down:
	docker-compose down

# Docker 日志
docker-logs:
	docker-compose logs -f

# Docker 构建
docker-build:
	docker-compose build
