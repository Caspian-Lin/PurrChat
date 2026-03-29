// 用户数据类型定义
export interface User {
  id: string;
  uid: number;
  username: string;
  avatar_url: string;
  email?: string;
  email_verified: boolean;
  phone?: string;
  phone_verified: boolean;
  created_at: string;
}

// 注册请求
export interface RegisterRequest {
  username: string;
  password: string;
  email: string;
  phone: string;
}

// 登录请求
export interface LoginRequest {
  email: string;
  password: string;
}

// 登录响应
export interface LoginResponse {
  token: string;
  user: User;
}

// API 响应
export interface ApiResponse<T = any> {
  success: boolean;
  message?: string;
  data?: T;
}

// 会话类型
export type ConversationType = 'direct' | 'group';

// 好友状态
export type FriendshipStatus = 'pending' | 'accepted' | 'rejected' | 'blocked';

// Enrollment角色
export type EnrollmentRole = 'owner' | 'admin' | 'member';

// Enrollment类型
export interface Enrollment {
  id: string;
  conversation_id: string;
  user_id: string;
  role: EnrollmentRole;
  joined_at: string;
  last_read_at?: string;
  user?: User;
}

// 会话类型
export interface Conversation {
  id: string;
  conversation_type: ConversationType;
  name?: string;
  created_by?: string;
  created_at: string;
  updated_at: string;
  members?: Enrollment[];
  last_message?: Message;
  unread_count?: number;
  friendship_status?: FriendshipStatus; // 好友关系状态（仅私聊会话）
}

// 好友关系类型
export interface Friendship {
  id: string;
  user_id: string;
  friend_id: string;
  conversation_id: string;
  status: FriendshipStatus;
  created_at: string;
  user?: User;
  friend?: User;
}

// 消息类型
export interface Message {
  id: string;
  conversation_id: string;
  sender_id: string;
  content: string;
  msg_type: 'text' | 'image';
  created_at: string;
  sender?: User;
  is_read?: boolean; // 消息是否已读
  sendStatus?: 'sending' | 'sent' | 'failed'; // 消息发送状态：发送中、已发送、发送失败
}

// 群组类型
export interface Group {
  id: string;
  name: string;
  owner_id: string;
  avatar_url: string;
  created_at: string;
}

// 搜索用户请求
export interface SearchUsersRequest {
  query: string;
}

// 发送消息请求
export interface SendMessageRequest {
  conversation_id: string;
  content: string;
  msg_type: 'text' | 'image';
}

// 创建会话请求
export interface CreateConversationRequest {
  target_user_id: string;
}

// 发送好友请求
export interface SendFriendRequest {
  target_user_id: string;
}

// 处理好友请求
export interface HandleFriendRequest {
  conversation_id: string;
  action: 'accept' | 'reject';
}

// 更新个人资料请求
export interface UpdateProfileRequest {
  nickname?: string;
  email?: string;
  phone?: string;
}

// 创建群聊请求
export interface CreateGroupRequest {
  name: string;
  members: string[]; // 成员用户ID列表
}

// 添加成员请求
export interface AddMemberRequest {
  conversation_id: string;
  user_id: string;
  role: EnrollmentRole;
}

// 移除成员请求
export interface RemoveMemberRequest {
  conversation_id: string;
  user_id: string;
}
