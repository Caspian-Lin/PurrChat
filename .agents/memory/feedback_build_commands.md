---
name: feedback_build_commands
description: 后端编译必须 cd 到 apps/backend 目录，不能用相对路径 go build
type: feedback
originSessionId: f815f4ea-427b-4322-a47e-e1dcb935022f
---
# 后端编译命令

**规则：** 后端 Go 项目的 go.mod 在 `apps/backend/` 下，当前工作目录是 monorepo 根目录 `PurrChat/`。直接运行 `go build ./...` 会报错 `pattern ./...: directory prefix . does not contain main module`。

**Why：** 这个错误已出现多次，浪费上下文。

**How to apply：** 编译、测试后端时，始终使用 `cd /home/lxx/Lab/PurrChat/apps/backend && go build ./...` 或 `cd /home/lxx/Lab/PurrChat/apps/backend && go test ./...`。也可以使用项目根目录的 `make` 命令。

## PR 验证记录

**规则：** PR 描述中的验证部分保持简短，只列出关键命令和汇总结果；不要粘贴完整测试输出、覆盖率明细或冗长日志。

## 记忆文件提交

**规则：** `.agents/memory/` 目录下的记忆变更随当前任务的功能 commit 一并提交，不另开独立 commit 或 issue。
