import { z } from 'zod';
import type { NodeDefinition } from '../types.js';
import { replaceVariables } from '../ports.js';

const toolConfigSchema = z.object({
  method: z.string().optional().default('GET'),
  url: z.string(),
  headers: z.record(z.string()).optional(),
  timeout: z.number().optional().default(10000),
});

export const toolNode: NodeDefinition<z.infer<typeof toolConfigSchema>> = {
  type: 'tool',
  label: '工具',
  category: 'processing',
  icon: '🔌',
  configSchema: toolConfigSchema,
  async execute(input, config, ctx) {
    const cfg = config as any;
    const method = cfg.method || 'GET';
    let url = cfg.url || '';

    // 变量替换
    url = replaceVariables(url, {
      nodeOutputs: {},
      variables: ctx.variables,
      eventOutputs: ctx.eventOutputs,
      contextBuffer: ctx.contextBuffer,
      finalReply: '',
      nameResolver: {},
    });

    const body = input.ports['in_body'] || '';
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(cfg.headers || {}),
    };

    try {
      const resp = await fetch(url, {
        method,
        headers,
        body: method !== 'GET' && method !== 'HEAD' ? body : undefined,
        signal: AbortSignal.timeout(cfg.timeout || 10000),
      });

      const output = await resp.text();
      const status = String(resp.status);

      return {
        ports: {
          out_output: output,
          out_status: status,
          out_error: '',
          out_exec: 'true',
        },
      };
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : String(err);
      return {
        ports: {
          out_output: '',
          out_status: '0',
          out_error: errorMsg,
          out_exec: 'true',
        },
      };
    }
  },
};
