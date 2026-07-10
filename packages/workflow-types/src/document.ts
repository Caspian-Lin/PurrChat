/**
 * 版本化 Workflow Document — Bot 工作流的唯一规范格式
 *
 * 设计见 docs/bot-engine/BOT_SYSTEM_AUDIT_2026-07-09.md §三
 *
 * apiVersion/kind 保证文档可演进；revision 用于 ETag 乐观锁；
 * spec 是编辑器、API 和运行时共享的工作流定义。
 */

import type { EventType, EventPort, FlowConnection } from './ports.js';
import type { TriggerSpec, WorkflowEndCondition } from './workflow.js';

export const WORKFLOW_API_VERSION = 'purrchat.ai/v1alpha1' as const;
export const WORKFLOW_KIND = 'BotWorkflow' as const;

export interface WorkflowDocumentMetadata {
  name: string;
  description?: string;
  revision: number;
  updatedAt?: string;
}

export interface WorkflowDocumentNode {
  id: string;
  type: EventType;
  name: string;
  /** 稳定 key：用于变量引用 ${nodes.<key>.outputs.<port>}，改名不影响 key */
  key?: string;
  config: Record<string, any>;
  ports?: EventPort[];
  position?: { x: number; y: number };
}

export interface WorkflowDocumentSpec {
  trigger: TriggerSpec;
  nodes: WorkflowDocumentNode[];
  connections: FlowConnection[];
  endConditions: WorkflowEndCondition[];
}

export interface WorkflowDocument {
  apiVersion: typeof WORKFLOW_API_VERSION;
  kind: typeof WORKFLOW_KIND;
  metadata: WorkflowDocumentMetadata;
  spec: WorkflowDocumentSpec;
}

export function createEmptyDocument(name: string): WorkflowDocument {
  return {
    apiVersion: WORKFLOW_API_VERSION,
    kind: WORKFLOW_KIND,
    metadata: { name, revision: 0 },
    spec: {
      trigger: { type: 'rule', rules: [] },
      nodes: [],
      connections: [],
      endConditions: [],
    },
  };
}
