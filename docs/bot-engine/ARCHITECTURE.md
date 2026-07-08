# Bot 引擎架构：评估、设计与路线图

> 最后更新：2026-05-28
> 状态：设计中

---

## 一、现有引擎审查结论

对 `botengine/` 包（约 1400 行 Go 代码）的全面审查发现以下问题：

### P0 级 Bug

| 问题                       | 影响                       |
| -------------------------- | -------------------------- |
| trigger 节点不注入上下文值 | 所有依赖上游数据的节点失效 |
| merge 节点空实现           | 多分支汇聚场景完全不可用   |
| template 节点走错分支      | template 节点抛异常        |
| 控制流端口名不一致         | 某些连线配置下流程中断     |
| wait 节点语义错误          | 等待功能形同虚设           |

### 架构问题

| 问题               | 描述                                                      |
| ------------------ | --------------------------------------------------------- |
| 500 行 switch-case | `followControlFlow` 巨型 switch，新增节点类型需改引擎核心 |
| 递归深度风险       | 纯递归执行，嵌套 loop + if 可能栈溢出                     |
| 无并发控制         | 同会话多条消息并发执行导致数据竞争                        |
| 状态无持久化       | sync.Map 内存存储，重启丢失所有会话                       |
| 上下文贫乏         | BotMessage 仅 5 字段，SenderName 未使用                   |

### 命名问题

"特殊模式"（special_mode）命名不直观，已重构为"工作流"（workflow）。

---

## 二、框架评估

### 候选框架对比

| 框架                | GitHub Stars | npm 周下载 | 零依赖                   | 评估                   |
| ------------------- | ------------ | ---------- | ------------------------ | ---------------------- |
| **Temporal TS SDK** | ~12K         | **1.2M**   | 否（需 Temporal Server） | 过重，需额外基础设施   |
| **XState**          | **29,380**   | ~500K      | **是**                   | **推荐**               |
| **OpenWorkflow**    | ~200         | **4.1K**   | **是**                   | 有潜力但较新           |
| flowcraft           | ~50          | 771        | 是                       | 过于小众               |
| pi 架构             | 各项目不同   | N/A        | 否                       | 不适用（编程代理框架） |

### 排除的选项

**Temporal**：需要运行 Temporal Server（独立 Go 服务 + 数据库），对 PurrChat 来说运维成本过高。

**pi 架构**：是 Pi 编程代理的多代理编排框架（mavam/pi-agents、@davidorex/pi-workflows 等），专门用于协调多个 Pi 子进程完成代码任务，不是通用工作流引擎。每次执行都要 spawn 子进程，不适合聊天机器人的实时响应场景。

**flowcraft**：过于小众（771 周下载，1-2 人维护），文档和社区支持有限。

### 选定方案：XState + 自建工作流抽象层

**XState**（29K stars，350 贡献者，484 版本）是状态机/状态图库，提供强大的底层原语：

| XState 原语       | PurrChat 映射                     |
| ----------------- | --------------------------------- |
| State Machine     | Blueprint（工作流定义）           |
| Actor             | 每个 Bot 会话的运行实例           |
| Context           | 端口值 + 会话变量                 |
| Services (invoke) | 异步节点执行（LLM、HTTP、Python） |
| Actions           | 副作用（发送回复、写入变量）      |
| Guards            | 条件分支（IF 节点）               |
| Parallel States   | 并行分支 + Merge                  |
| Events            | 用户消息、等待唤醒                |

**核心优势**：

- 29K stars，极其成熟，零依赖
- 原生 Vue 支持（`@xstate/vue`）
- 官方可视化编辑器（Stately）
- Actor 模型天然适合"每个 Bot 会话是一个 Actor"
- 内建 snapshot 持久化 → 崩溃恢复
- 内建 Inspector 调试工具

---

## 三、架构设计

### 分层架构

