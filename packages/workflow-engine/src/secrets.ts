/**
 * Secret 引用解析
 *
 * 工作流配置中用 `secrets.<name>` 代替明文（如 api_key、webhook auth），
 * 运行时由 compiler 在节点执行前解析为实际解密值。
 *
 * 设计见 docs/bot-engine/BOT_APP_MODEL.md §5。
 */

import { Capability } from '@purrchat/workflow-types';

/** secrets.<name> 引用占位符正则 */
const SECRET_REF_RE = /^secrets\.([a-zA-Z0-9_]+)$/;

/** 递归扫描 config 中出现的所有 secrets.<name> 引用的 key name */
export function extractSecretRefs(value: unknown): string[] {
  const refs: string[] = [];
  collectRefs(value, refs);
  return [...new Set(refs)];
}

function collectRefs(value: unknown, out: string[]): void {
  if (typeof value === 'string') {
    const m = value.trim().match(SECRET_REF_RE);
    if (m) out.push(m[1]);
    return;
  }
  if (Array.isArray(value)) {
    for (const v of value) collectRefs(v, out);
    return;
  }
  if (value && typeof value === 'object') {
    for (const v of Object.values(value)) collectRefs(v, out);
  }
}

/**
 * 递归解析 config 中的 secrets.<name> 引用，替换为实际值。
 * - 完整匹配 `secrets.<name>` → 替换为该 secret 的原始值（保留类型）
 * - 缺失的引用替换为空字符串（不阻断执行，节点自行处理缺失情况）
 */
export function resolveSecrets(
  config: unknown,
  secrets: Record<string, string> | undefined,
): unknown {
  if (!secrets || Object.keys(secrets).length === 0) return config;
  return resolveNode(config, secrets);
}

function resolveNode(value: unknown, secrets: Record<string, string>): unknown {
  if (typeof value === 'string') {
    const m = value.trim().match(SECRET_REF_RE);
    if (m) return secrets[m[1]] ?? '';
    return value;
  }
  if (Array.isArray(value)) {
    return value.map((v) => resolveNode(v, secrets));
  }
  if (value && typeof value === 'object') {
    const out: Record<string, unknown> = {};
    for (const [k, v] of Object.entries(value)) {
      out[k] = resolveNode(v, secrets);
    }
    return out;
  }
  return value;
}

/**
 * 校验节点 config 引用了 secret 但未授予 secrets:use capability。
 * 返回缺失的 capability（空数组表示通过）。
 */
export function checkSecretCapability(
  config: unknown,
  grantedCapabilities: string[],
): string[] {
  const refs = extractSecretRefs(config);
  if (refs.length === 0) return [];
  const granted = new Set(grantedCapabilities);
  if (!granted.has(Capability.SecretsUse)) {
    return [Capability.SecretsUse];
  }
  return [];
}
