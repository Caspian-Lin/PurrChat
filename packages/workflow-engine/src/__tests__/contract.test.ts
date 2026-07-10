import { describe, it, expect } from 'vitest';
import { z } from 'zod';
import {
  Compiler,
  WorkflowRuntime,
  NodeRegistry,
  allNodes,
  type Blueprint,
  type BlueprintNode,
  type BlueprintConnection,
} from '../index.js';
import { getDefaultPorts, type EventType } from '@purrchat/workflow-types';
import { ExecutionStatus } from '../types.js';

// ─── 测试辅助 ────────────────────────────────────────────────

function makeRuntime(): { runtime: WorkflowRuntime; registry: NodeRegistry } {
  const registry = new NodeRegistry();
  registry.registerAll(allNodes);
  const compiler = new Compiler(registry);
  return { runtime: new WorkflowRuntime(compiler), registry };
}

function node(
  id: string,
  type: EventType,
  name: string,
  config: Record<string, any> = {},
): BlueprintNode {
  return { id, type, name, config, ports: getDefaultPorts(type) };
}

function conn(
  id: string,
  sourceNodeId: string,
  sourcePortId: string,
  targetNodeId: string,
  targetPortId: string,
): BlueprintConnection {
  return { id, sourceNodeId, sourcePortId, targetNodeId, targetPortId };
}

function bp(nodes: BlueprintNode[], connections: BlueprintConnection[], endConditions: any[] = []): Blueprint {
  return { nodes, connections, endConditions };
}

// ─── AC1: rawInput / sender / 变量正确进入 context ─────────

describe('变量与 context 注入', () => {
  it('rawInput、sender、args 正确进入回复模板', async () => {
    const { runtime } = makeRuntime();
    const blueprint = bp(
      [
        node('t', 'trigger', '触发'),
        node('r', 'reply', '回复', { template: 'user=$username args={args} msg={触发.用户消息}' }),
        node('e', 'end', '结束'),
      ],
      [
        conn('c1', 't', 'out_exec', 'r', 'in_exec'),
        conn('c2', 'r', 'out_exec', 'e', 'in_exec'),
      ],
    );

    const result = await runtime.execute(blueprint, {
      rawInput: 'hello world',
      senderName: 'alice',
      timeoutMs: 3000,
    });

    expect(result.status).toBe(ExecutionStatus.Done);
    expect(result.reply).toBe('user=alice args=hello world msg=hello world');
  });

  it('senderName 通过 $username 变量可用', async () => {
    const { runtime } = makeRuntime();
    const blueprint = bp(
      [
        node('t', 'trigger', '触发'),
        node('r', 'reply', '回复', { template: 'hi $username' }),
        node('e', 'end', '结束'),
      ],
      [
        conn('c1', 't', 'out_exec', 'r', 'in_exec'),
        conn('c2', 'r', 'out_exec', 'e', 'in_exec'),
      ],
    );

    const result = await runtime.execute(blueprint, {
      rawInput: 'x',
      senderName: 'bob',
      timeoutMs: 3000,
    });

    expect(result.reply).toBe('hi bob');
  });

  it('机器格式 $nodeId:portId 与人类格式 {节点.端口} 等价', async () => {
    const { runtime } = makeRuntime();
    const blueprint = bp(
      [
        node('t', 'trigger', '触发'),
        node('r', 'reply', '回复', { template: 'a=$t:out_input b={触发.用户消息}' }),
        node('e', 'end', '结束'),
      ],
      [
        conn('c1', 't', 'out_exec', 'r', 'in_exec'),
        conn('c2', 'r', 'out_exec', 'e', 'in_exec'),
      ],
    );

    const result = await runtime.execute(blueprint, {
      rawInput: 'ping',
      senderName: 's',
      timeoutMs: 3000,
    });

    expect(result.reply).toBe('a=ping b=ping');
  });
});

// ─── AC2: 无显式 end 的合法流程可确定性结束 ─────────────────

