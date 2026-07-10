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

  it.each(['python', 'loop', 'switch', 'merge'] as const)(
    'keeps %s hidden from production',
    (type) => {
      expect(NODE_MANIFEST.find((entry) => entry.type === type)?.productionReady).toBe(false);
    },
  );

  it('does not claim unverified external nodes are tested', () => {
    for (const type of ['tool', 'dify', 'n8n', 'llm'] as const) {
      expect(NODE_MANIFEST.find((entry) => entry.type === type)?.tested).toBe(false);
    }
  });

  it('rejects hidden nodes during document validation', () => {
    const document = createEmptyDocument('unsupported');
    document.spec.nodes = [
      { id: 'trigger', type: 'trigger', name: '触发', config: {} },
      { id: 'loop', type: 'loop', name: '循环', config: { max_iterations: 10 } },
    ];

    const result = validateWorkflowDocument(document, registry);

    expect(result.issues).toContainEqual(
      expect.objectContaining({
        level: 'error',
        code: 'node_not_production_ready',
        nodeId: 'loop',
      }),
    );
  });
});
