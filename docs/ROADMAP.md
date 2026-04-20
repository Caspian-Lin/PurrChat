# PurrChat 开发路线图

> 最后更新: 2026-04-20

---

## 已知问题

### ISSUE-001: R2 r2.dev 域名在中国大陆无法访问

- **影响**: 头像、文件等通过 R2 公开 URL 加载的资源对国内用户不可用
- **现象**: 浏览器报 `ERR_CONNECTION_CLOSED`，需 VPN 才能正常加载
- **根因**: Cloudflare R2 的 `pub-xxx.r2.dev` 子域名被 GFW 阻断，属于网络层面阻断而非配置问题
- **已尝试**: `referrerpolicy="no-referrer"` 无效，确认非防盗链问题
- **待实施方案**:
  - 存储服务代理模式（后端从 R2 拉取再返回前端，服务端不受 GFW 影响，头像文件小开销可控）
  - R2 自定义域名（走 Cloudflare CDN 常规路径，可能改善但不保证）
  - 切换到国内对象存储（阿里云 OSS / 腾讯 COS 等）
- **诊断日志**: 已在 `r2_provider.go`、`file_service.go`、`useAvatarUpload.ts` 中添加 URL 生成日志；前端头像 `<img>` 已添加 `@error` 事件监听

---

## 待实现功能列表

### 高优先级 (Near-term)

#### FEAT-001: 输入状态广播 (Typing Indicator)

- **来源**: 代码中已有占位 (`apps/backend/internal/websocket/handler.go:251`)
  ```go
  case "typing":
      // TODO: 实现输入状态广播
  ```
- **描述**: 用户在聊天窗口输入时，对方看到 "正在输入..." 提示
- **涉及**:
  - 后端: WebSocket `typing` 消息处理、广播给会话内其他用户
  - 前端: ChatPanel 监听 typing 事件、显示输入状态 UI、发送 typing 开始/停止消息
  - 防抖: 用户停止输入 3 秒后自动清除状态

#### FEAT-002: 消息撤回与删除

- **描述**: 用户可撤回自己发送的消息（2 分钟内），或删除本地消息记录
- **涉及**:
  - 后端: 新增 `DELETE /api/messages/:id` 端点、撤回时间窗口校验
  - 数据库: messages 表添加 `deleted_at` 字段（软删除）
  - WebSocket: 广播 `message_deleted` 事件
  - 前端: 消息气泡长按菜单、撤回提示 UI

#### FEAT-003: 离线消息推送

- **描述**: 用户离线期间收到的消息，上线后自动同步
- **涉及**:
  - 后端: 离线消息队列或通过 `last_read_at` 增量拉取
  - 前端: 连接 WebSocket 后自动拉取离线消息
  - 已有基础: `GET /api/messages/incremental` 端点已实现

#### FEAT-004: 消息已读回执

- **描述**: 显示消息是否已被对方阅读
- **涉及**:
  - 数据库: enrollments 表已有 `last_read_at` 字段
  - 后端: 更新 `last_read_at`、查询消息已读状态
  - WebSocket: 广播 `message_read` 事件
  - 前端: 消息气泡下方显示 "已读" 标识、聊天窗口切换时发送已读确认

---

### 中优先级 (Mid-term)

#### FEAT-005: 群聊管理功能

- **描述**: 完善群聊的创建、管理和权限控制
- **涉及**:
  - 群主/管理员角色权限 (owner/admin/member)
  - 修改群名、群头像
  - 踢出成员、解散群聊
  - 转让群主
  - 成员列表查看
  - 已有基础: `POST /api/conversations/group` 已实现基本群聊创建
- **状态**: 已实现群名称编辑、群头像上传、成员角色管理、解散群聊等

#### FEAT-006: 文件消息完整支持

- **描述**: 聊天中支持发送/接收/预览各种类型文件
- **涉及**:
  - 图片消息: 缩略图预览、大图查看、图片压缩
  - 文件消息: 文件图标、大小显示、下载
  - 视频/音频消息: 内联播放
  - 已有基础: 存储服务已支持多类型，`msg_type` 已支持 `file`

#### FEAT-007: 用户在线状态

- **描述**: 显示好友的在线/离线状态
- **涉及**:
  - 后端: WebSocket 连接/断开时广播 `user_online_status` 事件（架构已定义）
  - 前端: 好友列表/会话列表显示在线状态指示器
  - 已有基础: `websocketEventManager.ts` 已有 `user_online_status` 事件处理

#### FEAT-008: 消息搜索

- **描述**: 在聊天记录中搜索关键词
- **涉及**:
  - 后端: PostgreSQL 全文搜索 (tsvector/tsquery)
  - 前端: 搜索输入框、搜索结果高亮显示

