import type { z } from 'zod';
import type { FlowConnection, EventPort, RunTrace } from '@purrchat/workflow-types';

// ─── 节点定义 ────────────────────────────────────────────────

export interface NodeDefinition<TConfig extends Record<string, any> = Record<string, any>> {
  type: string;
  label: string;
  category: 'trigger' | 'processing' | 'control' | 'output';
  icon: string;
  configSchema: z.ZodType<TConfig, z.ZodTypeDef, any>;
  execute: (input: NodeInput, config: Record<string, any>, ctx: NodeContext) => Promise<NodeOutput>;
}

export interface NodeInput {
  ports: Record<string, string>;  // portId -> resolved value
  rawInput: string;               // 原始用户消息
}

export interface NodeOutput {
  ports: Record<string, string>;  // output portId -> value
}

export interface NodeContext {
  variables: Record<string, string>;
  eventOutputs: Record<string, string>;
  contextBuffer: Array<{ role: string; content: string }>;
  // 完整执行上下文，供节点内部 resolveTemplate / evaluateCondition 使用
  nodeOutputs: Record<string, Record<string, string>>;
  nameResolver: Record<string, string>;
  finalReply: string;
  // 统一变量解析器所需上下文
  nodeKeyMap: Record<string, string>;
  rawInput: string;
  senderId: string;
  senderName: string;
  conversationId: string;
  history: Array<{ role: string; content: string }>;
  secrets: Record<string, string>;
  session: Record<string, string>;
}

// ─── Blueprint（工作流定义） ──────────────────────────────────

export interface BlueprintNode {
  id: string;
  type: string;
  name: string;
  /** 稳定 key：用于变量引用 ${nodes.<key>.outputs.<port>} */
  key?: string;
  config: Record<string, any>;
  ports?: EventPort[];
  position?: { x: number; y: number };
}

export interface BlueprintConnection {
  id: string;
  sourceNodeId: string;
  sourcePortId: string;
  targetNodeId: string;
  targetPortId: string;
}

export interface Blueprint {
  nodes: BlueprintNode[];
  connections: BlueprintConnection[];
  endConditions: Array<{ type: string; pattern?: string; value?: number }>;
}

// ─── 执行上下文（XState machine context） ───────────────────

export interface ExecutionContext {
  nodeOutputs: Record<string, Record<string, string>>;  // nodeId -> { portId -> value }
  variables: Record<string, string>;
  eventOutputs: Record<string, string>;  // eventId -> output
  contextBuffer: Array<{ role: string; content: string }>;
  finalReply: string;
  nameResolver: Record<string, string>;  // "nodeName.portName" -> "nodeID:portID"
  nodeKeyMap: Record<string, string>;  // "nodeKey" -> "nodeId"
  // 会话元信息
  senderId: string;
  senderName: string;
  conversationId: string;
  rawInput: string;
  history: Array<{ role: string; content: string }>;
  session: Record<string, string>;
  /** 安装时授予的 capabilities，运行时强制校验用（granted ⊆ requested）；undefined 表示不校验 */
  grantedCapabilities?: string[];
  /** 最近一次节点执行错误（用于透传 capability 拒绝等错误信息） */
  lastError?: string;
  /** 运行时解密后的 secret（key→value），引用解析用 */
  secrets?: Record<string, string>;
}

// ─── Actor 输入（runtime -> machine） ────────────────────────

/**
 * 创建 actor 时传入的输入，用于初始化 machine context。
 * runtime 负责构建，compiler 在 context 工厂函数中合并并注入 nameResolver。
 */
export interface ActorInput {
  rawInput: string;
  senderName: string;
  senderId: string;
  conversationId: string;
  time: string;
  contextBuffer: Array<{ role: string; content: string }>;
  variables: Record<string, string>;  // 额外会话变量
  /** 安装时授予的 capabilities */
  grantedCapabilities?: string[];
  /** 运行时解密后的 secret 明文（key→value），用于解析 secrets.<name> 引用 */
  secrets?: Record<string, string>;
}

// ─── XState 事件 ─────────────────────────────────────────────

/**
 * 用户消息事件。trigger / wait 节点监听此事件以推进工作流。
 */
export interface UserMessageEvent {
  type: 'USER_MESSAGE';
  input: string;
  senderName?: string;
  senderId?: string;
  conversationId?: string;
  time?: string;
}

export type WorkflowEvent = UserMessageEvent | { type: string };

// ─── 执行结果 ────────────────────────────────────────────────

export const enum ExecutionStatus {
  Done = 'done',
  Waiting = 'waiting',  // 工作流暂停在 wait 节点，等待下一条消息
  Error = 'error',
}

export interface ExecuteResult {
  reply: string;
  status: ExecutionStatus;
  /** 工作流是否仍保持会话（停在 wait 节点等待后续消息） */
  sessionActive: boolean;
  /** 当前会话轮次（从 1 开始） */
  round: number;
  /** 本次执行的运行 ID（由调用方生成） */
  runId?: string;
  /** 本次执行的节点 trace（一次性执行时由 TraceCollector 生成） */
  trace?: RunTrace;
}
