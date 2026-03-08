# WebSocket 调试指南

## 概述

本文档提供调试 WebSocket 事件和消息更新的指南，帮助您诊断和解决 WebSocket 相关问题。

## 调试工具

### 1. WebSocket 调试页面

访问 `/ws-debug` 路由可以打开 WebSocket 调试工具页面。

**功能**：
- 查看 WebSocket 连接状态
- 设置当前会话 ID
- 发送测试消息
- 查看实时事件日志

**使用步骤**：
1. 登录系统
2. 访问 `http://localhost:5173/ws-debug`
3. 点击"连接 WebSocket"按钮
4. 设置当前会话 ID（可选）
5. 发送测试消息
6. 查看事件日志

### 2. 浏览器开发者工具

#### 查看网络请求

1. 打开浏览器开发者工具（F12）
2. 切换到"网络"（Network）标签
3. 筛选 WS（WebSocket）请求
4. 查看 WebSocket 连接和消息

#### 查看控制台日志

所有 WebSocket 相关的日志都会在控制台中显示，使用以下前缀过滤：

- `[WebSocket]` - WebSocket 服务日志
- `[WebSocketEventManager]` - WebSocket 事件管理器日志
- `[MessageStore]` - 消息 store 日志
- `[ChatPanel]` - ChatPanel 组件日志
- `[HomeView]` - HomeView 组件日志
- `[FriendsPanel]` - FriendsPanel 组件日志

### 3. Vue DevTools

1. 安装 Vue DevTools 浏览器扩展
2. 打开 Vue DevTools
3. 查看 Pinia stores 中的状态变化
4. 查看 message store 中的消息数据

## 调试流程

### 问题：消息没有自动显示在聊天窗口

**调试步骤**：

1. **检查 WebSocket 连接**
   - 打开调试页面 `/ws-debug`
   - 确认 WebSocket 已连接
   - 查看连接状态是否为"已连接"

2. **检查当前会话设置**
   - 在调试页面设置当前会话 ID
   - 确认设置的会话 ID 与收消息的会话 ID 一致

3. **查看事件日志**
   - 发送一条测试消息
   - 查看日志中是否有以下事件：
     - `===== 新消息事件开始 =====`
     - `当前会话ID: xxx`
     - `消息会话ID: xxx`
     - `是否是当前会话: true/false`

4. **检查消息 store**
   - 打开 Vue DevTools
   - 查看 `message` store
   - 确认消息是否已添加到 store 中

5. **检查组件响应**
   - 查看控制台日志：
     - `[ChatPanel] 收到消息更新事件`
     - `[ChatPanel] 是当前会话，准备滚动到底部`

**常见问题**：

| 问题 | 原因 | 解决方案 |
|-----|------|---------|
| 消息未显示 | 当前会话 ID 未设置 | 在 ChatPanel 中选择会话 |
| 消息未显示 | 消息会话 ID 与当前会话不一致 | 检查会话 ID 是否正确 |
| 消息未显示 | messages 是独立 ref，未从 store 获取 | 确保 ChatPanel 使用 computed 从 store 获取消息 |

### 问题：会话列表没有更新

**调试步骤**：

1. **查看事件日志**
   - 发送一条消息
   - 查看日志中是否有：
     - `调用会话更新回调`
     - `[ChatPanel] 收到会话更新事件`
     - `[HomeView] 会话更新事件`

2. **检查会话列表重新加载**
   - 查看日志：
     - `[ChatPanel] 准备重新加载会话列表`
     - `[useConversations] loadConversations 开始`
     - `[useConversations] 会话列表加载成功`

3. **检查会话排序**
   - 查看日志：
     - `[ConversationList] sortedConversations recomputed`
     - 检查会话是否按时间正确排序

**常见问题**：

| 问题 | 原因 | 解决方案 |
|-----|------|---------|
| 会话列表未更新 | conversationUpdate 回调未注册 | 确认在 onMounted 中注册了回调 |
| 会话列表未更新 | loadConversations 未调用 | 检查回调函数是否正确调用 loadConversations |
| 会话排序错误 | last_message 时间未更新 | 检查后端返回的 last_message 数据 |

### 问题：好友请求没有刷新

**调试步骤**：

1. **查看事件日志**
   - 发送好友请求
   - 查看日志中是否有：
     - `===== 新好友请求事件 =====`
     - `触发好友请求回调`
     - `[FriendsPanel] 收到好友请求更新事件`

2. **检查数据重新加载**
   - 查看日志：
     - `[FriendsPanel] 重新加载相关数据`
     - `[useFriends] loadFriends 开始`
     - `[useFriends] loadPendingRequests 开始`

**常见问题**：

| 问题 | 原因 | 解决方案 |
|-----|------|---------|
| 好友请求未刷新 | friendRequest 回调未注册 | 确认在 onMounted 中注册了回调 |
| 好友请求未刷新 | loadFriends/loadPendingRequests 未调用 | 检查回调函数是否正确调用 |

## 日志说明

### WebSocket 事件管理器日志

