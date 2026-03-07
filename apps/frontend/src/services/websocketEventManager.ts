import { websocketService } from './websocket';
import { useMessageStore } from '../stores/message';
import { useAuthStore } from '../stores/auth';
import type { Message, Conversation, Friendship, User } from '../models/types';

// WebSocket事件数据类型定义
export interface NewMessageEventData {
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

export interface NewFriendRequestEventData {
  conversation_id: string;
  sender_id: string;
  status: 'pending';
  sender?: User;
}

export interface FriendRequestUpdateEventData {
  conversation_id: string;
  status: 'accepted' | 'rejected';
  action: 'accept' | 'reject';
  user_id?: string;
}

export interface NewGroupConversationEventData {
  conversation_id: string;
  name: string;
  created_by: string;
  member_count: number;
  conversation?: Conversation;
}

export interface ConversationMemberAddedEventData {
  conversation_id: string;
  user_id: string;
  role: 'owner' | 'admin' | 'member';
  added_by: string;
  user?: User;
  conversation?: Conversation;
}

export interface ConversationMemberRemovedEventData {
  conversation_id: string;
  user_id: string;
  removed_by: string;
  conversation?: Conversation;
}

export interface UserOnlineStatusEventData {
  user_id: string;
  online: boolean;
  last_seen?: string;
}

// 回调函数类型定义
export type ConversationUpdateCallback = (conversation: Conversation) => void;
export type MessageUpdateCallback = (conversationId: string, message: Message) => void;
export type FriendRequestCallback = (request: Friendship) => void;
export type OnlineStatusCallback = (userId: string, online: boolean) => void;

/**
 * WebSocket事件管理器
 * 负责处理所有WebSocket事件并自动更新数据结构和视图
 */
class WebSocketEventManager {
  // 回调函数集合
  private conversationUpdateCallbacks: Set<ConversationUpdateCallback> = new Set();
  private messageUpdateCallbacks: Set<MessageUpdateCallback> = new Set();
  private friendRequestCallbacks: Set<FriendRequestCallback> = new Set();
  private onlineStatusCallbacks: Set<OnlineStatusCallback> = new Set();

  // 当前选中的会话ID
  private currentConversationId: string | null = null;

  // 用户在线状态缓存
  private onlineStatusCache: Map<string, boolean> = new Map();

  constructor() {
    this.setupEventHandlers();
  }

  /**
   * 设置WebSocket事件处理器
   */
  private setupEventHandlers() {
    // 新消息事件
    websocketService.on('new_message', this.handleNewMessage.bind(this));

    // 新好友请求事件
    websocketService.on('new_friend_request', this.handleNewFriendRequest.bind(this));

    // 好友请求更新事件
    websocketService.on('friend_request_update', this.handleFriendRequestUpdate.bind(this));

    // 新群聊创建事件
    websocketService.on('new_group_conversation', this.handleNewGroupConversation.bind(this));

    // 会话成员添加事件
    websocketService.on('conversation_member_added', this.handleConversationMemberAdded.bind(this));

    // 会话成员移除事件
    websocketService.on('conversation_member_removed', this.handleConversationMemberRemoved.bind(this));

    // 用户在线状态事件
    websocketService.on('user_online_status', this.handleUserOnlineStatus.bind(this));
  }

  /**
   * 设置当前选中的会话ID
   */
  setCurrentConversation(conversationId: string | null) {
    this.currentConversationId = conversationId;
  }

  /**
   * 获取用户在线状态
   */
  getUserOnlineStatus(userId: string): boolean {
    return this.onlineStatusCache.get(userId) ?? false;
  }

  /**
   * 注册会话更新回调
   */
  onConversationUpdate(callback: ConversationUpdateCallback) {
    this.conversationUpdateCallbacks.add(callback);
  }

  /**
   * 移除会话更新回调
   */
  offConversationUpdate(callback: ConversationUpdateCallback) {
    this.conversationUpdateCallbacks.delete(callback);
  }

  /**
   * 注册消息更新回调
   */
  onMessageUpdate(callback: MessageUpdateCallback) {
    this.messageUpdateCallbacks.add(callback);
  }

  /**
   * 移除消息更新回调
   */
  offMessageUpdate(callback: MessageUpdateCallback) {
    this.messageUpdateCallbacks.delete(callback);
  }

  /**
   * 注册好友请求回调
   */
  onFriendRequest(callback: FriendRequestCallback) {
    this.friendRequestCallbacks.add(callback);
  }

  /**
   * 移除好友请求回调
   */
  offFriendRequest(callback: FriendRequestCallback) {
    this.friendRequestCallbacks.delete(callback);
  }

