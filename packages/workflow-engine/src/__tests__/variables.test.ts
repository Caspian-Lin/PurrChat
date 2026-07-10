import { describe, it, expect } from 'vitest';
import {
  resolveTemplate,
  type ResolveContext,
} from '../resolver.js';
import {
  Compiler,
  WorkflowRuntime,
  NodeRegistry,
  allNodes,
  validateWorkflowDocument,
  type Blueprint,
  type BlueprintNode,
  type BlueprintConnection,
} from '../index.js';
import { getDefaultPorts, type EventType, createEmptyDocument } from '@purrchat/workflow-types';
import { ExecutionStatus } from '../types.js';

// ─── Resolver 单元测试 ──────────────────────────────────────

function makeCtx(overrides: Partial<ResolveContext> = {}): ResolveContext {
  return {
    nodeOutputs: {},
    nameResolver: {},
    nodeKeyMap: {},
    variables: {},
    eventOutputs: {},
    rawInput: '',
    senderId: '',
    senderName: '',
    conversationId: '',
    history: [],
    secrets: {},
    session: {},
    ...overrides,
  };
}

describe('统一变量解析器', () => {
  describe('${path} 规范格式', () => {
    it('${input.text} 解析为完整用户消息', () => {
      const ctx = makeCtx({ rawInput: 'hello world' });
      expect(resolveTemplate('${input.text}', ctx)).toBe('hello world');
    });

    it('${input.args.N} 解析为第 N 个词', () => {
      const ctx = makeCtx({ rawInput: 'foo bar baz' });
      expect(resolveTemplate('${input.args.1}', ctx)).toBe('foo');
      expect(resolveTemplate('${input.args.2}', ctx)).toBe('bar');
      expect(resolveTemplate('${input.args.3}', ctx)).toBe('baz');
      expect(resolveTemplate('${input.args.4}', ctx)).toBe('');
    });

    it('${sender.id} 和 ${sender.name} 解析为发送者信息', () => {
      const ctx = makeCtx({ senderId: 'u123', senderName: 'alice' });
      expect(resolveTemplate('${sender.id}', ctx)).toBe('u123');
      expect(resolveTemplate('${sender.name}', ctx)).toBe('alice');
    });

    it('${conversation.id} 解析为会话 ID', () => {
      const ctx = makeCtx({ conversationId: 'conv_abc' });
      expect(resolveTemplate('${conversation.id}', ctx)).toBe('conv_abc');
    });

    it('${nodes.<key>.outputs.<port>} 通过 nodeKeyMap 解析', () => {
      const ctx = makeCtx({
        nodeKeyMap: { llm_1: 'n_abc' },
        nodeOutputs: { n_abc: { out_output: 'result text' } },
      });
      expect(resolveTemplate('${nodes.llm_1.outputs.out_output}', ctx)).toBe('result text');
    });

    it('${nodes.<key>.outputs.<port>} 引用不存在的 key 时保留原文', () => {
      const ctx = makeCtx({});
      expect(resolveTemplate('${nodes.unknown.outputs.out}', ctx)).toBe('${nodes.unknown.outputs.out}');
    });

    it('${secrets.<name>} 从运行时 secret 表解析', () => {
      const ctx = makeCtx({ secrets: { api_key: 'sk-12345' } });
      expect(resolveTemplate('${secrets.api_key}', ctx)).toBe('sk-12345');
    });

    it('${history.messages} 格式化消息历史', () => {
      const ctx = makeCtx({
        history: [
          { role: 'user', content: '你好' },
          { role: 'assistant', content: '你好！' },
        ],
      });
      const result = resolveTemplate('${history.messages}', ctx);
      expect(result).toContain('用户: 你好');
      expect(result).toContain('AI: 你好！');
    });

    it('${session.<name>} 从会话变量表解析', () => {
      const ctx = makeCtx({ session: { mood: 'happy' } });
      expect(resolveTemplate('${session.mood}', ctx)).toBe('happy');
    });

    it('混合 ${path} 在同一字符串中同时解析', () => {
      const ctx = makeCtx({
        rawInput: 'hi',
        senderName: 'bob',
        nodeKeyMap: { reply_1: 'r1' },
        nodeOutputs: { r1: { out_exec: 'true' } },
      });
      const result = resolveTemplate('${sender.name} said ${input.text}, exec=${nodes.reply_1.outputs.out_exec}', ctx);
      expect(result).toBe('bob said hi, exec=true');
    });
  });

  describe('遗留格式向后兼容', () => {
    it('{args} 解析为完整输入', () => {
      const ctx = makeCtx({ rawInput: 'hello world' });
      expect(resolveTemplate('{args}', ctx)).toBe('hello world');
    });

    it('{args:N} 解析为第 N 个词', () => {
      const ctx = makeCtx({ rawInput: 'one two three' });
      expect(resolveTemplate('{args:1}', ctx)).toBe('one');
      expect(resolveTemplate('{args:2}', ctx)).toBe('two');
    });

    it('{nodeName.portName} 通过 nameResolver 解析', () => {
      const ctx = makeCtx({
        nameResolver: { '触发.用户消息': 't:out_input' },
        nodeOutputs: { t: { out_input: 'hello' } },
      });
      expect(resolveTemplate('{触发.用户消息}', ctx)).toBe('hello');
    });

    it('$nodeId:portId 直接解析', () => {
      const ctx = makeCtx({
        nodeOutputs: { t: { out_input: 'ping' } },
      });
      expect(resolveTemplate('$t:out_input', ctx)).toBe('ping');
    });

    it('$variableName 从 variables 表解析', () => {
      const ctx = makeCtx({ variables: { username: 'alice' } });
      expect(resolveTemplate('hi $username', ctx)).toBe('hi alice');
    });

    it('{varName} 裸变量从 variables 表解析（template 节点遗留行为）', () => {
      const ctx = makeCtx({ variables: { mood: 'happy' } });
      expect(resolveTemplate('{mood}', ctx)).toBe('happy');
    });

    it('${path} 优先于 {name.port} 遗留格式', () => {
      const ctx = makeCtx({
        rawInput: 'canonical',
        nameResolver: { 'input.text': 't:out_input' },
        nodeOutputs: { t: { out_input: 'legacy' } },
      });
      // ${input.text} 应优先解析为 canonical，不被 {input.text} 的 nameResolver 覆盖
      expect(resolveTemplate('${input.text}', ctx)).toBe('canonical');
    });
  });
});

