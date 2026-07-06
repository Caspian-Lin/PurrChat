---
name: bot_phase3_status
description: Bot Studio 工作流引擎重构状态和下一阶段计划
metadata:
  node_type: memory
  type: project
  originSessionId: 3f7fc27f-7ebd-483d-b1fc-50007fdfadd0
---

# Bot Studio 工作流引擎重构状态

## 已完成的 Phase

- Phase 0: 群聊管理 ✅
- Phase 1: Bot 基础系统 ✅
- Phase 2: 触发与回复系统 ✅
- Phase 3: 特殊模式/Agent 模式 ✅
- Phase 4: Bot 分享市场 + 系统消息 ✅
- Phase A: 命名重构 `special_mode` → `workflow` ✅ (2026-05-28)
- Phase B: 简单机制编译为工作流 ✅ (2026-05-28)
- Phase 0 (XState): 共享类型包 `@purrchat/workflow-types` ✅ (2026-05-29)
- Phase 1 (XState): Bot 微服务骨架 ✅ (2026-05-29)
- Phase C: Go 引擎 P0 Bug 修复 ✅ (2026-05-29)
- Phase D: 上下文增强 ✅ (2026-05-29)

## Phase 0 (XState) 已实现内容

- `packages/workflow-types/` 共享类型包
- 从 `types.ts` 和 `portTypes.ts` 抽取工作流相关类型
- 前端通过 `workspace:*` 引用

## Phase 1 (XState) 已实现内容

- `packages/workflow-engine/` 引擎包：NodeRegistry、Compiler、Runtime
- 7 个节点定义：trigger、end、reply、llm、if、builtin、wait
- `apps/bot-engine/` Hono 微服务：POST /execute、GET /health
- XState v5 `fromPromise` 模式处理异步节点

## Phase C 已修复的 P0 Bug

1. Trigger 节点不注入上下文值 → ExecuteFlow 注入端口值
2. Merge 节点空实现 → 输入端口追踪 + 计数器
3. Template 节点走错分支 → 分离为独立 case，读 in_input
4. 控制流端口名不一致 → out_exec 优先 + trigger fallback
5. Wait 节点语义错误 → 注入 out_exec 端口值

## Phase D 已实现内容

- HandleWorkflow 注入 username/time/sender_id 到 session.Variables
- ExecuteSimpleFlow 已有 username/time 注入

## 下一阶段计划

- Phase 2: 前端 DAG 编辑器对接 Bot 微服务
- Phase 3: 迁移剩余节点类型（loop、switch、merge、tool、dify、n8n、python、history）

## 详细架构文档

- `docs/bot-engine/ARCHITECTURE.md` — XState 迁移方案 + 节点→状态机映射
- `docs/bot-engine/WORKFLOW_REFACTOR_PLAN.md` — Phase A/B 实施计划
