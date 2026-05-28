import { serve } from '@hono/node-server';
import { createServer } from './server.js';

const PORT = parseInt(process.env['PORT'] || '3001', 10);
const app = createServer();

serve({
  fetch: app.fetch,
  port: PORT,
}, (info) => {
  console.log(`[BotEngine] Server running on http://localhost:${info.port}`);
});
