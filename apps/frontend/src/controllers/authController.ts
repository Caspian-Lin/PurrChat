import { useAuth } from '../stores/auth';
import { useRouter } from 'vue-router';

// 认证控制器
export function useAuthController() {
  const auth = useAuth();
  const router = useRouter();

  // 注册处理
  const handleRegister = async (
    username: string,
    password: string,
    email: string,
    phone: string
  ) => {
    const success = await auth.register(username, password, email, phone);
    if (success) {
      router.push('/');
    }
    return success;
  };

  // 登录处理
  const handleLogin = async (username: string, password: string) => {
    const success = await auth.login(username, password);
    if (success) {
      router.push('/');
    }
    return success;
  };

  // 登出处理
  const handleLogout = () => {
    auth.logout();
    router.push('/login');
  };

  // 检查认证状态
  const checkAuth = async () => {
    if (auth.isAuthenticated.value) {
      await auth.fetchUser();
    }
    return auth.isAuthenticated.value;
  };

  // 路由守卫 - 需要认证
  const requireAuth = () => {
    if (!auth.isAuthenticated.value) {
      router.push('/login');
      return false;
    }
    return true;
  };

  // 路由守卫 - 未认证
  const requireGuest = () => {
    if (auth.isAuthenticated.value) {
      router.push('/');
      return false;
    }
    return true;
  };

  return {
    ...auth,
    handleRegister,
    handleLogin,
    handleLogout,
    checkAuth,
    requireAuth,
    requireGuest,
  };
}
