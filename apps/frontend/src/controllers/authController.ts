import { useAuthStore } from '../stores/auth';
import { useRouter } from 'vue-router';
import { storeToRefs } from 'pinia';

// 认证控制器
export function useAuthController() {
  const auth = useAuthStore();
  const router = useRouter();

  // 使用 storeToRefs 保持响应性
  const { currentUser, isAuthenticated, loading, error, token } = storeToRefs(auth);

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

  // 清空错误信息
  const clearError = () => {
    auth.error = null;
  };

  // 返回响应式属性和方法
  return {
    currentUser,
    isAuthenticated,
    loading,
    error,
    token,
    handleRegister,
    handleLogin,
    handleLogout,
    checkAuth,
    requireAuth,
    requireGuest,
    clearError,
  };
}
