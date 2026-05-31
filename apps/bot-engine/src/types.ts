import type { MechanismConfig, DebugBotRequest } from '@purrchat/workflow-types';

// ─── 执行请求/响应 ───────────────────────────────────────────

export interface ExecuteRequest {
  conversation_id: string;
  bot_id: string;
  bot_name: string;
  sender_id: string;
  sender_name: string;
  content: string;
  msg_type: string;
  mechanism_config: MechanismConfig;
  context_messages?: Array<{ role: string; content: string }>;
}

export interface ExecuteResponse {
  reply: string;
  session_active: boolean;
  session_id?: string;
  triggered: boolean;
  mechanism_id?: string;
  mechanism_name?: string;
  reply_type?: string;
  execution_ms?: number;
}

// ─── 调试请求/响应 ───────────────────────────────────────────

export type DebugRequest = DebugBotRequest & {
  bot_id: string;
};

export interface DebugStepRequest {
  session_id: string;
}

export interface DebugResetRequest {
  session_id: string;
}
