# WebSocket 事件管理架构文档

## 概述

PurrChat 前端采用独立的事件管理架构来处理 WebSocket 事件，确保数据更新与视图组件的自动刷新。

## 架构组件

### 1. WebSocket 服务 (`websocket.ts`)

负责底层的 WebSocket 连接管理：

- **连接管理**：建立、维护和断开 WebSocket 连接
- **消息发送**：向服务器发送消息
- **事件注册**：允许组件注册特定类型的事件处理器
- **重连机制**：自动重连失败的连接

**核心功能**：

```typescript
export const websocketService = new WebSocketService();
export function useWebSocket() {
  return {
    connect,
    disconnect,
    send,
    on, // 注册事件处理器
    off, // 移除事件处理器
  };
}
```

### 2. WebSocket 事件管理器 (`websocketEventManager.ts`)

独立的事件管理器，负责处理所有 WebSocket 事件并自动更新数据结构和视图。

**核心功能**：

- 统一处理所有 WebSocket 事件
- 自动更新 Pinia stores（消息、会话等）
- 提供回调机制通知组件更新
- 维护用户在线状态缓存

**事件类型**：

| 事件类型                      | 数据类型                             | 处理逻辑                                           |
| ----------------------------- | ------------------------------------ | -------------------------------------------------- |
| `new_message`                 | `NewMessageEventData`                | 更新消息 store，触发消息更新回调，通知会话列表更新 |
| `new_friend_request`          | `NewFriendRequestEventData`          | 触发好友请求回调，通知会话列表更新                 |
| `friend_request_update`       | `FriendRequestUpdateEventData`       | 触发好友请求回调，通知会话列表更新                 |
| `new_group_conversation`      | `NewGroupConversationEventData`      | 通知会话列表更新                                   |
| `conversation_member_added`   | `ConversationMemberAddedEventData`   | 通知会话列表更新                                   |
| `conversation_member_removed` | `ConversationMemberRemovedEventData` | 清理消息，通知会话列表更新                         |
| `user_online_status`          | `UserOnlineStatusEventData`          | 更新在线状态缓存，触发在线状态回调                 |
| `bot_workflow_started`        | `BotWorkflowEventData`               | 触发工作流状态回调，显示工作流运行中横幅           |
| `bot_workflow_ended`          | `BotWorkflowEventData`               | 触发工作流状态回调，隐藏工作流运行中横幅           |

**回调机制**：

```typescript
// 会话更新回调
export type ConversationUpdateCallback = (conversation: Conversation) => void;

// 消息更新回调
export type MessageUpdateCallback = (conversationId: string, message: Message) => void;

// 好友请求回调
export type FriendRequestCallback = (request: Friendship) => void;

// 在线状态回调
export type OnlineStatusCallback = (userId: string, online: boolean) => void;

// 工作流状态变更回调
export type WorkflowChangeCallback = (
  event: 'started' | 'ended',
  data: { bot_id: string; bot_name: string; conversation_id: string }
) => void;
```

**使用示例**：

```typescript
import { useWebSocketEventManager } from '@/services/websocketEventManager';

const {
  setCurrentConversation,
  onConversationUpdate,
  onMessageUpdate,
  onFriendRequest,
  onOnlineStatus,
  onWorkflowChange,
} = useWebSocketEventManager();

// 设置当前会话
setCurrentConversation(conversationId);

// 注册回调
onConversationUpdate(async (conversation) => {
  // 会话更新时重新加载会话列表
  await loadConversations();
});

onMessageUpdate((conversationId, message) => {
  // 收到新消息时自动滚动到底部
  if (conversationId === currentConversationId) {
    scrollToBottom();
  }
});

onWorkflowChange((event, data) => {
  // 工作流状态变更时更新 UI
  if (event === 'started') {
    activeWorkflow.value = data;
  } else {
    activeWorkflow.value = null;
  }
});

// 清理回调
offConversationUpdate(callback);
```

### 3. 组件集成

#### HomeView 组件

**职责**：

- 建立 WebSocket 连接
- 注册全局事件回调
- 处理通知和全局数据更新

**实现**：