  /**
   * 注册在线状态回调
   */
  onOnlineStatus(callback: OnlineStatusCallback) {
    this.onlineStatusCallbacks.add(callback);
  }

  /**
   * 移除在线状态回调
   */
  offOnlineStatus(callback: OnlineStatusCallback) {
    this.onlineStatusCallbacks.delete(callback);
  }

  /**
   * 处理新消息事件
   */
  private handleNewMessage(data: NewMessageEventData) {
    console.log('[WebSocketEventManager] ===== 新消息事件开始 =====');
    console.log('[WebSocketEventManager] 新消息事件数据:', JSON.stringify(data, null, 2));
    console.log('[WebSocketEventManager] 当前会话ID:', this.currentConversationId);
    console.log('[WebSocketEventManager] 消息会话ID:', data.conversation_id);
    console.log('[WebSocketEventManager] 是否是当前会话:', this.currentConversationId === data.conversation_id);

    // 更新消息store
    const messageStore = useMessageStore();
    const message: Message = {
      id: data.id,
      conversation_id: data.conversation_id,
      sender_id: data.sender_id,
      content: data.content,
      msg_type: data.msg_type as 'text' | 'image',
      created_at: data.created_at,
      sender: data.sender
        ? {
            id: data.sender.id,
            uid: 0,
            username: data.sender.username,
            avatar_url: data.sender.avatar_url || '',
            email_verified: false,
            phone_verified: false,
            created_at: '',
          }
        : undefined,
    };

    console.log('[WebSocketEventManager] 准备添加消息到messageStore:', message);
    // 添加消息到store
    messageStore.addMessage(data.conversation_id, message);
    console.log('[WebSocketEventManager] 消息已添加到messageStore');

    // 如果是当前会话，触发消息更新回调
    if (this.currentConversationId === data.conversation_id) {
      console.log('[WebSocketEventManager] 是当前会话，触发消息更新回调');
      console.log('[WebSocketEventManager] 消息更新回调数量:', this.messageUpdateCallbacks.size);
      this.messageUpdateCallbacks.forEach((callback) => {
        console.log('[WebSocketEventManager] 调用消息更新回调');
        callback(data.conversation_id, message);
      });
    } else {
      console.log('[WebSocketEventManager] 不是当前会话，不触发消息更新回调');
    }

    // 通知会话列表需要更新（触发重新加载）
    console.log('[WebSocketEventManager] 会话更新回调数量:', this.conversationUpdateCallbacks.size);
    this.conversationUpdateCallbacks.forEach((callback) => {
      // 创建一个临时的会话对象用于通知
      const tempConversation: Conversation = {
        id: data.conversation_id,
        conversation_type: 'direct', // 临时值，实际应该从API获取
        created_at: data.created_at,
        updated_at: data.created_at,
        last_message: message,
      };
      console.log('[WebSocketEventManager] 调用会话更新回调');
      callback(tempConversation);
    });
    console.log('[WebSocketEventManager] ===== 新消息事件结束 =====');
  }

  /**
   * 处理新好友请求事件
   */
  private handleNewFriendRequest(data: NewFriendRequestEventData) {
    console.log('[WebSocketEventManager] 新好友请求事件:', data);

    // 通知好友请求更新
    if (data.sender) {
      const tempFriendship: Friendship = {
        id: data.conversation_id,
        user_id: data.sender_id,
        friend_id: data.sender_id,
        conversation_id: data.conversation_id,
        status: 'pending',
        created_at: new Date().toISOString(),
        user: data.sender,
      };

      this.friendRequestCallbacks.forEach((callback) => {
        callback(tempFriendship);
      });
    }

    // 通知会话列表需要更新
    this.conversationUpdateCallbacks.forEach((callback) => {
      const tempConversation: Conversation = {
        id: data.conversation_id,
        conversation_type: 'direct',
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        friendship_status: 'pending',
      };
      callback(tempConversation);
    });
  }

  /**
   * 处理好友请求更新事件
   */
  private handleFriendRequestUpdate(data: FriendRequestUpdateEventData) {
    console.log('[WebSocketEventManager] 好友请求更新事件:', data);

    // 通知好友请求更新
    const tempFriendship: Friendship = {
      id: data.conversation_id,
      user_id: data.user_id || '',
      friend_id: data.user_id || '',
      conversation_id: data.conversation_id,
      status: data.status,
      created_at: new Date().toISOString(),
    };

    this.friendRequestCallbacks.forEach((callback) => {
      callback(tempFriendship);
    });

    // 通知会话列表需要更新
    this.conversationUpdateCallbacks.forEach((callback) => {
      const tempConversation: Conversation = {
        id: data.conversation_id,
        conversation_type: 'direct',
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        friendship_status: data.status,
      };
      callback(tempConversation);
    });
  }

