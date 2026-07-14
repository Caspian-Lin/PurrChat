---
name: feedback_github_cli_sandbox_network
description: GitHub CLI 写操作在沙箱内可能无法连接 api.github.com，需要提权执行
type: feedback
---

# GitHub CLI 沙箱网络问题

**规则：** 在当前 PurrChat 工作区中，`gh issue create` 等 GitHub 写操作可能在沙箱内稳定或间歇性报错：

```text
error connecting to api.github.com
check your internet connection or https://githubstatus.com
```

同一命令使用 `sandbox_permissions: require_escalated` 后可以正常完成。因此，后续 GitHub 写操作遇到该错误时应直接提权重试，不要反复在沙箱中执行，也不要误判为 GitHub 服务或认证故障。

只读 `gh issue list`、`gh api` 查询有时能在沙箱内成功，但不能据此假设写操作网络也可用。

## 记录场景

- 日期：2026-07-13
- 任务：批量创建 Bot Workflow Studio 2.0 milestone/issues
- 现象：五个并行 `gh issue create` 全部在沙箱内连接失败；提权后顺序创建 #106–#110 成功。
