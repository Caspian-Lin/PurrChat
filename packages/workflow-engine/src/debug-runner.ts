/**
 * Debug Runner
 *
 * 图遍历式工作流调试执行器。与生产 XState runtime 共用相同的节点 execute()
 * 函数和 resolveTemplate，但不依赖 XState 状态机——从而获得精确的逐步执行、
 * trace 收集和副作用 mock 能力。
 *
 * 支持模式：
 * - full run: 一次性执行所有可达节点
 * - step mode: 每次只执行一个节点，暂停等待继续
 * - mock side effects: tool/dify/n8n/llm 节点返回预设 mock 数据
 * - cancel: 中途取消执行
 */

import type {
  WorkflowDocument,
  RunTrace,
  NodeTrace,
  NodeTraceStatus,
  SideEffectPolicy,
} from '@purrchat/workflow-types';
import type { NodeRegistry } from './registry.js';
import type {
  Blueprint,
  BlueprintNode,
  BlueprintConnection,
  ExecutionContext,
  NodeInput,
  NodeContext,
  NodeOutput,
} from './types.js';
import { toBlueprint } from './validator.js';
import { resolveSecrets } from './secrets.js';
import { resolveTemplate } from './resolver.js';
import { sanitizePorts } from './sanitize.js';
import { resolveControlFlowRoute } from './control-flow.js';

/** 外部副作用节点类型 — mock 模式下返回预设数据 */
const EXTERNAL_NODE_TYPES = new Set(['tool', 'dify', 'n8n', 'llm']);

/** mock 返回值映射 */
const MOCK_OUTPUTS: Record<string, NodeOutput> = {
  tool: {
    ports: {
      out_output: JSON.stringify({ mock: true, message: '[mocked HTTP response]' }),
      out_exec: 'true',
    },
  },
  dify: {
    ports: {
      out_output: '[mocked Dify response]',
      out_text: '[mocked Dify response]',
      out_exec: 'true',
    },
  },
  n8n: {
    ports: {
      out_output: '[mocked n8n response]',
      out_exec: 'true',
    },
  },
  llm: {
    ports: {
      out_output: '[mocked LLM response]',
      out_exec: 'true',
    },
  },
};

/** debug 会话状态 */
interface DebugSession {
  sessionId: string;
  blueprint: Blueprint;
  context: ExecutionContext;
  /** 已完成的节点 ID 集合 */
  completed: Set<string>;
  /** 被跳过的节点 ID 集合（if 分支未走的路径） */
  skipped: Set<string>;
  /** 待执行的节点队列 */
  queue: string[];
  /** trace 收集 */
  traces: Map<string, NodeTrace>;
  /** 是否 step 模式 */
  stepMode: boolean;
  /** 副作用策略 */
  sideEffects: SideEffectPolicy;
  /** 原始输入 */
  input: string;
  senderName: string;
  startedAt: number;
  /** 是否已取消 */
  cancelled: boolean;
  /** runId */
  runId: string;
}

let runIdCounter = 0;

export class DebugRunner {
  private sessions = new Map<string, DebugSession>();

  constructor(private registry: NodeRegistry) {}

