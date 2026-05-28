import { z } from 'zod';
import type { NodeDefinition } from '../types.js';

export const endNode: NodeDefinition = {
  type: 'end',
  label: '结束',
  category: 'output',
  icon: '⏹',
  configSchema: z.object({}),
  async execute() {
    return { ports: {} };
  },
};
