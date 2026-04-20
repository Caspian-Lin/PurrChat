.PHONY: help install dev build test lint lint-fix clean format type-check docker-up docker-down docker-logs docker-build migrate

# 日志目录
LOG_DIR := logs

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
	@echo ""
	@echo "存储服务:"
	@echo "  make dev-storage  - 启动存储服务"

ifneq (,$(wildcard ./apps/backend/.env))
    include ./apps/backend/.env
    export
endif

ifneq (,$(wildcard ./apps/storage/.env))
    include ./apps/storage/.env
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

# 运行测试（包含所有CI中的lint和test）
test:
	@mkdir -p $(LOG_DIR)
	@TIMESTAMP=$$(date +%Y%m%d-%H%M%S); \
	LOG_FILE="$(LOG_DIR)/test-$$TIMESTAMP.log"; \
	echo "======================================" | tee $$LOG_FILE; \
	echo "  运行完整的CI测试流程" | tee -a $$LOG_FILE; \
	echo "======================================" | tee -a $$LOG_FILE; \
	echo "" | tee -a $$LOG_FILE; \
	echo "[1/5] 前端 Lint..." | tee -a $$LOG_FILE; \
	$(MAKE) lint-frontend >> $$LOG_FILE 2>&1 || (echo "" | tee -a $$LOG_FILE && echo "======================================" | tee -a $$LOG_FILE && echo "  ❌ 测试失败！" | tee -a $$LOG_FILE && echo "======================================" | tee -a $$LOG_FILE && exit 1); \
	echo "" | tee -a $$LOG_FILE; \
	echo "[2/5] 后端 Lint..." | tee -a $$LOG_FILE; \
	$(MAKE) lint-backend >> $$LOG_FILE 2>&1 || (echo "" | tee -a $$LOG_FILE && echo "======================================" | tee -a $$LOG_FILE && echo "  ❌ 测试失败！" | tee -a $$LOG_FILE && echo "======================================" | tee -a $$LOG_FILE && exit 1); \
	echo "" | tee -a $$LOG_FILE; \
	echo "[3/5] 前端测试..." | tee -a $$LOG_FILE; \
	$(MAKE) test-frontend >> $$LOG_FILE 2>&1 || (echo "" | tee -a $$LOG_FILE && echo "======================================" | tee -a $$LOG_FILE && echo "  ❌ 测试失败！" | tee -a $$LOG_FILE && echo "======================================" | tee -a $$LOG_FILE && exit 1); \
	echo "" | tee -a $$LOG_FILE; \
	echo "[4/5] 后端测试..." | tee -a $$LOG_FILE; \
	$(MAKE) test-backend >> $$LOG_FILE 2>&1 || (echo "" | tee -a $$LOG_FILE && echo "======================================" | tee -a $$LOG_FILE && echo "  ❌ 测试失败！" | tee -a $$LOG_FILE && echo "======================================" | tee -a $$LOG_FILE && exit 1); \
	echo "" | tee -a $$LOG_FILE; \
	echo "[5/5] 存储服务测试..." | tee -a $$LOG_FILE; \
	$(MAKE) test-storage >> $$LOG_FILE 2>&1 || (echo "" | tee -a $$LOG_FILE && echo "======================================" | tee -a $$LOG_FILE && echo "  ❌ 测试失败！" | tee -a $$LOG_FILE && echo "======================================" | tee -a $$LOG_FILE && exit 1); \
	echo "" | tee -a $$LOG_FILE; \
	echo "======================================" | tee -a $$LOG_FILE; \
	echo "  ✅ 所有测试通过！" | tee -a $$LOG_FILE; \
	echo "======================================" | tee -a $$LOG_FILE; \
	echo "" | tee -a $$LOG_FILE; \
	echo "日志文件: $$LOG_FILE" | tee -a $$LOG_FILE

# 代码检查
lint:
	@$(MAKE) lint-frontend
	@$(MAKE) lint-backend
	@$(MAKE) lint-storage

# 前端 Lint
lint-frontend:
	@echo "运行前端 Lint..."
	cd apps/frontend && pnpm lint

# 后端 Lint
lint-backend:
	@echo "运行后端 Lint..."
	@if ! cd apps/backend && gofmt -l . | grep -q .; then \
		echo "❌ Go 文件格式不正确，请运行 make format 修复："; \
		gofmt -l .; \
		exit 1; \
	fi
	cd apps/backend && golangci-lint run --timeout=5m

# 前端测试
test-frontend:
	@echo "运行前端测试..."
	cd apps/frontend && pnpm test:coverage --run

# 后端测试
test-backend:
	@echo "运行后端测试..."
	cd apps/backend && go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

# 自动修复代码问题
lint-fix:
	pnpm run lint:fix
	@echo "格式化后端代码..."
	cd apps/backend && gofmt -w .

# 格式化代码
format:
	pnpm run format
	@echo "格式化后端代码..."
	cd apps/backend && gofmt -w .

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

# 启动存储服务（独立开发）
dev-storage:
	cd apps/storage && go run cmd/server/main.go

# 存储服务测试
test-storage:
	@echo "Running storage tests..."
	cd apps/storage && go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

# 存储服务 Lint
lint-storage:
	@echo "Running storage lint..."
	cd apps/storage && golangci-lint run --timeout=5m 2>/dev/null || go vet ./...

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