  /**
   * 启动一次调试运行。
   * - stepMode=false: 执行全部可达节点，返回完整 trace
   * - stepMode=true: 执行到第一个非 trigger 节点后暂停
   */
  async run(options: {
    document: WorkflowDocument;
    message: string;
    sideEffects?: SideEffectPolicy;
    stepMode?: boolean;
    senderName?: string;
    sessionId?: string;
    secrets?: Record<string, string>;
    contextBuffer?: Array<{ role: string; content: string }>;
  }): Promise<RunTrace> {
    const sideEffects = options.sideEffects ?? 'mock';
    const stepMode = options.stepMode ?? false;
    const blueprint = toBlueprint(options.document);
    const contextBuffer = options.contextBuffer ?? [];

    // 查找 trigger 节点
    const trigger = blueprint.nodes.find((n) => n.type === 'trigger');
    if (!trigger) {
      throw new Error('工作流缺少触发节点');
    }

    const sessionId = options.sessionId || `debug-${Date.now()}-${++runIdCounter}`;
    const runId = `run-${sessionId}-${Date.now()}`;
    const startedAt = Date.now();

    // 构建初始上下文
    const nameResolver = this.buildNameResolver(blueprint);
    const nodeKeyMap = this.buildNodeKeyMap(blueprint);
    const context: ExecutionContext = {
      nodeOutputs: {},
      variables: {
        __rawInput__: options.message,
        username: options.senderName ?? '',
        sender_id: '',
        conversation_id: '',
        time: new Date().toLocaleTimeString('zh-CN', { hour12: false }),
      },
      eventOutputs: {},
      contextBuffer,
      finalReply: '',
      nameResolver,
      nodeKeyMap,
      senderId: '',
      senderName: options.senderName ?? '',
      conversationId: '',
      rawInput: options.message,
      history: contextBuffer,
      session: {},
      secrets: options.secrets ?? {},
    };

    // 初始化所有节点 trace
    const traces = new Map<string, NodeTrace>();
    for (const node of blueprint.nodes) {
      traces.set(node.id, {
        nodeId: node.id,
        nodeKey: node.key,
        nodeType: node.type,
        nodeName: node.name,
        status: 'pending',
      });
    }

    const session: DebugSession = {
      sessionId,
      blueprint,
      context,
      completed: new Set<string>(),
      skipped: new Set<string>(),
      queue: [],
      traces,
      stepMode,
      sideEffects,
      input: options.message,
      senderName: options.senderName ?? '',
      startedAt,
      cancelled: false,
      runId,
    };

    this.sessions.set(sessionId, session);

    // 执行 trigger
    await this.executeNode(session, trigger);

    // 收集 trigger 后继
    this.enqueueNext(session, trigger);

    if (stepMode) {
      // step 模式：不自动继续，等待 step() 调用
      return this.buildRunTrace(session);
    }

    // full run: 执行队列中所有节点
    return this.drainQueue(session);
  }

  /**
   * 单步执行：执行队列中的下一个节点。
   * 只在 stepMode 会话中使用。
   */
  async step(sessionId: string): Promise<RunTrace> {
    const session = this.sessions.get(sessionId);
    if (!session) throw new Error(`Debug session ${sessionId} not found`);

    if (session.queue.length === 0) {
      return this.buildRunTrace(session);
    }

    const nodeId = session.queue.shift()!;
    const node = session.blueprint.nodes.find((n) => n.id === nodeId);
    if (!node) return this.buildRunTrace(session);

    await this.executeNode(session, node);
    this.enqueueNext(session, node);

    return this.buildRunTrace(session);
  }

  /** 取消正在进行的运行 */
  cancel(sessionId: string): void {
    const session = this.sessions.get(sessionId);
    if (session) {
      session.cancelled = true;
    }
  }

  /** 重置会话 */
  reset(sessionId: string): void {
    this.sessions.delete(sessionId);
  }

  /** 获取会话的当前 trace */
  getTrace(sessionId: string): RunTrace | null {
    const session = this.sessions.get(sessionId);
    if (!session) return null;
    return this.buildRunTrace(session);
  }

  // ─── 内部方法 ──────────────────────────────────────────────

