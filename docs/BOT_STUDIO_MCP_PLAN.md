# Bot Studio MCP Server — 实现计划

> 为 Claude Code 提供 Bot Studio 节点图的结构化编辑能力，替代"盲写 YAML"的工作方式。

## 一、问题与目标

### 当前痛点

Claude Code 编辑 Bot 特殊模式只能通过：
1. **编辑 YAML 文件** — 无 schema 上下文、无验证反馈、无调试能力
2. **编辑 JSON** — UUID 不可读、缺少类型信息

### 目标

通过 MCP Server 让 Claude Code 获得：

| 能力 | 工具名 | 说明 |
|------|--------|------|
| 查询节点类型 schema | `list_node_types` | Claude Code 知道"能做什么" |
| 查询单节点详情 | `get_node_schema` | 精确了解某类节点的端口和 config |
| 读取当前流程 | `get_flow` | 获取 bot 的事件链 YAML |
| 更新流程 | `update_flow` | 带校验的写入 |
| 验证流程 | `validate_flow` | 不保存，仅返回错误/警告 |
| 执行测试 | `execute_test` | 发送测试消息，获取执行 trace |
| 单步调试 | `debug_step` | 逐步执行查看状态 |
| 重置调试 | `debug_reset` | 清除调试会话 |

## 二、技术方案

### 2.1 传输方式：stdio

```
Claude Code ←→ stdio (JSON-RPC) ←→ purrchat-mcp-server ←→ HTTP ←→ PurrChat Backend
```

**选择 stdio 的理由：**
- Claude Code 将 MCP Server 作为子进程启动，stdio 是原生协议
- 无需额外端口、无需 HTTP 服务器
- 认证通过环境变量注入，安全性更高（不在配置文件中暴露 token）
- 部署简单：`npx purrchat-mcp-server` 即可

### 2.2 技术栈

- **语言**：TypeScript（与前端共享类型定义，可复用 portTypes.ts 等）
- **SDK**：`@modelcontextprotocol/sdk` v1.x
- **运行时**：Node.js（通过 `npx` 或 `ts-node` 启动）
- **依赖**：`zod`（schema 验证）、`js-yaml`（YAML 解析）

### 2.3 项目位置

```
apps/backend/tools/purrchat-mcp/
├── package.json
├── tsconfig.json
├── src/
│   ├── index.ts              # 入口：创建 Server + 连接 Transport
│   ├── server.ts             # MCP Server 定义（tools + resources）
│   ├── client.ts             # HTTP 客户端：与 PurrChat Backend 通信
│   ├── schema.ts             # Zod schema 定义（复用前端类型）
│   └── types.ts              # 类型定义（从前端 types.ts 提取）
└── README.md
```

### 2.4 Claude Code 配置

在项目根目录 `.mcp.json`：

```json
{
  "mcpServers": {
    "purrchat": {
      "type": "stdio",
      "command": "npx",
      "args": ["--yes", "./apps/backend/tools/purrchat-mcp"],
      "env": {
        "PURRCHAT_API_URL": "http://localhost:8080",
        "PURRCHAT_API_TOKEN": "${PURRCHAT_API_TOKEN}"
      }
    }
  }
}
```

## 三、Tools 详细设计

### 3.1 `list_node_types` — 列出所有节点类型

```typescript
// 输入：无
// 输出：节点类型列表
{
  node_types: [
    {
      type: "trigger",
      label: "触发",
      icon: "🚀",
      category: "control",
      description: "起始节点",
      ports: {
        inputs: [],
        outputs: [
          { id: "out_exec", name: "执行", dataType: "trigger" },
          { id: "out_input", name: "用户消息", dataType: "string" },
          { id: "out_username", name: "发送者", dataType: "string" },
          { id: "out_time", name: "时间", dataType: "string" },
          { id: "out_args", name: "参数", dataType: "string" }
        ]
      },
      config_schema: null  // trigger 无 config
    },
    {
      type: "if",
      label: "条件",
      icon: "◇",
      category: "control",
      description: "条件分支",
      ports: {
        inputs: [{ id: "in_exec", name: "执行", dataType: "trigger" }],
        outputs: [
          { id: "out_true", name: "真", dataType: "trigger" },
          { id: "out_false", name: "假", dataType: "trigger" }
        ]
      },
      config_schema: {
        logic: { type: "string", enum: ["AND", "OR"], default: "AND" },
        conditions: {
          type: "array",
          items: {
            left: { type: "string", description: "左值，支持 {节点.端口} 变量引用" },
            operator: { type: "string", enum: ["==", "!=", "contains", ">", "<", "startsWith", "endsWith", "regex"] },
            right: { type: "string", description: "右值" }
          }
        }
      }
    }
    // ... 其余 12 种节点类型
  ]
}
```

