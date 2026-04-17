import { describe, it, expect, beforeEach, vi } from 'vitest';
import { useConversations } from '../composables/useConversations';
import type { Conversation } from '../models/types';

// Mock api module
vi.mock('../models/api', () => ({
  api: {
    getConversations: vi.fn(),
    createConversation: vi.fn(),
    deleteConversation: vi.fn(),
  },
}));

describe('useConversations', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const mockConversation: Conversation = {
    id: 'conv-1',
    conversation_type: 'direct',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  };

  describe('loadConversations', () => {
    it('should load conversations and set ref', async () => {
      const { api } = await import('../models/api');
      vi.mocked(api.getConversations).mockResolvedValueOnce({
        success: true,
        data: [mockConversation],
      });

      const { conversations, loadConversations } = useConversations();
      await loadConversations();

      expect(conversations.value).toHaveLength(1);
      expect(conversations.value[0].id).toBe('conv-1');
    });

    it('should handle API failure gracefully', async () => {
      const { api } = await import('../models/api');
      vi.mocked(api.getConversations).mockResolvedValueOnce({
        success: false,
        message: 'Failed to load',
      });

      const { conversations, loadConversations } = useConversations();
      await loadConversations();

      expect(conversations.value).toHaveLength(0);
    });

    it('should handle network error', async () => {
      const { api } = await import('../models/api');
      vi.mocked(api.getConversations).mockRejectedValueOnce(new Error('Network error'));

      const { conversations, loadConversations } = useConversations();
      await loadConversations();

      expect(conversations.value).toHaveLength(0);
    });
  });

  describe('createConversation', () => {
    it('should create conversation and return created object', async () => {
      const { api } = await import('../models/api');
      vi.mocked(api.createConversation).mockResolvedValueOnce({
        success: true,
        data: mockConversation,
      });
      vi.mocked(api.getConversations).mockResolvedValueOnce({
        success: true,
        data: [mockConversation],
      });

      const { createConversation } = useConversations();
      const result = await createConversation('user-123');

      expect(result).not.toBeNull();
      expect(result!.id).toBe('conv-1');
    });

    it('should return null on API failure', async () => {
      const { api } = await import('../models/api');
      vi.mocked(api.createConversation).mockResolvedValueOnce({
        success: false,
        message: 'Failed to create',
      });

      const { createConversation } = useConversations();
      const result = await createConversation('user-123');

      expect(result).toBeNull();
    });

    it('should handle network error', async () => {
      const { api } = await import('../models/api');
      vi.mocked(api.createConversation).mockRejectedValueOnce(new Error('Network error'));

      const { createConversation } = useConversations();
      const result = await createConversation('user-123');

      expect(result).toBeNull();
    });
  });

  describe('deleteConversation', () => {
    it('should delete conversation and return true', async () => {
      const { api } = await import('../models/api');
      vi.mocked(api.deleteConversation).mockResolvedValueOnce({
        success: true,
      });
      vi.mocked(api.getConversations).mockResolvedValueOnce({
        success: true,
        data: [],
      });

      const { deleteConversation } = useConversations();
      const result = await deleteConversation('conv-1');

      expect(result).toBe(true);
    });

    it('should return false on API failure', async () => {
      const { api } = await import('../models/api');
      vi.mocked(api.deleteConversation).mockResolvedValueOnce({
        success: false,
        message: 'Failed to delete',
      });

      const { deleteConversation } = useConversations();
      const result = await deleteConversation('conv-1');

      expect(result).toBe(false);
    });

    it('should handle network error', async () => {
      const { api } = await import('../models/api');
      vi.mocked(api.deleteConversation).mockRejectedValueOnce(new Error('Network error'));

      const { deleteConversation } = useConversations();
      const result = await deleteConversation('conv-1');

      expect(result).toBe(false);
    });
  });
});
