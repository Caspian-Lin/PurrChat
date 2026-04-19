import type { SpecialModeEvent } from '../models/types';
import type { Node, Edge } from '@vue-flow/core';

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
    default:
      return '';
  }
}

export function eventsToFlowNodes(
  events: SpecialModeEvent[],
  positions?: Record<string, { x: number; y: number }>
): Node[] {
  return events.map((evt, index) => ({
    id: evt.id,
    type: 'event',
    position: positions?.[evt.id] || { x: index * 280, y: 50 + (index % 3) * 120 },
    data: {
      label: evt.name,
      eventType: evt.type,
      summary: getEventSummary(evt),
      icon: typeIcons[evt.type] || '',
    },
  }));
}

export function eventsToFlowEdges(events: SpecialModeEvent[]): Edge[] {
  const edges: Edge[] = [];
  for (const evt of events) {
    for (const nextId of evt.next || []) {
      edges.push({
        id: `${evt.id}-${nextId}`,
        source: evt.id,
        target: nextId,
      });
    }
  }
  return edges;
}
