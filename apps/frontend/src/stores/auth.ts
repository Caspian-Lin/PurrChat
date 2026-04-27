import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import type { User } from '../models/types';
import { api } from '../models/api';
import { messageCacheService } from '../services/messageCache';
import { conversationStateCacheService } from '../services/conversationStateCache';
import { useBotStore } from './bot';

// 错误信息映射：将后端返回的英文错误翻译成中文
function getErrorMessage(backendError: string): string {
  const errorMap: Record<string, string> = {
    // 注册相关错误
    'username already exists': '该用户名已被占用，请使用其他用户名',
    'email already exists': '该邮箱已被注册，请使用其他邮箱或直接登录',
    'phone already exists': '该手机号已被注册，请使用其他手机号或直接登录',
    'Invalid request': '请求格式错误，请检查输入信息',

    // 登录相关错误
    'invalid email or password': '邮箱或密码错误，请检查后重试',
    'Invalid request: invalid email or password': '邮箱或密码错误，请检查后重试',

    // 通用错误
    'Registration successful': '注册成功',
    'Login successful': '登录成功',
    Unauthorized: '未授权，请重新登录',
    'User not found': '用户不存在',
    'invalid current password': '当前密码不正确',
    'new password must be different from current password': '新密码不能与当前密码相同',
    'invalid password': '密码错误，请确认后重试',
    'failed to delete account': '注销失败，请稍后重试',
  };

  // 优先进行精确匹配
  if (errorMap[backendError]) {
    return errorMap[backendError];
  }

  // 检查是否包含某些关键词（作为后备方案）
  if (backendError.toLowerCase().includes('username')) {
    return '用户名相关错误：' + backendError;
  }
  if (backendError.toLowerCase().includes('email')) {
    return '邮箱相关错误：' + backendError;
  }
  if (backendError.toLowerCase().includes('password')) {
    return '密码相关错误：' + backendError;
  }
  if (backendError.toLowerCase().includes('phone')) {
    return '手机号相关错误：' + backendError;
  }

  // 如果没有匹配，返回原始错误信息
  return backendError;
}

export const useAuthStore = defineStore('auth', () => {
  // 认证状态 — token 仅存储在内存中（Pinia ref），由 HttpOnly Cookie 管理实际认证
  const token = ref<string | null>(null);
  const user = ref<User | null>(
    localStorage.getItem('user') ? JSON.parse(localStorage.getItem('user')!) : null
  );
  const loading = ref(false);
  const error = ref<string | null>(null);

  // 计算属性 — Cookie 认证下以 user 为判断依据（user 通过 localStorage 持久化，新页面可立即恢复）
  const isAuthenticated = computed(() => !!user.value);
  const currentUser = computed(() => user.value);

  // 设置 token 和用户信息
  function setAuth(authToken: string, authUser: User) {
    token.value = authToken;
    user.value = authUser;
    // 仅保存用户信息到 localStorage（不再保存 token）
    localStorage.setItem('user', JSON.stringify(authUser));
    // 切换缓存服务到当前用户（不删除其他用户数据）
    switchStorageUser(authUser.id);
  }

  // 切换缓存服务到指定用户
  function switchStorageUser(userId: string) {
    console.log('[auth store] Switching storage to user:', userId);
    messageCacheService.init(userId);
    conversationStateCacheService.init(userId);
    const botStore = useBotStore();
    botStore.reset();
  }

  // 清除认证信息
  function clearAuth() {
    token.value = null;
    user.value = null;
    localStorage.removeItem('user');
  }

  // 用户注册
  async function register(
    username: string,
    password: string,
    email: string,
    phone: string,
    turnstileToken?: string
  ) {
    loading.value = true;
    error.value = null;
    try {
      const response = await api.register({
        username,
        password,
        email,
        phone,
        turnstile_token: turnstileToken,
      });
      if (response.success && response.data) {
        setAuth(response.data.token, response.data.user);
        return true;
      }
      // 使用 getErrorMessage 将英文错误转换为中文
      error.value = getErrorMessage(response.message || '注册失败');
      return false;
    } catch (err: any) {
      error.value = err.response?.data?.message || '注册失败';
      return false;
    } finally {
      loading.value = false;
    }
  }

  // 用户登录
  async function login(email: string, password: string) {
    console.log('[auth store] login 开始', { email, password: '***' });
    loading.value = true;
    error.value = null;
    try {
      console.log('[auth store] 调用 api.login');
      const response = await api.login({ email, password });
      console.log('[auth store] api.login 响应', response);
      if (response.success && response.data) {
        console.log('[auth store] 登录成功，设置认证信息');
        setAuth(response.data.token, response.data.user);
        return true;
      }
      console.log('[auth store] 登录失败', response.message);
      // 使用 getErrorMessage 将英文错误转换为中文
      const errorMessage = getErrorMessage(response.message || '登录失败');
      console.log('[auth store] 转换后的错误信息:', errorMessage);
      error.value = errorMessage;
      console.log('[auth store] 设置错误信息后的 auth.error:', error.value);
      return false;
    } finally {
      loading.value = false;
    }
  }

  // 获取当前用户信息（同时用于验证 Cookie 有效性）
  async function fetchUser() {
    loading.value = true;
    error.value = null;
    try {
      const response = await api.me();
      if (response.success && response.data) {
        user.value = response.data;
        localStorage.setItem('user', JSON.stringify(response.data));
        return true;
      }
      // Cookie 无效或过期，清除本地状态
      clearAuth();
      return false;
    } catch (err: any) {
      // 401 由 axios 拦截器处理（重定向到登录页），这里清除本地过期数据
      if (err.response?.status === 401) {
        clearAuth();
      }
      error.value = err.response?.data?.message || '获取用户信息失败';
      return false;
    } finally {
      loading.value = false;
    }
  }

  // 登出 — 调用后端清除 Cookie
  async function logout() {
    try {
      await api.logout();
    } catch {
      // 即使后端调用失败，也清除本地状态
    }
    clearAuth();
  }

  // 注销账号
  async function deleteAccount(password: string) {
    loading.value = true;
    error.value = null;
    try {
      const response = await api.deleteAccount({ password });
      if (response.success) {
        clearAuth();
        return true;
      }
      error.value = getErrorMessage(response.message || '注销失败');
      return false;
    } catch (err: any) {
      error.value = err.response?.data?.message || '注销失败';
      return false;
    } finally {
      loading.value = false;
    }
  }

  return {
    // 状态
    token,
    user,
    loading,
    error,
    // 计算属性
    isAuthenticated,
    currentUser,
    // 方法
    register,
    login,
    logout,
    deleteAccount,
    fetchUser,
    setAuth,
    clearAuth,
  };
});
