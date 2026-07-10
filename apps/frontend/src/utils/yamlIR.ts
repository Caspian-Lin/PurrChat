/**
 * YAML IR — 节点图的 YAML 中间表示层
 *
 * 提供人类可读的 YAML 格式作为 JSON (events + connections) 的双向桥梁。
 * Claude Code 和技术用户可以直接编辑 YAML，非技术用户通过拖拽节点编辑。
 *
 * YAML 格式示例：
 * ```yaml
 * version: 1
 * nodes:
 *   - name: 用户输入
 *     type: trigger
 *   - name: AI 思考
 *     type: llm
 *     config:
 *       model: gpt-4
 *       system_prompt: "你是助手"
 *   - name: 回复
 *     type: reply
 *     config:
 *       content: "{AI 思考.output}"
 * connections:
 *   - [用户输入, trigger, AI 思考, in_exec]
 *   - [AI 思考, out_output, 回复, in_content]
 * ```
 */

import yaml from 'js-yaml';
import type { WorkflowEvent, FlowConnection } from '../models/types';
import { getDefaultPorts } from './portTypes';
import {
  documentToYaml,
  sanitizeForExport,
  type WorkflowDocument,
} from '@purrchat/workflow-engine';
import { WORKFLOW_API_VERSION, WORKFLOW_KIND } from '@purrchat/workflow-types';

// ──────────────────────────────────────────────────────────
// Types
// ──────────────────────────────────────────────────────────

/** YAML 节点定义 */
interface YamlNode {
  name: string;
  type: string;
  id?: string; // 可选，导入时自动生成
  config?: Record<string, any>;
}

/** YAML 连接定义：[源节点名, 源端口, 目标节点名, 目标端口] */
type YamlConnection = [string, string, string, string];

/** YAML IR 根结构 */
interface YamlFlow {
  version: number;
  nodes: YamlNode[];
  connections: YamlConnection[];
}

// ──────────────────────────────────────────────────────────
// Export: JSON → YAML
// ──────────────────────────────────────────────────────────

/**
 * 将 events + connections 转换为人类可读的 YAML 字符串
 */
export function flowToYaml(events: WorkflowEvent[], connections: FlowConnection[] = []): string {
  // 构建 id → name 映射
  const nameMap = buildNameMap(events);

  const yamlNodes: YamlNode[] = events.map((evt) => {
    const node: YamlNode = {
      name: evt.name,
      type: evt.type,
    };
    // 保留 config（仅非空字段）
    if (evt.config && Object.keys(evt.config).length > 0) {
      node.config = cleanConfig(evt.config);
    }
    return node;
  });

  const yamlConnections: YamlConnection[] = connections.map((conn) => {
    const srcName = nameMap.get(conn.sourceNodeId) || conn.sourceNodeId;
    const tgtName = nameMap.get(conn.targetNodeId) || conn.targetNodeId;
    return [srcName, conn.sourcePortId, tgtName, conn.targetPortId];
  });

  const yamlData: YamlFlow = {
    version: 1,
    nodes: yamlNodes,
    connections: yamlConnections,
  };

  const header = [
    '# PurrChat Agent Flow — YAML IR v1',
    '# 此文件描述了 Bot 工作流的事件链和连接关系',
    '# 可由 Claude Code 或用户手动编辑，导入后自动转换为可视化节点图',
    '#',
    '# connections 格式: [源节点, 源端口, 目标节点, 目标端口]',
    '',
  ].join('\n');

  return header + yaml.dump(yamlData, { indent: 2, lineWidth: 120, noRefs: true });
}

// ──────────────────────────────────────────────────────────
// Import: YAML → JSON
// ──────────────────────────────────────────────────────────

export interface YamlImportResult {
  events: WorkflowEvent[];
  connections: FlowConnection[];
  errors: string[];
}

/**
 * 将 YAML 字符串解析为 events + connections
 * 自动生成 ID、补全 ports
 */
