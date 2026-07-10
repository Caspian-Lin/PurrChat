/**
 * Debug Trace 类型定义
 *
 * 调试 test-run 返回的结构化 trace，用于前端渲染节点执行流。
 * 这些类型同时被 workflow-engine (生成) 和前端 (消费) 使用。
 */

/** 节点执行状态 */
export type NodeTraceStatus = 'pending' | 'running' | 'success' | 'error' | 'skip';

/** 单个节点的 trace 记录 */
export interface NodeTrace {
  /** Blueprint 节点 ID */
  nodeId: string;
  /** 稳定 key（如果已分配） */
  nodeKey?: string;
  /** 节点类型 (trigger / reply / if / llm / tool ...) */
  nodeType: string;
  /** 节点显示名称 */
  nodeName?: string;
  /** 执行状态 */
  status: NodeTraceStatus;
  /** 解析后的输入端口摘要（已脱敏） */
  input?: Record<string, string>;
  /** 解析后的输出端口摘要（已脱敏） */
  output?: Record<string, string>;
  /** if 节点的分支结果: 'true' | 'false' */
  branch?: string;
  /** 错误信息（status=error 时） */
  error?: string;
  /** 开始时间 (epoch ms) */
  startTime?: number;
  /** 结束时间 (epoch ms) */
  endTime?: number;
  /** 耗时 (ms) */
  durationMs?: number;
}

/** 整次 test-run 的状态 */
export type RunTraceStatus = 'running' | 'completed' | 'error' | 'cancelled';

/** 完整的 test-run trace */
export interface RunTrace {
  /** 唯一运行 ID */
  runId: string;
  /** 运行状态 */
  status: RunTraceStatus;
  /** 各节点的 trace */
  nodes: NodeTrace[];
  /** 开始时间 (epoch ms) */
  startedAt: number;
  /** 结束时间 (epoch ms) */
  completedAt?: number;
  /** 总耗时 (ms) */
  durationMs?: number;
  /** Bot 最终回复 */
  reply?: string;
  /** 原始输入消息 */
  input: string;
  /** 发送者名称 */
  senderName?: string;
  /** 是否等待下一步（step mode） */
  waitingForStep?: boolean;
}

/** 副作用策略 */
export type SideEffectPolicy = 'mock' | 'sandbox';

/** Debug run 请求选项 */
export interface DebugRunRequest {
  /** 模拟消息内容 */
  message: string;
  /** 工作流文档（草稿或已发布） */
  document: unknown;
  /** 副作用策略，默认 'mock' */
  sideEffects?: SideEffectPolicy;
  /** 单步模式 */
  stepMode?: boolean;
  /** 发送者名称 */
  senderName?: string;
  /** 续接已有会话 */
  sessionId?: string;
  /** 运行时 secret（key→value），调试模式可为空 */
  secrets?: Record<string, string>;
}

/** Debug step 请求 */
export interface DebugStepRequest {
  sessionId: string;
}

/** Debug cancel 请求 */
export interface DebugCancelRequest {
  sessionId: string;
}

/** Debug reset 请求 */
export interface DebugResetRequest {
  sessionId: string;
}
