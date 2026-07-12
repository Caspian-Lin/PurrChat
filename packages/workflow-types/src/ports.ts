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
  | 'switch'
  | 'merge'
  | 'tool'
  | 'dify'
  | 'n8n'
  | 'llm'
  | 'builtin'
  | 'reply'
  | 'template'
  | 'history';

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
  loop: { label: '循环', icon: '↻', category: 'control', description: '循环执行子链（回环模式）' },
  switch: { label: '分支', icon: '⑂', category: 'control', description: '多条件分支路由' },
  merge: { label: '汇聚', icon: '⑃', category: 'control', description: '多分支汇聚' },
  tool: { label: '工具', icon: '🔌', category: 'process', description: 'HTTP 请求 / 外部工具调用' },
  dify: { label: 'Dify', icon: '🔮', category: 'process', description: '调用 Dify 工作流' },
  n8n: { label: 'n8n', icon: '⚡', category: 'process', description: '调用 n8n Webhook' },
  llm: { label: 'LLM', icon: '🧠', category: 'process', description: 'LLM 调用' },
  builtin: { label: '内置', icon: '⚙', category: 'process', description: '内置事件' },
  template: { label: '模板', icon: '📋', category: 'process', description: '模板渲染' },
  reply: { label: '回复', icon: '💬', category: 'output', description: '发送回复' },
  history: { label: '历史消息', icon: '📜', category: 'process', description: '获取历史消息记录' },
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
  trigger: ports(
    [],
    [
      ['out_exec', 'trigger', '执行'],
      ['out_input', 'string', '用户消息'],
      ['out_username', 'string', '发送者'],
      ['out_time', 'string', '时间'],
      ['out_args', 'string', '参数'],
    ]
  ),
  end: ports([['in_exec', 'trigger', '执行']], []),
  reply: ports(
    [
      ['in_exec', 'trigger', '执行'],
      ['in_content', 'string', '内容'],
    ],
    [['out_exec', 'trigger', '执行']]
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
  // Switch: 1 input + N cases (config.cases) + 1 default
  switch: ports(
    [
      ['in_exec', 'trigger', '执行'],
      ['in_value', 'any', '匹配值'],
    ],
    [
      ['out_case_0', 'trigger', '分支 1'],
      ['out_case_1', 'trigger', '分支 2'],
      ['out_default', 'trigger', '默认'],
    ]
  ),
  // Merge: N trigger inputs + 1 trigger output
  merge: ports(
    [
      ['in_exec_0', 'trigger', '输入 1'],
      ['in_exec_1', 'trigger', '输入 2'],
    ],
    [['out_exec', 'trigger', '执行']]
  ),
  // Tool: HTTP request
  tool: ports(
    [
      ['in_exec', 'trigger', '执行'],
      ['in_body', 'string', '请求体'],
    ],
    [
      ['out_exec', 'trigger', '执行'],
      ['out_output', 'string', '响应'],
      ['out_status', 'number', '状态码'],
      ['out_error', 'string', '错误'],
    ]
  ),
  // Dify: external workflow
  dify: ports(
    [
      ['in_exec', 'trigger', '执行'],
      ['in_input', 'string', '输入'],
    ],
    [
      ['out_exec', 'trigger', '执行'],
      ['out_output', 'string', '输出'],
      ['out_error', 'string', '错误'],
    ]
  ),
  // n8n: external webhook
  n8n: ports(
    [
      ['in_exec', 'trigger', '执行'],
      ['in_input', 'string', '输入'],
    ],
    [
      ['out_exec', 'trigger', '执行'],
      ['out_output', 'string', '输出'],
      ['out_error', 'string', '错误'],
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
      ['out_error', 'string', '错误'],
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
  history: ports(
    [
      ['in_exec', 'trigger', '执行'],
      ['in_count', 'number', '消息数量'],
    ],
    [
      ['out_exec', 'trigger', '执行'],
      ['out_history', 'string', '历史记录'],
    ]
  ),
};

export function getDefaultPorts(eventType: EventType): EventPort[] {
  return DEFAULT_PORTS[eventType];
}

/** Build control-flow ports from persisted configuration. */
export function getPortsForConfig(
  eventType: EventType,
  config: Record<string, unknown> = {},
): EventPort[] {
  if (eventType === 'if') {
    const branches = Array.isArray(config.branches) ? config.branches : [];
    const elifPorts = branches.slice(1).map((_, index): EventPort => ({
      id: `out_elif_${index}`,
      name: `否则如果 ${index + 1}`,
      dataType: 'trigger',
      direction: 'output',
    }));
    return [
      ...DEFAULT_PORTS.if.filter((port) => port.id === 'in_exec' || port.id === 'out_true'),
      ...elifPorts,
      ...DEFAULT_PORTS.if.filter((port) => port.id === 'out_false'),
    ];
  }

  if (eventType === 'switch') {
    const cases = Array.isArray(config.cases) ? config.cases : [];
    const inputs = DEFAULT_PORTS.switch.filter((port) => port.direction === 'input');
    const outputs = cases.map((item, index): EventPort => ({
      id: `out_case_${index}`,
      name: typeof item === 'object' && item && typeof (item as { label?: unknown }).label === 'string'
        ? (item as { label: string }).label || `分支 ${index + 1}`
        : `分支 ${index + 1}`,
      dataType: 'trigger',
      direction: 'output',
    }));
    return [...inputs, ...outputs, { id: 'out_default', name: '默认', dataType: 'trigger', direction: 'output' }];
  }

  if (eventType === 'merge') {
    const configured = typeof config.input_count === 'number' ? config.input_count : 2;
    const inputCount = Math.max(2, Math.min(10, Math.trunc(configured)));
    return [
      ...Array.from({ length: inputCount }, (_, index): EventPort => ({
        id: `in_exec_${index}`,
        name: `输入 ${index + 1}`,
        dataType: 'trigger',
        direction: 'input',
      })),
      { id: 'out_exec', name: '执行', dataType: 'trigger', direction: 'output' },
    ];
  }

  return getDefaultPorts(eventType);
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