**关键**：config_schema 让 Claude Code 知道每种节点接受哪些配置字段，避免盲目猜测。

### 3.2 `get_node_schema` — 查询单节点详情

```typescript
// 输入
{ type: "llm" }
// 输出：同 list_node_types 中的单个节点对象
// 用途：Claude Code 在编辑某类节点时精确查询
```

### 3.3 `get_flow` — 读取当前流程

```typescript
// 输入
{ bot_id: "uuid" }
// 输出
{
  bot_id: "uuid",
  bot_name: "My Bot",
  events: [...],       // SpecialModeEvent[]
  connections: [...],   // FlowConnection[]
  yaml: "...",          // YAML IR 格式的人类可读版本
  node_count: 12,
  connection_count: 18
}
```

**实现**：调用 `GET /api/bots/:id`，从 `mechanism_config` 中提取 `special_mode`，同时生成 YAML。

### 3.4 `update_flow` — 更新流程

```typescript
// 输入（二选一）
{
  bot_id: "uuid",
  // 方式 A：YAML 格式
  yaml: "version: 1\nnodes: ...",
  // 方式 B：JSON 格式
  events: [...],
  connections: [...]
}
// 输出
{
  success: true,
  warnings: ["节点 'AI思考' 名称重复，已自动添加后缀"],
  updated_events_count: 5,
  updated_connections_count: 8
}
```

**实现**：
1. 如果传入 YAML，先 `yamlToFlow()` 解析为 JSON
2. 调用 `ValidatePortedFlow()` 后端验证（通过新的 API 端点或本地校验）
3. 调用 `PUT /api/bots/:id` 更新 `mechanism_config`

### 3.5 `validate_flow` — 验证流程（不保存）

```typescript
// 输入
{
  yaml: "...",           // YAML 格式
  // 或
  events: [...],
  connections: [...]
}
// 输出
{
  valid: false,
  errors: [
    "节点 'AI思考' 没有输入连接",
    "连接 [用户输入 → AI思考] 引用了不存在的端口 out_exec（trigger 节点无此端口）"
  ],
  warnings: [
    "节点 '回复' 未连接任何输出，执行将在该节点终止"
  ]
}
```

**实现**：复用 `ValidatePortedFlow()` 逻辑 + 自定义验证规则。

### 3.6 `execute_test` — 执行测试

```typescript
// 输入
{
  bot_id: "uuid",
  message: "你好",        // 测试消息
  step_mode: false        // 是否单步模式
}
// 输出
{
  session_id: "debug_xxx",
  reply: "你好！有什么可以帮你的？",
  round: 1,
  event_traces: [
    {
      event_id: "evt_llm_abc",
      event_type: "llm",
      event_name: "AI 思考",
      status: "success",
      input: "你好",
      output: "你好！有什么可以帮你的？",
      duration_ms: 1234
    },
    {
      event_id: "evt_reply_def",
      event_type: "reply",
      event_name: "回复",
      status: "success",
      output: "你好！有什么可以帮你的？",
      duration_ms: 1
    }
  ]
}
```

**实现**：调用 `POST /api/bots/:id/debug`。

### 3.7 `debug_step` / `debug_reset`

```typescript
// debug_step 输入
{ bot_id: "uuid", session_id: "debug_xxx" }
// debug_reset 输入
{ bot_id: "uuid", session_id: "debug_xxx" }
```

**实现**：调用 `POST /api/bots/:id/debug/step` 和 `/debug/reset`。

## 四、Resources 设计

Resources 提供被动查询能力，Claude Code 可通过 `mcp__purrchat__resource_name` 访问。

