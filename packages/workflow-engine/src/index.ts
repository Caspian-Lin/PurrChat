// @purrchat/workflow-engine
// Bot 工作流引擎 — 基于 XState 的 DAG 执行引擎

export { NodeRegistry } from './registry.js';
export { Compiler } from './compiler.js';
export { WorkflowRuntime } from './runtime.js';
export { resolveInputPorts, replaceVariables, evaluateCondition } from './ports.js';
export type { VariableResolveContext } from './ports.js';
export { resolveTemplate } from './resolver.js';
export type { ResolveContext } from './resolver.js';
export { allNodes } from './nodes/index.js';
export { deriveCapabilities, getMissingCapabilities } from './capabilities.js';
export { resolveSecrets, extractSecretRefs, checkSecretCapability } from './secrets.js';

// Workflow Document Validator
export { validateWorkflowDocument, toBlueprint } from './validator.js';
export type { ValidationResult, ValidationIssue, ValidationLevel } from './validator.js';

// Debug Runner
export { DebugRunner } from './debug-runner.js';

// YAML / JSON 往返
export {
  documentToYaml,
  yamlToDocument,
  documentToJson,
  jsonToDocument,
  sanitizeForExport,
  verifyRoundTrip,
} from './yaml.js';

export type {
  NodeDefinition,
  NodeInput,
  NodeOutput,
  NodeContext,
  Blueprint,
  BlueprintNode,
  BlueprintConnection,
  ExecutionContext,
  ActorInput,
  UserMessageEvent,
  WorkflowEvent,
  ExecuteResult,
  ExecutionStatus,
} from './types.js';
