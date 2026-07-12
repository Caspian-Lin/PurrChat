import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';
import { useBotStore } from '../stores/bot';
import type { BotCallLog } from '../models/types';

vi.mock('../models/api', () => ({
  api: {
    getBotCallLogs: vi.fn(),
  },
}));

function createCallLog(id: string): BotCallLog {
  return {
    id,
    bot_id: 'bot-1',
    conversation_id: 'conversation-1',
    sender_id: 'user-1',
    sender_name: 'User',
    trigger_message: 'hello',
    reply_content: 'hi',
    mechanism_id: 'mechanism-1',
    mechanism_name: 'Default',
    reply_type: 'text',
    execution_path: '',
    success: true,
    duration_ms: 10,
    created_at: '2026-07-13T00:00:00Z',
  };
}

describe('Bot Store call logs', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
  });

  it('normalizes a null log collection from an older backend', async () => {
    const { api } = await import('../models/api');
    vi.mocked(api.getBotCallLogs).mockResolvedValueOnce({
      success: true,
      data: { logs: null, total: 0, limit: 20, offset: 0 },
    });

    const store = useBotStore();
    await store.loadCallLogs('bot-1');

    expect(store.callLogs).toEqual([]);
    expect(store.callLogsHasMore).toBe(false);
    expect(store.callLogsLoading).toBe(false);
  });

  it('keeps existing logs when a next page is encoded as null', async () => {
    const { api } = await import('../models/api');
    vi.mocked(api.getBotCallLogs)
      .mockResolvedValueOnce({
        success: true,
        data: { logs: [createCallLog('log-1')], total: 2, limit: 20, offset: 0 },
      })
      .mockResolvedValueOnce({
        success: true,
        data: { logs: null, total: 2, limit: 20, offset: 20 },
      });

    const store = useBotStore();
    await store.loadCallLogs('bot-1');
    await store.loadMoreCallLogs('bot-1');

    expect(store.callLogs.map((log) => log.id)).toEqual(['log-1']);
    expect(store.callLogsLoading).toBe(false);
  });
});
