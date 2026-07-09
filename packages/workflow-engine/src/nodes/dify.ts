import { z } from 'zod';
import type { NodeDefinition } from '../types.js';
import { replaceVariables } from '../ports.js';

const difyConfigSchema = z.object({
  api_base: z.string(),
  api_key: z.string(),
  app_type: z.enum(['workflow', 'chatflow']).optional().default('workflow'),
  response_mode: z.enum(['blocking', 'streaming']).optional().default('blocking'),
  inputs_mapping: z.string().optional(),
});

export const difyNode: NodeDefinition<z.infer<typeof difyConfigSchema>> = {
  type: 'dify',
  label: 'Dify',
  category: 'processing',
  icon: '🔮',
  configSchema: difyConfigSchema,
  async execute(input, config, ctx) {
    const cfg = config as any;
    const apiBase = cfg.api_base?.replace(/\/$/, '');
    const apiKey = cfg.api_key;
    const appType = cfg.app_type || 'workflow';

    if (!apiBase || !apiKey) {
      return {
        ports: {
          out_error: 'api_base and api_key are required',
          out_output: '',
          out_exec: 'true',
        },
      };
    }

    // 解析输入
    let inputs: Record<string, any> = {};
    const inputsMapping = cfg.inputs_mapping;
    if (inputsMapping) {
      try {
        const mapping = JSON.parse(inputsMapping);
        for (const [key, ref] of Object.entries(mapping)) {
          inputs[key] = replaceVariables(ref as string, ctx);
        }
      } catch {
        // 忽略解析错误
      }
    }

    // 如果映射为空，从 in_input 端口获取
    if (Object.keys(inputs).length === 0) {
      const inputVal = input.ports['in_input'] || '';
      if (inputVal) {
        inputs = { query: inputVal };
      }
    }

    // 构建请求
    let endpoint: string;
    let reqBody: Record<string, any>;

    if (appType === 'chatflow') {
      endpoint = `${apiBase}/v1/chat-messages`;
      reqBody = {
        query: inputs.query || '',
        inputs,
        response_mode: cfg.response_mode || 'blocking',
        user: 'purrchat',
      };
      // 多轮上下文
      const convKey = `dify_conversation_${config.__nodeId__}`;
      const convId = ctx.variables[convKey];
      if (convId) {
        reqBody.conversation_id = convId;
      }
    } else {
      endpoint = `${apiBase}/v1/workflows/run`;
      reqBody = {
        inputs,
        response_mode: cfg.response_mode || 'blocking',
        user: 'purrchat',
      };
    }

    try {
      const resp = await fetch(endpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${apiKey}`,
        },
        body: JSON.stringify(reqBody),
        signal: AbortSignal.timeout(60000),
      });

      if (!resp.ok) {
        const errorBody = await resp.text();
        return {
          ports: {
            out_error: `HTTP ${resp.status}: ${errorBody}`,
            out_output: '',
            out_exec: 'true',
          },
        };
      }

      const data = await resp.json() as any;

      // 提取输出
      let output = '';
      if (appType === 'chatflow') {
        output = data.answer || '';
        // 保存 conversation_id
        if (data.conversation_id) {
          const convKey = `dify_conversation_${config.__nodeId__}`;
          ctx.variables[convKey] = data.conversation_id;
        }
      } else {
        output = data.data?.outputs?.text || data.data?.outputs?.result || '';
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
