// 端口类型系统
export type { PortDataType, EventType, EventPort, FlowConnection, NodeTypeMeta } from './ports';
export { PORT_COLORS, NODE_TYPE_META, isPortCompatible, getDefaultPorts, getPortsForConfig, canConnect, getPortById } from './ports';

// 节点发布清单
export { NODE_MANIFEST } from './manifest';
export type { NodeManifestEntry } from './manifest';

// Capability 权限模型
export { Capability, ALL_CAPABILITIES, NODE_CAPABILITIES, CAPABILITY_META, getNodeCapabilities, isSensitiveCapability } from './capabilities';
export type { CapabilityMeta } from './capabilities';

// 工作流核心类型
export type {
  MechanismConfig,
  Mechanism,
  TriggerSpec,
  TriggerRule,
  ReplySpec,
  PredefinedConfig,
  LLMConfig,
  WorkflowSpec,
  WorkflowEvent,
  WorkflowEndCondition,
  LLMEventConfig,
  BuiltinEventConfig,
  PythonEventConfig,
  ReplyEventConfig,
  WorkflowSession,
} from './workflow';

// 调试类型
export type {
  EventTrace,
  DebugContextMessage,
  DebugTraceResult,
  DebugBotRequest,
  DebugStepRequest,
  DebugResetRequest,
} from './debug';

// 版本化 Workflow Document
export {
  WORKFLOW_API_VERSION,
  WORKFLOW_KIND,
  createEmptyDocument,
} from './document';
export type {
  WorkflowDocument,
  WorkflowDocumentMetadata,
  WorkflowDocumentNode,
  WorkflowDocumentSpec,
} from './document';

// 迁移工具
export { migrateMechanismToDocument, isWorkflowDocument } from './migration';

// 统一变量模型
export {
  VARIABLE_SCOPES,
  VARIABLE_REF_RE,
  BUILTIN_VARIABLES,
  extractVariablePaths,
  nodeOutputPath,
  parseNodeOutputPath,
  isSecretPath,
  parseSecretName,
  generateNodeKey,
} from './variables';
export type { VariableScope, BuiltinVariableMeta } from './variables';

// Debug Trace 类型
export type {
  NodeTraceStatus,
  NodeTrace,
  RunTraceStatus,
  RunTrace,
  SideEffectPolicy,
  DebugRunRequest,
  DebugStepRequest as TraceDebugStepRequest,
  DebugCancelRequest,
  DebugResetRequest as TraceDebugResetRequest,
} from './trace';