```
┌─────────────────────────────────────────────────┐
│           前端 DAG 编辑器 (Vue Flow)              │
│  可视化节点拖拽 → Blueprint JSON                   │
└──────────────────────┬──────────────────────────┘
                       │ Blueprint JSON
┌──────────────────────▼──────────────────────────┐
│         @purrchat/workflow-types                  │
│  NodeDefinition / PortType / Blueprint / ...      │
└──────────────────────┬──────────────────────────┘
                       │ Blueprint → XState Machine
┌──────────────────────▼──────────────────────────┐
│         @purrchat/workflow-engine                 │
│  NodeRegistry / Compiler / Runtime / Session      │
│  ┌─────────────────────────────────────────────┐ │
│  │  XState (底层状态管理引擎)                    │ │
│  │  State Machine / Actor / Context / Actions   │ │
│  └─────────────────────────────────────────────┘ │
└──────────────────────┬──────────────────────────┘
                       │ HTTP / gRPC
┌──────────────────────▼──────────────────────────┐
│         Bot 微服务 (Node.js / TS)                 │
│  接收消息 → 查找会话 Actor → 发送事件 → 获取回复   │
└─────────────────────────────────────────────────┘
```

### 核心抽象 1：NodeDefinition（节点定义）

替代当前 Go 代码中的 500 行 switch-case。每种节点类型是一个独立的 `NodeDefinition`：

```typescript
import { z } from 'zod';

export interface NodeDefinition<
  TInput extends Record<string, any> = Record<string, any>,
  TOutput extends Record<string, any> = Record<string, any>,
  TConfig extends Record<string, any> = Record<string, any>,
> {
  type: string; // "llm" | "reply" | "if" | ...
  label: string; // 显示名称
  category: 'trigger' | 'processing' | 'control' | 'output' | 'integration';
  icon: string;
  color: string;
  ports: {
    inputs: PortDefinition[];
    outputs: PortDefinition[];
  };
  configSchema: z.ZodType<TConfig>; // Zod schema，前端自动生成表单
  execute: (input: TInput, config: TConfig, ctx: ExecutionContext) => Promise<TOutput>;
}
```

**新增节点类型 = 新增一个文件，不修改引擎代码。**

### 核心抽象 2：NodeRegistry（节点注册表）

```typescript
export class NodeRegistry {
  private nodes = new Map<string, NodeDefinition>();

  register(def: NodeDefinition): void {
    this.nodes.set(def.type, def);
  }
  registerAll(defs: NodeDefinition[]): void {
    for (const d of defs) this.register(d);
  }
  get(type: string): NodeDefinition | undefined {
    return this.nodes.get(type);
  }
  getAll(): NodeDefinition[] {
    return Array.from(this.nodes.values());
  }
  getByCategory(cat: string): NodeDefinition[] {
    return this.getAll().filter((n) => n.category === cat);
  }
}
```

### 核心抽象 3：Blueprint（工作流定义）

纯 JSON，可存入 PostgreSQL JSONB：

```typescript
export interface Blueprint {
  id: string;
  name: string;
  nodes: BlueprintNode[];
  connections: BlueprintConnection[];
  endConditions: EndCondition[];
}

export interface BlueprintNode {
  id: string;
  type: string; // 对应 NodeDefinition.type
  name: string;
  config: Record<string, any>;
  position: { x: number; y: number };
}

export interface BlueprintConnection {
  id: string;
  source: { nodeId: string; portId: string };
  target: { nodeId: string; portId: string };
}
```

### 核心抽象 4：Compiler（Blueprint → XState Machine）

将 Blueprint 编译为 XState 状态机。核心流程：

1. 拓扑排序节点
2. 为每个节点创建 XState 状态
3. 根据连线创建状态转换
4. 根据节点类型选择 invoke（异步处理）或 guard（条件分支）

### 核心抽象 5：Runtime（运行时）

```typescript
export class WorkflowRuntime {
  private registry: NodeRegistry;
  private sessions = new Map<string, ReturnType<typeof createActor>>();

  /** 单次执行（无状态，用于 simple 机制） */
  async execute(blueprint: Blueprint, input: string): Promise<string> {
    /* ... */
  }

  /** 创建持久化会话（多轮对话） */
  createSession(sessionId: string, blueprint: Blueprint): void {
    /* ... */
  }

  /** 向会话发送消息 */
  async sendMessage(sessionId: string, input: string): Promise<string> {
    /* ... */
  }

  /** 停止会话 */
  destroySession(sessionId: string): void {
    /* ... */
  }
}
```

