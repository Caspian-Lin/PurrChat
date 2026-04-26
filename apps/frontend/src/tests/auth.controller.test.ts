import { describe, it, expect, beforeEach, vi } from 'vitest';
import { useAuthController } from '../controllers/authController';
import { useAuthStore } from '../stores/auth';
import { api } from '../models/api';

// Mock vue-router
const mockRouter = {
  push: vi.fn(),
};

vi.mock('vue-router', () => ({
  useRouter: vi.fn(() => mockRouter),
}));

// Mock API
vi.mock('../models/api', () => ({
  api: {
    register: vi.fn(),
    login: vi.fn(),
    me: vi.fn(),
    logout: vi.fn(),
  },
}));

describe('Auth Controller', () => {
  let authStore: ReturnType<typeof useAuthStore>;

  beforeEach(() => {
    vi.clearAllMocks();
    mockRouter.push.mockClear();

    // Get real auth store
    authStore = useAuthStore();

    // Reset store state
    authStore.token = null;
    authStore.user = null;
    authStore.loading = false;
    authStore.error = null;

    // Reset API mocks
    vi.mocked(api.register).mockReset();
    vi.mocked(api.login).mockReset();
    vi.mocked(api.me).mockReset();
    vi.mocked(api.logout).mockReset();
  });

  describe('handleRegister', () => {
    it('should handle successful registration', async () => {
      vi.mocked(api.register).mockResolvedValueOnce({
        success: true,
        data: {
          token: 'test-token',
          user: {
            id: '1',
            uid: 1,
            username: 'testuser',
            avatar_url: 'http://example.com/avatar.png',
            email: 'test@example.com',
            email_verified: true,
            phone: '1234567890',
            phone_verified: true,
            created_at: '2024-01-01T00:00:00Z',
          },
        },
      });

      const controller = useAuthController();
      const result = await controller.handleRegister(
        'testuser',
        'password123',
        'test@example.com',
        '1234567890'
      );

      expect(result).toBe(true);
      expect(api.register).toHaveBeenCalledWith({
        username: 'testuser',
        password: 'password123',
        email: 'test@example.com',
        phone: '1234567890',
      });
      expect(mockRouter.push).toHaveBeenCalledWith('/');
    });

    it('should handle failed registration', async () => {
      vi.mocked(api.register).mockResolvedValueOnce({
        success: false,
        message: 'username already exists',
      });

      const controller = useAuthController();
      const result = await controller.handleRegister(
        'testuser',
        'password123',
        'test@example.com',
        '1234567890'
      );

      expect(result).toBe(false);
      expect(mockRouter.push).not.toHaveBeenCalled();
    });
  });

  describe('handleLogin', () => {
    it('should handle successful login', async () => {
      vi.mocked(api.login).mockResolvedValueOnce({
        success: true,
        data: {
          token: 'test-token',
          user: {
            id: '1',
            uid: 1,
            username: 'testuser',
            avatar_url: 'http://example.com/avatar.png',
            email: 'test@example.com',
            email_verified: true,
            phone: '1234567890',
            phone_verified: true,
            created_at: '2024-01-01T00:00:00Z',
          },
        },
      });

      const controller = useAuthController();
      const result = await controller.handleLogin('test@example.com', 'password123');

      expect(result).toBe(true);
      expect(api.login).toHaveBeenCalledWith({
        email: 'test@example.com',
        password: 'password123',
      });
      expect(mockRouter.push).toHaveBeenCalledWith('/');
    });

    it('should handle failed login', async () => {
      vi.mocked(api.login).mockResolvedValueOnce({
        success: false,
        message: 'invalid email or password',
      });

      const controller = useAuthController();
      const result = await controller.handleLogin('test@example.com', 'wrongpassword');

      expect(result).toBe(false);
      expect(mockRouter.push).not.toHaveBeenCalled();
    });
  });

  describe('handleLogout', () => {
    it('should handle logout and redirect to login', async () => {
      // Set up authenticated state
      authStore.token = 'test-token';
      authStore.user = {
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

      vi.mocked(api.logout).mockResolvedValueOnce(undefined);

      const controller = useAuthController();
      await controller.handleLogout();

      expect(authStore.token).toBe(null);
      expect(authStore.user).toBe(null);
      expect(mockRouter.push).toHaveBeenCalledWith('/login');
    });
  });

  describe('checkAuth', () => {
    it('should fetch user when authenticated', async () => {
      // Set up authenticated state
      authStore.token = 'test-token';

      vi.mocked(api.me).mockResolvedValueOnce({
        success: true,
        data: {
          id: '1',
          uid: 1,
          username: 'testuser',
          avatar_url: 'http://example.com/avatar.png',
          email: 'test@example.com',
          email_verified: true,
          phone: '1234567890',
          phone_verified: true,
          created_at: '2024-01-01T00:00:00Z',
        },
      });

      const controller = useAuthController();
      const result = await controller.checkAuth();

      expect(result).toBe(true);
      expect(api.me).toHaveBeenCalled();
    });

    it('should not fetch user when not authenticated', async () => {
      const controller = useAuthController();
      const result = await controller.checkAuth();

      expect(result).toBe(false);
      expect(api.me).not.toHaveBeenCalled();
    });
  });

  describe('requireAuth', () => {
    it('should redirect to login when not authenticated', () => {
      const controller = useAuthController();
      const result = controller.requireAuth();

      expect(result).toBe(false);
      expect(mockRouter.push).toHaveBeenCalledWith('/login');
    });

    it('should not redirect when authenticated', () => {
      // Set up authenticated state
      authStore.token = 'test-token';

      const controller = useAuthController();
      const result = controller.requireAuth();

      expect(result).toBe(true);
      expect(mockRouter.push).not.toHaveBeenCalled();
    });
  });

  describe('requireGuest', () => {
    it('should redirect to home when authenticated', () => {
      // Set up authenticated state
      authStore.token = 'test-token';

      const controller = useAuthController();
      const result = controller.requireGuest();

      expect(result).toBe(false);
      expect(mockRouter.push).toHaveBeenCalledWith('/');
    });

    it('should not redirect when not authenticated', () => {
      const controller = useAuthController();
      const result = controller.requireGuest();

      expect(result).toBe(true);
      expect(mockRouter.push).not.toHaveBeenCalled();
    });
  });
});
