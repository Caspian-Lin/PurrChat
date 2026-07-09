/**
 * Capability 权限模型 — Bot Studio 节点能力声明
 *
 * 定义 Bot 工作流节点所需的能力(capability)，用于发布时推导、安装时授权、
 * 运行时强制校验三层权限链。设计见 docs/bot-engine/BOT_APP_MODEL.md §2。
 */

import type { EventType } from './ports.js';

// ─── Capability 常量与类型 ───────────────────────────────────

export const Capability = {
  /** 读取触发消息（几乎所有 Bot 都需要） */
  ReadTrigger: 'messages:read_trigger',
  /** 读取上下文历史消息（安装者可关闭） */
  ReadHistory: 'messages:read_history',
  /** 发送回复消息 */
  Send: 'messages:send',
  /** 读取成员列表（预留） */
  MembersRead: 'members:read',
  /** 数据外发到第三方服务 */
  NetworkExternal: 'network:external',
  /** 使用 owner 配置的加密密钥 */
  SecretsUse: 'secrets:use',
} as const;

export type Capability = (typeof Capability)[keyof typeof Capability];

export const ALL_CAPABILITIES: Capability[] = [
  Capability.ReadTrigger,
  Capability.ReadHistory,
  Capability.Send,
  Capability.MembersRead,
  Capability.NetworkExternal,
  Capability.SecretsUse,
];

// ─── Capability 元数据（前端展示用） ─────────────────────────

export interface CapabilityMeta {
  label: string;
  icon: string;
  description: string;
  /** 敏感能力：安装时需强调展示（如外发网络） */
  sensitive?: boolean;
}

export const CAPABILITY_META: Record<Capability, CapabilityMeta> = {
  [Capability.ReadTrigger]: {
    label: '读取触发消息',
    icon: '✉️',
    description: '读取你发送的触发消息',
  },
  [Capability.ReadHistory]: {
    label: '读取历史消息',
    icon: '📜',
    description: '读取会话的上下文历史消息',
  },
  [Capability.Send]: {
    label: '发送回复',
    icon: '📤',
    description: '在会话中发送回复消息',
  },
  [Capability.MembersRead]: {
    label: '读取成员',
    icon: '👥',
    description: '读取会话成员信息',
  },
  [Capability.NetworkExternal]: {
    label: '访问外部网络',
    icon: '🌐',
    description: '将对话内容发送到外部服务（LLM / 自动化平台 / webhook）',
    sensitive: true,
  },
  [Capability.SecretsUse]: {
    label: '使用密钥',
    icon: '🔑',
    description: '使用创建者配置的加密密钥',
    sensitive: true,
  },
};

// ─── 节点类型 → 所需 Capability 映射 ─────────────────────────

/**
 * 每种节点类型固有的 required capabilities。
 * 发布时遍历工作流节点图，对所有节点的 capability 取并集，
 * 即得到整个 Bot 的 requested_capabilities。
 */
export const NODE_CAPABILITIES: Partial<Record<EventType, Capability[]>> = {
  trigger: [Capability.ReadTrigger],
  llm: [Capability.NetworkExternal, Capability.ReadHistory],
  tool: [Capability.NetworkExternal],
  dify: [Capability.NetworkExternal],
  n8n: [Capability.NetworkExternal],
  history: [Capability.ReadHistory],
  reply: [Capability.Send],
  template: [Capability.Send],
};

/**
 * 获取某节点类型所需的 capabilities。
 * 未声明（控制流节点等）返回空数组。
 */
export function getNodeCapabilities(nodeType: EventType): Capability[] {
  return NODE_CAPABILITIES[nodeType] ?? [];
}

/**
 * 判断 capability 是否为敏感（安装时需强调展示）。
 */
export function isSensitiveCapability(cap: string): boolean {
  return cap === Capability.NetworkExternal || cap === Capability.SecretsUse;
}
