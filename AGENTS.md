# PurrChat Agent Guide

进入任务先读本文件；再按需读 `README.md`、`docs/ROADMAP.md`、`docs/DESIGN_SYSTEM.md`、相关架构文档和源码。`CLAUDE.md` 的要求已合并到这里，后续可删除。

## 项目记忆

每次对话开始先了解当前状态：

- 读 `docs/ROADMAP.md` 的「已知问题」和「待实现功能列表」。
- 读 `.agents/memory/` 下的项目记忆 markdown；该目录是多 agent 共享读写的项目级记忆区。

## 项目概况

PurrChat 是全栈实时聊天平台，探索人与 AI 的情感连接和灵感边界。核心特性是 Bot Studio：可视化、事件驱动的 Bot 编排系统，用户可通过 DAG 节点编辑器构建复杂 AI 角色。

Monorepo 结构：

- `apps/frontend`：Vue 3 + Vite + Tailwind CSS + Pinia；承载 Web、Tauri Desktop、Tauri Mobile。
- `apps/backend`：Go + Gin + PostgreSQL；认证、好友、会话、消息、Bot、WebSocket。
- `apps/storage`：Go 文件存储服务；Cloudflare R2 / MinIO，两阶段预签名上传。
- `packages/workflow-types`：Bot 工作流类型定义。
- `packages/workflow-engine`：TypeScript 工作流引擎。
- `docs`：设计系统、路线图、部署、WebSocket、Bot 引擎等文档。

品牌关键词：**Intimate · Refined · Alive**（亲密 · 精致 · 鲜活）。

## 开发流程

### 1. 分析需求 / Bug / 技术方案

实现前先明确：

- 新功能：用户目标、核心路径、涉及模块、数据模型、API、验收方式。
- Bug：复现路径或症状、根因、影响范围、回归风险、测试缺口。
- 技术选型：优先复用项目已有模式；只有现有架构不足时才引入依赖或抽象。
- UI/样式：先读 `docs/DESIGN_SYSTEM.md`，并检查现有组件风格。
- Roadmap 事项：对照 `docs/ROADMAP.md`。

不确定的产品行为、权限边界、数据保留、外部服务配置、迁移方案、设计取舍，先问项目 owner。

### 2. Milestone 与 Issue

完整功能或跨模块改动：先建 milestone，再拆 milestone 下的子 issue。

流程：

1. 在 GitHub 创建 milestone，名称表达用户可验收的阶段目标。
2. 拆成可独立交付的 issue；每个 issue 只覆盖一个清晰任务。
3. issue 关联 milestone，并确保自动同步到 GitHub Project 看板。
4. issue 粒度应便于独立分支、PR、CI 和用户验收。

GitHub 交互优先用 `gh`。如 Project 看板、GraphQL、权限或自动化能力不足，找 owner 配 MCP、token scope 或权限。

每个 issue 必须包含：

- 这个任务为什么存在？用户价值、业务目标、bug 根因或技术债背景。
- 完成标准是什么？用可验证 checklist 描述行为、测试、文档、兼容性和验收标准。
- 有哪些约束？技术栈、设计系统、性能、安全、迁移、兼容性、不可改边界。
- 相关设计讨论在哪里？GitHub discussion、PR comment、issue comment、设计文档、ADR 或本地 docs。
- 最终 PR 是哪个？创建时可写 `TBD`，PR 创建后回填。

Issue 模板：

```markdown
## Why

## Acceptance Criteria

- [ ]

## Constraints

-

## Design Discussion

- TBD

## Final PR

- TBD
```

### 3. 分支策略

- `main`：稳定主线。只有 `dev` 经充分用户测试并完成一个 milestone 后，才统一合入 `main`。
- `dev`：日常集成分支。功能分支从 `dev` 创建，PR 合入 `dev`。
- 功能分支：每个 issue 一个分支，建议 `feat/<issue-id>-short-name`、`fix/<issue-id>-short-name`、`docs/<issue-id>-short-name`。
- 不在 `main` 直接开发；不把未完成、未测试、未 review 的功能直接合入 `dev`。

### 4. 单任务开发与 PR

1. 从最新 `dev` 创建功能分支。
2. 按 issue 范围实现，不夹带无关重构。
3. 本地运行匹配影响范围的测试；跨模块或不确定影响时运行完整 `make test`。
4. 提交前遵守本文 commit 流程。
5. 创建 PR，目标分支 `dev`。
6. 等 CI 通过。
7. 开发者复查 PR diff、测试结果和 issue 完成标准。
8. 请求 owner 或指定 reviewer 确认。
9. reviewer 确认后才合并到 `dev`。
10. milestone 完成并经用户测试后，再从 `dev` 统一合入 `main`。

PR 描述包含：关联 issue、实现内容、本地验证命令与结果、风险/迁移/配置/回滚说明、UI 截图或录屏说明（如适用）。

## GitHub 操作

优先用 `gh`：

