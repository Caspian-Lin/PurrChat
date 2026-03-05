#!/bin/bash

# 本地 CI/CD 运行脚本
# 用途：在从 dev 分支合并到 test 分支前，本地运行完整的 CI/CD 流程

set -e  # 遇到错误立即退出

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查当前分支
check_branch() {
    local current_branch=$(git branch --show-current)
    if [ "$current_branch" != "test" ]; then
        print_error "当前分支是 '$current_branch'，请切换到 test 分支运行此脚本"
        print_info "运行: git checkout test"
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
    echo "======================================"
    echo "  本地 CI/CD 运行脚本"
    echo "  分支: dev → test"
    echo "======================================"
    echo ""

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
    echo ""
    echo "======================================"
    echo "  Lint 阶段"
    echo "======================================"
    lint_frontend
    lint_backend

    # Test 阶段
    echo ""
    echo "======================================"
    echo "  Test 阶段"
    echo "======================================"
    test_frontend
    test_backend

    # Build 阶段
    echo ""
    echo "======================================"
    echo "  Build 阶段"
    echo "======================================"
    build_frontend
    build_backend

    # 完成
    echo ""
    echo "======================================"
    print_success "本地 CI/CD 全部通过！"
    echo "======================================"
    echo ""
    print_info "现在可以安全地将 test 分支合并到 main 分支"
    print_info "运行: git checkout main && git merge test"
    echo ""
}

# 运行主函数
main