  /**
   * 执行单个节点，收集 trace。
   */
  private async executeNode(session: DebugSession, node: BlueprintNode): Promise<void> {
    if (session.cancelled) return;
    if (session.skipped.has(node.id)) return;

    const trace = session.traces.get(node.id);
    if (!trace) return;

    const startTime = Date.now();
    trace.status = 'running';
    trace.startTime = startTime;

    try {
      // 解析输入端口
      const ports = this.resolveNodeInputs(node.id, session.blueprint.connections, session.context);
      trace.input = sanitizePorts(ports);

      let output: NodeOutput;

      if (session.sideEffects === 'mock' && EXTERNAL_NODE_TYPES.has(node.type)) {
        // mock 模式：外部副作用节点返回预设数据
        output = MOCK_OUTPUTS[node.type] ?? { ports: { out_exec: 'true' } };
        // 对 mock 输出也做变量解析
        const mockPorts: Record<string, string> = {};
        for (const [k, v] of Object.entries(output.ports)) {
          mockPorts[k] = resolveTemplate(v, this.buildNodeContext(session, node));
        }
        output = { ports: mockPorts };
      } else {
        // 真实执行
        const def = this.registry.get(node.type);
        if (!def) throw new Error(`未知节点类型: ${node.type}`);

        // 解析 secrets 引用
        const resolvedConfig = resolveSecrets(node.config, session.context.secrets);

        const nodeInput: NodeInput = {
          ports,
          rawInput: session.context.rawInput,
        };

        output = await def.execute(nodeInput, resolvedConfig as Record<string, any>, this.buildNodeContext(session, node));
      }

      const route = resolveControlFlowRoute(node, output.ports, session.context.session);
      if (route) {
        session.context.session = route.session;
        output = {
          ports: { ...output.ports, __branch__: route.portId, [route.portId]: 'true' },
        };
      }

      const endTime = Date.now();
      trace.endTime = endTime;
      trace.durationMs = endTime - startTime;
      trace.output = sanitizePorts(output.ports);
      trace.status = 'success';

      if (output.ports['__branch__']) {
        trace.branch = output.ports['__branch__'];
      }

      // 存储输出到上下文
      session.context.nodeOutputs[node.id] = output.ports;

      // 提取 reply
      const reply = output.ports['__reply__'];
      if (reply) {
        session.context.finalReply = reply;
      }

      // 提取 event output
      const eventOutput = output.ports['out_output'];
      if (eventOutput) {
        session.context.eventOutputs[node.id] = eventOutput;
      }

      // 更新 rawInput for trigger output ports
      if (node.type === 'trigger') {
        session.context.nodeOutputs[node.id] = {
          out_input: session.input,
          out_username: session.senderName,
          out_time: new Date().toLocaleTimeString('zh-CN', { hour12: false }),
          out_args: '',
          out_exec: 'true',
        };
      }
    } catch (err) {
      const endTime = Date.now();
      trace.endTime = endTime;
      trace.durationMs = endTime - startTime;
      trace.status = 'error';
      trace.error = err instanceof Error ? err.message : String(err);
    }

    session.completed.add(node.id);
  }

  /**
   * 将节点的后继加入执行队列。
   * 控制流节点只入队当前路由的后继；另一分支的节点在 buildRunTrace 时标记为 skip。
   */
  private enqueueNext(session: DebugSession, node: BlueprintNode): void {
    if (['if', 'switch', 'loop', 'merge'].includes(node.type)) {
      const branchPort = session.context.nodeOutputs[node.id]?.['__branch__'];

      // 只入队活跃分支的目标
      const activeConns = session.blueprint.connections.filter(
        (c) => c.sourceNodeId === node.id && c.sourcePortId === branchPort,
      );
      for (const conn of activeConns) {
        if (this.canEnqueue(session, conn.targetNodeId)) {
          session.queue.push(conn.targetNodeId);
        }
      }
    } else {
      // 非 if 节点：所有出边连接的目标都入队
      const outConns = session.blueprint.connections.filter(
        (c) => c.sourceNodeId === node.id,
      );
      for (const conn of outConns) {
        if (this.canEnqueue(session, conn.targetNodeId)) {
          session.queue.push(conn.targetNodeId);
        }
      }
    }

    // 去重队列
    session.queue = [...new Set(session.queue)];
  }