```
[WebSocketEventManager] ===== 新消息事件开始 =====
[WebSocketEventManager] 新消息事件数据: {...}
[WebSocketEventManager] 当前会话ID: xxx
[WebSocketEventManager] 消息会话ID: xxx
[WebSocketEventManager] 是否是当前会话: true/false
[WebSocketEventManager] 准备添加消息到messageStore: {...}
[WebSocketEventManager] 消息已添加到messageStore
[WebSocketEventManager] 是当前会话，触发消息更新回调
[WebSocketEventManager] 消息更新回调数量: 1
[WebSocketEventManager] 调用消息更新回调
[WebSocketEventManager] 会话更新回调数量: 2
[WebSocketEventManager] 调用会话更新回调
[WebSocketEventManager] ===== 新消息事件结束 =====
```

### 消息 Store 日志

```
[MessageStore] ===== 添加消息开始 =====
[MessageStore] 会话ID: xxx
[MessageStore] 消息ID: xxx
[MessageStore] 消息内容: xxx
[MessageStore] 发送者ID: xxx
[MessageStore] 创建时间: xxx
[MessageStore] 当前消息数量: 5
[MessageStore] 消息是否已存在: false
[MessageStore] 消息已添加，新消息数量: 6
[MessageStore] 所有消息ID: [...]
[MessageStore] 消息已缓存
[MessageStore] ===== 添加消息结束 =====
```

### ChatPanel 日志

```
[ChatPanel] ===== 收到消息更新事件 =====
[ChatPanel] 消息会话ID: xxx
[ChatPanel] 当前选中会话ID: xxx
[ChatPanel] 消息内容: xxx
[ChatPanel] 消息ID: xxx
[ChatPanel] 发送者ID: xxx
[ChatPanel] 是当前会话，准备滚动到底部
[ChatPanel] 调用scrollToBottom
[ChatPanel] ===== 消息更新事件处理完成 =====
```

### HomeView 日志

```
[HomeView] ===== 会话更新事件 =====
[HomeView] 会话ID: xxx
[HomeView] 会话名称: xxx
[HomeView] 最后消息: xxx
[HomeView] 准备重新加载会话列表
[HomeView] 会话列表重新加载完成
[HomeView] ===== 会话更新事件处理完成 =====
```

## 测试场景

### 场景 1：发送消息并查看自动更新

1. 打开 `/ws-debug` 页面
2. 连接 WebSocket
3. 设置当前会话 ID
4. 发送测试消息
5. 观察日志：
   - WebSocket 事件管理器是否收到新消息事件
   - 消息是否添加到 messageStore
   - ChatPanel 是否收到消息更新回调
   - HomeView 是否收到会话更新回调
   - 会话列表是否重新加载并排序

### 场景 2：跨会话消息更新

1. 打开两个浏览器窗口
2. 窗口 A：登录并打开聊天页面，选择会话 1
3. 窗口 B：登录并打开聊天页面，选择会话 2
4. 窗口 B 发送消息到会话 1
5. 窗口 A 观察：
   - 是否收到新消息通知
   - 消息是否自动显示在聊天窗口
   - 会话列表是否更新

### 场景 3：好友请求更新

1. 打开 `/ws-debug` 页面
2. 连接 WebSocket
3. 发送好友请求
4. 观察日志：
   - WebSocket 事件管理器是否收到新好友请求事件
   - FriendsPanel 是否收到好友请求更新回调
   - 好友列表和待处理请求是否刷新

## 常见问题排查

### 问题：WebSocket 连接失败

**检查项**：
1. 后端服务是否运行
2. WebSocket URL 是否正确
3. 用户 token 是否有效
4. 网络连接是否正常

**调试步骤**：
1. 打开浏览器开发者工具
2. 查看 Network 标签
3. 筛选 WS 请求
4. 查看 WebSocket 连接状态和错误信息

### 问题：事件回调未触发

**检查项**：
1. 回调是否在 onMounted 中注册
2. 回调是否在 onUnmounted 中清理
3. 回调函数是否正确定义
4. 事件管理器是否正确初始化

**调试步骤**：
1. 在回调函数中添加 console.log
2. 查看控制台是否有日志输出
3. 检查回调是否被正确注册

### 问题：消息重复显示

**检查项**：
1. 消息 ID 是否唯一
2. 消息是否已存在于 store 中
3. 是否多次添加了相同的消息

**调试步骤**：
1. 查看 MessageStore 日志
2. 检查 `消息是否已存在` 日志
3. 确认消息去重逻辑是否正常工作

## 性能优化建议

1. **减少日志输出**：生产环境中禁用详细日志
2. **使用节流**：对频繁触发的事件使用节流
3. **优化更新频率**：避免不必要的会话列表重新加载
4. **使用虚拟滚动**：对大量消息使用虚拟滚动

## 总结

使用本调试指南，您可以：

1. 使用 `/ws-debug` 页面进行实时调试
2. 通过浏览器开发者工具查看网络请求和日志
3. 使用 Vue DevTools 查看 store 状态
4. 按照调试流程逐步排查问题
5. 参考日志说明理解事件流程
6. 使用测试场景验证功能

如果问题仍然存在，请提供详细的日志信息以便进一步排查。
