/**
 * Workflow Document 验证 composable
 *
 * 在保存前调用 workflow-engine 的 validator，返回结构化错误。
 * 前端编辑器据此阻止保存或展示警告。
 */

import { useMemoize } from '@vueuse/core';
import {
  NodeRegistry,
  allNodes,
  validateWorkflowDocument,
  type ValidationResult,
} from '@purrchat/workflow-engine';

const registry = new NodeRegistry();
registry.registerAll(allNodes);

export function useWorkflowValidator() {
  function validate(doc: unknown): ValidationResult {
    return validateWorkflowDocument(doc, registry);
  }

  const memoizedValidate = useMemoize(validate, {
    getKey: (doc) => JSON.stringify(doc),
  });

  return {
    validate,
    validateCached: memoizedValidate,
    registry,
  };
}

export type { ValidationResult };
