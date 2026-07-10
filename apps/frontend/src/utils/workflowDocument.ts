import {
  NODE_MANIFEST,
  generateNodeKey,
  type WorkflowDocument,
  type WorkflowEvent,
} from '@purrchat/workflow-types';
import { yamlToDocument, type ValidationResult } from '@purrchat/workflow-engine';

export const PRODUCTION_NODE_MANIFEST = NODE_MANIFEST.filter(
  (node) => node.implemented && node.tested && node.productionReady
);

export function cloneWorkflowDocument(document: WorkflowDocument): WorkflowDocument {
  return structuredClone(document);
}

export function cloneWorkflowEvent(event: WorkflowEvent): WorkflowEvent {
  return JSON.parse(JSON.stringify(event)) as WorkflowEvent;
}

export function serializeWorkflowDocument(document: WorkflowDocument): string {
  return JSON.stringify(document);
}

export function nextUniqueNodeKey(document: WorkflowDocument, type: string): string {
  const keys = new Set(document.spec.nodes.map((node) => node.key).filter(Boolean));
  let index = 1;
  let key = generateNodeKey(type, index);
  while (keys.has(key)) {
    key = generateNodeKey(type, ++index);
  }
  return key;
}

export interface WorkflowGateResult {
  allowed: boolean;
  errors: string[];
  warnings: string[];
}

export function evaluateWorkflowGate(
  result: Pick<ValidationResult, 'issues'>,
  confirmWarnings: (_message: string) => boolean
): WorkflowGateResult {
  const errors = result.issues
    .filter((issue) => issue.level === 'error')
    .map((issue) => issue.message);
  const warnings = result.issues
    .filter((issue) => issue.level === 'warning')
    .map((issue) => issue.message);

  return {
    allowed:
      errors.length === 0 &&
      (warnings.length === 0 ||
        confirmWarnings(`工作流包含以下警告：\n\n${warnings.join('\n')}\n\n仍要继续吗？`)),
    errors,
    warnings,
  };
}

export interface YamlDocumentCandidate {
  candidate?: WorkflowDocument;
  errors: string[];
  warnings: string[];
}

export function parseWorkflowYamlCandidate(
  source: string,
  validate: (_document: unknown) => ValidationResult
): YamlDocumentCandidate {
  try {
    const candidate = yamlToDocument(source);
    const result = validate(candidate);
    const errors = result.issues
      .filter((issue) => issue.level === 'error')
      .map((issue) => issue.message);
    const warnings = result.issues
      .filter((issue) => issue.level === 'warning')
      .map((issue) => issue.message);

    return errors.length > 0 ? { errors, warnings } : { candidate, errors, warnings };
  } catch (error) {
    return {
      errors: [error instanceof Error ? error.message : 'YAML 解析失败'],
      warnings: [],
    };
  }
}
