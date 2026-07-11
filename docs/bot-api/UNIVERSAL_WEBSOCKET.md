# Universal WebSocket

PurrChat 的 Universal WebSocket 为外部 Bot 提供 OneBot 风格的双向传输。当前版本端点为 `GET /api/bot/v1/ws`。

## 连接与认证

- 客户端必须使用 `Authorization: Bearer <bot-credential>`。Cookie、query token 和所有 query 参数均被拒绝。
- 生产环境必须通过 `wss://` 连接。服务可以在可信反向代理后终止 TLS，但代理到应用的链路也应位于受控网络。
- credential 在握手时解析为不可变 principal：`bot_id`、`identity_id`、`credential_id`。连接内不能切换身份。
- credential rotate/revoke 会断开该 credential 建立的全部连接；Bot disabled/delete 会断开该 Bot 的全部连接。
- installation pause/uninstall 不会全局断开 Bot。#68、#69 实现后，每次 Action 和 Event 必须按对应 installation scope 重新授权。

连接成功后服务发送 `meta_event/lifecycle/connect`。服务使用 WebSocket Ping/Pong 检测半开连接，并可周期发送 `meta_event/heartbeat`。客户端应由 WebSocket 库自动回复 Pong。

## 消息模型

同一文本连接承载 OneBot Event、Action Request 和 Action Response。二进制消息不受支持。

Action 可以并发执行，响应顺序不保证与请求顺序一致。响应会原样保留请求中的任意 JSON `echo`，客户端必须使用 `echo` 关联请求。registry 中尚未实现的 Action 返回稳定的 `10005` unsupported 响应；未知或明确拒绝的 Action 保留 registry 定义的错误语义。

`PublishBotEvent` 会以 best-effort 方式向该 Bot 当前进程内的全部连接广播。发送成功仅表示进入各连接的内存队列，不表示远端已经处理；断线期间不持久化、不补发。多个连接可能重复收到同一事件，应使用 `event_id` 去重。

## 默认限制

| 限制 | 默认值 |
| --- | ---: |
| 全局连接 | 1000 |
| 每 Bot 连接 | 3 |
| 每连接并发 Action | 8 |
| 每连接发送队列 | 64 条 |
| 单 frame / 单消息大小 | 16 KiB / 16 KiB |
| 读/Pong 超时 | 90 秒 |
| 写超时 | 10 秒 |
| Action 超时 | 30 秒 |
| Ping / heartbeat | 30 秒 |

限制是服务端配置，部署环境可以收紧。队列满时服务主动断开慢消费者，避免单个连接造成无界内存增长。

## 关闭码

| Code | 含义 |
| ---: | --- |
| 4000 | 非法消息或不支持的 frame 类型 |
| 1009 | frame/message 超过大小限制（WebSocket 标准码） |
| 4002 | 有界发送队列溢出 |
| 4003 | 每 Bot 或全局连接上限 |
| 4004 | credential rotate/revoke |
| 4005 | Bot disabled/delete |
| 4006 | 服务优雅关闭 |
| 4007 | 保留给需要关闭连接的 Action timeout 策略；当前 Action timeout 返回失败响应，连接保持可用 |

标准 WebSocket 关闭码仍可能用于正常关闭、网络错误或底层协议错误。

## 状态与运维

Bot owner 可使用 JWT 请求 `GET /api/bots/:id/ws-status`，获取在线状态、连接数、最近心跳和最近错误。Manager 还提供无锁原子指标快照，包括连接、消息、Action、协议错误和队列溢出计数。指标与状态均为当前进程内存数据，重启后清零。

当前 Manager 是单实例内存实现。多实例部署必须采用以下方案之一：

- 对同一 Bot 使用 sticky session，让连接和事件生产者落在同一实例。
- 引入共享 broker，将 Bot Event 和 credential/Bot lifecycle 断连通知扇出到所有实例。

仅对 WebSocket 做 sticky session 但让事件在其他实例产生，会造成静默漏投；生产部署必须同时处理事件路由。
