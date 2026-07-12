import dagre from '@dagrejs/dagre';
import type { WorkflowEvent, FlowConnection } from '../models/types';
import type { Node, Edge } from '@vue-flow/core';
import { NODE_TYPE_META } from './portTypes';
import { buildDefaultPorts } from './eventPorts';

// ─── 节点类型图标（兼容旧代码，新代码优先使用 NODE_TYPE_META） ──

export const typeIcons: Record<string, string> = {
  llm: '🧠',
  builtin: '⚙',
  reply: '💬',
};

export function getEventSummary(evt: WorkflowEvent): string {
  switch (evt.type) {
    case 'llm':
      return evt.config.model || 'LLM 调用';
    case 'builtin':
      return evt.config.builtin_type || '内置事件';
    case 'reply':
      return evt.config.template ? evt.config.template.slice(0, 30) + '...' : '发送回复';
    case 'template':
      return evt.config.template ? evt.config.template.slice(0, 30) + '...' : '模板渲染';
    case 'trigger':
      return '起始';
    case 'end':
      return '结束';
    case 'wait':
      if (evt.config.wait_type === 'custom') return '等待条件';
      return '等待用户消息';
    case 'if':
      return evt.config.condition || '条件分支';
    case 'loop':
      return `循环${evt.config.max_iterations ? ` (最多${evt.config.max_iterations}次)` : ''}`;
    case 'history':
      return `最近 ${evt.config.count || 20} 条消息`;
    default:
      return '';
  }
}

// ─── 节点 & 边转换（端口化模型） ──────────────────────────────

/**
 * 将 events 转换为 VueFlow Node[]。
 * 所有节点（包括 loop）都以普通节点形式渲染在扁平画布上。
 */
export function eventsToFlowNodes(
  events: WorkflowEvent[],
  positions?: Record<string, { x: number; y: number }>,
  _connections?: FlowConnection[]
): Node[] {
  const nodes: Node[] = [];

  for (const evt of events) {
    const ports = buildDefaultPorts(evt);
    const meta = NODE_TYPE_META[evt.type as keyof typeof NODE_TYPE_META];

    nodes.push({
      id: evt.id,
      type: evt.type,
      position: evt.position || positions?.[evt.id] || { x: nodes.length * 280, y: 50 },
      data: {
        label: evt.name,
        eventType: evt.type,
        summary: getEventSummary(evt),
        icon: meta?.icon || typeIcons[evt.type] || '',
        ports,
        config: evt.config,
      },
    });
  }

  return nodes;
}

/** 将 FlowConnection[] 转换为 VueFlow Edge[]（端口化连线） */
export function connectionsToFlowEdges(
  connections: FlowConnection[],
  events?: WorkflowEvent[]
): Edge[] {
  // 构建 portId → dataType 映射
  const portTypeMap = new Map<string, string>();
  if (events) {
    for (const evt of events) {
      const ports = buildDefaultPorts(evt);
      for (const port of ports) {
        portTypeMap.set(`${evt.id}:${port.id}`, port.dataType);
      }
    }
  }

  // 收集 loop 节点 ID（回环模式下所有连线都可见，不需要过滤）
  return connections.map((conn) => {
    const sourceDataType = portTypeMap.get(`${conn.sourceNodeId}:${conn.sourcePortId}`) || 'any';
    return {
      id: conn.id,
      source: conn.sourceNodeId,
      target: conn.targetNodeId,
      sourceHandle: conn.sourcePortId,
      targetHandle: conn.targetPortId,
      type: 'event',
      data: {
        sourcePortId: conn.sourcePortId,
        targetPortId: conn.targetPortId,
        dataType: sourceDataType,
        isExec: sourceDataType === 'trigger',
      },
    };
  });
}

/** 将 FlowConnection[] 转换为 VueFlow Edge[] */
export function eventsToFlowEdges(events: WorkflowEvent[], connections?: FlowConnection[]): Edge[] {
  if (connections && connections.length > 0) {
    return connectionsToFlowEdges(connections, events);
  }
  // 无连接时返回空数组
  return [];
}

// ─── 自动布局（dagre） ───────────────────────────────────────

/**
 * 使用 dagre 对事件链进行自动布局，返回更新后的 Node 数组。
 * direction: 'TB'（上到下）| 'LR'（左到右）
 *
 * 回环模式下所有节点都是扁平的，用单层 dagre 即可。
 * dagre 本身能处理图中的环（回边），布局不会出错。
 */
