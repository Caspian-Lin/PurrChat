import { z } from 'zod';
import type { NodeDefinition } from '../types.js';

/** 硬上限：防止无界内存和 token 消耗 */
const MAX_HISTORY_LIMIT = 100;

const historyConfigSchema = z.object({
  count: z.number().optional().default(20),
  message_types: z.array(z.string()).optional(),
  sort_order: z.enum(['asc', 'desc']).optional().default('asc'),
});

export const historyNode: NodeDefinition<z.infer<typeof historyConfigSchema>> = {
  type: 'history',
  label: '历史消息',
  category: 'processing',
  icon: '📜',
  configSchema: historyConfigSchema,
  async execute(input, config, ctx) {
    const cfg = config as any;
    let count = cfg.count || 20;

    const portCount = input.ports['in_count'];
    if (portCount) {
      const parsed = parseInt(portCount, 10);
      if (!isNaN(parsed) && parsed > 0) {
        count = parsed;
      }
    }

    count = Math.min(count, MAX_HISTORY_LIMIT);

    let messages = [...ctx.history];

    const messageTypes = cfg.message_types;
    if (Array.isArray(messageTypes) && messageTypes.length > 0) {
      const allowed = new Set(messageTypes);
      messages = messages.filter((m) => allowed.has(m.role));
    }

    messages = messages.slice(-count);

    if (cfg.sort_order === 'desc') {
      messages = [...messages].reverse();
    }

    const historyPrompt = messages
      .map((m) => `[${m.role}]: ${m.content}`)
      .join('\n');

    return {
      ports: {
        out_history: historyPrompt,
        out_exec: 'true',
      },
    };
  },
};
