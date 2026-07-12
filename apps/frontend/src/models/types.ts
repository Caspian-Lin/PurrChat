// 用户数据类型定义

// 端口类型系统（从 portTypes.ts 重导出，供其他模块统一从 types.ts 导入）
import type { PortDataType, EventType, EventPort, FlowConnection } from '../utils/portTypes';
export type { PortDataType, EventType, EventPort, FlowConnection };

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
  is_bot?: boolean;
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
  avatar_url?: string;
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
  msg_type: 'text' | 'image' | 'file' | 'system';
  created_at: string;
  sender?: User;
  is_read?: boolean; // 消息是否已读
  sendStatus?: 'sending' | 'sent' | 'failed'; // 消息发送状态：发送中、已发送、发送失败
  bot_id?: string; // Bot 消息标识
  bot_name?: string; // Bot 名称
  client_message_id?: string; // 客户端幂等消息 ID，用于发送确认去重
}

// 系统消息内容（JSON 格式存储在 Message.content 中）
export interface SystemMessageContent {
  type: 'workflow_start' | 'workflow_end' | 'bot_deployed' | 'bot_undeployed';
  bot_id?: string;
  bot_name?: string;
  user_id?: string;
  user_name?: string;
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
  msg_type: 'text' | 'image' | 'file' | 'system';
  client_message_id?: string;
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
  username?: string;
  avatar_url?: string;
  email?: string;
  phone?: string;
}

