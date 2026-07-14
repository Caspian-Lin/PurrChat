/**
 * Workflow Document Validator
 *
 * 在编辑器保存前、API 写入前和运行时编译前共享同一套校验逻辑。
 * 返回结构化 issues，前端据此阻止保存或展示警告。
 *
 * 校验范围：
 * 1. 文档结构（apiVersion / kind / metadata / spec）
 * 2. 节点类型存在性、trigger 唯一性、configSchema（zod）
 * 3. 连线节点/端口存在性、方向、类型兼容性
 * 4. 有向图环路检测（DFS）
 * 5. secret 明文检测（敏感字段不得含明文密钥）
 * 6. capability 推导
 */

import type {
  WorkflowDocument,
  WorkflowDocumentNode,
  EventPort,
  FlowConnection,
} from '@purrchat/workflow-types';
import {
  NODE_MANIFEST,
  getDefaultPorts,
  getPortsForConfig,
  isPortCompatible,
  extractVariablePaths,
  parseNodeOutputPath,
  parseSecretName,
} from '@purrchat/workflow-types';
import type { NodeRegistry } from './registry.js';
import { deriveCapabilities } from './capabilities.js';
import { extractSecretRefs } from './secrets.js';
import type { Blueprint } from './types.js';

export type ValidationLevel = 'error' | 'warning';

export interface ValidationIssue {
  level: ValidationLevel;
  code: string;
  message: string;
  path?: string;
  nodeId?: string;
  connectionId?: string;
}

export interface ValidationResult {
  valid: boolean;
  issues: ValidationIssue[];
  derivedCapabilities: string[];
}

const SENSITIVE_CONFIG_KEYS = new Set([
  'api_key',
  'apikey',
  'secret',
  'token',
  'password',
  'webhook_secret',
  'authorization',
]);

const PRODUCTION_NODE_TYPES = new Set(
  NODE_MANIFEST.filter(
    (node) => node.implemented && node.tested && node.productionReady,
  ).map((node) => node.type),
);

export function validateWorkflowDocument(
  doc: unknown,
  registry: NodeRegistry,
): ValidationResult {
  const issues: ValidationIssue[] = [];

  validateStructure(doc, issues);

  if (issues.length > 0) {
    return { valid: false, issues, derivedCapabilities: [] };
  }

  const document = doc as WorkflowDocument;
  const nodeMap = new Map<string, WorkflowDocumentNode>();
  for (const n of document.spec.nodes) nodeMap.set(n.id, n);

  validateNodes(document, registry, issues);
  validateConnections(document, nodeMap, issues);
  validateControlFlow(document, nodeMap, issues);
  validateVariableRefs(document, nodeMap, issues);
  detectCycles(document, issues);
  validateSecrets(document, issues);

  const derived = deriveCapabilities(toBlueprint(document));

  return {
    valid: issues.filter((i) => i.level === 'error').length === 0,
    issues,
    derivedCapabilities: derived,
  };
}

// ─── 结构校验 ──────────────────────────────────────────────────

function validateStructure(doc: unknown, issues: ValidationIssue[]): void {
  if (typeof doc !== 'object' || doc === null) {
    issues.push(err('invalid_type', '文档必须是对象'));
    return;
  }

  const d = doc as Record<string, any>;

  if (d.apiVersion !== 'purrchat.ai/v1alpha1') {
    issues.push(
      err('unknown_api_version', `apiVersion 必须是 purrchat.ai/v1alpha1，实际: ${d.apiVersion ?? '(缺失)'}`, 'apiVersion'),
    );
  }

  if (d.kind !== 'BotWorkflow') {
    issues.push(
      err('unknown_kind', `kind 必须是 BotWorkflow，实际: ${d.kind ?? '(缺失)'}`, 'kind'),
    );
  }

  if (!d.metadata || typeof d.metadata.name !== 'string') {
    issues.push(err('missing_metadata', 'metadata.name 必须存在', 'metadata.name'));
  }

  if (typeof d.metadata?.revision !== 'number') {
    issues.push(err('missing_revision', 'metadata.revision 必须是数字', 'metadata.revision'));
  }

  if (!d.spec || typeof d.spec !== 'object') {
    issues.push(err('missing_spec', 'spec 字段缺失', 'spec'));
    return;
  }

  if (!Array.isArray(d.spec.nodes)) {
    issues.push(err('invalid_nodes', 'spec.nodes 必须是数组', 'spec.nodes'));
  }
  if (!Array.isArray(d.spec.connections)) {
    issues.push(err('invalid_connections', 'spec.connections 必须是数组', 'spec.connections'));
  }
  if (!Array.isArray(d.spec.endConditions)) {
    issues.push(err('invalid_end_conditions', 'spec.endConditions 必须是数组', 'spec.endConditions'));
  }
  if (!d.spec.trigger || typeof d.spec.trigger !== 'object') {
    issues.push(err('invalid_trigger', 'spec.trigger 缺失', 'spec.trigger'));
  }
}

