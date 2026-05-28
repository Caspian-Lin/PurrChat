/**
 * 端口类型系统 — Bot Studio 事件链编辑器
 *
 * 从 @purrchat/workflow-types 重导出，保持向后兼容。
 * 所有类型定义已迁移到 packages/workflow-types/src/ports.ts。
 */

// 从共享包重导出所有内容
export type {
  PortDataType,
  EventType,
  EventPort,
  FlowConnection,
  NodeTypeMeta,
} from '@purrchat/workflow-types';
export {
  PORT_COLORS,
  NODE_TYPE_META,
  isPortCompatible,
  getDefaultPorts,
  canConnect,
  getPortById,
} from '@purrchat/workflow-types';
