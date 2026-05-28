import type { NodeDefinition } from './types.js';

export class NodeRegistry {
  private nodes = new Map<string, NodeDefinition>();

  register(def: NodeDefinition): void {
    this.nodes.set(def.type, def);
  }

  registerAll(defs: NodeDefinition[]): void {
    for (const d of defs) this.register(d);
  }

  get(type: string): NodeDefinition | undefined {
    return this.nodes.get(type);
  }

  getAll(): NodeDefinition[] {
    return Array.from(this.nodes.values());
  }

  getByCategory(category: string): NodeDefinition[] {
    return this.getAll().filter((n) => n.category === category);
  }

  has(type: string): boolean {
    return this.nodes.has(type);
  }
}
