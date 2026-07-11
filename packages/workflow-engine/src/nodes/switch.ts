import { z } from 'zod';
import type { NodeDefinition } from '../types.js';

const switchConfigSchema = z.object({
  cases: z.array(z.object({
    value: z.string(),
    label: z.string().optional(),
  })).optional().default([]),
});

export const switchNode: NodeDefinition<z.infer<typeof switchConfigSchema>> = {
  type: 'switch',
  label: '分支',
  category: 'control',
  icon: '⑂',
  configSchema: switchConfigSchema,
  async execute(input, config, _ctx) {
    const cfg = config as any;
    const matchValue = input.ports['in_value'] || '';
    const cases = cfg.cases || [];

    // 找到匹配的分支
    let matchedIndex = -1;
    for (let i = 0; i < cases.length; i++) {
      if (cases[i].value === matchValue) {
        matchedIndex = i;
        break;
      }
    }

    // 返回分支标记，由编译器处理跳转
    const ports: Record<string, string> = {
      __branch__: matchedIndex >= 0 ? `out_case_${matchedIndex}` : 'out_default',
    };

    // 生成每个 case 的输出端口
    for (let i = 0; i < cases.length; i++) {
      ports[`out_case_${i}`] = matchedIndex === i ? 'true' : 'false';
    }
    ports['out_default'] = matchedIndex < 0 ? 'true' : 'false';

    return { ports };
  },
};
