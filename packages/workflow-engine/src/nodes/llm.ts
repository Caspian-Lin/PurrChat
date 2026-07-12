import { z } from 'zod';
import type { NodeDefinition } from '../types.js';
import { resolveTemplate } from '../resolver.js';

const llmConfigSchema = z.object({
  api_url: z.string(),
  api_key: z.string().optional().default(''),
  model: z.string(),
  system_prompt: z.string().optional().default(''),
  temperature: z.number().optional().default(0.7),
  max_tokens: z.number().optional().default(2048),
  context_window: z.number().optional().default(20),
  context_scope: z.string().optional(),
  timeout: z.number().optional().default(30000),
});

export const llmNode: NodeDefinition<z.infer<typeof llmConfigSchema>> = {
  type: 'llm',
  label: 'LLM',
  category: 'processing',
  icon: '🧠',
  configSchema: llmConfigSchema,
  async execute(input, config, ctx) {
    const prompt = input.ports['in_prompt'] || input.rawInput;

    // 构建消息列表
    const messages: Array<{ role: string; content: string }> = [];

    if (config.system_prompt) {
      // LLM 节点的 system_prompt 也支持统一变量解析
      messages.push({ role: 'system', content: resolveTemplate(config.system_prompt, ctx) });
    }

    // 截取上下文窗口
    const windowSize = config.context_window || 20;
    const contextMsgs = ctx.contextBuffer.slice(-windowSize);
    for (const msg of contextMsgs) {
      messages.push({ role: msg.role, content: msg.content });
    }

    // 添加当前用户消息
    messages.push({ role: 'user', content: prompt });

    // 构建请求体
    const reqBody: Record<string, any> = {
      model: config.model,
      messages,
      max_tokens: config.max_tokens,
    };
    if (config.temperature > 0) {
      reqBody.temperature = config.temperature;
    }

    // 发送请求
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };
    if (config.api_key) {
      headers['Authorization'] = `Bearer ${config.api_key}`;
    }

    try {
      const resp = await fetch(config.api_url, {
        method: 'POST',
        headers,
        body: JSON.stringify(reqBody),
        signal: AbortSignal.timeout(config.timeout || 30000),
      });

      if (!resp.ok) {
        const body = await resp.text();
        return {
          ports: {
            out_output: '',
            out_error: `[provider] HTTP ${resp.status}: ${body}`,
            out_exec: 'true',
          },
        };
      }

      const data = await resp.json() as {
        choices?: Array<{ message?: { content?: string } }>;
      };

      const output = data.choices?.[0]?.message?.content || '';

      return {
        ports: {
          out_output: output,
          out_error: '',
          out_exec: 'true',
        },
      };
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : String(err);
      const category = errorMsg.includes('timeout') || errorMsg.includes('aborted')
        ? 'timeout'
        : 'network';
      return {
        ports: {
          out_output: '',
          out_error: `[${category}] ${errorMsg}`,
          out_exec: 'true',
        },
      };
    }
  },
};
