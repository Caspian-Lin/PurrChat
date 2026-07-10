/**
 * WorkflowDocument YAML 往返
 *
 * JSON ↔ YAML 语义等价：round-trip 不丢失 nodes、connections、endConditions。
 * Secret 安全：导出时将明文密钥替换为 secrets.<name> 引用占位符。
 */

import yaml from 'js-yaml';
import type { WorkflowDocument } from '@purrchat/workflow-types';
import {
  WORKFLOW_API_VERSION,
  WORKFLOW_KIND,
} from '@purrchat/workflow-types';

const SENSITIVE_KEYS = new Set([
  'api_key',
  'apikey',
  'secret',
  'token',
  'password',
  'webhook_secret',
  'authorization',
]);

export function documentToYaml(doc: WorkflowDocument): string {
  const header = `# PurrChat Bot Workflow\n# apiVersion: ${WORKFLOW_API_VERSION}\n`;
  const body = yaml.dump(doc, {
    indent: 2,
    lineWidth: 120,
    noRefs: true,
    sortKeys: false,
  });
  return header + body;
}

export function yamlToDocument(yamlStr: string): WorkflowDocument {
  const parsed = yaml.load(yamlStr);
  if (typeof parsed !== 'object' || parsed === null) {
    throw new Error('YAML 内容必须是对象');
  }
  return parsed as WorkflowDocument;
}

export function documentToJson(doc: WorkflowDocument): string {
  return JSON.stringify(doc, null, 2);
}

export function jsonToDocument(jsonStr: string): WorkflowDocument {
  const parsed = JSON.parse(jsonStr);
  if (typeof parsed !== 'object' || parsed === null) {
    throw new Error('JSON 内容必须是对象');
  }
  return parsed as WorkflowDocument;
}

/**
 * 导出安全版本：明文密钥替换为 secrets.<推断名> 引用。
 * 不修改原对象，返回深拷贝。
 */
export function sanitizeForExport(doc: WorkflowDocument): WorkflowDocument {
  const cloned: WorkflowDocument = JSON.parse(JSON.stringify(doc));
  for (const node of cloned.spec.nodes) {
    node.config = sanitizeConfig(node.config);
  }
  return cloned;
}

function sanitizeConfig(config: Record<string, any>): Record<string, any> {
  const out: Record<string, any> = {};
  for (const [key, value] of Object.entries(config)) {
    if (SENSITIVE_KEYS.has(key.toLowerCase())) {
      if (typeof value === 'string' && value.length > 0 && !isSecretRef(value)) {
        out[key] = inferSecretRef(key);
        continue;
      }
    }
    if (value && typeof value === 'object' && !Array.isArray(value)) {
      out[key] = sanitizeConfig(value);
    } else {
      out[key] = value;
    }
  }
  return out;
}

function isSecretRef(value: string): boolean {
  return /^secrets\.[a-zA-Z0-9_]+$/.test(value.trim());
}

function inferSecretRef(key: string): string {
  const name = key.replace(/[^a-zA-Z0-9_]/g, '_').toLowerCase();
  return `secrets.${name}`;
}

/**
 * 验证 JSON ↔ YAML 往返语义等价。
 * 返回 true 表示 round-trip 无损。
 */
export function verifyRoundTrip(doc: WorkflowDocument): boolean {
  const yamlStr = documentToYaml(doc);
  const parsed = yamlToDocument(yamlStr);
  return JSON.stringify(sortKeys(doc)) === JSON.stringify(sortKeys(parsed));
}

function sortKeys(obj: unknown): unknown {
  if (Array.isArray(obj)) return obj.map(sortKeys);
  if (obj && typeof obj === 'object') {
    const sorted: Record<string, unknown> = {};
    for (const key of Object.keys(obj as Record<string, unknown>).sort()) {
      sorted[key] = sortKeys((obj as Record<string, unknown>)[key]);
    }
    return sorted;
  }
  return obj;
}
