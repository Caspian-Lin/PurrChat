import { describe, it, expect } from 'vitest';
import {
  validateWorkflowDocument,
  NodeRegistry,
  allNodes,
  documentToYaml,
  yamlToDocument,
  documentToJson,
  jsonToDocument,
  sanitizeForExport,
  verifyRoundTrip,
  type ValidationResult,
} from '../index.js';
import {
  createEmptyDocument,
  migrateMechanismToDocument,
  isWorkflowDocument,
  getDefaultPorts,
  type WorkflowDocument,
} from '@purrchat/workflow-types';

function makeRegistry(): NodeRegistry {
  const r = new NodeRegistry();
  r.registerAll(allNodes);
  return r;
}

function validDoc(): WorkflowDocument {
  return {
    apiVersion: 'purrchat.ai/v1alpha1',
    kind: 'BotWorkflow',
    metadata: { name: 'test-bot', revision: 0 },
    spec: {
      trigger: { type: 'rule', rules: [] },
      nodes: [
        { id: 'n1', type: 'trigger', name: '触发', config: {}, ports: getDefaultPorts('trigger') },
        { id: 'n2', type: 'reply', name: '回复', config: { template: 'hello' }, ports: getDefaultPorts('reply') },
        { id: 'n3', type: 'end', name: '结束', config: {}, ports: getDefaultPorts('end') },
      ],
      connections: [
        { id: 'c1', sourceNodeId: 'n1', sourcePortId: 'out_exec', targetNodeId: 'n2', targetPortId: 'in_exec' },
        { id: 'c2', sourceNodeId: 'n2', sourcePortId: 'out_exec', targetNodeId: 'n3', targetPortId: 'in_exec' },
      ],
      endConditions: [{ type: 'max_rounds', value: 5 }],
    },
  };
}

// ─── Validator ────────────────────────────────────────────────

