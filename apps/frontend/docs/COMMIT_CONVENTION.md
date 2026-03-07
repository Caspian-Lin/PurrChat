# 提交规范 (Commit Convention)

本文档定义了 PurrChat 项目的 Git 提交信息规范，基于 [Conventional Commits](https://www.conventionalcommits.org/) 规范。

## 规范概述

提交信息格式：

```
<type>(<scope>): <subject>

<body>

<footer>
```

## Type 类型

| Type | 说明 | 示例 |
|------|------|------|
| `feat` | 新功能 | `feat(frontend): add user profile page` |
| `fix` | 修复 Bug | `fix(backend): resolve authentication token expiration` |
| `docs` | 文档更新 | `docs: update deployment guide` |
| `style` | 代码格式调整（不影响功能） | `style(frontend): format code with prettier` |
| `refactor` | 重构（既不是新功能也不是修复） | `refactor(backend): simplify user repository` |
| `perf` | 性能优化 | `perf(frontend): optimize image loading` |
| `test` | 测试相关 | `test(backend): add integration tests for auth` |
| `chore` | 构建过程或辅助工具的变动 | `chore: update dependencies` |
| `ci` | CI/CD 配置变更 | `ci: add GitHub Actions workflow` |
| `revert` | 回退提交 | `revert: feat(api): remove deprecated endpoint` |

## Scope 范围

| Scope | 说明 |
|-------|------|
| `frontend` | 前端相关 (Vue 3 应用) |
| `backend` | 后端相关 (Go API) |
| `ci` | CI/CD 相关 (GitLab CI / GitHub Actions) |
| `docker` | Docker 相关 (Dockerfile, docker-compose) |
| `docs` | 文档相关 |
| `deps` | 依赖更新 |

## Subject 标题

- 使用动词开头，使用现在时态（如 "add" 而不是 "added"）
- 首字母小写
- 结尾不加句号
- 限制在 50 个字符以内

**示例：**

```
✅ feat(frontend): add user profile page
✅ fix(backend): resolve authentication token expiration
✅ docs: update deployment guide

❌ feat(frontend): Added user profile page
❌ Fix(backend): Resolve authentication token expiration.
❌ docs: Update Deployment Guide
```

## Body 正文

- 详细描述提交的内容
- 说明 "为什么" 和 "是什么"，而不是 "怎么做"
- 每行限制在 72 个字符以内
- 使用祈使句

**示例：**

```
feat(frontend): add user profile page

Implement a new user profile page that displays:
- User avatar and basic information
- Account settings
- Privacy options

The page uses the existing user API endpoint and follows
the design system guidelines.
```

## Footer 页脚

- 列出所有 Breaking Changes（破坏性变更）
- 引用相关的 Issue
- 列出所有关闭的 Issue

**示例：**

```
feat(frontend)!: redesign authentication flow

BREAKING CHANGE: The authentication API has been completely redesigned.
Old endpoints are deprecated. Please update your client applications
to use the new authentication flow.

Closes #123
```

## 完整示例

### 新功能

```
feat(frontend): add user profile page

Implement a new user profile page that displays:
- User avatar and basic information
- Account settings
- Privacy options

The page uses the existing user API endpoint and follows
the design system guidelines.

Closes #45
```

### 修复 Bug

```
fix(backend): resolve authentication token expiration issue

The JWT token expiration time was incorrectly calculated,
causing tokens to expire prematurely. This fix ensures
the correct expiration time is set based on the configured
JWT_EXPIRATION environment variable.

Fixes #67
```

### 文档更新

```
docs: update deployment guide

Add detailed instructions for deploying to production
environment, including:
- Environment variable configuration
- Database migration steps
- Health check procedures
- Rollback strategies
```

### 重构

```
refactor(backend): simplify user repository logic

Extract common database operations into a base repository
class to reduce code duplication and improve maintainability.

This change does not affect the public API or functionality.
```

### CI/CD 变更

```
ci: add GitHub Actions workflow

Implement a complete CI/CD pipeline using GitHub Actions,
replacing the previous GitLab CI configuration.

The new workflow includes:
- Automated linting
- Testing with coverage reports
- Docker image building and pushing
- Automated deployment to dev/prod environments
```

### 破坏性变更

```
feat(frontend)!: redesign authentication flow

BREAKING CHANGE: The authentication API has been completely redesigned.
Old endpoints are deprecated. Please update your client applications
to use the new authentication flow.

Changes:
- POST /api/auth/register now requires email verification
- POST /api/auth/login returns a different token format
- POST /api/auth/logout is now required for proper session cleanup

Migration guide: https://docs.purrchat.com/migration/v2-to-v3
```

## 提交前检查

项目配置了 Husky 和 lint-staged，提交前会自动运行：

```bash
# 安装 Git hooks
pnpm install

# 提交时自动运行
git commit -m "feat: add new feature"
```

lint-staged 会自动：
- 运行 ESLint 检查前端代码
- 运行 Prettier 格式化代码
- 运行 golangci-lint 检查后端代码

## 常见错误

### 错误示例

```
❌ Update README
❌ Fixed bug
❌ Added new feature
❌ update readme
❌ fix: fix bug
❌ feat: add feature (too long description that exceeds 50 characters)
```

### 正确示例

```
✅ docs: update README
✅ fix: resolve authentication bug
✅ feat: add user profile
✅ docs: update readme
✅ fix: resolve authentication issue
✅ feat: add user profile page
```

## 工具推荐

### Commitizen

使用 Commitizen 可以帮助你生成符合规范的提交信息：

```bash
# 安装 commitizen
pnpm add -D commitizen cz-conventional-changelog

# 初始化
echo "module.exports = {extends: ['cz-conventional-changelog']}" > .czrc

# 使用 commitizen 提交
pnpm commit
```

### Commitlint

使用 Commitlint 可以验证提交信息是否符合规范：

```bash
# 安装 commitlint
pnpm add -D @commitlint/cli @commitlint/config-conventional

# 配置
echo "module.exports = {extends: ['@commitlint/config-conventional']}" > commitlint.config.js
```

## 参考资料

- [Conventional Commits](https://www.conventionalcommits.org/)
- [Angular Commit Message Format](https://github.com/angular/angular/blob/master/CONTRIBUTING.md#-commit-message-format)
- [Commitizen](https://commitizen-tools.github.io/commitizen/)
- [Commitlint](https://commitlint.js.org/)
