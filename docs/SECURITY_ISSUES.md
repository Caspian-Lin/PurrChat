# PurrChat 安全问题待修复列表

> 最后更新: 2026-04-15
> 基于 2026-04-15 的全面安全审计

---

## P0 - 紧急 (Critical)

### SEC-001: CORS 配置允许任意来源且携带凭证

- **文件**: `apps/backend/cmd/server/main.go:200-201`
- **问题**: `Access-Control-Allow-Origin: *` 与 `Access-Control-Allow-Credentials: true` 同时使用
- **风险**: 任何恶意网站都可以携带用户凭据发送跨域请求，实施 CSRF 攻击
- **修复方案**:
  ```go
  // 替换为白名单校验
  origin := c.Request.Header.Get("Origin")
  allowedOrigins := []string{"https://purrchat.com", "https://app.purrchat.com"}
  for _, allowed := range allowedOrigins {
      if origin == allowed {
          c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
          break
      }
  }
  ```

### SEC-002: 全局零速率限制

- **涉及文件**: 所有 handler、`apps/backend/cmd/server/main.go` 全路由
- **问题**: 整个后端没有任何速率限制中间件
- **风险**: DDoS 攻击、暴力破解、批量注册、消息轰炸、API 滥用
- **影响端点**:
  - `POST /api/register` — 批量注册
  - `POST /api/login` — 暴力破解密码
  - `GET /api/users/search` — 用户枚举
  - `POST /api/messages` — 消息轰炸
  - `POST /api/friends/request` — 好友请求洪水
  - `GET /api/ws` — WebSocket 连接洪水
- **修复方案**: 添加 per-IP 和 per-User 速率限制中间件，建议使用 `golang.org/x/time/rate` 或 `github.com/ulule/limiter`

### SEC-003: JWT 默认密钥不阻止启动

- **文件**: `apps/backend/pkg/config/config.go:54,98-100`
- **问题**: 未设置 `JWT_SECRET` 时仅打印 WARNING 但照常运行，默认值 `default_secret_change_me` 可被攻击者利用伪造任意用户身份
- **修复方案**: 将 `log.Println` 改为 `log.Fatal`，生产环境强制要求设置强密钥

---

## P1 - 高优先级 (High)

### SEC-004: WebSocket Origin 完全不校验

- **文件**: `apps/backend/internal/websocket/handler.go:20-22`
- **问题**: `CheckOrigin` 直接 `return true`，任何网站都可以建立跨站 WebSocket 连接
- **风险**: 恶意网站可窃取实时消息、窃听用户聊天内容
- **修复方案**: 校验请求 Origin 是否在允许列表中

### SEC-005: 用户搜索可枚举任意用户隐私信息

- **文件**: `apps/backend/internal/handlers/chat_handler.go:36-81`
- **问题**: 搜索端点返回完整用户信息（邮箱、手机号），且无长度限制
- **风险**: 通过遍历邮箱前缀/手机号前缀枚举全量用户隐私
- **修复方案**:
  1. 对邮箱/手机号结果进行脱敏（如 `138****5678`）
  2. 限制搜索 query 长度（已有 model 校验但 handler 绕过了）
  3. 考虑仅允许搜索好友或添加过好友的用户

### SEC-006: 前端 Token 存储在 localStorage

- **文件**: `apps/frontend/src/stores/auth.ts:68-69`
- **问题**: JWT 存储在 `localStorage`，任何 XSS 漏洞都可窃取令牌
- **修复方案**: 改用 HttpOnly + Secure + SameSite=Strict 的 Cookie 存储 Token

### SEC-007: 消息内容无 XSS 过滤

- **文件**: `apps/backend/internal/handlers/chat_handler.go:336`
- **问题**: 消息内容无长度限制、无 HTML 转义、无 XSS 清理
- **风险**: 存储型 XSS — 恶意用户注入 `<script>` payload，在其他用户客户端执行
- **修复方案**:
  1. 后端: 对所有用户输入进行 HTML 转义
  2. 前端: 使用 `v-html` 之外的方式渲染消息，或使用 DOMPurify 消毒

### SEC-008: 注册流程无反自动化措施

- **文件**: `apps/backend/internal/handlers/auth_handler.go:37-65`
- **问题**: 无 CAPTCHA、无邮箱验证强制、无 IP 频率限制、无设备指纹
- **风险**: 脚本批量注册垃圾账号，发送垃圾好友请求和消息
- **修复方案**:
  1. 注册添加 CAPTCHA 验证
  2. 强制邮箱验证后才可使用完整功能
  3. 结合 SEC-002 的速率限制