#### FEAT-009: 邮箱验证流程

- **描述**: 注册后发送验证邮件，验证后才可使用完整功能
- **涉及**:
  - 后端: 邮件发送服务、验证 token 生成与校验
  - 数据库: `email_verified` 字段已存在
  - 前端: 验证提示页、重新发送验证邮件

#### FEAT-010: 用户资料完善

- **描述**: 丰富的用户个人资料和设置
- **涉及**:
  - 个性签名、生日、地区等扩展字段
  - 资料页面展示
  - 用户名唯一性、修改限制

---

### 低优先级 (Long-term)

#### FEAT-017: 外部 Bot 接入

- **来源**: Bot Studio 计划 Phase 5
- **描述**: 参考 OneBot API 11 协议，允许外部程序通过 WebSocket 接入 PurrChat 作为 Bot 运行，支持反向 WebSocket 连接
- **核心架构**:
  - Bot 类型新增 `external`（与现有 `active`/`disabled` 状态区分）
  - 外部 Bot 配置包含：WebSocket 端点 URL + 鉴权 Token + 可选的 IP 白名单
  - 后端将群聊消息转发到已连接的外部 Bot 端点
  - 外部 Bot 处理后通过 API 回调发送消息到会话
- **连接模式**:
  - **反向 WebSocket（推荐）**: 外部服务器主动连接到 PurrChat，保持长连接
    - PurrChat 新增 `/api/external-bot/ws` WebSocket 端点
    - 外部 Bot 通过 `Authorization: Bearer <token>` 鉴权
    - 连接建立后，PurrChat 推送会话消息，外部 Bot 回复
  - **HTTP 回调（备选）**: PurrChat 通过 POST 请求推送消息到外部 Bot 的 HTTP 端点
    - 适用于无法维持长连接的场景
    - 外部 Bot 收到请求后通过 API 回复
- **OneBot API 11 兼容性**:
  - 消息格式兼容 CQ 码 / OneBot 消息段格式
  - 支持的动作（Action）:
    - `send_message` — 发送消息到指定会话
    - `send_private_msg` — 发送私聊消息
    - `send_group_msg` — 发送群聊消息
    - `get_group_msg_history` — 获取群聊历史消息
    - `get_group_info` — 获取群聊信息
    - `get_group_member_list` — 获取群成员列表
    - `get_login_info` — 获取 Bot 登录信息
  - 事件推送（Event）:
    - `message` — 新消息事件（包含群聊/私聊消息）
    - `notice` — 通知事件（成员加入/退出等）
- **数据库变更**:
  - `bots` 表新增字段:
    - `bot_type VARCHAR(20) DEFAULT 'builtin'` — 区分内置 Bot (`builtin`) 和外部 Bot (`external`)
    - `ws_endpoint TEXT` — 反向 WebSocket 连接的回调地址（HTTP 模式使用）
    - `ws_token TEXT` — 鉴权 Token（加密存储）
    - `ip_whitelist TEXT[]` — IP 白名单（PostgreSQL 数组类型）
  - 新增 `external_bot_connections` 表 — 记录活跃的外部 Bot 连接:
    ```sql
    CREATE TABLE external_bot_connections (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        bot_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
        connected_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        last_heartbeat TIMESTAMP,
        remote_addr TEXT,
        status VARCHAR(20) NOT NULL DEFAULT 'connected'  -- connected / disconnected
    );
    ```
- **后端模块**:
  - `apps/backend/internal/externalbot/`
    - `manager.go` — 外部 Bot 连接管理器（连接生命周期、心跳检测、重连策略）
    - `protocol.go` — OneBot 11 协议编解码（消息段解析、Action 路由、Event 构建）
    - `handler.go` — WebSocket 连接处理器（鉴权、消息收发、错误处理）
    - `rate_limiter.go` — 消息速率限制器（每 Bot 独立令牌桶）
  - 修改 `apps/backend/internal/botengine/engine.go`:
    - `processMessage` 中检测外部 Bot 部署，通过 `manager.ForwardMessage` 转发
  - 新增路由:
    - `GET /api/external-bot/ws?token=<token>` — 反向 WebSocket 连接端点
    - `POST /api/external-bot/callback` — HTTP 回调端点
- **安全设计**:
  - 每个外部 Bot 独立鉴权 Token，Token 加密存储在数据库
  - 消息转发速率限制（默认 20 条/分钟，可配置）
  - IP 白名单支持（可选，为空时不限制）
  - 连接心跳检测（30 秒间隔，超时 90 秒自动断开）
  - 消息大小限制（单条消息最大 10KB）
  - 连接数限制（单个外部 Bot 最多 3 个并发连接）
