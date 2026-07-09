import type {
  MechanismConfig,
  Mechanism,
  TriggerSpec,
  TriggerRule,
  WorkflowSpec,
} from '@purrchat/workflow-types';
import { WorkflowRuntime, Compiler, NodeRegistry, allNodes } from '@purrchat/workflow-engine';
import type { Blueprint } from '@purrchat/workflow-engine';
import type { ExecuteRequest, ExecuteResponse } from '../types.js';

export class BotExecutor {
  private registry: NodeRegistry;
  private compiler: Compiler;
  private runtime: WorkflowRuntime;
  private sessions = new Map<string, { botId: string; conversationId: string }>();

  constructor() {
    this.registry = new NodeRegistry();
    this.registry.registerAll(allNodes);
    this.compiler = new Compiler(this.registry);
    this.runtime = new WorkflowRuntime(this.compiler);
  }

  async execute(req: ExecuteRequest): Promise<ExecuteResponse> {
    const startTime = Date.now();
    const noMatch: ExecuteResponse = {
      reply: '',
      session_active: false,
      triggered: false,
      execution_ms: 0,
    };

    const config = this.normalizeConfig(req.mechanism_config);
    const enabledCount = config.mechanisms.filter((m) => m.enabled).length;

    console.log(
      `[BotExecutor] START botID=${req.bot_id} senderID=${req.sender_id} ` +
        `senderName=${req.sender_name} mechanisms=${config.mechanisms.length} ` +
        `enabled=${enabledCount} contentLen=${req.content?.length || 0}`,
    );

    if (req.content) {
      const preview = req.content.length > 80 ? req.content.slice(0, 80) + '...' : req.content;
      console.log(`[BotExecutor]   content preview: "${preview}"`);
    }

    for (const mech of config.mechanisms) {
      if (!mech.enabled) continue;

      const evalStart = Date.now();
      const matched = this.evaluateTrigger(mech.trigger, req.content);
      const evalMs = Date.now() - evalStart;

      console.log(
        `[BotExecutor] TRIGGER mech=${mech.id} name="${mech.name}" ` +
          `type=${mech.trigger.type} matched=${matched} evalMs=${evalMs}`,
      );

      if (!matched) continue;

      // 触发匹配成功，执行回复
      const execStart = Date.now();
      let result: ExecuteResponse;

      if (mech.reply.type === 'workflow' && mech.reply.workflow) {
        console.log(`[BotExecutor] WORKFLOW_START mech=${mech.id}`);
        result = await this.executeWorkflow(req, mech.reply.workflow);
        console.log(
          `[BotExecutor] WORKFLOW_END mech=${mech.id} ` +
            `sessionActive=${result.session_active} replyLen=${result.reply?.length || 0}`,
        );
      } else if (mech.reply.type === 'predefined' || mech.reply.type === 'llm') {
        const blueprint = this.compileSimpleMechanism(mech);
        if (blueprint) {
          console.log(`[BotExecutor] SIMPLE_FLOW_START mech=${mech.id} type=${mech.reply.type}`);
          result = await this.executeSimpleFlow(req, blueprint);
          console.log(
            `[BotExecutor] SIMPLE_FLOW_END mech=${mech.id} replyLen=${result.reply?.length || 0}`,
          );
        } else {
          console.warn(`[BotExecutor] COMPILE_FAILED mech=${mech.id} type=${mech.reply.type}`);
          result = { reply: '', session_active: false, triggered: true, mechanism_id: mech.id, mechanism_name: mech.name, reply_type: mech.reply.type };
        }
      } else {
        console.warn(`[BotExecutor] UNKNOWN_REPLY_TYPE mech=${mech.id} type=${mech.reply.type}`);
        result = { reply: '', session_active: false, triggered: true, mechanism_id: mech.id, mechanism_name: mech.name, reply_type: mech.reply.type };
      }

      const totalMs = Date.now() - startTime;
      result.triggered = true;
      result.mechanism_id = mech.id;
      result.mechanism_name = mech.name;
      result.reply_type = mech.reply.type;
      result.execution_ms = totalMs;

      console.log(
        `[BotExecutor] DONE botID=${req.bot_id} replyLen=${result.reply?.length || 0} ` +
          `triggered=true mechanism=${mech.id} totalMs=${totalMs}`,
      );
      return result;
    }

    const totalMs = Date.now() - startTime;
    console.log(`[BotExecutor] NO_MATCH botID=${req.bot_id} totalMs=${totalMs}`);
    noMatch.execution_ms = totalMs;
    return noMatch;
  }

