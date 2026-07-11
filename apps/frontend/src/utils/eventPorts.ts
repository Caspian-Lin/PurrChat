/**
 * 事件端口运行时工具 — 构建默认端口 & 确保端口完整性
 */

import type { WorkflowEvent, EventPort } from '../models/types';
import { getPortsForConfig } from './portTypes';

/** 为事件生成端口：已有则复用，否则按类型生成默认值 */
export function buildDefaultPorts(event: WorkflowEvent): EventPort[] {
  if (event.ports && event.ports.length > 0) return event.ports;

  return getPortsForConfig(event.type, event.config);
}

/** 确保所有事件都有 ports 字段（始终返回新对象，避免响应式引用循环） */
export function ensurePorts(events: WorkflowEvent[]): WorkflowEvent[] {
  return events.map((event) => {
    if (event.ports && event.ports.length > 0) return { ...event };
    return { ...event, ports: getPortsForConfig(event.type, event.config) };
  });
}