describe('终止语义', () => {
  it('trigger -> reply（无 end）也能正常结束，不超时', async () => {
    const { runtime } = makeRuntime();
    const blueprint = bp(
      [
        node('t', 'trigger', '触发'),
        node('r', 'reply', '回复', { template: 'no end here' }),
      ],
      [conn('c1', 't', 'out_exec', 'r', 'in_exec')],
    );

    const result = await runtime.execute(blueprint, {
      rawInput: 'go',
      timeoutMs: 3000,
    });

    expect(result.status).toBe(ExecutionStatus.Done);
    expect(result.reply).toBe('no end here');
  });

  it('错误不伪装为 "..."，节点抛错时 execute 抛出', async () => {
    const { runtime } = makeRuntime();
    // builtin template 空配置会抛错
    const blueprint = bp(
      [
        node('t', 'trigger', '触发'),
        node('b', 'builtin', '内置', { builtin_type: 'template', template: '' }),
        node('e', 'end', '结束'),
      ],
      [
        conn('c1', 't', 'out_exec', 'b', 'in_exec'),
        conn('c2', 'b', 'out_exec', 'e', 'in_exec'),
      ],
    );

    await expect(
      runtime.execute(blueprint, { rawInput: 'x', timeoutMs: 3000 }),
    ).rejects.toThrow('template is empty');
  });

  it('未知节点类型在编译期失败', async () => {
    const { runtime } = makeRuntime();
    const blueprint = bp(
      [
        node('t', 'trigger', '触发'),
        { ...node('x', 'builtin', '神秘'), type: 'does_not_exist' },
      ],
      [conn('c1', 't', 'out_exec', 'x', 'in_exec')],
    );

    await expect(
      runtime.execute(blueprint, { rawInput: 'x', timeoutMs: 1000 }),
    ).rejects.toThrow(/Unknown node type/);
  });
});

describe('模板节点', () => {
  it('渲染变量并把输出传给回复节点', async () => {
    const { runtime } = makeRuntime();
    const blueprint = bp(
      [
        node('t', 'trigger', '触发'),
        node('tpl', 'template', '模板', { template: 'hello {触发.用户消息}' }),
        node('r', 'reply', '回复'),
        node('e', 'end', '结束'),
      ],
      [
        conn('c1', 't', 'out_exec', 'tpl', 'in_exec'),
        conn('c2', 'tpl', 'out_output', 'r', 'in_content'),
        conn('c3', 'tpl', 'out_exec', 'r', 'in_exec'),
        conn('c4', 'r', 'out_exec', 'e', 'in_exec'),
      ],
    );

    const result = await runtime.execute(blueprint, {
      rawInput: 'world',
      timeoutMs: 3000,
    });

    expect(result.reply).toBe('hello world');
  });
});

// ─── AC5: if 条件分支（成功与失败路径） ─────────────────────

describe('if 条件分支', () => {
  function ifBlueprint() {
    return bp(
      [
        node('t', 'trigger', '触发'),
        node('cond', 'if', '条件', {
          conditions: [
            { left: '{触发.用户消息}', operator: 'contains', right: 'hello' },
          ],
          logic: 'and',
        }),
        node('rt', 'reply', '肯定', { template: 'yes' }),
        node('rf', 'reply', '否定', { template: 'no' }),
        node('e', 'end', '结束'),
      ],
      [
        conn('c1', 't', 'out_exec', 'cond', 'in_exec'),
        conn('c2', 'cond', 'out_true', 'rt', 'in_exec'),
        conn('c3', 'cond', 'out_false', 'rf', 'in_exec'),
        conn('c4', 'rt', 'out_exec', 'e', 'in_exec'),
        conn('c5', 'rf', 'out_exec', 'e', 'in_exec'),
      ],
    );
  }

  it('条件命中 true 分支', async () => {
    const { runtime } = makeRuntime();
    const result = await runtime.execute(ifBlueprint(), {
      rawInput: 'hello there',
      timeoutMs: 3000,
    });
    expect(result.reply).toBe('yes');
  });

  it('条件命中 false 分支', async () => {
    const { runtime } = makeRuntime();
    const result = await runtime.execute(ifBlueprint(), {
      rawInput: 'goodbye',
      timeoutMs: 3000,
    });
    expect(result.reply).toBe('no');
  });

  it('if 无匹配分支连接时兜底结束，不卡死', async () => {
    const { runtime } = makeRuntime();
    const blueprint = bp(
      [
        node('t', 'trigger', '触发'),
        node('cond', 'if', '条件', {
          conditions: [{ left: 'a', operator: '==', right: 'b' }],
        }),
        // 仅连接 true 分支，false 不连接
        node('rt', 'reply', '肯定', { template: 'yes' }),
        node('e', 'end', '结束'),
      ],
      [
        conn('c1', 't', 'out_exec', 'cond', 'in_exec'),
        conn('c2', 'cond', 'out_true', 'rt', 'in_exec'),
        conn('c3', 'rt', 'out_exec', 'e', 'in_exec'),
      ],
    );

    // 条件为 false（a != b），无 false 连接，应兜底结束
    const result = await runtime.execute(blueprint, {
      rawInput: 'x',
      timeoutMs: 3000,
    });
    expect(result.status).toBe(ExecutionStatus.Done);
  });
});