  /** 持续执行队列直到空或取消 */
  private async drainQueue(session: DebugSession): Promise<RunTrace> {
    while (session.queue.length > 0 && !session.cancelled) {
      const nodeId = session.queue.shift()!;

      const node = session.blueprint.nodes.find((n) => n.id === nodeId);
      if (!node) continue;
      if ((session.completed.has(nodeId) && node.type !== 'loop') || session.skipped.has(nodeId)) continue;

      await this.executeNode(session, node);
      this.enqueueNext(session, node);
    }

    return this.buildRunTrace(session);
  }

  private canEnqueue(session: DebugSession, nodeId: string): boolean {
    const target = session.blueprint.nodes.find((node) => node.id === nodeId);
    return !!target && !session.skipped.has(nodeId) && (target.type === 'loop' || !session.completed.has(nodeId));
  }

  // ─── 工具方法 ──────────────────────────────────────────────

  private resolveNodeInputs(
    nodeId: string,
    connections: BlueprintConnection[],
    context: ExecutionContext,
  ): Record<string, string> {
    const result: Record<string, string> = {};
    for (const conn of connections) {
      if (conn.targetNodeId === nodeId) {
        const val = context.nodeOutputs[conn.sourceNodeId]?.[conn.sourcePortId];
        if (val !== undefined) {
          result[conn.targetPortId] = val;
        }
      }
    }
    return result;
  }

  private buildNodeContext(session: DebugSession, _node: BlueprintNode): NodeContext {
    return {
      variables: session.context.variables,
      eventOutputs: session.context.eventOutputs,
      contextBuffer: session.context.contextBuffer,
      nodeOutputs: session.context.nodeOutputs,
      nameResolver: session.context.nameResolver,
      finalReply: session.context.finalReply,
      nodeKeyMap: session.context.nodeKeyMap,
      rawInput: session.context.rawInput,
      senderId: session.context.senderId,
      senderName: session.context.senderName,
      conversationId: session.context.conversationId,
      history: session.context.history,
      secrets: session.context.secrets ?? {},
      session: session.context.session,
    };
  }

  private buildNameResolver(blueprint: Blueprint): Record<string, string> {
    const resolver: Record<string, string> = {};
    for (const node of blueprint.nodes) {
      const ports = node.ports || [];
      for (const port of ports) {
        if (port.direction === 'output') {
          resolver[`${node.name}.${port.name}`] = `${node.id}:${port.id}`;
        }
      }
    }
    return resolver;
  }

  private buildNodeKeyMap(blueprint: Blueprint): Record<string, string> {
    const map: Record<string, string> = {};
    for (const node of blueprint.nodes) {
      if (node.key) {
        map[node.key] = node.id;
      }
    }
    return map;
  }

  private buildRunTrace(session: DebugSession): RunTrace {
    const isDone = session.queue.length === 0 || session.cancelled;

    const nodes = session.blueprint.nodes.map((n) => {
      const trace = session.traces.get(n.id);
      if (trace) {
        // 执行结束后：将仍在队列外且 pending 的节点标记为 skip（未走的 if 分支等）
        if (isDone && trace.status === 'pending' && !session.completed.has(n.id)) {
          trace.status = 'skip';
          session.skipped.add(n.id);
        }
        return trace;
      }
      return {
        nodeId: n.id,
        nodeType: n.type,
        nodeName: n.name,
        status: 'pending' as NodeTraceStatus,
      };
    });

    const hasError = nodes.some((n) => n.status === 'error');

    let status: RunTrace['status'];
    if (session.cancelled) {
      status = 'cancelled';
    } else if (hasError) {
      status = 'error';
    } else if (isDone) {
      status = 'completed';
    } else {
      status = 'running';
    }

    const completedAt = isDone ? Date.now() : undefined;

    return {
      runId: session.runId,
      status,
      nodes,
      startedAt: session.startedAt,
      completedAt,
      durationMs: completedAt ? completedAt - session.startedAt : undefined,
      reply: session.context.finalReply || undefined,
      input: session.input,
      senderName: session.senderName,
      waitingForStep: session.stepMode && !isDone,
    };
  }
}
