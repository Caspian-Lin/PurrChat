/**
 * 统一变量模型 — 所有节点共用的变量命名空间
 *
 * 规范变量路径（`${path}` 语法）：
 *   input.text              用户消息全文
 *   input.args.N            用户消息第 N 个词（1-indexed）
 *   sender.id               发送者 ID
 *   sender.name             发送者名称
 *   conversation.id         会话 ID
 *   history.messages        消息历史（格式化字符串）
 *   nodes.<key>.outputs.<port>  节点输出端口值
 *   session.<name>          会话变量
 *   secrets.<name>          secret 引用（运行时解密）
 *
 * 设计见 docs/bot-engine/BOT_SYSTEM_AUDIT_2026-07-09.md §四
 */

/** 规范变量 scope */
export const VARIABLE_SCOPES = {
  input: 'input',
  sender: 'sender',
  conversation: 'conversation',
  history: 'history',
  nodes: 'nodes',
  session: 'session',
  secrets: 'secrets',
} as const;

export type VariableScope = (typeof VARIABLE_SCOPES)[keyof typeof VARIABLE_SCOPES];

/** 变量引用正则: ${scope.path} */
export const VARIABLE_REF_RE = /\$\{([^}]+)\}/g;

/** 提取 ${...} 引用中的路径 */
export function extractVariablePaths(text: string): string[] {
  const paths: string[] = [];
  const re = new RegExp(VARIABLE_REF_RE);
  let m: RegExpExecArray | null;
  while ((m = re.exec(text)) !== null) {
    paths.push(m[1].trim());
  }
  return paths;
}

/** 内置变量元信息（供 picker 渲染） */
export interface BuiltinVariableMeta {
  path: string;
  label: string;
  dataType: 'string' | 'number' | 'boolean';
  description: string;
}

/** 所有内置规范变量 */
export const BUILTIN_VARIABLES: BuiltinVariableMeta[] = [
  { path: 'input.text', label: '用户消息', dataType: 'string', description: '完整的用户输入文本' },
  { path: 'input.args.1', label: '第 1 个参数', dataType: 'string', description: '用户消息的第一个词' },
  { path: 'sender.id', label: '发送者 ID', dataType: 'string', description: '消息发送者的唯一标识' },
  { path: 'sender.name', label: '发送者名称', dataType: 'string', description: '消息发送者的显示名' },
  { path: 'conversation.id', label: '会话 ID', dataType: 'string', description: '当前会话的唯一标识' },
  { path: 'history.messages', label: '消息历史', dataType: 'string', description: '最近的对话历史' },
];

/** 从 node key + port 构建 nodes.<key>.outputs.<port> 引用路径 */
export function nodeOutputPath(nodeKey: string, portId: string): string {
  return `nodes.${nodeKey}.outputs.${portId}`;
}

/** 解析 nodes.<key>.outputs.<port> 路径 */
export function parseNodeOutputPath(path: string): { nodeKey: string; portId: string } | null {
  const m = path.match(/^nodes\.([^.]+)\.outputs\.([^.]+)$/);
  if (!m) return null;
  return { nodeKey: m[1], portId: m[2] };
}

/** 判断路径是否为 secrets 引用 */
export function isSecretPath(path: string): boolean {
  return path.startsWith('secrets.');
}

/** 从 secrets.<name> 路径中提取 secret name */
export function parseSecretName(path: string): string | null {
  const m = path.match(/^secrets\.([a-zA-Z0-9_]+)$/);
  return m ? m[1] : null;
}

/**
 * 为节点生成稳定 key。
 * key 基于 type + 序号，在 workflow 内唯一。
 * 改名不影响 key，因此引用不会断裂。
 */
export function generateNodeKey(type: string, index: number): string {
  return `${type}_${index}`;
}
