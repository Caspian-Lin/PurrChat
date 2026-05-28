import type { NodeDefinition } from '../types.js';
import { triggerNode } from './trigger.js';
import { endNode } from './end.js';
import { replyNode } from './reply.js';
import { llmNode } from './llm.js';
import { ifNode } from './if.js';
import { builtinNode } from './builtin.js';
import { waitNode } from './wait.js';
import { loopNode } from './loop.js';
import { switchNode } from './switch.js';
import { mergeNode } from './merge.js';
import { historyNode } from './history.js';
import { toolNode } from './tool.js';
import { difyNode } from './dify.js';
import { n8nNode } from './n8n.js';
import { templateNode } from './template.js';

export const allNodes: NodeDefinition[] = [
  triggerNode,
  endNode,
  replyNode,
  llmNode,
  ifNode,
  builtinNode,
  waitNode,
  loopNode,
  switchNode,
  mergeNode,
  historyNode,
  toolNode,
  difyNode,
  n8nNode,
  templateNode,
];
