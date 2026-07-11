import { z } from 'zod';
import type { NodeDefinition } from '../types.js';
import { resolveTemplate } from '../resolver.js';

const loopConfigSchema = z.object({
  max_iterations: z.number().int().min(1).max(100).optional().default(10),
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
    const condition = input.ports.in_condition ?? resolveTemplate((config as any).condition || '', ctx);

    return {
      ports: {
        __loop_max__: String(maxIterations),
        __loop_condition__: condition,
      },
    };
  },
};
