import { z } from 'zod';
import type { NodeDefinition } from '../types.js';

export const waitNode: NodeDefinition = {
  type: 'wait',
  label: '等待',
  category: 'control',
  icon: '⏳',
  configSchema: z.object({}),
  async execute(input, _config, _ctx) {
    // Phase 1：Go 兼容行为，读取当前消息
    // Phase 2：升级为 XState 原生事件等待
    return {
      ports: {
        out_user_input: input.rawInput,
        out_exec: 'true',
      },
    };
  },
};