export function autoLayoutEvents(
  events: WorkflowEvent[],
  direction: 'TB' | 'LR' = 'TB',
  connections?: FlowConnection[]
): Node[] {
  const allNodes = eventsToFlowNodes(events, undefined, connections);
  const edges = connections ? connectionsToFlowEdges(connections, events) : [];

  if (allNodes.length === 0) return allNodes;

  const g = new dagre.graphlib.Graph();
  g.setDefaultEdgeLabel(() => ({}));
  g.setGraph({
    rankdir: direction,
    ranksep: 80,
    nodesep: 50,
    edgesep: 30,
  });

  for (const node of allNodes) {
    g.setNode(node.id, { width: 220, height: 60 });
  }

  for (const edge of edges) {
    g.setEdge(edge.source, edge.target);
  }

  dagre.layout(g);

  return allNodes.map((node) => {
    const pos = g.node(node.id);
    return {
      ...node,
      position: {
        x: (pos?.x ?? 0) - 110,
        y: (pos?.y ?? 0) - 30,
      },
    };
  });
}

// ─── 验证 ────────────────────────────────────────────────────

export interface ValidationIssue {
  type: 'warning' | 'error';
  message: string;
  eventId?: string;
}

/**
 * 验证事件链结构的合法性。
 *
 * 回环模式下，合法的环只有一种：loop 节点的回边（某个节点 → loop.in_exec）。
 * 其它环路视为错误。
 */
export function validateEventChain(
  events: WorkflowEvent[],
  connections?: FlowConnection[]
): ValidationIssue[] {
  const issues: ValidationIssue[] = [];
  if (events.length === 0) return issues;

  const eventIds = new Set(events.map((e) => e.id));
  const loopIds = new Set(events.filter((e) => e.type === 'loop').map((e) => e.id));

  // 收集合法的回边集合：任意节点 → loop 节点的 in_exec 端口
  const legalBackEdges = new Set<string>();
  if (connections) {
    for (const conn of connections) {
      if (loopIds.has(conn.targetNodeId) && conn.targetPortId === 'in_exec') {
        legalBackEdges.add(`${conn.sourceNodeId}:${conn.targetNodeId}`);
      }
    }
  }

  // 检查 trigger 节点
  const triggerEvents = events.filter((e) => e.type === 'trigger');
  if (triggerEvents.length === 0) {
    issues.push({ type: 'error', message: '事件链缺少起始节点（trigger）' });
  } else if (triggerEvents.length > 1) {
    issues.push({ type: 'error', message: '事件链只能有一个起始节点（trigger）' });
  }

  // 检查断开的连线（connections 模式）
  if (connections && connections.length > 0) {
    for (const conn of connections) {
      if (!eventIds.has(conn.sourceNodeId)) {
        const srcEvt = events.find((e) => e.id === conn.sourceNodeId);
        issues.push({
          type: 'error',
          message: `"${srcEvt?.name || conn.sourceNodeId}" 连接到不存在的事件`,
          eventId: conn.sourceNodeId,
        });
      }
    }
  }

  // 检测非法环路（DFS）— 排除 loop 回边
  const visiting = new Set<string>();
  const visited = new Set<string>();

  // 构建邻接表（基于 connections）
  const adj = new Map<string, string[]>();
  if (connections && connections.length > 0) {
    for (const conn of connections) {
      if (!adj.has(conn.sourceNodeId)) adj.set(conn.sourceNodeId, []);
      adj.get(conn.sourceNodeId)!.push(conn.targetNodeId);
    }
  }

  function hasIllegalCycle(id: string): boolean {
    if (visiting.has(id)) return true;
    if (visited.has(id)) return false;
    visiting.add(id);
    for (const nextId of adj.get(id) || []) {
      // 跳过合法的 loop 回边
      if (legalBackEdges.has(`${id}:${nextId}`)) continue;
      if (eventIds.has(nextId) && hasIllegalCycle(nextId)) return true;
    }
    visiting.delete(id);
    visited.add(id);
    return false;
  }

  for (const evt of events) {
    if (hasIllegalCycle(evt.id)) {
      issues.push({
        type: 'error',
        message: '事件链中存在非法环路（只有循环节点允许回边）',
        eventId: evt.id,
      });
      break;
    }
  }

  // 检查是否有 reply 类型事件
  const hasReply = events.some((e) => e.type === 'reply');
  if (!hasReply) {
    issues.push({
      type: 'warning',
      message: '事件链中没有回复事件，Bot 将不会发送任何消息',
    });
  }

  // 检查孤儿节点
  const triggerEvent = events.find((e) => e.type === 'trigger') || events[0];
  const isTargetOf = new Set<string>();

  if (connections && connections.length > 0) {
    for (const conn of connections) {
      isTargetOf.add(conn.targetNodeId);
    }
  }

  for (const evt of events) {
    if (evt.type === 'trigger') continue;
    if (triggerEvent && evt.id !== triggerEvent.id && !isTargetOf.has(evt.id)) {
      issues.push({
        type: 'warning',
        message: `"${evt.name}" 没有被任何事件连接，不会被执行`,
        eventId: evt.id,
      });
    }
  }

  return issues;
}
