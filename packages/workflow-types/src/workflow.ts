import type { EventPort, FlowConnection, EventType } from './ports';

// ─── 机制配置 ────────────────────────────────────────────────

// 机制配置（Bot 的新统一配置格式）
export interface MechanismConfig {
  mechanisms: Mechanism[];
}

// 单个机制 = 触发规则 + 回复设置
export interface Mechanism {
  id: string;
  name: string;
  enabled: boolean;
  trigger: TriggerSpec;
  reply: ReplySpec;
}

// ─── 触发规格 ────────────────────────────────────────────────

// 触发规格
export interface TriggerSpec {
  type: 'rule' | 'probability';
  rules?: TriggerRule[];
  probability?: number;
}

// 触发规则
export interface TriggerRule {
  type: 'keyword' | 'regex' | 'command' | 'equals';
  pattern: string;
  case_sensitive?: boolean;
}

// ─── 回复规格 ────────────────────────────────────────────────

// 回复规格
export interface ReplySpec {
  type: 'predefined' | 'llm' | 'workflow' | 'special_mode';
  predefined?: PredefinedConfig;
  llm?: LLMConfig;
  workflow?: WorkflowSpec;
  special_mode?: WorkflowSpec;
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

// ─── 工作流规格 ──────────────────────────────────────────────

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

// ─── 事件配置类型 ────────────────────────────────────────────

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

// ─── 运行时类型 ──────────────────────────────────────────────

// 工作流运行时会话（调试用）
export interface WorkflowSession {
  conversation_id: string;
  bot_id: string;
  bot_name: string;
  round: number;
  started_at: string;
  event_outputs: Record<string, string>;
}
