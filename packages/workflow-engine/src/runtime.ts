import { createActor, type ActorRefFrom, type AnyStateMachine, type Snapshot, waitFor } from 'xstate';
import type { RunTrace, RunTraceStatus } from '@purrchat/workflow-types';
import type { Blueprint, ExecutionContext, ActorInput, UserMessageEvent } from './types.js';
import { ExecutionStatus, type ExecuteResult } from './types.js';
import type { Compiler } from './compiler.js';
import { TraceCollector } from './trace-collector.js';

/** 终态状态名，需与 compiler 保持一致 */
const ERROR_STATE = '__error';

/**
 * 运行时快照类型。AnyStateMachine 的 Snapshot 是联合类型，active 分支
 * 在类型上没有 context/matches，这里统一为可访问形式以便读取运行时数据。
 */
type RuntimeSnapshot = Snapshot<AnyStateMachine> & {
  context?: unknown;
  value?: unknown;
  matches?: (state: string) => boolean;
};

/** 解析后的结束条件 */
interface ResolvedEndConditions {
  maxRounds?: number;
  sessionTimeoutMs?: number;
  messageMatchPatterns: RegExp[];
}

export interface ExecuteOptions {
  rawInput: string;
  senderName?: string;
  senderId?: string;
  conversationId?: string;
  time?: string;
  contextBuffer?: Array<{ role: string; content: string }>;
  variables?: Record<string, string>;
  /** 安装时授予的 capabilities，运行时强制校验用 */
  grantedCapabilities?: string[];
  /** 运行时解密后的 secret（key→value），引用解析用 */
  secrets?: Record<string, string>;
  /** 单次执行等待超时（毫秒），默认 30000 */
  timeoutMs?: number;
  /** 调用方生成的 run_id，透传到 ExecuteResult */
  runId?: string;
}

export interface SendMessageOptions {
  senderName?: string;
  senderId?: string;
  conversationId?: string;
  time?: string;
  /** 单次发送等待超时（毫秒），默认 30000 */
  timeoutMs?: number;
  /** 调用方生成的 run_id，透传到 ExecuteResult */
  runId?: string;
}

export interface SessionState {
  sessionId: string;
  blueprint: Blueprint;
  actor: ActorRefFrom<AnyStateMachine>;
  round: number;
  startedAt: Date;
  waitNodeIds: Set<string>;
  endConditions: ResolvedEndConditions;
  /** 并发锁：同一会话同时只处理一条消息 */
  busy: boolean;
  /** 当前轮次的 trace collector */
  traceCollector?: TraceCollector;
}

/**
 * 工作流执行被取消/超时时抛出。
 */
export class WorkflowTimeoutError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'WorkflowTimeoutError';
  }
}

/**
 * 工作流执行出错时抛出，携带 trace 供上层持久化。
 */
export class WorkflowExecutionError extends Error {
  trace?: RunTrace;
  constructor(message: string, trace?: RunTrace) {
    super(message);
    this.name = 'WorkflowExecutionError';
    this.trace = trace;
  }
}

export class WorkflowRuntime {
  private sessions = new Map<string, SessionState>();
  private compiler: Compiler;

  constructor(compiler: Compiler) {
    this.compiler = compiler;
  }

  /**
   * 一次性执行（无状态，用于无 wait 节点的简单工作流）。
   * 内部仍通过发送 USER_MESSAGE 事件触发 trigger，保证与多轮会话语义一致。
   */
  async execute(blueprint: Blueprint, options: ExecuteOptions): Promise<ExecuteResult> {
    const machine = this.compiler.compile(blueprint);
    const waitNodeIds = this.extractWaitNodeIds(blueprint);

    const runId = options.runId ?? crypto.randomUUID();
    const traceCollector = new TraceCollector({
      runId,
      blueprint,
      input: options.rawInput,
      senderName: options.senderName,
    });

    const actor = createActor(machine, { input: this.buildActorInput(options) });
    traceCollector.attach(actor);
    actor.start();

    try {
      // 触发入口 trigger
      actor.send(this.buildUserMessageEvent(options.rawInput, options));

      const snapshot = await this.waitForSettled(actor, waitNodeIds, options.timeoutMs ?? 30000);
      const status = this.classifySnapshot(snapshot, waitNodeIds);
      const context = (snapshot as RuntimeSnapshot).context as ExecutionContext;

      if (status === ExecutionStatus.Error) {
        const trace = traceCollector.buildRunTrace(context, 'error', context.finalReply);
        throw new WorkflowExecutionError(context.lastError || 'Workflow execution failed', trace);
      }
      if (status === ExecutionStatus.Waiting) {
        // 一次性执行不允许暂停在 wait 节点
        throw new Error('Workflow paused at wait node; use a session-based execution instead');
      }

      const trace = traceCollector.buildRunTrace(context, 'completed', context.finalReply);
      return {
        reply: context.finalReply ?? '',
        status: ExecutionStatus.Done,
        sessionActive: false,
        round: 1,
        runId,
        trace,
      };
    } finally {
      traceCollector.detach();
      actor.stop();
    }
  }