// 修改密码请求
export interface ChangePasswordRequest {
  old_password: string;
  new_password: string;
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

// 更新群聊信息请求
export interface UpdateConversationRequest {
  name?: string;
  avatar_url?: string;
}

// 更新成员角色请求
export interface UpdateMemberRoleRequest {
  conversation_id: string;
  user_id: string;
  role: EnrollmentRole;
}

// ===== AI 对话相关类型定义 =====

// AI 配置
export interface AiConfig {
  id: string;
  name: string;
  apiUrl: string;
  apiKey: string;
  model: string;
  temperature: number;
  maxTokens?: number;
  createdAt: string;
  updatedAt: string;
}

// AI 消息角色
export type AiMessageRole = 'system' | 'user' | 'assistant';

// AI 消息
export interface AiMessage {
  id: string;
  role: AiMessageRole;
  content: string;
  thinking?: string; // 思维链内容（reasoning models）
  createdAt: string;
  isStreaming?: boolean;
  isThinking?: boolean; // 是否正在思考（流式思维链阶段）
}

// AI 会话
export interface AiConversation {
  id: string;
  configId: string;
  title: string;
  messages: AiMessage[];
  createdAt: string;
  updatedAt: string;
}

// ===== 文件存储相关类型定义 =====

// 文件上传申请请求
export interface UploadRequest {
  file_name: string;
  file_size: number;
  content_type: string;
  category: string;
  usage: string;
}

// 文件上传申请响应
export interface UploadResponse {
  upload_id: string;
  object_key: string;
  upload_url: string;
  method: string;
  expires_in: number;
}

// 文件上传确认请求
export interface ConfirmUploadRequest {
  upload_id: string;
  object_key: string;
}

// 文件上传确认响应
export interface ConfirmUploadResponse {
  file_id: string;
  object_key: string;
  public_url: string;
}

// 文件消息内容结构（存储在 Message.content 中的 JSON）
export interface FileMessageContent {
  file_id: string;
  file_name: string;
  file_size: number;
  content_type: string;
  thumbnail_url?: string;
  public_url: string;
  category: 'chat-image' | 'file';
}

// ===== 用户设置相关类型定义 =====

// 设置分类 ID
export type SettingsCategoryId = 'account' | 'panels' | 'notifications' | 'general' | 'about';

// 面板可见性设置
export interface PanelVisibilitySettings {
  visiblePanels: ('chat' | 'friends' | 'ai' | 'bots')[];
}

// 通知设置
export interface NotificationSettings {
  messageNotification: boolean;
  friendRequestNotification: boolean;
  groupInviteNotification: boolean;
  systemNotification: boolean;
  soundEnabled: boolean;
  desktopNotificationEnabled: boolean;
}

// 通用设置
export interface GeneralSettings {
  themeMode: 'light' | 'dark';
  themeColor: 'sage' | 'iris' | 'ocean' | 'ember' | 'rose' | 'slate' | 'clay' | 'honey';
  language: string;
  fontSize: 'small' | 'medium' | 'large';
}

// 用户设置（完整）
export interface UserSettings {
  panels: PanelVisibilitySettings;
  notifications: NotificationSettings;
  general: GeneralSettings;
}

// 设置更新请求
export interface UpdateSettingsRequest {
  settings: Partial<UserSettings>;
}

// ===== Bot 相关类型定义 =====

// Bot 状态
export type BotStatus = 'active' | 'disabled';

// Bot 可见性
export type BotVisibility = 'private' | 'public' | 'global';

// Bot 类型
export type BotType = 'builtin' | 'workflow' | 'external';

// Bot 模型
export interface Bot {
  id: string;
  owner_id: string;
  name: string;
  avatar_url: string;
  description: string;
  status: BotStatus;
  bot_type: BotType;
  visibility: BotVisibility;
  discoverability?: 'unlisted' | 'listed' | 'featured';
  published_version?: number;
  requested_capabilities?: string[];
  mechanism_config?: MechanismConfig;
  created_at: string;
  updated_at: string;
}

export type BotApiStatus =
  | 'stable'
  | 'beta'
  | 'partial'
  | 'planned'
  | 'blocked'
  | 'not_applicable'
  | 'rejected';

export interface BotApiProfile {
  version: string;
  id_format: string;
  conversation_key: string;
  message_format: string;
  cq_code_core_format: boolean;
}

export interface BotApiActionCapability {
  name: string;
  aliases?: string[];
  category: string;
  status: BotApiStatus;
  transports: ('universal_websocket' | 'http')[];
  required_capability?: string;
  version: string;
  compatibility_note?: string;
  source: string;
  request_example?: Record<string, unknown>;
  response_example?: Record<string, unknown>;
  references?: string[];
}

export interface BotApiEventCapability {
  post_type: string;
  detail_type: string;
  sub_types?: string[];
  category: string;
  status: BotApiStatus;
  transports: ('universal_websocket' | 'http')[];
  required_capability?: string;
  version: string;
  compatibility_note?: string;
  source: string;
  event_example?: Record<string, unknown>;
  references?: string[];
}

export interface BotApiSegmentCapability {
  type: string;
  status: BotApiStatus;
  compatibility_note?: string;
  fields: { name: string; type: string; required: boolean }[];
}

export interface BotApiCapabilities {
  profile: BotApiProfile;
  actions: BotApiActionCapability[];
  events: BotApiEventCapability[];
  segments: BotApiSegmentCapability[];
}

// Bot 部署（对齐后端 BotInstallation）
export interface BotDeployment {
  id: string;
  app_id: string;
  installed_by: string;
  target_type: 'user' | 'conversation';
  target_id: string;
  granted_capabilities: string[];
  diagnostics_consent?: 'denied' | 'granted';
  status: 'active' | 'paused' | 'disabled';
  installed_at: string;
  updated_at: string;
  app?: Bot;
  target_name?: string;
  target_conversation_type?: string;
}

// 公开 Bot 详情（含统计信息）
export interface PublicBotDetail extends Bot {
  deployment_count: number;
  owner_name: string;
  trigger_summary: string;
}

// 分页搜索结果
export interface PaginatedSearchResult {
  bots: PublicBotDetail[];
  total: number;
  limit: number;
  offset: number;
}

// 可部署的会话
export interface DeployableConversation {
  id: string;
  name: string;
  conversation_type: 'group' | 'direct';
  avatar_url?: string;
  member_count: number;
}

// 创建 Bot 请求
export interface CreateBotRequest {
  name: string;
  avatar_url?: string;
  description?: string;
  bot_type?: BotType;
  visibility?: BotVisibility;
}

// 更新 Bot 请求
export interface UpdateBotRequest {
  name?: string;
  avatar_url?: string;
  description?: string;
  status?: BotStatus;
  visibility?: BotVisibility;
  mechanism_config?: MechanismConfig;
  /** @deprecated 使用 mechanism_config */
  trigger_config?: TriggerConfig;
  /** @deprecated 使用 mechanism_config */
  reply_config?: ReplyConfig;
}

// 部署 Bot 请求
export interface DeployBotRequest {
  conversation_id: string;
}

export interface CreateBotInstallationRequest {
  target_type: 'user' | 'conversation';
  target_id: string;
  granted_capabilities: string[];
  diagnostics_consent?: 'denied' | 'granted';
}

export interface UpdateBotInstallationRequest {
  status?: 'active' | 'paused' | 'disabled';
  granted_capabilities?: string[];
  diagnostics_consent?: 'denied' | 'granted';
}

// 更新部署状态请求
export interface UpdateDeploymentStatusRequest {
  conversation_id: string;
  status: 'active' | 'paused';
}

// 触发配置
/** @deprecated 使用 TriggerSpec（在 MechanismConfig 中） */
export interface TriggerConfig {
  mode: 'rule' | 'probability' | 'conditional';
  rules?: TriggerRule[];
  probability?: number;
  condition?: ConditionConfig;
}

// 触发规则
export interface TriggerRule {
  type: 'keyword' | 'regex' | 'command' | 'equals';
  pattern: string;
  case_sensitive?: boolean;
}

/** @deprecated 使用 TriggerSpec（在 MechanismConfig 中） */
export interface ConditionConfig {
  start_expression: string;
  end_expression: string;
}

// 回复配置
/** @deprecated 使用 ReplySpec（在 MechanismConfig 中） */
export interface ReplyConfig {
  type: 'predefined' | 'llm';
  predefined?: PredefinedConfig;
  llm?: LLMConfig;
}

// 预定义回复配置
export interface PredefinedConfig {
  mode: 'fixed' | 'random' | 'template';
  replies?: string[];
  template?: string;
}

// LLM 回复配置
export interface LLMConfig {
  api_url: string;
  api_key: string;
  model: string;
  system_prompt: string;
  temperature?: number;
  max_tokens?: number;
  context_window?: number;
}

// ===== 机制列表类型定义 =====

// 机制配置（Bot 的新统一配置格式）
export interface MechanismConfig {
  mechanisms: Mechanism[];
}

// 单个机制 = 触发规则（回复行为由 mechanism 级工作流文档定义）
export interface Mechanism {
  id: string;
  name: string;
  enabled: boolean;
  trigger: TriggerSpec;
}

// 触发规格
export interface TriggerSpec {
  type: 'rule' | 'probability';
  rules?: TriggerRule[];
  probability?: number;
}

// 回复规格
export interface ReplySpec {
  type: 'predefined' | 'llm' | 'workflow';
  predefined?: PredefinedConfig;
  llm?: LLMConfig;
  workflow?: WorkflowSpec;
}

// 工作流规格（嵌套在机制中）
export interface WorkflowSpec {
  events: WorkflowEvent[];
  connections?: FlowConnection[];
  end_conditions: WorkflowEndCondition[];
}

// 工作流事件
export interface WorkflowEvent {
  id: string;
  type: EventType;
  name: string;
  key?: string;
  config: Record<string, any>;
  ports?: EventPort[];
  position?: { x: number; y: number };
}

// 工作流结束条件
export interface WorkflowEndCondition {
  type: 'message_match' | 'max_rounds' | 'timeout';
  pattern?: string;
  value?: number;
}

// LLM 事件配置
export interface LLMEventConfig {
  api_url: string;
  api_key: string;
  model: string;
  system_prompt: string;
  temperature?: number;
  max_tokens?: number;
  context_window?: number;
  context_scope?: 'session' | string;
}

// 内置事件配置
export interface BuiltinEventConfig {
  builtin_type: 'random_number' | 'haiku' | 'echo' | 'count' | 'template';
  min?: number;
  max?: number;
  integer?: boolean;
  topic?: string;
  prefix?: string;
  suffix?: string;
  counter_key?: string;
  template?: string;
}

// Python 事件配置
export interface PythonEventConfig {
  code: string;
  timeout_ms?: number;
  input_schema?: Record<string, string>;
  output_schema?: Record<string, string>;
}

// 回复事件配置
export interface ReplyEventConfig {
  template: string;
}

// ─── 调试相关类型 ───

export interface EventTrace {
  event_id: string;
  event_type: 'llm' | 'builtin' | 'reply';
  event_name: string;
  status: 'pending' | 'running' | 'success' | 'error';
  input: string;
  output: string;
  error?: string;
  duration_ms: number;
  context_messages?: DebugContextMessage[];
}

export interface DebugContextMessage {
  role: 'user' | 'assistant' | 'system';
  content: string;
}

export interface DebugTraceResult {
  session_id: string;
  reply: string;
  context_messages: DebugContextMessage[];
  event_traces: EventTrace[];
  waiting_for_step: boolean;
  next_event_id?: string;
  round: number;
}

export interface DebugBotRequest {
  message: string;
  step_mode?: boolean;
  session_id?: string;
  sender_name?: string;
}

export interface DebugStepRequest {
  session_id: string;
}

export interface DebugResetRequest {
  session_id: string;
}

// Bot 调用记录
export interface BotCallLog {
  id: string;
  bot_id: string;
  conversation_id: string;
  sender_id: string;
  sender_name: string;
  trigger_message: string;
  reply_content: string;
  mechanism_id: string;
  mechanism_name: string;
  reply_type: string;
  execution_path: string;
  success: boolean;
  error_message?: string;
  duration_ms: number;
  created_at: string;
  conversation_name?: string;
}

// Bot 调用记录列表响应
export interface BotCallLogListResponse {
  // Older backend versions can encode an empty Go slice as null.
  logs: BotCallLog[] | null;
  total: number;
  limit: number;
  offset: number;
}

// ─── Workflow Document API 类型 (#13) ─────────────────────────

export interface WorkflowDocumentResponse {
  document: import('@purrchat/workflow-types').WorkflowDocument;
  revision: number;
  etag: string;
  published_revision?: number;
}

export interface WorkflowVersion {
  id: string;
  bot_id: string;
  mechanism_id: string;
  revision: number;
  document: import('@purrchat/workflow-types').WorkflowDocument;
  capabilities: string[];
  published_by?: string;
  published_at: string;
}

export interface WorkflowValidationIssue {
  level: 'error' | 'warning';
  code: string;
  message: string;
  path?: string;
  node_id?: string;
  connection_id?: string;
}

export interface WorkflowValidationResponse {
  valid: boolean;
  issues: WorkflowValidationIssue[];
  derived_capabilities?: string[];
}

// ─── Debug Trace 类型 (#15) ───────────────────────────────────

export type NodeTraceStatus = 'pending' | 'running' | 'success' | 'error' | 'skip';

export interface NodeTrace {
  nodeId: string;
  nodeKey?: string;
  nodeType: string;
  nodeName?: string;
  status: NodeTraceStatus;
  input?: Record<string, string>;
  output?: Record<string, string>;
  branch?: string;
  error?: string;
  startTime?: number;
  endTime?: number;
  durationMs?: number;
}

export type RunTraceStatus = 'running' | 'completed' | 'error' | 'cancelled';

export interface RunTrace {
  runId: string;
  status: RunTraceStatus;
  nodes: NodeTrace[];
  startedAt: number;
  completedAt?: number;
  durationMs?: number;
  reply?: string;
  input: string;
  senderName?: string;
  waitingForStep?: boolean;
  session_id?: string;
}

// ===== Bot API Credential =====

export interface BotAPICredential {
  id: string;
  bot_id: string;
  name: string;
  token_prefix: string;
  last_used_at: string | null;
  expires_at: string | null;
  revoked_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface BotAPICredentialSecret {
  credential: BotAPICredential;
  token: string;
}

export interface CreateBotAPICredentialRequest {
  name: string;
  expires_at?: string | null;
}
