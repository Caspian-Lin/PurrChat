import { z } from 'zod';
import type { NodeDefinition } from '../types.js';
import { resolveTemplate } from '../resolver.js';

export const replyNode: NodeDefinition = {
  type: 'reply',
  label: '回复',
  category: 'output',
  icon: '💬',
  configSchema: z.object({
    template: z.string().optional(),
  }),
  async execute(input, config, ctx) {
    // 优先从 in_content 端口获取，否则从 config.template 获取
    let content = input.ports['in_content'] || '';
    if (!content && (config as any).template) {
      content = (config as any).template;
    }

    // 统一变量替换（支持 ${path} 规范格式和所有遗留格式）
    content = resolveTemplate(content, ctx);

    return {
      ports: {
        __reply__: content,
        out_exec: 'true',
      },
    };
  },
};
