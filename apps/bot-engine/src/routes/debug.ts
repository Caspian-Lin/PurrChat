import { Hono } from 'hono';
import type { BotExecutor } from '../services/bot-executor.js';
import { NodeRegistry, allNodes, DebugRunner } from '@purrchat/workflow-engine';
import type {
  WorkflowDocument,
  RunTrace,
  SideEffectPolicy,
} from '@purrchat/workflow-types';

// 共享 DebugRunner 实例（in-memory 会话）
const registry = new NodeRegistry();
registry.registerAll(allNodes);
const debugRunner = new DebugRunner(registry);

/** 将 Go 侧的 WorkflowSpec（旧格式）转为 WorkflowDocument — 兼容旧调用方 */
function specToDocument(spec: unknown): WorkflowDocument | null {
  if (!spec || typeof spec !== 'object') return null;
  const s = spec as any;
  if (s.apiVersion && s.spec) return s as WorkflowDocument;

  // 尝试从旧格式 events/connections 转换
  if (s.events && Array.isArray(s.events)) {
    return {
      apiVersion: 'purrchat/v1',
      kind: 'WorkflowDocument',
      metadata: { name: 'debug', version: '1.0.0' },
      spec: {
        nodes: s.events.map((e: any, i: number) => ({
          id: e.id,
          type: e.type,
          name: e.name,
          key: e.key ?? `${e.type}_${i}`,
          config: e.config ?? {},
          ports: e.ports,
          position: e.position,
        })),
        connections: (s.connections ?? []).map((c: any, i: number) => ({
          id: c.id ?? `conn_${i}`,
          sourceNodeId: c.sourceNodeId,
          sourcePortId: c.sourcePortId,
          targetNodeId: c.targetNodeId,
          targetPortId: c.targetPortId,
        })),
        endConditions: s.end_conditions ?? [],
      },
    };
  }

  return null;
}

export function createDebugRoutes(_executor: BotExecutor): Hono {
  const routes = new Hono();

  // POST /debug — 启动调试运行
  routes.post('/debug', async (c) => {
    try {
      const body = await c.req.json<{
        message: string;
        document?: unknown;
        workflow_config?: unknown;
        side_effects?: SideEffectPolicy;
        step_mode?: boolean;
        sender_name?: string;
        session_id?: string;
        secrets?: Record<string, string>;
      }>();

      // 优先使用 document (新格式)，回退到 workflow_config (旧格式)
      const doc = body.document
        ? (body.document as WorkflowDocument)
        : specToDocument(body.workflow_config);

      if (!doc) {
        return c.json({ error: 'document or workflow_config is required' }, 400);
      }

      const trace = await debugRunner.run({
        document: doc,
        message: body.message,
        sideEffects: body.side_effects ?? 'mock',
        stepMode: body.step_mode ?? false,
        senderName: body.sender_name,
        sessionId: body.session_id,
        secrets: body.secrets,
      });

      const response: RunTrace & { session_id?: string } = {
        ...trace,
        session_id: trace.runId,
      };

      return c.json(response);
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      console.error('[BotEngine] Debug run error:', message);
      return c.json({ error: message }, 500);
    }
  });

  // POST /debug/step — 单步执行
  routes.post('/debug/step', async (c) => {
    try {
      const body = await c.req.json<{ session_id: string }>();
      if (!body.session_id) {
        return c.json({ error: 'session_id is required' }, 400);
      }

      const trace = await debugRunner.step(body.session_id);
      const response: RunTrace & { session_id?: string } = {
        ...trace,
        session_id: trace.runId,
      };

      return c.json(response);
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      return c.json({ error: message }, 500);
    }
  });

  // POST /debug/cancel — 取消运行
  routes.post('/debug/cancel', async (c) => {
    try {
      const body = await c.req.json<{ session_id: string }>();
      if (!body.session_id) {
        return c.json({ error: 'session_id is required' }, 400);
      }
      debugRunner.cancel(body.session_id);
      return c.json({ status: 'ok' });
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      return c.json({ error: message }, 500);
    }
  });

  // POST /debug/reset — 重置会话
  routes.post('/debug/reset', async (c) => {
    try {
      const body = await c.req.json<{ session_id?: string }>();
      if (body.session_id) {
        debugRunner.reset(body.session_id);
        _executor.destroySession(body.session_id);
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
