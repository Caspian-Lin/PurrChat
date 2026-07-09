/**
 * Capability 推导与运行时校验
 *
 * - deriveCapabilities: 发布时遍历工作流节点图，按节点类型推导所需 capability 集合
 * - checkNodeCapabilities: 运行时执行节点前校验 granted ⊇ required
 *
 * 设计见 docs/bot-engine/BOT_APP_MODEL.md §2。
 */

import { getNodeCapabilities } from '@purrchat/workflow-types';
import type { Blueprint, BlueprintNode } from './types.js';

/**
 * 遍历工作流节点图，推导 Bot 所需的全部 capability（取并集）。
 * 发布时调用，结果写入 bot_apps.requested_capabilities。
 */
export function deriveCapabilities(blueprint: Blueprint): string[] {
  const set = new Set<string>();
  for (const node of blueprint.nodes) {
    const caps = getNodeCapabilities(node.type as any);
    for (const cap of caps) {
      set.add(cap);
    }
  }
  return [...set];
}

/**
 * 校验给定节点是否满足 capability 要求。
 * 返回缺失的 capability 列表（空数组表示全部满足）。
 */
export function getMissingCapabilities(node: BlueprintNode, granted: string[]): string[] {
  const required = getNodeCapabilities(node.type as any);
  if (required.length === 0) return [];
  const grantedSet = new Set(granted);
  return required.filter((cap) => !grantedSet.has(cap));
}
