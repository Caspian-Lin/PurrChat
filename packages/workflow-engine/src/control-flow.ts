import type { BlueprintNode } from './types.js';

export interface ControlFlowRoute {
  portId: string;
  session: Record<string, string>;
}

/**
 * Resolves the sole outgoing control-flow port for a completed node.
 * Workflows currently carry one execution token, so Merge is deliberately an
 * exclusive (OR) merge rather than a misleading all-input barrier.
 */
export function resolveControlFlowRoute(
  node: BlueprintNode,
  ports: Record<string, string>,
  session: Record<string, string>,
): ControlFlowRoute | null {
  switch (node.type) {
    case 'if':
    case 'switch':
      return { portId: ports.__branch__ ?? '', session };
    case 'merge':
      return { portId: 'out_exec', session };
    case 'loop':
      return resolveLoopRoute(node, ports, session);
    default:
      return null;
  }
}

function resolveLoopRoute(
  node: BlueprintNode,
  ports: Record<string, string>,
  session: Record<string, string>,
): ControlFlowRoute {
  const key = `loop:${node.id}`;
  const iterations = Number.parseInt(session[key] ?? '0', 10) || 0;
  const maxIterations = Number.parseInt(ports.__loop_max__ ?? '10', 10) || 10;
  const shouldContinue = isTruthy(ports.__loop_condition__ ?? '');

  if (shouldContinue && iterations < maxIterations) {
    return {
      portId: 'out_body',
      session: { ...session, [key]: String(iterations + 1) },
    };
  }

  const nextSession = { ...session };
  delete nextSession[key];
  return { portId: 'out_done', session: nextSession };
}

function isTruthy(value: string): boolean {
  return ['true', '1', 'yes', 'on'].includes(value.trim().toLowerCase());
}
