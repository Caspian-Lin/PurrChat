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

const CLOSE_CONNECTION_REPLACED = 4001;

export class WebSocketService {
  private ws: WebSocket | null = null;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 10;
  private baseReconnectDelay = 1000;
  private maxReconnectDelay = 30000;
  // eslint-disable-next-line no-unused-vars
  private messageHandlers = new Map<string, Array<(data: any) => void>>();
  private isManualClose = false;
  private shouldReconnect = true;

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
  connect() {
    if (
      this.ws &&
      (this.ws.readyState === WebSocket.CONNECTING || this.ws.readyState === WebSocket.OPEN)
    ) {
      logger.log('WebSocket connection already active');
      return;
    }

    this.clearReconnectTimer();
    this.connecting.value = true;
    this.connectionStore.setConnecting(true);
    this.isManualClose = false;
    this.shouldReconnect = true;

    const wsUrl = getWebSocketUrl();
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

      const socket = this.ws;
      socket.onopen = () => this.handleOpen(socket);
      socket.onmessage = (event) => this.handleMessage(socket, event);
      socket.onerror = (event) => this.handleError(socket, event);
      socket.onclose = (event) => this.handleClose(socket, event);
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
    this.shouldReconnect = false;
    this.clearReconnectTimer();
    if (this.ws) {
      this.ws.close(1000, 'client disconnect');
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
  private handleOpen(socket: WebSocket) {
    if (socket !== this.ws) {
      socket.close(1000, 'superseded connection');
      return;
    }

    logger.log('WebSocket connected');
    this.clearReconnectTimer();
    this.connected.value = true;
    this.connecting.value = false;
    this.connectionStore.setConnected(true);
    this.connectionStore.setConnecting(false);
    this.reconnectAttempts = 0;

    // 发送ping保持连接
    this.send({ type: 'ping' });
  }

  // 处理接收到的消息
  private handleMessage(socket: WebSocket, event: MessageEvent) {
    if (socket !== this.ws) {
      return;
    }

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
  private handleError(socket: WebSocket, error: Event) {
    if (socket !== this.ws) {
      return;
    }

    logger.error('WebSocket error', error);
    this.connecting.value = false;
    this.connectionStore.setConnecting(false);
  }

  // 处理连接关闭
  private handleClose(socket: WebSocket, event: CloseEvent) {
    if (socket !== this.ws) {
      return;
    }

    this.ws = null;
    logger.log('WebSocket closed', { code: event.code, reason: event.reason });
    this.connected.value = false;
    this.connecting.value = false;
    this.connectionStore.setConnected(false);
    this.connectionStore.setConnecting(false);

    // 1008 = Policy Violation (auth failure) — don't reconnect
    if (event.code === 1008) {
      this.shouldReconnect = false;
      logger.error('WebSocket auth failure, not reconnecting');
      return;
    }

    // 4001 = 当前连接已被同一用户的新连接替代。重连会继续淘汰其他连接，形成循环。
    if (event.code === CLOSE_CONNECTION_REPLACED) {
      this.shouldReconnect = false;
      logger.warn('WebSocket connection replaced by a newer session, not reconnecting');
      return;
    }

    // 1000 = Normal closure — don't reconnect if manual close
    if (event.code === 1000 && this.isManualClose) {
      return;
    }

    if (this.shouldReconnect && !this.isManualClose) {
      this.scheduleReconnect();
    }
  }

  // 安排重连 — 指数退避 + jitter
  private scheduleReconnect() {
    if (this.reconnectTimer || !this.shouldReconnect || this.isManualClose) {
      return;
    }

    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      logger.error('Max reconnection attempts reached');
      return;
    }

    this.reconnectAttempts++;
    this.connectionStore.setReconnectAttempts(this.reconnectAttempts);

    // 指数退避: base * 2^(attempt-1)，上限 maxReconnectDelay
    const exponentialDelay = Math.min(
      this.baseReconnectDelay * Math.pow(2, this.reconnectAttempts - 1),
      this.maxReconnectDelay
    );
    // jitter: ±25% of the delay
    const jitter = exponentialDelay * 0.25 * (Math.random() * 2 - 1);
    const delay = Math.round(exponentialDelay + jitter);

    logger.log(
      `Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`
    );

    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      const auth = useAuthStore();
      if (auth.isAuthenticated && auth.user) {
        this.connect();
      }
    }, delay);
  }

  private clearReconnectTimer() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
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
      websocketService.connect();
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
