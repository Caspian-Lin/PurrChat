import { z } from 'zod';
import type { NodeDefinition } from '../types.js';
import { evaluateOperatorCondition, replaceVariables } from '../ports.js';

const conditionSchema = z.object({
  left: z.string(),
  operator: z.string(),
  right: z.string(),
});

const ifConfigSchema = z.object({
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
    let result = false;

    if (cfg.conditions && cfg.conditions.length > 0) {
      // 新版多条件格式
      const logic = cfg.logic || 'and';
      const results = cfg.conditions.map((c: { left: string; operator: string; right: string }) => {
        // 先解析 {节点名.端口名} / $变量 / $nodeId:portId 引用，再查输入端口
        const left = resolvePortValue(replaceVariables(c.left, ctx), input.ports);
        const right = resolvePortValue(replaceVariables(c.right, ctx), input.ports);
        return evaluateOperatorCondition(left, right, c.operator);
      });

      result = logic === 'and'
        ? results.every(Boolean)
        : results.some(Boolean);
    } else if (cfg.operator) {
      // 旧版单条件格式
      const left = replaceVariables(input.ports['in_exec'] || '', ctx);
      const right = replaceVariables(cfg.value || '', ctx);
      result = evaluateOperatorCondition(left, right, cfg.operator);
    } else {
      // 无条件配置，检查 in_exec 端口值
      const val = input.ports['in_exec'] || '';
      result = val === 'true' || val !== '';
    }

    return {
      ports: {
        __branch__: result ? 'true' : 'false',
        out_exec: 'true',
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
