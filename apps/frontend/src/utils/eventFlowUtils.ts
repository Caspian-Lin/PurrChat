import dagre from '@dagrejs/dagre';
import type { SpecialModeEvent, FlowConnection } from '../models/types';
import type { Node, Edge } from '@vue-flow/core';
import { NODE_TYPE_META } from './portTypes';
import { buildDefaultPorts } from './eventPorts';

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

// ─── 循环体子节点推断 ──────────────────────────────────────

/**
 * 从 connections 推断循环体父子关系。
 * 返回 Map<childEventId, parentLoopId>
 *
 * 逻辑：对每个 loop 节点，从 out_body 连接开始 BFS，
 * 沿 trigger 类型输出连接标记所有可达节点为 loop 的子节点。
 */
export function inferLoopChildren(
  events: SpecialModeEvent[],
  connections: FlowConnection[]
): Map<string, string> {
  const result = new Map<string, string>();

  // 构建 nodeId → outgoing connections
  const outConns = new Map<string, FlowConnection[]>();
  for (const conn of connections) {
    if (!outConns.has(conn.sourceNodeId)) outConns.set(conn.sourceNodeId, []);
    outConns.get(conn.sourceNodeId)!.push(conn);
  }

  // 构建端口 ID → dataType 映射
  const portTypes = new Map<string, string>();
  for (const evt of events) {
    for (const port of evt.ports || []) {
      portTypes.set(`${evt.id}:${port.id}`, port.dataType);
    }
  }

  const loopNodes = events.filter((e) => e.type === 'loop');

  for (const loopNode of loopNodes) {
    const bodyConns = (outConns.get(loopNode.id) || []).filter(
      (c) => c.sourcePortId === 'out_body'
    );
    if (bodyConns.length === 0) continue;

    const firstChildId = bodyConns[0]!.targetNodeId;
    const visited = new Set<string>();
    const queue = [firstChildId];

    while (queue.length > 0) {
      const nodeId = queue.shift()!;
      if (visited.has(nodeId)) continue;
      visited.add(nodeId);
      result.set(nodeId, loopNode.id);

      // 沿 trigger 类型输出连接继续
      const nextConns = (outConns.get(nodeId) || []).filter((c) => {
        const dataType = portTypes.get(`${c.sourceNodeId}:${c.sourcePortId}`);
        return dataType === 'trigger';
      });

      for (const conn of nextConns) {
        // 回边到 loop 自身则跳过
        if (conn.targetNodeId === loopNode.id) continue;
        queue.push(conn.targetNodeId);
      }
    }
  }

  return result;
}

// ─── 循环体链末端查找 ──────────────────────────────────────

/**
 * 获取循环体执行链的末端节点 ID（即回边到 loop 自身的那个节点）。
 * 用于在链尾插入新节点时定位断开点。
 */
export function getLoopChainEnd(loopId: string, connections: FlowConnection[]): string | null {
  for (const conn of connections) {
    if (conn.targetNodeId === loopId && conn.targetPortId === 'in_exec') {
      return conn.sourceNodeId;
    }
  }
  return null;
}

// ─── 节点 & 边转换（端口化模型） ──────────────────────────────

/**
 * 将 events 转换为 VueFlow Node[]。
 * 如果提供 connections，loop 节点会渲染为 loop 类型，
 * 其子节点通过 parentNode 关联。
 */
