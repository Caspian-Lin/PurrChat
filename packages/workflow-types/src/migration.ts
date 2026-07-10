/**
 * 旧 mechanism_config → WorkflowDocument 迁移
 *
 * mechanism_config 是嵌套结构：mechanisms[].reply.workflow.events/connections/end_conditions
 * WorkflowDocument 将 trigger、nodes、connections、endConditions 提升为顶层 spec 字段。
 *
 * 迁移规则：
 * 1. 取第一个 enabled mechanism（或第一个）作为 trigger + workflow 来源
 * 2. reply.type !== 'workflow' / 'special_mode' 的机制不迁移（predefined/llm 保持旧逻辑）
 * 3. migration 保持 revision=0（未发布草稿）
 */

import type { MechanismConfig, WorkflowSpec } from './workflow.js';
import type { WorkflowDocument, WorkflowDocumentNode } from './document.js';
import { createEmptyDocument } from './document.js';

export function migrateMechanismToDocument(
  raw: unknown,
  botName: string,
): WorkflowDocument {
  const doc = createEmptyDocument(botName);

  if (!raw || typeof raw !== 'object') return doc;

  const config = raw as MechanismConfig;
  if (!Array.isArray(config.mechanisms) || config.mechanisms.length === 0) return doc;

  const mech = config.mechanisms.find((m) => m.enabled) ?? config.mechanisms[0];
  if (!mech) return doc;

  doc.spec.trigger = mech.trigger;

  const wf: WorkflowSpec | undefined =
    mech.reply?.workflow ?? mech.reply?.special_mode;
  if (wf) {
    doc.spec.nodes = (wf.events ?? []).map((e): WorkflowDocumentNode => ({
      id: e.id,
      type: e.type,
      name: e.name,
      config: e.config ?? {},
      ports: e.ports,
      position: e.position,
    }));
    doc.spec.connections = wf.connections ?? [];
    doc.spec.endConditions = wf.end_conditions ?? [];
  }

  return doc;
}

export function isWorkflowDocument(raw: unknown): raw is WorkflowDocument {
  return (
    typeof raw === 'object' &&
    raw !== null &&
    (raw as any).apiVersion === 'purrchat.ai/v1alpha1' &&
    (raw as any).kind === 'BotWorkflow'
  );
}
