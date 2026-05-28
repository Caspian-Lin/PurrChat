import { Hono } from 'hono';
import type { BotExecutor } from '../services/bot-executor.js';

export function createDebugRoutes(executor: BotExecutor): Hono {
  const routes = new Hono();

  routes.post('/debug', async (c) => {
    try {
      const body = await c.req.json();
      // Phase 1: 基础调试支持
      // 完整的调试功能在后续版本实现
      return c.json({
        error: 'Debug endpoint not yet implemented',
        message: 'Full debug support will be available in a future release',
      }, 501);
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      return c.json({ error: message }, 500);
    }
  });

  routes.post('/debug/step', async (c) => {
    return c.json({
      error: 'Debug step endpoint not yet implemented',
    }, 501);
  });

  routes.post('/debug/reset', async (c) => {
    try {
      const body = await c.req.json<{ session_id?: string }>();
      if (body.session_id) {
        executor.destroySession(body.session_id);
        return c.json({ status: 'ok' });
      }
      return c.json({ error: 'session_id is required' }, 400);
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      return c.json({ error: message }, 500);
    }
  });

  return routes;
}
