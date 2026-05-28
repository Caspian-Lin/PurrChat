import { setup, assign, fromPromise, type AnyStateMachine } from 'xstate';
import type { Blueprint, BlueprintNode, ExecutionContext } from './types.js';
import type { NodeRegistry } from './registry.js';

export class Compiler {
  constructor(private registry: NodeRegistry) {}

  compile(blueprint: Blueprint): AnyStateMachine {
    const triggerNode = blueprint.nodes.find((n) => n.type === 'trigger');
    if (!triggerNode) throw new Error('Blueprint must have a trigger node');

    const nameResolver = this.buildNameResolver(blueprint);
    const states: Record<string, any> = {};

    // 收集所有需要的 actors
    const actors: Record<string, any> = {};

    for (const node of blueprint.nodes) {
      if (node.type === 'trigger' || node.type === 'end' || node.type === 'wait') {
        states[node.id] = this.compileNode(node, blueprint);
      } else {
        const actorKey = `node_${node.type}_${node.id}`;
        actors[actorKey] = this.createNodeActor(node, blueprint);
        states[node.id] = this.compileInvokeNode(node, blueprint, actorKey);
      }
    }

    // 添加 __error 状态
    states['__error'] = { type: 'final' };

    const machine = setup({
      types: {
        context: {} as ExecutionContext,
        events: {} as { type: 'USER_MESSAGE'; input: string } | { type: string },
      },
      actors,
    }).createMachine({
      id: blueprint.nodes.map((n) => n.id).join('-') || 'workflow',
      initial: triggerNode.id,
      context: {
        nodeOutputs: {},
        variables: {},
        eventOutputs: {},
        contextBuffer: [],
        finalReply: '',
        nameResolver,
      },
      states,
    });

    return machine as AnyStateMachine;
  }

  private createNodeActor(node: BlueprintNode, blueprint: Blueprint) {
    const def = this.registry.get(node.type);
    if (!def) throw new Error(`Unknown node type: ${node.type}`);

    return fromPromise(async ({ input }: { input: { context: ExecutionContext } }) => {
      const context = input.context;
      const nodeInput = {
        ports: this.resolveNodeInputs(node.id, blueprint.connections, context),
        rawInput: context.variables['__rawInput__'] || '',
      };
      const nodeCtx = {
        variables: context.variables,
        eventOutputs: context.eventOutputs,
        contextBuffer: context.contextBuffer,
      };
      return def.execute(nodeInput, node.config || {}, nodeCtx);
    });
  }

  private compileNode(node: BlueprintNode, blueprint: Blueprint): any {
    switch (node.type) {
      case 'trigger':
        return this.compileTriggerNode(node, blueprint);
      case 'end':
        return { type: 'final' };
      case 'wait':
        return this.compileWaitNode(node, blueprint);
      default:
        throw new Error(`Unexpected node type in compileNode: ${node.type}`);
    }
  }

  private compileTriggerNode(node: BlueprintNode, blueprint: Blueprint): any {
    const outConn = this.findOutputConnection(node.id, 'out_exec', blueprint);

    return {
      entry: assign(({ context }: { context: ExecutionContext }) => ({
        nodeOutputs: {
          ...context.nodeOutputs,
          [node.id]: {
            out_input: context.variables['__rawInput__'] || '',
            out_username: context.variables['username'] || '',
            out_time: context.variables['time'] || new Date().toLocaleTimeString('zh-CN', { hour12: false }),
            out_args: '',
            out_exec: 'true',
          },
        },
      })),
      ...(outConn ? { always: { target: outConn.targetNodeId } } : { type: 'final' }),
    };
  }

  private compileInvokeNode(node: BlueprintNode, blueprint: Blueprint, actorKey: string): any {
    const outConn = this.findOutputConnection(node.id, 'out_exec', blueprint);
    const trueConn = this.findOutputConnection(node.id, 'out_true', blueprint);
    const falseConn = this.findOutputConnection(node.id, 'out_false', blueprint);
    const isIfNode = node.type === 'if';

    return {
      invoke: {
        id: `invoke-${node.id}`,
        src: actorKey,
        input: ({ context }: { context: ExecutionContext }) => ({ context }),
        onDone: {
          actions: assign({
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
          }),
          ...(isIfNode
            ? {
                always: [
                  ...(trueConn
                    ? [{
                        guard: ({ context }: { context: ExecutionContext }) =>
                          context.nodeOutputs[node.id]?.['__branch__'] === 'true',
                        target: trueConn.targetNodeId,
                      }]
                    : []),
                  ...(falseConn
                    ? [{
                        guard: ({ context }: { context: ExecutionContext }) =>
                          context.nodeOutputs[node.id]?.['__branch__'] === 'false',
                        target: falseConn.targetNodeId,
                      }]
                    : []),
                ],
              }
            : {}),
          ...(!isIfNode && outConn ? { target: outConn.targetNodeId } : {}),
          ...(!isIfNode && !outConn ? { type: 'final' } : {}),
        },
        onError: { target: '__error' },
      },
    };
  }

  private compileWaitNode(node: BlueprintNode, blueprint: Blueprint): any {
    const outConn = this.findOutputConnection(node.id, 'out_exec', blueprint);

    return {
      on: {
        USER_MESSAGE: {
          actions: assign({
            nodeOutputs: ({ context, event }: { context: ExecutionContext; event: any }) => ({
              ...context.nodeOutputs,
              [node.id]: {
                out_user_input: event.input || '',
                out_exec: 'true',
              },
            }),
          }),
          ...(outConn ? { target: outConn.targetNodeId } : {}),
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
    connections: any[],
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
}
