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
  UpdateConversationRequest,
  UpdateMemberRoleRequest,
  Enrollment,
  UploadRequest,
  UploadResponse,
  ConfirmUploadRequest,
  ConfirmUploadResponse,
  UserSettings,
  UpdateSettingsRequest,
  ChangePasswordRequest,
  DeleteAccountRequest,
  Bot,
  CreateBotRequest,
  UpdateBotRequest,
  DeployBotRequest,
  UpdateDeploymentStatusRequest,
  BotDeployment,
  PaginatedSearchResult,
  DeployableConversation,
  DebugBotRequest,
  DebugStepRequest,
  DebugResetRequest,
  DebugTraceResult,
} from './types';
import { getApiBaseUrl, getStorageApiBaseUrl, getBotEngineUrl, logger } from '../config/app';

// 创建 axios 实例
const apiClient: AxiosInstance = axios.create({
  baseURL: getApiBaseUrl(),
  headers: {
    'Content-Type': 'application/json',
  },
  withCredentials: true, // 启用 Cookie 携带
});

// 记录配置信息
logger.info('API 配置', {
  baseURL: getApiBaseUrl(),
  env: import.meta.env.VITE_APP_ENV,
  client: import.meta.env.VITE_APP_CLIENT,
});

// 请求拦截器 - 日志记录（Cookie 由浏览器自动发送）
apiClient.interceptors.request.use(
  (config) => {
    console.log('[axios] 请求拦截器', {
      method: config.method?.toUpperCase(),
      url: config.url,
      baseURL: config.baseURL,
      fullURL: `${config.baseURL}${config.url}`,
      data: config.data,
    });
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
        // Cookie 过期或无效，清除本地用户信息并跳转登录页
        localStorage.removeItem('user');
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

  // 修改密码
  changePassword: (data: ChangePasswordRequest): Promise<ApiResponse<void>> => {
    return apiClient.put('/api/password', data).then((res) => res.data);
  },

  // 用户登出（清除服务端 Cookie）
  logout: (): Promise<ApiResponse<void>> => {
    return apiClient.post('/api/logout').then((res) => res.data);
  },

  // 注销账号
  deleteAccount: (data: DeleteAccountRequest): Promise<ApiResponse<void>> => {
    return apiClient.delete('/api/account', { data }).then((res) => res.data);
  },

  // 获取 Turnstile 配置（site_key）
  getTurnstileConfig: (): Promise<{ enabled: boolean; site_key?: string }> => {
    return apiClient.get('/api/turnstile-config').then((res) => res.data);
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

  // 更新群聊信息（名称、头像）
  updateConversation: (
    conversationId: string,
    data: UpdateConversationRequest
  ): Promise<ApiResponse<void>> => {
    return apiClient
      .put('/api/conversations', data, { params: { conversation_id: conversationId } })
      .then((res) => res.data);
  },

  // 更新成员角色（转让群主、设置管理员）
  updateMemberRole: (data: UpdateMemberRoleRequest): Promise<ApiResponse<void>> => {
    return apiClient.put('/api/conversations/members/role', data).then((res) => res.data);
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

  // 拍一拍
  pokeMessage: (conversationId: string, targetUserId: string): Promise<ApiResponse<Message>> => {
    return apiClient
      .post('/api/messages/poke', {
        conversation_id: conversationId,
        target_user_id: targetUserId,
      })
      .then((res) => res.data);
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

  // 获取用户设置
  getSettings: (): Promise<ApiResponse<UserSettings>> => {
    return apiClient.get('/api/settings').then((res) => res.data);
  },

  // 更新用户设置
  updateSettings: (data: UpdateSettingsRequest): Promise<ApiResponse<UserSettings>> => {
    return apiClient.put('/api/settings', data).then((res) => res.data);
  },

  // ===== Bot API =====

  // 获取用户创建的 Bot 列表
  getBots: (): Promise<ApiResponse<Bot[]>> => {
    return apiClient.get('/api/bots').then((res) => res.data);
  },

  // 搜索公开 Bot（分页）
  searchBots: (
    query: string,
    limit = 20,
    offset = 0
  ): Promise<ApiResponse<PaginatedSearchResult>> => {
    return apiClient
      .get('/api/bots/search', { params: { query, limit, offset } })
      .then((res) => res.data);
  },

  // 获取可部署 Bot 的群聊列表
  getDeployableConversations: (botId: string): Promise<ApiResponse<DeployableConversation[]>> => {
    return apiClient.get(`/api/bots/${botId}/deployable-conversations`).then((res) => res.data);
  },

  // 获取 Bot 部署列表
  getBotDeployments: (): Promise<ApiResponse<BotDeployment[]>> => {
    return apiClient.get('/api/bots/deployments').then((res) => res.data);
  },

  // 获取会话中的活跃 Bot 列表
  getConversationBots: (conversationId: string): Promise<ApiResponse<BotDeployment[]>> => {
    return apiClient.get(`/api/conversations/${conversationId}/bots`).then((res) => res.data);
  },

  // 获取 Bot 详情
  getBot: (botId: string): Promise<ApiResponse<Bot>> => {
    return apiClient.get(`/api/bots/${botId}`).then((res) => res.data);
  },

  // 创建 Bot
  createBot: (data: CreateBotRequest): Promise<ApiResponse<Bot>> => {
    return apiClient.post('/api/bots', data).then((res) => res.data);
  },

  // 更新 Bot
  updateBot: (botId: string, data: UpdateBotRequest): Promise<ApiResponse<Bot>> => {
    return apiClient.put(`/api/bots/${botId}`, data).then((res) => res.data);
  },

  // 删除 Bot
  deleteBot: (botId: string): Promise<ApiResponse<void>> => {
    return apiClient.delete(`/api/bots/${botId}`).then((res) => res.data);
  },

  // 部署 Bot 到会话
  deployBot: (botId: string, data: DeployBotRequest): Promise<ApiResponse<BotDeployment>> => {
    return apiClient.post(`/api/bots/${botId}/deploy`, data).then((res) => res.data);
  },

  // 从会话移除 Bot
  undeployBot: (botId: string, conversationId: string): Promise<ApiResponse<void>> => {
    return apiClient
      .delete(`/api/bots/${botId}/deploy`, {
        params: { conversation_id: conversationId },
      })
      .then((res) => res.data);
  },

  // 更新部署状态（暂停/恢复）
  updateDeploymentStatus: (
    botId: string,
    data: UpdateDeploymentStatusRequest
  ): Promise<ApiResponse<void>> => {
    return apiClient.put(`/api/bots/${botId}/deploy/status`, data).then((res) => res.data);
  },

  // 创建与 Bot 的私聊会话
  createBotConversation: (botId: string): Promise<ApiResponse<Conversation>> => {
    return apiClient.post(`/api/bots/${botId}/conversation`).then((res) => res.data);
  },

  // 激活 Bot 工作流
  activateWorkflow: (botId: string, conversationId: string): Promise<ApiResponse<void>> => {
    return apiClient
      .post(`/api/bots/${botId}/workflow/activate`, { conversation_id: conversationId })
      .then((res) => res.data);
  },

  // 停用 Bot 工作流
  deactivateWorkflow: (botId: string, conversationId: string): Promise<ApiResponse<void>> => {
    return apiClient
      .post(`/api/bots/${botId}/workflow/deactivate`, { conversation_id: conversationId })
      .then((res) => res.data);
  },

  // ─── 调试 API ───

  debugBot: (botId: string, data: DebugBotRequest): Promise<ApiResponse<DebugTraceResult>> => {
    return apiClient.post(`/api/bots/${botId}/debug`, data).then((res) => res.data);
  },

  debugStep: (botId: string, data: DebugStepRequest): Promise<ApiResponse<DebugTraceResult>> => {
    return apiClient.post(`/api/bots/${botId}/debug/step`, data).then((res) => res.data);
  },

  debugReset: (botId: string, data: DebugResetRequest): Promise<ApiResponse<void>> => {
    return apiClient.post(`/api/bots/${botId}/debug/reset`, data).then((res) => res.data);
  },
};

// ─── Bot 微服务 API（XState 引擎） ───

const botEngineUrl = getBotEngineUrl();

export const botEngineApi = {
  // 是否配置了 Bot 微服务
  isAvailable: (): boolean => !!botEngineUrl,

  // 执行消息处理
  execute: async (data: {
    conversation_id: string;
    bot_id: string;
    bot_name: string;
    sender_id: string;
    sender_name: string;
    content: string;
    msg_type: string;
    mechanism_config: any;
    context_messages?: Array<{ role: string; content: string }>;
  }): Promise<{ reply: string; session_active: boolean; session_id?: string }> => {
    const resp = await fetch(`${botEngineUrl}/execute`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    if (!resp.ok) throw new Error(`Bot engine error: ${resp.status}`);
    return resp.json();
  },

  // 健康检查
  healthCheck: async (): Promise<boolean> => {
    try {
      const resp = await fetch(`${botEngineUrl}/health`);
      return resp.ok;
    } catch {
      return false;
    }
  },
};

// 存储服务 API 客户端
const storageApiClient: AxiosInstance = axios.create({
  baseURL: getStorageApiBaseUrl(),
  headers: {
    'Content-Type': 'application/json',
  },
  withCredentials: true, // 启用 Cookie 携带
});

// 存储服务请求拦截器 - Cookie 由浏览器自动发送
storageApiClient.interceptors.request.use(
  (config) => {
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// 存储服务 API 方法
export const storageApi = {
  // 申请上传（获取预签名 URL）
  requestUpload: (data: UploadRequest): Promise<ApiResponse<UploadResponse>> => {
    return storageApiClient.post('/api/files/upload/request', data).then((res) => res.data);
  },

  // 确认上传
  confirmUpload: (data: ConfirmUploadRequest): Promise<ApiResponse<ConfirmUploadResponse>> => {
    return storageApiClient.post('/api/files/upload/confirm', data).then((res) => res.data);
  },
};

export default apiClient;
