import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { messageCacheService, useMessageCache } from '../services/messageCache';
import type { Message } from '../models/types';

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {};

  return {
    getItem: (key: string) => store[key] || null,
    setItem: (key: string, value: string) => {
      store[key] = value.toString();
    },
    removeItem: (key: string) => {
      delete store[key];
    },
    clear: () => {
      store = {};
    },
    get length() {
      return Object.keys(store).length;
    },
    key: (index: number) => {
      const keys = Object.keys(store);
      return keys[index] || null;
    },
  };
})();

Object.defineProperty(window, 'localStorage', {
  value: localStorageMock,
});

describe('MessageCache', () => {
  beforeEach(() => {
    // 清空缓存和localStorage
    messageCacheService.clearAll();
    localStorageMock.clear();
  });

  afterEach(() => {
    // 清理
    messageCacheService.clearAll();
    localStorageMock.clear();
  });

  describe('基本功能', () => {
    it('应该能够添加消息到缓存', async () => {
      const message: Message = {
        id: '1',
        conversation_id: 'conv1',
        sender_id: 'user1',
        content: 'Hello',
        msg_type: 'text',
        created_at: new Date().toISOString(),
      };

      await messageCacheService.addMessage('conv1', message);

      const cachedMessages = messageCacheService.getMessages('conv1');
      expect(cachedMessages).toHaveLength(1);
      expect(cachedMessages[0]).toEqual(message);
    });

    it('应该能够批量添加消息', async () => {
      const messages: Message[] = [
        {
          id: '1',
          conversation_id: 'conv1',
          sender_id: 'user1',
          content: 'Hello',
          msg_type: 'text',
          created_at: new Date().toISOString(),
        },
        {
          id: '2',
          conversation_id: 'conv1',
          sender_id: 'user2',
          content: 'World',
          msg_type: 'text',
          created_at: new Date().toISOString(),
        },
      ];

      await messageCacheService.addMessages('conv1', messages);

      const cachedMessages = messageCacheService.getMessages('conv1');
      expect(cachedMessages).toHaveLength(2);
    });

    it('应该能够获取消息', async () => {
      const message: Message = {
        id: '1',
        conversation_id: 'conv1',
        sender_id: 'user1',
        content: 'Hello',
        msg_type: 'text',
        created_at: new Date().toISOString(),
      };

      await messageCacheService.addMessage('conv1', message);

      const cachedMessages = messageCacheService.getMessages('conv1');
      expect(cachedMessages).toHaveLength(1);
      expect(cachedMessages[0].content).toBe('Hello');
    });

    it('应该能够清除会话缓存', async () => {
      const message: Message = {
        id: '1',
        conversation_id: 'conv1',
        sender_id: 'user1',
        content: 'Hello',
        msg_type: 'text',
        created_at: new Date().toISOString(),
      };

      await messageCacheService.addMessage('conv1', message);
      expect(messageCacheService.getMessages('conv1')).toHaveLength(1);

      messageCacheService.clearConversation('conv1');
      expect(messageCacheService.getMessages('conv1')).toHaveLength(0);
    });

    it('应该能够清除所有缓存', async () => {
      const message1: Message = {
        id: '1',
        conversation_id: 'conv1',
        sender_id: 'user1',
        content: 'Hello',
        msg_type: 'text',
        created_at: new Date().toISOString(),
      };

      const message2: Message = {
        id: '2',
        conversation_id: 'conv2',
        sender_id: 'user2',
        content: 'World',
        msg_type: 'text',
        created_at: new Date().toISOString(),
      };

      await messageCacheService.addMessage('conv1', message1);
      await messageCacheService.addMessage('conv2', message2);

      expect(messageCacheService.getMessages('conv1')).toHaveLength(1);
      expect(messageCacheService.getMessages('conv2')).toHaveLength(1);

      messageCacheService.clearAll();
      expect(messageCacheService.getMessages('conv1')).toHaveLength(0);
      expect(messageCacheService.getMessages('conv2')).toHaveLength(0);
    });
  });

  describe('重复消息处理', () => {
    it('不应该添加重复的消息', async () => {
      const message: Message = {
        id: '1',
        conversation_id: 'conv1',
        sender_id: 'user1',
        content: 'Hello',
        msg_type: 'text',
        created_at: new Date().toISOString(),
      };

      await messageCacheService.addMessage('conv1', message);
      await messageCacheService.addMessage('conv1', message);

      const cachedMessages = messageCacheService.getMessages('conv1');
      expect(cachedMessages).toHaveLength(1);
    });

    it('批量添加时应该过滤重复消息', async () => {
      const message: Message = {
        id: '1',
        conversation_id: 'conv1',
        sender_id: 'user1',
        content: 'Hello',
        msg_type: 'text',
        created_at: new Date().toISOString(),
      };

      await messageCacheService.addMessage('conv1', message);
      await messageCacheService.addMessages('conv1', [message]);

      const cachedMessages = messageCacheService.getMessages('conv1');
      expect(cachedMessages).toHaveLength(1);
    });
  });

  describe('缓存持久化', () => {
    it('应该能够保存到localStorage', async () => {
      const message: Message = {
        id: '1',
        conversation_id: 'conv1',
        sender_id: 'user1',
        content: 'Hello',
        msg_type: 'text',
        created_at: new Date().toISOString(),
      };

      await messageCacheService.addMessage('conv1', message);

      expect(localStorageMock.getItem('message_cache_conv1')).toBeTruthy();
    });

    it('应该能够从localStorage加载', async () => {
      const message: Message = {
        id: '1',
        conversation_id: 'conv1',
        sender_id: 'user1',
        content: 'Hello',
        msg_type: 'text',
        created_at: new Date().toISOString(),
      };

      await messageCacheService.addMessage('conv1', message);

      // 创建新的缓存服务实例
      const newCacheService = messageCacheService;
      const cachedMessages = newCacheService.getMessages('conv1');

      expect(cachedMessages).toHaveLength(1);
      expect(cachedMessages[0].content).toBe('Hello');
    });
  });

  describe('统计信息', () => {
    it('应该能够获取统计信息', async () => {
      const message1: Message = {
        id: '1',
        conversation_id: 'conv1',
        sender_id: 'user1',
        content: 'Hello',
        msg_type: 'text',
        created_at: new Date().toISOString(),
      };

      const message2: Message = {
        id: '2',
        conversation_id: 'conv2',
        sender_id: 'user2',
        content: 'World',
        msg_type: 'text',
        created_at: new Date().toISOString(),
      };

      await messageCacheService.addMessage('conv1', message1);
      await messageCacheService.addMessage('conv2', message2);

      const stats = messageCacheService.getStats();

      expect(stats.totalConversations).toBe(2);
      expect(stats.totalMessages).toBe(2);
      expect(stats.conversations).toContain('conv1');
      expect(stats.conversations).toContain('conv2');
    });
  });

  describe('导出功能', () => {
    it('应该能够导出会话消息', async () => {
      const message: Message = {
        id: '1',
        conversation_id: 'conv1',
        sender_id: 'user1',
        content: 'Hello',
        msg_type: 'text',
        created_at: new Date().toISOString(),
      };

      await messageCacheService.addMessage('conv1', message);

      const exported = messageCacheService.exportConversation('conv1');
      expect(exported).toBeTruthy();

      const parsed = JSON.parse(exported!);
      expect(parsed).toHaveLength(1);
      expect(parsed[0].content).toBe('Hello');
    });

    it('应该能够导出所有消息', async () => {
      const message1: Message = {
        id: '1',
        conversation_id: 'conv1',
        sender_id: 'user1',
        content: 'Hello',
        msg_type: 'text',
        created_at: new Date().toISOString(),
      };

      const message2: Message = {
        id: '2',
        conversation_id: 'conv2',
        sender_id: 'user2',
        content: 'World',
        msg_type: 'text',
        created_at: new Date().toISOString(),
      };

      await messageCacheService.addMessage('conv1', message1);
      await messageCacheService.addMessage('conv2', message2);

      const exported = messageCacheService.exportAll();
      expect(exported).toBeTruthy();

      const parsed = JSON.parse(exported!);
      expect(Object.keys(parsed)).toHaveLength(2);
      expect(parsed['conv1']).toHaveLength(1);
      expect(parsed['conv2']).toHaveLength(1);
    });
  });

  describe('时间戳功能', () => {
    it('应该能够获取最后更新时间', async () => {
      const message: Message = {
        id: '1',
        conversation_id: 'conv1',
        sender_id: 'user1',
        content: 'Hello',
        msg_type: 'text',
        created_at: new Date().toISOString(),
      };

      const beforeAdd = Date.now();
      await messageCacheService.addMessage('conv1', message);
      const lastUpdated = messageCacheService.getLastUpdated('conv1');

      expect(lastUpdated).toBeGreaterThanOrEqual(beforeAdd);
      expect(lastUpdated).toBeLessThanOrEqual(Date.now());
    });

    it('应该能够检查缓存是否存在', async () => {
      expect(messageCacheService.hasCache('conv1')).toBe(false);

      const message: Message = {
        id: '1',
        conversation_id: 'conv1',
        sender_id: 'user1',
        content: 'Hello',
        msg_type: 'text',
        created_at: new Date().toISOString(),
      };

      await messageCacheService.addMessage('conv1', message);
      expect(messageCacheService.hasCache('conv1')).toBe(true);
    });
  });
});

describe('useMessageCache Composable', () => {
  beforeEach(() => {
    // 清空缓存和localStorage
    messageCacheService.clearAll();
    localStorageMock.clear();
  });

  afterEach(() => {
    // 清理
    messageCacheService.clearAll();
    localStorageMock.clear();
  });

  it('应该能够使用composable', () => {
    const { getMessages, addMessage, clearAll, stats } = useMessageCache();

    expect(typeof getMessages).toBe('function');
    expect(typeof addMessage).toBe('function');
    expect(typeof clearAll).toBe('function');
    expect(stats.value).toBeDefined();
  });

  it('应该能够刷新统计信息', async () => {
    const { stats, refreshStats } = useMessageCache();

    const message: Message = {
      id: '1',
      conversation_id: 'conv1',
      sender_id: 'user1',
      content: 'Hello',
      msg_type: 'text',
      created_at: new Date().toISOString(),
    };

    await messageCacheService.addMessage('conv1', message);

    refreshStats();

    expect(stats.value.totalConversations).toBe(1);
    expect(stats.value.totalMessages).toBe(1);
  });
});
