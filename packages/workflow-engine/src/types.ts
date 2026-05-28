import type { z } from 'zod';
import type { FlowConnection, EventPort } from '@purrchat/workflow-types';

// ─── 节点定义 ────────────────────────────────────────────────

export interface NodeDefinition<TConfig extends Record<string, any> = Record<string, any>> {
  type: string;
  label: string;
  category: 'trigger' | 'processing' | 'control' | 'output';
  icon: string;
  configSchema: z.ZodType<TConfig, z.ZodTypeDef, any>;
  execute: (input: NodeInput, config: Record<string, any>, ctx: NodeContext) => Promise<NodeOutput>;
}

export interface NodeInput {
  ports: Record<string, string>;  // portId -> resolved value
  rawInput: string;               // 原始用户消息
}

export interface NodeOutput {
  ports: Record<string, string>;  // output portId -> value
}

export interface NodeContext {
  variables: Record<string, string>;
  eventOutputs: Record<string, string>;
  contextBuffer: Array<{ role: string; content: string }>;
}

// ─── Blueprint（工作流定义） ──────────────────────────────────

export interface BlueprintNode {
  id: string;
  type: string;
  name: string;
  config: Record<string, any>;
  ports?: EventPort[];
  position?: { x: number; y: number };
}

export interface BlueprintConnection {
  id: string;
  sourceNodeId: string;
  sourcePortId: string;
  targetNodeId: string;
  targetPortId: string;
}

export interface Blueprint {
  nodes: BlueprintNode[];
  connections: BlueprintConnection[];
  endConditions: Array<{ type: string; pattern?: string; value?: number }>;
}

// ─── 执行上下文 ──────────────────────────────────────────────

export interface ExecutionContext {
  nodeOutputs: Record<string, Record<string, string>>;  // nodeId -> { portId -> value }
  variables: Record<string, string>;
  eventOutputs: Record<string, string>;  // eventId -> output
  contextBuffer: Array<{ role: string; content: string }>;
  finalReply: string;
  nameResolver: Record<string, string>;  // "nodeName.portName" -> "nodeID:portID"
}