// ─── 节点校验 ──────────────────────────────────────────────────

function validateNodes(
  doc: WorkflowDocument,
  registry: NodeRegistry,
  issues: ValidationIssue[],
): void {
  const nodes = doc.spec.nodes;
  let triggerCount = 0;
  const seenIds = new Set<string>();
  const seenNames = new Set<string>();

  for (let i = 0; i < nodes.length; i++) {
    const node = nodes[i];
    const base = `spec.nodes[${i}]`;

    if (seenIds.has(node.id)) {
      issues.push(err('duplicate_node_id', `节点 ID 重复: ${node.id}`, `${base}.id`, node.id));
    }
    seenIds.add(node.id);

    if (node.name && seenNames.has(node.name)) {
      issues.push(warn('duplicate_node_name', `节点名称重复: ${node.name}`, `${base}.name`, node.id));
    }
    if (node.name) seenNames.add(node.name);

    if (!registry.has(node.type)) {
      issues.push(
        err('unknown_node_type', `未知节点类型: ${node.type}`, `${base}.type`, node.id),
      );
      continue;
    }

    if (!PRODUCTION_NODE_TYPES.has(node.type)) {
      issues.push(
        err(
          'node_not_production_ready',
          `节点类型尚未通过生产验证: ${node.type}`,
          `${base}.type`,
          node.id,
        ),
      );
      continue;
    }

    if (node.type === 'trigger') triggerCount++;

    const def = registry.get(node.type)!;
    const result = def.configSchema.safeParse(node.config);
    if (!result.success) {
      for (const zErr of result.error.issues) {
        const pathStr = zErr.path.length > 0 ? zErr.path.join('.') : '(root)';
        issues.push(
          err(
            'invalid_config',
            `节点 ${node.name || node.id} 配置字段 ${pathStr}: ${zErr.message}`,
            `${base}.config.${pathStr}`,
            node.id,
          ),
        );
      }
    }
  }

  if (triggerCount === 0) {
    issues.push(err('no_trigger', '工作流必须包含恰好一个 trigger 节点'));
  } else if (triggerCount > 1) {
    issues.push(err('multiple_trigger', `工作流只能有一个 trigger 节点，实际: ${triggerCount}`));
  }
}

// ─── 连线校验 ──────────────────────────────────────────────────

