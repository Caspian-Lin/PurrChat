import { describe, it, expect, beforeEach, vi } from 'vitest';
import { conversationStateCacheService, useConversationStateCache } from '../services/conversationStateCache';

// Mock localStorage with Proxy (same pattern as messageCache.test.ts)
const localStorageMock = (() => {
  const store: Record<string, string> = {};

  const handler: ProxyHandler<typeof store> = {
    get(target, prop, receiver) {
      if (prop === 'getItem') return (key: string) => target[key] ?? null;
      if (prop === 'setItem')
        return (key: string, value: string) => {
          target[key] = String(value);
        };
      if (prop === 'removeItem')
        return (key: string) => {
          delete target[key];
        };
      if (prop === 'clear')
        return () => {
          for (const k of Object.keys(target)) delete target[k];
        };
      if (prop === 'length') return Object.keys(target).length;
      if (prop === 'key') return (index: number) => Object.keys(target)[index] ?? null;
      return Reflect.get(target, prop, receiver);
    },
    ownKeys(target) {
      return Reflect.ownKeys(target);
    },
    has(target, prop) {
      return Reflect.has(target, prop);
    },
    getOwnPropertyDescriptor(target, prop) {
      return Reflect.getOwnPropertyDescriptor(target, prop);
    },
  };

  return new Proxy(store, handler);
})();

Object.defineProperty(window, 'localStorage', {
  value: localStorageMock,
});

