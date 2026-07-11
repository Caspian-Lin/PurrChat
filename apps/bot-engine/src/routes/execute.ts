import { Hono } from 'hono';
import type { BotExecutor } from '../services/bot-executor.js';
import type { ExecuteRequest } from '../types.js';

export function createExecuteRoutes(executor: BotExecutor): Hono {
  const routes = new Hono();

  routes.post('/execute', async (c) => {
    const startTime = Date.now();
    try {
      const body = await c.req.json<ExecuteRequest>();

      if (!body.content || !body.document) {
        console.log(
          `[BotEngine:route] /execute REJECTED missing fields content=${!!body.content} document=${!!body.document}`,
        );
        return c.json({ error: 'content and document are required' }, 400);
      }

      console.log(
        `[BotEngine:route] /execute REQUEST bot=${body.bot_id} sender=${body.sender_id} ` +
          `revision=${body.revision} contentLen=${body.content?.length || 0} ` +
          `nodes=${body.document?.spec?.nodes?.length || 0}`,
      );

      const result = await executor.execute(body);
      const ms = Date.now() - startTime;

      console.log(
        `[BotEngine:route] /execute RESPONSE triggered=${result.triggered} ` +
          `replyLen=${result.reply?.length || 0} sessionActive=${result.session_active} ms=${ms}`,
      );

      return c.json(result);
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      console.error(`[BotEngine:route] /execute ERROR ms=${Date.now() - startTime}:`, message);
      return c.json({ error: message }, 500);
    }
  });

  return routes;
}
