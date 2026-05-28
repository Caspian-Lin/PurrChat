import { Hono } from 'hono';
import type { BotExecutor } from '../services/bot-executor.js';
import type { ExecuteRequest } from '../types.js';

export function createExecuteRoutes(executor: BotExecutor): Hono {
  const routes = new Hono();

  routes.post('/execute', async (c) => {
    try {
      const body = await c.req.json<ExecuteRequest>();

      // 验证必填字段
      if (!body.content || !body.mechanism_config) {
        return c.json({ error: 'content and mechanism_config are required' }, 400);
      }

      const result = await executor.execute(body);
      return c.json(result);
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      console.error('[BotEngine] Execute error:', message);
      return c.json({ error: message }, 500);
    }
  });

  return routes;
}
