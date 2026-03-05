import { describe, it, expect, beforeEach, vi } from 'vitest';
import { useAuth } from '../stores/auth';
import type { User } from '../models/types';

// Mock api module
vi.mock('../models/api', () => ({
  api: {
    register: vi.fn(),
    login: vi.fn(),
    me: vi.fn(),
  },
}));

describe('Auth Store', () => {
  beforeEach(() => {
    // Clear localStorage before each test
    localStorage.clear();
    // Reset auth state
    const auth = useAuth();
    auth.clearAuth();
  });

  describe('Initial State', () => {
    it('should have null token and user initially', () => {
      const auth = useAuth();
      expect(auth.token.value).toBeNull();
      expect(auth.user.value).toBeNull();
      expect(auth.isAuthenticated.value).toBe(false);
    });
  });

  describe('setAuth', () => {
    it('should set token and user', () => {
      const auth = useAuth();
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

      auth.setAuth('test-token', mockUser);

      expect(auth.token.value).toBe('test-token');
      expect(auth.user.value).toEqual(mockUser);
      expect(localStorage.getItem('token')).toBe('test-token');
      expect(localStorage.getItem('user')).toBe(JSON.stringify(mockUser));
    });
  });

  describe('clearAuth', () => {
    it('should clear token and user', () => {
      const auth = useAuth();
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

      auth.setAuth('test-token', mockUser);
      expect(auth.isAuthenticated.value).toBe(true);

      auth.clearAuth();

      expect(auth.token.value).toBeNull();
      expect(auth.user.value).toBeNull();
      expect(auth.isAuthenticated.value).toBe(false);
      expect(localStorage.getItem('token')).toBeNull();
      expect(localStorage.getItem('user')).toBeNull();
    });
  });

  describe('register', () => {
    it('should register successfully', async () => {
      const { api } = await import('../models/api');
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

      vi.mocked(api.register).mockResolvedValueOnce({
        success: true,
        data: { token: 'test-token', user: mockUser } as any,
      });

      const auth = useAuth();
      const result = await auth.register(
        'testuser',
        'password123',
        'test@example.com',
        '1234567890'
      );

      expect(result).toBe(true);
      expect(auth.token.value).toBe('test-token');
      expect(auth.user.value).toEqual(mockUser);
      expect(api.register).toHaveBeenCalledWith({
        username: 'testuser',
        password: 'password123',
        email: 'test@example.com',
        phone: '1234567890',
      });
    });

    it('should handle registration failure', async () => {
      const { api } = await import('../models/api');

      vi.mocked(api.register).mockResolvedValueOnce({
        success: false,
        message: 'Username already exists',
      });

      const auth = useAuth();
      const result = await auth.register(
        'testuser',
        'password123',
        'test@example.com',
        '1234567890'
      );

      expect(result).toBe(false);
      expect(auth.error.value).toBe('Username already exists');
    });

    it('should handle registration error', async () => {
      const { api } = await import('../models/api');

      vi.mocked(api.register).mockRejectedValueOnce({
        response: {
          data: {
            message: 'Network error',
          },
        },
      });

      const auth = useAuth();
      const result = await auth.register(
        'testuser',
        'password123',
        'test@example.com',
        '1234567890'
      );

      expect(result).toBe(false);
      expect(auth.error.value).toBe('Network error');
    });
  });

  describe('login', () => {
    it('should login successfully', async () => {
      const { api } = await import('../models/api');
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

      vi.mocked(api.login).mockResolvedValueOnce({
        success: true,
        data: { token: 'test-token', user: mockUser } as any,
      });

      const auth = useAuth();
      const result = await auth.login('test@example.com', 'password123');

      expect(result).toBe(true);
      expect(auth.token.value).toBe('test-token');
      expect(auth.user.value).toEqual(mockUser);
      expect(api.login).toHaveBeenCalledWith({
        email: 'test@example.com',
        password: 'password123',
      });
    });

    it('should handle login failure', async () => {
      const { api } = await import('../models/api');

      vi.mocked(api.login).mockResolvedValueOnce({
        success: false,
        message: 'Invalid credentials',
      });

      const auth = useAuth();
      const result = await auth.login('test@example.com', 'wrongpassword');

      expect(result).toBe(false);
      expect(auth.error.value).toBe('Invalid credentials');
    });
  });

  describe('fetchUser', () => {
    it('should fetch user successfully', async () => {
      const { api } = await import('../models/api');
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

      vi.mocked(api.me).mockResolvedValueOnce({
        success: true,
        data: mockUser,
      });

      const auth = useAuth();
      const result = await auth.fetchUser();

      expect(result).toBe(true);
      expect(auth.user.value).toEqual(mockUser);
      expect(localStorage.getItem('user')).toBe(JSON.stringify(mockUser));
    });

    it('should handle fetch user failure', async () => {
      const { api } = await import('../models/api');

      vi.mocked(api.me).mockResolvedValueOnce({
        success: false,
        message: 'User not found',
      });

      const auth = useAuth();
      const result = await auth.fetchUser();

      expect(result).toBe(false);
    });
  });

  describe('logout', () => {
    it('should clear auth state', () => {
      const auth = useAuth();
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

      auth.setAuth('test-token', mockUser);
      expect(auth.isAuthenticated.value).toBe(true);

      auth.logout();

      expect(auth.token.value).toBeNull();
      expect(auth.user.value).toBeNull();
      expect(auth.isAuthenticated.value).toBe(false);
    });
  });
});
