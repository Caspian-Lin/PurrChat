import { useAuthStore } from '../stores/auth';
import { useRouter } from 'vue-router';

// 认证控制器
export function useAuthController() {
  const auth = useAuthStore();
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
  const handleLogin = async (email: string, password: string) => {
    console.log('[authController] handleLogin 开始', { email, password: '***' });
    const success = await auth.login(email, password);
    console.log('[authController] handleLogin 结果', { success });
    if (success) {
      console.log('[authController] 跳转到首页');
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
    if (auth.isAuthenticated) {
      await auth.fetchUser();
    }
    return auth.isAuthenticated;
  };

  // 路由守卫 - 需要认证
  const requireAuth = () => {
    if (!auth.isAuthenticated) {
      router.push('/login');
      return false;
    }
    return true;
  };

  // 路由守卫 - 未认证
  const requireGuest = () => {
    if (auth.isAuthenticated) {
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
