import { z } from 'zod';
import type { NodeDefinition } from '../types.js';
import { replaceVariables } from '../ports.js';

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

    // 变量替换（使用完整上下文）
    let result = replaceVariables(template, ctx);

    // 替换 {args} 和 {args:N}
    const rawInput = input.rawInput || '';
    const args = rawInput.trim().split(/\s+/);
    result = result.replace(/\{args:(\d+)\}/g, (_match, index: string) => {
      const i = parseInt(index, 10) - 1;
      return i >= 0 && i < args.length ? args[i]! : '';
    });
    result = result.replaceAll('{args}', rawInput.trim());

    // 替换 {变量名} 格式
    for (const [key, value] of Object.entries(ctx.variables)) {
      result = result.replaceAll(`{${key}}`, value);
    }

    return {
      ports: {
        out_output: result,
        out_exec: 'true',
      },
    };
  },
};