// ─── AC3: 首条消息只执行一次 + wait 多轮会话 ─────────────────

describe('wait 多轮会话', () => {
  function waitBlueprint(endConditions: any[] = []) {
    return bp(
      [
        node('t', 'trigger', '触发'),
        node('r1', 'reply', '提问', { template: 'what is your name?' }),
        node('w', 'wait', '等待'),
        node('r2', 'reply', '回声', { template: 'echo:{等待.用户输入}' }),
        node('e', 'end', '结束'),
      ],
      [
        conn('c1', 't', 'out_exec', 'r1', 'in_exec'),
        conn('c2', 'r1', 'out_exec', 'w', 'in_exec'),
        conn('c3', 'w', 'out_exec', 'r2', 'in_exec'),
        conn('c4', 'r2', 'out_exec', 'e', 'in_exec'),
      ],
      endConditions,
    );
  }

  it('首条消息触发提问并暂停在 wait；第二条消息唤醒并结束', async () => {
    const { runtime } = makeRuntime();
    const sessionId = 's1';
    runtime.createSession(sessionId, waitBlueprint(), { senderName: 'alice' });

    // 第一轮：trigger -> r1 -> wait（暂停）
    const r1 = await runtime.sendMessage(sessionId, 'hi', { timeoutMs: 3000 });
    expect(r1.reply).toBe('what is your name?');
    expect(r1.status).toBe(ExecutionStatus.Waiting);
    expect(r1.sessionActive).toBe(true);
    expect(runtime.hasSession(sessionId)).toBe(true);

    // 第二轮：wait -> r2 -> end（结束，会话销毁）
    const r2 = await runtime.sendMessage(sessionId, 'alice', { timeoutMs: 3000 });
    expect(r2.reply).toBe('echo:alice');
    expect(r2.sessionActive).toBe(false);
    expect(runtime.hasSession(sessionId)).toBe(false);
  });

  it('首条消息只执行一次：提问节点不会在第二轮重复', async () => {
    const { runtime } = makeRuntime();
    const sessionId = 's2';
    runtime.createSession(sessionId, waitBlueprint(), { senderName: 'alice' });

    const r1 = await runtime.sendMessage(sessionId, 'hi', { timeoutMs: 3000 });
    expect(r1.reply).toBe('what is your name?');

    // 第二轮不应再次返回提问
    const r2 = await runtime.sendMessage(sessionId, 'bob', { timeoutMs: 3000 });
    expect(r2.reply).not.toBe('what is your name?');
    expect(r2.reply).toBe('echo:bob');
  });

  it('会话结束后再次发送会因会话不存在而抛错', async () => {
    const { runtime } = makeRuntime();
    const sessionId = 's3';
    runtime.createSession(sessionId, waitBlueprint(), { senderName: 'alice' });

    await runtime.sendMessage(sessionId, 'hi', { timeoutMs: 3000 });
    await runtime.sendMessage(sessionId, 'alice', { timeoutMs: 3000 });

    // 会话已销毁
    await expect(
      runtime.sendMessage(sessionId, 'again', { timeoutMs: 1000 }),
    ).rejects.toThrow(/not found/);
  });
});

