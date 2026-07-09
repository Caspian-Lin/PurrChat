# Bot App 化权限模型(ADR)

- 状态:已确认,待实现(issue #12)
- 日期:2026-07-09
- 关联:`docs/bot-engine/BOT_SYSTEM_AUDIT_2026-07-09.md` 第八节、issue #12 / #19 / #20

## 背景

当前 Bot 系统把身份、安装、权限、生命周期四件事耦合在 `users + bots + friendships + enrollments + bot_deployments` 五张表里,核心痛点:

1. **身份混入账号**:Bot 是 `users` 表一行(`is_bot=true`)+ `bots` 表一行,共用 UUID;可被加好友、可作 enrollment 成员、可作 sender_id,但不可登录。
2. **安装分裂为三套**:`friendship`(私聊 Bot)+ `enrollment`(群聊 Bot 成员)+ `bot_deployments`(部署审计)职责重叠,会出现「已入群无部署记录」的不一致。
3. **权限缺失**:无 RBAC、无 capability、无 scope,全靠 service 层 if-else;Bot 能读什么、发什么、调什么外部 API 完全不声明、不授权、不限制。
4. **visibility 三值混合三个正交语义**:`private|public|global` 混合了可发现性、安装权限、系统所有权。
5. **public Bot 好友请求是死路**:加好友进 pending,但 friend_id 是 Bot(不可登录),owner 非友谊参与方,无人能审批。
6. **secret 明文**:`mechanism_config` JSONB 明文存 LLM/Dify/n8n/tool 的 API key、webhook URL、密码;`/execute` 原样发给零鉴权的 TS bot-engine。
7. **调用日志隐私**:`bot_call_logs` 记 trigger_message 原文,owner 可跨会话查看他人私聊消息。

## 决策

### 1. 三分离模型

把 Bot 拆成三个正交实体,参考 Discord 的 Application / Bot User / Installation 分层,适配 PurrChat 会话语义。

#### BotApp(应用容器,owner 拥有)

```
bot_apps(
  id              uuid primary key,
  owner_id        uuid not null references users(id),
  name            text not null,
  avatar_url      text,
  description     text,
  bot_type        text not null default 'workflow',   -- builtin | workflow | external
  discoverability text not null default 'unlisted',    -- unlisted | listed | featured
  is_system       boolean not null default false,
  status          text not null default 'active',      -- active | disabled | suspended
  published_version int,                               -- 当前发布的 workflow 版本号
  requested_capabilities text[],                      -- 发布时由工作流节点推导
  created_at      timestamptz not null default now(),
  updated_at      timestamptz not null default now()
)
```

- `discoverability` 只表达「能否被搜索」:`unlisted`(默认,不可搜) / `listed`(可搜) / `featured`(官方推荐)。
- `bot_type` 预留 `external`,为 OneBot 等外部接入(#20)铺路。
- `is_system` 标记系统拥有的 Bot(原 `global` 语义)。

#### BotIdentity(系统身份投影,不可登录不可好友)

```
bot_identities(
  app_id        uuid primary key references bot_apps(id),
  user_id       uuid not null unique references users(id),  -- 消息 sender_id 用
  display_name  text not null,
  avatar_url    text
)
```

- `user_id` 指向 `users` 表投影行,仅用于 `message.sender_id`、WebSocket 推送。
- **不可登录**(`auth_service.go:122` 已有 is_bot 拒绝)、**不参与好友审批**、**不进 friendships**。
- 保留 users 表投影是必要的:消息分表、enrollment、WS 推送都依赖 sender_id 指向 users。

#### BotInstallation(安装,统一替代 friendship + enrollment + deployment)

```
bot_installations(
  id                    uuid primary key,
  app_id                uuid not null references bot_apps(id),
  installed_by          uuid not null references users(id),
  target_type           text not null,          -- user | conversation
  target_id             uuid not null,          -- user_id 或 conversation_id
  granted_capabilities  text[] not null,        -- 实际授予(<= requested,安装者可缩减)
  diagnostics_consent   text not null default 'denied',  -- denied | granted
  status                text not null default 'active',   -- active | paused | disabled
  config                jsonb,                  -- 安装级覆盖配置
  installed_at          timestamptz not null default now(),
  updated_at            timestamptz not null default now(),
  unique(target_type, target_id, app_id)
)
```

- `target_type=user` 解决私聊,`target_type=conversation` 解决群聊,一套机制替代三套并行。
- Bot 仍作为 `enrollment(member)` 加入会话(消息路由权威来源是 enrollment,`engine.go:178`),但「安装」的真实记录是 installation 表。
- `bot_deployments` 表弃用:回填为 `installation(target_type=conversation)` 后停止写入,P3 删除。

### 2. Capability 权限模型

声明 → 授权 → 运行时强制校验,三层。

#### capability 集合

| capability | 由哪些节点触发 | 说明 |
|---|---|---|
| `messages:read_trigger` | trigger 节点 | 读取触发消息(几乎所有 Bot 都需要) |
| `messages:read_history` | history 节点 / context window | 读取上下文历史(安装者可关闭) |
| `messages:send` | reply / template | 发送回复 |
| `members:read` | (预留) | 读取成员列表 |
| `network:external` | llm / dify / n8n / tool / external | 数据外发到第三方 |
| `secrets:use` | 引用 `secrets.*` 的节点 | 使用 owner 配置的 secret |

#### requested_capabilities(发布时推导)

Bot 发布工作流时,`workflow-engine` 遍历节点图,自动产出所需 capability 集合,写入 `bot_apps.requested_capabilities`。例如:

- 含 llm/dify/n8n/tool 节点 → `network:external`
- 含 history 节点 → `messages:read_history`

工作流不能超出声明的 capability 执行。

#### granted_capabilities(安装时授权)

安装时展示 `requested_capabilities`,安装者可**缩减**授予(如关掉 `messages:read_history`)。实际授予的存 `bot_installations.granted_capabilities`。

#### 运行时强制校验(P1 上线)

`workflow-engine` 执行每个节点前,检查 `granted_capabilities ⊇ node.required_capability`。不满足则拒绝执行该节点。这是纵深防御:即使工作流声明了,实际授予的才生效。

### 3. 诊断数据共享(diagnostics_consent)

替代原来的「基础披露」。安装时安装者决定是否给 Bot owner 查看诊断数据,默认隐私优先。

```
bot_installations.diagnostics_consent: denied(默认) | granted
```

- **denied(默认)**:owner 只见执行元数据(bot_id、会话、触发时间、耗时、成功/失败、机制名),**不见任何消息内容**。
- **granted**:owner 可见**执行窗口**——从触发某机制的消息开始,到该机制执行结束(回复发出)期间会话内的所有消息。无论 owner 是否是该会话成员,安装者主动授权即生效。

`bot_call_logs` 记录规则:
- denied:只记元数据,不记 `trigger_message` / `context_messages` 内容。
- granted:记执行窗口完整消息。

不再需要单独的「调用日志脱敏」逻辑——denied 直接不记内容,比脱敏更干净。

**外发 Bot 强制 granted 且不可关**:当 Bot 声明 `network:external`,`diagnostics_consent` 强制为 `granted`,安装者不可改。理由:数据已必然到达 owner(经其配置的第三方),在 PurrChat 侧强行隐藏只是自欺,且造成 owner 调试困难。

### 4. 可见性解耦

把 `private | public | global` 拆成三个正交维度:

| 维度 | 字段 | 取值 |
|---|---|---|
| 可发现性 | `discoverability` | unlisted / listed / featured |
| 安装权限 | `installability`(由 bot_type + owner 推导) | owner_only / any_user / admin_only |
| 系统标记 | `is_system` | bool |

旧值映射:`private→(unlisted, owner_only)`、`public→(listed, any_user)`、`global→(listed, any_user, is_system)`。

### 5. Secret 治理

当前痛点:`mechanism_config` 明文存第三方 API key/webhook URL/密码,可经 `/execute` 外泄。

#### 方案:应用层 AES-256-GCM + 环境变量主密钥

- 封装 `SecretCipher` 接口:`Encrypt(plaintext)→ciphertext` / `Decrypt(ciphertext)→plaintext`。
- 实现:AES-256-GCM + 每 secret 独立随机 IV(Nonce),主密钥从环境变量 `PURRCHAT_MASTER_KEY` 读取。
- 不引入云 KMS / Vault(当前阶段无 Docker、快速迭代、本地开发为主;Cloudflare 无独立 KMS 产品;AWS/GCP KMS 绑厂商)。
- 接口抽象,未来上云后可换 KMS 后端,不破坏调用方。

#### Secret 引用机制

- BotApp 有独立的 secret store(owner 管理),secret 加密存储。
- 工作流配置里用 `secrets.<name>` 引用代替明文,运行时 `workflow-engine` 注入实际值。
- 需要 `secrets:use` capability。
- export 工作流时,引用不导出明文——secret 永不离开 owner 的 secret store。

#### 出站端点治理

BotApp 级域名白名单。引擎调用外部 URL 前校验 domain 是否在白名单内。防 SSRF + 防数据任意外泄。

### 6. 服务间鉴权

当前 TS bot-engine 零鉴权(`server.ts:15` CORS `origin:'*'`,`/execute` 无 auth)。改造:

- Go→TS 请求加 shared-secret header(或 mTLS)。
- 去掉 CORS 全开,限定为后端来源。
- `/execute` 校验请求来源合法。

### 7. 安全提示与责任划分

#### 安装时提示

| Bot 类型 | 提示 |
|---|---|
| 所有 Bot 安装 | 展示选项「允许 Bot 创建者查看本会话消息以改进 Bot(诊断数据)」(默认**关**) |
| 声明 `network:external` | 额外强制展示外发警示;`diagnostics_consent` 强制 granted |

外发警示文案:

```
⚠ 此 Bot 会将对话内容发送到外部服务

该 Bot 由 @<owner> 创建,工作流会将本会话消息发送到外部端点以生成回复
(LLM 服务 / 外部自动化平台 / 自定义 webhook)。

你的消息内容将离开 PurrChat,由 Bot 创建者所选第三方接收与处理。
请仅在你信任该 Bot 创建者及第三方的情况下安装。
```

#### 群聊安装广播

安装声明 `network:external` 的 Bot 到群聊后,强制向全体成员发系统消息告知外发。

#### 责任三层划分

| 主体 | 责任 |
|---|---|
| Bot Owner(应用作者) | 配置的外部端点/key/数据用途;如实声明数据去向;选择合规第三方;对 Bot 回复内容与数据外发行为负责;遵守 GDPR/PIPL 等 |
| Installer(安装者/群主) | 安装时审阅 capabilities;在群聊里代表群成员授权 Bot 读取群消息;可随时卸载/暂停 |
| PurrChat 平台 | 提供 capability 声明与授权技术基座;安装 UI 强制展示数据外发提示;secret 加密存储;服务间鉴权;调用日志按 diagnostics_consent 控制。**作为传输管道,不对 Bot 创建者所选第三方的数据处理行为负责**(ToS 明确) |

#### Capability 详情透明化

Bot 详情页用权限图标展示 capability 明细(像手机 App 权限):✉️读消息 / 📤发回复 / 🌐访问外部网络 / 👥读成员。

### 8. 外部接入预留(OneBot 等)

`bot_type=external`,走反向连接(外部主动连入 PurrChat,而非 PurrChat 调外部):

- OneBot 11(#20):外部实现 OneBot 接口,PurrChat 作 Action 端点接收发送请求。
- capability 仍由 PurrChat 侧在处理请求前校验 `granted_capabilities`(尤其 `messages:send`)。
- 连接 token(外部持有)存 secret store,不走明文配置。
- 外部 Bot 必然外发消息历史,`network:external` 不可缩减,`diagnostics_consent` 强制 granted。

## 取舍

| 取舍 | 选择 | 理由 |
|---|---|---|
| 保留 users 表投影 vs 独立 Bot identity 表 | 保留投影 | 消息分表/enrollment/WS 推送都依赖 sender_id 指向 users;重写成本远大于保留一个投影行 |
| 统一 installation vs 保留三套表 | 统一 installation | 三套表职责重叠导致不一致;统一后语义清晰 |
| capability 强制校验 P1 vs P2 | P1 强制 | 安全优先,工作流不满足 capability 直接报错,逼发布时声明完整 |
| secret KMS vs 应用层加密 | 应用层 AES | 当前阶段无 Docker、快速迭代;Cloudflare 无 KMS;不绑厂商;接口可演进 |
| diagnostics 默认 denied vs granted | denied | 隐私默认最强;安装者主动授权才共享 |
| 外发 Bot diagnostics 强制 vs 可关 | 强制 granted | 数据已必然到 owner(经第三方),隐藏无意义 |

## 影响与迁移路线

### P1(本 issue #12 子任务)

1. 新建 `bot_apps / bot_identities / bot_installations` 表。
2. Installation API(create / get / list / pause / uninstall / reauthorize)。
3. capability 声明(发布推导)+ 强制校验。
4. 安全提示(安装 UI、群聊广播)。
5. diagnostics_consent 字段与 call_logs 联动。
6. SecretCipher 接口 + secret 引用机制骨架。
7. 服务间鉴权(Go↔TS)。
8. 回填现有 `bot_deployments` / Bot `friendship` → installations。
9. 新 Bot 不再建 friendship;移除 friend_service Bot 分支。

### P2 / P3

- 弃用并删除 `bot_deployments` 表。
- 移除 deprecated Go workflow 引擎(#18)。
- 出站端点白名单完整实现。
- OneBot 适配层(#20)。