function validateConnections(
  doc: WorkflowDocument,
  nodeMap: Map<string, WorkflowDocumentNode>,
  issues: ValidationIssue[],
): void {
  const conns = doc.spec.connections;

  for (let i = 0; i < conns.length; i++) {
    const conn = conns[i];
    const base = `spec.connections[${i}]`;

    const srcNode = nodeMap.get(conn.sourceNodeId);
    const tgtNode = nodeMap.get(conn.targetNodeId);

    if (!srcNode) {
      issues.push(
        err('dangling_connection_source', `连线引用了不存在的源节点: ${conn.sourceNodeId}`, base, undefined, conn.id),
      );
      continue;
    }
    if (!tgtNode) {
      issues.push(
        err('dangling_connection_target', `连线引用了不存在的目标节点: ${conn.targetNodeId}`, base, undefined, conn.id),
      );
      continue;
    }

    const srcPorts = getEffectivePorts(srcNode);
    const tgtPorts = getEffectivePorts(tgtNode);

    const srcPort = srcPorts.find((p) => p.id === conn.sourcePortId && p.direction === 'output');
    const tgtPort = tgtPorts.find((p) => p.id === conn.targetPortId && p.direction === 'input');

    if (!srcPort) {
      issues.push(
        err('invalid_source_port', `源端口不存在: ${conn.sourcePortId}`, `${base}.sourcePortId`, srcNode.id, conn.id),
      );
    }
    if (!tgtPort) {
      issues.push(
        err('invalid_target_port', `目标端口不存在: ${conn.targetPortId}`, `${base}.targetPortId`, tgtNode.id, conn.id),
      );
    }

    if (srcPort && tgtPort && !isPortCompatible(srcPort.dataType, tgtPort.dataType)) {
      issues.push(
        warn(
          'port_type_mismatch',
          `端口类型不兼容: ${srcPort.dataType} → ${tgtPort.dataType}`,
          base,
          srcNode.id,
          conn.id,
        ),
      );
    }
  }
}

function getEffectivePorts(node: WorkflowDocumentNode): EventPort[] {
  if (node.type === 'if' || node.type === 'switch' || node.type === 'merge') {
    return getPortsForConfig(node.type, node.config);
  }
  if (node.ports && node.ports.length > 0) return node.ports;
  return getDefaultPorts(node.type);
}

function validateControlFlow(
  doc: WorkflowDocument,
  nodeMap: Map<string, WorkflowDocumentNode>,
  issues: ValidationIssue[],
): void {
  for (const node of doc.spec.nodes) {
    if (node.type === 'switch') {
      const cases = Array.isArray(node.config.cases) ? node.config.cases : [];
      const values = new Set<string>();
      for (const [index, item] of cases.entries()) {
        const value = typeof item === 'object' && item ? (item as { value?: unknown }).value : undefined;
        if (typeof value !== 'string' || !value.trim()) {
          issues.push(err('switch_case_value_required', `Switch 分支 ${index + 1} 必须设置匹配值`, undefined, node.id));
        } else if (values.has(value)) {
          issues.push(err('switch_case_value_duplicate', `Switch 分支匹配值重复: ${value}`, undefined, node.id));
        } else {
          values.add(value);
        }
      }
      if (!hasOutputConnection(doc, node.id, 'out_default')) {
        issues.push(err('switch_default_missing', 'Switch 必须连接默认分支', undefined, node.id));
      }
    }

    if (node.type === 'merge') {
      for (const port of getEffectivePorts(node).filter((port) => port.direction === 'input')) {
        const arrivals = doc.spec.connections.filter(
          (connection) => connection.targetNodeId === node.id && connection.targetPortId === port.id,
        );
        if (arrivals.length !== 1) {
          issues.push(err('merge_input_invalid', `Merge 输入 ${port.name} 必须恰好连接一次`, undefined, node.id));
        }
      }
    }

    if (node.type === 'loop') {
      const bodyConnections = doc.spec.connections.filter(
        (connection) => connection.sourceNodeId === node.id && connection.sourcePortId === 'out_body',
      );
      if (bodyConnections.length !== 1) {
        issues.push(err('loop_body_invalid', 'Loop 循环体必须恰好连接一次', undefined, node.id));
      }
    }
  }
}

function hasOutputConnection(doc: WorkflowDocument, nodeId: string, portId: string): boolean {
  return doc.spec.connections.some(
    (connection) => connection.sourceNodeId === nodeId && connection.sourcePortId === portId,
  );
}

// ─── 环路检测（DFS） ───────────────────────────────────────────

