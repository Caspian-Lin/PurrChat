import dagre from '@dagrejs/dagre';
import type { SpecialModeEvent, FlowConnection } from '../models/types';
import type { Node, Edge } from '@vue-flow/core';
import { NODE_TYPE_META } from './portTypes';
import { buildDefaultPorts } from './eventMigration';

// ─── 节点类型图标（兼容旧代码，新代码优先使用 NODE_TYPE_META） ──

export const typeIcons: Record<string, string> = {
  llm: '🧠',
  builtin: '⚙',
  python: '🐍',
  reply: '💬',
};

export function getEventSummary(evt: SpecialModeEvent): string {
  switch (evt.type) {
    case 'llm':
      return evt.config.model || 'LLM 调用';
    case 'builtin':
      return evt.config.builtin_type || '内置事件';
    case 'python':
      return evt.config.code ? `${evt.config.code.split('\n')[0]}...` : 'Python 脚本';
    case 'reply':
      return evt.config.template ? evt.config.template.slice(0, 30) + '...' : '发送回复';
    case 'template':
      return evt.config.template ? evt.config.template.slice(0, 30) + '...' : '模板渲染';
    case 'trigger':
      return '起始';
    case 'end':
      return '结束';
    case 'wait':
      return evt.config.wait_type || '等待用户输入';
    case 'if':
      return '条件分支';
    case 'loop':
      return `循环${evt.config.max_iterations ? ` (最多${evt.config.max_iterations}次)` : ''}`;
    default:
      return '';
  }
}

// ─── 节点 & 边转换（端口化模型） ──────────────────────────────

export function eventsToFlowNodes(
  events: SpecialModeEvent[],
  positions?: Record<string, { x: number; y: number }>
): Node[] {
  return events.map((evt, index) => {
    const ports = buildDefaultPorts(evt);
    const meta = NODE_TYPE_META[evt.type as keyof typeof NODE_TYPE_META];
    return {
      id: evt.id,
      type: 'event',
      position: evt.position ||
        positions?.[evt.id] || { x: index * 280, y: 50 + (index % 3) * 120 },
      data: {
        label: evt.name,
        eventType: evt.type,
        summary: getEventSummary(evt),
        icon: meta?.icon || typeIcons[evt.type] || '',
        ports,
      },
    };
  });
}

/** 将 FlowConnection[] 转换为 VueFlow Edge[]（端口化连线） */
export function connectionsToFlowEdges(
  connections: FlowConnection[],
  events?: SpecialModeEvent[]
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

/** 将旧 next[] 转换为 VueFlow Edge[]（向后兼容） */
export function eventsToFlowEdges(events: SpecialModeEvent[]): Edge[] {
  const edges: Edge[] = [];
  for (const evt of events) {
    for (const nextId of evt.next || []) {
      edges.push({
        id: `${evt.id}-${nextId}`,
        source: evt.id,
        target: nextId,
        type: 'event',
      });
    }
  }
  return edges;
}

// ─── 自动布局（dagre） ───────────────────────────────────────

/**
 * 使用 dagre 对事件链进行自动布局，返回更新后的 Node 数组。
 * direction: 'TB'（上到下）| 'LR'（左到右）
 */
export function autoLayoutEvents(
  events: SpecialModeEvent[],
  direction: 'TB' | 'LR' = 'TB'
): Node[] {
  const nodes = eventsToFlowNodes(events);
  const edges = eventsToFlowEdges(events);

  if (nodes.length === 0) return nodes;

  const g = new dagre.graphlib.Graph();
  g.setDefaultEdgeLabel(() => ({}));
  g.setGraph({
    rankdir: direction,
    ranksep: 80,
    nodesep: 50,
    edgesep: 30,
  });

  for (const node of nodes) {
    g.setNode(node.id, { width: 220, height: 60 });
  }
  for (const edge of edges) {
    g.setEdge(edge.source, edge.target);
  }

  dagre.layout(g);

  return nodes.map((node) => {
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
 * 检测：缺少 trigger/end 节点、多个 trigger、环路、孤儿节点、
 * 缺少 reply 类型事件、断开的连线
 */
export function validateEventChain(events: SpecialModeEvent[]): ValidationIssue[] {
  const issues: ValidationIssue[] = [];
  if (events.length === 0) return issues;

  const eventIds = new Set(events.map((e) => e.id));

  // 检查 trigger 节点
  const triggerEvents = events.filter((e) => e.type === 'trigger');
  if (triggerEvents.length === 0) {
    issues.push({ type: 'error', message: '事件链缺少起始节点（trigger）' });
  } else if (triggerEvents.length > 1) {
    issues.push({ type: 'error', message: '事件链只能有一个起始节点（trigger）' });
  }

  // 检查断开的连线（next 指向不存在的事件）
  for (const evt of events) {
    for (const nextId of evt.next || []) {
      if (!eventIds.has(nextId)) {
        issues.push({
          type: 'error',
          message: `"${evt.name}" 连接到不存在的事件`,
          eventId: evt.id,
        });
      }
    }
  }

  // 检测环路（DFS）
  const visiting = new Set<string>();
  const visited = new Set<string>();

  function hasCycle(id: string): boolean {
    if (visiting.has(id)) return true;
    if (visited.has(id)) return false;
    visiting.add(id);
    const evt = events.find((e) => e.id === id);
    if (evt) {
      for (const nextId of evt.next || []) {
        if (eventIds.has(nextId) && hasCycle(nextId)) return true;
      }
    }
    visiting.delete(id);
    visited.add(id);
    return false;
  }

  for (const evt of events) {
    if (hasCycle(evt.id)) {
      issues.push({
        type: 'error',
        message: '事件链中存在环路',
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

  // 检查孤儿节点（没有任何事件连接到它，且它不是 trigger）
  const triggerEvent = events.find((e) => e.type === 'trigger') || events[0];
  const isTargetOf = new Set<string>();
  for (const evt of events) {
    for (const nextId of evt.next || []) {
      isTargetOf.add(nextId);
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
