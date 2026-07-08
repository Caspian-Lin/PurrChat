# Bot 系统调优调查报告

> 调查日期：2026-07-09
> 基线分支：`dev` (`d89c4f7`)
> 目标：让 Bot 从创建、配置、安装、触发到调试形成可验证的闭环

## 结论摘要

当前 Bot 系统需要重构，但不建议推倒重写。

应保留 Go 作为聊天领域和消息网关，保留 TypeScript 作为唯一工作流运行时；重构重点是删除双引擎语义、建立一个可验证的工作流契约，并把 Bot 从“特殊好友”升级为“带系统身份投影的 App”。

当前问题不是单纯的编辑器体验问题，而是运行时、配置、调试和测试之间存在多处断链：

1. XState 已接入，但最小工作流仍可能超时，消息和变量没有进入 machine context。
2. 前端展示 16 种节点，TS 注册 15 种；Python 无实现，Loop/Switch/Merge 只有节点文件，没有对应编译语义。
3. TS 调试端点固定返回 `501`，实际调试仍走旧 Go 引擎，与生产执行结果不一致。
4. YAML 仅是前端文件导入导出工具，不是 API 契约；错误只写 console，仍会自动保存。
5. 默认部署没有 `bot-engine` 服务，未配置 TS 时工作流只回复 `"..."`。
6. 没有覆盖 Bot 创建、安装、真实消息触发、Bot 回复入库和 WebSocket 到达的端到端测试。
7. `public` Bot 好友请求会进入无法审批的 pending 状态，不符合“公开后可直接添加”的产品定义。

因此，Roadmap 中“Bot 引擎编排迁移至 TS 微服务”和“调试面板完成”应视为部分完成，而不是可用性完成。

## 一、当前架构与 XState 的实际作用

### 生产调用链

```text
用户消息
  -> Go MessageService / BotEngine.OnMessage
  -> 查询 conversation enrollment 中的 Bot
  -> 每个 Bot 调一次 TS POST /execute
  -> TS BotExecutor 评估 mechanism trigger
  -> workflow-engine: Blueprint -> XState machine -> Actor
  -> TS 返回 reply 字符串
  -> Go 写消息分表并通过 WebSocket 广播
```

Go 仍拥有 Bot 查询、消息上下文读取、回复持久化和广播。XState 只负责 TS 服务内部的工作流控制流与会话 Actor，不应被理解为整个 Bot 系统。

相关证据：

- Go 在每条消息上探活并调用 TS，失败后记录日志但不执行 fallback：`apps/backend/internal/botengine/engine.go:201`
- TS 按 `conversation_id:bot_id` 建内存 session：`apps/bot-engine/src/services/bot-executor.ts:110`
- XState compiler 将节点编译成状态和 promise actor：`packages/workflow-engine/src/compiler.ts:31`
- Go 负责最终回复入库和 WebSocket 广播：`apps/backend/internal/botengine/engine.go:229`

### XState 当前的价值

XState 适合保留，原因不是“有状态机就更先进”，而是它能提供：

- 明确的等待、恢复、取消和终止语义；
- 每个安装实例或会话一个 Actor；
- 可序列化 snapshot；
- 与执行 trace 对齐的状态转换。

但当前实现尚未兑现这些能力：

- machine 使用固定 `context`，没有从 actor `input` 初始化；
- session 只存在内存，未持久化、未过期清理；
- workflow 在 `createSession()` 时立即启动，首条消息随后才发送；
- 普通节点没有后继时，把 `type: final` 放在 transition 上，状态不会结束；
- `end_conditions` 被传入 Blueprint，但 compiler/runtime 从未读取；
- schema 只挂在 NodeDefinition 上，从未执行 `parse`/`safeParse`。

本地最小验证：

```text
trigger -> reply
结果：30 秒后 timeout

trigger -> reply -> end
输入：hello world, username=alice
模板：user=$username args={args} readable={触发.用户消息} machine=$t:out_input
实际：user=$username args= readable={触发.用户消息} machine=$t:out_input
```

### 重构判断

推荐“边界重构”，不推荐重写全部系统：

- 保留：Go 消息入口、会话权限、消息持久化、WebSocket。
- 保留并修正：TS NodeRegistry、Compiler、Runtime、XState。
- 删除：Go 工作流执行和 Go 调试 fallback。
- 统一：触发、变量、节点 schema、trace、错误模型全部由 TS 合同定义。
- 部署要求：`bot-engine` 成为必需服务；不可用时显式标记 Bot 执行失败，不发送占位回复。

