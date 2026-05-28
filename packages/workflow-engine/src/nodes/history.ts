import { z } from 'zod';
import type { NodeDefinition } from '../types.js';

const historyConfigSchema = z.object({
  count: z.number().optional().default(20),
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

    // 从输入端口读取 N
    const portCount = input.ports['in_count'];
    if (portCount) {
      const parsed = parseInt(portCount, 10);
      if (!isNaN(parsed) && parsed > 0) {
        count = parsed;
      }
    }

    // 获取最近 N 条消息
    const messages = ctx.contextBuffer.slice(-count);

    // 格式化为 prompt 字符串
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