  /**
   * 创建持久化会话（多轮对话）。
   * Actor 启动后停在 trigger 状态，等待第一条消息。
   */
  createSession(
    sessionId: string,
    blueprint: Blueprint,
    options: Omit<ExecuteOptions, 'rawInput' | 'timeoutMs'> = {},
  ): void {
    // 已存在则先销毁
    if (this.sessions.has(sessionId)) {
      this.destroySession(sessionId);
    }

    const machine = this.compiler.compile(blueprint);
    const actor = createActor(machine, {
      input: this.buildActorInput({ rawInput: '', ...options }),
    });
    actor.start();

    this.sessions.set(sessionId, {
      sessionId,
      blueprint,
      actor,
      round: 0,
      startedAt: new Date(),
      waitNodeIds: this.extractWaitNodeIds(blueprint),
      endConditions: this.resolveEndConditions(blueprint),
      busy: false,
    });
  }

  /**
   * 向会话发送消息，驱动工作流推进。
   * - 若工作流到达终态（done/error），会话自动销毁。
   * - 若工作流暂停在 wait 节点，返回 waiting 状态。
   */
  async sendMessage(sessionId: string, input: string, options: SendMessageOptions = {}): Promise<ExecuteResult> {
    const session = this.sessions.get(sessionId);
    if (!session) throw new Error(`Session ${sessionId} not found`);

    if (session.busy) {
      throw new Error(`Session ${sessionId} is busy processing another message`);
    }
    session.busy = true;

    const runId = options.runId ?? crypto.randomUUID();
    const traceCollector = new TraceCollector({
      runId,
      blueprint: session.blueprint,
      input,
      senderName: options.senderName,
    });
    traceCollector.attach(session.actor);
    session.traceCollector = traceCollector;

    try {
      // 会话级超时检查
      if (this.isSessionTimedOut(session)) {
        this.destroySession(sessionId);
        return {
          reply: '',
          status: ExecutionStatus.Done,
          sessionActive: false,
          round: session.round,
          runId,
        };
      }

      session.round++;

      const event = this.buildUserMessageEvent(input, options);
      session.actor.send(event);

      const snapshot = await this.waitForSettled(
        session.actor,
        session.waitNodeIds,
        options.timeoutMs ?? 30000,
      );
      const status = this.classifySnapshot(snapshot, session.waitNodeIds);
      const context = (snapshot as RuntimeSnapshot).context as ExecutionContext;

      // 判断是否应在本次回复后结束会话
      const reachedMaxRounds =
        session.endConditions.maxRounds !== undefined &&
        session.round >= session.endConditions.maxRounds;
      const matchedEndPattern = this.matchesEndPattern(input, session.endConditions.messageMatchPatterns);

      const shouldEnd =
        status === ExecutionStatus.Done ||
        status === ExecutionStatus.Error ||
        reachedMaxRounds ||
        matchedEndPattern;

      if (shouldEnd) {
        this.destroySession(sessionId);
      }

      if (status === ExecutionStatus.Error) {
        // 不伪装占位回复；返回空 reply，让上层决定是否记录/通知
        const trace = traceCollector.buildRunTrace(context, 'error', '');
        return {
          reply: '',
          status: ExecutionStatus.Error,
          sessionActive: false,
          round: session.round,
          runId,
          trace,
        };
      }

      const traceStatus: RunTraceStatus = shouldEnd ? 'completed' : 'running';
      const trace = traceCollector.buildRunTrace(context, traceStatus, context.finalReply);
      return {
        reply: context.finalReply ?? '',
        status: shouldEnd ? ExecutionStatus.Done : status,
        sessionActive: !shouldEnd && status === ExecutionStatus.Waiting,
        round: session.round,
        runId,
        trace,
      };
    } finally {
      // 会话可能已被销毁
      const s = this.sessions.get(sessionId);
      if (s) {
        s.busy = false;
        s.traceCollector?.detach();
        s.traceCollector = undefined;
      }
    }
  }

  /** 销毁会话 */
  destroySession(sessionId: string): void {
    const session = this.sessions.get(sessionId);
    if (session) {
      session.traceCollector?.detach();
      try {
        session.actor.stop();
      } catch {
        // 已停止则忽略
      }
      this.sessions.delete(sessionId);
    }
  }

