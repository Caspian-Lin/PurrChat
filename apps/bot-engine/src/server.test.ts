import { describe, expect, it } from 'vitest';
import { createServer } from './server.js';

const document = {
  apiVersion: 'purrchat.ai/v1alpha1' as const,
  kind: 'BotWorkflow' as const,
  metadata: { name: 'bot-engine-contract', revision: 1 },
  spec: {
    trigger: { type: 'rule' as const, rules: [] },
    nodes: [
      { id: 'trigger', type: 'trigger' as const, name: 'Trigger', config: {} },
      {
        id: 'reply',
        type: 'reply' as const,
        name: 'Reply',
        config: { template: 'contract: ${input.text}' },
      },
      { id: 'end', type: 'end' as const, name: 'End', config: {} },
    ],
    connections: [
      {
        id: 'c1',
        sourceNodeId: 'trigger',
        sourcePortId: 'out_exec',
        targetNodeId: 'reply',
        targetPortId: 'in_exec',
      },
      {
        id: 'c2',
        sourceNodeId: 'reply',
        sourcePortId: 'out_exec',
        targetNodeId: 'end',
        targetPortId: 'in_exec',
      },
    ],
    endConditions: [{ type: 'max_rounds' as const, value: 5 }],
  },
};

describe('bot-engine HTTP contract', () => {
  it('reports health', async () => {
    const response = await createServer().request('/health');

    expect(response.status).toBe(200);
    await expect(response.json()).resolves.toMatchObject({ status: 'ok', version: '0.1.0' });
  });

  it('executes a published workflow document through the production route', async () => {
    const response = await createServer().request('/execute', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        conversation_id: 'conversation-1',
        bot_id: 'bot-1',
        bot_name: 'Contract Bot',
        sender_id: 'user-1',
        sender_name: 'Tester',
        content: 'hello',
        msg_type: 'text',
        document,
        revision: 1,
        granted_capabilities: ['messages:read_trigger', 'messages:send'],
      }),
    });

    expect(response.status).toBe(200);
    await expect(response.json()).resolves.toMatchObject({
      reply: 'contract: hello',
      triggered: true,
      session_active: false,
      status: 'completed',
      revision: 1,
    });
  });
});