  /**
   * 处理新群聊创建事件
   */
  private handleNewGroupConversation(data: NewGroupConversationEventData) {
    console.log('[WebSocketEventManager] 新群聊创建事件:', data);

    // 如果有完整的会话数据，直接使用
    if (data.conversation) {
      this.conversationUpdateCallbacks.forEach((callback) => {
        callback(data.conversation!);
      });
    } else {
      // 否则创建临时会话对象
      const tempConversation: Conversation = {
        id: data.conversation_id,
        conversation_type: 'group',
        name: data.name,
        created_by: data.created_by,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      };
      this.conversationUpdateCallbacks.forEach((callback) => {
        callback(tempConversation);
      });
    }
  }

  /**
   * 处理会话成员添加事件
   */
  private handleConversationMemberAdded(data: ConversationMemberAddedEventData) {
    console.log('[WebSocketEventManager] 会话成员添加事件:', data);

    // 如果有完整的会话数据，直接使用
    if (data.conversation) {
      this.conversationUpdateCallbacks.forEach((callback) => {
        callback(data.conversation!);
      });
    } else {
      // 否则创建临时会话对象
      const tempConversation: Conversation = {
        id: data.conversation_id,
        conversation_type: 'group',
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      };
      this.conversationUpdateCallbacks.forEach((callback) => {
        callback(tempConversation);
      });
    }
  }

  /**
   * 处理会话成员移除事件
   */
  private handleConversationMemberRemoved(data: ConversationMemberRemovedEventData) {
    console.log('[WebSocketEventManager] 会话成员移除事件:', data);

    const authStore = useAuthStore();
    const currentUserId = authStore.user?.id;

    // 如果当前用户被移除，需要特殊处理
    if (data.user_id === currentUserId) {
      // 清除该会话的消息
      const messageStore = useMessageStore();
      messageStore.clearMessages(data.conversation_id);

      // 如果是当前会话，清除选中状态
      if (this.currentConversationId === data.conversation_id) {
        this.currentConversationId = null;
      }
    }

    // 通知会话列表需要更新
    if (data.conversation) {
      this.conversationUpdateCallbacks.forEach((callback) => {
        callback(data.conversation!);
      });
    } else {
      const tempConversation: Conversation = {
        id: data.conversation_id,
        conversation_type: 'group',
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      };
      this.conversationUpdateCallbacks.forEach((callback) => {
        callback(tempConversation);
      });
    }
  }

  /**
   * 处理用户在线状态事件
   */
  private handleUserOnlineStatus(data: UserOnlineStatusEventData) {
    console.log('[WebSocketEventManager] 用户在线状态事件:', data);

    // 更新在线状态缓存
    this.onlineStatusCache.set(data.user_id, data.online);

    // 通知所有注册的回调
    this.onlineStatusCallbacks.forEach((callback) => {
      callback(data.user_id, data.online);
    });
  }

  /**
   * 清理所有回调
   */
  destroy() {
    this.conversationUpdateCallbacks.clear();
    this.messageUpdateCallbacks.clear();
    this.friendRequestCallbacks.clear();
    this.onlineStatusCallbacks.clear();
    this.onlineStatusCache.clear();
    this.currentConversationId = null;
  }
}

// 导出全局实例
export const websocketEventManager = new WebSocketEventManager();

// 导出Vue composable
export function useWebSocketEventManager() {
  return {
    setCurrentConversation: (conversationId: string | null) => {
      websocketEventManager.setCurrentConversation(conversationId);
    },
    getUserOnlineStatus: (userId: string) => {
      return websocketEventManager.getUserOnlineStatus(userId);
    },
    onConversationUpdate: (callback: ConversationUpdateCallback) => {
      websocketEventManager.onConversationUpdate(callback);
    },
    offConversationUpdate: (callback: ConversationUpdateCallback) => {
      websocketEventManager.offConversationUpdate(callback);
    },
    onMessageUpdate: (callback: MessageUpdateCallback) => {
      websocketEventManager.onMessageUpdate(callback);
    },
    offMessageUpdate: (callback: MessageUpdateCallback) => {
      websocketEventManager.offMessageUpdate(callback);
    },
    onFriendRequest: (callback: FriendRequestCallback) => {
      websocketEventManager.onFriendRequest(callback);
    },
    offFriendRequest: (callback: FriendRequestCallback) => {
      websocketEventManager.offFriendRequest(callback);
    },
    onOnlineStatus: (callback: OnlineStatusCallback) => {
      websocketEventManager.onOnlineStatus(callback);
    },
    offOnlineStatus: (callback: OnlineStatusCallback) => {
      websocketEventManager.offOnlineStatus(callback);
    },
  };
}
