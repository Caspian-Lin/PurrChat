/**
 * 端口值脱敏工具（生产 trace 与调试 trace 共享）
 *
 * 规则：
 * - key 含 secret/api_key/token/password → [REDACTED]
 * - 值超过 500 字符 → 截断并追加 "..."
 */

/** 脱敏端口值 */
export function sanitizePorts(ports: Record<string, string>): Record<string, string> {
  const result: Record<string, string> = {};
  for (const [key, value] of Object.entries(ports)) {
    const keyLower = key.toLowerCase();
    if (
      keyLower.includes('secret') ||
      keyLower.includes('api_key') ||
      keyLower.includes('token') ||
      keyLower.includes('password')
    ) {
      result[key] = '[REDACTED]';
    } else {
      result[key] = value.length > 500 ? value.slice(0, 500) + '...' : value;
    }
  }
  return result;
}