---

## 四、节点 → XState 状态机映射

### 处理类节点（llm / tool / dify / n8n / python / builtin / template）

特征：执行异步操作，然后沿控制流继续。

映射为 **invoke（异步调用）**：

```
状态图：  ┌──────────┐     invoke      ┌──────────┐
         │ llm 节点 │ ──────────────→ │ 下一节点 │
         └──────────┘   onDone: 写入   └──────────┘
                          输出端口值
                           ↓ onError
                       ┌──────────┐
                       │  __error │
                       └──────────┘
```

```typescript
states['llm-1'] = {
  invoke: {
    src: 'executeNode',
    input: ({ context }) => ({
      nodeId: 'llm-1',
      config: { api_url: '...', model: '...' },
      input: resolveInputPorts('llm-1', connections, context),
    }),
    onDone: {
      actions: assign({
        nodeOutputs: ({ context, event }) => ({
          ...context.nodeOutputs,
          'llm-1': event.output,
        }),
      }),
      target: 'reply-1', // 沿连线跳转
    },
    onError: { target: '__error' },
  },
};
```

`invoke` 天然处理异步操作（LLM 调用可能需要 10 秒），XState 自动管理等待、超时、取消。

### 控制流类节点（if / switch）

特征：不做异步操作，根据条件决定跳转。

映射为 **guard（守卫）+ 多个转换**：

```
                  ┌───────────┐
              ┌──→│ llm (true)│──→ reply-2
┌──────────┐  │  └───────────┘
│ if 节点  │──┤
└──────────┘  │  ┌───────────────┐
              └──→│ reply (false) │──→ end
                  └───────────────┘
```

```typescript
states['if-1'] = {
  entry: assign({
    nodeOutputs: ({ context }) => {
      const left = resolvePort('if-1', 'in_left', context);
      const right = resolvePort('if-1', 'in_right', context);
      const result = evaluate(left, '==', right);
      return { ...context.nodeOutputs, 'if-1': { result } };
    },
  }),
  always: [
    { guard: ({ context }) => context.nodeOutputs['if-1']?.result === true, target: 'llm-2' },
    { guard: ({ context }) => context.nodeOutputs['if-1']?.result === false, target: 'reply-3' },
  ],
};
```

**SWITCH 节点**同理，多个 guard 对应多个分支。

### 循环节点（loop）

映射为 **自循环 + guard 退出条件**：

```
┌──────────┐    ┌────────────┐    ┌──────────┐
│ loop 入口│───→│ 循环体节点 │───→│ loop 入口│  (自循环)
└──────────┘    └────────────┘    └──────────┘
     │                                 ↑
     └──[超过最大次数]──→ 下一节点 ─────┘
```

```typescript
states['loop-1'] = {
  entry: assign({
    loopState: ({ context }) => ({
      iteration: context.loopState?.iteration ?? 0,
      maxIterations: 10,
    }),
  }),
  always: [
    {
      guard: ({ context }) => context.loopState.iteration >= context.loopState.maxIterations,
      target: 'next-node',
    },
    { target: 'loop-body-first-node' },
  ],
};

// 循环体最后一节点跳回 loop-1
states['loop-body-last'] = {
  invoke: {
    src: 'executeNode',
    onDone: {
      actions: assign({
        loopState: ({ context }) => ({
          ...context.loopState,
          iteration: context.loopState.iteration + 1,
        }),
      }),
      target: 'loop-1', // 自循环
    },
  },
};
```

### 等待节点（wait）

映射为 **等待外部事件的状态**：

```
┌──────────┐    等待 USER_MESSAGE    ┌──────────┐
│ wait 节点│ ──────────────────────→ │ 下一节点 │
└──────────┘   (状态机暂停)           └──────────┘
```

