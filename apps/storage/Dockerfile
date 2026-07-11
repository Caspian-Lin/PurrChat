# 多阶段构建 - 构建阶段
FROM golang:1.24 AS builder

# 设置工作目录
WORKDIR /workspace/apps/storage

# 安装必要的工具
RUN apt-get update && apt-get install -y \
    git \
    ca-certificates \
    tzdata \
    && rm -rf /var/lib/apt/lists/*

# 复制go mod文件
COPY apps/storage/go.mod apps/storage/go.sum ./
COPY packages/db-migrate /workspace/packages/db-migrate

# 配置代理
ENV GOPROXY=https://goproxy.cn,direct

# 下载依赖
RUN go mod download

# 复制源代码
COPY apps/storage/ .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/bin/storage \
    ./cmd/server

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/bin/migrate \
    ./cmd/migrate

# 运行阶段
FROM ubuntu:24.04

# 设置环境变量以避免交互式提示
ENV DEBIAN_FRONTEND=noninteractive

# 安装必要的运行时依赖
RUN apt-get update && apt-get install -y \
    ca-certificates \
    tzdata \
    wget \
    && rm -rf /var/lib/apt/lists/*

# 创建非root用户
RUN groupadd -g 1001 appuser && \
    useradd -r -u 1001 -g appuser appuser

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/bin/storage /app/storage
COPY --from=builder /app/bin/migrate /app/migrate

# 创建必要的目录
RUN mkdir -p /app/logs /app/migrations && \
    chown -R appuser:appuser /app && \
    chmod -R 755 /app/logs

# 复制迁移文件
COPY --from=builder /workspace/apps/storage/migrations/ /app/migrations/

# 切换到非root用户
USER appuser

# 暴露端口
EXPOSE 8081

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8081/health || exit 1

# 启动应用
CMD ["/app/storage"]
