/**
 * Trace Collector
 *
 * 订阅 XState actor 的 snapshot 变化，构建与 DebugRunner 一致的 RunTrace。
 * 生产路径（WorkflowRuntime）和调试路径（DebugRunner）共享相同的 NodeTrace
 * 字段定义和 sanitizePorts 脱敏规则。
 */

import type { RunTrace, NodeTrace, RunTraceStatus } from '@purrchat/workflow-types';
import type { Blueprint, ExecutionContext } from './types.js';
import { sanitizePorts } from './sanitize.js';

interface TraceCollectorOptions {
  runId: string;
  blueprint: Blueprint;
  input: string;
  senderName?: string;
}

export class TraceCollector {
  private runId: string;
  private blueprint: Blueprint;
  private input: string;
  private senderName?: string;
  private startedAt: number;
  private traces: Map<string, NodeTrace>;
  private prevValue: string | null = null;
  private prevTimestamp: number | null = null;
  private unsubscribe?: () => void;

  constructor(opts: TraceCollectorOptions) {
    this.runId = opts.runId;
    this.blueprint = opts.blueprint;
    this.input = opts.input;
    this.senderName = opts.senderName;
    this.startedAt = Date.now();
    this.traces = new Map();

    for (const node of opts.blueprint.nodes) {
      this.traces.set(node.id, {
        nodeId: node.id,
        nodeKey: node.key,
        nodeType: node.type,
        nodeName: node.name,
        status: 'pending',
      });
    }
  }

  /**
   * 订阅 actor snapshot 变化。
   * 每次 snapshot 更新时检查当前状态值（对应节点 ID），
   * 标记前一个节点完成、当前节点开始运行。
   */
  attach(
    actor: { subscribe: (cb: (snapshot: any) => void) => { unsubscribe: () => void }; getSnapshot: () => any },
  ): void {
    const handleChange = (snapshot: any) => {
      const value = snapshot.value;
      const context = snapshot.context as ExecutionContext | undefined;
      const now = Date.now();

      // 前一个节点完成
      if (this.prevValue !== null && this.prevValue !== value) {
        const prevTrace = this.traces.get(this.prevValue);
        if (prevTrace && prevTrace.status === 'running') {
          const output = context?.nodeOutputs?.[this.prevValue];
          if (output) {
            prevTrace.status = 'success';
            prevTrace.endTime = now;
            prevTrace.durationMs = this.prevTimestamp ? now - this.prevTimestamp : undefined;
            prevTrace.output = sanitizePorts(output);
          } else if (snapshot.status === 'done' || snapshot.matches?.('__error')) {
            // actor 结束但无输出 — 可能是错误
            prevTrace.status = 'error';
            prevTrace.endTime = now;
            prevTrace.durationMs = this.prevTimestamp ? now - this.prevTimestamp : undefined;
          }
        }
      }

      // 当前节点开始运行
      if (typeof value === 'string' && value !== this.prevValue) {
        const trace = this.traces.get(value);
        if (trace && trace.status === 'pending') {
          trace.status = 'running';
          trace.startTime = now;

          // 尝试记录输入（从 context.nodeOutputs 解析连接输入）
          if (context?.nodeOutputs) {
            const inputPorts = this.resolveInputPorts(value, context);
            if (Object.keys(inputPorts).length > 0) {
              trace.input = sanitizePorts(inputPorts);
            }
          }
        }
      }

      this.prevValue = typeof value === 'string' ? value : this.prevValue;
      this.prevTimestamp = now;
    };

    // 立即处理当前 snapshot
    handleChange(actor.getSnapshot());

    // 订阅后续变化
    const sub = actor.subscribe(handleChange);
    this.unsubscribe = sub.unsubscribe.bind(sub);
  }

  detach(): void {
    if (this.unsubscribe) {
      this.unsubscribe();
      this.unsubscribe = undefined;
    }
  }

  buildRunTrace(context: ExecutionContext, status: RunTraceStatus, reply?: string): RunTrace {
    const now = Date.now();

    // 标记仍在 pending 的节点为 skip
    for (const trace of this.traces.values()) {
      if (trace.status === 'pending') {
        trace.status = 'skip';
      }
      // 标记仍在 running 的节点为 success 或 error
      if (trace.status === 'running') {
        const hasOutput = context.nodeOutputs?.[trace.nodeId] !== undefined;
        trace.status = hasOutput ? 'success' : 'error';
        trace.endTime = now;
      }
    }

    const isDone = status === 'completed' || status === 'error' || status === 'cancelled';

    return {
      runId: this.runId,
      status,
      nodes: Array.from(this.traces.values()),
      startedAt: this.startedAt,
      completedAt: isDone ? now : undefined,
      durationMs: isDone ? now - this.startedAt : undefined,
      reply: reply || undefined,
      input: this.input,
      senderName: this.senderName,
    };
  }

  private resolveInputPorts(
    nodeId: string,
    context: ExecutionContext,
  ): Record<string, string> {
    const result: Record<string, string> = {};
    for (const conn of this.blueprint.connections) {
      if (conn.targetNodeId === nodeId) {
        const val = context.nodeOutputs[conn.sourceNodeId]?.[conn.sourcePortId];
        if (val !== undefined) {
          result[conn.targetPortId] = val;
        }
      }
    }
    return result;
  }
}