// ─── 端到端集成测试 ─────────────────────────────────────────

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
  key?: string,
): BlueprintNode {
  return { id, type, name, key, config, ports: getDefaultPorts(type) };
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

describe('端到端：统一变量在 workflow 执行中', () => {
  it('${input.text} 和 ${sender.name} 在 reply 模板中生效', async () => {
    const { runtime } = makeRuntime();
    const blueprint = bp(
      [
        node('t', 'trigger', '触发', {}, 'trigger_1'),
        node('r', 'reply', '回复', { template: '${sender.name}: ${input.text}' }, 'reply_1'),
        node('e', 'end', '结束', {}, 'end_1'),
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

    expect(result.reply).toBe('alice: hello world');
  });

  it('${nodes.<key>.outputs.<port>} 引用上游节点输出', async () => {
    const { runtime } = makeRuntime();
    const blueprint = bp(
      [
        node('t', 'trigger', '触发', {}, 'trigger_1'),
        node('r', 'reply', '回复', { template: '${nodes.trigger_1.outputs.out_input}' }, 'reply_1'),
        node('e', 'end', '结束', {}, 'end_1'),
      ],
      [
        conn('c1', 't', 'out_exec', 'r', 'in_exec'),
        conn('c2', 'r', 'out_exec', 'e', 'in_exec'),
      ],
    );

    const result = await runtime.execute(blueprint, {
      rawInput: 'test message',
      senderName: 'u',
      timeoutMs: 3000,
    });

    expect(result.reply).toBe('test message');
  });

  it('遗留格式 $nodeId:portId 仍然正常工作（向后兼容）', async () => {
    const { runtime } = makeRuntime();
    const blueprint = bp(
      [
        node('t', 'trigger', '触发'),
        node('r', 'reply', '回复', { template: 'msg=$t:out_input' }),
        node('e', 'end', '结束'),
      ],
      [
        conn('c1', 't', 'out_exec', 'r', 'in_exec'),
        conn('c2', 'r', 'out_exec', 'e', 'in_exec'),
      ],
    );

    const result = await runtime.execute(blueprint, {
      rawInput: 'legacy',
      senderName: 's',
      timeoutMs: 3000,
    });

    expect(result.reply).toBe('msg=legacy');
  });

  it('遗留格式 {args} 和 {触发.用户消息} 仍然正常工作', async () => {
    const { runtime } = makeRuntime();
    const blueprint = bp(
      [
        node('t', 'trigger', '触发'),
        node('r', 'reply', '回复', { template: 'args={args} msg={触发.用户消息}' }),
        node('e', 'end', '结束'),
      ],
      [
        conn('c1', 't', 'out_exec', 'r', 'in_exec'),
        conn('c2', 'r', 'out_exec', 'e', 'in_exec'),
      ],
    );

    const result = await runtime.execute(blueprint, {
      rawInput: 'hello world',
      senderName: 's',
      timeoutMs: 3000,
    });

    expect(result.reply).toBe('args=hello world msg=hello world');
  });

  it('新格式和旧格式可在同一模板混用', async () => {
    const { runtime } = makeRuntime();
    const blueprint = bp(
      [
        node('t', 'trigger', '触发', {}, 'trigger_1'),
        node('r', 'reply', '回复', {
          template: '${sender.name} said {args} via ${nodes.trigger_1.outputs.out_input}',
        }, 'reply_1'),
        node('e', 'end', '结束', {}, 'end_1'),
      ],
      [
        conn('c1', 't', 'out_exec', 'r', 'in_exec'),
        conn('c2', 'r', 'out_exec', 'e', 'in_exec'),
      ],
    );

    const result = await runtime.execute(blueprint, {
      rawInput: 'hello there',
      senderName: 'alice',
      timeoutMs: 3000,
    });

    expect(result.reply).toBe('alice said hello there via hello there');
  });

  it('if 条件中 ${input.text} 解析后正确分支', async () => {
    const { runtime } = makeRuntime();
    const blueprint = bp(
      [
        node('t', 'trigger', '触发', {}, 'trigger_1'),
        node('cond', 'if', '条件', {
          conditions: [
            { left: '${input.text}', operator: 'contains', right: 'hello' },
          ],
          logic: 'and',
        }, 'if_1'),
        node('rt', 'reply', '肯定', { template: 'yes' }, 'reply_1'),
        node('rf', 'reply', '否定', { template: 'no' }, 'reply_2'),
        node('e', 'end', '结束', {}, 'end_1'),
      ],
      [
        conn('c1', 't', 'out_exec', 'cond', 'in_exec'),
        conn('c2', 'cond', 'out_true', 'rt', 'in_exec'),
        conn('c3', 'cond', 'out_false', 'rf', 'in_exec'),
        conn('c4', 'rt', 'out_exec', 'e', 'in_exec'),
        conn('c5', 'rf', 'out_exec', 'e', 'in_exec'),
      ],
    );

    const r1 = await runtime.execute(blueprint, {
      rawInput: 'hello world',
      senderName: 's',
      timeoutMs: 3000,
    });
    expect(r1.reply).toBe('yes');

    const r2 = await runtime.execute(blueprint, {
      rawInput: 'goodbye',
      senderName: 's',
      timeoutMs: 3000,
    });
    expect(r2.reply).toBe('no');
  });
});

// ─── Validator 变量引用校验测试 ─────────────────────────────

describe('Validator 变量引用校验', () => {
  function validateDoc(doc: any) {
    const registry = new NodeRegistry();
    registry.registerAll(allNodes);
    return validateWorkflowDocument(doc, registry);
  }

  it('${nodes.<key>.outputs.<port>} 引用存在的节点端口 → 无 warning', () => {
    const doc = createEmptyDocument('test');
    doc.spec.nodes = [
      { id: 't', type: 'trigger', key: 'trigger_1', name: '触发', config: {}, ports: getDefaultPorts('trigger') },
      { id: 'r', type: 'reply', key: 'reply_1', name: '回复', config: { template: '${nodes.trigger_1.outputs.out_input}' }, ports: getDefaultPorts('reply') },
    ];
    doc.spec.connections = [
      { id: 'c1', sourceNodeId: 't', sourcePortId: 'out_exec', targetNodeId: 'r', targetPortId: 'in_exec' },
    ];

    const result = validateDoc(doc);
    const varWarnings = result.issues.filter((i) => i.code.startsWith('var_ref_'));
    expect(varWarnings).toHaveLength(0);
  });

  it('${nodes.<key>.outputs.<port>} 引用不存在的 key → warning', () => {
    const doc = createEmptyDocument('test');
    doc.spec.nodes = [
      { id: 't', type: 'trigger', key: 'trigger_1', name: '触发', config: {}, ports: getDefaultPorts('trigger') },
      { id: 'r', type: 'reply', key: 'reply_1', name: '回复', config: { template: '${nodes.nonexistent.outputs.out_input}' }, ports: getDefaultPorts('reply') },
    ];
    doc.spec.connections = [
      { id: 'c1', sourceNodeId: 't', sourcePortId: 'out_exec', targetNodeId: 'r', targetPortId: 'in_exec' },
    ];

    const result = validateDoc(doc);
    const varWarnings = result.issues.filter((i) => i.code === 'var_ref_unknown_node');
    expect(varWarnings.length).toBeGreaterThan(0);
  });

  it('${nodes.<key>.outputs.<port>} 引用不存在的端口 → warning', () => {
    const doc = createEmptyDocument('test');
    doc.spec.nodes = [
      { id: 't', type: 'trigger', key: 'trigger_1', name: '触发', config: {}, ports: getDefaultPorts('trigger') },
      { id: 'r', type: 'reply', key: 'reply_1', name: '回复', config: { template: '${nodes.trigger_1.outputs.nonexistent_port}' }, ports: getDefaultPorts('reply') },
    ];
    doc.spec.connections = [
      { id: 'c1', sourceNodeId: 't', sourcePortId: 'out_exec', targetNodeId: 'r', targetPortId: 'in_exec' },
    ];

    const result = validateDoc(doc);
    const varWarnings = result.issues.filter((i) => i.code === 'var_ref_unknown_port');
    expect(varWarnings.length).toBeGreaterThan(0);
  });

  it('${input.text} 等内置变量不产生 warning', () => {
    const doc = createEmptyDocument('test');
    doc.spec.nodes = [
      { id: 't', type: 'trigger', key: 'trigger_1', name: '触发', config: {}, ports: getDefaultPorts('trigger') },
      { id: 'r', type: 'reply', key: 'reply_1', name: '回复', config: { template: '${input.text} ${sender.name} ${conversation.id}' }, ports: getDefaultPorts('reply') },
    ];
    doc.spec.connections = [
      { id: 'c1', sourceNodeId: 't', sourcePortId: 'out_exec', targetNodeId: 'r', targetPortId: 'in_exec' },
    ];

    const result = validateDoc(doc);
    const varWarnings = result.issues.filter((i) => i.code.startsWith('var_ref_'));
    expect(varWarnings).toHaveLength(0);
  });
});