describe('ConversationStateCacheService', () => {
  beforeEach(() => {
    localStorage.clear();
    // Reset service state by re-initializing
    conversationStateCacheService.init('test-user-1');
  });

  describe('init', () => {
    it('should load cached states from localStorage for user', () => {
      // Pre-populate localStorage
      const prefix = 'conv_state_test-user-1_';
      const state1 = { conversationId: 'conv-1', isHidden: false, unreadCount: 3, lastUpdated: Date.now() };
      const state2 = { conversationId: 'conv-2', isHidden: true, unreadCount: 0, lastUpdated: Date.now() };
      localStorage.setItem(`${prefix}conv-1`, JSON.stringify(state1));
      localStorage.setItem(`${prefix}conv-2`, JSON.stringify(state2));

      conversationStateCacheService.init('test-user-1');

      expect(conversationStateCacheService.getState('conv-1')).toEqual(state1);
      expect(conversationStateCacheService.getState('conv-2')).toEqual(state2);
    });

    it('should clear in-memory cache before loading', () => {
      conversationStateCacheService.incrementUnreadCount('conv-1', 5);
      expect(conversationStateCacheService.getUnreadCount('conv-1')).toBe(5);

      // Re-init: clears cache then reloads from localStorage
      // Since incrementUnreadCount also persists to localStorage, the data will be reloaded
      conversationStateCacheService.init('test-user-1');
      expect(conversationStateCacheService.getUnreadCount('conv-1')).toBe(5);
    });

    it('should only load entries with user-specific prefix', () => {
      const prefix1 = 'conv_state_test-user-1_';
      const prefix2 = 'conv_state_test-user-2_';
      localStorage.setItem(`${prefix1}conv-a`, JSON.stringify({ conversationId: 'conv-a', isHidden: false, unreadCount: 1, lastUpdated: Date.now() }));
      localStorage.setItem(`${prefix2}conv-b`, JSON.stringify({ conversationId: 'conv-b', isHidden: false, unreadCount: 2, lastUpdated: Date.now() }));

      conversationStateCacheService.init('test-user-1');

      expect(conversationStateCacheService.getState('conv-a')).not.toBeNull();
      expect(conversationStateCacheService.getState('conv-b')).toBeNull();
    });

    it('should handle corrupted JSON gracefully (remove bad entries)', () => {
      const prefix = 'conv_state_test-user-1_';
      localStorage.setItem(`${prefix}conv-1`, '{invalid json');

      conversationStateCacheService.init('test-user-1');

      expect(conversationStateCacheService.getState('conv-1')).toBeNull();
      // Corrupted entry should be removed
      expect(localStorage.getItem(`${prefix}conv-1`)).toBeNull();
    });
  });

  describe('getState / isHidden', () => {
    it('should return null for unknown conversation', () => {
      expect(conversationStateCacheService.getState('unknown')).toBeNull();
    });

    it('should return false for isHidden of unknown conversation', () => {
      expect(conversationStateCacheService.isHidden('unknown')).toBe(false);
    });
  });

  describe('hideConversation / showConversation', () => {
    it('should set isHidden=true and persist', () => {
      conversationStateCacheService.hideConversation('conv-1');

      expect(conversationStateCacheService.isHidden('conv-1')).toBe(true);
      const prefix = 'conv_state_test-user-1_';
      const stored = JSON.parse(localStorage.getItem(`${prefix}conv-1`)!);
      expect(stored.isHidden).toBe(true);
    });

    it('should create new state if not exists', () => {
      conversationStateCacheService.hideConversation('conv-1');

      const state = conversationStateCacheService.getState('conv-1');
      expect(state).not.toBeNull();
      expect(state!.conversationId).toBe('conv-1');
      expect(state!.unreadCount).toBe(0);
    });

    it('should update lastUpdated timestamp', () => {
      conversationStateCacheService.hideConversation('conv-1');
      const first = conversationStateCacheService.getState('conv-1')!.lastUpdated;

      // Small delay
      const now = Date.now() + 100;
      vi.spyOn(Date, 'now').mockReturnValue(now);
      conversationStateCacheService.showConversation('conv-1');
      const second = conversationStateCacheService.getState('conv-1')!.lastUpdated;

      expect(second).toBeGreaterThan(first);
    });
  });

  describe('Unread count', () => {
    it('getUnreadCount should return 0 for unknown conversation', () => {
      expect(conversationStateCacheService.getUnreadCount('unknown')).toBe(0);
    });

    it('incrementUnreadCount should increment count (default +1)', () => {
      conversationStateCacheService.incrementUnreadCount('conv-1');
      expect(conversationStateCacheService.getUnreadCount('conv-1')).toBe(1);

      conversationStateCacheService.incrementUnreadCount('conv-1');
      expect(conversationStateCacheService.getUnreadCount('conv-1')).toBe(2);
    });

    it('incrementUnreadCount should accept custom count', () => {
      conversationStateCacheService.incrementUnreadCount('conv-1', 5);
      expect(conversationStateCacheService.getUnreadCount('conv-1')).toBe(5);
    });

    it('clearUnreadCount should reset to 0', () => {
      conversationStateCacheService.incrementUnreadCount('conv-1', 10);
      conversationStateCacheService.clearUnreadCount('conv-1');
      expect(conversationStateCacheService.getUnreadCount('conv-1')).toBe(0);
    });
  });

  describe('clearConversationState', () => {
    it('should remove from cache and localStorage', () => {
      conversationStateCacheService.incrementUnreadCount('conv-1', 3);
      expect(conversationStateCacheService.getState('conv-1')).not.toBeNull();

      conversationStateCacheService.clearConversationState('conv-1');

      expect(conversationStateCacheService.getState('conv-1')).toBeNull();
      const prefix = 'conv_state_test-user-1_';
      expect(localStorage.getItem(`${prefix}conv-1`)).toBeNull();
    });
  });

  describe('clearAll', () => {
    it('should clear all entries for current user', () => {
      conversationStateCacheService.incrementUnreadCount('conv-1', 1);
      conversationStateCacheService.incrementUnreadCount('conv-2', 2);

      conversationStateCacheService.clearAll();

      expect(conversationStateCacheService.getState('conv-1')).toBeNull();
      expect(conversationStateCacheService.getState('conv-2')).toBeNull();
    });

    it('should not remove entries for other users', () => {
      conversationStateCacheService.incrementUnreadCount('conv-1', 1);

      // Add data for another user directly
      localStorage.setItem('conv_state_other-user_conv-a', JSON.stringify({
        conversationId: 'conv-a',
        isHidden: false,
        unreadCount: 5,
        lastUpdated: Date.now(),
      }));

      conversationStateCacheService.clearAll();

      expect(localStorage.getItem('conv_state_other-user_conv-a')).not.toBeNull();
    });
  });

  describe('getVisibleConversationIds', () => {
    it('should return only non-hidden conversation IDs', () => {
      conversationStateCacheService.incrementUnreadCount('conv-1', 1);
      conversationStateCacheService.incrementUnreadCount('conv-2', 1);
      conversationStateCacheService.hideConversation('conv-2');
      conversationStateCacheService.incrementUnreadCount('conv-3', 1);

      const visibleIds = conversationStateCacheService.getVisibleConversationIds();
      expect(visibleIds).toContain('conv-1');
      expect(visibleIds).not.toContain('conv-2');
      expect(visibleIds).toContain('conv-3');
      expect(visibleIds).toHaveLength(2);
    });
  });

  describe('getStats', () => {
    it('should return correct total/hidden/visible/unread counts', () => {
      conversationStateCacheService.incrementUnreadCount('conv-1', 3);
      conversationStateCacheService.incrementUnreadCount('conv-2', 5);
      conversationStateCacheService.hideConversation('conv-2');
      conversationStateCacheService.incrementUnreadCount('conv-3', 1);

      const stats = conversationStateCacheService.getStats();
      expect(stats.totalConversations).toBe(3);
      expect(stats.hiddenConversations).toBe(1);
      expect(stats.visibleConversations).toBe(2);
      expect(stats.totalUnread).toBe(9);
      expect(stats.conversations).toHaveLength(3);
    });
  });

  describe('useConversationStateCache composable', () => {
    it('should expose all service methods', () => {
      const cache = useConversationStateCache();
      expect(typeof cache.init).toBe('function');
      expect(typeof cache.getState).toBe('function');
      expect(typeof cache.isHidden).toBe('function');
      expect(typeof cache.hideConversation).toBe('function');
      expect(typeof cache.showConversation).toBe('function');
      expect(typeof cache.getUnreadCount).toBe('function');
      expect(typeof cache.incrementUnreadCount).toBe('function');
      expect(typeof cache.clearUnreadCount).toBe('function');
      expect(typeof cache.clearConversationState).toBe('function');
      expect(typeof cache.clearAll).toBe('function');
      expect(typeof cache.getVisibleConversationIds).toBe('function');
    });

    it('refreshStats should update stats value', () => {
      const cache = useConversationStateCache();
      expect(cache.stats.value.totalConversations).toBe(0);

      conversationStateCacheService.incrementUnreadCount('conv-1', 1);
      cache.refreshStats();

      expect(cache.stats.value.totalConversations).toBe(1);
    });
  });
});