// ─── AC4: endConditions ──────────────────────────────────────

describe('endConditions', () => {
  function twoRoundBlueprint(endConditions: any[]) {
    return bp(
      [
        node('t', 'trigger', '触发'),
        node('r1', 'reply', '回合1', { template: 'round1' }),
        node('w', 'wait', '等待'),
        node('r2', 'reply', '回合2', { template: 'round2' }),
        node('e', 'end', '结束'),
      ],
      [
        conn('c1', 't', 'out_exec', 'r1', 'in_exec'),
        conn('c2', 'r1', 'out_exec', 'w', 'in_exec'),
        conn('c3', 'w', 'out_exec', 'r2', 'in_exec'),
        conn('c4', 'r2', 'out_exec', 'e', 'in_exec'),
      ],
      endConditions,
    );
  }

  it('max_rounds 达到后提前结束会话', async () => {
    const { runtime } = makeRuntime();
    const sessionId = 'mr';
    runtime.createSession(sessionId, twoRoundBlueprint([{ type: 'max_rounds', value: 1 }]), {
      senderName: 'alice',
    });

    const r1 = await runtime.sendMessage(sessionId, 'a', { timeoutMs: 3000 });
    // 第一轮仍回复
    expect(r1.reply).toBe('round1');
    // 但达到 max_rounds=1 后立即结束
    expect(r1.sessionActive).toBe(false);
    expect(runtime.hasSession(sessionId)).toBe(false);

    // 会话已销毁，再次发送失败
    await expect(
      runtime.sendMessage(sessionId, 'b', { timeoutMs: 1000 }),
    ).rejects.toThrow(/not found/);
  });

  it('message_match 命中后提前结束会话', async () => {
    const { runtime } = makeRuntime();
    const sessionId = 'mm';
    runtime.createSession(
      sessionId,
      twoRoundBlueprint([{ type: 'message_match', pattern: '^stop$' }]),
      { senderName: 'alice' },
    );

    // 第一轮消息匹配 stop，本轮回复后立即结束
    const r1 = await runtime.sendMessage(sessionId, 'stop', { timeoutMs: 3000 });
    expect(r1.reply).toBe('round1');
    expect(r1.sessionActive).toBe(false);
    expect(runtime.hasSession(sessionId)).toBe(false);
  });
});

// ─── 并发：同一会话同时只处理一条消息 ───────────────────────

describe('并发控制', () => {
  it('同一会话并发消息被拒绝', async () => {
    const { runtime, registry } = makeRuntime();

    // 注册一个慢节点以制造在途处理
    let resolveSlow: () => void;
    const slowPromise = new Promise<void>((r) => (resolveSlow = r));
    registry.register({
      type: 'slow',
      label: '慢节点',
      category: 'processing',
      icon: '~',
      configSchema: z.object({}),
      async execute() {
        await slowPromise;
        return { ports: { out_output: 'done', out_exec: 'true' } };
      },
    });

    const blueprint = bp(
      [
        node('t', 'trigger', '触发'),
        { ...node('s', 'builtin', '慢'), type: 'slow' },
        node('r', 'reply', '回复', { template: 'ok' }),
        node('e', 'end', '结束'),
      ],
      [
        conn('c1', 't', 'out_exec', 's', 'in_exec'),
        conn('c2', 's', 'out_exec', 'r', 'in_exec'),
        conn('c3', 'r', 'out_exec', 'e', 'in_exec'),
      ],
    );

    const sessionId = 'cc';
    runtime.createSession(sessionId, blueprint, { senderName: 'alice' });

    const first = runtime.sendMessage(sessionId, 'x', { timeoutMs: 5000 });
    // 让事件循环推进一点，确保第一条进入 busy
    await new Promise((r) => setTimeout(r, 10));

    await expect(
      runtime.sendMessage(sessionId, 'y', { timeoutMs: 1000 }),
    ).rejects.toThrow(/busy/);

    resolveSlow!();
    const result = await first;
    expect(result.reply).toBe('ok');
  });
});
