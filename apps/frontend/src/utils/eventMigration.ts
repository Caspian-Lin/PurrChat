/**
 * 事件数据迁移 — 旧 next[] 格式到新 connections[] 格式的自动转换
 */

import type { SpecialModeEvent, FlowConnection, EventPort } from '../models/types';
import { getDefaultPorts } from './portTypes';

/** 为事件生成端口：已有则复用，否则按类型生成默认值 */
export function buildDefaultPorts(event: SpecialModeEvent): EventPort[] {
  if (event.ports && event.ports.length > 0) return event.ports;
  return getDefaultPorts(event.type);
}

/** 将旧 next[] 关系转换为 FlowConnection[] */
export function migrateLegacyConnections(events: SpecialModeEvent[]): FlowConnection[] {
  const eventMap = new Map(events.map((e) => [e.id, e]));
  const seen = new Set<string>();
  const connections: FlowConnection[] = [];

  for (const event of events) {
    if (!event.next?.length) continue;

    const sourcePorts = buildDefaultPorts(event);
    const outExec = sourcePorts.find((p) => p.direction === 'output' && p.dataType === 'trigger');
    if (!outExec) continue;

    for (const targetId of event.next) {
      const target = eventMap.get(targetId);
      if (!target) continue;

      const targetPorts = buildDefaultPorts(target);
      const inExec = targetPorts.find((p) => p.direction === 'input' && p.dataType === 'trigger');
      if (!inExec) continue;

      const key = `${event.id}->${targetId}`;
      if (seen.has(key)) continue;
      seen.add(key);

      connections.push({
        id: `conn_${event.id}_${targetId}`,
        sourceNodeId: event.id,
        sourcePortId: outExec.id,
        targetNodeId: targetId,
        targetPortId: inExec.id,
      });
    }
  }

  return connections;
}

/** 确保所有事件都有 ports 字段（始终返回新对象，避免响应式引用循环） */
export function ensurePorts(events: SpecialModeEvent[]): SpecialModeEvent[] {
  return events.map((event) => {
    if (event.ports && event.ports.length > 0) return { ...event };
    return { ...event, ports: getDefaultPorts(event.type) };
  });
}

/** 检查是否需要从 next[] 迁移到 connections[] */
export function needsMigration(
  events: SpecialModeEvent[],
  connections?: FlowConnection[]
): boolean {
  const hasLegacy = events.some((e) => e.next && e.next.length > 0);
  const hasConnections = connections && connections.length > 0;
  return hasLegacy && !hasConnections;
}
