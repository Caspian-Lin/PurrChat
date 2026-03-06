#!/bin/bash

# 本地 CI/CD 运行脚本
# 用途：在从 dev 分支合并到 beta 分支前，本地运行完整的 CI/CD 流程

set -e  # 遇到错误立即退出

# 日志目录
LOG_DIR="logs"

# 创建日志目录
mkdir -p "$LOG_DIR"

# 日志文件名（使用开始时间）
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
LOG_FILE="$LOG_DIR/local-ci-$TIMESTAMP.log"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印带颜色的消息（同时输出到控制台和日志文件）
print_info() {
    local msg="${BLUE}[INFO]${NC} $1"
    echo -e "$msg" | tee -a "$LOG_FILE"
}

print_success() {
    local msg="${GREEN}[SUCCESS]${NC} $1"
    echo -e "$msg" | tee -a "$LOG_FILE"
}

print_warning() {
    local msg="${YELLOW}[WARNING]${NC} $1"
    echo -e "$msg" | tee -a "$LOG_FILE"
}

print_error() {
    local msg="${RED}[ERROR]${NC} $1"
    echo -e "$msg" | tee -a "$LOG_FILE"
}

# 检查当前分支
check_branch() {
    local current_branch=$(git branch --show-current)
    if [ "$current_branch" != "beta" ]; then
        print_error "当前分支是 '$current_branch'，请切换到 beta 分支运行此脚本"
        print_info "运行: git checkout beta"
        exit 1
    fi
    print_success "当前分支: $current_branch"
}

# 检查是否有未提交的更改
check_clean_state() {
    if [ -n "$(git status --porcelain)" ]; then
        print_error "工作目录有未提交的更改，请先提交或暂存"
        git status
        exit 1
    fi
    print_success "工作目录干净"
}

# 安装前端依赖
install_frontend_deps() {
    print_info "安装前端依赖..."
    cd apps/frontend
    pnpm install
    cd ../..
    print_success "前端依赖安装完成"
}

# 前端 Lint
lint_frontend() {
    print_info "运行前端 Lint..."
    cd apps/frontend
    pnpm lint
    cd ../..
    print_success "前端 Lint 通过"
}

# 后端 Lint
lint_backend() {
    print_info "运行后端 Lint..."
    cd apps/backend
    golangci-lint run --timeout=5m
    cd ../..
    print_success "后端 Lint 通过"
}

# 前端测试
test_frontend() {
    print_info "运行前端测试..."
    cd apps/frontend
    pnpm test:coverage
    cd ../..
    print_success "前端测试通过"
}

# 后端测试
test_backend() {
    print_info "运行后端测试..."
    cd apps/backend
    go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
    cd ../..
    print_success "后端测试通过"
}

# 构建前端
build_frontend() {
    print_info "构建前端..."
    cd apps/frontend
    pnpm build
    cd ../..
    print_success "前端构建完成"
}

# 构建后端
build_backend() {
    print_info "构建后端..."
    cd apps/backend
    go build -o bin/server ./cmd/server
    cd ../..
    print_success "后端构建完成"
}

# 主函数
main() {
    echo "======================================" | tee -a "$LOG_FILE"
    echo "  本地 CI/CD 运行脚本" | tee -a "$LOG_FILE"
    echo "  分支: dev → beta" | tee -a "$LOG_FILE"
    echo "======================================" | tee -a "$LOG_FILE"
    echo "" | tee -a "$LOG_FILE"

    # 检查环境
    check_branch
    check_clean_state

    # 检查必要的工具
    print_info "检查必要的工具..."
    command -v pnpm >/dev/null 2>&1 || { print_error "pnpm 未安装，请先安装"; exit 1; }
    command -v go >/dev/null 2>&1 || { print_error "go 未安装，请先安装"; exit 1; }
    command -v golangci-lint >/dev/null 2>&1 || { print_error "golangci-lint 未安装，请先安装"; exit 1; }
    print_success "所有必要工具已安装"

    # 安装依赖
    install_frontend_deps

    # Lint 阶段
    echo "" | tee -a "$LOG_FILE"
    echo "======================================" | tee -a "$LOG_FILE"
    echo "  Lint 阶段" | tee -a "$LOG_FILE"
    echo "======================================" | tee -a "$LOG_FILE"
    lint_frontend
    lint_backend

    # Test 阶段
    echo "" | tee -a "$LOG_FILE"
    echo "======================================" | tee -a "$LOG_FILE"
    echo "  Test 阶段" | tee -a "$LOG_FILE"
    echo "======================================" | tee -a "$LOG_FILE"
    test_frontend
    test_backend

    # Build 阶段
    echo "" | tee -a "$LOG_FILE"
    echo "======================================" | tee -a "$LOG_FILE"
    echo "  Build 阶段" | tee -a "$LOG_FILE"
    echo "======================================" | tee -a "$LOG_FILE"
    build_frontend
    build_backend

    # 完成
    echo "" | tee -a "$LOG_FILE"
    echo "======================================" | tee -a "$LOG_FILE"
    print_success "本地 CI/CD 全部通过！"
    echo "======================================" | tee -a "$LOG_FILE"
    echo "" | tee -a "$LOG_FILE"
    print_info "现在可以安全地将 beta 分支合并到 main 分支"
    print_info "运行: git checkout main && git merge beta"
    echo "" | tee -a "$LOG_FILE"
    print_info "日志文件: $LOG_FILE"
    echo "" | tee -a "$LOG_FILE"
}

# 运行主函数
main
