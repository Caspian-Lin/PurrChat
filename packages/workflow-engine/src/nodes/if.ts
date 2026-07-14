import { z } from 'zod';
import type { NodeDefinition } from '../types.js';
import { evaluateOperatorCondition } from '../ports.js';
import { resolveTemplate } from '../resolver.js';

const conditionSchema = z.object({
  left: z.string(),
  operator: z.string(),
  right: z.string(),
});

const ifConfigSchema = z.object({
  // Ordered branches represent if / else if. The first matching branch wins.
  branches: z.array(z.object({
    conditions: z.array(conditionSchema).min(1),
    logic: z.enum(['and', 'or']).optional().default('and'),
  })).min(1).optional(),
  conditions: z.array(conditionSchema).optional(),
  logic: z.enum(['and', 'or']).optional().default('and'),
  // 旧版单条件格式
  operator: z.string().optional(),
  value: z.string().optional(),
});

export const ifNode: NodeDefinition<z.infer<typeof ifConfigSchema>> = {
  type: 'if',
  label: '条件',
  category: 'control',
  icon: '◇',
  configSchema: ifConfigSchema,
  async execute(input, config, ctx) {
    const cfg = config as z.infer<typeof ifConfigSchema>;
    if (cfg.branches?.length) {
      for (let index = 0; index < cfg.branches.length; index++) {
        const branch = cfg.branches[index];
        if (evaluateConditions(branch.conditions, branch.logic, input.ports, ctx)) {
          return {
            ports: {
              __branch__: index === 0 ? 'out_true' : `out_elif_${index - 1}`,
            },
          };
        }
      }

      return { ports: { __branch__: 'out_false' } };
    }

    let result = false;

    if (cfg.conditions && cfg.conditions.length > 0) {
      result = evaluateConditions(cfg.conditions, cfg.logic, input.ports, ctx);
    } else if (cfg.operator) {
      // 旧版单条件格式
      const left = resolveTemplate(input.ports['in_exec'] || '', ctx);
      const right = resolveTemplate(cfg.value || '', ctx);
      result = evaluateOperatorCondition(left, right, cfg.operator);
    } else {
      // 无条件配置，检查 in_exec 端口值
      const val = input.ports['in_exec'] || '';
      result = val === 'true' || val !== '';
    }

    return {
      ports: {
        __branch__: result ? 'out_true' : 'out_false',
      },
    };
  },
};

/**
 * 解析端口值引用
 * 支持格式：直接值、{nodeName.portName}（已由上层 replaceVariables 解析）、$nodeID:portID
 */
function resolvePortValue(ref: string, ports: Record<string, string>): string {
  // 直接是端口 ID
  if (ports[ref] !== undefined) return ports[ref];

  return ref;
}

function evaluateConditions(
  conditions: z.infer<typeof conditionSchema>[],
  logic: 'and' | 'or' | undefined,
  ports: Record<string, string>,
  ctx: Parameters<typeof resolveTemplate>[1],
): boolean {
  const results = conditions.map((condition) => {
    const left = resolvePortValue(resolveTemplate(condition.left, ctx), ports);
    const right = resolvePortValue(resolveTemplate(condition.right, ctx), ports);
    return evaluateOperatorCondition(left, right, condition.operator);
  });

  return (logic ?? 'and') === 'and' ? results.every(Boolean) : results.some(Boolean);
}