function detectCycles(doc: WorkflowDocument, issues: ValidationIssue[]): void {
  const adj = new Map<string, string[]>();
  for (const node of doc.spec.nodes) adj.set(node.id, []);
  for (const conn of doc.spec.connections) {
    if (isLegalLoopBackEdge(doc, conn)) continue;
    const neighbors = adj.get(conn.sourceNodeId);
    if (neighbors) neighbors.push(conn.targetNodeId);
  }

  const WHITE = 0, GRAY = 1, BLACK = 2;
  const color = new Map<string, number>();
  for (const node of doc.spec.nodes) color.set(node.id, WHITE);

  for (const node of doc.spec.nodes) {
    if (color.get(node.id) === WHITE) {
      const cycleNode = dfsVisit(node.id, adj, color, WHITE, GRAY, BLACK);
      if (cycleNode) {
        issues.push(err('cycle_detected', `检测到环路，涉及节点: ${cycleNode}`, undefined, cycleNode));
      }
    }
  }
}

/** Only a path originating at Loop.out_body may close a Loop.in_exec cycle. */
function isLegalLoopBackEdge(doc: WorkflowDocument, candidate: FlowConnection): boolean {
  const target = doc.spec.nodes.find((node) => node.id === candidate.targetNodeId);
  if (target?.type !== 'loop' || candidate.targetPortId !== 'in_exec' || candidate.sourceNodeId === target.id) {
    return false;
  }

  const bodyTargets = doc.spec.connections
    .filter((connection) => connection.sourceNodeId === target.id && connection.sourcePortId === 'out_body')
    .map((connection) => connection.targetNodeId);
  const visited = new Set<string>();
  const queue = [...bodyTargets];

  while (queue.length > 0) {
    const current = queue.shift()!;
    if (current === candidate.sourceNodeId) return true;
    if (current === target.id || visited.has(current)) continue;
    visited.add(current);
    for (const connection of doc.spec.connections) {
      if (connection.id !== candidate.id && connection.sourceNodeId === current) {
        queue.push(connection.targetNodeId);
      }
    }
  }

  return false;
}

function dfsVisit(
  start: string,
  adj: Map<string, string[]>,
  color: Map<string, number>,
  WHITE: number,
  GRAY: number,
  BLACK: number,
): string | null {
  const stack: Array<{ node: string; idx: number }> = [{ node: start, idx: 0 }];
  color.set(start, GRAY);

  while (stack.length > 0) {
    const top = stack[stack.length - 1];
    const neighbors = adj.get(top.node) ?? [];

    if (top.idx < neighbors.length) {
      const next = neighbors[top.idx];
      top.idx++;
      const c = color.get(next);
      if (c === GRAY) return next;
      if (c === WHITE) {
        color.set(next, GRAY);
        stack.push({ node: next, idx: 0 });
      }
    } else {
      color.set(top.node, BLACK);
      stack.pop();
    }
  }
  return null;
}

// ─── Secret 明文检测 ───────────────────────────────────────────

function validateSecrets(doc: WorkflowDocument, issues: ValidationIssue[]): void {
  for (const node of doc.spec.nodes) {
    checkPlaintextSecrets(node.config, node.id, node.name, '', issues);
  }
}

function checkPlaintextSecrets(
  obj: Record<string, any>,
  nodeId: string,
  nodeName: string,
  path: string,
  issues: ValidationIssue[],
): void {
  for (const [key, value] of Object.entries(obj)) {
    const currentPath = path ? `${path}.${key}` : key;

    if (SENSITIVE_CONFIG_KEYS.has(key.toLowerCase())) {
      if (typeof value === 'string' && value.length > 0 && !isSecretRef(value)) {
        issues.push(
          warn(
            'plaintext_secret',
            `节点 ${nodeName || nodeId} 的字段 ${currentPath} 包含明文密钥，建议使用 secrets.<name> 引用`,
            `config.${currentPath}`,
            nodeId,
          ),
        );
      }
    }

    if (value && typeof value === 'object' && !Array.isArray(value)) {
      checkPlaintextSecrets(value, nodeId, nodeName, currentPath, issues);
    }
  }
}

function isSecretRef(value: string): boolean {
  return /^secrets\.[a-zA-Z0-9_]+$/.test(value.trim());
}

// ─── 变量引用校验 ──────────────────────────────────────────────

/**
 * 扫描节点 config 中的 ${path} 变量引用，检查：
 * - nodes.<key>.outputs.<port>: 节点 key 和端口是否存在
 * - secrets.<name>: 是否需要 secrets:use capability（已在 validateSecrets 中检查明文，
 *   这里仅对 ${secrets.<name>} 规范格式做额外 warning）
 */
