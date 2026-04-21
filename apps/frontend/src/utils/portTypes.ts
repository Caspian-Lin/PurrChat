/**
 * 端口类型系统 — Bot Studio 事件链编辑器
 *
 * 定义节点的 typed input/output 端口、连线模型、类型兼容性规则，
 * 以及各节点类型的默认端口配置。
 */

// ─── 基础类型 ────────────────────────────────────────────────

export type PortDataType = 'string' | 'number' | 'boolean' | 'trigger' | 'any';

export type EventType =
  | 'trigger'
  | 'end'
  | 'wait'
  | 'if'
  | 'loop'
  | 'llm'
  | 'builtin'
  | 'python'
  | 'reply'
  | 'template';

// ─── 端口 & 连线 ────────────────────────────────────────────

export interface EventPort {
  id: string;
  name: string;
  dataType: PortDataType;
  direction: 'input' | 'output';
}

export interface FlowConnection {
  id: string;
  sourceNodeId: string;
  sourcePortId: string;
  targetNodeId: string;
  targetPortId: string;
}

// ─── 端口颜色（设计系统内低饱和度色板） ──────────────────────

export const PORT_COLORS: Record<PortDataType, string> = {
  trigger: '#5A8F4E',
  string: '#3B82F6',
  number: '#E6A23C',
  boolean: '#F56C6C',
  any: '#909399',
};

// ─── 节点类型元信息 ──────────────────────────────────────────

export interface NodeTypeMeta {
  label: string;
  icon: string;
  category: 'control' | 'process' | 'output';
  description: string;
}

export const NODE_TYPE_META: Record<EventType, NodeTypeMeta> = {
  trigger: { label: '触发', icon: '🚀', category: 'control', description: '起始节点' },
  end: { label: '结束', icon: '⏹', category: 'control', description: '结束节点' },
  wait: { label: '等待', icon: '⏳', category: 'control', description: '等待条件' },
  if: { label: '条件', icon: '◇', category: 'control', description: '条件分支' },
  loop: { label: '循环', icon: '↻', category: 'control', description: '循环' },
  llm: { label: 'LLM', icon: '🧠', category: 'process', description: 'LLM 调用' },
  builtin: { label: '内置', icon: '⚙', category: 'process', description: '内置事件' },
  python: { label: 'Python', icon: '🐍', category: 'process', description: 'Python 脚本' },
  template: { label: '模板', icon: '📋', category: 'process', description: '模板渲染' },
  reply: { label: '回复', icon: '💬', category: 'output', description: '发送回复' },
};

// ─── 类型兼容性 ──────────────────────────────────────────────

export function isPortCompatible(source: PortDataType, target: PortDataType): boolean {
  if (source === target) return true;
  if (source === 'any' || target === 'any') return true;
  if (source === 'trigger' || target === 'trigger') return false;
  if (source === 'number' && target === 'string') return true;
  return false;
}

// ─── 默认端口配置 ────────────────────────────────────────────

function ports(
  inputs: [string, PortDataType, string][],
  outputs: [string, PortDataType, string][]
): EventPort[] {
  return [
    ...inputs.map(
      ([id, dataType, name]): EventPort => ({ id, name, dataType, direction: 'input' })
    ),
    ...outputs.map(
      ([id, dataType, name]): EventPort => ({ id, name, dataType, direction: 'output' })
    ),
  ];
}

const DEFAULT_PORTS: Record<EventType, EventPort[]> = {
  trigger: ports([], [['out_exec', 'trigger', '执行']]),
  end: ports([['in_exec', 'trigger', '执行']], []),
  reply: ports(
    [
      ['in_exec', 'trigger', '执行'],
      ['in_content', 'string', '内容'],
    ],
    []
  ),
  wait: ports(
    [['in_exec', 'trigger', '执行']],
    [
      ['out_exec', 'trigger', '执行'],
      ['out_user_input', 'string', '用户输入'],
    ]
  ),
  if: ports(
    [
      ['in_exec', 'trigger', '执行'],
      ['in_condition', 'boolean', '条件'],
    ],
    [
      ['out_true', 'trigger', '真'],
      ['out_false', 'trigger', '假'],
    ]
  ),
  loop: ports(
    [
      ['in_exec', 'trigger', '执行'],
      ['in_condition', 'boolean', '条件'],
    ],
    [
      ['out_body', 'trigger', '循环体'],
      ['out_done', 'trigger', '完成'],
    ]
  ),
  llm: ports(
    [
      ['in_exec', 'trigger', '执行'],
      ['in_prompt', 'string', '提示词'],
    ],
    [
      ['out_exec', 'trigger', '执行'],
      ['out_output', 'string', '输出'],
    ]
  ),
  builtin: ports(
    [
      ['in_exec', 'trigger', '执行'],
      ['in_input', 'string', '输入'],
    ],
    [
      ['out_exec', 'trigger', '执行'],
      ['out_output', 'string', '输出'],
    ]
  ),
  python: ports(
    [
      ['in_exec', 'trigger', '执行'],
      ['in_input', 'string', '输入'],
    ],
    [
      ['out_exec', 'trigger', '执行'],
      ['out_output', 'string', '输出'],
    ]
  ),
  template: ports(
    [
      ['in_exec', 'trigger', '执行'],
      ['in_input', 'string', '输入'],
    ],
    [
      ['out_exec', 'trigger', '执行'],
      ['out_output', 'string', '输出'],
    ]
  ),
};

export function getDefaultPorts(eventType: EventType): EventPort[] {
  return DEFAULT_PORTS[eventType];
}

// ─── 连线校验 ────────────────────────────────────────────────

export function canConnect(sourcePort: EventPort, targetPort: EventPort): boolean {
  if (sourcePort.direction !== 'output' || targetPort.direction !== 'input') return false;
  if (sourcePort.id === targetPort.id) return false;
  return isPortCompatible(sourcePort.dataType, targetPort.dataType);
}

// ─── 工具函数 ────────────────────────────────────────────────

export function getPortById(ports: EventPort[], portId: string): EventPort | undefined {
  return ports.find((p) => p.id === portId);
}