```typescript
states['wait-1'] = {
  on: {
    USER_MESSAGE: {
      actions: assign({
        nodeOutputs: ({ context, event }) => ({
          ...context.nodeOutputs,
          'wait-1': { userInput: event.input },
        }),
      }),
      target: 'next-node',
    },
  },
};
```

这是 XState 的杀手级优势——原生支持"暂停等待事件"。当前 Go 引擎的 wait 节点是伪实现（直接取当前消息），XState 可以真正实现"工作流暂停，等用户下一条消息后再继续"。

### 合并节点（merge）

映射为 **parallel state（并行状态）+ onDone**：

```
               ┌──→ branch-a: node-a1 → node-a2 ──┐
parallel-region│                                    ├──→ merge-1 → next
               └──→ branch-b: node-b1 → node-b2 ──┘
                     (两个分支都完成后才继续)
```

```typescript
states['parallel-region'] = {
  type: 'parallel',
  states: {
    branch_a: {
      initial: 'node-a1',
      states: {
        /* ... */
      },
    },
    branch_b: {
      initial: 'node-b1',
      states: {
        /* ... */
      },
    },
  },
  onDone: { target: 'merge-1' },
};
```

### 输出类节点（reply / end）

**Reply 节点**：同步 action（构建回复内容）+ 副作用（发送回复）：

```typescript
states['reply-1'] = {
  entry: [
    assign({
      finalReply: ({ context }) => {
        return replaceVariables('{llm-1.response}', context.nodeOutputs);
      },
    }),
    ({ context }) => {
      context.sendReply(context.finalReply);
    },
  ],
  always: { target: 'end-1' },
};
```

**End 节点**：终态：

```typescript
states['end-1'] = { type: 'final' };
```

### 完整示例：简单工作流的状态机

Blueprint：

```
trigger → llm → reply → end
```

编译后的 XState 状态机：

```
┌─────────┐     ┌──────────────┐     ┌──────────────┐     ┌──────┐
│ trigger │ ──→ │ llm (invoke) │ ──→ │ reply (entry)│ ──→ │ end  │
│ (idle)  │     │ GPT-4 调用   │     │ 发送回复     │     │ final│
└─────────┘     └──────────────┘     └──────────────┘     └──────┘
                  ↓ onError
                ┌──────────┐
                │  __error │
                └──────────┘
```

运行时流程：

1. 用户发消息 → 创建 Actor，进入 `trigger` 状态
2. `trigger` 立即跳转到 `llm`（invoke 异步调用 GPT-4）
3. GPT-4 返回 → 输出写入 context，跳转到 `reply`
4. `reply` 的 entry action 发送回复，跳转到 `end`
5. `end` 是 final 状态，Actor 结束

---

## 五、与当前 Go 引擎的对比

| 维度       | 当前 Go 引擎         | XState + 抽象层                |
| ---------- | -------------------- | ------------------------------ |
| 节点注册   | 500 行 switch-case   | `registry.register(node)`      |
| 新增节点   | 修改引擎核心文件     | 新增一个文件                   |
| 条件分支   | 手写 if/else 逻辑    | XState guards（声明式）        |
| 循环       | 手写 for 循环        | XState 自循环 + guard          |
| 并发       | 无                   | XState parallel states         |
| 等待       | 伪实现（取当前消息） | 原生事件等待（状态机暂停）     |
| 状态持久化 | sync.Map（内存）     | XState snapshot → JSON         |
| 崩溃恢复   | 无（重启丢失）       | 从 snapshot 重建 actor         |
| 调试       | 自建 trace           | XState Inspector（官方工具）   |
| 可视化     | 自建 DAG 编辑器      | Stately 编辑器 + Vue Flow      |
| 类型安全   | Go 强类型            | TypeScript + Zod schema        |
| 测试       | 需要 mock 整个引擎   | 纯函数测试节点 + snapshot 断言 |

---

## 六、实施路线图

### Phase 0：抽取共享类型包 `@purrchat/workflow-types`（零风险）