export function yamlToFlow(yamlStr: string): YamlImportResult {
  const errors: string[] = [];

  const parsed = yaml.load(yamlStr) as YamlFlow | null;
  if (!parsed || !Array.isArray(parsed.nodes)) {
    return { events: [], connections: [], errors: ['无效的 YAML 格式：缺少 nodes 数组'] };
  }

  // 第一遍：建立 name → id 映射
  const nameToId = new Map<string, string>();
  const events: WorkflowEvent[] = [];

  for (const yamlNode of parsed.nodes) {
    if (!yamlNode.name || !yamlNode.type) {
      errors.push(`跳过无效节点：缺少 name 或 type`);
      continue;
    }

    // 检查重名
    if (nameToId.has(yamlNode.name)) {
      errors.push(`节点名称 "${yamlNode.name}" 重复，后者将覆盖前者`);
    }

    const id = yamlNode.id || generateNodeId(yamlNode.type);
    nameToId.set(yamlNode.name, id);

    // 获取默认端口
    const defaultPorts = getDefaultPorts(yamlNode.type as any);

    events.push({
      id,
      type: yamlNode.type as any,
      name: yamlNode.name,
      config: yamlNode.config || {},
      ports: defaultPorts,
    });
  }

  // 第二遍：解析 connections
  const connections: FlowConnection[] = [];
  const yamlConns = parsed.connections || [];

  for (const yc of yamlConns) {
    if (!Array.isArray(yc) || yc.length < 4) {
      errors.push(`跳过无效连接：${JSON.stringify(yc)}`);
      continue;
    }

    const [srcName, srcPort, tgtName, tgtPort] = yc;
    const srcId = nameToId.get(srcName);
    const tgtId = nameToId.get(tgtName);

    if (!srcId) {
      errors.push(`连接引用了不存在的源节点：${srcName}`);
      continue;
    }
    if (!tgtId) {
      errors.push(`连接引用了不存在的目标节点：${tgtName}`);
      continue;
    }

    connections.push({
      id: `conn_${srcId}_${srcPort}_${tgtId}_${tgtPort}`,
      sourceNodeId: srcId,
      sourcePortId: srcPort as string,
      targetNodeId: tgtId,
      targetPortId: tgtPort as string,
    });
  }

  return { events, connections, errors };
}

// ──────────────────────────────────────────────────────────
// WorkflowDocument 格式（apiVersion/kind/metadata/spec）
// ──────────────────────────────────────────────────────────

/**
 * 将编辑器的 events + connections + trigger + endConditions 组装为 WorkflowDocument
 */
export function flowToDocument(
  events: WorkflowEvent[],
  connections: FlowConnection[] = [],
  trigger: { type: string; rules?: any[] } = { type: 'rule', rules: [] },
  endConditions: any[] = [],
  botName = 'untitled',
  revision = 0
): WorkflowDocument {
  return {
    apiVersion: WORKFLOW_API_VERSION,
    kind: WORKFLOW_KIND,
    metadata: { name: botName, revision },
    spec: {
      trigger: trigger as any,
      nodes: events.map((e) => ({
        id: e.id,
        type: e.type as any,
        name: e.name,
        config: e.config ?? {},
        ports: e.ports,
        position: e.position,
      })),
      connections,
      endConditions,
    },
  };
}

/**
 * 将 WorkflowDocument 拆解为编辑器内部格式
 */
export function documentToFlow(doc: WorkflowDocument): {
  events: WorkflowEvent[];
  connections: FlowConnection[];
  trigger: any;
  endConditions: any[];
} {
  return {
    events: (doc.spec.nodes || []).map((n) => ({
      id: n.id,
      type: n.type as any,
      name: n.name,
      config: n.config ?? {},
      ports: n.ports,
      position: n.position,
    })),
    connections: doc.spec.connections || [],
    trigger: doc.spec.trigger,
    endConditions: doc.spec.endConditions || [],
  };
}

/**
 * 以 WorkflowDocument 格式导出 YAML（含 endConditions，明文密钥自动替换为引用）
 */
export function flowToDocumentYaml(
  events: WorkflowEvent[],
  connections: FlowConnection[] = [],
  trigger: { type: string; rules?: any[] } = { type: 'rule', rules: [] },
  endConditions: any[] = [],
  botName = 'untitled'
): string {
  const doc = flowToDocument(events, connections, trigger, endConditions, botName);
  const safe = sanitizeForExport(doc);
  return documentToYaml(safe);
}

/**
 * 从 YAML 解析 WorkflowDocument（支持新旧两种格式）
 */
export function yamlToFlowDocument(yamlStr: string): YamlImportResult & {
  document?: WorkflowDocument;
  endConditions?: any[];
  trigger?: any;
} {
  const parsed = yaml.load(yamlStr);

  if (parsed && typeof parsed === 'object' && (parsed as any).apiVersion === WORKFLOW_API_VERSION) {
    const doc = parsed as WorkflowDocument;
    const { events, connections, trigger, endConditions } = documentToFlow(doc);
    return {
      events,
      connections,
      errors: [],
      document: doc,
      trigger,
      endConditions,
    };
  }

  return yamlToFlow(yamlStr);
}

// ──────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────

/** 生成节点 ID */
function generateNodeId(type: string): string {
  return `evt_${type}_${Math.random().toString(36).slice(2, 8)}`;
}

/** 构建 id → name 映射 */
function buildNameMap(events: WorkflowEvent[]): Map<string, string> {
  const map = new Map<string, string>();
  for (const evt of events) {
    map.set(evt.id, evt.name);
  }
  return map;
}

/** 清理 config：移除空值和内部字段 */
function cleanConfig(config: Record<string, any>): Record<string, any> {
  const cleaned: Record<string, any> = {};
  for (const [key, value] of Object.entries(config)) {
    if (value !== undefined && value !== null && value !== '') {
      cleaned[key] = value;
    }
  }
  return cleaned;
}
