import { describe, it, expect, beforeEach, vi } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';
import type { Message } from '../models/types';

// Mock messageCache service — must be before store import
const mockAddMessage = vi.fn();
const mockAddMessages = vi.fn();
const mockGetMessages = vi.fn().mockReturnValue([]);
const mockHasCache = vi.fn().mockReturnValue(false);
const mockGetLastUpdated = vi.fn().mockReturnValue(0);

vi.mock('../services/messageCache', () => ({
  useMessageCache: () => ({
    addMessage: mockAddMessage,
    addMessages: mockAddMessages,
    getMessages: mockGetMessages,
    hasCache: mockHasCache,
    getLastUpdated: mockGetLastUpdated,
    init: vi.fn(),
    clearAll: vi.fn(),
  }),
}));

// Import store AFTER mock setup
import { useMessageStore } from '../stores/message';

describe('Message Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
    mockGetMessages.mockReturnValue([]);
    mockHasCache.mockReturnValue(false);
  });

  const createMessage = (id: string, overrides?: Partial<Message>): Message => ({
    id,
    conversation_id: 'conv-1',
    sender_id: 'user-1',
    content: `Message ${id}`,
    msg_type: 'text',
    created_at: new Date().toISOString(),
    ...overrides,
  });

  describe('Initial State', () => {
    it('should have empty messages Map, loading Set, error Record', () => {
      const store = useMessageStore();
      expect(store.messages.size).toBe(0);
      expect(store.loading.size).toBe(0);
      expect(Object.keys(store.error)).toHaveLength(0);
    });
  });

  describe('setMessages', () => {
    it('should set messages for a conversation', () => {
      const store = useMessageStore();
      const msgs = [createMessage('m1'), createMessage('m2')];

      store.setMessages('conv-1', msgs);

      expect(store.messages.get('conv-1')).toHaveLength(2);
    });

    it('should overwrite existing messages for a conversation', () => {
      const store = useMessageStore();
      store.setMessages('conv-1', [createMessage('m1')]);
      store.setMessages('conv-1', [createMessage('m2'), createMessage('m3')]);

      expect(store.messages.get('conv-1')).toHaveLength(2);
    });
  });

  describe('addMessage', () => {
    it('should add message to conversation', () => {
      const store = useMessageStore();
      const msg = createMessage('m1');

      store.addMessage('conv-1', msg);

      expect(store.messages.get('conv-1')).toHaveLength(1);
      expect(mockAddMessage).toHaveBeenCalledWith('conv-1', msg);
    });

    it('should not add duplicate message with same id', () => {
      const store = useMessageStore();
      const msg = createMessage('m1');

      store.addMessage('conv-1', msg);
      store.addMessage('conv-1', msg);

      expect(store.messages.get('conv-1')).toHaveLength(1);
      expect(mockAddMessage).toHaveBeenCalledTimes(1);
    });
  });

  describe('addMessages', () => {
    it('should add multiple messages to conversation', () => {
      const store = useMessageStore();
      const msgs = [createMessage('m1'), createMessage('m2')];

      store.addMessages('conv-1', msgs);

      expect(store.messages.get('conv-1')).toHaveLength(2);
      expect(mockAddMessages).toHaveBeenCalledWith('conv-1', msgs);
    });

    it('should filter out duplicate messages', () => {
      const store = useMessageStore();
      const msg1 = createMessage('m1');
      const msg2 = createMessage('m2');

      store.addMessages('conv-1', [msg1, msg2]);
      store.addMessages('conv-1', [msg2, createMessage('m3')]);

      expect(store.messages.get('conv-1')).toHaveLength(3);
      expect(mockAddMessages).toHaveBeenCalledTimes(2);
    });
  });

  describe('clearMessages / clearAllMessages', () => {
    it('should clear messages for specific conversation', () => {
      const store = useMessageStore();
      store.addMessage('conv-1', createMessage('m1'));
      store.addMessage('conv-2', createMessage('m2'));

      store.clearMessages('conv-1');

      expect(store.messages.get('conv-1')).toBeUndefined();
      expect(store.messages.get('conv-2')).toHaveLength(1);
    });

    it('should clear all messages', () => {
      const store = useMessageStore();
      store.addMessage('conv-1', createMessage('m1'));
      store.addMessage('conv-2', createMessage('m2'));

      store.clearAllMessages();

      expect(store.messages.size).toBe(0);
    });
  });

  describe('Loading and error state', () => {
    it('setLoading should add/remove from loading Set', () => {
      const store = useMessageStore();

      store.setLoading('conv-1', true);
      expect(store.isLoading('conv-1')).toBe(true);

      store.setLoading('conv-1', false);
      expect(store.isLoading('conv-1')).toBe(false);
    });

    it('setError should set and clear error for conversation', () => {
      const store = useMessageStore();

      store.setError('conv-1', 'Network error');
      expect(store.getError('conv-1')).toBe('Network error');

      store.setError('conv-1', null);
      expect(store.getError('conv-1')).toBeNull();
    });
  });

  describe('totalMessageCount', () => {
    it('should return 0 when empty', () => {
      const store = useMessageStore();
      expect(store.totalMessageCount).toBe(0);
    });

    it('should sum messages across all conversations', () => {
      const store = useMessageStore();
      store.addMessage('conv-1', createMessage('m1'));
      store.addMessage('conv-1', createMessage('m2'));
      store.addMessage('conv-2', createMessage('m3'));

      expect(store.totalMessageCount).toBe(3);
    });
  });

  describe('updateLastMessage', () => {
    it('should add message if last message has different id', () => {
      const store = useMessageStore();
      store.addMessage('conv-1', createMessage('m1'));
      const newMsg = createMessage('m2');

      store.updateLastMessage('conv-1', newMsg);

      const msgs = store.messages.get('conv-1');
      expect(msgs).toHaveLength(2);
    });

    it('should not add if last message has same id', () => {
      const store = useMessageStore();
      const msg = createMessage('m1');
      store.addMessage('conv-1', msg);

      store.updateLastMessage('conv-1', msg);

      const msgs = store.messages.get('conv-1');
      expect(msgs).toHaveLength(1);
    });

    it('should do nothing when conversation has no messages', () => {
      const store = useMessageStore();

      store.updateLastMessage('conv-1', createMessage('m1'));

      expect(store.messages.get('conv-1')).toBeUndefined();
    });
  });

  describe('loadFromCache', () => {
    it('should load from cache and set messages', async () => {
      mockGetMessages.mockReturnValueOnce([createMessage('m1'), createMessage('m2')]);

      const store = useMessageStore();
      const loaded = await store.loadFromCache('conv-1');

      expect(loaded).toHaveLength(2);
      expect(store.messages.get('conv-1')).toHaveLength(2);
    });

    it('should return empty array when no cache', async () => {
      mockGetMessages.mockReturnValueOnce([]);

      const store = useMessageStore();
      const loaded = await store.loadFromCache('conv-1');

      expect(loaded).toEqual([]);
      expect(store.messages.get('conv-1')).toBeUndefined();
    });
  });

  describe('updateMessageStatus', () => {
    it('should update sendStatus of specific message', () => {
      const store = useMessageStore();
      store.addMessage('conv-1', createMessage('m1'));

      store.updateMessageStatus('conv-1', 'm1', 'sent');

      const msgs = store.messages.get('conv-1');
      expect(msgs![0].sendStatus).toBe('sent');
    });

    it('should handle non-existent conversation gracefully', () => {
      const store = useMessageStore();
      store.updateMessageStatus('conv-999', 'm1', 'sent');
      // Should not throw
    });

    it('should handle non-existent message gracefully', () => {
      const store = useMessageStore();
      store.addMessage('conv-1', createMessage('m1'));

      store.updateMessageStatus('conv-1', 'non-existent', 'sent');
      // Should not throw
    });
  });
});