describe('validateWorkflowDocument', () => {
  const registry = makeRegistry();

  it('有效文档通过校验', () => {
    const result = validateWorkflowDocument(validDoc(), registry);
    expect(result.valid).toBe(true);
    expect(result.issues.filter((i) => i.level === 'error')).toHaveLength(0);
  });

  it('apiVersion 错误返回结构化错误', () => {
    const doc = validDoc();
    doc.apiVersion = 'v2' as any;
    const result = validateWorkflowDocument(doc, registry);
    expect(result.valid).toBe(false);
    expect(result.issues.some((i) => i.code === 'unknown_api_version')).toBe(true);
  });

  it('未知节点类型返回 error', () => {
    const doc = validDoc();
    doc.spec.nodes[1].type = 'unknown_type' as any;
    const result = validateWorkflowDocument(doc, registry);
    expect(result.valid).toBe(false);
    expect(result.issues.some((i) => i.code === 'unknown_node_type')).toBe(true);
  });

  it('缺少 trigger 节点返回 error', () => {
    const doc = validDoc();
    doc.spec.nodes = doc.spec.nodes.filter((n) => n.type !== 'trigger');
    const result = validateWorkflowDocument(doc, registry);
    expect(result.issues.some((i) => i.code === 'no_trigger')).toBe(true);
  });

  it('多个 trigger 节点返回 error', () => {
    const doc = validDoc();
    doc.spec.nodes.push({
      id: 'extra_trigger', type: 'trigger', name: '额外触发', config: {}, ports: getDefaultPorts('trigger'),
    });
    const result = validateWorkflowDocument(doc, registry);
    expect(result.issues.some((i) => i.code === 'multiple_trigger')).toBe(true);
  });

  it('连线引用不存在节点返回 error', () => {
    const doc = validDoc();
    doc.spec.connections[0].targetNodeId = 'nonexistent';
    const result = validateWorkflowDocument(doc, registry);
    expect(result.issues.some((i) => i.code === 'dangling_connection_target')).toBe(true);
  });

  it('检测环路', () => {
    const doc = validDoc();
    doc.spec.connections.push({
      id: 'back_edge', sourceNodeId: 'n2', sourcePortId: 'out_exec', targetNodeId: 'n2', targetPortId: 'in_exec',
    });
    const result = validateWorkflowDocument(doc, registry);
    expect(result.issues.some((i) => i.code === 'cycle_detected')).toBe(true);
  });

  it('允许 Loop 循环体唯一回边', () => {
    const doc = createEmptyDocument('loop');
    doc.spec.nodes = [
      { id: 't', type: 'trigger', name: '触发', config: {} },
      { id: 'l', type: 'loop', name: '循环', config: { condition: 'true', max_iterations: 2 } },
      { id: 'b', type: 'template', name: '循环体', config: { template: 'body' } },
      { id: 'r', type: 'reply', name: '完成', config: { template: 'done' } },
    ];
    doc.spec.connections = [
      { id: 'c1', sourceNodeId: 't', sourcePortId: 'out_exec', targetNodeId: 'l', targetPortId: 'in_exec' },
      { id: 'c2', sourceNodeId: 'l', sourcePortId: 'out_body', targetNodeId: 'b', targetPortId: 'in_exec' },
      { id: 'c3', sourceNodeId: 'b', sourcePortId: 'out_exec', targetNodeId: 'l', targetPortId: 'in_exec' },
      { id: 'c4', sourceNodeId: 'l', sourcePortId: 'out_done', targetNodeId: 'r', targetPortId: 'in_exec' },
    ];

    expect(validateWorkflowDocument(doc, registry).valid).toBe(true);
  });

  it('拒绝重复 Switch case 值并要求默认分支', () => {
    const doc = createEmptyDocument('switch');
    doc.spec.nodes = [
      { id: 't', type: 'trigger', name: '触发', config: {} },
      { id: 's', type: 'switch', name: '分支', config: { cases: [{ value: 'same' }, { value: 'same' }] } },
    ];
    doc.spec.connections = [
      { id: 'c1', sourceNodeId: 't', sourcePortId: 'out_exec', targetNodeId: 's', targetPortId: 'in_exec' },
      { id: 'c2', sourceNodeId: 't', sourcePortId: 'out_input', targetNodeId: 's', targetPortId: 'in_value' },
    ];

    const result = validateWorkflowDocument(doc, registry);
    expect(result.issues.some((issue) => issue.code === 'switch_case_value_duplicate')).toBe(true);
    expect(result.issues.some((issue) => issue.code === 'switch_default_missing')).toBe(true);
  });

  it('configSchema 校验非法字段', () => {
    const doc = validDoc();
    doc.spec.nodes[1].config = { template: 12345 };
    const result = validateWorkflowDocument(doc, registry);
    expect(result.issues.some((i) => i.code === 'invalid_config')).toBe(true);
  });

  it('明文 secret 返回 warning', () => {
    const doc = validDoc();
    doc.spec.nodes[1] = {
      id: 'n2', type: 'llm', name: 'LLM', config: {
        api_url: 'https://example.com/v1',
        api_key: 'sk-1234567890abcdef',
        model: 'gpt-4',
      }, ports: getDefaultPorts('llm'),
    };
    const result = validateWorkflowDocument(doc, registry);
    expect(result.issues.some((i) => i.code === 'plaintext_secret' && i.level === 'warning')).toBe(true);
  });

  it('secrets.<name> 引用不产生 warning', () => {
    const doc = validDoc();
    doc.spec.nodes[1] = {
      id: 'n2', type: 'llm', name: 'LLM', config: {
        api_url: 'https://example.com/v1',
        api_key: 'secrets.openai_key',
        model: 'gpt-4',
      }, ports: getDefaultPorts('llm'),
    };
    const result = validateWorkflowDocument(doc, registry);
    expect(result.issues.some((i) => i.code === 'plaintext_secret')).toBe(false);
  });

  it('推导 derivedCapabilities', () => {
    const result = validateWorkflowDocument(validDoc(), registry);
    expect(result.derivedCapabilities).toContain('messages:read_trigger');
    expect(result.derivedCapabilities).toContain('messages:send');
  });
});