function validateVariableRefs(
  doc: WorkflowDocument,
  nodeMap: Map<string, WorkflowDocumentNode>,
  issues: ValidationIssue[],
): void {
  // 构建 key → node 映射
  const keyMap = new Map<string, WorkflowDocumentNode>();
  for (const n of doc.spec.nodes) {
    if (n.key) keyMap.set(n.key, n);
  }

  for (const node of doc.spec.nodes) {
    scanConfigForVarRefs(node.config, node, keyMap, issues);
  }
}

function scanConfigForVarRefs(
  obj: Record<string, any>,
  node: WorkflowDocumentNode,
  keyMap: Map<string, WorkflowDocumentNode>,
  issues: ValidationIssue[],
): void {
  for (const [, value] of Object.entries(obj)) {
    if (typeof value === 'string') {
      const paths = extractVariablePaths(value);
      for (const path of paths) {
        validateSingleVarRef(path, node, keyMap, issues);
      }
    } else if (value && typeof value === 'object' && !Array.isArray(value)) {
      scanConfigForVarRefs(value, node, keyMap, issues);
    } else if (Array.isArray(value)) {
      for (const item of value) {
        if (typeof item === 'string') {
          const paths = extractVariablePaths(item);
          for (const path of paths) {
            validateSingleVarRef(path, node, keyMap, issues);
          }
        } else if (item && typeof item === 'object') {
          scanConfigForVarRefs(item, node, keyMap, issues);
        }
      }
    }
  }
}

function validateSingleVarRef(
  path: string,
  node: WorkflowDocumentNode,
  keyMap: Map<string, WorkflowDocumentNode>,
  issues: ValidationIssue[],
): void {
  // nodes.<key>.outputs.<port>
  const nodeRef = parseNodeOutputPath(path);
  if (nodeRef) {
    const targetNode = keyMap.get(nodeRef.nodeKey);
    if (!targetNode) {
      issues.push(
        warn(
          'var_ref_unknown_node',
          `变量 ${path} 引用了不存在的节点 key: ${nodeRef.nodeKey}`,
          `config`,
          node.id,
        ),
      );
      return;
    }
    const ports = getEffectivePorts(targetNode);
    const portExists = ports.some(
      (p) => p.id === nodeRef.portId && p.direction === 'output',
    );
    if (!portExists) {
      issues.push(
        warn(
          'var_ref_unknown_port',
          `变量 ${path} 引用了节点 ${targetNode.name} 不存在的输出端口: ${nodeRef.portId}`,
          `config`,
          node.id,
        ),
      );
    }
    return;
  }

  // secrets.<name> — 规范格式引用
  const secretName = parseSecretName(path);
  if (secretName) {
    // secrets 引用通过现有 validateSecrets + checkSecretCapability 处理
    // 这里不重复检查
    return;
  }

  // input.* / sender.* / conversation.* / history.* — 内置变量，始终可用
  if (
    path.startsWith('input.') ||
    path.startsWith('sender.') ||
    path.startsWith('conversation.') ||
    path.startsWith('history.')
  ) {
    return;
  }

  // session.* — 运行时变量，静态无法校验
  if (path.startsWith('session.')) {
    return;
  }
}

// ─── 辅助 ──────────────────────────────────────────────────────

export function toBlueprint(doc: WorkflowDocument): Blueprint {
  return {
    nodes: doc.spec.nodes.map((n) => ({
      id: n.id,
      type: n.type,
      name: n.name,
      key: n.key,
      config: n.config,
      ports: n.ports,
      position: n.position,
    })),
    connections: doc.spec.connections as any[],
    endConditions: doc.spec.endConditions as any[],
  };
}

function err(code: string, message: string, path?: string, nodeId?: string, connectionId?: string): ValidationIssue {
  return { level: 'error', code, message, path, nodeId, connectionId };
}

function warn(code: string, message: string, path?: string, nodeId?: string, connectionId?: string): ValidationIssue {
  return { level: 'warning', code, message, path, nodeId, connectionId };
}
