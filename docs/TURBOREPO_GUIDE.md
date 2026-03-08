# Turborepo 配置指南

本文档详细说明 PurrChat 项目的 Turborepo 配置和使用方法。

## 目录

- [Turborepo 概述](#turborepo-概述)
- [项目结构](#项目结构)
- [配置文件说明](#配置文件说明)
- [任务配置](#任务配置)
- [缓存机制](#缓存机制)
- [常用命令](#常用命令)
- [最佳实践](#最佳实践)

## Turborepo 概述

Turborepo 是一个用于 JavaScript/TypeScript monorepo 的高性能构建系统，提供：

- **智能缓存**: 基于输入的增量构建
- **并行执行**: 自动并行化任务
- **任务依赖**: 定义任务之间的依赖关系
- **远程缓存**: 团队共享缓存
- **过滤**: 选择性运行特定包的任务

## 项目结构

```
PurrChat/
├── apps/
│   ├── frontend/           # 前端应用
│   │   └── package.json    # 前端包配置
│   └── backend/            # 后端应用
│       └── package.json    # 后端包配置
├── packages/              # 共享包（可选）
├── turbo.json             # Turborepo 配置
├── package.json           # 根包配置
└── pnpm-workspace.yaml    # pnpm 工作区配置
```

## 配置文件说明

### turbo.json

[`turbo.json`](turbo.json:1) 是 Turborepo 的核心配置文件。

#### 基本结构

```json
{
  "$schema": "https://turborepo.dev/schema.json",
  "ui": "tui",
  "globalEnv": ["环境变量列表"],
  "tasks": {
    "任务名": {
      "dependsOn": ["依赖任务"],
      "inputs": ["输入文件"],
      "outputs": ["输出文件"],
      "cache": "缓存配置",
      "persistent": "持久化配置",
      "env": ["环境变量"]
    }
  }
}
```

#### 配置项说明

| 配置项 | 类型 | 说明 |
|--------|------|------|
| `$schema` | string | JSON Schema 验证 |
| `ui` | string | UI 模式：`tui` (终端) 或 `stream` (流式) |
| `globalEnv` | string[] | 全局环境变量列表 |
| `tasks` | object | 任务配置对象 |

### package.json

根目录的 [`package.json`](package.json:1) 定义了 monorepo 的脚本和工作区。

#### 工作区配置

```json
{
  "workspaces": [
    "apps/*",
    "packages/*"
  ]
}
```

这告诉包管理器在 `apps/` 和 `packages/` 目录中查找工作区。

#### 脚本命令

```json
{
  "scripts": {
    "build": "turbo run build",
    "dev": "turbo run dev",
    "lint": "turbo run lint",
    "test": "turbo run test",
    "format": "prettier --write \"**/*.{ts,tsx,js,jsx,md,json,go}\"",
    "clean": "turbo run clean && rm -rf node_modules",
    "type-check": "turbo run type-check"
  }
}
```

## 任务配置

### build 任务

```json
{
  "build": {
    "dependsOn": ["^build"],
    "inputs": ["$TURBO_DEFAULT$", ".env*", "tsconfig.json", "vite.config.ts", "tailwind.config.js"],
    "outputs": ["dist/**", ".next/**", "!.next/cache/**", "bin/**"],
    "env": ["NODE_ENV"]
  }
}
```

**说明**:
- `dependsOn: ["^build"]`: 依赖所有依赖包的 build 任务
- `inputs`: 指定输入文件，`$TURBO_DEFAULT$` 是默认输入模式
- `outputs`: 指定输出文件，用于缓存
- `env`: 指定影响任务的环境变量

### dev 任务

```json
{
  "dev": {
    "cache": false,
    "persistent": true,
    "dependsOn": ["^build"]
  }
}
```

**说明**:
- `cache: false`: 禁用缓存，开发模式总是运行
- `persistent: true`: 任务会持续运行，不会自动退出
- `dependsOn: ["^build"]`: 先构建依赖包

### lint 任务

```json
{
  "lint": {
    "dependsOn": ["^lint"],
    "outputs": []
  }
}
```

**说明**:
- `outputs: []`: 没有输出文件，不缓存输出

### test 任务

```json
{
  "test": {
    "dependsOn": ["^build"],
    "outputs": ["coverage/**", "coverage.out", "coverage.html"],
    "inputs": ["$TURBO_DEFAULT$", "tests/**", "**/*.test.ts", "**/*.test.go"]
  }
}
```

**说明**:
- 依赖构建完成
- 缓存测试覆盖率报告
- 包含测试文件作为输入

### type-check 任务

```json
{
  "type-check": {
    "dependsOn": ["^type-check"],
    "outputs": []
  }
}
```

**说明**:
- 类型检查任务，无输出
- 依赖所有依赖包的类型检查

### clean 任务

```json
{
  "clean": {
    "cache": false
  }
}
```

**说明**:
- 清理任务，禁用缓存

## 缓存机制

### 缓存原理

Turborepo 使用以下因素计算缓存键：

1. **任务名称**: 任务标识
2. **输入文件**: 源代码、配置文件等
3. **环境变量**: 指定的环境变量
4. **全局配置**: turbo.json 配置
5. **依赖关系**: 任务依赖图

### 缓存策略

#### 本地缓存

默认启用，存储在 `node_modules/.cache/turbo` 目录。

```bash
# 清除本地缓存
turbo prune
rm -rf node_modules/.cache/turbo
```

#### 远程缓存

配置远程缓存，团队共享构建结果：

```bash
# 登录 Turborepo
turbo login

# 链接项目
turbo link

# 使用远程缓存
turbo run build
```

### 缓存失效

缓存会在以下情况下失效：

1. 输入文件发生变化
2. 环境变量发生变化
3. 配置文件发生变化
4. 依赖包发生变化

### 缓存优化

#### 减少缓存失效

```json
{
  "inputs": [
    "$TURBO_DEFAULT$",
    "!.env.local",
    "!.env.*.local",
    "!.git/**"
  ]
}
```

#### 指定精确输出

```json
{
  "outputs": [
    "dist/**",
    "!.dist/cache/**",
    "!.dist/temp/**"
  ]
}
```

## 常用命令

### 基本命令

```bash
# 运行所有包的构建任务
turbo run build

# 运行特定包的任务
turbo run build --filter=frontend

# 运行多个任务
turbo run build test lint

# 并行运行不依赖的任务
turbo run build --parallel
```

### 过滤命令

```bash
# 只运行 frontend 包的任务
turbo run build --filter=frontend

# 运行 frontend 及其依赖的任务
turbo run build --filter=frontend...

# 运行依赖 frontend 的包的任务
turbo run build --filter...frontend

# 运行所有包含 "app" 的包的任务
turbo run build --filter=*app*

# 运行特定目录的任务
turbo run build --filter=./apps/*
```

### 缓存命令

```bash
# 强制重新运行，忽略缓存
turbo run build --force

# 只运行缓存未命中的任务
turbo run build --dry-run

# 清除缓存
turbo prune
```

### 调试命令

```bash
# 显示任务依赖图
turbo run build --graph

# 显示任务执行计划
turbo run build --dry-run --verbose

# 显示缓存统计
turbo run build --cache-dir=./cache
```

## 最佳实践

### 1. 合理定义任务依赖

```json
{
  "build": {
    "dependsOn": ["^build"]
  },
  "test": {
    "dependsOn": ["^build"]
  },
  "lint": {
    "dependsOn": ["^lint"]
  }
}
```

### 2. 精确指定输入输出

```json
{
  "build": {
    "inputs": [
      "$TURBO_DEFAULT$",
      "src/**",
      "package.json",
      "tsconfig.json",
      "vite.config.ts"
    ],
    "outputs": ["dist/**"]
  }
}
```

### 3. 使用环境变量

```json
{
  "globalEnv": ["NODE_ENV", "API_URL"],
  "tasks": {
    "build": {
      "env": ["NODE_ENV"]
    }
  }
}
```

### 4. 禁用持久化任务的缓存

```json
{
  "dev": {
    "cache": false,
    "persistent": true
  }
}
```

### 5. 使用过滤提高效率

```bash
# 只运行变更包的任务
turbo run build --filter=[HEAD^1]

# 只运行特定包的任务
turbo run build --filter=frontend
```

### 6. 配置远程缓存

```bash
# 登录并链接
turbo login
turbo link

# 在 CI/CD 中使用
turbo run build --token=$TURBO_TOKEN --team=$TURBO_TEAM
```

### 7. 使用 Dry Run 预览

```bash
# 预览将要执行的任务
turbo run build --dry-run

# 查看详细的执行计划
turbo run build --dry-run --verbose
```

### 8. 监控缓存命中率

```bash
# 查看缓存统计
turbo run build --cache-stats

# 导出缓存报告
turbo run build --cache-dir=./cache --export=cache.tgz
```

## 性能优化

### 1. 并行执行

```bash
# 并行运行不依赖的任务
turbo run build --parallel
```

### 2. 增量构建

```bash
# 只构建变更的包
turbo run build --filter=[HEAD^1]
```

### 3. 缓存预热

```bash
# 在 CI 中预热缓存
turbo run build --force
```

### 4. 减少输入文件

```json
{
  "inputs": [
    "$TURBO_DEFAULT$",
    "!.git/**",
    "!.env.local",
    "!node_modules/**"
  ]
}
```

## 故障排查

### 缓存问题

**问题**: 缓存未命中

**解决方案**:

```bash
# 清除缓存
turbo prune

# 强制重新运行
turbo run build --force

# 检查输入文件
turbo run build --dry-run --verbose
```

### 依赖问题

**问题**: 任务依赖循环

**解决方案**:

```json
{
  "build": {
    "dependsOn": ["^build"]
  }
}
```

确保依赖关系是 DAG（有向无环图）。

### 性能问题

**问题**: 构建速度慢

**解决方案**:

```bash
# 使用远程缓存
turbo login && turbo link

# 并行执行
turbo run build --parallel

# 增量构建
turbo run build --filter=[HEAD^1]
```

## 参考资料

- [Turborepo 官方文档](https://turbo.build/repo/docs)
- [Turborepo 配置参考](https://turbo.build/repo/docs/reference/configuration)
- [Turborepo CLI 参考](https://turbo.build/repo/docs/reference/command-line-reference)
- [Monorepo 最佳实践](https://turbo.build/repo/docs/core-concepts/monorepos)