  private async executeWorkflow(
    req: ExecuteRequest,
    spec: WorkflowSpec,
  ): Promise<ExecuteResponse> {
    const sessionId = `${req.conversation_id}:${req.bot_id}`;
    const blueprint = this.specToBlueprint(spec);

    const senderInfo = {
      senderName: req.sender_name,
      senderId: req.sender_id,
      conversationId: req.conversation_id,
    };

    if (this.runtime.hasSession(sessionId)) {
      console.log(`[BotExecutor]   session EXISTS sessionId=${sessionId}`);
      const result = await this.runtime.sendMessage(sessionId, req.content, senderInfo);
      if (!result.sessionActive) {
        this.sessions.delete(sessionId);
      }
      return { reply: result.reply, session_active: result.sessionActive, session_id: sessionId, triggered: true };
    }

    console.log(`[BotExecutor]   session NEW sessionId=${sessionId}`);
    this.runtime.createSession(sessionId, blueprint, {
      senderName: req.sender_name,
      senderId: req.sender_id,
      conversationId: req.conversation_id,
      contextBuffer: req.context_messages,
    });

    const result = await this.runtime.sendMessage(sessionId, req.content, senderInfo);
    if (result.sessionActive) {
      this.sessions.set(sessionId, {
        botId: req.bot_id,
        conversationId: req.conversation_id,
      });
    }

    return { reply: result.reply, session_active: result.sessionActive, session_id: sessionId, triggered: true };
  }

  private async executeSimpleFlow(
    req: ExecuteRequest,
    blueprint: Blueprint,
  ): Promise<ExecuteResponse> {
    const result = await this.runtime.execute(blueprint, {
      rawInput: req.content,
      senderName: req.sender_name,
      senderId: req.sender_id,
      conversationId: req.conversation_id,
      contextBuffer: req.context_messages,
    });

    return { reply: result.reply, session_active: false, triggered: true };
  }

  // ─── 触发评估 ──────────────────────────────────────────────

  evaluateTrigger(trigger: TriggerSpec, content: string): boolean {
    if (trigger.type === 'probability') {
      const probability = trigger.probability || 0;
      const result = Math.random() < probability;
      console.log(`[BotExecutor]   PROBABILITY p=${probability} result=${result}`);
      return result;
    }

    if (trigger.type === 'rule' && trigger.rules) {
      return this.evaluateRules(trigger.rules, content);
    }

    return false;
  }

  private evaluateRules(rules: TriggerRule[], content: string): boolean {
    if (rules.length === 0) {
      return true; // 无规则时默认触发（与 Go engine 一致）
    }
    for (const rule of rules) {
      if (this.evaluateRule(rule, content)) return true;
    }
    return false;
  }

  private evaluateRule(rule: TriggerRule, content: string): boolean {
    const text = rule.case_sensitive ? content : content.toLowerCase();
    const pattern = rule.case_sensitive ? rule.pattern : rule.pattern.toLowerCase();

    let matched = false;
    switch (rule.type) {
      case 'keyword':
        matched = text.includes(pattern);
        break;
      case 'equals':
        matched = text === pattern;
        break;
      case 'command':
        matched = text.startsWith(pattern);
        break;
      case 'regex':
        try {
          const flags = rule.case_sensitive ? '' : 'i';
          matched = new RegExp(rule.pattern, flags).test(content);
        } catch {
          matched = false;
        }
        break;
    }

    const patternPreview = rule.pattern.length > 50 ? rule.pattern.slice(0, 50) + '...' : rule.pattern;
    console.log(`[BotExecutor]   RULE type=${rule.type} pattern="${patternPreview}" matched=${matched}`);
    return matched;
  }

