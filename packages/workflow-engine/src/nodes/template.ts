import { z } from 'zod';
import type { NodeDefinition } from '../types.js';
import { resolveTemplate } from '../resolver.js';

const templateConfigSchema = z.object({
  template: z.string().optional(),
});

export const templateNode: NodeDefinition<z.infer<typeof templateConfigSchema>> = {
  type: 'template',
  label: '模板',
  category: 'processing',
  icon: '📋',
  configSchema: templateConfigSchema,
  async execute(input, config, ctx) {
    const cfg = config as any;
    let template = cfg.template || '';

    // 从输入端口获取模板（如果有连接）
    const inputVal = input.ports['in_input'] || '';
    if (inputVal) {
      template = inputVal;
    }

    if (!template) {
      return {
        ports: {
          out_output: '',
          out_exec: 'true',
        },
      };
    }

    // 统一变量替换（${path} + 所有遗留格式由同一 resolver 处理）
    const result = resolveTemplate(template, ctx);

    return {
      ports: {
        out_output: result,
        out_exec: 'true',
      },
    };
  },
};
