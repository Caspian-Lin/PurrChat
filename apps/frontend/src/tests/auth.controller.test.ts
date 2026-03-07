import { describe, it, expect, beforeEach, vi } from 'vitest';
import { useAuthController } from '../controllers/authController';

// Mock vue-router
const mockRouter = {
  push: vi.fn(),
};

vi.mock('vue-router', () => ({
  useRouter: vi.fn(() => mockRouter),
}));

// Mock auth store
const mockAuth = {
  token: null,
  user: null,
  loading: false,
  error: null,
  isAuthenticated: false,
  currentUser: null,
  register: vi.fn(),
  login: vi.fn(),
  logout: vi.fn(),
  fetchUser: vi.fn(),
  setAuth: vi.fn(),
  clearAuth: vi.fn(),
};

vi.mock('../stores/auth', () => ({
  useAuthStore: vi.fn(() => mockAuth),
}));

describe('Auth Controller', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockRouter.push.mockClear();
    mockAuth.token = null;
    mockAuth.user = null;
    mockAuth.isAuthenticated = false;
    mockAuth.loading = false;
    mockAuth.error = null;
  });

  describe('handleRegister', () => {
    it('should handle successful registration', async () => {
      mockAuth.register.mockResolvedValueOnce(true);

      const controller = useAuthController();
      const result = await controller.handleRegister(
        'testuser',
        'password123',
        'test@example.com',
        '1234567890'
      );

      expect(result).toBe(true);
      expect(mockAuth.register).toHaveBeenCalledWith(
        'testuser',
        'password123',
        'test@example.com',
        '1234567890'
      );
      expect(mockRouter.push).toHaveBeenCalledWith('/');
    });

    it('should handle failed registration', async () => {
      mockAuth.register.mockResolvedValueOnce(false);

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
      mockAuth.login.mockResolvedValueOnce(true);

      const controller = useAuthController();
      const result = await controller.handleLogin('test@example.com', 'password123');

      expect(result).toBe(true);
      expect(mockAuth.login).toHaveBeenCalledWith('test@example.com', 'password123');
      expect(mockRouter.push).toHaveBeenCalledWith('/');
    });

    it('should handle failed login', async () => {
      mockAuth.login.mockResolvedValueOnce(false);

      const controller = useAuthController();
      const result = await controller.handleLogin('test@example.com', 'wrongpassword');

      expect(result).toBe(false);
      expect(mockRouter.push).not.toHaveBeenCalled();
    });
  });

  describe('handleLogout', () => {
    it('should handle logout and redirect to login', () => {
      const controller = useAuthController();
      controller.handleLogout();

      expect(mockAuth.logout).toHaveBeenCalled();
      expect(mockRouter.push).toHaveBeenCalledWith('/login');
    });
  });

  describe('checkAuth', () => {
    it('should fetch user when authenticated', async () => {
      mockAuth.isAuthenticated = true;
      mockAuth.fetchUser.mockResolvedValueOnce(true);

      const controller = useAuthController();
      const result = await controller.checkAuth();

      expect(result).toBe(true);
      expect(mockAuth.fetchUser).toHaveBeenCalled();
    });

    it('should not fetch user when not authenticated', async () => {
      mockAuth.isAuthenticated = false;

      const controller = useAuthController();
      const result = await controller.checkAuth();

      expect(result).toBe(false);
      expect(mockAuth.fetchUser).not.toHaveBeenCalled();
    });
  });

  describe('requireAuth', () => {
    it('should redirect to login when not authenticated', () => {
      mockAuth.isAuthenticated = false;

      const controller = useAuthController();
      const result = controller.requireAuth();

      expect(result).toBe(false);
      expect(mockRouter.push).toHaveBeenCalledWith('/login');
    });

    it('should not redirect when authenticated', () => {
      mockAuth.isAuthenticated = true;

      const controller = useAuthController();
      const result = controller.requireAuth();

      expect(result).toBe(true);
      expect(mockRouter.push).not.toHaveBeenCalled();
    });
  });

  describe('requireGuest', () => {
    it('should redirect to home when authenticated', () => {
      mockAuth.isAuthenticated = true;

      const controller = useAuthController();
      const result = controller.requireGuest();

      expect(result).toBe(false);
      expect(mockRouter.push).toHaveBeenCalledWith('/');
    });

    it('should not redirect when not authenticated', () => {
      mockAuth.isAuthenticated = false;

      const controller = useAuthController();
      const result = controller.requireGuest();

      expect(result).toBe(true);
      expect(mockRouter.push).not.toHaveBeenCalled();
    });
  });
});
