import type { WorkflowSpec } from './workflow';

// ─── 调试相关类型 ────────────────────────────────────────────

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
  workflow_config?: WorkflowSpec;
  special_mode_config?: WorkflowSpec;
}

export interface DebugStepRequest {
  session_id: string;
}

export interface DebugResetRequest {
  session_id: string;
}