  /** 获取会话快照（用于持久化/调试） */
  getSnapshot(sessionId: string): RuntimeSnapshot | null {
    const session = this.sessions.get(sessionId);
    if (!session) return null;
    return session.actor.getSnapshot() as RuntimeSnapshot;
  }

  /** 获取当前轮次的 RunTrace（如果 traceCollector 存在） */
  getSessionTrace(sessionId: string): RunTrace | null {
    const session = this.sessions.get(sessionId);
    if (!session?.traceCollector) return null;
    const snapshot = session.actor.getSnapshot() as RuntimeSnapshot;
    const context = (snapshot?.context as ExecutionContext) ?? {} as ExecutionContext;
    return session.traceCollector.buildRunTrace(context, 'completed', context.finalReply);
  }

  hasSession(sessionId: string): boolean {
    return this.sessions.has(sessionId);
  }

  getSessionState(sessionId: string): SessionState | undefined {
    return this.sessions.get(sessionId);
  }

  // ─── 内部工具 ──────────────────────────────────────────────

  private buildActorInput(options: ExecuteOptions): ActorInput {
    return {
      rawInput: options.rawInput ?? '',
      senderName: options.senderName ?? '',
      senderId: options.senderId ?? '',
      conversationId: options.conversationId ?? '',
      time: options.time ?? new Date().toLocaleTimeString('zh-CN', { hour12: false }),
      contextBuffer: options.contextBuffer ?? [],
      variables: options.variables ?? {},
      grantedCapabilities: options.grantedCapabilities,
      secrets: options.secrets,
    };
  }

  private buildUserMessageEvent(input: string, options: SendMessageOptions): UserMessageEvent {
    return {
      type: 'USER_MESSAGE',
      input,
      senderName: options.senderName,
      senderId: options.senderId,
      conversationId: options.conversationId,
      time: options.time,
    };
  }

  private extractWaitNodeIds(blueprint: Blueprint): Set<string> {
    return new Set(blueprint.nodes.filter((n) => n.type === 'wait').map((n) => n.id));
  }

  private resolveEndConditions(blueprint: Blueprint): ResolvedEndConditions {
    const ec = blueprint.endConditions ?? [];
    const result: ResolvedEndConditions = { messageMatchPatterns: [] };

    const maxRounds = ec.find((e) => e.type === 'max_rounds');
    if (maxRounds?.value !== undefined) result.maxRounds = maxRounds.value;

    const timeout = ec.find((e) => e.type === 'timeout');
    if (timeout?.value !== undefined) result.sessionTimeoutMs = timeout.value * 1000;

    for (const e of ec) {
      if (e.type === 'message_match' && e.pattern) {
        try {
          result.messageMatchPatterns.push(new RegExp(e.pattern));
        } catch {
          // 非法正则忽略
        }
      }
    }

    return result;
  }

  private isSessionTimedOut(session: SessionState): boolean {
    if (session.endConditions.sessionTimeoutMs === undefined) return false;
    return Date.now() - session.startedAt.getTime() > session.endConditions.sessionTimeoutMs;
  }

  private matchesEndPattern(input: string, patterns: RegExp[]): boolean {
    return patterns.some((p) => p.test(input));
  }

  /**
   * 判断 snapshot 是否已稳定（终态或暂停在 wait 节点）。
   * 注意：__error 是 final 状态，snapshot.status 也是 'done'，
   * 因此必须先检查 __error，再判断 done。
   */
  private classifySnapshot(
    snapshot: RuntimeSnapshot,
    waitNodeIds: Set<string>,
  ): ExecutionStatus {
    if (snapshot.matches?.(ERROR_STATE)) return ExecutionStatus.Error;
    if (snapshot.status === 'done') return ExecutionStatus.Done;
    const value = snapshot.value;
    if (typeof value === 'string' && waitNodeIds.has(value)) {
      return ExecutionStatus.Waiting;
    }
    // 既未结束也不在已知 wait 节点，视为仍在处理
    return ExecutionStatus.Waiting;
  }

  private isSettled(
    snapshot: RuntimeSnapshot,
    waitNodeIds: Set<string>,
  ): boolean {
    if (snapshot.status === 'done') return true;
    if (snapshot.matches?.(ERROR_STATE)) return true;
    const value = snapshot.value;
    if (typeof value === 'string' && waitNodeIds.has(value)) return true;
    return false;
  }

  private waitForSettled(
    actor: ActorRefFrom<AnyStateMachine>,
    waitNodeIds: Set<string>,
    timeoutMs: number,
  ): Promise<RuntimeSnapshot> {
    return waitFor(
      actor,
      (s) => this.isSettled(s as RuntimeSnapshot, waitNodeIds),
      { timeout: timeoutMs },
    ) as Promise<RuntimeSnapshot>;
  }
}
