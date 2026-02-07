import { describe, it, expect, beforeEach, vi } from 'vitest';
import axios from 'axios';
import { api } from '../models/api';
import type { User, Conversation, Message, Friendship } from '../models/types';

// Mock axios
vi.mock('axios');

describe('API Client', () => {
  const mockedAxios = vi.mocked(axios);

  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
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
          conversation_type: 'friend',
          user1_id: '1',
          user2_id: '2',
          has_pending_request: false,
          request_status: 'accepted',
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

      mockedAxios.create.mockReturnValueOnce(mockedAxios as any);
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
        conversation_type: 'stranger',
        user1_id: '1',
        user2_id: '2',
        has_pending_request: false,
        request_status: 'none',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      };

      const mockResponse = {
        data: {
          success: true,
          data: mockConversation,
        },
      };

      mockedAxios.create.mockReturnValueOnce(mockedAxios as any);
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
        conversation_type: 'friend',
        user1_id: '1',
        user2_id: '2',
        has_pending_request: true,
        request_status: 'pending',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      };

      const mockResponse = {
        data: {
          success: true,
          data: mockConversation,
        },
      };

      mockedAxios.create.mockReturnValueOnce(mockedAxios as any);
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
        conversation_type: 'friend',
        user1_id: '1',
        user2_id: '2',
        has_pending_request: false,
        request_status: 'accepted',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      };

      const mockResponse = {
        data: {
          success: true,
          data: mockConversation,
        },
      };

      mockedAxios.create.mockReturnValueOnce(mockedAxios as any);
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
});