- 创建 milestone、issue、PR。
- 查询 CI、PR review/comment、issue 状态。
- 更新 issue body，回填最终 PR。
- 在 PR comment 中沉淀代码实现决策。

涉及 GitHub Project 看板同步时，先尝试 `gh` 和 GraphQL；需要额外 token scope、MCP 或 GitHub App 权限时，向 owner 说明能力和原因。

未经用户明确要求，不要 push、merge、close issue、删除远程分支或修改远程保护规则。

## Commit Workflow

当用户显式请求提交（如 `/commit`、`提交`、`commit`），或对话结束前检测到有未提交代码改动时，执行：

0. 运行 `make lint-fix`，确保代码正确格式化。

第一步：创建提交计划

1. 运行 `git status` 和 `git diff` 了解所有变更。
2. 按逻辑分组为独立 commit，不同关注点分开提交。
3. 每个 commit 用中文拟定简洁 message，格式 `<type>(<scope>): <描述>`；type 使用 `feat`、`fix`、`refactor`、`style`、`docs` 等。
4. 向用户展示分组计划，包含每组文件列表和 commit message。
5. 等待用户确认或调整；未确认前不要执行任何 `git add` / `git commit`。

第二步：执行提交

1. 用户确认后，按计划依次 `git add` 和 `git commit`。
2. 最后运行 `git status` + `git log --oneline -N` 展示结果。

规则：

- 不要 push，除非用户显式要求。
- commit message 可多行，用 markdown 式分点，简洁描述改动目的而非机械罗列内容。
- 已有暂存区内容时，一并纳入分析。
- 不添加 `Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>`。
- 遵守 `docs/COMMIT_CONVENTION.md`。

## 决策记录与 Knowhow

开发过程中的代码实现决策优先记录在 PR comment 中，和 diff、review、CI 结果绑定，例如：

- API 边界选择。
- 复用某个 store、service、repository 或组件模式的原因。
- 不引入新依赖的原因。
- bug 根因和防回归测试点。
- 数据迁移、兼容策略或回滚注意事项。

只有重大、结构性的 Architecture Decision Record 写入本地 `docs`，例如：

- 改变 Bot 工作流引擎架构。
- 引入新的跨服务通信方式。
- 改变认证、权限、数据模型或部署拓扑。
- 长期影响多个 milestone 的技术路线选择。

本地 ADR 放在 `docs/` 合适位置，至少包含背景、决策、取舍、影响和迁移计划。

## 不确定事项与外部依赖

遇到以下情况先问 owner：

- 产品需求或验收标准不明确。
- 需要下载工具、依赖、模型、数据集或外部资源。
- 配置环境时遇到网络、权限或系统依赖问题。
- 需要 GitHub MCP、Linear MCP、数据库、云服务、对象存储、CI secret 或额外 token scope。
- 需要修改远程仓库设置、Project 自动化、分支保护、CI 权限。
- 需要破坏性操作，如删除数据、重写历史、强推、清理不属于当前任务的文件。

能通过本地代码和文档确认的先自行调查；仍有关键未知时，列出已确认事实、未知点和建议选项，再请 owner 决策。

## 设计上下文

UI/样式工作必须遵循 `docs/DESIGN_SYSTEM.md`。

品牌：PurrChat 面向技术爱好者和 AI 探索者，探索人与 AI 的情感连接和灵感边界。

设计理念：**Soft Architecture（柔软建筑）**。PurrChat 不是工具，是空间：一个安静、精致、有呼吸感的空间。

设计原则：

1. **Quiet Confidence**：通过克制建立层级，高级意味着知道什么不该加。
2. **Living Geometry**：统一圆角系统，消灭所有直角，创造有机视觉和谐。
3. **Substance Over Surface**：每个设计元素都必须 earns its place。
4. **Breathing Space**：充裕留白让内容有空间呼吸。
5. **Material Honesty**：表面感觉像真实材料，色彩有自然质感。

反模式，绝不使用：

- 左侧色条（`border-left > 1px`）
- 渐变文字
- glassmorphism
- 紫蓝渐变
- neon accent
- bounce / elastic 动效

核心设计决策：

- 默认强调色：Sage 绿（`#5A8F4E`），低饱和矿物感。
- 暗色模式：蓝调深灰（`#111116`），暮色天空感。
- 正文：Onest；标题：Bricolage Grotesque；CJK：Noto Sans SC。
- 6 级统一圆角 token（`4px` 到 `9999px`）。
- 单层柔和阴影，褐调色彩。
- 比一般聊天应用更慷慨的间距。

## 常用验证命令

按改动范围选择：

```bash
make lint-fix
make lint
make test
pnpm test
pnpm build
cd apps/frontend && pnpm test --run
cd apps/backend && go test ./...
cd apps/storage && go test ./...
```

跨前后端、WebSocket、数据库迁移、存储服务、Bot 引擎或工作流相关改动，优先运行更完整验证，并在 PR 记录实际结果。
