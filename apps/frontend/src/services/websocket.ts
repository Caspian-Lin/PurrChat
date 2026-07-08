import { ref, onUnmounted } from 'vue';
import { useAuthStore } from '../stores/auth';
import { useConnectionStore } from '../stores/connection';
import { getWebSocketUrl, logger } from '../config/app';
import { getCurrentPlatformCapabilities } from '../platform';

export interface WebSocketMessage {
  type: string;
  data: any;
  timestamp: number;
}

export interface NewMessageData {
  id: string;
  conversation_id: string;
  sender_id: string;
  content: string;
  msg_type: string;
  created_at: string;
  sender?: {
    id: string;
    username: string;
    avatar_url?: string;
  };
}

export interface FriendRequestUpdateData {
  conversation_id: string;
  status: 'accepted' | 'rejected';
  action: 'accept' | 'reject';
}

export interface NewFriendRequestData {
  conversation_id: string;
  sender_id: string;
  status: 'pending';
}

export interface NewGroupConversationData {
  conversation_id: string;
  name: string;
  created_by: string;
  member_count: number;
}

export interface ConversationMemberAddedData {
  conversation_id: string;
  user_id: string;
  role: 'owner' | 'admin' | 'member';
  added_by: string;
}

export interface ConversationMemberRemovedData {
  conversation_id: string;
  user_id: string;
  removed_by: string;
}

class WebSocketService {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 3000;
  // eslint-disable-next-line no-unused-vars
  private messageHandlers = new Map<string, Array<(data: any) => void>>();
  private isManualClose = false;

  // 连接状态
  public connected = ref(false);
  public connecting = ref(false);

  // 连接状态 store
  private connectionStore = useConnectionStore();

  constructor() {
    this.setupMessageHandlers();
  }

  // 初始化消息处理器
  private setupMessageHandlers() {
    // 默认的消息处理器
    this.on('new_message', this.handleNewMessage.bind(this));
    this.on('new_friend_request', this.handleNewFriendRequest.bind(this));
    this.on('friend_request_update', this.handleFriendRequestUpdate.bind(this));
    this.on('new_group_conversation', this.handleNewGroupConversation.bind(this));
    this.on('conversation_member_added', this.handleConversationMemberAdded.bind(this));
    this.on('conversation_member_removed', this.handleConversationMemberRemoved.bind(this));
    this.on('pong', this.handlePong.bind(this));
  }

