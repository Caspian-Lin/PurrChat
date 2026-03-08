# 多客户端部署指南

本文档说明如何配置和部署 PurrChat 的多客户端架构，支持网页端、Tauri 桌面端和未来可能的移动端。

## 目录

- [架构概述](#架构概述)
- [环境配置](#环境配置)
- [构建和部署](#构建和部署)
- [Nginx 配置](#nginx-配置)
- [WebSocket 配置](#websocket-配置)
- [常见问题](#常见问题)

## 架构概述

PurrChat 支持多种客户端类型，每种客户端通过不同的方式连接到后端 API：

| 客户端类型 | API 配置 | WebSocket 配置 | 部署方式 |
|-----------|---------|---------------|---------|
| 网页端 | 相对路径 `/` | 相对路径 `/api/ws` | 通过 Nginx 反向代理 |
| Tauri 桌面端 | 绝对 URL | 绝对 URL | 独立安装包 |
| 移动端 | 绝对 URL | 绝对 URL | 应用商店分发 |

### 环境变量

- `VITE_API_BASE_URL`: API 基础 URL
- `VITE_APP_ENV`: 环境标识 (`development` | `production`)
- `VITE_APP_CLIENT`: 客户端类型 (`web` | `tauri` | `mobile`)

## 环境配置

### 1. 开发环境

文件：`.env.development`

```bash
# API Base URL - 本地开发环境
VITE_API_BASE_URL=http://localhost:8080

# 环境标识
VITE_APP_ENV=development
```

使用方式：
```bash
pnpm dev
```

### 2. 生产环境 - 网页端

文件：`.env.production`

```bash
# API Base URL - 生产环境（网页端使用相对路径，通过 nginx 反向代理）
VITE_API_BASE_URL=/

# 环境标识
VITE_APP_ENV=production
VITE_APP_CLIENT=web
```

使用方式：
```bash
pnpm run build:prod
```

### 3. Tauri 桌面端

文件：`.env.tauri`

```bash
# API Base URL - Tauri 桌面端（连接到服务器）
# 请将此地址替换为您的实际服务器地址
VITE_API_BASE_URL=https://your-server.com

# 环境标识
VITE_APP_ENV=production
VITE_APP_CLIENT=tauri
```

使用方式：
```bash
pnpm run tauri:build:win64
```

### 4. 移动端（未来）

文件：`.env.mobile`

```bash
# API Base URL - 移动端（连接到服务器）
# 请将此地址替换为您的实际服务器地址
VITE_API_BASE_URL=https://your-server.com

# 环境标识
VITE_APP_ENV=production
VITE_APP_CLIENT=mobile
```

使用方式：
```bash
pnpm run build:mobile
```

## 构建和部署

### 网页端部署

1. **构建生产版本**：
   ```bash
   cd apps/frontend
   pnpm run build:prod
   ```

2. **部署到服务器**：
   ```bash
   # 将 dist 目录上传到服务器
   scp -r dist/* user@server:/var/www/purrchat/
   ```

3. **配置 Nginx**（见下文）

### Tauri 桌面端部署

1. **配置服务器地址**：
   编辑 `.env.tauri`，设置正确的服务器地址：
   ```bash
   VITE_API_BASE_URL=https://your-server.com
   ```

2. **构建 Windows 版本**：
   ```bash
   cd apps/frontend
   pnpm run tauri:build:win64
   ```

3. **分发安装包**：
   - 可执行文件：`src-tau/target/x86_64-pc-windows-msvc/release/purr-chat.exe`
   - 安装程序：`src-tau/target/x86_64-pc-windows-msvc/release/bundle/nsis/Purr Chat_0.0.0_x64-setup.exe`

### 移动端部署（未来）

1. **配置服务器地址**：
   编辑 `.env.mobile`，设置正确的服务器地址

2. **构建移动应用**：
   ```bash
   cd apps/frontend
   pnpm run build:mobile
   ```

3. **打包为移动应用**：
   使用相应的工具（如 Capacitor、React Native 等）打包

## Nginx 配置

### 基本配置

```nginx
server {
    listen 80;
    server_name your-server.com;

    # 前端静态文件
    location / {
        root /var/www/purrchat;
        try_files $uri $uri/ /index.html;
    }

    # 后端 API 反向代理
    location /api/ {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # WebSocket 反向代理
    location /api/ws {
        proxy_pass http://localhost:8080/api/ws;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 86400;
    }
}
```

### HTTPS 配置（推荐）

```nginx
server {
    listen 443 ssl http2;
    server_name your-server.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    # 前端静态文件
    location / {
        root /var/www/purrchat;
        try_files $uri $uri/ /index.html;
    }

    # 后端 API 反向代理
    location /api/ {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # WebSocket 反向代理
    location /api/ws {
        proxy_pass http://localhost:8080/api/ws;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 86400;
    }
}

# HTTP 重定向到 HTTPS
server {
    listen 80;
    server_name your-server.com;
    return 301 https://$server_name$request_uri;
}
```

## WebSocket 配置

### 网页端

网页端使用相对路径，通过 Nginx 反向代理连接到 WebSocket：

```javascript
// 自动使用当前协议和主机
const wsUrl = `${protocol}//${host}/api/ws?token=${token}&user_id=${userId}`;
```

### Tauri 桌面端

Tauri 桌面端使用绝对 URL 直接连接到服务器：

```javascript
// 使用配置的服务器地址
const wsUrl = `wss://your-server.com/api/ws?token=${token}&user_id=${userId}`;
```

### 移动端

移动端使用绝对 URL 直接连接到服务器：

```javascript
// 使用配置的服务器地址
const wsUrl = `wss://your-server.com/api/ws?token=${token}&user_id=${userId}`;
```

## 常见问题

### 1. 如何更改服务器地址？

**网页端**：
- 修改 Nginx 配置中的 `proxy_pass` 地址

**Tauri 桌面端**：
- 修改 `.env.tauri` 文件中的 `VITE_API_BASE_URL`
- 重新构建应用

**移动端**：
- 修改 `.env.mobile` 文件中的 `VITE_API_BASE_URL`
- 重新构建应用

### 2. WebSocket 连接失败

**检查项**：
1. 确认后端 WebSocket 服务正常运行
2. 检查 Nginx 配置中的 WebSocket 代理设置
3. 确认防火墙允许 WebSocket 连接
4. 检查浏览器控制台的错误信息

**常见错误**：
- `WebSocket connection failed`: 检查后端服务是否运行
- `404 Not Found`: 检查 Nginx 配置中的路径是否正确
- `Connection refused`: 检查后端端口是否正确

### 3. 如何在本地测试生产环境配置？

1. **启动后端服务**：
   ```bash
   cd apps/backend
   go run cmd/server/main.go
   ```

2. **启动前端开发服务器**：
   ```bash
   cd apps/frontend
   pnpm dev
   ```

3. **配置本地 Nginx**（可选）：
   参考 [Nginx 配置](#nginx-配置) 部分

### 4. 如何添加新的客户端类型？

1. **创建新的环境配置文件**：
   ```bash
   # .env.newclient
   VITE_API_BASE_URL=https://your-server.com
   VITE_APP_ENV=production
   VITE_APP_CLIENT=newclient
   ```

2. **添加构建脚本**：
   在 `package.json` 中添加：
   ```json
   "build:newclient": "vue-tsc -b && vite build --mode newclient"
   ```

3. **更新配置工具**：
   在 `src/config/app.ts` 中添加新的客户端类型判断

### 5. 如何处理 CORS 问题？

**网页端**：
- 使用相对路径，通过 Nginx 反向代理，无需处理 CORS

**Tauri 桌面端和移动端**：
- 后端需要配置 CORS 允许跨域请求
- 在 Go 后端中添加 CORS 中间件

## 总结

PurrChat 的多客户端架构通过环境变量和配置工具实现了灵活的部署方式：

1. **网页端**：使用相对路径，通过 Nginx 反向代理
2. **Tauri 桌面端**：使用绝对 URL，直接连接到服务器
3. **移动端**：使用绝对 URL，直接连接到服务器

这种架构使得同一套前端代码可以适配多种客户端类型，只需通过环境变量进行配置即可。