export function eventsToFlowNodes(
  events: SpecialModeEvent[],
  positions?: Record<string, { x: number; y: number }>,
  connections?: FlowConnection[]
): Node[] {
  const parentMap = connections
    ? inferLoopChildren(events, connections)
    : new Map<string, string>();

  const nodes: Node[] = [];

  // 第一遍：loop 框体节点（父节点必须排在子节点前面）
  for (const evt of events) {
    if (evt.type !== 'loop') continue;
    const ports = buildDefaultPorts(evt);
    const meta = NODE_TYPE_META[evt.type];
    nodes.push({
      id: evt.id,
      type: 'loop',
      position: evt.position || positions?.[evt.id] || { x: 280, y: 50 },
      style: { width: '500px', height: '300px' },
      data: {
        label: evt.name,
        eventType: evt.type,
        summary: getEventSummary(evt),
        icon: meta?.icon || '',
        ports,
        config: evt.config,
      },
    });
  }

  // 第二遍：非 loop 节点
  let childIndex = 0;
  for (const evt of events) {
    if (evt.type === 'loop') continue;
    const ports = buildDefaultPorts(evt);
    const meta = NODE_TYPE_META[evt.type as keyof typeof NODE_TYPE_META];
    const parentId = parentMap.get(evt.id);

    const node: Node = {
      id: evt.id,
      type: evt.type,
      position: { x: 0, y: 0 },
      data: {
        label: evt.name,
        eventType: evt.type,
        summary: getEventSummary(evt),
        icon: meta?.icon || typeIcons[evt.type] || '',
        ports,
        config: evt.config,
      },
    };

    if (parentId) {
      node.parentNode = parentId;
      node.extent = 'parent';
      node.expandParent = true;
      // 子节点位置相对于父节点
      node.position = evt.position || { x: 60, y: 60 + childIndex * 90 };
      childIndex++;
    } else {
      node.position = evt.position ||
        positions?.[evt.id] || {
          x: nodes.filter((n) => !n.parentNode).length * 280,
          y: 50,
        };
    }

    nodes.push(node);
  }

  return nodes;
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

  // 收集 loop 节点 ID，用于过滤结构边
  const loopIds = new Set((events || []).filter((e) => e.type === 'loop').map((e) => e.id));

  // 推断循环体子节点关系，用于精确识别回边
  const childMap =
    events && events.length > 0
      ? inferLoopChildren(events, connections)
      : new Map<string, string>();

  // 过滤循环结构边（入口/出口），这些边由框体视觉隐含
  const visibleConns = connections.filter((conn) => {
    // 入口边：loop.out_body → 子节点
    if (loopIds.has(conn.sourceNodeId) && conn.sourcePortId === 'out_body') return false;
    // 出口边：子节点 → loop.in_exec（回边）— 仅当 source 是该 loop 的子节点时
    if (loopIds.has(conn.targetNodeId) && conn.targetPortId === 'in_exec') {
      if (childMap.get(conn.sourceNodeId) === conn.targetNodeId) return false;
    }
    return true;
  });

  return visibleConns.map((conn) => {
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
export function eventsToFlowEdges(
  events: SpecialModeEvent[],
  connections?: FlowConnection[]
): Edge[] {
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
 */
export function autoLayoutEvents(
  events: SpecialModeEvent[],
  direction: 'TB' | 'LR' = 'TB',
  connections?: FlowConnection[]
): Node[] {
  const parentMap = connections
    ? inferLoopChildren(events, connections)
    : new Map<string, string>();

  const allNodes = eventsToFlowNodes(events, undefined, connections);
  const edges = eventsToFlowEdges(events, connections);

  if (allNodes.length === 0) return allNodes;

  // 分离顶层节点和子节点
  const topNodes = allNodes.filter((n) => !n.parentNode);
  const childNodesByParent = new Map<string, Node[]>();
  for (const node of allNodes) {
    if (node.parentNode) {
      if (!childNodesByParent.has(node.parentNode)) {
        childNodesByParent.set(node.parentNode, []);
      }
      childNodesByParent.get(node.parentNode)!.push(node);
    }
  }

  // 外层 dagre 布局（仅顶层节点）
  const g = new dagre.graphlib.Graph();
  g.setDefaultEdgeLabel(() => ({}));
  g.setGraph({
    rankdir: direction,
    ranksep: 80,
    nodesep: 50,
    edgesep: 30,
  });

  for (const node of topNodes) {
    g.setNode(node.id, {
      width: node.type === 'loop' ? 500 : 220,
      height: node.type === 'loop' ? 300 : 60,
    });
  }

  // 仅添加顶层边（跳过 loop body 内部的边）
  for (const edge of edges) {
    const srcParent = parentMap.get(edge.source);
    const tgtParent = parentMap.get(edge.target);
    if (srcParent && tgtParent && srcParent === tgtParent) continue;
    g.setEdge(edge.source, edge.target);
  }

  dagre.layout(g);

  const result: Node[] = [];

  for (const node of topNodes) {
    const pos = g.node(node.id);
    const w = node.type === 'loop' ? 500 : 220;
    const h = node.type === 'loop' ? 300 : 60;
    result.push({
      ...node,
      position: {
        x: (pos?.x ?? 0) - w / 2,
        y: (pos?.y ?? 0) - h / 2,
      },
    });

    // 内层 dagre 布局（loop body 内的子节点）
    if (childNodesByParent.has(node.id)) {
      const children = childNodesByParent.get(node.id)!;
      const childEdges = edges.filter(
        (e) => children.some((c) => c.id === e.source) && children.some((c) => c.id === e.target)
      );

      if (children.length <= 1) {
        const child = children[0];
        if (child) {
          child.position = { x: 60, y: 80 };
          result.push(child);
        }
      } else {
        const subG = new dagre.graphlib.Graph();
        subG.setDefaultEdgeLabel(() => ({}));
        subG.setGraph({ rankdir: direction, ranksep: 60, nodesep: 40 });

        for (const child of children) {
          subG.setNode(child.id, { width: 220, height: 60 });
        }
        for (const edge of childEdges) {
          subG.setEdge(edge.source, edge.target);
        }

        dagre.layout(subG);

        for (const child of children) {
          const childPos = subG.node(child.id);
          child.position = {
            x: 40 + (childPos?.x ?? 0) - 110,
            y: 60 + (childPos?.y ?? 0) - 30,
          };
          result.push(child);
        }
      }
    }
  }

  return result;
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
export function validateEventChain(
  events: SpecialModeEvent[],
  connections?: FlowConnection[]
): ValidationIssue[] {
  const issues: ValidationIssue[] = [];
  if (events.length === 0) return issues;

  const eventIds = new Set(events.map((e) => e.id));

  // 循环体内子节点集合，用于排除误报
  const loopChildSet = connections
    ? inferLoopChildren(events, connections)
    : new Map<string, string>();

  // 收集 loop 回边（子节点 → loop 自身），排除在环路检测之外
  const loopBackTargets = new Set<string>();
  if (connections) {
    for (const conn of connections) {
      const parentLoopId = loopChildSet.get(conn.sourceNodeId);
      if (parentLoopId && conn.targetNodeId === parentLoopId) {
        loopBackTargets.add(conn.sourceNodeId);
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

  // 检查断开的连线
  // （已由上方 connections 检查覆盖，无需额外检查）

  // 检测环路（DFS）— 排除 loop 回边
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

  function hasCycle(id: string): boolean {
    if (visiting.has(id)) return true;
    if (visited.has(id)) return false;
    visiting.add(id);
    for (const nextId of adj.get(id) || []) {
      // 跳过已知的 loop 回边
      if (
        loopBackTargets.has(id) &&
        loopChildSet.has(nextId) &&
        loopChildSet.get(nextId) === loopChildSet.get(id)
      ) {
        // 这是从 loop 子节点回到 loop 自身的合法回边，跳过
        // 但只有当 target 是 loop 节点自身时才跳过
        if (nextId === loopChildSet.get(id)) continue;
      }
      if (eventIds.has(nextId) && hasCycle(nextId)) return true;
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

  // 检查孤儿节点（没有任何事件连接到它，且它不是 trigger 且不在 loop body 中）
  const triggerEvent = events.find((e) => e.type === 'trigger') || events[0];
  const isTargetOf = new Set<string>();

  if (connections && connections.length > 0) {
    for (const conn of connections) {
      isTargetOf.add(conn.targetNodeId);
    }
  }

  for (const evt of events) {
    if (evt.type === 'trigger') continue;
    // loop body 内的第一个子节点由 out_body 连接，不算孤儿
    if (loopChildSet.has(evt.id)) continue;
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
