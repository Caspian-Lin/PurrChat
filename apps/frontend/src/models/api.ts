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
  CreateGroupRequest,
  AddMemberRequest,
  RemoveMemberRequest,
  Enrollment,
} from './types';

// API 基础 URL
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';

// 处理相对路径
const getBaseUrl = (): string => {
  if (API_BASE_URL.startsWith('/')) {
    // 相对路径，使用当前协议和主机
    return `${window.location.protocol}//${window.location.host}${API_BASE_URL}`;
  }
  return API_BASE_URL;
};

// 创建 axios 实例
const apiClient: AxiosInstance = axios.create({
  baseURL: getBaseUrl(),
  headers: {
    'Content-Type': 'application/json',
  },
});

// 请求拦截器 - 添加 token
apiClient.interceptors.request.use(
  (config) => {
    console.log('[axios] 请求拦截器', {
      method: config.method?.toUpperCase(),
      url: config.url,
      baseURL: config.baseURL,
      fullURL: `${config.baseURL}${config.url}`,
      headers: config.headers,
      data: config.data,
    });
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    console.error('[axios] 请求拦截器错误', error);
    return Promise.reject(error);
  }
);

// 响应拦截器 - 处理错误
apiClient.interceptors.response.use(
  (response) => {
    console.log('[axios] 响应拦截器成功', {
      status: response.status,
      data: response.data,
      url: response.config.url,
    });
    return response;
  },
  (error) => {
    console.error('[axios] 响应拦截器错误', {
      message: error.message,
      status: error.response?.status,
      data: error.response?.data,
      url: error.config?.url,
    });
    if (error.response?.status === 401) {
      const url = error.config?.url || '';
      // 只有在非登录/注册接口返回 401 时才跳转到登录页
      // 登录和注册接口返回 401 是正常的业务错误（如密码错误），不应触发页面刷新
      if (!url.includes('/api/login') && !url.includes('/api/register')) {
        // Token 过期或无效，清除本地存储
        localStorage.removeItem('token');
        localStorage.removeItem('user');
        // 跳转到登录页
        window.location.href = '/login';
      }
    }
    return Promise.reject(error);
  }
);

// API 方法
export const api = {
  // 用户注册
  register: (data: RegisterRequest): Promise<ApiResponse<LoginResponse>> => {
    return apiClient
      .post('/api/register', data)
      .then((res) => res.data)
      .catch((error) => {
        console.error('[api] 注册请求失败', error);
        // 返回包含错误信息的响应对象，而不是抛出异常
        const errorMessage = error.response?.data?.message || error.message || '注册失败';
        return {
          success: false,
          message: errorMessage,
        };
      });
  },

  // 用户登录
  login: (data: LoginRequest): Promise<ApiResponse<LoginResponse>> => {
    console.log('[api] 发送登录请求', { url: '/api/login', data: { ...data, password: '***' } });
    return apiClient
      .post('/api/login', data)
      .then((res) => {
        console.log('[api] 登录请求响应', res.data);
        return res.data;
      })
      .catch((error) => {
        console.error('[api] 登录请求失败', error);
        // 返回包含错误信息的响应对象，而不是抛出异常
        const errorMessage = error.response?.data?.message || error.message || '登录失败';
        return {
          success: false,
          message: errorMessage,
        };
      });
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

  // 删除会话
  deleteConversation: (conversationId: string): Promise<ApiResponse<void>> => {
    return apiClient.delete(`/api/conversations/${conversationId}`).then((res) => res.data);
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

  // 导出会话的所有消息
  exportMessages: (conversationId: string): Promise<ApiResponse<Message[]>> => {
    return apiClient
      .get('/api/messages/export', {
        params: { conversation_id: conversationId },
      })
      .then((res) => res.data);
  },

  // 增量获取会话的消息（从指定时间之后）
  getMessagesIncremental: (
    conversationId: string,
    sinceTimestamp: number
  ): Promise<ApiResponse<Message[]>> => {
    return apiClient
      .get('/api/messages/incremental', {
        params: { conversation_id: conversationId, since_timestamp: sinceTimestamp },
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

  // 获取待处理的好友请求
  getPendingFriendRequests: (): Promise<ApiResponse<Friendship[]>> => {
    return apiClient.get('/api/friends/pending').then((res) => res.data);
  },

  // 获取所有好友申请记录
  getAllFriendRequests: (): Promise<ApiResponse<Friendship[]>> => {
    return apiClient.get('/api/friends/requests').then((res) => res.data);
  },

  // 发送好友请求
  sendFriendRequest: (data: SendFriendRequest): Promise<ApiResponse<Conversation>> => {
    return apiClient.post('/api/friends/request', data).then((res) => res.data);
  },

  // 处理好友请求
  handleFriendRequest: (data: HandleFriendRequest): Promise<ApiResponse<Conversation>> => {
    return apiClient.post('/api/friends/handle', data).then((res) => res.data);
  },

  // 创建群聊
  createGroup: (data: CreateGroupRequest): Promise<ApiResponse<Conversation>> => {
    return apiClient.post('/api/conversations/group', data).then((res) => res.data);
  },

  // 获取会话成员
  getConversationMembers: (conversationId: string): Promise<ApiResponse<Enrollment[]>> => {
    return apiClient
      .get('/api/conversations/members', {
        params: { conversation_id: conversationId },
      })
      .then((res) => res.data);
  },

  // 添加成员到会话
  addMemberToConversation: (data: AddMemberRequest): Promise<ApiResponse<void>> => {
    return apiClient.post('/api/conversations/members', data).then((res) => res.data);
  },

  // 从会话中移除成员
  removeMemberFromConversation: (data: RemoveMemberRequest): Promise<ApiResponse<void>> => {
    return apiClient.delete('/api/conversations/members', { data }).then((res) => res.data);
  },

  // 健康检查
  health: (): Promise<{ status: string; message: string }> => {
    return apiClient.get('/health').then((res) => res.data);
  },
};

export default apiClient;
