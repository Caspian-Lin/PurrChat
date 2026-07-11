import { setup, assign, fromPromise, type AnyStateMachine } from 'xstate';
import type {
  Blueprint,
  BlueprintNode,
  BlueprintConnection,
  ExecutionContext,
  ActorInput,
  NodeContext,
  UserMessageEvent,
} from './types.js';
import type { NodeRegistry } from './registry.js';
import { getMissingCapabilities } from './capabilities.js';
import { resolveSecrets, checkSecretCapability } from './secrets.js';
import { resolveControlFlowRoute } from './control-flow.js';

/** 终态：节点无后继连接时统一进入 */
const DONE_STATE = '__done';
/** 错误终态：节点 invoke 抛错时进入 */
const ERROR_STATE = '__error';

export class Compiler {
  constructor(private registry: NodeRegistry) {}

  compile(blueprint: Blueprint): AnyStateMachine {
    const triggerNode = blueprint.nodes.find((n) => n.type === 'trigger');
    if (!triggerNode) throw new Error('Blueprint must have a trigger node');

    const nameResolver = this.buildNameResolver(blueprint);
    const nodeKeyMap = this.buildNodeKeyMap(blueprint);
    const states: Record<string, any> = {};
    const actors: Record<string, any> = {};

    for (const node of blueprint.nodes) {
      if (node.type === 'trigger') {
        states[node.id] = this.compileTriggerNode(node, blueprint);
      } else if (node.type === 'end') {
        states[node.id] = { type: 'final' };
      } else if (node.type === 'wait') {
        states[node.id] = this.compileWaitNode(node, blueprint);
      } else {
        const actorKey = `node_${node.type}_${node.id}`;
        actors[actorKey] = this.createNodeActor(node, blueprint);
        states[node.id] = this.compileInvokeNode(node, blueprint, actorKey);
      }
    }

    // 终态与错误态
    states[DONE_STATE] = { type: 'final' };
    states[ERROR_STATE] = { type: 'final' };

    const machine = setup({
      types: {
        context: {} as ExecutionContext,
        events: {} as { type: 'USER_MESSAGE'; input: string } | { type: string },
        input: {} as ActorInput,
      },
      actors,
    }).createMachine({
      id: blueprint.nodes.map((n) => n.id).join('-') || 'workflow',
      initial: triggerNode.id,
      context: ({ input }) => this.buildInitialContext(input, nameResolver, nodeKeyMap),
      states,
    });

    return machine as AnyStateMachine;
  }

  /** 从 actor input 构建初始 context，注入 compile 期构建的 nameResolver + nodeKeyMap */
  private buildInitialContext(
    input: ActorInput,
    nameResolver: Record<string, string>,
    nodeKeyMap: Record<string, string>,
  ): ExecutionContext {
    return {
      nodeOutputs: {},
      variables: {
        __rawInput__: input?.rawInput ?? '',
        username: input?.senderName ?? '',
        sender_id: input?.senderId ?? '',
        conversation_id: input?.conversationId ?? '',
        time: input?.time ?? new Date().toLocaleTimeString('zh-CN', { hour12: false }),
        ...(input?.variables ?? {}),
      },
      eventOutputs: {},
      contextBuffer: input?.contextBuffer ?? [],
      finalReply: '',
      nameResolver,
      nodeKeyMap,
      senderId: input?.senderId ?? '',
      senderName: input?.senderName ?? '',
      conversationId: input?.conversationId ?? '',
      rawInput: input?.rawInput ?? '',
      history: input?.contextBuffer ?? [],
      session: {},
      grantedCapabilities: input?.grantedCapabilities,
      secrets: input?.secrets,
    };
  }

