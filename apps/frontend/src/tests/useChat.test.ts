import { describe, it, expect, beforeEach, vi } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';
import type { Message } from '../models/types';

const mocks = vi.hoisted(() => ({
  api: {
    getMessages: vi.fn(),
    getMessagesIncremental: vi.fn(),
    sendMessage: vi.fn(),
    exportMessages: vi.fn(),
  },
  messageCache: {
    addMessage: vi.fn(),
    addMessages: vi.fn(),
    getMessages: vi.fn(),
    hasCache: vi.fn(),
    getLastUpdated: vi.fn(),
    init: vi.fn(),
    clearAll: vi.fn(),
  },
  notify: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

vi.mock('../models/api', () => ({
  api: mocks.api,
}));

vi.mock('../services/messageCache', () => ({
  useMessageCache: () => mocks.messageCache,
}));

vi.mock('../composables/useNotification', () => ({
  useNotification: () => mocks.notify,
}));

import { useChat } from '../composables/useChat';
import { useMessageStore } from '../stores/message';

describe('useChat', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();

    mocks.messageCache.getMessages.mockReturnValue([]);
    mocks.messageCache.hasCache.mockReturnValue(false);
    mocks.messageCache.getLastUpdated.mockReturnValue(0);
    mocks.messageCache.addMessage.mockResolvedValue(undefined);
    mocks.messageCache.addMessages.mockResolvedValue(undefined);
    mocks.api.getMessages.mockResolvedValue({ success: true, data: [] });
    mocks.api.getMessagesIncremental.mockResolvedValue({ success: true, data: [] });
  });

  const createMessage = (id: string, createdAt: string): Message => ({
    id,
    conversation_id: 'conv-1',
    sender_id: 'user-1',
    content: `Message ${id}`,
    msg_type: 'text',
    created_at: createdAt,
  });

  it('restores cached messages before checking incremental updates', async () => {
    const olderMessage = createMessage('m1', '2026-07-07T10:00:00.000Z');
    const newerMessage = createMessage('m2', '2026-07-07T10:05:00.000Z');

    mocks.messageCache.hasCache.mockReturnValue(true);
    mocks.messageCache.getLastUpdated.mockReturnValue(Date.parse('2026-07-07T11:00:00.000Z'));
    mocks.messageCache.getMessages.mockReturnValue([newerMessage, olderMessage]);

    const { checkAndLoadIncremental } = useChat();
    const loadedCount = await checkAndLoadIncremental('conv-1');

    const store = useMessageStore();
    expect(loadedCount).toBe(0);
    expect(store.getMessages('conv-1').map((message) => message.id)).toEqual(['m1', 'm2']);
    expect(mocks.api.getMessages).not.toHaveBeenCalled();
    expect(mocks.messageCache.getLastUpdated).not.toHaveBeenCalled();
    expect(mocks.api.getMessagesIncremental).toHaveBeenCalledWith(
      'conv-1',
      Date.parse(newerMessage.created_at)
    );
  });

  it('keeps restored history when incremental messages are appended', async () => {
    const cachedMessage = createMessage('m1', '2026-07-07T10:00:00.000Z');
    const incrementalMessage = createMessage('m2', '2026-07-07T10:01:00.000Z');

    mocks.messageCache.hasCache.mockReturnValue(true);
    mocks.messageCache.getMessages.mockReturnValue([cachedMessage]);
    mocks.api.getMessagesIncremental.mockResolvedValue({
      success: true,
      data: [incrementalMessage],
    });

    const { checkAndLoadIncremental } = useChat();
    const loadedCount = await checkAndLoadIncremental('conv-1');

    const store = useMessageStore();
    expect(loadedCount).toBe(1);
    expect(store.getMessages('conv-1').map((message) => message.id)).toEqual(['m1', 'm2']);
  });

  it('refreshes cached bot messages that only have second-level timestamps', async () => {
    const humanTrigger = createMessage('human-trigger', '2026-07-07T10:00:00.250Z');
    const cachedBotReply: Message = {
      ...createMessage('bot-reply', '2026-07-07T10:00:00Z'),
      sender_id: 'bot-1',
      bot_id: 'bot-1',
      bot_name: 'Bot',
    };
    const serverBotReply: Message = {
      ...cachedBotReply,
      created_at: '2026-07-07T10:00:00.500Z',
    };

    mocks.messageCache.hasCache.mockReturnValue(true);
    mocks.messageCache.getMessages.mockReturnValue([cachedBotReply, humanTrigger]);
    mocks.api.getMessages.mockResolvedValue({
      success: true,
      data: [serverBotReply, humanTrigger],
    });

    const { checkAndLoadIncremental } = useChat();
    await checkAndLoadIncremental('conv-1');

    const store = useMessageStore();
    expect(mocks.api.getMessages).toHaveBeenCalledWith('conv-1');
    expect(store.getMessages('conv-1').map((message) => message.id)).toEqual([
      'human-trigger',
      'bot-reply',
    ]);
    expect(store.getMessages('conv-1')[1].created_at).toBe('2026-07-07T10:00:00.500Z');
    expect(mocks.api.getMessagesIncremental).toHaveBeenCalledWith(
      'conv-1',
      Date.parse('2026-07-07T10:00:00.500Z')
    );
  });
});
