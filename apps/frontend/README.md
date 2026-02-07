# PurrChat Frontend

PurrChat 前端应用，基于 Vue 3 + Vite + Naive UI + Tauri 构建。

## 技术栈

- **框架**: Vue 3.5+ (Composition API)
- **构建工具**: Vite 7.2+
- **UI 组件库**: Naive UI 2.43+
- **状态管理**: Pinia 3.0+
- **路由**: Vue Router 4.6+
- **桌面应用**: Tauri 2.9+
- **样式**: Tailwind CSS 3.4+
- **测试**: Vitest 2.1+ + Vue Test Utils

## 开发命令

```bash
# 安装依赖
pnpm install

# 启动开发服务器
pnpm dev

# 构建生产版本
pnpm build

# 预览生产构建
pnpm preview

# 运行测试
pnpm test

# 测试覆盖率
pnpm test:coverage

# 代码检查
pnpm lint

# 类型检查
pnpm type-check

# Tauri 开发
pnpm tauri:dev

# Tauri 构建
pnpm tauri:build
```

## 项目结构

```
frontend/
├── src/
│   ├── assets/          # 静态资源
│   ├── components/      # Vue 组件
│   ├── config/          # 配置文件
│   ├── controllers/     # 控制器
│   ├── models/          # 类型定义
│   ├── stores/          # Pinia 状态管理
│   ├── tests/           # 测试文件
│   ├── views/           # 页面视图
│   ├── App.vue          # 根组件
│   └── main.ts          # 入口文件
├── public/              # 公共资源
├── src-tau/             # Tauri 源码
├── index.html           # HTML 模板
├── vite.config.ts       # Vite 配置
├── tailwind.config.js   # Tailwind 配置
└── tsconfig.json        # TypeScript 配置
```

## 环境变量

在 `.env` 文件中配置：

```env
VITE_API_BASE_URL=http://localhost:8080
```

## Docker 构建

```bash
# 构建镜像
docker build -t purrchat-frontend .

# 运行容器
docker run -p 80:80 purrchat-frontend
```

## 测试

项目使用 Vitest 进行单元测试：

```bash
# 运行所有测试
pnpm test

# 运行测试并生成覆盖率报告
pnpm test:coverage

# 运行测试 UI
pnpm test:ui
```

## Tauri 桌面应用

PurrChat 支持打包为桌面应用：

```bash
# 开发模式
pnpm tauri:dev

# 构建桌面应用
pnpm tauri:build
```

支持平台：
- Windows (x86_64, i686)
- macOS
- Linux

## 注意事项

1. 确保 Node.js 版本 >= 18
2. 前端默认运行在 http://localhost:5173
3. API 基础 URL 通过环境变量 `VITE_API_BASE_URL` 配置
4. 构建前运行类型检查：`pnpm type-check`
