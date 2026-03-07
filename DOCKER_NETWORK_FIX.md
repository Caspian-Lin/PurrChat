# Docker 网络问题修复说明

## 问题描述

在 Docker 中运行 PurrChat 时，前端出现网络错误：

```
[axios] 响应拦截器错误 {message: 'Network Error', status: undefined, data: undefined, url: '/api/friends'}
WebSocket error: Event {isTrusted: true, type: 'error', ...}
WebSocket closed: 1006
```

错误日志显示：
- `[WebSocket] API Base URL: http://localhost:8080`
- `[WebSocket] VITE_API_BASE_URL env: undefined`

## 根本原因分析

### 1. 环境变量未正确注入

Vite 的环境变量是在**构建时**注入的，而不是运行时。但是：

1. `.dockerignore` 文件忽略了 `.env` 文件（第46行）
2. Dockerfile 在构建时没有接收环境变量参数
3. docker-compose.yml 中的 `environment` 配置只在运行时生效

这导致 `VITE_API_BASE_URL` 在构建时为 `undefined`，前端代码使用了默认值 `http://localhost:8080`。

### 2. 错误的默认 URL

在 Docker 容器中，`localhost` 指的是容器本身，而不是宿主机或其他服务。因此：
- HTTP 请求 `http://localhost:8080/api/...` 失败
- WebSocket 连接 `ws://localhost:8080/api/ws` 失败

### 3. WebSocket URL 构建错误

当 `VITE_API_BASE_URL=/api` 时，WebSocket URL 被错误地构建为 `ws://<host>/api/api/ws`（重复了 `/api`）。

## 修复方案

### 1. 修改 Dockerfile

在 [`apps/frontend/Dockerfile`](apps/frontend/Dockerfile) 中添加构建参数支持：

```dockerfile
# 定义构建参数
ARG VITE_API_BASE_URL=/

# 创建 .env 文件用于构建时
RUN echo "VITE_API_BASE_URL=${VITE_API_BASE_URL}" > .env
```

这样可以在构建时通过 `--build-arg` 传入环境变量。

### 2. 更新 docker-compose.yml

在 [`docker-compose.yml`](docker-compose.yml) 中使用 `build.args` 传递构建参数：

```yaml
frontend:
  build:
    context: ./apps/frontend
    dockerfile: Dockerfile
    args:
      - VITE_API_BASE_URL=${VITE_API_BASE_URL:-/}
```

注意：
- 移除了 `environment` 配置（因为它只在运行时生效）
- 使用 `build.args` 在构建时传递参数
- 默认值必须是 `/`（根路径），不能是 `/api`，否则会导致 URL 重复（如 `/api/api/me`）

### 3. 修复 WebSocket URL 构建

在 [`apps/frontend/src/services/websocket.ts`](apps/frontend/src/services/websocket.ts) 中修复 URL 构建逻辑：

```typescript
if (apiBaseUrl.startsWith('/')) {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const host = window.location.host;
  // 如果 apiBaseUrl 是 '/' 或空，直接使用 /api/ws
  const basePath = apiBaseUrl === '/' ? '' : apiBaseUrl;
  wsUrl = `${protocol}//${host}${basePath}/api/ws?token=${encodeURIComponent(token)}&user_id=${userId}`;
}
```

这样确保 WebSocket URL 正确构建为 `ws://<host>/api/ws`。

## 工作原理

### 架构说明

```
用户浏览器
    ↓
nginx-proxy (端口 80)
    ├─ / → frontend:80 (静态文件)
    └─ /api/ → backend:8080 (API + WebSocket)
```

### 请求流程

1. **HTTP API 请求**：
   - 前端代码：`apiClient.get('/api/friends')`
   - 完整 URL：`http://<host>/api/friends`
   - nginx-proxy 代理到：`http://backend:8080/api/friends`

2. **WebSocket 连接**：
   - 前端代码构建：`ws://<host>/api/ws?token=...&user_id=...`
   - nginx-proxy 代理到：`ws://backend:8080/api/ws?token=...&user_id=...`

