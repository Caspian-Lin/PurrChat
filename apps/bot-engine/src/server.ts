import { Hono } from 'hono';
import type { MiddlewareHandler } from 'hono';
import { logger } from 'hono/logger';
import { healthRoutes } from './routes/health.js';
import { createExecuteRoutes } from './routes/execute.js';
import { createDebugRoutes } from './routes/debug.js';
import { BotExecutor } from './services/bot-executor.js';

const SHARED_SECRET = process.env['BOT_ENGINE_SHARED_SECRET'] ?? '';
const ALLOWED_ORIGIN = process.env['BOT_ENGINE_CORS_ORIGIN'] ?? '';

/** 服务间鉴权中间件：校验 X-Bot-Engine-Secret header */
const sharedSecretMiddleware: MiddlewareHandler = async (c, next) => {
  // 健康检查放行
  if (c.req.path === '/health') return next();
  // 未配置 shared secret 时跳过校验（开发环境向后兼容）
  if (SHARED_SECRET === '') return next();
  const provided = c.req.header('X-Bot-Engine-Secret');
  if (provided !== SHARED_SECRET) {
    console.warn(`[BotEngine] Rejected request: invalid or missing shared secret from ${c.req.header('x-forwarded-for') ?? 'unknown'}`);
    return c.json({ error: 'Unauthorized' }, 401);
  }
  return next();
};

export function createServer(): Hono {
  const app = new Hono();
  const executor = new BotExecutor();

  // 中间件
  app.use('*', logger());
  // 收紧 CORS：仅允许配置的来源（bot-engine 只应被后端 Go 调用）
  if (ALLOWED_ORIGIN) {
    app.use('*', async (c, next) => {
      const origin = c.req.header('Origin');
      c.header('Access-Control-Allow-Origin', origin === ALLOWED_ORIGIN ? origin : '');
      c.header('Access-Control-Allow-Methods', 'GET, POST, OPTIONS');
      c.header('Access-Control-Allow-Headers', 'Content-Type, Authorization, X-Bot-Engine-Secret');
      if (c.req.method === 'OPTIONS') return c.body(null, 204);
      return next();
    });
  }
  app.use('*', sharedSecretMiddleware);

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