- **前端变更**:
  - `BotEditor.vue` — 创建/编辑 Bot 时可选类型（内置 / 外部）
  - 外部 Bot 配置表单: WebSocket 端点 URL、Token、IP 白名单
  - BotStudioPanel Bot 列表中外部 Bot 显示连接状态指示器（绿点=已连接，红点=断开）
  - 新增 `ExternalBotConfig.vue` — 外部 Bot 配置组件
- **验证方案**:
  1. 创建外部 Bot，配置 Token 和端点
  2. 外部程序通过反向 WebSocket 连接到 PurrChat
  3. 在群聊中发送消息，外部 Bot 收到并回复
  4. 断开外部 Bot 连接，验证心跳检测和自动清理
  5. 速率限制验证：超过限制后消息被丢弃
  6. IP 白名单验证：非白名单 IP 连接被拒绝

#### FEAT-011: 消息多媒体扩展

- 语音消息: 录音、播放
- 视频通话: WebRTC 集成
- 表情包系统: 自定义表情、GIF 搜索

#### FEAT-012: 通知系统

- 桌面通知: Notification API
- 声音提醒: 新消息/好友请求
- 通知免打扰模式
- 未读消息计数徽标

#### FEAT-013: 消息转发与引用

- 消息转发: 选择消息转发到其他会话
- 消息引用: 回复特定消息（引用显示）
- 消息多选操作

#### FEAT-014: 黑名单与屏蔽

- 屏蔽用户: 不接收消息和好友请求
- 举报用户/消息
- 隐私设置: 谁可以添加我为好友

#### FEAT-015: 桌面端增强 (Tauri)

- 系统托盘图标
- 最小化到托盘
- 开机自启动
- 原生通知
- 文件拖拽发送
- 多窗口查看聊天会话

#### FEAT-016: 国际化 (i18n)

- 多语言支持
- 日期/时间本地化
- 时区自动检测

---

## 功能开发进度

| ID       | 功能                      | 优先级 | 状态          | PR  |
| -------- | ------------------------- | ------ | ------------- | --- |
| FEAT-001 | 输入状态广播              | 高     | 待实现        | -   |
| FEAT-002 | 消息撤回与删除            | 高     | 待实现        | -   |
| FEAT-003 | 离线消息推送              | 高     | 待实现        | -   |
| FEAT-004 | 消息已读回执              | 高     | 待实现        | -   |
| FEAT-005 | 群聊管理功能              | 中     | ✅ 基础已完成 | -   |
| FEAT-006 | 文件消息完整支持          | 中     | 待实现        | -   |
| FEAT-007 | 用户在线状态              | 中     | 待实现        | -   |
| FEAT-008 | 消息搜索                  | 中     | 待实现        | -   |
| FEAT-009 | 邮箱验证流程              | 中     | 待实现        | -   |
| FEAT-010 | 用户资料完善              | 中     | 待实现        | -   |
| FEAT-011 | 消息多媒体扩展            | 低     | 待实现        | -   |
| FEAT-012 | 通知系统                  | 低     | 待实现        | -   |
| FEAT-013 | 消息转发与引用            | 低     | 待实现        | -   |
| FEAT-014 | 黑名单与屏蔽              | 低     | 待实现        | -   |
| FEAT-015 | 桌面端增强                | 低     | 待实现        | -   |
| FEAT-016 | 国际化                    | 低     | 待实现        | -   |
| FEAT-017 | 外部 Bot 接入 (OneBot 11) | 低     | 规划中        | -   |

---

## 已完成功能

- [x] 用户注册与登录 (JWT 认证)
- [x] 个人资料修改
- [x] 用户搜索 (UID/邮箱/手机号)
- [x] 好友系统 (请求/接受/拒绝)
- [x] 一对一聊天
- [x] 群聊基础 (创建群聊)
- [x] 实时消息 (WebSocket)
- [x] 头像上传 (R2/MinIO 存储)
- [x] 背景图上传
- [x] 消息缓存 (IndexedDB)
- [x] 增量消息加载
- [x] 多客户端支持 (Web/Tauri)
- [x] 时区统一 (UTC 存储)
- [x] 头像/背景旧文件自动清理 (cleanupOldFile bug 修复: 查询排除新文件 ID)
- [x] Bot Studio — Bot CRUD、部署到会话、触发与回复系统 (Phase 1-2)
- [x] Bot 特殊模式 (Agent) — 事件链引擎、Python 沙箱、DAG 编辑器、调试面板 (Phase 3)
- [x] Bot 系统消息 — Bot 部署/移除/模式启停的系统通知消息 (Phase 4 Track B)
- [x] Bot 发现与分享 — 分页搜索、部署统计、部署到群聊弹窗 (Phase 4 Track A)
