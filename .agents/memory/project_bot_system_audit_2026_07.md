---
name: bot_system_audit_2026_07
description: 2026-07-09 Bot 系统可用性、XState、变量、测试与 App 化调查结论
type: project
---

# Bot 系统调查结论

详细报告：`docs/bot-engine/BOT_SYSTEM_AUDIT_2026-07-09.md`

## 关键事实

- 调查基线为 `dev` 分支 `d89c4f7`。
- Go 仍负责消息入口、Bot 查询、回复入库与 WebSocket；XState 只在 TS workflow-engine 内执行 Blueprint。
- XState compiler 使用固定 context，actor input 中的 rawInput/username 未注入。
- `trigger -> reply` 最小流程会等待 30 秒超时；加 end 后能结束，但变量保持原样。
- TS `/debug` 和 `/debug/step` 返回 501；前端调试实际仍走 deprecated Go 引擎。
- 前端有 16 类节点，TS 无 Python；Loop/Switch/Merge 缺少真正 compiler 语义。
- `end_conditions` 和 NodeDefinition `configSchema` 在 TS 运行时未使用。
- 默认 Docker Compose 没有 bot-engine；未配置 TS 时 Go workflow fallback 只回复 `...`。
- TS 引擎/微服务没有测试；没有 Bot 全栈 E2E。

## 公开 Bot 好友问题

- `public` Bot 被写成 pending friendship，只有 `global` 自动接受。
- friendship 方向为普通用户 -> Bot；Bot 不能登录，owner 又不是 friendship 接收方，现有审批接口无法处理。
- 建议短期将 public 改为幂等自助安装，长期使用 BotApp + BotIdentity + BotInstallation，Bot 不再参与普通 friendship。

## 方向

- 不推倒重写：保留 Go 聊天领域层和 TS/XState 运行时。
- TS 成为唯一工作流执行和调试实现，删除 Go flow/debug fallback。
- 建立版本化 Workflow Document、统一 validator、变量 path/AST、trace 和 E2E。
- `bot_deployments` 演进为 installation；Bot 用户记录只作为消息身份投影。

## issue 11 实现完成（fix(bot): XState 运行时输入、终止与会话语义）

分支 `fix/11-xstate-runtime`，修复 audit P0 第 1 项。workflow-engine + bot-executor。

### 核心修复

1. **context 从 actor input 初始化**：`compiler.ts` 用 `context: ({ input }) => buildInitialContext(input, nameResolver)`，rawInput/username/time/sender 全部进入 machine context（旧实现写死固定 context，input 被忽略）。
2. **无后继节点可确定性结束**：新增 `__done` final 状态；普通节点无 outConn 时 onDone target `__done`；if 用 guarded onDone transitions 数组 + 兜底 `__done`。旧的 `type:'final'` 错放在 onDone 内导致 `trigger→reply` 30s timeout。
3. **trigger/wait 统一为事件驱动入口**：机器启动后停在 trigger，等第一条 `USER_MESSAGE` 才初始化并流转。这统一了一次性 execute 和多轮 session 语义，并保证「首条消息只执行一次」。
4. **变量解析传入完整 context**：reply/template/tool/dify/n8n/if 调 `replaceVariables` 改用完整 NodeContext（含 nodeOutputs/nameResolver），不再传空对象。`{节点.端口}`、`$nodeId:portId`、`$变量` 现在都能解析。
5. **endConditions/超时/清理/并发**：runtime 实现 max_rounds、message_match、session timeout；会话到达终态自动销毁（下次重新创建）；同一 session 并发消息用 busy 锁拒绝。
6. **错误不伪装为 `...`**：runtime error 返回空 reply + error status；bot-executor 去掉 compile 失败/未知类型的 `'...'` 占位。

### 关键实现决策

- **if 编译用 guarded onDone transitions 数组**，不能用 `always`（always 是 state 级属性，放在 onDone transition 内无效，会导致 if 流程卡死）。
- **`__error` 是 final 状态，snapshot.status 也是 `'done'`**：`classifySnapshot` 必须先检查 `matches(__error)` 再判断 done，否则错误被误判为成功。
- **判断工作流暂停**：runtime 在 compile 期收集 wait 节点 id 集合，`isSettled` 把「停在 wait 节点」视为 settled(waiting)。

### Breaking change

`runtime.execute` / `sendMessage` 返回 `ExecuteResult`（含 reply/status/sessionActive/round）而非 string。仅 bot-executor 和测试调用，均已适配；frontend 不直接用 WorkflowRuntime。

### 验证

`packages/workflow-engine/src/__tests__/contract.test.ts` 15 个契约测试：变量注入、$nodeId:portId、无 end 终止、错误抛出、未知节点、if true/false/兜底、wait 多轮、首条消息只执行一次、会话销毁、endConditions(max_rounds/message_match)、并发拒绝。`pnpm -F @purrchat/workflow-engine test` 全绿。

### 未覆盖（属后续 issue）

- debug 端点仍 501（audit P1）。
- Go flow/debug fallback 未删除（audit P2）。
- 服务集成测试（Go→TS /execute）、全栈 E2E（audit P0 第 5 项）。

## 创建者私聊/群聊固定回复不触发（2026-07-13）

- 根因：Bot 创建及首次发布前的群聊安装会把当时为空的 `requested_capabilities` 固化为 installation 的空 `granted_capabilities`；发布只更新 Bot，没有同步创建者安装。消息在 `messages:read_trigger` 门禁被静默跳过。
- 修复：发布事务同步 `installed_by = owner` 的私聊与群聊安装能力；第三方安装不自动扩权。迁移 `014_sync_owner_installation_capabilities.sql` 回填存量 owner 安装。
- 可诊断性：缺少 `messages:read_trigger` 时写入 `capability_not_granted` 调用记录，不再只有服务端日志。
