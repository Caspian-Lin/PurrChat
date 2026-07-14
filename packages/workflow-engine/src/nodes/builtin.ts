import { z } from 'zod';
import type { NodeDefinition } from '../types.js';
import { resolveTemplate, type ResolveContext } from '../resolver.js';

const builtinConfigSchema = z.object({
  builtin_type: z.enum(['random_number', 'haiku', 'echo', 'count', 'template']),
  min: z.number().optional(),
  max: z.number().optional(),
  integer: z.boolean().optional().default(true),
  topic: z.string().optional(),
  prefix: z.string().optional(),
  suffix: z.string().optional(),
  counter_key: z.string().optional(),
  template: z.string().optional(),
});

export const builtinNode: NodeDefinition<z.infer<typeof builtinConfigSchema>> = {
  type: 'builtin',
  label: '内置',
  category: 'processing',
  icon: '⚙',
  configSchema: builtinConfigSchema,
  async execute(input, config, ctx) {
    const cfg = config as z.infer<typeof builtinConfigSchema>;
    let output = '';

    switch (cfg.builtin_type) {
      case 'random_number':
        output = builtinRandomNumber(cfg);
        break;
      case 'haiku':
        output = builtinHaiku(cfg, input.rawInput);
        break;
      case 'echo':
        output = builtinEcho(cfg, input.rawInput);
        break;
      case 'count':
        output = builtinCount(cfg, ctx.variables);
        break;
      case 'template':
        output = builtinTemplate(cfg, ctx);
        break;
      default:
        throw new Error(`Unknown builtin type: ${cfg.builtin_type}`);
    }

    return {
      ports: {
        out_output: output,
        out_exec: 'true',
      },
    };
  },
};

function builtinRandomNumber(config: z.infer<typeof builtinConfigSchema>): string {
  const min = config.min ?? 0;
  const max = config.max ?? 100;
  const isInteger = config.integer ?? true;

  const lo = Math.min(min, max);
  const hi = Math.max(min, max);

  if (isInteger) {
    return String(Math.floor(Math.random() * (hi - lo + 1)) + lo);
  }
  return String(Math.random() * (hi - lo) + lo);
}

function builtinHaiku(config: z.infer<typeof builtinConfigSchema>, input: string): string {
  const topic = config.topic || input;

  const haikus = [
    `${topic}的光芒\n照亮了前行的路\n脚步不停歇`,
    `${topic}轻声吟唱\n风中传来回响\n万物皆有灵`,
    `静观${topic}变\n一叶落而知秋至\n心如止水`,
    `${topic}如流水\n昼夜不息向前\n奔向大海`,
  ];

  return haikus[Math.floor(Math.random() * haikus.length)]!;
}

function builtinEcho(config: z.infer<typeof builtinConfigSchema>, input: string): string {
  let result = input;
  if (config.prefix) result = config.prefix + result;
  if (config.suffix) result = result + config.suffix;
  return result;
}

function builtinCount(
  config: z.infer<typeof builtinConfigSchema>,
  variables: Record<string, string>,
): string {
  const key = config.counter_key || 'message_count';
  const current = parseInt(variables[key] || '0', 10) || 0;
  const next = current + 1;
  variables[key] = String(next);
  return String(next);
}

function builtinTemplate(
  config: z.infer<typeof builtinConfigSchema>,
  ctx: ResolveContext,
): string {
  const template = config.template;
  if (!template) throw new Error('template is empty');

  // 统一变量替换（${path} + 所有遗留格式）
  return resolveTemplate(template, ctx);
}