### SEC-009: WebSocket Token 通过 URL Query 传递

- **文件**: `apps/backend/internal/websocket/handler.go:81`
- **问题**: Token 作为 URL query 参数传递，会被记录在服务器日志、代理日志、浏览器历史中
- **修复方案**: 改用 WebSocket 子协议 (Sec-WebSocket-Protocol) 或在连接建立后通过首条消息发送 Token

---

## P2 - 中优先级 (Medium)

### SEC-010: 文件下载无权限校验

- **文件**: `apps/storage/internal/services/file_service.go:156-187`
- **问题**: `GetDownloadURL` 只验证文件存在和已确认，不验证请求者是否有权访问该文件
- **风险**: 知道 `file_id` 的任何认证用户可下载任意用户的聊天文件
- **修复方案**: 添加文件归属校验，或根据文件分类检查访问权限（头像=公开，聊天文件=会话成员）

### SEC-011: 错误信息泄露内部细节

- **涉及文件**:
  - `apps/backend/internal/handlers/auth_handler.go:43,89`
  - `apps/backend/internal/handlers/chat_handler.go:99,341,369`
- **问题**: `err.Error()` 直接返回给客户端，可能包含 SQL 错误、表名、列名等内部信息
- **修复方案**: 对外返回通用错误消息，详细错误仅记录到日志

### SEC-012: 缺少安全响应头

- **文件**: `apps/backend/cmd/server/main.go`
- **问题**: 未设置任何安全相关的 HTTP 响应头
- **修复方案**: 添加以下安全头
  ```
  Content-Security-Policy: default-src 'self'
  X-Content-Type-Options: nosniff
  X-Frame-Options: DENY
  X-XSS-Protection: 1; mode=block
  Strict-Transport-Security: max-age=31536000; includeSubDomains
  Referrer-Policy: strict-origin-when-cross-origin
  ```

### SEC-013: 用户信息查询无权限控制

- **文件**: `apps/backend/internal/handlers/chat_handler.go:608-646`
- **问题**: `GET /api/users/:id` 和 `GET /api/users/uid/:uid` 任何认证用户可查询任意用户完整信息
- **修复方案**: 对非好友关系用户返回脱敏信息，仅返回公开字段（用户名、头像、UID）

### SEC-014: 好友请求无频率限制

- **文件**: `apps/backend/cmd/server/main.go:286`
- **问题**: 可对任意用户发送好友请求，无发送频率限制和总数限制
- **修复方案**: 添加 per-user 发送频率限制和每日上限

### SEC-015: WebSocket 消息无频率限制

- **文件**: `apps/backend/internal/websocket/handler.go:148-166`
- **问题**: 虽然有 512 字节的读取限制，但客户端可以无限频率发送小消息
- **修复方案**: 在 `readPump` 中添加消息频率限制

---

## 修复进度追踪

| ID | 标题 | 优先级 | 状态 | PR |
|----|------|--------|------|----|
| SEC-001 | CORS 白名单 | P0 | 待修复 | - |
| SEC-002 | 全局限流 | P0 | 待修复 | - |
| SEC-003 | JWT 密钥强制 | P0 | 待修复 | - |
| SEC-004 | WS Origin 校验 | P1 | 待修复 | - |
| SEC-005 | 搜索脱敏 | P1 | 待修复 | - |
| SEC-006 | Token 存储安全 | P1 | 待修复 | - |
| SEC-007 | XSS 过滤 | P1 | 待修复 | - |
| SEC-008 | 注册反自动化 | P1 | 待修复 | - |
| SEC-009 | WS Token 传递方式 | P1 | 待修复 | - |
| SEC-010 | 文件下载权限 | P2 | 待修复 | - |
| SEC-011 | 错误信息脱敏 | P2 | 待修复 | - |
| SEC-012 | 安全响应头 | P2 | 待修复 | - |
| SEC-013 | 用户信息权限 | P2 | 待修复 | - |
| SEC-014 | 好友请求限流 | P2 | 待修复 | - |
| SEC-015 | WS 消息频率 | P2 | 待修复 | - |

---

## 安全方面的正面实践（无需修改）

- SQL 注入防护: 所有查询使用参数化 (`$1, $2...`)，动态表名通过 PG 函数处理
- 密码安全: salt + bcrypt 哈希，`password_hash`/`salt` 标记 `json:"-"` 不序列化
- 主键设计: 使用 UUID 防止 IDOR 顺序遍历
- 文件上传: 两阶段上传 (Request → Confirm) + content type/file size 校验