  private createNodeActor(node: BlueprintNode, blueprint: Blueprint) {
    const def = this.registry.get(node.type);
    if (!def) throw new Error(`Unknown node type: ${node.type}`);

    return fromPromise(async ({ input }: { input: { context: ExecutionContext } }) => {
      const context = input.context;

      // 运行时强制校验:仅当显式传入 grantedCapabilities 时校验（向后兼容旧调用方）
      if (context.grantedCapabilities !== undefined) {
        const missing = getMissingCapabilities(node, context.grantedCapabilities);
        if (missing.length > 0) {
          throw new Error(
            `Capability denied: node "${node.name}" (${node.type}) requires [${missing.join(', ')}] but not granted`,
          );
        }
        // 校验 secrets:use（引用了 secret 但未授予时拒绝）
        const secretMissing = checkSecretCapability(node.config, context.grantedCapabilities);
        if (secretMissing.length > 0) {
          throw new Error(
            `Capability denied: node "${node.name}" references secrets.* but [${secretMissing.join(', ')}] is not granted`,
          );
        }
      }

      // 解析 secrets.<name> 引用，注入实际解密值
      const resolvedConfig = resolveSecrets(node.config, context.secrets);

      const nodeInput = {
        ports: this.resolveNodeInputs(node.id, blueprint.connections, context),
        rawInput: context.variables['__rawInput__'] ?? '',
      };
      const nodeCtx: NodeContext = {
        variables: context.variables,
        eventOutputs: context.eventOutputs,
        contextBuffer: context.contextBuffer,
        nodeOutputs: context.nodeOutputs,
        nameResolver: context.nameResolver,
        finalReply: context.finalReply,
        nodeKeyMap: context.nodeKeyMap,
        rawInput: context.rawInput,
        senderId: context.senderId,
        senderName: context.senderName,
        conversationId: context.conversationId,
        history: context.history,
        secrets: context.secrets ?? {},
        session: context.session,
      };
      return def.execute(nodeInput, resolvedConfig as Record<string, any>, nodeCtx);
    });
  }

  /**
   * Trigger 是事件驱动的入口状态。
   * 机器启动后停在 trigger，等待第一条 USER_MESSAGE 事件初始化输入，
   * 随后流转到下一个节点。这保证「首条消息只执行一次」且与多轮会话语义统一。
   */
  private compileTriggerNode(node: BlueprintNode, blueprint: Blueprint): any {
    const outConn = this.findOutputConnection(node.id, 'out_exec', blueprint);
    const target = outConn ? outConn.targetNodeId : DONE_STATE;

    return {
      on: {
        USER_MESSAGE: {
          actions: assign(({ context, event }: { context: ExecutionContext; event: UserMessageEvent }) => {
            const time = event.time ?? new Date().toLocaleTimeString('zh-CN', { hour12: false });
            const username = event.senderName ?? context.senderName ?? context.variables['username'];
            const rawInput = event.input;
            return {
              senderName: username,
              variables: {
                ...context.variables,
                __rawInput__: rawInput,
                username,
                time,
              },
              nodeOutputs: {
                ...context.nodeOutputs,
                [node.id]: {
                  out_input: rawInput,
                  out_username: username,
                  out_time: time,
                  out_args: '',
                  out_exec: 'true',
                },
              },
            };
          }),
          target,
        },
      },
    };
  }

  private compileInvokeNode(node: BlueprintNode, blueprint: Blueprint, actorKey: string): any {
    const onDoneActions = assign({
      nodeOutputs: ({ context, event }: { context: ExecutionContext; event: any }) => ({
        ...context.nodeOutputs,
        [node.id]: event.output?.ports || {},
      }),
      eventOutputs: ({ context, event }: { context: ExecutionContext; event: any }) => {
        const value = event.output?.ports?.['out_output'] || '';
        if (value) {
          return { ...context.eventOutputs, [node.id]: value };
        }
        return context.eventOutputs;
      },
      finalReply: ({ context, event }: { context: ExecutionContext; event: any }) => {
        const reply = event.output?.ports?.['__reply__'];
        return reply || context.finalReply;
      },
    });

    if (['if', 'switch', 'loop', 'merge'].includes(node.type)) {
      const onDoneTransitions: any[] = [
        ...this.getControlRoutes(node, blueprint).map(({ portId, target }) => ({
          guard: ({ context, event }: { context: ExecutionContext; event: any }) =>
            resolveControlFlowRoute(node, event.output?.ports ?? {}, context.session)?.portId === portId,
          actions: [onDoneActions, this.assignControlRoute(node)],
          target,
        })),
        { actions: [onDoneActions, this.assignControlRoute(node)], target: DONE_STATE },
      ];

      return {
        invoke: {
          id: `invoke-${node.id}`,
          src: actorKey,
          input: ({ context }: { context: ExecutionContext }) => ({ context }),
          onDone: onDoneTransitions,
          onError: {
            target: ERROR_STATE,
            actions: assign({
              lastError: ({ event }: { event: any }) =>
                event?.error?.message ?? event?.data?.message ?? 'node execution failed',
            }),
          },
        },
      };
    }

    // 普通节点：有后继则流转，无后继则结束
    const outConn = this.findOutputConnection(node.id, 'out_exec', blueprint);
    return {
      invoke: {
        id: `invoke-${node.id}`,
        src: actorKey,
        input: ({ context }: { context: ExecutionContext }) => ({ context }),
        onDone: {
          actions: onDoneActions,
          target: outConn ? outConn.targetNodeId : DONE_STATE,
        },
        onError: {
          target: ERROR_STATE,
          actions: assign({
            lastError: ({ event }: { event: any }) =>
              event?.error?.message ?? event?.data?.message ?? 'node execution failed',
          }),
        },
      },
    };
  }

