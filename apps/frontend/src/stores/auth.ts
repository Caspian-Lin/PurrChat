import { ref, computed } from 'vue';
import type { User } from '../models/types';
import { api } from '../models/api';

// 认证状态
const token = ref<string | null>(localStorage.getItem('token'));
const user = ref<User | null>(
  localStorage.getItem('user') ? JSON.parse(localStorage.getItem('user')!) : null
);
const loading = ref(false);
const error = ref<string | null>(null);

// 计算属性
const isAuthenticated = computed(() => !!token.value);
const currentUser = computed(() => user.value);

// 设置 token 和用户信息
function setAuth(authToken: string, authUser: User) {
  token.value = authToken;
  user.value = authUser;
  localStorage.setItem('token', authToken);
  localStorage.setItem('user', JSON.stringify(authUser));
}

// 清除认证信息
function clearAuth() {
  token.value = null;
  user.value = null;
  localStorage.removeItem('token');
  localStorage.removeItem('user');
}

// 用户注册
async function register(username: string, password: string, email: string, phone: string) {
  loading.value = true;
  error.value = null;
  try {
    const response = await api.register({ username, password, email, phone });
    if (response.success && response.data) {
      const loginResp = response.data as unknown as { token: string; user: User };
      setAuth(loginResp.token, loginResp.user);
      return true;
    }
    error.value = response.message || '注册失败';
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
  loading.value = true;
  error.value = null;
  try {
    const response = await api.login({ email, password });
    if (response.success && response.data) {
      const loginResp = response.data as unknown as { token: string; user: User };
      setAuth(loginResp.token, loginResp.user);
      return true;
    }
    error.value = response.message || '登录失败';
    return false;
  } catch (err: any) {
    error.value = err.response?.data?.message || '登录失败';
    return false;
  } finally {
    loading.value = false;
  }
}

// 获取当前用户信息
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
    return false;
  } catch (err: any) {
    error.value = err.response?.data?.message || '获取用户信息失败';
    return false;
  } finally {
    loading.value = false;
  }
}

// 登出
function logout() {
  clearAuth();
}

// 导出认证 store
export const useAuth = () => ({
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
  fetchUser,
  setAuth,
  clearAuth,
});
