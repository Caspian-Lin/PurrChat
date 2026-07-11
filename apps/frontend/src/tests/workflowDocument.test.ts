import { describe, expect, it, vi } from 'vitest';
import { reactive } from 'vue';
import {
  createEmptyDocument,
  generateNodeKey,
  nodeOutputPath,
  type WorkflowDocument,
} from '@purrchat/workflow-types';
import { documentToYaml } from '@purrchat/workflow-engine';
import { flowToDocumentYaml, yamlToFlowDocument } from '../utils/yamlIR';
import {
  PRODUCTION_NODE_MANIFEST,
  cloneWorkflowEvent,
  evaluateWorkflowGate,
  nextUniqueNodeKey,
  parseWorkflowYamlCandidate,
} from '../utils/workflowDocument';

describe('生产工作流节点', () => {
  it('只暴露 manifest 中可生产使用的节点并隐藏未实现节点', () => {
    const types = PRODUCTION_NODE_MANIFEST.map((node) => node.type);

    expect(types).not.toContain('python');
    expect(types).toContain('loop');
    expect(types).toContain('switch');
    expect(types).toContain('merge');
    expect(
      PRODUCTION_NODE_MANIFEST.every(
        (node) => node.implemented && node.tested && node.productionReady
      )
    ).toBe(true);
  });
});

describe('WorkflowDocument YAML', () => {
  it('完整往返 key、position、trigger 与 endConditions', () => {
    const events = [
      {
        id: 'trigger-id',
        key: 'trigger_1',
        type: 'trigger' as const,
        name: '触发',
        config: {},
        position: { x: 12, y: 34 },
      },
      {
        id: 'reply-id',
        key: 'reply_1',
        type: 'reply' as const,
        name: '回复',
        config: { template: '${input.text}' },
        position: { x: 280, y: 34 },
      },
    ];
    const connections = [
      {
        id: 'connection-1',
        sourceNodeId: 'trigger-id',
        sourcePortId: 'out_exec',
        targetNodeId: 'reply-id',
        targetPortId: 'in_exec',
      },
    ];
    const trigger = {
      type: 'rule' as const,
      rules: [{ type: 'keyword' as const, pattern: '猫' }],
    };
    const endConditions = [{ type: 'max_rounds' as const, value: 5 }];

    const yaml = flowToDocumentYaml(events, connections, trigger, endConditions, '往返测试');
    const result = yamlToFlowDocument(yaml);

    expect(result.errors).toEqual([]);
    expect(result.events.map(({ key, position }) => ({ key, position }))).toEqual([
      { key: 'trigger_1', position: { x: 12, y: 34 } },
      { key: 'reply_1', position: { x: 280, y: 34 } },
    ]);
    expect(result.trigger).toEqual(trigger);
    expect(result.endConditions).toEqual(endConditions);
  });

  it('非法 YAML 和 validation error 都不产生可应用候选', () => {
    const valid = createEmptyDocument('候选');
    const invalidYaml = parseWorkflowYamlCandidate('nodes: [', () => ({
      valid: true,
      issues: [],
      derivedCapabilities: [],
    }));
    const invalidDocument = parseWorkflowYamlCandidate(documentToYaml(valid), () => ({
      valid: false,
      issues: [{ level: 'error', code: 'invalid', message: '缺少触发节点' }],
      derivedCapabilities: [],
    }));

    expect(invalidYaml.candidate).toBeUndefined();
    expect(invalidYaml.errors.length).toBeGreaterThan(0);
    expect(invalidDocument.candidate).toBeUndefined();
    expect(invalidDocument.errors).toEqual(['缺少触发节点']);
  });
});

describe('稳定节点 key 与验证 gate', () => {
  it('可复制包含嵌套响应式配置的节点用于编辑', () => {
    const event = reactive({
      id: 'reply',
      key: 'reply_1',
      type: 'reply' as const,
      name: '回复',
      config: { template: '${input.text}', nested: { enabled: true } },
    });

    const cloned = cloneWorkflowEvent(event);

    expect(cloned).toEqual(event);
    expect(cloned).not.toBe(event);
    expect(cloned.config).not.toBe(event.config);
  });

  it('为同文档变量引用生成唯一且与节点名称无关的 key', () => {
    const document = createEmptyDocument('key');
    document.spec.nodes = [node('a', 'reply_1'), node('b', 'reply_2')];

    const key = nextUniqueNodeKey(document, 'reply');

    expect(key).toBe(generateNodeKey('reply', 3));
    expect(nodeOutputPath(key, 'out_output')).toBe('nodes.reply_3.outputs.out_output');
  });

  it('error 阻止保存发布，warning 仅在明确确认后放行', () => {
    const confirm = vi.fn(() => false);
    const errorGate = evaluateWorkflowGate(
      { issues: [{ level: 'error', code: 'bad', message: '错误' }] },
      confirm
    );
    const rejectedWarning = evaluateWorkflowGate(
      { issues: [{ level: 'warning', code: 'warn', message: '警告' }] },
      confirm
    );
    confirm.mockReturnValue(true);
    const acceptedWarning = evaluateWorkflowGate(
      { issues: [{ level: 'warning', code: 'warn', message: '警告' }] },
      confirm
    );

    expect(errorGate.allowed).toBe(false);
    expect(rejectedWarning.allowed).toBe(false);
    expect(acceptedWarning.allowed).toBe(true);
  });
});

function node(id: string, key: string): WorkflowDocument['spec']['nodes'][number] {
  return {
    id,
    key,
    type: 'reply',
    name: `节点 ${id}`,
    config: { template: '' },
  };
}
