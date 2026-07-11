#!/bin/bash

# ======================================
# PurrChat 数据库清理脚本
# ======================================
#
# 此脚本用于删除并重建 PurrChat 数据库。
# 这是最彻底的清理方式，不依赖于现有的数据库结构。
#
# 警告：此脚本将删除整个数据库！请谨慎使用！
#
# 使用方法：
#   ./scripts/cleanup_database.sh [options]
#
# 选项：
#   -b, --backup    在清理前备份数据库
#   -y, --yes       跳过确认提示
#   -h, --help      显示帮助信息
# ======================================

set -e  # 遇到错误时退出

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 默认值
BACKUP=false
SKIP_CONFIRM=false

# 数据库配置（从环境变量读取，如果没有则使用默认值）
# 注意：清理脚本使用管理用户连接到 postgres 数据库，然后创建应用数据库和应用用户。
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-postgres}"  # 管理连接数据库，通常是 postgres
DB_USER="${DB_USER:-postgres}"  # 管理用户，需要 CREATEDB/CREATEROLE 权限
DB_PASSWORD="${DB_PASSWORD:-}"
APP_DB_NAME="${APP_DB_NAME:-purrchat}"
APP_DB_USER="${APP_DB_USER:-purrchat}"
APP_DB_PASSWORD="${APP_DB_PASSWORD:-purrchat_pw}"

# 解析命令行参数
while [[ $# -gt 0 ]]; do
    case $1 in
        -b|--backup)
            BACKUP=true
            shift
            ;;
        -y|--yes)
            SKIP_CONFIRM=true
            shift
            ;;
        -h|--help)
            echo "PurrChat 数据库清理脚本"
            echo ""
            echo "此脚本将删除并重建整个 PurrChat 数据库。"
            echo "这是最彻底的清理方式，不依赖于现有的数据库结构。"
            echo ""
            echo "使用方法："
            echo "  $0 [options]"
            echo ""
            echo "选项："
            echo "  -b, --backup    在清理前备份数据库"
            echo "  -y, --yes       跳过确认提示"
            echo "  -h, --help      显示帮助信息"
            echo ""
            echo "环境变量："
            echo "  DB_HOST         数据库主机 (默认: localhost)"
            echo "  DB_PORT         数据库端口 (默认: 5432)"
            echo "  DB_NAME          管理连接数据库 (默认: postgres)"
            echo "  DB_USER          管理数据库用户，需 CREATEDB/CREATEROLE 权限 (默认: postgres)"
            echo "  DB_PASSWORD      管理数据库用户密码"
            echo "  APP_DB_NAME      应用数据库名称 (默认: purrchat)"
            echo "  APP_DB_USER      应用迁移/运行用户 (默认: purrchat)"
            echo "  APP_DB_PASSWORD  应用数据库用户密码 (默认: purrchat_pw)"
            echo ""
            echo "示例："
            echo "  $0 --backup --yes"
            echo "  DB_USER=myuser DB_PASSWORD=mypass $0"
            echo ""
            echo "警告：此操作将删除整个 purrchat 数据库及其所有数据！"
            exit 0
            ;;
        *)
            echo -e "${RED}未知选项: $1${NC}"
            echo "使用 -h 或 --help 查看帮助信息"
            exit 1
            ;;
    esac
done

# 获取脚本所在目录
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
SQL_FILE="$SCRIPT_DIR/cleanup_database.sql"

# 检查 SQL 文件是否存在
if [ ! -f "$SQL_FILE" ]; then
    echo -e "${RED}错误: 找不到 SQL 文件: $SQL_FILE${NC}"
    exit 1
fi

# 显示配置信息
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}PurrChat 数据库清理脚本${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "数据库配置："
echo "  主机: $DB_HOST"
echo "  端口: $DB_PORT"
echo "  管理连接数据库: $DB_NAME"
echo "  管理用户: $DB_USER"
echo "  应用数据库: $APP_DB_NAME"
echo "  应用用户: $APP_DB_USER"
echo ""
echo "选项："
echo "  备份数据库: $BACKUP"
echo "  跳过确认: $SKIP_CONFIRM"
echo ""

# 备份数据库
if [ "$BACKUP" = true ]; then
    echo -e "${YELLOW}正在备份数据库...${NC}"
    
    BACKUP_DIR="$SCRIPT_DIR/backups"
    BACKUP_FILE="$BACKUP_DIR/${APP_DB_NAME}_backup_$(date +%Y%m%d_%H%M%S).sql"
    
    # 创建备份目录
    mkdir -p "$BACKUP_DIR"
    
    # 设置 PGPASSWORD 环境变量
    export PGPASSWORD="$DB_PASSWORD"
    
    # 执行备份
    if pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$APP_DB_NAME" > "$BACKUP_FILE" 2>/dev/null; then
        echo -e "${GREEN}✓ 数据库备份成功: $BACKUP_FILE${NC}"
    else
        echo -e "${RED}✗ 数据库备份失败${NC}"
        exit 1
    fi
    
    # 清理 PGPASSWORD 环境变量
    unset PGPASSWORD
    
    echo ""
fi

# 确认操作
if [ "$SKIP_CONFIRM" = false ]; then
    echo -e "${RED}========================================${NC}"
    echo -e "${RED}警告：此操作将删除整个 $APP_DB_NAME 数据库！${NC}"
    echo -e "${RED}========================================${NC}"
    echo ""
    read -p "确认要继续吗？(yes/no): " confirm
    if [ "$confirm" != "yes" ]; then
        echo "操作已取消"
        exit 0
    fi
    echo ""
fi

# 执行清理脚本
echo -e "${YELLOW}正在删除并重建 $APP_DB_NAME 数据库...${NC}"
echo ""

# 设置 PGPASSWORD 环境变量
export PGPASSWORD="$DB_PASSWORD"

# 执行 SQL 脚本（连接到 postgres 数据库）
if psql \
    -h "$DB_HOST" \
    -p "$DB_PORT" \
    -U "$DB_USER" \
    -d "$DB_NAME" \
    -v app_db_name="$APP_DB_NAME" \
    -v app_db_user="$APP_DB_USER" \
    -v app_db_password="$APP_DB_PASSWORD" \
    -f "$SQL_FILE"; then
    echo ""
    echo -e "${GREEN}✓ 数据库重建成功${NC}"
else
    echo ""
    echo -e "${RED}✗ 数据库重建失败${NC}"
    exit 1
fi

# 清理 PGPASSWORD 环境变量
unset PGPASSWORD

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}下一步操作：${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "运行迁移脚本以初始化数据库："
echo "  make migrate"
echo ""
echo "或者直接运行："
echo "  cd apps/backend && go run ./cmd/migrate up"
echo ""
echo -e "${BLUE}========================================${NC}"