## 二、节点与编辑器可用性

### 节点支持矩阵

| 节点 | 前端可添加 | TS 已注册 | Compiler 语义 | 结论 |
| --- | --- | --- | --- | --- |
| trigger/end/reply/llm/builtin/template/history/tool/dify/n8n | 是 | 是 | 基础串行 | 需补契约与错误处理 |
| if | 是 | 是 | 仅 true/false 分支 | 配置变量未解析 |
| wait | 是 | 是 | 等 USER_MESSAGE | 首条消息/会话语义错误 |
| loop | 是 | 是 | 无 loop 分支编译 | 不可用 |
| switch | 是 | 是 | 无 case 分支编译 | 不可用 |
| merge | 是 | 是 | passthrough，无 fan-in | 名称与行为不符 |
| python | 是 | 否 | Unknown node type | 不可用 |

前端允许用户保存这些节点，后端也不验证配置，导致“能画、能保存、运行失败”。

### 编辑体验问题

- 保存按钮不会被 validation error 阻止。
- 后端 `UpdateBot` 原样接受 `mechanism_config`。
- YAML 导入错误只 `console.warn`，仍会覆盖并自动保存。
- YAML v1 不包含 `end_conditions`、画布 position、schema version 迁移信息。
- 节点 schema 在前端、共享 types、TS node definition 三处重复，配置字段已有漂移。
- 自定义端口可由 UI 添加，但运行时节点并不理解任意自定义端口的业务语义。
- 自动布局在节点/连接数量变化时触发，容易覆盖用户布局预期。

### 建议

1. 由 `workflow-types` 提供版本化 Workflow Document 和 Node Manifest。
2. TS NodeRegistry 成为节点能力唯一来源；前端通过 manifest 渲染可用节点与配置。
3. 只有 `implemented + tested` 的节点可在生产编辑器中选择。
4. 保存前调用服务端 validation；错误阻止保存，警告需显式确认。
5. 编辑器增加草稿、发布版本和未保存状态，不再直接覆盖线上运行配置。

## 三、API 与 YAML 工作流

可以并且应该支持非图形化组织。现有 YAML IR 是可用起点，但不能继续只存在浏览器中。

建议工作流文档：

```yaml
apiVersion: purrchat.ai/v1alpha1
kind: BotWorkflow
metadata:
  name: greeting
spec:
  trigger:
    type: rule
    rules:
      - type: command
        pattern: /hello
  nodes: []
  connections: []
  endConditions: []
```

建议 API：

- `GET /api/bots/:botId/workflows/:workflowId`
- `PUT /api/bots/:botId/workflows/:workflowId`
- `POST /api/bots/:botId/workflows/validate`
- `POST /api/bots/:botId/workflows/:workflowId/test-runs`
- `POST /api/bots/:botId/workflows/:workflowId/publish`
- JSON 为规范存储格式，YAML 通过 content negotiation 或 CLI 转换。
- 更新请求携带 revision/ETag，避免图形编辑器与 YAML 覆盖彼此。

验收要求：

- YAML -> JSON -> YAML 语义等价；
- 未知节点、端口、字段和环路得到结构化错误；
- secret 只允许引用 secret ID，不允许导出明文；
- API、编辑器和运行时使用同一个 validator；
- 文档版本可迁移且旧版本有明确弃用窗口。

## 四、变量系统根因

变量“不生效”是运行时缺陷，不只是 UI 问题。

### 已确认断链

1. Runtime 把 `rawInput`、`username` 放进 actor input，但 compiler 的固定 context 不读取 input。
2. Compiler 创建节点上下文时没有传 `nodeOutputs` 和 `nameResolver`。
3. Reply/Template/Tool/N8n/Dify 调用 `replaceVariables` 时主动传入空的 `nodeOutputs` 或 `nameResolver`。
4. If 的 `{node.port}` 只返回原字符串，没有调用统一 resolver。
5. VarReferencePicker 展示人类可读名称，但实际插入 `$nodeId:portId`，用户仍面对不稳定 ID。
6. Go 与 TS 的 `{args}` 含义和索引起点不同。

### 目标变量模型

只保留一种表达式语法，并提供静态类型：

```text
input.text
sender.id
sender.name
conversation.id
history.messages
nodes.<stableKey>.outputs.<port>
session.<name>
secrets.<name>
```

