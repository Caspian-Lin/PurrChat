import { describe, it, expect } from 'vitest';
import {
  Compiler,
  WorkflowRuntime,
  NodeRegistry,
  allNodes,
  deriveCapabilities,
  type Blueprint,
  type BlueprintNode,
  type BlueprintConnection,
} from '../index.js';
import { getDefaultPorts, type EventType } from '@purrchat/workflow-types';
import { ExecutionStatus } from '../types.js';

// ─── 测试辅助 ────────────────────────────────────────────────

function makeRuntime(): { runtime: WorkflowRuntime; compiler: Compiler } {
  const registry = new NodeRegistry();
  registry.registerAll(allNodes);
  const compiler = new Compiler(registry);
  return { runtime: new WorkflowRuntime(compiler), compiler };
}

function node(id: string, type: EventType, name: string, config: Record<string, any> = {}): BlueprintNode {
  return { id, type, name, config, ports: getDefaultPorts(type) };
}

function conn(id: string, s: string, sp: string, t: string, tp: string): BlueprintConnection {
  return { id, sourceNodeId: s, sourcePortId: sp, targetNodeId: t, targetPortId: tp };
}

function bp(nodes: BlueprintNode[], connections: BlueprintConnection[]): Blueprint {
  return { nodes, connections, endConditions: [] };
}

// ─── AC: capability 推导 ─────────────────────────────────────

describe('deriveCapabilities', () => {
  it('纯本地工作流(trigger→reply)推导为 read_trigger + send', () => {
    const blueprint = bp(
      [node('t', 'trigger', '触发'), node('r', 'reply', '回复'), node('e', 'end', '结束')],
      [conn('c1', 't', 'out_exec', 'r', 'in_exec'), conn('c2', 'r', 'out_exec', 'e', 'in_exec')],
    );
    const caps = deriveCapabilities(blueprint).sort();
    expect(caps).toEqual(['messages:read_trigger', 'messages:send']);
    // 纯本地 Bot 不产生 network:external
    expect(caps).not.toContain('network:external');
  });

  it('含 llm 节点推导出 network:external + read_history', () => {
    const blueprint = bp(
      [
        node('t', 'trigger', '触发'),
        node('l', 'llm', 'LLM', { api_url: 'http://x', model: 'm' }),
        node('r', 'reply', '回复'),
        node('e', 'end', '结束'),
      ],
      [
        conn('c1', 't', 'out_exec', 'l', 'in_exec'),
        conn('c2', 'l', 'out_exec', 'r', 'in_exec'),
        conn('c3', 'r', 'out_exec', 'e', 'in_exec'),
      ],
    );
    const caps = deriveCapabilities(blueprint).sort();
    expect(caps).toContain('network:external');
    expect(caps).toContain('messages:read_history');
    expect(caps).toContain('messages:read_trigger');
    expect(caps).toContain('messages:send');
  });

  it('含 tool/dify/n8n 节点推导出 network:external', () => {
    for (const t of ['tool', 'dify', 'n8n'] as EventType[]) {
      const blueprint = bp(
        [node('t', 'trigger', '触发'), node('x', t, t, {}), node('e', 'end', '结束')],
        [conn('c1', 't', 'out_exec', 'x', 'in_exec'), conn('c2', 'x', 'out_exec', 'e', 'in_exec')],
      );
      expect(deriveCapabilities(blueprint)).toContain('network:external');
    }
  });

  it('含 history 节点推导出 read_history', () => {
    const blueprint = bp(
      [node('t', 'trigger', '触发'), node('h', 'history', '历史'), node('e', 'end', '结束')],
      [conn('c1', 't', 'out_exec', 'h', 'in_exec'), conn('c2', 'h', 'out_exec', 'e', 'in_exec')],
    );
    expect(deriveCapabilities(blueprint)).toContain('messages:read_history');
  });

  it('控制流节点(if/loop/switch/merge/wait)不产生 capability', () => {
    const blueprint = bp(
      [node('t', 'trigger', '触发'), node('i', 'if', '条件'), node('e', 'end', '结束')],
      [conn('c1', 't', 'out_exec', 'i', 'in_exec'), conn('c2', 'i', 'out_true', 'e', 'in_exec')],
    );
    const caps = deriveCapabilities(blueprint);
    expect(caps).toEqual(['messages:read_trigger']);
  });
});

// ─── AC: 运行时强制校验 ──────────────────────────────────────

describe('运行时 capability 强制校验', () => {
  it('granted 包含全部 required 时正常执行纯本地工作流', async () => {
    const { runtime } = makeRuntime();
    const blueprint = bp(
      [node('t', 'trigger', '触发'), node('r', 'reply', '回复', { template: 'ok' }), node('e', 'end', '结束')],
      [conn('c1', 't', 'out_exec', 'r', 'in_exec'), conn('c2', 'r', 'out_exec', 'e', 'in_exec')],
    );
    const result = await runtime.execute(blueprint, {
      rawInput: 'hi',
      grantedCapabilities: ['messages:read_trigger', 'messages:send'],
      timeoutMs: 3000,
    });
    expect(result.status).toBe(ExecutionStatus.Done);
    expect(result.reply).toBe('ok');
  });

  it('granted 缺失 messages:send 时 reply 节点拒绝执行', async () => {
    const { runtime } = makeRuntime();
    const blueprint = bp(
      [node('t', 'trigger', '触发'), node('r', 'reply', '回复', { template: 'ok' }), node('e', 'end', '结束')],
      [conn('c1', 't', 'out_exec', 'r', 'in_exec'), conn('c2', 'r', 'out_exec', 'e', 'in_exec')],
    );
    await expect(
      runtime.execute(blueprint, {
        rawInput: 'hi',
        // 只授予 read_trigger，缺失 send
        grantedCapabilities: ['messages:read_trigger'],
        timeoutMs: 3000,
      }),
    ).rejects.toThrow('Capability denied');
  });

  it('granted 缺失 network:external 时 llm 节点拒绝执行(不发起网络请求)', async () => {
    const { runtime } = makeRuntime();
    const blueprint = bp(
      [
        node('t', 'trigger', '触发'),
        node('l', 'llm', 'LLM', { api_url: 'http://should-not-be-called.invalid', model: 'm' }),
        node('r', 'reply', '回复'),
        node('e', 'end', '结束'),
      ],
      [
        conn('c1', 't', 'out_exec', 'l', 'in_exec'),
        conn('c2', 'l', 'out_exec', 'r', 'in_exec'),
        conn('c3', 'r', 'out_exec', 'e', 'in_exec'),
      ],
    );
    await expect(
      runtime.execute(blueprint, {
        rawInput: 'hi',
        // 缺 network:external + read_history
        grantedCapabilities: ['messages:read_trigger', 'messages:send'],
        timeoutMs: 3000,
      }),
    ).rejects.toThrow('Capability denied');
  });

  it('granted 缩减 read_history 时 history 节点拒绝执行', async () => {
    const { runtime } = makeRuntime();
    const blueprint = bp(
      [node('t', 'trigger', '触发'), node('h', 'history', '历史'), node('e', 'end', '结束')],
      [conn('c1', 't', 'out_exec', 'h', 'in_exec'), conn('c2', 'h', 'out_exec', 'e', 'in_exec')],
    );
    await expect(
      runtime.execute(blueprint, {
        rawInput: 'hi',
        grantedCapabilities: ['messages:read_trigger'], // 缺 read_history
        timeoutMs: 3000,
      }),
    ).rejects.toThrow('Capability denied');
  });
});
