import { z } from 'zod';
import type { NodeDefinition } from '../types.js';

const loopConfigSchema = z.object({
  max_iterations: z.number().optional().default(10),
  condition: z.string().optional(),
});

export const loopNode: NodeDefinition<z.infer<typeof loopConfigSchema>> = {
  type: 'loop',
  label: '循环',
  category: 'control',
  icon: '↻',
  configSchema: loopConfigSchema,
  async execute(input, config, ctx) {
    const maxIterations = (config as any).max_iterations || 10;
    const condition = (config as any).condition || '';

    // 返回循环元数据，由编译器处理循环逻辑
    return {
      ports: {
        out_exec: 'true',
        __loop_max__: String(maxIterations),
        __loop_condition__: condition,
      },
    };
  },
};