要求：

- 节点有稳定 key，显示名称可修改且不影响引用；
- 编辑器变量选择器写入结构化 AST 或稳定 path，不拼 UUID 字符串；
- validator 能判断变量是否存在、类型是否兼容、是否越权访问 secret；
- trace 展示每个节点的 resolved input/output，并默认脱敏 secret；
- 同一套 resolver 被 If、Template、Reply、HTTP、LLM 等全部节点复用。

## 五、调试与可观测性

当前前端调试请求进入 Go `DebugExecute`，执行的是 deprecated Go flow engine；TS `/debug` 和 `/debug/step` 固定返回 `501`。因此调试结果不能代表生产。

目标是“同运行时、不同副作用策略”：

- 调试和生产都调用 TS Runtime；
- test run 可选择 mock side effects 或真实 sandbox；
- 每次运行有 `run_id`，每个节点有 start/success/error/skip trace；
- 显示 resolved input、output、duration、branch、retry 和错误；
- 支持从已保存版本或未发布草稿执行；
- 外部 HTTP/LLM 有超时、取消、重试和脱敏；
- Go 调用日志只记录 run_id 和摘要，详细 trace 由运行服务持久化。

## 六、真实端到端测试闭环

当前覆盖：

- Go botengine 只有 `{args}` 和时间戳单测；
- TS workflow-engine 和 bot-engine 没有测试文件；
- 前端没有工作流编辑器、YAML、变量、节点或调试测试；
- 后端集成测试没有 Bot 创建、公开、安装、触发和回复断言。

建议测试金字塔：

### 1. 工作流契约测试

- 每种节点 manifest、schema、端口与执行器一致；
- Blueprint validation；
- 变量解析与类型检查；
- trigger/if/switch/loop/merge/wait/end；
- timeout、cancel、retry、错误分支；
- snapshot 恢复与 session 隔离。

### 2. 服务集成测试

- Go -> TS `/execute` 合同；
- TS 不可用、超时和 4xx/5xx；
- Bot 配置保存与发布；
- deployment 状态确实控制执行；
- 回复落库、call log 和隐私脱敏。

### 3. 全栈 E2E

使用真实 PostgreSQL、Go、bot-engine 和 WebSocket：

1. 用户 A 创建 Bot。
2. 设置公开并发布 workflow。
3. 用户 B 安装到私聊或群聊。
4. 用户 B 发送真实消息。
5. 断言 trigger 命中、trace 完成、Bot 回复入库。
6. 断言 B 的 WebSocket 收到相同 message ID。
7. 刷新并从历史接口读取同一回复。
8. 暂停/卸载后再次发送，断言 Bot 不执行。

LLM/HTTP 节点使用仓库内 fake provider，禁止依赖公网。

CI 至少设置：

- `workflow-contract`
- `bot-engine-integration`
- `bot-e2e`

## 七、旧模块去留

### 应删除

- `apps/backend/internal/botengine/flow_engine.go`
- `apps/backend/internal/botengine/debug.go`
- `apps/backend/internal/botengine/workflow.go` 中旧执行/session 逻辑
- Python Go sandbox（除非产品确认 Python 节点继续存在）
- `special_mode` 类型和前端 fallback 字段
- Go 与 TS 双份 trigger/reply/args 语义

删除前提：

- TS Runtime 覆盖全部保留节点；
- debug 已切到 TS；
- 数据迁移已把旧配置升级为版本化 document；
- E2E 覆盖旧配置迁移；
- bot-engine 被纳入开发和生产部署。

### 应保留或迁移

- Go BotEngine 作为消息路由 adapter，但缩减为调用执行服务与发送结果；
- `bot_deployments`，演进为 app installation；
- call logs，但修复隐私问题并关联 run_id；
- `users.is_bot` 可暂时作为消息发送者投影，不再代表可登录或可交友的人类账号。

## 八、公开 Bot 好友失败与 Bot-as-App

### 当前根因

`public` 的模型注释是“所有人可搜索添加”，实际代码只有 `global` 自动接受；`public` 创建 pending friendship，并通知 owner。

但是 pending 记录是“普通用户 -> Bot”。处理接口只查询当前处理者自己的 friendship，并要求当前用户是 `friend_id`。Bot 不能登录，owner 又不是 friendship 的参与者，因此 owner 无法批准。重复添加随后返回 `friend request already pending`。

此外：

