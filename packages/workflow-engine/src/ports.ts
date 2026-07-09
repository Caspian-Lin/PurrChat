import type { FlowConnection } from '@purrchat/workflow-types';

/**
 * 变量解析所需的最小上下文结构。
 * ExecutionContext 与 NodeContext 均满足此结构。
 */
export interface VariableResolveContext {
  nodeOutputs: Record<string, Record<string, string>>;
  eventOutputs: Record<string, string>;
  variables: Record<string, string>;
  nameResolver: Record<string, string>;
}

// ─── 端口值解析 ──────────────────────────────────────────────

/**
 * 解析节点的输入端口值
 * 优先级：直接存储值 > 连接源端口值 > trigger/exec 默认 true > 空字符串
 */
export function resolveInputPorts(
  nodeId: string,
  connections: FlowConnection[],
  context: VariableResolveContext,
): Record<string, string> {
  const result: Record<string, string> = {};

  // 找到所有指向该节点的连接
  for (const conn of connections) {
    if (conn.targetNodeId === nodeId) {
      const srcKey = `${conn.sourceNodeId}:${conn.sourcePortId}`;
      const val = context.nodeOutputs[conn.sourceNodeId]?.[conn.sourcePortId];
      if (val !== undefined) {
        result[conn.targetPortId] = val;
      }
    }
  }

  return result;
}

/**
 * 获取端口值（带默认值）
 */
export function getPortValue(
  nodeId: string,
  portId: string,
  connections: FlowConnection[],
  context: VariableResolveContext,
): string {
  const key = `${nodeId}:${portId}`;

  // 检查直接存储的值
  const direct = context.nodeOutputs[nodeId]?.[portId];
  if (direct !== undefined) return direct;

  // 查找输入连接，获取源端口值
  for (const conn of connections) {
    if (conn.targetNodeId === nodeId && conn.targetPortId === portId) {
      const srcVal = context.nodeOutputs[conn.sourceNodeId]?.[conn.sourcePortId];
      if (srcVal !== undefined) return srcVal;
    }
  }

  // trigger/exec 端口默认返回 true
  if (portId.includes('exec') || portId.includes('trigger')) {
    return 'true';
  }

  return '';
}

// ─── 变量替换 ────────────────────────────────────────────────

/**
 * 替换模板中的变量引用
 * 支持两种格式：
 *   - {nodeName.portName} — 人类可读格式（优先解析）
 *   - $nodeID:portID / $evtID.output / $variable — 机器格式（向后兼容）
 */
export function replaceVariables(
  s: string,
  context: VariableResolveContext,
): string {
  // 替换 {name.port} 格式（最高优先级）
  s = s.replace(/\{([^}]+)\}/g, (match, ref: string) => {
    const mappedKey = context.nameResolver[ref];
    if (mappedKey) {
      const [nodeId, portId] = mappedKey.split(':');
      const val = context.nodeOutputs[nodeId]?.[portId];
      if (val !== undefined) return val;
    }
    return match; // 未找到映射，原样返回
  });

  // 替换端口值引用 $nodeID:portID
  for (const [nodeId, ports] of Object.entries(context.nodeOutputs)) {
    for (const [portId, val] of Object.entries(ports)) {
      s = s.replaceAll(`$${nodeId}:${portId}`, val);
    }
  }

  // 替换事件输出引用 $evtID.output
  for (const [evtId, output] of Object.entries(context.eventOutputs)) {
    s = s.replaceAll(`$${evtId}.output`, output);
  }

  // 替换会话变量
  for (const [key, value] of Object.entries(context.variables)) {
    s = s.replaceAll(`$${key}`, value);
  }

  return s;
}

// ─── 条件求值 ────────────────────────────────────────────────

/**
 * 求值条件表达式
 * 支持格式：
 *   - "true" / "false" — 字面值
 *   - "left == right" / "left != right" — 字符串比较
 *   - 非空字符串视为 true
 */
export function evaluateCondition(
  condition: string,
  context: VariableResolveContext,
): boolean {
  if (!condition) return false;

  const resolved = replaceVariables(condition, context).trim();

  if (resolved === 'true') return true;
  if (resolved === 'false') return false;

  // 不等于比较
  if (resolved.includes('!=')) {
    const [left, right] = resolved.split('!=', 2);
    return left.trim() !== right.trim();
  }

  // 等于比较
  if (resolved.includes('==')) {
    const [left, right] = resolved.split('==', 2);
    return left.trim() === right.trim();
  }

  // 非空字符串视为 true
  return resolved !== '';
}

/**
 * 使用指定运算符比较两个值
 */
export function evaluateOperatorCondition(
  left: string,
  right: string,
  operator: string,
): boolean {
  left = left.trim();
  right = right.trim();

  switch (operator) {
    case '==':
      return left === right;
    case '!=':
      return left !== right;
    case 'contains':
      return left.includes(right);
    case '>':
      return compareNumeric(left, right) > 0;
    case '<':
      return compareNumeric(left, right) < 0;
    case 'startsWith':
      return left.startsWith(right);
    case 'endsWith':
      return left.endsWith(right);
    case 'regex':
      try {
        return new RegExp(right).test(left);
      } catch {
        return false;
      }
    default:
      return left === right;
  }
}

function compareNumeric(a: string, b: string): number {
  const af = parseFloat(a);
  const bf = parseFloat(b);
  if (isNaN(af) || isNaN(bf)) return a < b ? -1 : a > b ? 1 : 0;
  return af - bf;
}
