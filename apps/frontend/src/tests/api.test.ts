import { describe, it, expect, beforeEach, vi } from 'vitest';
import axios from 'axios';
import { api } from '../models/api';
import type { User, Conversation, Message, Friendship } from '../models/types';

// Mock axios
vi.mock('axios', () => {
  const mockAxios = {
    create: vi.fn(() => mockAxios),
    get: vi.fn(),
    post: vi.fn(),
    patch: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
    interceptors: {
      request: {
        use: vi.fn(),
      },
      response: {
        use: vi.fn(),
      },
    },
  };
  return {
    default: mockAxios,
  };
});

describe('API Client', () => {
  const mockedAxios = vi.mocked(axios);

  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    // Reset the mock to return itself
    mockedAxios.create.mockReturnValue(mockedAxios as any);
  });

  it('gets the Bot API capability catalog from the Registry endpoint', async () => {
    const catalog = { profile: {}, actions: [], events: [], segments: [] };
    mockedAxios.get.mockResolvedValueOnce({ data: catalog } as any);

    await expect(api.getBotApiCapabilities()).resolves.toEqual(catalog);
    expect(mockedAxios.get).toHaveBeenCalledWith('/api/bot/v1/capabilities');
  });

  it('creates and updates Bot installations with explicit capabilities', async () => {
    const installation = {
      id: 'installation-1',
      app_id: 'bot-1',
      installed_by: 'user-1',
      target_type: 'conversation' as const,
      target_id: 'conversation-1',
      granted_capabilities: ['messages:read_trigger', 'messages:send'],
      status: 'active' as const,
      installed_at: '2026-07-13T00:00:00Z',
      updated_at: '2026-07-13T00:00:00Z',
    };
    mockedAxios.post.mockResolvedValueOnce({
      data: { success: true, data: { installation } },
    } as any);
    mockedAxios.patch.mockResolvedValueOnce({
      data: { success: true, data: { installation } },
    } as any);

    const createRequest = {
      target_type: 'conversation' as const,
      target_id: 'conversation-1',
      granted_capabilities: ['messages:read_trigger', 'messages:send'],
      diagnostics_consent: 'denied' as const,
    };
    await api.createBotInstallation('bot-1', createRequest);
    await api.updateBotInstallation('installation-1', {
      granted_capabilities: createRequest.granted_capabilities,
    });

    expect(mockedAxios.post).toHaveBeenCalledWith('/api/bots/bot-1/installations', createRequest);
    expect(mockedAxios.patch).toHaveBeenCalledWith('/api/installations/installation-1', {
      granted_capabilities: createRequest.granted_capabilities,
    });
  });

  it('returns structured error code when updateBotInstallation gets HTTP error', async () => {
    mockedAxios.patch.mockRejectedValueOnce({
      response: {
        data: { success: false, code: 'granted_exceeds_requested', message: '超权' },
      },
    } as any);

    const result = await api.updateBotInstallation('inst-1', {
      granted_capabilities: ['secrets:use'],
    });

    expect(result.success).toBe(false);
    expect(result.code).toBe('granted_exceeds_requested');
    expect(result.message).toBe('超权');
  });

  it('returns structured error code when createBotInstallation gets HTTP error', async () => {
    mockedAxios.post.mockRejectedValueOnce({
      response: {
        data: { success: false, code: 'forbidden', message: '无权' },
      },
    } as any);

    const result = await api.createBotInstallation('bot-1', {
      target_type: 'user',
      target_id: 'user-1',
    });

    expect(result.success).toBe(false);
    expect(result.code).toBe('forbidden');
    expect(result.message).toBe('无权');
  });

  describe('register', () => {
    it('should register a new user', async () => {
      const mockUser: User = {
        id: '1',
        uid: 1,
        username: 'testuser',
        avatar_url: 'http://example.com/avatar.png',
        email: 'test@example.com',
        email_verified: true,
        phone: '1234567890',
        phone_verified: true,
        created_at: '2024-01-01T00:00:00Z',
      };

      const mockResponse = {
        data: {
          success: true,
          data: { token: 'test-token', user: mockUser },
        },
      };

      mockedAxios.create.mockReturnValueOnce(mockedAxios as any);
      mockedAxios.post.mockResolvedValueOnce(mockResponse as any);

      const result = await api.register({
        username: 'testuser',
        password: 'password123',
        email: 'test@example.com',
        phone: '1234567890',
      });

      expect(result.success).toBe(true);
      expect(result.data).toEqual({ token: 'test-token', user: mockUser });
    });
  });

  describe('login', () => {
    it('should login a user', async () => {
      const mockUser: User = {
        id: '1',
        uid: 1,
        username: 'testuser',
        avatar_url: 'http://example.com/avatar.png',
        email: 'test@example.com',
        email_verified: true,
        phone: '1234567890',
        phone_verified: true,
        created_at: '2024-01-01T00:00:00Z',
      };

      const mockResponse = {
        data: {
          success: true,
          data: { token: 'test-token', user: mockUser },
        },
      };

      mockedAxios.create.mockReturnValueOnce(mockedAxios as any);
      mockedAxios.post.mockResolvedValueOnce(mockResponse as any);

      const result = await api.login({
        email: 'test@example.com',
        password: 'password123',
      });

      expect(result.success).toBe(true);
      expect(result.data).toEqual({ token: 'test-token', user: mockUser });
    });
  });

  describe('me', () => {
    it('should get current user info', async () => {
      const mockUser: User = {
        id: '1',
        uid: 1,
        username: 'testuser',
        avatar_url: 'http://example.com/avatar.png',
        email: 'test@example.com',
        email_verified: true,
        phone: '1234567890',
        phone_verified: true,
        created_at: '2024-01-01T00:00:00Z',
      };

      const mockResponse = {
        data: {
          success: true,
          data: mockUser,
        },
      };

      mockedAxios.create.mockReturnValueOnce(mockedAxios as any);
      mockedAxios.get.mockResolvedValueOnce(mockResponse as any);

      const result = await api.me();

      expect(result.success).toBe(true);
      expect(result.data).toEqual(mockUser);
    });
  });

  describe('updateProfile', () => {
    it('should update user profile', async () => {
      const mockUser: User = {
        id: '1',
        uid: 1,
        username: 'testuser',
        avatar_url: 'http://example.com/avatar.png',
        email: 'newemail@example.com',
        email_verified: true,
        phone: '1234567890',
        phone_verified: true,
        created_at: '2024-01-01T00:00:00Z',
      };

      const mockResponse = {
        data: {
          success: true,
          data: mockUser,
        },
      };

      mockedAxios.create.mockReturnValueOnce(mockedAxios as any);
      mockedAxios.put.mockResolvedValueOnce(mockResponse as any);

      const result = await api.updateProfile({
        email: 'newemail@example.com',
      });

      expect(result.success).toBe(true);
      expect(result.data).toEqual(mockUser);
    });
  });

  describe('searchUsers', () => {
    it('should search users', async () => {
      const mockUsers: User[] = [
        {
          id: '1',
          uid: 1,
          username: 'testuser1',
          avatar_url: 'http://example.com/avatar1.png',
          email: 'test1@example.com',
          email_verified: true,
          phone: '1234567890',
          phone_verified: true,
          created_at: '2024-01-01T00:00:00Z',
        },
      ];

      const mockResponse = {
        data: {
          success: true,
          data: mockUsers,
        },
      };

      mockedAxios.create.mockReturnValueOnce(mockedAxios as any);
      mockedAxios.get.mockResolvedValueOnce(mockResponse as any);

      const result = await api.searchUsers('test');

      expect(result.success).toBe(true);
      expect(result.data).toEqual(mockUsers);
    });
  });

  describe('getConversations', () => {
    it('should get conversations list', async () => {
      const mockConversations: Conversation[] = [
        {
          id: '1',
          conversation_type: 'direct',
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
        },
      ];

      const mockResponse = {
        data: {
          success: true,
          data: mockConversations,
        },
      };

      mockedAxios.get.mockResolvedValueOnce(mockResponse as any);

      const result = await api.getConversations();

      expect(result.success).toBe(true);
      expect(result.data).toEqual(mockConversations);
    });
  });

  describe('createConversation', () => {
    it('should create a conversation', async () => {
      const mockConversation: Conversation = {
        id: '1',
        conversation_type: 'direct',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      };

      const mockResponse = {
        data: {
          success: true,
          data: mockConversation,
        },
      };

      mockedAxios.post.mockResolvedValueOnce(mockResponse as any);

      const result = await api.createConversation({
        target_user_id: '2',
      });

      expect(result.success).toBe(true);
      expect(result.data).toEqual(mockConversation);
    });
  });

  describe('getMessages', () => {
    it('should get messages for a conversation', async () => {
      const mockMessages: Message[] = [
        {
          id: '1',
          conversation_id: '1',
          sender_id: '1',
          content: 'Hello',
          msg_type: 'text',
          created_at: '2024-01-01T00:00:00Z',
        },
      ];

      const mockResponse = {
        data: {
          success: true,
          data: mockMessages,
        },
      };

      mockedAxios.create.mockReturnValueOnce(mockedAxios as any);
      mockedAxios.get.mockResolvedValueOnce(mockResponse as any);

      const result = await api.getMessages('1', 10, 0);

      expect(result.success).toBe(true);
      expect(result.data).toEqual(mockMessages);
    });
  });

  describe('sendMessage', () => {
    it('should send a message', async () => {
      const mockMessage: Message = {
        id: '1',
        conversation_id: '1',
        sender_id: '1',
        content: 'Hello',
        msg_type: 'text',
        created_at: '2024-01-01T00:00:00Z',
      };

      const mockResponse = {
        data: {
          success: true,
          data: mockMessage,
        },
      };

      mockedAxios.create.mockReturnValueOnce(mockedAxios as any);
      mockedAxios.post.mockResolvedValueOnce(mockResponse as any);

      const result = await api.sendMessage({
        conversation_id: '1',
        content: 'Hello',
        msg_type: 'text',
      });

      expect(result.success).toBe(true);
      expect(result.data).toEqual(mockMessage);
    });
  });

  describe('getFriends', () => {
    it('should get friends list', async () => {
      const mockFriends: Friendship[] = [
        {
          id: '1',
          user_id: '1',
          friend_id: '2',
          status: 'accepted',
          created_at: '2024-01-01T00:00:00Z',
        },
      ];

      const mockResponse = {
        data: {
          success: true,
          data: mockFriends,
        },
      };

      mockedAxios.create.mockReturnValueOnce(mockedAxios as any);
      mockedAxios.get.mockResolvedValueOnce(mockResponse as any);

      const result = await api.getFriends();

      expect(result.success).toBe(true);
      expect(result.data).toEqual(mockFriends);
    });
  });

  describe('sendFriendRequest', () => {
    it('should send a friend request', async () => {
      const mockConversation: Conversation = {
        id: '1',
        conversation_type: 'direct',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      };

      const mockResponse = {
        data: {
          success: true,
          data: mockConversation,
        },
      };

      mockedAxios.post.mockResolvedValueOnce(mockResponse as any);

      const result = await api.sendFriendRequest({
        target_user_id: '2',
      });

      expect(result.success).toBe(true);
      expect(result.data).toEqual(mockConversation);
    });
  });

  describe('handleFriendRequest', () => {
    it('should handle a friend request', async () => {
      const mockConversation: Conversation = {
        id: '1',
        conversation_type: 'direct',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      };

      const mockResponse = {
        data: {
          success: true,
          data: mockConversation,
        },
      };

      mockedAxios.post.mockResolvedValueOnce(mockResponse as any);

      const result = await api.handleFriendRequest({
        conversation_id: '1',
        action: 'accept',
      });

      expect(result.success).toBe(true);
      expect(result.data).toEqual(mockConversation);
    });
  });

  describe('health', () => {
    it('should check health status', async () => {
      const mockResponse = {
        data: {
          status: 'ok',
          message: 'Service is healthy',
        },
      };

      mockedAxios.create.mockReturnValueOnce(mockedAxios as any);
      mockedAxios.get.mockResolvedValueOnce(mockResponse as any);

      const result = await api.health();

      expect(result.status).toBe('ok');
      expect(result.message).toBe('Service is healthy');
    });
  });

  describe('createGroup', () => {
    it('should create a group conversation', async () => {
      const mockConversation: Conversation = {
        id: '1',
        conversation_type: 'group',
        name: 'Test Group',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      };

      const mockResponse = {
        data: {
          success: true,
          data: mockConversation,
        },
      };

      mockedAxios.post.mockResolvedValueOnce(mockResponse as any);

      const result = await api.createGroup({
        name: 'Test Group',
        members: ['2', '3'],
      });

      expect(result.success).toBe(true);
      expect(result.data).toEqual(mockConversation);
    });
  });

  describe('getConversationMembers', () => {
    it('should get conversation members', async () => {
      const mockMembers = [
        {
          id: '1',
          conversation_id: '1',
          user_id: '1',
          role: 'owner',
          joined_at: '2024-01-01T00:00:00Z',
        },
      ];

      const mockResponse = {
        data: {
          success: true,
          data: mockMembers,
        },
      };

      mockedAxios.create.mockReturnValueOnce(mockedAxios as any);
      mockedAxios.get.mockResolvedValueOnce(mockResponse as any);

      const result = await api.getConversationMembers('1');

      expect(result.success).toBe(true);
      expect(result.data).toEqual(mockMembers);
    });
  });

  describe('addMemberToConversation', () => {
    it('should add member to conversation', async () => {
      const mockResponse = {
        data: {
          success: true,
          message: 'Member added successfully',
        },
      };

      mockedAxios.post.mockResolvedValueOnce(mockResponse as any);

      const result = await api.addMemberToConversation({
        conversation_id: '1',
        user_id: '2',
        role: 'member',
      });

      expect(result.success).toBe(true);
      expect(result.message).toBe('Member added successfully');
    });
  });

  describe('removeMemberFromConversation', () => {
    it('should remove member from conversation', async () => {
      const mockResponse = {
        data: {
          success: true,
          message: 'Member removed successfully',
        },
      };

      mockedAxios.delete.mockResolvedValueOnce(mockResponse as any);

      const result = await api.removeMemberFromConversation({
        conversation_id: '1',
        user_id: '2',
      });

      expect(result.success).toBe(true);
      expect(result.message).toBe('Member removed successfully');
    });
  });
});
