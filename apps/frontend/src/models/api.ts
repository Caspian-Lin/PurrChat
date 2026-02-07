import axios from 'axios';
import type { AxiosInstance } from 'axios';
import type {
  ApiResponse,
  RegisterRequest,
  LoginRequest,
  LoginResponse,
  User,
  Conversation,
  Message,
  Friendship,
  SendMessageRequest,
  CreateConversationRequest,
  SendFriendRequest,
  HandleFriendRequest,
  UpdateProfileRequest,
} from './types';

// API 基础 URL
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';

// 创建 axios 实例
const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// 请求拦截器 - 添加 token
apiClient.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// 响应拦截器 - 处理错误
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Token 过期或无效，清除本地存储
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      // 可以在这里跳转到登录页
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

// API 方法
export const api = {
  // 用户注册
  register: (data: RegisterRequest): Promise<ApiResponse<User>> => {
    return apiClient.post('/api/register', data).then((res) => res.data);
  },

  // 用户登录
  login: (data: LoginRequest): Promise<ApiResponse<LoginResponse>> => {
    return apiClient.post('/api/login', data).then((res) => res.data);
  },

  // 获取当前用户信息
  me: (): Promise<ApiResponse<User>> => {
    return apiClient.get('/api/me').then((res) => res.data);
  },

  // 更新个人资料
  updateProfile: (data: UpdateProfileRequest): Promise<ApiResponse<User>> => {
    return apiClient.put('/api/profile', data).then((res) => res.data);
  },

  // 搜索用户
  searchUsers: (query: string): Promise<ApiResponse<User[]>> => {
    return apiClient.get('/api/users/search', { params: { query } }).then((res) => res.data);
  },

  // 根据ID获取用户信息
  getUserById: (id: string): Promise<ApiResponse<User>> => {
    return apiClient.get(`/api/users/${id}`).then((res) => res.data);
  },

  // 获取会话列表
  getConversations: (): Promise<ApiResponse<Conversation[]>> => {
    return apiClient.get('/api/conversations').then((res) => res.data);
  },

  // 创建会话
  createConversation: (data: CreateConversationRequest): Promise<ApiResponse<Conversation>> => {
    return apiClient.post('/api/conversations', data).then((res) => res.data);
  },

  // 获取消息列表
  getMessages: (
    conversationId: string,
    limit?: number,
    offset?: number
  ): Promise<ApiResponse<Message[]>> => {
    return apiClient
      .get('/api/messages', {
        params: { conversation_id: conversationId, limit, offset },
      })
      .then((res) => res.data);
  },

  // 发送消息
  sendMessage: (data: SendMessageRequest): Promise<ApiResponse<Message>> => {
    return apiClient.post('/api/messages', data).then((res) => res.data);
  },

  // 获取好友列表
  getFriends: (): Promise<ApiResponse<Friendship[]>> => {
    return apiClient.get('/api/friends').then((res) => res.data);
  },

  // 发送好友请求
  sendFriendRequest: (data: SendFriendRequest): Promise<ApiResponse<Conversation>> => {
    return apiClient.post('/api/friends/request', data).then((res) => res.data);
  },

  // 处理好友请求
  handleFriendRequest: (data: HandleFriendRequest): Promise<ApiResponse<Conversation>> => {
    return apiClient.post('/api/friends/handle', data).then((res) => res.data);
  },

  // 健康检查
  health: (): Promise<{ status: string; message: string }> => {
    return apiClient.get('/health').then((res) => res.data);
  },
};

export default apiClient;
