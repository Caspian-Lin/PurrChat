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

