import { describe, it, expect, beforeEach, vi } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';
import { useAuthStore } from '../stores/auth';
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
    // Create a fresh pinia instance for each test
    setActivePinia(createPinia());
    // Clear localStorage before each test
    localStorage.clear();
    // Reset auth state
    const auth = useAuthStore();
    auth.clearAuth();
  });

  describe('Initial State', () => {
    it('should have null token and user initially', () => {
      const auth = useAuthStore();
      expect(auth.token).toBeNull();
      expect(auth.user).toBeNull();
      expect(auth.isAuthenticated).toBe(false);
    });
  });

  describe('setAuth', () => {
    it('should set token and user', () => {
      const auth = useAuthStore();
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

      expect(auth.token).toBe('test-token');
      expect(auth.user).toEqual(mockUser);
      expect(localStorage.getItem('token')).toBeNull(); // token 不再存 localStorage
      expect(localStorage.getItem('user')).toBe(JSON.stringify(mockUser));
    });
  });

  describe('clearAuth', () => {
    it('should clear token and user', () => {
      const auth = useAuthStore();
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
      expect(auth.isAuthenticated).toBe(true);

      auth.clearAuth();

      expect(auth.token).toBeNull();
      expect(auth.user).toBeNull();
      expect(auth.isAuthenticated).toBe(false);
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

      const auth = useAuthStore();
      const result = await auth.register(
        'testuser',
        'password123',
        'test@example.com',
        '1234567890'
      );

      expect(result).toBe(true);
      expect(auth.token).toBe('test-token');
      expect(auth.user).toEqual(mockUser);
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

      const auth = useAuthStore();
      const result = await auth.register(
        'testuser',
        'password123',
        'test@example.com',
        '1234567890'
      );

      expect(result).toBe(false);
      expect(auth.error).toBe('用户名相关错误：Username already exists');
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

      const auth = useAuthStore();
      const result = await auth.register(
        'testuser',
        'password123',
        'test@example.com',
        '1234567890'
      );

      expect(result).toBe(false);
      expect(auth.error).toBe('Network error');
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

      const auth = useAuthStore();
      const result = await auth.login('test@example.com', 'password123');

      expect(result).toBe(true);
      expect(auth.token).toBe('test-token');
      expect(auth.user).toEqual(mockUser);
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

      const auth = useAuthStore();
      const result = await auth.login('test@example.com', 'wrongpassword');

      expect(result).toBe(false);
      expect(auth.error).toBe('Invalid credentials');
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

      const auth = useAuthStore();
      const result = await auth.fetchUser();

      expect(result).toBe(true);
      expect(auth.user).toEqual(mockUser);
      expect(localStorage.getItem('user')).toBe(JSON.stringify(mockUser));
    });

    it('should handle fetch user failure', async () => {
      const { api } = await import('../models/api');

      vi.mocked(api.me).mockResolvedValueOnce({
        success: false,
        message: 'User not found',
      });

      const auth = useAuthStore();
      const result = await auth.fetchUser();

      expect(result).toBe(false);
    });
  });

  describe('logout', () => {
    it('should clear auth state', () => {
      const auth = useAuthStore();
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
      expect(auth.isAuthenticated).toBe(true);

      auth.logout();

      expect(auth.token).toBeNull();
      expect(auth.user).toBeNull();
      expect(auth.isAuthenticated).toBe(false);
    });
  });
});
