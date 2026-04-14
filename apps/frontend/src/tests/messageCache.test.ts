import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { messageCacheService, useMessageCache } from '../services/messageCache';
import type { Message } from '../models/types';

// Mock localStorage - store data as direct properties so Object.keys works
const localStorageMock = (() => {
  const store: Record<string, string> = {};

  const handler: ProxyHandler<typeof store> = {
    get(target, prop, receiver) {
      if (prop === 'getItem') return (key: string) => target[key] ?? null;
      if (prop === 'setItem') return (key: string, value: string) => { target[key] = String(value); };
      if (prop === 'removeItem') return (key: string) => { delete target[key]; };
      if (prop === 'clear') return () => { for (const k of Object.keys(target)) delete target[k]; };
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

const TEST_USER_ID = 'test-user-123';

describe('MessageCache', () => {
  beforeEach(async () => {
    // 清空缓存和localStorage
    localStorageMock.clear();
    await messageCacheService.init(TEST_USER_ID);
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
    it('应该能够保存到localStorage（带用户前缀）', async () => {
      const message: Message = {
        id: '1',
        conversation_id: 'conv1',
        sender_id: 'user1',
        content: 'Hello',
        msg_type: 'text',
        created_at: new Date().toISOString(),
      };

      await messageCacheService.addMessage('conv1', message);

      expect(localStorageMock.getItem(`msg_${TEST_USER_ID}_conv1`)).toBeTruthy();
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

      // 重新初始化，模拟页面刷新
      await messageCacheService.init(TEST_USER_ID);
      const cachedMessages = messageCacheService.getMessages('conv1');

      expect(cachedMessages).toHaveLength(1);
      expect(cachedMessages[0].content).toBe('Hello');
    });
  });

  describe('用户隔离', () => {
    it('不同用户的数据应该隔离', async () => {
      const message: Message = {
        id: '1',
        conversation_id: 'conv1',
        sender_id: 'user1',
        content: 'Hello from user A',
        msg_type: 'text',
        created_at: new Date().toISOString(),
      };

      // 用户 A 添加消息
      await messageCacheService.addMessage('conv1', message);

      // 切换到用户 B
      await messageCacheService.init('user-B');
      expect(messageCacheService.getMessages('conv1')).toHaveLength(0);

      // 用户 B 添加自己的消息
      const messageB: Message = {
        id: '2',
        conversation_id: 'conv1',
        sender_id: 'user2',
        content: 'Hello from user B',
        msg_type: 'text',
        created_at: new Date().toISOString(),
      };
      await messageCacheService.addMessage('conv1', messageB);

      // 切回用户 A，数据应保留
      await messageCacheService.init(TEST_USER_ID);
      expect(messageCacheService.getMessages('conv1')).toHaveLength(1);
      expect(messageCacheService.getMessages('conv1')[0].content).toBe('Hello from user A');
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

  describe('服务器消息校准', () => {
    it('应该移除本地存在但服务器不存在的消息', async () => {
      const messages: Message[] = [
        {
          id: '1',
          conversation_id: 'conv1',
          sender_id: 'user1',
          content: '保留',
          msg_type: 'text',
          created_at: new Date().toISOString(),
        },
        {
          id: '2',
          conversation_id: 'conv1',
          sender_id: 'user1',
          content: '删除',
          msg_type: 'text',
          created_at: new Date().toISOString(),
        },
      ];

      await messageCacheService.addMessages('conv1', messages);
      expect(messageCacheService.getMessages('conv1')).toHaveLength(2);

      // 服务器只返回消息 1
      const serverIds = new Set(['1']);
      const removed = await messageCacheService.reconcileWithServer('conv1', serverIds);

      expect(removed).toBe(1);
      expect(messageCacheService.getMessages('conv1')).toHaveLength(1);
      expect(messageCacheService.getMessages('conv1')[0].id).toBe('1');
    });
  });
});

describe('useMessageCache Composable', () => {
  beforeEach(async () => {
    localStorageMock.clear();
    await messageCacheService.init(TEST_USER_ID);
  });

  afterEach(() => {
    messageCacheService.clearAll();
    localStorageMock.clear();
  });

  it('应该能够使用composable', () => {
    const { getMessages, addMessage, clearAll, stats, init, isInitialized } = useMessageCache();

    expect(typeof getMessages).toBe('function');
    expect(typeof addMessage).toBe('function');
    expect(typeof clearAll).toBe('function');
    expect(typeof init).toBe('function');
    expect(typeof isInitialized).toBe('function');
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