```typescript
onMounted(async () => {
  await auth.checkAuth();
  if (auth.currentUser.value) {
    connect(); // 连接 WebSocket

    // 注册事件回调
    onConversationUpdate(handleConversationUpdate);
    onMessageUpdate(handleMessageUpdate);
    onFriendRequest(handleFriendRequestUpdate);
  }
});

onUnmounted(() => {
  // 清理回调
  offConversationUpdate(handleConversationUpdate);
  offMessageUpdate(handleMessageUpdate);
  offFriendRequest(handleFriendRequestUpdate);

  disconnect(); // 断开连接
});
```

#### ChatPanel 组件

**职责**：

- 管理当前选中的会话
- 响应消息更新事件
- 自动滚动到最新消息

**实现**：

```typescript
// 设置当前会话
const handleSelectConversation = async (conversation: Conversation) => {
  selectedConversation.value = conversation;
  setCurrentConversation(conversation.id); // 通知事件管理器
  await checkAndLoadIncremental(conversation.id);
};

// 响应消息更新
const handleMessageUpdate = async (conversationId: string, message: Message) => {
  if (conversationId === selectedConversation.value?.id) {
    await nextTick();
    chatWindowRef.value?.scrollToBottom(); // 自动滚动到底部
  }
};
```

#### FriendsPanel 组件

**职责**：

- 响应好友请求更新
- 自动刷新好友列表和待处理请求

**实现**：

```typescript
const handleFriendRequestUpdate = async (friendship: Friendship) => {
  // 重新加载相关数据
  await loadFriends();
  await loadPendingRequests();
  await loadAllFriendRequests();
};

onMounted(async () => {
  // 注册好友请求回调
  onFriendRequest(handleFriendRequestUpdate);
});
```

## 数据流

### 收到新消息

```
WebSocket 事件
  ↓
websocketEventManager.handleNewMessage()
  ↓
messageStore.addMessage()  // 更新消息 store
  ↓
触发 messageUpdateCallbacks  // 如果是当前会话
  ↓
ChatPanel.handleMessageUpdate()
  ↓
chatWindowRef.scrollToBottom()  // 自动滚动到底部
  ↓
触发 conversationUpdateCallbacks
  ↓
HomeView.handleConversationUpdate()
  ↓
loadConversations()  // 重新加载会话列表（更新排序、时间、摘要）
  ↓
ConversationList 重新渲染
```

### 收到好友申请

```
WebSocket 事件
  ↓
websocketEventManager.handleNewFriendRequest()
  ↓
触发 friendRequestCallbacks
  ↓
FriendsPanel.handleFriendRequestUpdate()
  ↓
loadFriends() + loadPendingRequests()  // 刷新好友列表
  ↓
触发 conversationUpdateCallbacks
  ↓
HomeView.handleConversationUpdate()
  ↓
loadConversations()  // 刷新会话列表
```

### 新群聊创建/成员添加

```
WebSocket 事件
  ↓
websocketEventManager.handleNewGroupConversation()
  ↓
触发 conversationUpdateCallbacks
  ↓
HomeView.handleConversationUpdate()
  ↓
loadConversations()  // 刷新会话列表
  ↓
ConversationList 重新渲染（新群聊出现在顶部）
```

## 最佳实践

### 1. 组件注册回调

```typescript
// ✅ 正确：在 onMounted 中注册，在 onUnmounted 中清理
onMounted(() => {
  onMessageUpdate(handleMessageUpdate);
});

onUnmounted(() => {
  offMessageUpdate(handleMessageUpdate);
});

// ❌ 错误：忘记清理回调会导致内存泄漏
```

### 2. 设置当前会话

```typescript
// ✅ 正确：在选择会话时设置当前会话 ID
const handleSelectConversation = (conversation: Conversation) => {
  setCurrentConversation(conversation.id);
  selectedConversation.value = conversation;
};

// ❌ 错误：不设置当前会话 ID，消息更新回调不会被触发
```

### 3. 使用回调而非直接操作

```typescript
// ✅ 正确：使用事件管理器的回调机制
onConversationUpdate(async (conversation) => {
  await loadConversations();
});

// ❌ 错误：直接在 websocketService.on 中处理逻辑
websocketService.on('new_message', async (data) => {
  // 应该在事件管理器中处理，而不是在组件中
});
```

### 4. 在线状态管理

```typescript
// ✅ 正确：使用事件管理器获取在线状态
const { getUserOnlineStatus } = useWebSocketEventManager();
const isOnline = getUserOnlineStatus(userId);

// ❌ 错误：自己维护在线状态，容易不同步
```