  // 连接WebSocket（token 通过 Cookie 或子协议传递，不再通过 URL query）
  connect(userId: string) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      logger.log('WebSocket already connected');
      return;
    }

    this.connecting.value = true;
    this.connectionStore.setConnecting(true);
    this.isManualClose = false;

    const wsUrl = getWebSocketUrl(userId);
    logger.info('Connecting to WebSocket', { url: wsUrl });

    try {
      const platform = getCurrentPlatformCapabilities();
      if (platform.runtime.isNative) {
        // 原生环境: 使用 Sec-WebSocket-Protocol 子协议传递 token
        const auth = useAuthStore();
        if (!auth.token) {
          logger.error('No auth token available for native WebSocket');
          this.connecting.value = false;
          this.connectionStore.setConnecting(false);
          return;
        }
        this.ws = new WebSocket(wsUrl, [`bearer,${auth.token}`]);
      } else {
        // Web 环境: 依赖浏览器自动携带 Cookie
        this.ws = new WebSocket(wsUrl);
      }

      this.ws.onopen = this.handleOpen.bind(this);
      this.ws.onmessage = this.handleMessage.bind(this);
      this.ws.onerror = this.handleError.bind(this);
      this.ws.onclose = this.handleClose.bind(this);
    } catch (error) {
      logger.error('Failed to create WebSocket connection', error);
      this.connecting.value = false;
      this.connectionStore.setConnecting(false);
      this.scheduleReconnect();
    }
  }

  // 断开连接
  disconnect() {
    this.isManualClose = true;
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.connected.value = false;
    this.connecting.value = false;
    this.connectionStore.setConnected(false);
    this.connectionStore.setConnecting(false);
  }

  // 发送消息
  send(message: any) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      logger.warn('WebSocket not connected, cannot send message');
      return false;
    }

    try {
      this.ws.send(JSON.stringify(message));
      return true;
    } catch (error) {
      logger.error('Failed to send WebSocket message', error);
      return false;
    }
  }

  // 注册消息处理器
  // eslint-disable-next-line no-unused-vars
  on(type: string, handler: (data: any) => void) {
    if (!this.messageHandlers.has(type)) {
      this.messageHandlers.set(type, []);
    }
    this.messageHandlers.get(type)!.push(handler);
  }

  // 移除消息处理器
  // eslint-disable-next-line no-unused-vars
  off(type: string, handler: (data: any) => void) {
    const handlers = this.messageHandlers.get(type);
    if (handlers) {
      const index = handlers.indexOf(handler);
      if (index !== -1) {
        handlers.splice(index, 1);
      }
    }
  }

  // 处理连接打开
  private handleOpen() {
    logger.log('WebSocket connected');
    this.connected.value = true;
    this.connecting.value = false;
    this.connectionStore.setConnected(true);
    this.connectionStore.setConnecting(false);
    this.reconnectAttempts = 0;

    // 发送ping保持连接
    this.send({ type: 'ping' });
  }

  // 处理接收到的消息
  private handleMessage(event: MessageEvent) {
    try {
      const message: WebSocketMessage = JSON.parse(event.data);
      logger.log('WebSocket message received', message);

      // 调用注册的处理器
      const handlers = this.messageHandlers.get(message.type);
      if (handlers) {
        handlers.forEach((handler) => handler(message.data));
      }
    } catch (error) {
      logger.error('Failed to parse WebSocket message', error);
    }
  }

  // 处理连接错误
  private handleError(error: Event) {
    logger.error('WebSocket error', error);
    this.connecting.value = false;
    this.connectionStore.setConnecting(false);
  }

  // 处理连接关闭
  private handleClose(event: CloseEvent) {
    logger.log('WebSocket closed', { code: event.code, reason: event.reason });
    this.connected.value = false;
    this.connecting.value = false;
    this.connectionStore.setConnected(false);
    this.connectionStore.setConnecting(false);

    if (!this.isManualClose) {
      this.scheduleReconnect();
    }
  }

  // 安排重连
  private scheduleReconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      logger.error('Max reconnection attempts reached');
      return;
    }

    this.reconnectAttempts++;
    this.connectionStore.setReconnectAttempts(this.reconnectAttempts);
    const delay = this.reconnectDelay * this.reconnectAttempts;

    logger.log(
      `Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`
    );

    setTimeout(() => {
      const auth = useAuthStore();
      if (auth.isAuthenticated && auth.user) {
        this.connect(auth.user.id);
      }
    }, delay);
  }

  // 处理新消息
  private handleNewMessage(data: NewMessageData) {
    logger.log('New message received', data);
    // 这个处理器会在组件中被覆盖
  }

  // 处理新好友请求
  private handleNewFriendRequest(data: NewFriendRequestData) {
    logger.log('New friend request received', data);
    // 这个处理器会在组件中被覆盖
  }

  // 处理好友请求更新
  private handleFriendRequestUpdate(data: FriendRequestUpdateData) {
    logger.log('Friend request update received', data);
    // 这个处理器会在组件中被覆盖
  }

  // 处理pong响应
  private handlePong() {
    logger.log('Pong received');
    // 可以在这里更新最后活跃时间
  }

  // 处理新群聊
  private handleNewGroupConversation(data: NewGroupConversationData) {
    logger.log('New group conversation received', data);
    // 这个处理器会在组件中被覆盖
  }

  // 处理会话成员添加
  private handleConversationMemberAdded(data: ConversationMemberAddedData) {
    logger.log('Conversation member added', data);
    // 这个处理器会在组件中被覆盖
  }

  // 处理会话成员移除
  private handleConversationMemberRemoved(data: ConversationMemberRemovedData) {
    logger.log('Conversation member removed', data);
    // 这个处理器会在组件中被覆盖
  }

  // 发送ping
  ping() {
    this.send({ type: 'ping' });
  }

  // 发送typing状态
  sendTyping(conversationId: string) {
    this.send({
      type: 'typing',
      conversation_id: conversationId,
    });
  }
}

// 创建全局WebSocket服务实例
export const websocketService = new WebSocketService();

// Vue composable
export function useWebSocket() {
  const connect = () => {
    const auth = useAuthStore();
    if (auth.isAuthenticated && auth.user) {
      websocketService.connect(auth.user.id);
    }
  };

  const disconnect = () => {
    websocketService.disconnect();
  };

  // 组件卸载时自动断开连接
  onUnmounted(() => {
    disconnect();
  });

  return {
    connected: websocketService.connected,
    connecting: websocketService.connecting,
    connect,
    disconnect,
    send: websocketService.send.bind(websocketService),
    on: websocketService.on.bind(websocketService),
    off: websocketService.off.bind(websocketService),
    ping: websocketService.ping.bind(websocketService),
    sendTyping: websocketService.sendTyping.bind(websocketService),
  };
}