| Resource URI | 说明 |
|-------------|------|
| `purrchat://bots` | Bot 列表（名称 + ID） |
| `purrchat://bots/{id}` | Bot 详情 + 机制配置 |
| `purrchat://bots/{id}/flow` | 特殊模式流程（YAML 格式） |
| `purrchat://bots/{id}/flow/schema` | 流程的 JSON Schema（用于编辑参考） |
| `purrchat://node-types` | 所有节点类型 schema（同 list_node_types） |

## 五、实现步骤

### Phase 1：基础框架（~1h）

1. 创建项目结构 `apps/backend/tools/purrchat-mcp/`
2. 初始化 `package.json`，安装 `@modelcontextprotocol/sdk` + `zod` + `js-yaml`
3. 实现 `index.ts`：stdio transport 启动
4. 实现 `client.ts`：封装 HTTP 请求到 PurrChat Backend（带 JWT token）
5. 实现 `schema.ts`：硬编码 14 种节点类型的元信息 + 端口定义（从 portTypes.ts 提取）
6. 在 `.mcp.json` 中配置

### Phase 2：读取类 Tools（~1h）

7. 实现 `list_node_types` tool
8. 实现 `get_node_schema` tool
9. 实现 `get_flow` tool
10. 实现 Resources（bots 列表 + flow）

### Phase 3：写入类 Tools（~1h）

11. 实现 `validate_flow` tool（本地校验，不调 API）
12. 实现 `update_flow` tool（YAML 解析 → 验证 → PUT API）
13. 复用前端 `yamlIR.ts` 的核心逻辑（或用 `js-yaml` 直接处理）

### Phase 4：调试类 Tools（~30min）

14. 实现 `execute_test` tool
15. 实现 `debug_step` / `debug_reset` tools

### Phase 5：文档与测试（~30min）

16. 编写 README.md
17. 端到端测试：Claude Code 调用 `get_flow` → 修改 YAML → `validate_flow` → `update_flow` → `execute_test`

## 六、后端补充 API

MCP Server 需要的后端 API 大部分已存在，可能需要新增：

| 端点 | 现状 | 说明 |
|------|------|------|
| `GET /api/bots/:id` | ✅ 已有 | 获取 bot 详情 |
| `PUT /api/bots/:id` | ✅ 已有 | 更新 bot（含 mechanism_config） |
| `POST /api/bots/:id/debug` | ✅ 已有 | 执行调试 |
| `POST /api/bots/:id/debug/step` | ✅ 已有 | 单步执行 |
| `POST /api/bots/:id/debug/reset` | ✅ 已有 | 重置会话 |

**无需新增后端 API**，MCP Server 纯粹是现有 API 的 MCP 包装层。

## 七、安全考量

1. **JWT Token**：通过环境变量 `PURRCHAT_API_TOKEN` 注入，不写入配置文件
2. **API URL**：默认 `http://localhost:8080`，可通过 `PURRCHAT_API_URL` 覆盖
3. **仅本地运行**：stdio transport 限制为本地进程，无远程暴露风险
4. **无写入文件能力**：MCP Server 不直接操作文件系统，所有修改通过 API 进行
5. **validate_flow 防护**：写入前先验证，减少损坏配置的风险

## 八、预期效果

使用 MCP Server 前（Claude Code 盲写 YAML）：
```
User: 帮我给这个 bot 加一个条件分支，如果消息包含"帮助"就走帮助流程
Claude Code: [盲猜 YAML 格式，不知道有哪些端口，不知道变量语法...]
```

使用 MCP Server 后：
```
User: 帮我给这个 bot 加一个条件分支，如果消息包含"帮助"就走帮助流程
Claude Code: [调用 get_flow 获取当前流程]
            [调用 get_node_schema("if") 了解 If 节点端口和 config]
            [生成正确的 YAML 配置]
            [调用 validate_flow 验证]
            [调用 update_flow 写入]
            [调用 execute_test 测试，发送 "帮我看看帮助文档"]
            → "流程验证通过，已添加条件分支。测试结果：消息'帮我看看帮助文档'
              成功匹配 contains 条件，走出了'真'分支。"
```
