/**
 * 统一变量解析器 — 所有节点共用唯一的变量替换入口
 *
 * 支持 `${path}` 规范格式和所有遗留格式：
 *   ${input.text} / ${sender.name} / ${nodes.<key>.outputs.<port>}  规范格式
 *   {nodeName.portName}  遗留人类可读格式（via nameResolver）
 *   {args} / {args:N}    遗留参数格式
 *   $nodeId:portId       遗留机器格式
 *   $evtId.output        遗留事件输出格式
 *   $variableName        遗留变量格式
 *
 * 设计见 docs/bot-engine/BOT_SYSTEM_AUDIT_2026-07-09.md §四
 */

import {
  parseNodeOutputPath,
  parseSecretName,
  VARIABLE_REF_RE,
} from '@purrchat/workflow-types';

/** 变量解析所需完整上下文 */
export interface ResolveContext {
  /** nodeId → { portId → value }，节点执行后写入 */
  nodeOutputs: Record<string, Record<string, string>>;
  /** 旧版 "nodeName.portName" → "nodeId:portId" 映射 */
  nameResolver: Record<string, string>;
  /** nodeKey → nodeId，用于 ${nodes.<key>.outputs.<port>} */
  nodeKeyMap: Record<string, string>;
  /** 旧版变量表（向后兼容 $variableName） */
  variables: Record<string, string>;
  /** 旧版事件输出（向后兼容 $evtId.output） */
  eventOutputs: Record<string, string>;
  /** 原始用户消息 */
  rawInput: string;
  /** 发送者 ID */
  senderId: string;
  /** 发送者名称 */
  senderName: string;
  /** 会话 ID */
  conversationId: string;
  /** 消息历史 */
  history: Array<{ role: string; content: string }>;
  /** 运行时解密后的 secret */
  secrets: Record<string, string>;
  /** 会话变量 */
  session: Record<string, string>;
}

/**
 * 统一模板解析：替换字符串中所有变量引用。
 * 优先解析规范 `${path}` 格式，然后回退到遗留格式。
 */
export function resolveTemplate(template: string, ctx: ResolveContext): string {
  if (!template) return '';

  let s = template;

  // 1. 规范格式 ${path}（最高优先级）
  s = s.replace(VARIABLE_REF_RE, (_match, path: string) => {
    const val = resolveCanonicalPath(path.trim(), ctx);
    return val !== undefined ? val : `\${${path}}`;
  });

  // 2. 遗留 {name.port} 格式
  s = s.replace(/\{([^}]+)\}/g, (match, ref: string) => {
    // 2a. {args} / {args:N} 特殊处理
    if (ref === 'args') {
      return ctx.rawInput.trim();
    }
    const argsMatch = ref.match(/^args:(\d+)$/);
    if (argsMatch) {
      const i = parseInt(argsMatch[1], 10) - 1;
      const parts = ctx.rawInput.trim().split(/\s+/);
      return i >= 0 && i < parts.length ? parts[i]! : '';
    }

    // 2b. {name.port} 格式 → nameResolver
    const mappedKey = ctx.nameResolver[ref];
    if (mappedKey) {
      const [nodeId, portId] = mappedKey.split(':');
      const val = ctx.nodeOutputs[nodeId]?.[portId];
      if (val !== undefined) return val;
    }

    // 2c. {variableName} 遗留裸变量（向后兼容 template/builtin 节点）
    if (ctx.variables[ref] !== undefined) {
      return ctx.variables[ref];
    }

    return match;
  });

  // 3. 遗留 $nodeId:portId 格式
  for (const [nodeId, ports] of Object.entries(ctx.nodeOutputs)) {
    for (const [portId, val] of Object.entries(ports)) {
      s = s.replaceAll(`$${nodeId}:${portId}`, val);
    }
  }

  // 4. 遗留 $evtId.output 格式
  for (const [evtId, output] of Object.entries(ctx.eventOutputs)) {
    s = s.replaceAll(`$${evtId}.output`, output);
  }

  // 5. 遗留 $variableName 格式
  for (const [key, value] of Object.entries(ctx.variables)) {
    s = s.replaceAll(`$${key}`, value);
  }

  return s;
}

/**
 * 解析规范变量路径 ${scope.path}
 * 返回 undefined 表示路径无法解析（保留原始 ${path}）
 */
function resolveCanonicalPath(path: string, ctx: ResolveContext): string | undefined {
  // input.text
  if (path === 'input.text') {
    return ctx.rawInput;
  }

  // input.args.N
  if (path.startsWith('input.args.')) {
    const n = parseInt(path.slice('input.args.'.length), 10);
    if (isNaN(n) || n < 1) return undefined;
    const parts = ctx.rawInput.trim().split(/\s+/);
    return n <= parts.length ? parts[n - 1] : '';
  }

  // sender.id / sender.name
  if (path === 'sender.id') return ctx.senderId;
  if (path === 'sender.name') return ctx.senderName;

  // conversation.id
  if (path === 'conversation.id') return ctx.conversationId;

  // history.messages
  if (path === 'history.messages') {
    return formatHistory(ctx.history);
  }

  // nodes.<key>.outputs.<port>
  const nodeRef = parseNodeOutputPath(path);
  if (nodeRef) {
    const nodeId = ctx.nodeKeyMap[nodeRef.nodeKey];
    if (!nodeId) return undefined;
    const val = ctx.nodeOutputs[nodeId]?.[nodeRef.portId];
    return val;
  }

  // session.<name>
  if (path.startsWith('session.')) {
    const name = path.slice('session.'.length);
    return ctx.session[name];
  }

  // secrets.<name>
  if (path.startsWith('secrets.')) {
    const name = parseSecretName(path);
    if (name && ctx.secrets[name] !== undefined) {
      return ctx.secrets[name];
    }
    return undefined;
  }

  return undefined;
}

/** 将消息历史格式化为可读字符串 */
function formatHistory(history: Array<{ role: string; content: string }>): string {
  if (!history || history.length === 0) return '';
  return history
    .map((m) => {
      const speaker = m.role === 'user' ? '用户' : m.role === 'assistant' ? 'AI' : m.role;
      return `${speaker}: ${m.content}`;
    })
    .join('\n');
}
