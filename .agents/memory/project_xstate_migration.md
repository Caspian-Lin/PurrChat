---
name: xstate_migration
description: Bot 引擎 XState 微服务迁移计划和状态
metadata: 
  node_type: memory
  type: project
  originSessionId: 3f7fc27f-7ebd-483d-b1fc-50007fdfadd0
---

# Bot 引擎 XState 微服务迁移

## 背景

当前 Go 引擎存在架构问题（500 行 switch-case、递归深度风险、无并发控制、状态无持久化）。评估后选定 XState + 自建工作流抽象层作为目标架构。

## 框架选型

- **选定**: XState（29K stars，零依赖，原生 Vue 支持，Actor 模型）
- **排除**: Temporal（需额外基础设施）、pi 架构（编程代理框架）、flowcraft（过于小众）

## 实施阶段

### Phase 0：抽取共享类型包 `@purrchat/workflow-types`（1-2 天）
- 从 `apps/frontend/src/models/types.ts` 抽取工作流相关类型
- 创建 `packages/workflow-types/` 独立包
- 前后端共享类型定义

### Phase 1：搭建 Bot 微服务骨架（2-3 周）
- 创建 `apps/bot-engine/`（Node.js + TypeScript）
- 实现 NodeRegistry + Compiler + Runtime
- 逐个迁移节点实现
- Go 后端改为调用 Bot 服务

### Phase 2：前端对接
- DAG 编辑器对接 Bot 微服务
- 配置面板根据 Zod schema 自动生成
- 调试通过 WebSocket 实时流式获取

## 核心抽象

1. **NodeDefinition**：节点定义（替代 500 行 switch-case）
2. **NodeRegistry**：节点注册表
3. **Blueprint**：工作流定义（纯 JSON，存入 PostgreSQL JSONB）
4. **Compiler**：Blueprint → XState Machine
5. **Runtime**：运行时（execute / createSession / sendMessage）

## 详细设计

见 `docs/bot-engine/ARCHITECTURE.md`