### 环境变量配置

| 环境 | VITE_API_BASE_URL | 说明 |
|------|-------------------|------|
| Docker | `/` | 使用根路径，由 nginx 代理到 `/api/` |
| 本地开发 | `http://localhost:8080` | 直接连接本地后端 |

**重要**：Docker 环境下必须使用 `/`，不能使用 `/api`，否则会导致 URL 重复（如 `/api/api/me`）。

## 使用方法

### Docker 部署

```bash
# 使用默认配置（VITE_API_BASE_URL=/）
docker-compose up --build

# 或自定义配置
VITE_API_BASE_URL=/ docker-compose up --build
```

### 本地开发

```bash
# 前端
cd apps/frontend
npm install
npm run dev

# 后端
cd apps/backend
go run cmd/server/main.go
```

本地开发时，前端会读取 `apps/frontend/.env` 文件中的配置。

## 验证修复

修复后，应该看到：

1. **环境变量正确注入**：
   ```
   [WebSocket] API Base URL: /
   [WebSocket] VITE_API_BASE_URL env: /
   ```

2. **HTTP 请求成功**：
   ```
   [axios] 请求拦截器 {method: 'GET', url: '/api/me', baseURL: 'http://localhost', fullURL: 'http://localhost/api/me', ...}
   [axios] 响应拦截器成功 {status: 200, data: {...}, url: '/api/me'}
   ```

3. **WebSocket 连接成功**：
   ```
   Connecting to WebSocket: ws://<host>/api/ws?token=...
   WebSocket connected
   ```

4. **后端日志显示正确的请求**：
   ```
   [GIN] 2026/03/07 - 19:28:40 | 200 | ... | 172.19.0.1 | GET "/api/me"
   ```

## 常见错误

### 错误 1：URL 重复 `/api/api/...`

如果看到以下错误：
```
[GIN] 2026/03/07 - 19:28:40 | 404 | ... | GET "/api/api/me"
```

**原因**：`VITE_API_BASE_URL` 被设置为 `/api`，导致 URL 重复。

**解决**：确保 `VITE_API_BASE_URL` 设置为 `/`（根路径），而不是 `/api`。

### 错误 2：环境变量未定义

如果看到以下错误：
```
[WebSocket] VITE_API_BASE_URL env: undefined
```

**原因**：Docker 镜像未使用 `--build-arg` 传递环境变量。

**解决**：
1. 确保 Dockerfile 中有 `ARG VITE_API_BASE_URL=/`
2. 确保 docker-compose.yml 中有 `build.args`
3. 重新构建镜像：`docker-compose build --no-cache frontend`

### 错误 3：Network Error

如果看到以下错误：
```
[axios] 响应拦截器错误 {message: 'Network Error', ...}
```

**原因**：前端尝试连接 `localhost:8080`，但在 Docker 容器中 `localhost` 指的是容器本身。

**解决**：确保 `VITE_API_BASE_URL` 设置为 `/`，让 nginx 代理处理请求。

## 相关文件

- [`apps/frontend/Dockerfile`](apps/frontend/Dockerfile) - 前端 Docker 镜像构建配置
- [`docker-compose.yml`](docker-compose.yml) - Docker Compose 编排配置
- [`apps/frontend/src/services/websocket.ts`](apps/frontend/src/services/websocket.ts) - WebSocket 服务
- [`apps/frontend/src/models/api.ts`](apps/frontend/src/models/api.ts) - API 客户端
- [`nginx-proxy.conf`](nginx-proxy.conf) - Nginx 反向代理配置

## 总结

问题的核心是 Vite 环境变量在构建时注入，而 Docker 的 `environment` 配置只在运行时生效。通过：

1. 在 Dockerfile 中使用 `ARG` 接收构建参数
2. 在构建前创建 `.env` 文件
3. 在 docker-compose.yml 中使用 `build.args` 传递参数
4. 修复 WebSocket URL 构建逻辑

成功解决了 Docker 网络连接问题。
