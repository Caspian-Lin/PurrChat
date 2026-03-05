#!/bin/bash

# 从 dev 分支合并到 test 分支的脚本
# 用途：自动化从 dev 合并到 test 的流程，包括运行本地 CI/CD

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

# 确认操作
confirm() {
    read -p "$1 (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_info "操作已取消"
        exit 0
    fi
}

# 主函数
main() {
    echo "======================================"
    echo "  从 dev 合并到 test 分支"
    echo "======================================"
    echo ""

    # 检查当前分支
    local current_branch=$(git branch --show-current)
    if [ "$current_branch" != "dev" ]; then
        print_error "当前分支是 '$current_branch'，请切换到 dev 分支运行此脚本"
        print_info "运行: git checkout dev"
        exit 1
    fi
    print_success "当前分支: $current_branch"

    # 检查是否有未提交的更改
    if [ -n "$(git status --porcelain)" ]; then
        print_error "工作目录有未提交的更改，请先提交或暂存"
        git status
        exit 1
    fi
    print_success "工作目录干净"

    # 拉取最新代码
    print_info "拉取最新代码..."
    git pull origin dev
    print_success "代码已更新"

    # 确认合并
    echo ""
    print_warning "即将执行以下操作："
    echo "  1. 切换到 test 分支"
    echo "  2. 拉取最新代码"
    echo "  3. 合并 dev 分支"
    echo "  4. 运行本地 CI/CD"
    echo "  5. 推送到远程 test 分支"
    echo ""
    confirm "是否继续？"

    # 切换到 test 分支
    print_info "切换到 test 分支..."
    git checkout test
    print_success "已切换到 test 分支"

    # 拉取最新代码
    print_info "拉取 test 分支最新代码..."
    git pull origin test
    print_success "test 分支代码已更新"

    # 合并 dev 分支
    print_info "合并 dev 分支到 test..."
    git merge dev
    print_success "合并完成"

    # 运行本地 CI/CD
    echo ""
    print_info "运行本地 CI/CD..."
    bash scripts/run-local-ci.sh

    # 推送到远程
    echo ""
    print_info "推送到远程 test 分支..."
    git push origin test
    print_success "推送成功"

    # 完成
    echo ""
    echo "======================================"
    print_success "合并完成！"
    echo "======================================"
    echo ""
    print_info "GitHub Actions 将在 test 分支上运行 CI/CD"
    print_info "CI/CD 通过后，可以将 test 分支合并到 main 分支"
    echo ""
    print_info "运行以下命令合并到 main:"
    echo "  git checkout main"
    echo "  git pull origin main"
    echo "  git merge test"
    echo "  git push origin main"
    echo ""
}

# 运行主函数
main
