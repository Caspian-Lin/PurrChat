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
    const config = this.normalizeConfig(req.mechanism_config);

    // 遍历机制列表，首个匹配即响应
    for (const mech of config.mechanisms) {
      if (!mech.enabled) continue;

      const matched = this.evaluateTrigger(mech.trigger, req.content);
      if (!matched) continue;

      // 触发匹配成功
      if (mech.reply.type === 'workflow' && mech.reply.workflow) {
        return this.executeWorkflow(req, mech.reply.workflow);
      }

      if (mech.reply.type === 'predefined' || mech.reply.type === 'llm') {
        const blueprint = this.compileSimpleMechanism(mech);
        if (blueprint) {
          return this.executeSimpleFlow(req, blueprint);
        }
      }
    }

    return { reply: '', session_active: false };
  }

  private async executeWorkflow(
    req: ExecuteRequest,
    spec: WorkflowSpec,
  ): Promise<ExecuteResponse> {
    const sessionId = `${req.conversation_id}:${req.bot_id}`;
    const blueprint = this.specToBlueprint(spec);

    if (this.runtime.hasSession(sessionId)) {
      // 已有会话，发送消息
      const reply = await this.runtime.sendMessage(sessionId, req.content);
      return { reply, session_active: true, session_id: sessionId };
    }

    // 创建新会话
    this.runtime.createSession(sessionId, blueprint, {
      username: req.sender_name,
      contextBuffer: req.context_messages,
    });

    const reply = await this.runtime.sendMessage(sessionId, req.content);
    this.sessions.set(sessionId, {
      botId: req.bot_id,
      conversationId: req.conversation_id,
    });

    return { reply, session_active: true, session_id: sessionId };
  }

  private async executeSimpleFlow(
    req: ExecuteRequest,
    blueprint: Blueprint,
  ): Promise<ExecuteResponse> {
    const reply = await this.runtime.execute(blueprint, {
      rawInput: req.content,
      username: req.sender_name,
      contextBuffer: req.context_messages,
      variables: {
        time: new Date().toLocaleTimeString('zh-CN', { hour12: false }),
      },
    });

    return { reply, session_active: false };
  }

  // ─── 触发评估 ──────────────────────────────────────────────

  evaluateTrigger(trigger: TriggerSpec, content: string): boolean {
    if (trigger.type === 'probability') {
      return Math.random() < (trigger.probability || 0);
    }

    if (trigger.type === 'rule' && trigger.rules) {
      return this.evaluateRules(trigger.rules, content);
    }

    return false;
  }

  private evaluateRules(rules: TriggerRule[], content: string): boolean {
    for (const rule of rules) {
      if (this.evaluateRule(rule, content)) return true;
    }
    return false;
  }

  private evaluateRule(rule: TriggerRule, content: string): boolean {
    const text = rule.case_sensitive ? content : content.toLowerCase();
    const pattern = rule.case_sensitive ? rule.pattern : rule.pattern.toLowerCase();

    switch (rule.type) {
      case 'keyword':
        return text.includes(pattern);
      case 'equals':
        return text === pattern;
      case 'command':
        return text.startsWith(pattern);
      case 'regex':
        try {
          const flags = rule.case_sensitive ? '' : 'i';
          return new RegExp(rule.pattern, flags).test(content);
        } catch {
          return false;
        }
      default:
        return false;
    }
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
      if (reply.special_mode && !reply.workflow) {
        reply.workflow = reply.special_mode;
        delete reply.special_mode;
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
