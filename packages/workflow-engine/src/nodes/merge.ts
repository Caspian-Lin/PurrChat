import { z } from 'zod';
import type { NodeDefinition } from '../types.js';

const mergeConfigSchema = z.object({
  input_count: z.number().int().min(2).max(10).optional().default(2),
});

export const mergeNode: NodeDefinition<z.infer<typeof mergeConfigSchema>> = {
  type: 'merge',
  label: '汇聚',
  category: 'control',
  icon: '⑃',
  configSchema: mergeConfigSchema,
  async execute(_input, _config, _ctx) {
    // Merge 节点是 passthrough，由编译器处理多分支汇聚逻辑
    return {
      ports: {
        out_exec: 'true',
      },
    };
  },
};