## 后端要求

### WebSocket 安全配置

后端通过环境变量配置 WebSocket 安全策略：

| 环境变量 | 默认值 | 说明 |
|---------|--------|------|
| `WS_ALLOWED_ORIGINS` | 空（允许所有） | 逗号分隔的允许 Origin 列表，生产环境必须配置 |
| `WS_ALLOW_QUERY_TOKEN` | `false` | 是否允许通过 URL query 传递 token（已弃用，仅兼容旧客户端） |
| `WS_READ_LIMIT` | `1048576` (1MB) | 单条消息最大字节数 |
| `WS_WRITE_TIMEOUT` | `10s` | 写入超时 |
| `WS_READ_TIMEOUT` | `60s` | 读取超时（Pong 后重置） |
| `WS_PING_INTERVAL` | `54s` | Ping 间隔 |
| `WS_SEND_QUEUE_SIZE` | `256` | 每连接发送队列大小 |

### Token 认证优先级

1. **Cookie**（`purrchat_token`）— 浏览器自动携带，优先级最高
2. **Sec-WebSocket-Protocol 子协议**（`bearer,<token>`）— Tauri 等原生客户端
3. **URL query**（`?token=...`）— 仅当 `WS_ALLOW_QUERY_TOKEN=true` 时启用，记录弃用告警

**禁止**同时使用多种 token 来源。第一个命中的来源即为最终 token，不存在降级。

### Origin 校验

- `CheckOrigin` 使用配置化 allowlist（`WS_ALLOWED_ORIGINS`）
- 空 Origin（非浏览器客户端）默认放行
- 未配置 allowlist 时放行所有 Origin（仅开发环境）
- 生产环境必须配置 allowlist

### 帧边界

每个逻辑事件写入独立 text frame，不合并多个 JSON 到同一 frame。

### 慢消费者与队列溢出

- 每连接有界发送队列（`WS_SEND_QUEUE_SIZE`）
- 队列满时断开连接（close code 1013 Try Again Later），不阻塞 Hub
- 广播时发现队列满的客户端会被收集后统一断开，不修改共享 map

### 关闭码

| Code | 含义 |
|------|------|
| 1000 | 正常关闭 |
| 1001 | 服务端关闭 / Going Away |
| 1008 | 鉴权失败（Policy Violation） |
| 1009 | 消息过大 |
| 1013 | 队列溢出 / 连接数超限（Try Again Later） |

### 指标

通过 `Hub.GetConnectionStats()` 获取：

- 当前连接数、用户数、设备分布
- 累计指标：总连接数、鉴权失败、Origin 拒绝、队列溢出、协议错误、Ping 超时

### 反向代理部署要求

- **Upgrade 头**：Nginx/HAProxy 必须正确传递 `Upgrade` 和 `Connection` 头
- **Origin**：反向代理应传递原始 `Origin` 头，不覆盖
- **超时**：代理读写超时应大于后端 `WS_READ_TIMEOUT` + `WS_PING_INTERVAL`
- **日志脱敏**：如果启用 `WS_ALLOW_QUERY_TOKEN`，代理访问日志必须脱敏 `token` 参数
- **TLS**：生产环境必须使用 `wss://`，Cookie 需设置 `Secure` 和 `HttpOnly`

### 前端重连策略

- 指数退避 + ±25% jitter，上限 30 秒
- 鉴权失败（close code 1008）不重连
- 正常关闭（close code 1000）仅在手动关闭时不重连
- 最大重连次数 10 次

### WebSocket 事件格式

后端需要发送以下类型的 WebSocket 事件：

#### 1. 新消息事件

```json
{
  "type": "new_message",
  "data": {
    "id": "message_id",
    "conversation_id": "conversation_id",
    "sender_id": "sender_id",
    "content": "message content",
    "msg_type": "text",
    "created_at": "2024-01-01T00:00:00Z",
    "sender": {
      "id": "sender_id",
      "username": "username",
      "avatar_url": "avatar_url"
    }
  }
}
```

#### 2. 新好友请求事件

```json
{
  "type": "new_friend_request",
  "data": {
    "conversation_id": "conversation_id",
    "sender_id": "sender_id",
    "status": "pending",
    "sender": {
      "id": "sender_id",
      "username": "username",
      "avatar_url": "avatar_url"
    }
  }
}
```

