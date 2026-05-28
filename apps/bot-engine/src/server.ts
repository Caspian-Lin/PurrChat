import { Hono } from 'hono';
import { cors } from 'hono/cors';
import { logger } from 'hono/logger';
import { healthRoutes } from './routes/health.js';
import { createExecuteRoutes } from './routes/execute.js';
import { createDebugRoutes } from './routes/debug.js';
import { BotExecutor } from './services/bot-executor.js';

export function createServer(): Hono {
  const app = new Hono();
  const executor = new BotExecutor();

  // 中间件
  app.use('*', logger());
  app.use('*', cors({
    origin: '*',
    allowMethods: ['GET', 'POST', 'OPTIONS'],
    allowHeaders: ['Content-Type', 'Authorization'],
  }));

  // 路由
  app.route('/', healthRoutes);
  app.route('/', createExecuteRoutes(executor));
  app.route('/', createDebugRoutes(executor));

  // 全局错误处理
  app.onError((err, c) => {
    console.error('[BotEngine] Unhandled error:', err);
    return c.json({ error: 'Internal server error' }, 500);
  });

  return app;
}
