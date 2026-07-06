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