**目标**：从现有前端代码中抽取工作流相关类型为独立包，前后端共享。

**收益**：

- 前后端节点类型定义一处修改、处处生效
- 新增节点类型时类型定义自动同步
- 为后续微服务拆分打下基础
- 零运行时风险——纯类型定义，不改变任何执行逻辑

**包结构**：

```
packages/
  workflow-types/
    src/
      index.ts              ← 统一导出
      nodes.ts              ← 节点类型定义（NodeDefinition）
      ports.ts              ← 端口类型定义（PortType, PortDefinition）
      connections.ts        ← 连线定义（Connection）
      blueprint.ts          ← 工作流序列化格式（Blueprint）
      triggers.ts           ← 触发条件类型（TriggerSpec, TriggerRule）
      mechanisms.ts         ← 机制配置类型（Mechanism, ReplySpec）
      end-conditions.ts     ← 结束条件类型（EndCondition）
      execution.ts          ← 执行上下文类型（ExecutionContext）
      debug.ts              ← 调试类型（DebugSession, EventTrace）
      system-messages.ts    ← 系统消息类型
      node-types/           ← 每种节点类型的配置 schema
        trigger.ts / llm.ts / reply.ts / builtin.ts / python.ts
        if.ts / loop.ts / switch.ts / merge.ts / tool.ts
        dify.ts / n8n.ts / wait.ts / history.ts / template.ts / end.ts
    package.json            ← name: "@purrchat/workflow-types"
```

**不涉及的变更**：不改变运行时逻辑、API、数据库、前端 UI。

### Phase 1：搭建 Bot 微服务骨架

**前提**：Phase 0 完成，类型包结构稳定。

**目标**：创建独立的 TS Bot 服务，基于 XState + 工作流抽象层。

**实施步骤**：

1. 创建 `apps/bot-engine/`（Node.js + TypeScript）
2. 安装 `xstate` 依赖
3. 实现 NodeRegistry + Compiler + Runtime
4. 逐个迁移节点实现（从 Go 代码翻译为 TS）
5. 实现 Bot 服务 API（execute / debug / debug/step）
6. Go 后端 Bot 回复改为调用 Bot 服务
7. 灰度切换，保留 Go 引擎作为 fallback

### Phase 2：前端对接

**目标**：前端 DAG 编辑器和调试面板改为对接 Bot 微服务。

**收益**：

- 节点类型定义从 `@purrchat/workflow-types` 自动读取
- 配置面板根据 `configSchema`（Zod）自动生成
- 调试通过 WebSocket 实时流式获取执行状态

### 原 Phase A-D（命名重构 + 简单机制编译）

这些是当前 Go 引擎的渐进改进，在微服务迁移完成前仍然需要：

- **Phase A**：命名重构 `special_mode` → `workflow`（✅ 已完成）
- **Phase B**：简单机制编译为工作流（✅ 已完成）
- **Phase C**：修复 P0 Bug（待实现）
- **Phase D**：上下文增强（待实现）

如果微服务迁移推进顺利，Phase C/D 可以跳过（TS 引擎天然不存在这些 bug）。

---

## 七、决策记录

| 决策               | 选项                         | 结论            | 理由                                                     |
| ------------------ | ---------------------------- | --------------- | -------------------------------------------------------- |
| 是否引入 pi 架构   | 是/否                        | **否**          | 编程代理框架，不适合聊天机器人场景                       |
| 是否引入 flowcraft | 是/否                        | **否**          | 过于小众（771 周下载）                                   |
| 是否引入 Temporal  | 是/否                        | **否**          | 需要额外基础设施（Temporal Server）                      |
| 底层引擎选型       | XState / OpenWorkflow / 自建 | **XState**      | 29K stars，零依赖，原生 Vue 支持，Actor 模型适合会话管理 |
| 是否抽取类型包     | 是/否                        | **是，Phase 0** | 零风险、立即收益、为未来打基础                           |
| 微服务拆分时机     | 现在/Phase 1/不拆            | **Phase 1**     | 先通过类型包验证方向                                     |