#### 3. 好友请求更新事件

```json
{
  "type": "friend_request_update",
  "data": {
    "conversation_id": "conversation_id",
    "status": "accepted",
    "action": "accept",
    "user_id": "user_id"
  }
}
```

#### 4. 新群聊创建事件

```json
{
  "type": "new_group_conversation",
  "data": {
    "conversation_id": "conversation_id",
    "name": "group name",
    "created_by": "creator_id",
    "member_count": 5,
    "conversation": {
      // 完整的会话对象（可选）
    }
  }
}
```

#### 5. 会话成员添加事件

```json
{
  "type": "conversation_member_added",
  "data": {
    "conversation_id": "conversation_id",
    "user_id": "user_id",
    "role": "member",
    "added_by": "adder_id",
    "user": {
      // 被添加的用户信息（可选）
    },
    "conversation": {
      // 完整的会话对象（可选）
    }
  }
}
```

#### 6. 会话成员移除事件

```json
{
  "type": "conversation_member_removed",
  "data": {
    "conversation_id": "conversation_id",
    "user_id": "user_id",
    "removed_by": "remover_id",
    "conversation": {
      // 完整的会话对象（可选）
    }
  }
}
```

#### 7. 用户在线状态事件

```json
{
  "type": "user_online_status",
  "data": {
    "user_id": "user_id",
    "online": true,
    "last_seen": "2024-01-01T00:00:00Z"
  }
}
```

#### 8. Bot 工作流状态事件

```json
{
  "type": "bot_workflow_started",
  "data": {
    "bot_id": "bot_id",
    "bot_name": "bot_name",
    "conversation_id": "conversation_id"
  }
}
```

```json
{
  "type": "bot_workflow_ended",
  "data": {
    "bot_id": "bot_id",
    "bot_name": "bot_name",
    "conversation_id": "conversation_id"
  }
}
```

### 在线状态管理

后端需要维护所有用户的在线状态：

1. **连接建立时**：将用户标记为在线
2. **连接断开时**：将用户标记为离线
3. **状态变化时**：向所有相关用户发送 `user_online_status` 事件

**建议实现**：

- 使用 Redis 或内存缓存维护在线状态
- 定期清理过期的在线状态
- 只向相关用户（好友、群成员）发送状态变化事件

## 故障排查

### 问题：消息没有自动滚动到底部

**原因**：

1. 没有设置当前会话 ID
2. ChatWindow 的 ref 没有正确设置
3. scrollToBottom 方法没有被调用

**解决方案**：

```typescript
// 确保在选择会话时设置当前会话 ID
const handleSelectConversation = (conversation: Conversation) => {
  setCurrentConversation(conversation.id); // ← 重要
  selectedConversation.value = conversation;
};

// 确保 ChatWindow 有 ref
<ChatWindow ref="chatWindowRef" ... />
```

### 问题：会话列表没有更新

**原因**：

1. 没有注册 conversationUpdate 回调
2. loadConversations 没有被调用

**解决方案**：

```typescript
// 确保注册了回调
onConversationUpdate(async (conversation) => {
  await loadConversations(); // ← 重新加载会话列表
});
```

### 问题：好友请求没有刷新

**原因**：

1. 没有注册 friendRequest 回调
2. loadPendingRequests 没有被调用

**解决方案**：

```typescript
// 确保注册了回调
onFriendRequest(async (friendship) => {
  await loadFriends();
  await loadPendingRequests(); // ← 重新加载待处理请求
});
```

## 总结

PurrChat 的 WebSocket 事件管理架构通过以下方式确保数据更新与视图自动刷新：

1. **独立的事件管理器**：统一处理所有 WebSocket 事件
2. **自动更新 stores**：自动更新 Pinia stores 中的数据
3. **回调机制**：通过回调通知组件进行视图更新
4. **生命周期管理**：在组件挂载时注册回调，卸载时清理
5. **当前会话跟踪**：跟踪当前选中的会话，实现精确的消息更新

这种架构确保了：

- 数据一致性：所有组件看到的数据是同步的
- 自动刷新：收到事件后自动更新视图
- 性能优化：只更新需要更新的组件
- 可维护性：清晰的职责分离，易于扩展
