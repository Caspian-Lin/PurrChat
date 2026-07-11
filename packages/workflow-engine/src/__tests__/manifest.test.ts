import { describe, expect, it } from 'vitest';
import {
  NODE_MANIFEST,
  NODE_TYPE_META,
  createEmptyDocument,
  getDefaultPorts,
  type EventType,
} from '@purrchat/workflow-types';
import { NodeRegistry, allNodes, validateWorkflowDocument } from '../index.js';

const registry = new NodeRegistry();
registry.registerAll(allNodes);

describe('NODE_MANIFEST', () => {
  it('covers every node type exactly once and is JSON serializable', () => {
    const manifestTypes = NODE_MANIFEST.map((entry) => entry.type);

    expect(new Set(manifestTypes).size).toBe(manifestTypes.length);
    expect(new Set(manifestTypes)).toEqual(new Set(Object.keys(NODE_TYPE_META) as EventType[]));
    expect(JSON.parse(JSON.stringify(NODE_MANIFEST))).toEqual(NODE_MANIFEST);
  });

  it('keeps implemented entries aligned with registry schemas and shared ports', () => {
    for (const entry of NODE_MANIFEST) {
      expect(entry.ports).toEqual(getDefaultPorts(entry.type));
      expect(entry.ports.length).toBeGreaterThan(0);

      if (!entry.implemented) continue;

      const definition = registry.get(entry.type);
      expect(definition, `${entry.type} must exist in the registry`).toBeDefined();
      expect(
        definition!.configSchema.safeParse(entry.defaultConfig).success,
        `${entry.type} defaultConfig must satisfy configSchema`,
      ).toBe(true);
    }
  });

  it('only publishes implemented and tested registry nodes', () => {
    for (const entry of NODE_MANIFEST.filter((item) => item.productionReady)) {
      expect(entry.implemented, entry.type).toBe(true);
      expect(entry.tested, entry.type).toBe(true);
      expect(registry.has(entry.type), entry.type).toBe(true);
    }
  });

  it('keeps only unimplemented Python hidden from production', () => {
    expect(NODE_MANIFEST.find((entry) => entry.type === 'python')?.productionReady).toBe(false);
  });

  it.each(['loop', 'switch', 'merge'] as const)('publishes verified %s control flow', (type) => {
    const entry = NODE_MANIFEST.find((item) => item.type === type);
    expect(entry?.tested).toBe(true);
    expect(entry?.productionReady).toBe(true);
  });

  it('publishes verified external nodes as tested', () => {
    for (const type of ['tool', 'dify', 'n8n', 'llm'] as const) {
      expect(NODE_MANIFEST.find((entry) => entry.type === type)?.tested).toBe(true);
    }
  });

  it('rejects an incomplete production Loop during document validation', () => {
    const document = createEmptyDocument('unsupported');
    document.spec.nodes = [
      { id: 'trigger', type: 'trigger', name: '触发', config: {} },
      { id: 'loop', type: 'loop', name: '循环', config: { max_iterations: 10 } },
    ];

    const result = validateWorkflowDocument(document, registry);

    expect(result.issues).toContainEqual(
      expect.objectContaining({
        level: 'error',
        code: 'loop_body_invalid',
        nodeId: 'loop',
      }),
    );
  });
});
