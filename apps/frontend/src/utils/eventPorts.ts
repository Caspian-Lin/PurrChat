/**
 * 事件端口运行时工具 — 构建默认端口 & 确保端口完整性
 */

import type { WorkflowEvent, EventPort } from '../models/types';
import { getDefaultPorts } from './portTypes';

/** 为事件生成端口：已有则复用，否则按类型生成默认值 */
export function buildDefaultPorts(event: WorkflowEvent): EventPort[] {
  if (event.ports && event.ports.length > 0) return event.ports;

  const defaults = getDefaultPorts(event.type);

  // 动态端口类型
  if (event.type === 'switch') {
    return buildSwitchPorts(event, defaults);
  }
  if (event.type === 'merge') {
    return buildMergePorts(event, defaults);
  }

  return defaults;
}

/** Switch 节点：根据 config.cases 动态生成输出端口 */
function buildSwitchPorts(event: WorkflowEvent, defaults: EventPort[]): EventPort[] {
  const cases = (event.config?.cases || []) as { value: string; label: string }[];
  const inputs = defaults.filter((p) => p.direction === 'input');
  const outputs: EventPort[] = cases.map((c, i) => ({
    id: `out_case_${i}`,
    name: c.label || `分支 ${i + 1}`,
    dataType: 'trigger' as const,
    direction: 'output' as const,
  }));
  outputs.push({
    id: 'out_default',
    name: '默认',
    dataType: 'trigger',
    direction: 'output',
  });
  return [...inputs, ...outputs];
}

/** Merge 节点：根据 config.input_count 动态生成输入端口 */
function buildMergePorts(event: WorkflowEvent, _defaults: EventPort[]): EventPort[] {
  const count = Math.max(2, (event.config?.input_count as number) || 2);
  const inputs: EventPort[] = Array.from({ length: count }, (_, i) => ({
    id: `in_exec_${i}`,
    name: `输入 ${i + 1}`,
    dataType: 'trigger' as const,
    direction: 'input' as const,
  }));
  const outputs = [
    { id: 'out_exec', name: '执行', dataType: 'trigger' as const, direction: 'output' as const },
  ];
  return [...inputs, ...outputs];
}

/** 确保所有事件都有 ports 字段（始终返回新对象，避免响应式引用循环） */
export function ensurePorts(events: WorkflowEvent[]): WorkflowEvent[] {
  return events.map((event) => {
    if (event.ports && event.ports.length > 0) return { ...event };
    return { ...event, ports: getDefaultPorts(event.type) };
  });
}