// ─── Migration ────────────────────────────────────────────────

describe('migrateMechanismToDocument', () => {
  it('空输入返回空文档', () => {
    const doc = migrateMechanismToDocument(null, 'test');
    expect(doc.apiVersion).toBe('purrchat.ai/v1alpha1');
    expect(doc.kind).toBe('BotWorkflow');
    expect(doc.spec.nodes).toHaveLength(0);
  });

  it('旧 mechanism_config 正确迁移', () => {
    const oldConfig = {
      mechanisms: [{
        id: 'm1', name: '默认', enabled: true,
        trigger: { type: 'rule', rules: [{ type: 'command', pattern: '/hello' }] },
        reply: {
          type: 'workflow',
          workflow: {
            events: [
              { id: 'e1', type: 'trigger', name: '触发', config: {} },
              { id: 'e2', type: 'reply', name: '回复', config: { template: 'hi' } },
            ],
            connections: [],
            end_conditions: [{ type: 'max_rounds', value: 3 }],
          },
        },
      }],
    };
    const doc = migrateMechanismToDocument(oldConfig, 'mybot');
    expect(doc.metadata.name).toBe('mybot');
    expect(doc.metadata.revision).toBe(0);
    expect(doc.spec.trigger.rules?.[0].pattern).toBe('/hello');
    expect(doc.spec.nodes).toHaveLength(2);
    expect(doc.spec.endConditions).toHaveLength(1);
  });

  it('isWorkflowDocument 判断', () => {
    expect(isWorkflowDocument(validDoc())).toBe(true);
    expect(isWorkflowDocument({ foo: 'bar' })).toBe(false);
    expect(isWorkflowDocument(null)).toBe(false);
  });
});

// ─── YAML / JSON 往返 ─────────────────────────────────────────

describe('JSON/YAML 往返', () => {
  it('JSON → YAML → JSON 语义等价', () => {
    const doc = validDoc();
    expect(verifyRoundTrip(doc)).toBe(true);
  });

  it('documentToYaml 覆盖 endConditions', () => {
    const doc = validDoc();
    const yaml = documentToYaml(doc);
    expect(yaml).toContain('endConditions');
    expect(yaml).toContain('max_rounds');
  });

  it('yamlToDocument 正确解析', () => {
    const doc = validDoc();
    const yaml = documentToYaml(doc);
    const parsed = yamlToDocument(yaml);
    expect(parsed.apiVersion).toBe('purrchat.ai/v1alpha1');
    expect(parsed.spec.nodes).toHaveLength(3);
  });

  it('JSON 序列化往返', () => {
    const doc = validDoc();
    const json = documentToJson(doc);
    const parsed = jsonToDocument(json);
    expect(parsed.metadata.name).toBe('test-bot');
    expect(parsed.spec.connections).toHaveLength(2);
  });

  it('sanitizeForExport 替换明文密钥', () => {
    const doc = validDoc();
    doc.spec.nodes[1] = {
      id: 'n2', type: 'llm', name: 'LLM', config: {
        api_url: 'https://example.com',
        api_key: 'sk-secret-key-12345',
        model: 'gpt-4',
      }, ports: getDefaultPorts('llm'),
    };
    const sanitized = sanitizeForExport(doc);
    expect(sanitized.spec.nodes[1].config.api_key).toBe('secrets.api_key');
    expect(sanitized.spec.nodes[1].config.api_url).toBe('https://example.com');
  });

  it('createEmptyDocument 生成有效空文档', () => {
    const doc = createEmptyDocument('new-bot');
    expect(doc.apiVersion).toBe('purrchat.ai/v1alpha1');
    expect(doc.kind).toBe('BotWorkflow');
    expect(doc.metadata.name).toBe('new-bot');
    expect(doc.metadata.revision).toBe(0);
    expect(doc.spec.nodes).toHaveLength(0);
  });
});
