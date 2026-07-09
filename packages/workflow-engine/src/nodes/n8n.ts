import { z } from 'zod';
import type { NodeDefinition } from '../types.js';
import { replaceVariables } from '../ports.js';

const n8nConfigSchema = z.object({
  webhook_url: z.string(),
  method: z.string().optional().default('POST'),
  auth_type: z.enum(['none', 'header', 'basic']).optional().default('none'),
  auth_header_name: z.string().optional(),
  auth_header_value: z.string().optional(),
  auth_username: z.string().optional(),
  auth_password: z.string().optional(),
});

export const n8nNode: NodeDefinition<z.infer<typeof n8nConfigSchema>> = {
  type: 'n8n',
  label: 'n8n',
  category: 'processing',
  icon: '⚡',
  configSchema: n8nConfigSchema,
  async execute(input, config, ctx) {
    const cfg = config as any;
    let webhookURL = cfg.webhook_url || '';

    // 变量替换
    webhookURL = replaceVariables(webhookURL, ctx);

    const method = cfg.method || 'POST';
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };

    // 认证
    if (cfg.auth_type === 'header' && cfg.auth_header_name) {
      headers[cfg.auth_header_name] = cfg.auth_header_value || '';
    }

    const body = input.ports['in_input'] || '';

    try {
      const fetchOptions: RequestInit = {
        method,
        headers,
        signal: AbortSignal.timeout(30000),
      };

      if (method !== 'GET' && method !== 'HEAD' && body) {
        fetchOptions.body = body;
      }

      // Basic auth
      if (cfg.auth_type === 'basic' && cfg.auth_username) {
        const auth = btoa(`${cfg.auth_username}:${cfg.auth_password || ''}`);
        headers['Authorization'] = `Basic ${auth}`;
      }

      const resp = await fetch(webhookURL, fetchOptions);
      const output = await resp.text();

      if (!resp.ok) {
        return {
          ports: {
            out_error: `HTTP ${resp.status}: ${output}`,
            out_output: '',
            out_exec: 'true',
          },
        };
      }

      return {
        ports: {
          out_output: output,
          out_error: '',
          out_exec: 'true',
        },
      };
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : String(err);
      return {
        ports: {
          out_error: errorMsg,
          out_output: '',
          out_exec: 'true',
        },
      };
    }
  },
};