  private getControlRoutes(node: BlueprintNode, blueprint: Blueprint): Array<{ portId: string; target: string }> {
    const ports = node.type === 'if'
      ? [
        'out_true',
        ...Array.from(
          { length: Math.max(0, (node.config.branches?.length ?? 1) - 1) },
          (_, index) => `out_elif_${index}`,
        ),
        'out_false',
      ]
      : node.type === 'switch'
        ? [...(node.config.cases ?? []).map((_: unknown, index: number) => `out_case_${index}`), 'out_default']
        : node.type === 'loop'
          ? ['out_body', 'out_done']
          : ['out_exec'];

    return ports.map((portId) => ({
      portId,
      target: this.findOutputConnection(node.id, portId, blueprint)?.targetNodeId ?? DONE_STATE,
    }));
  }

  private assignControlRoute(node: BlueprintNode) {
    return assign(({ context, event }: { context: ExecutionContext; event: any }) => {
      const output = event.output?.ports ?? {};
      const route = resolveControlFlowRoute(node, output, context.session);
      if (!route) return {};

      return {
        session: route.session,
        nodeOutputs: {
          ...context.nodeOutputs,
          [node.id]: { ...output, __branch__: route.portId, [route.portId]: 'true' },
        },
      };
    });
  }

  /**
   * Wait 节点：暂停工作流，等待下一条 USER_MESSAGE。
   * 收到消息后更新 rawInput 与 wait 输出端口，然后流转。
   */
  private compileWaitNode(node: BlueprintNode, blueprint: Blueprint): any {
    const outConn = this.findOutputConnection(node.id, 'out_exec', blueprint);
    const target = outConn ? outConn.targetNodeId : DONE_STATE;

    return {
      on: {
        USER_MESSAGE: {
          actions: assign(({ context, event }: { context: ExecutionContext; event: UserMessageEvent }) => {
            const rawInput = event.input;
            const time = event.time ?? new Date().toLocaleTimeString('zh-CN', { hour12: false });
            return {
              variables: {
                ...context.variables,
                __rawInput__: rawInput,
                time,
              },
              nodeOutputs: {
                ...context.nodeOutputs,
                [node.id]: {
                  out_user_input: rawInput,
                  out_exec: 'true',
                },
              },
            };
          }),
          target,
        },
      },
    };
  }

  private findOutputConnection(nodeId: string, portId: string, blueprint: Blueprint) {
    return blueprint.connections.find(
      (c) => c.sourceNodeId === nodeId && c.sourcePortId === portId,
    );
  }

  private resolveNodeInputs(
    nodeId: string,
    connections: BlueprintConnection[] | any[],
    context: ExecutionContext,
  ): Record<string, string> {
    const result: Record<string, string> = {};
    for (const conn of connections) {
      if (conn.targetNodeId === nodeId) {
        const val = context.nodeOutputs[conn.sourceNodeId]?.[conn.sourcePortId];
        if (val !== undefined) {
          result[conn.targetPortId] = val;
        }
      }
    }
    return result;
  }

  private buildNameResolver(blueprint: Blueprint): Record<string, string> {
    const resolver: Record<string, string> = {};
    for (const node of blueprint.nodes) {
      const ports = node.ports || [];
      for (const port of ports) {
        if (port.direction === 'output') {
          resolver[`${node.name}.${port.name}`] = `${node.id}:${port.id}`;
        }
      }
    }
    return resolver;
  }

  /** 构建 nodeKey → nodeId 映射，用于 ${nodes.<key>.outputs.<port>} 解析 */
  private buildNodeKeyMap(blueprint: Blueprint): Record<string, string> {
    const map: Record<string, string> = {};
    for (const node of blueprint.nodes) {
      if (node.key) {
        map[node.key] = node.id;
      }
    }
    return map;
  }
}
