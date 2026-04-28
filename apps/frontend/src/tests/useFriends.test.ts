import { describe, it, expect, beforeEach, vi } from 'vitest';
import { useFriends } from '../composables/useFriends';
import { api } from '../models/api';

// Mock the API
vi.mock('../models/api', () => ({
  api: {
    getFriends: vi.fn(),
    getPendingFriendRequests: vi.fn(),
    sendFriendRequest: vi.fn(),
    handleFriendRequest: vi.fn(),
  },
}));

// Mock the useNotification composable
const mockNotification = {
  success: vi.fn(),
  error: vi.fn(),
  warning: vi.fn(),
  info: vi.fn(),
};

vi.mock('../composables/useNotification', () => ({
  useNotification: vi.fn(() => mockNotification),
}));

describe('useFriends', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('loadFriends', () => {
    it('应该能够加载好友列表', async () => {
      const mockFriends = [
        {
          id: '1',
          user_id: 'user1',
          friend_id: 'user2',
          status: 'accepted' as const,
          created_at: '2024-01-01T00:00:00Z',
          friend: {
            id: 'user2',
            uid: 2,
            username: 'friend1',
            avatar_url: 'http://example.com/avatar1.png',
            email: 'friend1@example.com',
            email_verified: true,
            phone: '1234567890',
            phone_verified: true,
            created_at: '2024-01-01T00:00:00Z',
          },
        },
      ];

      vi.mocked(api.getFriends).mockResolvedValue({
        success: true,
        data: mockFriends,
      });

      const { friends, loadFriends } = useFriends();

      await loadFriends();

      expect(api.getFriends).toHaveBeenCalled();
      expect(friends.value).toEqual(mockFriends);
    });

    it('应该处理加载好友列表失败的情况', async () => {
      vi.mocked(api.getFriends).mockResolvedValue({
        success: false,
        message: 'Failed to load friends',
      });

      const { friends, loadFriends } = useFriends();

      await loadFriends();

      expect(api.getFriends).toHaveBeenCalled();
      expect(friends.value).toEqual([]);
    });
  });

  describe('loadPendingRequests', () => {
    it('应该能够加载待处理的好友请求', async () => {
      const mockRequests = [
        {
          id: '1',
          user_id: 'user2',
          friend_id: 'user1',
          status: 'pending' as const,
          created_at: '2024-01-01T00:00:00Z',
          user: {
            id: 'user2',
            uid: 2,
            username: 'sender1',
            avatar_url: 'http://example.com/avatar2.png',
            email: 'sender1@example.com',
            email_verified: true,
            phone: '1234567890',
            phone_verified: true,
            created_at: '2024-01-01T00:00:00Z',
          },
        },
      ];

      vi.mocked(api.getPendingFriendRequests).mockResolvedValue({
        success: true,
        data: mockRequests,
      });

      const { pendingRequests, loadPendingRequests } = useFriends();

      await loadPendingRequests();

      expect(api.getPendingFriendRequests).toHaveBeenCalled();
      expect(pendingRequests.value).toEqual(mockRequests);
    });

    it('应该处理加载待处理请求失败的情况', async () => {
      vi.mocked(api.getPendingFriendRequests).mockResolvedValue({
        success: false,
        message: 'Failed to load pending requests',
      });

      const { pendingRequests, loadPendingRequests } = useFriends();

      await loadPendingRequests();

      expect(api.getPendingFriendRequests).toHaveBeenCalled();
      expect(pendingRequests.value).toEqual([]);
    });
  });

  describe('sendFriendRequest', () => {
    it('应该能够发送好友请求', async () => {
      const mockConversation = {
        id: 'conv1',
        conversation_type: 'direct' as const,
        name: '',
        created_by: 'user1',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      };

      vi.mocked(api.sendFriendRequest).mockResolvedValue({
        success: true,
        data: mockConversation,
      });

      const { sendFriendRequest } = useFriends();

      const result = await sendFriendRequest('user2');

      expect(api.sendFriendRequest).toHaveBeenCalledWith({
        target_user_id: 'user2',
      });
      expect(result).toBe(true);
      expect(mockNotification.success).toHaveBeenCalledWith('好友请求已发送');
    });

    it('应该处理发送好友请求失败的情况', async () => {
      vi.mocked(api.sendFriendRequest).mockResolvedValue({
        success: false,
        message: 'Failed to send friend request',
      });

      const { sendFriendRequest } = useFriends();

      const result = await sendFriendRequest('user2');

      expect(api.sendFriendRequest).toHaveBeenCalledWith({
        target_user_id: 'user2',
      });
      expect(result).toBe(false);
      expect(mockNotification.error).toHaveBeenCalledWith('发送好友请求失败');
    });
  });

  describe('handleFriendRequest', () => {
    it('应该能够接受好友请求', async () => {
      const mockConversation = {
        id: 'conv1',
        conversation_type: 'direct' as const,
        name: '',
        created_by: 'user1',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      };

      vi.mocked(api.handleFriendRequest).mockResolvedValue({
        success: true,
        data: mockConversation,
      });

      vi.mocked(api.getPendingFriendRequests).mockResolvedValue({
        success: true,
        data: [],
      });

      const { handleFriendRequest } = useFriends();

      const result = await handleFriendRequest('conv1', 'accept');

      expect(api.handleFriendRequest).toHaveBeenCalledWith({
        conversation_id: 'conv1',
        action: 'accept',
      });
      expect(result).toBe(true);
      expect(mockNotification.success).toHaveBeenCalledWith('好友请求已接受');
    });

    it('应该能够拒绝好友请求', async () => {
      vi.mocked(api.handleFriendRequest).mockResolvedValue({
        success: true,
      });

      vi.mocked(api.getPendingFriendRequests).mockResolvedValue({
        success: true,
        data: [],
      });

      const { handleFriendRequest } = useFriends();

      const result = await handleFriendRequest('conv1', 'reject');

      expect(api.handleFriendRequest).toHaveBeenCalledWith({
        conversation_id: 'conv1',
        action: 'reject',
      });
      expect(result).toBe(true);
      expect(mockNotification.success).toHaveBeenCalledWith('好友请求已拒绝');
    });

    it('应该处理处理好友请求失败的情况', async () => {
      vi.mocked(api.handleFriendRequest).mockResolvedValue({
        success: false,
        message: 'Failed to handle friend request',
      });

      const { handleFriendRequest } = useFriends();

      const result = await handleFriendRequest('conv1', 'accept');

      expect(api.handleFriendRequest).toHaveBeenCalledWith({
        conversation_id: 'conv1',
        action: 'accept',
      });
      expect(result).toBe(false);
      expect(mockNotification.error).toHaveBeenCalledWith('处理好友请求失败');
    });
  });
});
