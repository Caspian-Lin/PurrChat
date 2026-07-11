import { describe, it, expect } from 'vitest';
import {
  Compiler,
  WorkflowRuntime,
  NodeRegistry,
  allNodes,
  DebugRunner,
  type Blueprint,
  type BlueprintNode,
  type BlueprintConnection,
} from '../index.js';
import { getDefaultPorts, type EventType } from '@purrchat/workflow-types';
import { createEmptyDocument, type WorkflowDocument } from '@purrchat/workflow-types';
import { ExecutionStatus } from '../types.js';

// ─── 测试辅助 ────────────────────────────────────────────────

function makeRuntime(): { runtime: WorkflowRuntime; registry: NodeRegistry } {
  const registry = new NodeRegistry();
  registry.registerAll(allNodes);
  const compiler = new Compiler(registry);
  return { runtime: new WorkflowRuntime(compiler), registry };
}

function makeRunner(): DebugRunner {
  const registry = new NodeRegistry();
  registry.registerAll(allNodes);
  return new DebugRunner(registry);
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

const SAMPLE_HISTORY = [
  { role: 'user', content: '你好' },
  { role: 'assistant', content: '你好呀' },
  { role: 'user', content: '今天天气如何' },
  { role: 'assistant', content: '晴天' },
  { role: 'system', content: '系统提示' },
];

function makeDoc(nodes: any[], connections: any[]): WorkflowDocument {
  const doc = createEmptyDocument('test-bot');
  doc.spec.nodes = nodes.map((n, i) => ({
    id: n.id,
    type: n.type,
    name: n.name ?? n.type,
    key: n.key ?? `${n.type}_${i}`,
    config: n.config ?? {},
  }));
  doc.spec.connections = connections.map((c, i) => ({
    id: c.id ?? `conn_${i}`,
    sourceNodeId: c.from?.nodeId ?? c.sourceNodeId,
    sourcePortId: c.from?.portId ?? c.sourcePortId,
    targetNodeId: c.to?.nodeId ?? c.targetNodeId,
    targetPortId: c.to?.portId ?? c.targetPortId,
  }));
  return doc;
}

// ─── AC1: 排序、截断、消息类型过滤、空历史行为 ──────────────────

describe('History 节点 — 排序、截断与过滤', () => {
  it('默认正序输出全部历史', async () => {
    const { runtime } = makeRuntime();
    const blueprint = bp(
      [node('t', 'trigger', '触发'), node('h', 'history', '历史'), node('e', 'end', '结束')],
      [conn('c1', 't', 'out_exec', 'h', 'in_exec'), conn('c2', 'h', 'out_exec', 'e', 'in_exec')],
    );

    const result = await runtime.execute(blueprint, {
      rawInput: 'hi',
      contextBuffer: SAMPLE_HISTORY,
      timeoutMs: 3000,
    });

    expect(result.status).toBe(ExecutionStatus.Done);
  });

  it('config.count 截断为最近 N 条', async () => {
    const { runtime } = makeRuntime();
    const blueprint = bp(
      [
        node('t', 'trigger', '触发'),
        node('h', 'history', '历史', { count: 2 }),
        node('r', 'reply', '回复', { template: '$h:out_history' }),
        node('e', 'end', '结束'),
      ],
      [
        conn('c1', 't', 'out_exec', 'h', 'in_exec'),
        conn('c2', 'h', 'out_exec', 'r', 'in_exec'),
        conn('c3', 'r', 'out_exec', 'e', 'in_exec'),
      ],
    );

    const result = await runtime.execute(blueprint, {
      rawInput: 'hi',
      contextBuffer: SAMPLE_HISTORY,
      timeoutMs: 3000,
    });

    expect(result.reply).toContain('[system]: 系统提示');
    expect(result.reply).toContain('[assistant]: 晴天');
    expect(result.reply).not.toContain('你好');
  });

  it('in_count 端口覆盖 config.count', async () => {
    const { runtime } = makeRuntime();
    const blueprint = bp(
      [
        node('t', 'trigger', '触发'),
        node('tpl', 'template', '数量', { template: '1' }),
        node('h', 'history', '历史'),
        node('r', 'reply', '回复', { template: '$h:out_history' }),
        node('e', 'end', '结束'),
      ],
      [
        conn('c1', 't', 'out_exec', 'tpl', 'in_exec'),
        conn('c2', 'tpl', 'out_exec', 'h', 'in_exec'),
        conn('c3', 'tpl', 'out_output', 'h', 'in_count'),
        conn('c4', 'h', 'out_exec', 'r', 'in_exec'),
        conn('c5', 'r', 'out_exec', 'e', 'in_exec'),
      ],
    );

    const result = await runtime.execute(blueprint, {
      rawInput: 'hi',
      contextBuffer: SAMPLE_HISTORY,
      timeoutMs: 3000,
    });

    expect(result.reply).toBe('[system]: 系统提示');
  });

  it('message_types 过滤只保留指定角色的消息', async () => {
    const { runtime } = makeRuntime();
    const blueprint = bp(
      [
        node('t', 'trigger', '触发'),
        node('h', 'history', '历史', { message_types: ['user'] }),
        node('r', 'reply', '回复', { template: '$h:out_history' }),
        node('e', 'end', '结束'),
      ],
      [
        conn('c1', 't', 'out_exec', 'h', 'in_exec'),
        conn('c2', 'h', 'out_exec', 'r', 'in_exec'),
        conn('c3', 'r', 'out_exec', 'e', 'in_exec'),
      ],
    );

    const result = await runtime.execute(blueprint, {
      rawInput: 'hi',
      contextBuffer: SAMPLE_HISTORY,
      timeoutMs: 3000,
    });

    expect(result.reply).toContain('[user]: 你好');
    expect(result.reply).toContain('[user]: 今天天气如何');
    expect(result.reply).not.toContain('[assistant]');
    expect(result.reply).not.toContain('[system]');
  });

  it('sort_order=desc 倒序输出（最新在前）', async () => {
    const { runtime } = makeRuntime();
    const blueprint = bp(
      [
        node('t', 'trigger', '触发'),
        node('h', 'history', '历史', { sort_order: 'desc' }),
        node('r', 'reply', '回复', { template: '$h:out_history' }),
        node('e', 'end', '结束'),
      ],
      [
        conn('c1', 't', 'out_exec', 'h', 'in_exec'),
        conn('c2', 'h', 'out_exec', 'r', 'in_exec'),
        conn('c3', 'r', 'out_exec', 'e', 'in_exec'),
      ],
    );

    const result = await runtime.execute(blueprint, {
      rawInput: 'hi',
      contextBuffer: SAMPLE_HISTORY,
      timeoutMs: 3000,
    });

    const lines = result.reply.split('\n');
    expect(lines[0]).toContain('系统提示');
    expect(lines[lines.length - 1]).toContain('你好');
  });
});

// ─── 空历史行为 ──────────────────────────────────────────────

describe('History 节点 — 空历史', () => {
  it('contextBuffer 为空时输出空字符串', async () => {
    const { runtime } = makeRuntime();
    const blueprint = bp(
      [
        node('t', 'trigger', '触发'),
        node('h', 'history', '历史'),
        node('r', 'reply', '回复', { template: '[$h:out_history]' }),
        node('e', 'end', '结束'),
      ],
      [
        conn('c1', 't', 'out_exec', 'h', 'in_exec'),
        conn('c2', 'h', 'out_exec', 'r', 'in_exec'),
        conn('c3', 'r', 'out_exec', 'e', 'in_exec'),
      ],
    );

    const result = await runtime.execute(blueprint, {
      rawInput: 'first message',
      contextBuffer: [],
      timeoutMs: 3000,
    });

    expect(result.reply).toBe('[]');
  });

  it('contextBuffer 为 undefined 时同样输出空字符串', async () => {
    const { runtime } = makeRuntime();
    const blueprint = bp(
      [
        node('t', 'trigger', '触发'),
        node('h', 'history', '历史'),
        node('r', 'reply', '回复', { template: '[$h:out_history]' }),
        node('e', 'end', '结束'),
      ],
      [
        conn('c1', 't', 'out_exec', 'h', 'in_exec'),
        conn('c2', 'h', 'out_exec', 'r', 'in_exec'),
        conn('c3', 'r', 'out_exec', 'e', 'in_exec'),
      ],
    );

    const result = await runtime.execute(blueprint, {
      rawInput: 'first',
      timeoutMs: 3000,
    });

    expect(result.reply).toBe('[]');
  });
});

// ─── 硬上限 ──────────────────────────────────────────────────

describe('History 节点 — 硬上限', () => {
  it('count 超过 100 时截断为 100', async () => {
    const { runtime } = makeRuntime();
    const bigHistory = Array.from({ length: 150 }, (_, i) => ({
      role: 'user',
      content: `msg-${i}`,
    }));

    const blueprint = bp(
      [
        node('t', 'trigger', '触发'),
        node('h', 'history', '历史', { count: 150 }),
        node('r', 'reply', '回复', { template: '$h:out_history' }),
        node('e', 'end', '结束'),
      ],
      [
        conn('c1', 't', 'out_exec', 'h', 'in_exec'),
        conn('c2', 'h', 'out_exec', 'r', 'in_exec'),
        conn('c3', 'r', 'out_exec', 'e', 'in_exec'),
      ],
    );

    const result = await runtime.execute(blueprint, {
      rawInput: 'hi',
      contextBuffer: bigHistory,
      timeoutMs: 3000,
    });

    const lines = result.reply.split('\n');
    expect(lines).toHaveLength(100);
    expect(lines[0]).toContain('msg-50');
    expect(lines[99]).toContain('msg-149');
  });
});

// ─── AC4: 稳定节点 key 引用 ──────────────────────────────────

describe('History 节点 — 稳定引用', () => {
  it('Template 节点通过 $nodeId:out_history 引用历史输出', async () => {
    const { runtime } = makeRuntime();
    const blueprint = bp(
      [
        node('t', 'trigger', '触发'),
        node('h', 'history', '历史'),
        node('tpl', 'template', '模板', { template: '对话记录:\n$h:out_history' }),
        node('r', 'reply', '回复'),
        node('e', 'end', '结束'),
      ],
      [
        conn('c1', 't', 'out_exec', 'h', 'in_exec'),
        conn('c2', 'h', 'out_exec', 'tpl', 'in_exec'),
        conn('c3', 'tpl', 'out_exec', 'r', 'in_exec'),
        conn('c4', 'tpl', 'out_output', 'r', 'in_content'),
        conn('c5', 'r', 'out_exec', 'e', 'in_exec'),
      ],
    );

    const result = await runtime.execute(blueprint, {
      rawInput: 'hello',
      contextBuffer: [{ role: 'user', content: '之前的话' }],
      timeoutMs: 3000,
    });

    expect(result.reply).toContain('对话记录:');
    expect(result.reply).toContain('[user]: 之前的话');
  });

  it('Reply 节点通过人类格式 {历史.历史记录} 引用', async () => {
    const { runtime } = makeRuntime();
    const blueprint = bp(
      [
        node('t', 'trigger', '触发'),
        node('h', 'history', '历史'),
        node('r', 'reply', '回复', { template: '之前: {历史.历史记录}' }),
        node('e', 'end', '结束'),
      ],
      [
        conn('c1', 't', 'out_exec', 'h', 'in_exec'),
        conn('c2', 'h', 'out_exec', 'r', 'in_exec'),
        conn('c3', 'r', 'out_exec', 'e', 'in_exec'),
      ],
    );

    const result = await runtime.execute(blueprint, {
      rawInput: 'hello',
      contextBuffer: [{ role: 'assistant', content: 'AI回复' }],
      timeoutMs: 3000,
    });

    expect(result.reply).toContain('[assistant]: AI回复');
  });

  it('If 节点可用历史内容做条件判断', async () => {
    const { runtime } = makeRuntime();
    const blueprint = bp(
      [
        node('t', 'trigger', '触发'),
        node('h', 'history', '历史'),
        node('cond', 'if', '条件', {
          conditions: [{ left: '$h:out_history', operator: 'contains', right: '重要' }],
        }),
        node('r1', 'reply', '有', { template: '包含重要内容' }),
        node('r2', 'reply', '无', { template: '不包含' }),
        node('e', 'end', '结束'),
      ],
      [
        conn('c1', 't', 'out_exec', 'h', 'in_exec'),
        conn('c2', 'h', 'out_exec', 'cond', 'in_exec'),
        conn('c3', 'cond', 'out_true', 'r1', 'in_exec'),
        conn('c4', 'cond', 'out_false', 'r2', 'in_exec'),
        conn('c5', 'r1', 'out_exec', 'e', 'in_exec'),
        conn('c6', 'r2', 'out_exec', 'e', 'in_exec'),
      ],
    );

    const result = await runtime.execute(blueprint, {
      rawInput: 'test',
      contextBuffer: [{ role: 'user', content: '这是重要信息' }],
      timeoutMs: 3000,
    });

    expect(result.reply).toBe('包含重要内容');
  });
});

// ─── AC2: capability 校验 ────────────────────────────────────

describe('History 节点 — Capability', () => {
  it('read_history 已授权时正常执行', async () => {
    const { runtime } = makeRuntime();
    const blueprint = bp(
      [
        node('t', 'trigger', '触发'),
        node('h', 'history', '历史'),
        node('r', 'reply', '回复', { template: '$h:out_history' }),
        node('e', 'end', '结束'),
      ],
      [
        conn('c1', 't', 'out_exec', 'h', 'in_exec'),
        conn('c2', 'h', 'out_exec', 'r', 'in_exec'),
        conn('c3', 'r', 'out_exec', 'e', 'in_exec'),
      ],
    );

    const result = await runtime.execute(blueprint, {
      rawInput: 'hi',
      contextBuffer: SAMPLE_HISTORY,
      grantedCapabilities: ['messages:read_trigger', 'messages:read_history', 'messages:send'],
      timeoutMs: 3000,
    });

    expect(result.status).toBe(ExecutionStatus.Done);
    expect(result.reply).toContain('你好');
  });

  it('read_history 未授权时拒绝执行', async () => {
    const { runtime } = makeRuntime();
    const blueprint = bp(
      [node('t', 'trigger', '触发'), node('h', 'history', '历史'), node('e', 'end', '结束')],
      [conn('c1', 't', 'out_exec', 'h', 'in_exec'), conn('c2', 'h', 'out_exec', 'e', 'in_exec')],
    );

    await expect(
      runtime.execute(blueprint, {
        rawInput: 'hi',
        grantedCapabilities: ['messages:read_trigger'],
        timeoutMs: 3000,
      }),
    ).rejects.toThrow('Capability denied');
  });
});

// ─── AC5: Production Runtime 与 DebugRunner 一致性 ─────────────

describe('History 节点 — Runtime / DebugRunner 一致性', () => {
  const HISTORY_DOC = makeDoc(
    [
      { id: 't', type: 'trigger', name: '触发' },
      { id: 'h', type: 'history', name: '历史', config: { count: 3 } },
      { id: 'r', type: 'reply', name: '回复', config: { template: '$h:out_history' } },
      { id: 'e', type: 'end', name: '结束' },
    ],
    [
      { from: { nodeId: 't', portId: 'out_exec' }, to: { nodeId: 'h', portId: 'in_exec' } },
      { from: { nodeId: 'h', portId: 'out_exec' }, to: { nodeId: 'r', portId: 'in_exec' } },
      { from: { nodeId: 'r', portId: 'out_exec' }, to: { nodeId: 'e', portId: 'in_exec' } },
    ],
  );

  const CTX: Array<{ role: string; content: string }> = [
    { role: 'user', content: 'msg-A' },
    { role: 'assistant', content: 'msg-B' },
    { role: 'user', content: 'msg-C' },
    { role: 'assistant', content: 'msg-D' },
  ];

  it('Production Runtime 正确输出最近 3 条历史', async () => {
    const { runtime } = makeRuntime();
    const { toBlueprint } = await import('../validator.js');
    const blueprint = toBlueprint(HISTORY_DOC);

    const result = await runtime.execute(blueprint, {
      rawInput: 'test',
      contextBuffer: CTX,
      timeoutMs: 3000,
    });

    expect(result.reply).toContain('msg-B');
    expect(result.reply).toContain('msg-C');
    expect(result.reply).toContain('msg-D');
    expect(result.reply).not.toContain('msg-A');
  });

  it('DebugRunner 正确输出最近 3 条历史', async () => {
    const runner = makeRunner();
    const trace = await runner.run({
      document: HISTORY_DOC,
      message: 'test',
      contextBuffer: CTX,
    });

    expect(trace.status).toBe('completed');
    const replyNode = trace.nodes.find((n) => n.nodeType === 'reply');
    expect(replyNode).toBeTruthy();
    const reply = replyNode!.output?.['__reply__'] ?? '';
    expect(reply).toContain('msg-B');
    expect(reply).toContain('msg-C');
    expect(reply).toContain('msg-D');
    expect(reply).not.toContain('msg-A');
  });

  it('两者输出一致', async () => {
    const { runtime } = makeRuntime();
    const runner = makeRunner();
    const { toBlueprint } = await import('../validator.js');
    const blueprint = toBlueprint(HISTORY_DOC);

    const runtimeResult = await runtime.execute(blueprint, {
      rawInput: 'test',
      contextBuffer: CTX,
      timeoutMs: 3000,
    });

    const trace = await runner.run({
      document: HISTORY_DOC,
      message: 'test',
      contextBuffer: CTX,
    });

    const debugReply = trace.nodes.find((n) => n.nodeType === 'reply')?.output?.['__reply__'] ?? '';

    expect(debugReply).toBe(runtimeResult.reply);
  });
});

// ─── AC6: 多轮会话上下文 ──────────────────────────────────────

describe('History 节点 — 多轮会话与新 session', () => {
  it('同一 session 多轮消息共享初始 contextBuffer', async () => {
    const { runtime } = makeRuntime();

    const blueprint = bp(
      [
        node('t', 'trigger', '触发'),
        node('h', 'history', '历史'),
        node('r1', 'reply', '提问', { template: '你说: $h:out_history' }),
        node('w', 'wait', '等待'),
        node('h2', 'history', '历史2'),
        node('r2', 'reply', '回答', { template: '回忆: $h2:out_history' }),
        node('e', 'end', '结束'),
      ],
      [
        conn('c1', 't', 'out_exec', 'h', 'in_exec'),
        conn('c2', 'h', 'out_exec', 'r1', 'in_exec'),
        conn('c3', 'r1', 'out_exec', 'w', 'in_exec'),
        conn('c4', 'w', 'out_exec', 'h2', 'in_exec'),
        conn('c5', 'h2', 'out_exec', 'r2', 'in_exec'),
        conn('c6', 'r2', 'out_exec', 'e', 'in_exec'),
      ],
    );

    const sessionId = 'multi-turn-history';
    const initialCtx = [
      { role: 'user', content: '旧消息1' },
      { role: 'assistant', content: '旧回复1' },
    ];

    runtime.createSession(sessionId, blueprint, {
      senderName: 'alice',
      contextBuffer: initialCtx,
    });

    const r1 = await runtime.sendMessage(sessionId, '你好', { timeoutMs: 3000 });
    expect(r1.status).toBe(ExecutionStatus.Waiting);
    expect(r1.reply).toContain('旧消息1');

    const r2 = await runtime.sendMessage(sessionId, '回忆', { timeoutMs: 3000 });
    expect(r2.status).toBe(ExecutionStatus.Done);
    expect(r2.reply).toContain('旧消息1');
  });

  it('新 session 使用新 contextBuffer，不残留旧数据', async () => {
    const { runtime } = makeRuntime();

    const blueprint = bp(
      [
        node('t', 'trigger', '触发'),
        node('h', 'history', '历史'),
        node('r', 'reply', '回复', { template: '$h:out_history' }),
        node('e', 'end', '结束'),
      ],
      [
        conn('c1', 't', 'out_exec', 'h', 'in_exec'),
        conn('c2', 'h', 'out_exec', 'r', 'in_exec'),
        conn('c3', 'r', 'out_exec', 'e', 'in_exec'),
      ],
    );

    const sid1 = 'session-old';
    runtime.createSession(sid1, blueprint, {
      contextBuffer: [{ role: 'user', content: 'OLD_DATA' }],
    });
    const r1 = await runtime.sendMessage(sid1, 'test', { timeoutMs: 3000 });
    expect(r1.reply).toContain('OLD_DATA');

    const sid2 = 'session-new';
    runtime.createSession(sid2, blueprint, {
      contextBuffer: [{ role: 'user', content: 'NEW_DATA' }],
    });
    const r2 = await runtime.sendMessage(sid2, 'test', { timeoutMs: 3000 });
    expect(r2.reply).toContain('NEW_DATA');
    expect(r2.reply).not.toContain('OLD_DATA');
  });
});

// ─── Trace 收集 ──────────────────────────────────────────────

describe('History 节点 — DebugRunner Trace', () => {
  it('trace 中包含 history 节点的输入和输出', async () => {
    const runner = makeRunner();
    const doc = makeDoc(
      [
        { id: 't', type: 'trigger', name: '触发' },
        { id: 'h', type: 'history', name: '历史', config: { count: 2 } },
        { id: 'r', type: 'reply', name: '回复', config: { template: '$h:out_history' } },
        { id: 'e', type: 'end', name: '结束' },
      ],
      [
        { from: { nodeId: 't', portId: 'out_exec' }, to: { nodeId: 'h', portId: 'in_exec' } },
        { from: { nodeId: 'h', portId: 'out_exec' }, to: { nodeId: 'r', portId: 'in_exec' } },
        { from: { nodeId: 'r', portId: 'out_exec' }, to: { nodeId: 'e', portId: 'in_exec' } },
      ],
    );

    const trace = await runner.run({
      document: doc,
      message: 'test',
      contextBuffer: SAMPLE_HISTORY,
    });

    const historyNode = trace.nodes.find((n) => n.nodeType === 'history');
    expect(historyNode).toBeTruthy();
    expect(historyNode!.status).toBe('success');
    expect(historyNode!.output?.out_history).toContain('系统提示');
    expect(historyNode!.durationMs).toBeGreaterThanOrEqual(0);
  });
});
