import { createActor, type ActorRefFrom, type AnyStateMachine, waitFor } from 'xstate';
import type { Blueprint, ExecutionContext } from './types.js';
import type { Compiler } from './compiler.js';

export interface ExecuteOptions {
  rawInput: string;
  username?: string;
  contextBuffer?: Array<{ role: string; content: string }>;
  variables?: Record<string, string>;
}

export interface SessionState {
  sessionId: string;
  blueprint: Blueprint;
  actor: ActorRefFrom<AnyStateMachine>;
  round: number;
  startedAt: Date;
}

export class WorkflowRuntime {
  private sessions = new Map<string, SessionState>();
  private compiler: Compiler;

  constructor(compiler: Compiler) {
    this.compiler = compiler;
  }

  /**
   * 一次性执行（无状态，用于简单机制）
   */
  async execute(blueprint: Blueprint, options: ExecuteOptions): Promise<string> {
    const machine = this.compiler.compile(blueprint);

    const actor = createActor(machine, {
      input: {
        nodeOutputs: {},
        variables: {
          ...options.variables,
          username: options.username || '',
          time: new Date().toLocaleTimeString('zh-CN', { hour12: false }),
          __rawInput__: options.rawInput,
        },
        eventOutputs: {},
        contextBuffer: options.contextBuffer || [],
        finalReply: '',
        nameResolver: {},
      },
    });

    actor.start();

    const snapshot = await waitFor(actor, (state) => {
      return state.status === 'done' || state.matches('__error');
    }, { timeout: 30000 });

    const context = snapshot.context as unknown as ExecutionContext;
    actor.stop();

    if (snapshot.matches('__error')) {
      throw new Error('Workflow execution failed');
    }

    return context.finalReply || '...';
  }

  /**
   * 创建持久化会话（多轮对话）
   */
  createSession(
    sessionId: string,
    blueprint: Blueprint,
    options: Omit<ExecuteOptions, 'rawInput'> = {},
  ): void {
    // 如果已存在，先销毁
    if (this.sessions.has(sessionId)) {
      this.destroySession(sessionId);
    }

    const machine = this.compiler.compile(blueprint);
    const actor = createActor(machine, {
      input: {
        nodeOutputs: {},
        variables: {
          ...options.variables,
          username: options.username || '',
          time: new Date().toLocaleTimeString('zh-CN', { hour12: false }),
          __rawInput__: '',
        },
        eventOutputs: {},
        contextBuffer: options.contextBuffer || [],
        finalReply: '',
        nameResolver: {},
      },
    });

    actor.start();

    this.sessions.set(sessionId, {
      sessionId,
      blueprint,
      actor,
      round: 0,
      startedAt: new Date(),
    });
  }

  /**
   * 向会话发送消息
   */
  async sendMessage(sessionId: string, input: string): Promise<string> {
    const session = this.sessions.get(sessionId);
    if (!session) throw new Error(`Session ${sessionId} not found`);

    session.round++;

    // 发送 USER_MESSAGE 事件
    session.actor.send({ type: 'USER_MESSAGE', input });

    // 等待状态稳定
    const snapshot = await waitFor(session.actor, (state) => {
      return state.status === 'done' || state.matches('__error');
    }, { timeout: 30000 });

    if (snapshot.matches('__error')) {
      throw new Error('Workflow execution failed');
    }

    return snapshot.context.finalReply || '...';
  }

  /**
   * 销毁会话
   */
  destroySession(sessionId: string): void {
    const session = this.sessions.get(sessionId);
    if (session) {
      session.actor.stop();
      this.sessions.delete(sessionId);
    }
  }

  /**
   * 获取会话快照（用于持久化）
   */
  getSnapshot(sessionId: string): any {
    const session = this.sessions.get(sessionId);
    if (!session) return null;
    return session.actor.getSnapshot();
  }

  /**
   * 检查会话是否存在
   */
  hasSession(sessionId: string): boolean {
    return this.sessions.has(sessionId);
  }

  /**
   * 获取会话状态
   */
  getSessionState(sessionId: string): SessionState | undefined {
    return this.sessions.get(sessionId);
  }
}
