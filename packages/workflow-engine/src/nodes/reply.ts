import { z } from 'zod';
import type { NodeDefinition } from '../types.js';
import { replaceVariables } from '../ports.js';

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

    // 变量替换（使用完整上下文，含 nodeOutputs / nameResolver）
    content = replaceVariables(content, ctx);

    // 替换 {args} 变量
    content = replaceArgsVars(content, input.rawInput);

    return {
      ports: {
        __reply__: content,
        out_exec: 'true',
      },
    };
  },
};

/**
 * 替换模板中的 {args} 和 {args:N} 变量
 * {args} = 完整输入
 * {args:N} = 输入的第 N 个单词（从 1 开始）
 */
function replaceArgsVars(template: string, input: string): string {
  const args = input.trim().split(/\s+/);
  let result = template;

  // 替换 {args:N}
  result = result.replace(/\{args:(\d+)\}/g, (_match, index: string) => {
    const i = parseInt(index, 10) - 1;
    return i >= 0 && i < args.length ? args[i] : '';
  });

  // 替换 {args}
  result = result.replace(/\{args\}/g, input.trim());

  return result;
}
