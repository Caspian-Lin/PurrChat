import { z } from 'zod';
import type { NodeDefinition } from '../types.js';

export const triggerNode: NodeDefinition = {
  type: 'trigger',
  label: '触发',
  category: 'trigger',
  icon: '🚀',
  configSchema: z.object({}),
  async execute(input, _config, ctx) {
    return {
      ports: {
        out_input: input.rawInput,
        out_username: ctx.variables['username'] || '',
        out_time: ctx.variables['time'] || new Date().toLocaleTimeString('zh-CN', { hour12: false }),
        out_args: '',
        out_exec: 'true',
      },
    };
  },
};
