import { ref } from 'vue';

export interface ConversationState {
  conversationId: string;
  isHidden: boolean;
  unreadCount: number;
  lastUpdated: number;
}

class ConversationStateCacheService {
  private cache = new Map<string, ConversationState>();
  private storageKeyPrefix = 'conversation_state_';

  constructor() {
    this.loadCacheFromStorage();
  }

  // 从localStorage加载缓存
  private loadCacheFromStorage() {
    try {
      const keys = Object.keys(localStorage);
      for (const key of keys) {
        if (key.startsWith(this.storageKeyPrefix)) {
          const conversationId = key.replace(this.storageKeyPrefix, '');
          const data = localStorage.getItem(key);
          if (data) {
            try {
              const state: ConversationState = JSON.parse(data);
              this.cache.set(conversationId, state);
            } catch (error) {
              console.error(
                '[ConversationStateCache] Failed to load state for conversation:',
                conversationId,
                error
              );
              localStorage.removeItem(key);
            }
          }
        }
      }
      console.log(`[ConversationStateCache] Loaded state for ${this.cache.size} conversations`);
    } catch (error) {
      console.error('[ConversationStateCache] Failed to load cache from storage:', error);
    }
  }

  // 保存缓存到localStorage
  private saveCacheToStorage(conversationId: string) {
    const state = this.cache.get(conversationId);
    if (!state) {
      return;
    }

    try {
      const data = JSON.stringify(state);
      localStorage.setItem(`${this.storageKeyPrefix}${conversationId}`, data);
    } catch (error) {
      console.error(
        '[ConversationStateCache] Failed to save state for conversation:',
        conversationId,
        error
      );
    }
  }

  // 获取会话状态
  getState(conversationId: string): ConversationState | null {
    return this.cache.get(conversationId) || null;
  }

  // 检查会话是否隐藏
  isHidden(conversationId: string): boolean {
    const state = this.cache.get(conversationId);
    return state ? state.isHidden : false;
  }

  // 隐藏会话
  hideConversation(conversationId: string) {
    let state = this.cache.get(conversationId);
    if (!state) {
      state = {
        conversationId,
        isHidden: true,
        unreadCount: 0,
        lastUpdated: Date.now(),
      };
    } else {
      state.isHidden = true;
      state.lastUpdated = Date.now();
    }
    this.cache.set(conversationId, state);
    this.saveCacheToStorage(conversationId);
    console.log(`[ConversationStateCache] Hidden conversation ${conversationId}`);
  }

  // 显示会话
  showConversation(conversationId: string) {
    let state = this.cache.get(conversationId);
    if (!state) {
      state = {
        conversationId,
        isHidden: false,
        unreadCount: 0,
        lastUpdated: Date.now(),
      };
    } else {
      state.isHidden = false;
      state.lastUpdated = Date.now();
    }
    this.cache.set(conversationId, state);
    this.saveCacheToStorage(conversationId);
    console.log(`[ConversationStateCache] Shown conversation ${conversationId}`);
  }

  // 获取未读消息数量
  getUnreadCount(conversationId: string): number {
    const state = this.cache.get(conversationId);
    return state ? state.unreadCount : 0;
  }

  // 增加未读消息数量
  incrementUnreadCount(conversationId: string, count: number = 1) {
    let state = this.cache.get(conversationId);
    if (!state) {
      state = {
        conversationId,
        isHidden: false,
        unreadCount: count,
        lastUpdated: Date.now(),
      };
    } else {
      state.unreadCount += count;
      state.lastUpdated = Date.now();
    }
    this.cache.set(conversationId, state);
    this.saveCacheToStorage(conversationId);
    console.log(
      `[ConversationStateCache] Incremented unread count for conversation ${conversationId} to ${state.unreadCount}`
    );
  }

  // 清除未读消息数量
  clearUnreadCount(conversationId: string) {
    let state = this.cache.get(conversationId);
    if (!state) {
      state = {
        conversationId,
        isHidden: false,
        unreadCount: 0,
        lastUpdated: Date.now(),
      };
    } else {
      state.unreadCount = 0;
      state.lastUpdated = Date.now();
    }
    this.cache.set(conversationId, state);
    this.saveCacheToStorage(conversationId);
    console.log(`[ConversationStateCache] Cleared unread count for conversation ${conversationId}`);
  }

  // 清除会话状态
  clearConversationState(conversationId: string) {
    this.cache.delete(conversationId);
    localStorage.removeItem(`${this.storageKeyPrefix}${conversationId}`);
    console.log(`[ConversationStateCache] Cleared state for conversation ${conversationId}`);
  }

  // 清除所有状态
  clearAll() {
    this.cache.clear();
    const keys = Object.keys(localStorage);
    keys.forEach((key) => {
      if (key.startsWith(this.storageKeyPrefix)) {
        localStorage.removeItem(key);
      }
    });
    console.log('[ConversationStateCache] Cleared all states');
  }

  // 获取所有未隐藏的会话ID
  getVisibleConversationIds(): string[] {
    const visibleIds: string[] = [];
    this.cache.forEach((state, conversationId) => {
      if (!state.isHidden) {
        visibleIds.push(conversationId);
      }
    });
    return visibleIds;
  }

  // 获取缓存统计信息
  getStats() {
    let totalConversations = 0;
    let hiddenConversations = 0;
    let totalUnread = 0;
    const conversations: string[] = [];

    this.cache.forEach((state, conversationId) => {
      totalConversations++;
      if (state.isHidden) {
        hiddenConversations++;
      }
      totalUnread += state.unreadCount;
      conversations.push(conversationId);
    });

    return {
      totalConversations,
      hiddenConversations,
      visibleConversations: totalConversations - hiddenConversations,
      totalUnread,
      conversations,
    };
  }
}

// 创建全局会话状态缓存服务实例
export const conversationStateCacheService = new ConversationStateCacheService();

// Vue composable
export function useConversationStateCache() {
  const stats = ref(conversationStateCacheService.getStats());

  const refreshStats = () => {
    stats.value = conversationStateCacheService.getStats();
  };

  return {
    stats,
    refreshStats,
    getState: conversationStateCacheService.getState.bind(conversationStateCacheService),
    isHidden: conversationStateCacheService.isHidden.bind(conversationStateCacheService),
    hideConversation: conversationStateCacheService.hideConversation.bind(
      conversationStateCacheService
    ),
    showConversation: conversationStateCacheService.showConversation.bind(
      conversationStateCacheService
    ),
    getUnreadCount: conversationStateCacheService.getUnreadCount.bind(
      conversationStateCacheService
    ),
    incrementUnreadCount: conversationStateCacheService.incrementUnreadCount.bind(
      conversationStateCacheService
    ),
    clearUnreadCount: conversationStateCacheService.clearUnreadCount.bind(
      conversationStateCacheService
    ),
    clearConversationState: conversationStateCacheService.clearConversationState.bind(
      conversationStateCacheService
    ),
    clearAll: conversationStateCacheService.clearAll.bind(conversationStateCacheService),
    getVisibleConversationIds: conversationStateCacheService.getVisibleConversationIds.bind(
      conversationStateCacheService
    ),
  };
}