  // ─── 简单机制编译 ──────────────────────────────────────────

  compileSimpleMechanism(mech: Mechanism): Blueprint | null {
    if (mech.reply.type === 'predefined' && mech.reply.predefined) {
      const config = mech.reply.predefined;
      let template = '';

      if (config.mode === 'fixed' && config.replies?.length) {
        template = config.replies[0] || '';
      } else if (config.mode === 'random' && config.replies?.length) {
        template = config.replies[Math.floor(Math.random() * config.replies.length)] || '';
      } else if (config.mode === 'template') {
        template = config.template || '';
      }

      return {
        nodes: [
          { id: 'compiled_trigger', type: 'trigger', name: '触发', config: {} },
          { id: 'compiled_reply', type: 'reply', name: '回复', config: { template } },
          { id: 'compiled_end', type: 'end', name: '结束', config: {} },
        ],
        connections: [
          { id: 'c1', sourceNodeId: 'compiled_trigger', sourcePortId: 'out_exec', targetNodeId: 'compiled_reply', targetPortId: 'in_exec' },
          { id: 'c2', sourceNodeId: 'compiled_reply', sourcePortId: 'out_exec', targetNodeId: 'compiled_end', targetPortId: 'in_exec' },
        ],
        endConditions: [],
      };
    }

    if (mech.reply.type === 'llm' && mech.reply.llm) {
      const config = mech.reply.llm;
      return {
        nodes: [
          { id: 'compiled_trigger', type: 'trigger', name: '触发', config: {} },
          {
            id: 'compiled_llm',
            type: 'llm',
            name: 'LLM',
            config: {
              api_url: config.api_url,
              api_key: config.api_key,
              model: config.model,
              system_prompt: config.system_prompt,
              temperature: config.temperature,
              max_tokens: config.max_tokens,
              context_window: config.context_window,
            },
          },
          { id: 'compiled_reply', type: 'reply', name: '回复', config: {} },
          { id: 'compiled_end', type: 'end', name: '结束', config: {} },
        ],
        connections: [
          { id: 'c1', sourceNodeId: 'compiled_trigger', sourcePortId: 'out_exec', targetNodeId: 'compiled_llm', targetPortId: 'in_prompt' },
          { id: 'c2', sourceNodeId: 'compiled_llm', sourcePortId: 'out_exec', targetNodeId: 'compiled_reply', targetPortId: 'in_content' },
          { id: 'c3', sourceNodeId: 'compiled_reply', sourcePortId: 'out_exec', targetNodeId: 'compiled_end', targetPortId: 'in_exec' },
        ],
        endConditions: [],
      };
    }

    return null;
  }

  // ─── 工具方法 ──────────────────────────────────────────────

  private specToBlueprint(spec: WorkflowSpec): Blueprint {
    return {
      nodes: spec.events.map((e) => ({
        id: e.id,
        type: e.type,
        name: e.name,
        config: e.config,
        ports: e.ports,
        position: e.position,
      })),
      connections: spec.connections || [],
      endConditions: spec.end_conditions || [],
    };
  }

  private normalizeConfig(config: MechanismConfig): MechanismConfig {
    const mechanisms = config.mechanisms.map((m) => {
      const reply = { ...m.reply };

      // 归一化旧版 special_mode
      if (reply.type === 'special_mode') {
        reply.type = 'workflow';
      }
      if ((reply as any).special_mode && !reply.workflow) {
        reply.workflow = (reply as any).special_mode;
        delete (reply as any).special_mode;
      }

      return { ...m, reply };
    });

    return { mechanisms };
  }

  destroySession(sessionId: string): void {
    this.runtime.destroySession(sessionId);
    this.sessions.delete(sessionId);
  }
}