- 创建 Bot 时会额外建立 owner <-> Bot 双向 friendship；
- friendship 与 enrollment/deployment 同时表示关系，职责重叠；
- deployment 创建失败被当成非致命错误，可能出现 Bot 已入群但无安装记录；
- `public/global/private` 混合了可发现性、安装权限和系统所有权。

### 是否学习 Discord

建议学习 Discord 的“Application + Bot User + Installation”分层，但不要照搬 Discord 的所有交互。

Discord 官方模型的关键点：

- Application 是凭证、OAuth2、Bot 配置和元数据的容器；
- Bot User 是独立自动化身份，不是普通用户；
- App 安装到 user 或 server，并在安装时授予 scopes/permissions；
- Bot 不能成为普通好友，进入 server 走 OAuth2 安装。

参考：

- https://docs.discord.com/developers/quick-start/overview-of-apps
- https://docs.discord.com/developers/resources/application
- https://docs.discord.com/developers/platform/oauth2-and-permissions
- https://discord.com/developers/docs/topics/oauth2

### PurrChat 目标模型

```text
BotApp
  owner_id
  metadata / visibility / status
  published_workflow_version
  requested_capabilities

BotIdentity
  app_id
  display_name / avatar
  system principal, cannot login, cannot friend

BotInstallation
  app_id
  installed_by
  target_type: user | conversation
  target_id
  granted_capabilities
  status
  config
  installed_at

WorkflowVersion
  app_id
  revision
  document
  published_at
```

建议交互：

- 私有 App：仅 owner 可安装。
- 公开 App：用户可直接“添加 App”，无需 owner 逐次审批。
- 用户安装：创建或复用用户与 BotIdentity 的私聊并创建 installation。
- 群聊安装：只有 owner/admin 可授权，BotIdentity 作为 member 投影加入。
- 卸载：撤销 installation 和 enrollment，不创建/删除 friendship。
- Bot 列表和联系人 UI 可继续展示相似外观，但数据来源不同。

建议权限：

- `messages:read_trigger`
- `messages:read_history`
- `messages:send`
- `members:read`
- `conversation:write_metadata`
- `network:external`
- `secrets:use`

工作流发布时计算所需 capability；安装时展示并授予最小权限。新增高风险节点或权限时需要重新授权。

### 迁移策略

1. 先修复公开 Bot：public 走幂等自助安装，避免继续产生 pending friendship。
2. 增加 installation API，并让私聊和群聊共用。
3. 将现有 `bot_deployments` 回填为 installation。
4. 停止为新 Bot 创建 owner friendship。
5. 迁移已有 Bot friendship，保留对应会话和 enrollment。
6. 最后移除 Bot 好友审批分支。

## 九、Milestone 建议拆分

### P0：止血与可信基线

1. 修复 XState context、终止、首条消息、session 生命周期。
2. 建立节点支持矩阵，隐藏 Python/Loop/Switch/Merge 等未完成节点。
3. 公开 Bot 改为幂等安装，修复好友失败。
4. 增加 Workflow Document schema 和服务端 validation。
5. 建立最小 E2E：创建 -> 发布 -> 安装 -> 消息 -> 回复 -> 历史。

### P1：统一体验

1. 变量 AST/path、类型检查和选择器。
2. TS 原生 trace/debug，移除 Go 调试路径。
3. YAML/API、草稿/发布/revision。
4. 完成或删除高级节点。
5. bot-engine 加入 Docker、健康检查和 CI。

### P2：App 化与清理

1. BotApp/BotIdentity/BotInstallation 数据迁移。
2. capability 授权和 secret 引用。
3. 删除 Go 引擎、special mode、旧 session 和重复类型。
4. 完成隐私、审计、速率限制和恢复策略。

## 十、Milestone 完成标准

- 默认开发和部署环境使用唯一 TS 工作流运行时。
- 编辑器中所有可选节点均有 contract test 和 E2E 证据。
- 无效图、YAML 或变量不能发布。
- 调试 trace 与真实运行使用同一 Runtime。
- 公开 Bot 可被其他用户直接安装并完成私聊触发。
- 群聊安装有明确管理员权限和 capability 授权。
- 暂停、卸载、禁用都能阻止执行。
- 服务重启后 session 行为有定义并经过测试。
- 旧 Go 执行和 `special_mode` 已删除，或有明确迁移截止版本。
- Bot 调用日志不泄露未授权会话的原始消息。

