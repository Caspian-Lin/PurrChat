import type { WorkflowDocument, RunTrace, RunTraceStatus } from '@purrchat/workflow-types';
import {
  WorkflowRuntime,
  Compiler,
  NodeRegistry,
  allNodes,
  validateWorkflowDocument,
  toBlueprint,
  WorkflowExecutionError,
} from '@purrchat/workflow-engine';
import type { Blueprint } from '@purrchat/workflow-engine';
import type { ExecuteRequest, ExecuteResponse } from '../types.js';

export class BotExecutor {
  private registry: NodeRegistry;
  private compiler: Compiler;
  private runtime: WorkflowRuntime;

  constructor() {
    this.registry = new NodeRegistry();
    this.registry.registerAll(allNodes);
    this.compiler = new Compiler(this.registry);
    this.runtime = new WorkflowRuntime(this.compiler);
  }

  async execute(req: ExecuteRequest): Promise<ExecuteResponse> {
    const startTime = Date.now();
    const runId = crypto.randomUUID();

    const buildResponse = (
      reply: string,
      triggered: boolean,
      sessionActive: boolean,
      status: RunTraceStatus,
      trace?: RunTrace,
      sessionId?: string,
    ): ExecuteResponse => ({
      run_id: runId,
      reply,
      session_active: sessionActive,
      session_id: sessionId,
      triggered,
      execution_ms: Date.now() - startTime,
      revision: req.revision,
      status,
      trace,
    });

    // 1. 校验文档
    const validationResult = validateWorkflowDocument(req.document, this.registry);
    if (!validationResult.valid) {
      const errors = validationResult.issues
        .filter((i) => i.level === 'error')
        .map((i) => `${i.code}: ${i.message}`)
        .join('; ');
      console.error(`[BotExecutor] Document validation failed bot=${req.bot_id} revision=${req.revision}: ${errors}`);
      return buildResponse('', false, false, 'error');
    }

    console.log(
      `[BotExecutor] START bot=${req.bot_id} runId=${runId} revision=${req.revision} sender=${req.sender_name} ` +
        `contentLen=${req.content?.length || 0} nodes=${req.document.spec.nodes.length}`,
    );

    if (req.content) {
      const preview = req.content.length > 80 ? req.content.slice(0, 80) + '...' : req.content;
      console.log(`[BotExecutor]   content preview: "${preview}"`);
    }

    // 2. 编译为 Blueprint 并执行
    const blueprint = toBlueprint(req.document);
    const sessionId = `${req.conversation_id}:${req.bot_id}`;

    const senderInfo = {
      senderName: req.sender_name,
      senderId: req.sender_id,
      conversationId: req.conversation_id,
    };

    try {
      // 如果已有活跃会话，通过 sendMessage 推进
      if (this.runtime.hasSession(sessionId)) {
        console.log(`[BotExecutor]   session EXISTS sessionId=${sessionId}`);
        const result = await this.runtime.sendMessage(sessionId, req.content, {
          ...senderInfo,
          runId,
        });

        const totalMs = Date.now() - startTime;
        const status: RunTraceStatus = result.status === 'error' ? 'error' : 'completed';
        console.log(
          `[BotExecutor] DONE bot=${req.bot_id} runId=${runId} replyLen=${result.reply?.length || 0} ` +
            `sessionActive=${result.sessionActive} ms=${totalMs}`,
        );
        return buildResponse(result.reply, true, result.sessionActive, status, result.trace, sessionId);
      }

      // 检查是否有 wait 节点（需要多轮会话）
      const hasWaitNodes = this.hasWaitNodes(blueprint);
      console.log(`[BotExecutor]   session NEW sessionId=${sessionId} hasWait=${hasWaitNodes}`);

      if (hasWaitNodes) {
        // 创建持久化会话
        this.runtime.createSession(sessionId, blueprint, {
          senderName: req.sender_name,
          senderId: req.sender_id,
          conversationId: req.conversation_id,
          contextBuffer: req.context_messages,
          grantedCapabilities: req.granted_capabilities,
          secrets: req.secrets,
        });

        const result = await this.runtime.sendMessage(sessionId, req.content, {
          ...senderInfo,
          runId,
        });

        const totalMs = Date.now() - startTime;
        const status: RunTraceStatus = result.status === 'error' ? 'error' : 'completed';
        console.log(
          `[BotExecutor] DONE bot=${req.bot_id} runId=${runId} replyLen=${result.reply?.length || 0} ` +
            `sessionActive=${result.sessionActive} ms=${totalMs}`,
        );
        return buildResponse(result.reply, true, result.sessionActive, status, result.trace, sessionId);
      }

      // 无 wait 节点：一次性执行
      const result = await this.runtime.execute(blueprint, {
        rawInput: req.content,
        senderName: req.sender_name,
        senderId: req.sender_id,
        conversationId: req.conversation_id,
        contextBuffer: req.context_messages,
        grantedCapabilities: req.granted_capabilities,
        secrets: req.secrets,
        runId,
      });

      const totalMs = Date.now() - startTime;
      console.log(
        `[BotExecutor] DONE bot=${req.bot_id} runId=${runId} replyLen=${result.reply?.length || 0} ms=${totalMs}`,
      );

      return buildResponse(result.reply, true, false, 'completed', result.trace);
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      const trace = err instanceof WorkflowExecutionError ? err.trace : undefined;
      console.error(`[BotExecutor] EXECUTION_ERROR bot=${req.bot_id} runId=${runId}:`, message);
      return buildResponse('', true, false, 'error', trace);
    }
  }

  private hasWaitNodes(blueprint: Blueprint): boolean {
    return blueprint.nodes.some((n) => n.type === 'wait');
  }

  destroySession(sessionId: string): void {
    this.runtime.destroySession(sessionId);
  }
}
