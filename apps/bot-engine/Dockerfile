# 多阶段构建 — bot-engine (TypeScript/Hono)
FROM node:20-slim AS builder

WORKDIR /workspace

RUN npm install -g pnpm

COPY pnpm-workspace.yaml package.json pnpm-lock.yaml ./
COPY packages/workflow-types ./packages/workflow-types
COPY packages/workflow-engine ./packages/workflow-engine
COPY apps/bot-engine ./apps/bot-engine

RUN pnpm install --frozen-lockfile

RUN pnpm -F bot-engine run build

# 运行阶段
FROM node:20-slim

WORKDIR /app

RUN groupadd -g 1001 appuser && \
    useradd -r -u 1001 -g appuser appuser

COPY --from=builder /workspace/apps/bot-engine/dist ./dist
COPY --from=builder /workspace/apps/bot-engine/package.json ./
COPY --from=builder /workspace/node_modules ./node_modules
COPY --from=builder /workspace/packages ./packages

USER appuser

EXPOSE 3001

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3001/health || exit 1

CMD ["node", "dist/index.js"]
