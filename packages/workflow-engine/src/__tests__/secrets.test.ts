import { describe, it, expect } from 'vitest';
import { resolveSecrets, extractSecretRefs, checkSecretCapability } from '../secrets.js';
import { deriveCapabilities } from '../capabilities.js';
import type { Blueprint } from '../types.js';

describe('extractSecretRefs', () => {
  it('extracts full secrets.<name> references from string fields', () => {
    const config = {
      api_key: 'secrets.openai_key',
      nested: { token: 'secrets.webhook_token' },
      arr: ['secrets.foo', 'plain'],
    };
    expect(extractSecretRefs(config).sort()).toEqual(['foo', 'openai_key', 'webhook_token']);
  });

  it('ignores substrings that are not full references', () => {
    expect(extractSecretRefs({ x: 'prefix secrets.foo suffix' })).toEqual([]);
    expect(extractSecretRefs({ x: 'secrets.' })).toEqual([]);
    expect(extractSecretRefs({ x: 'normal-value' })).toEqual([]);
  });

  it('handles empty / null values', () => {
    expect(extractSecretRefs({})).toEqual([]);
    expect(extractSecretRefs(null)).toEqual([]);
  });
});

describe('resolveSecrets', () => {
  it('replaces secrets.<name> with actual values', () => {
    const config = { api_key: 'secrets.openai_key', model: 'gpt-4' };
    const resolved = resolveSecrets(config, { openai_key: 'sk-real-key' });
    expect(resolved).toEqual({ api_key: 'sk-real-key', model: 'gpt-4' });
  });

  it('resolves nested references in arrays and objects', () => {
    const config = {
      headers: [{ name: 'Authorization', value: 'secrets.token' }],
      url: 'https://api.example.com',
    };
    const resolved = resolveSecrets(config, { token: 'Bearer abc' });
    expect(resolved).toEqual({
      headers: [{ name: 'Authorization', value: 'Bearer abc' }],
      url: 'https://api.example.com',
    });
  });

  it('missing secret reference becomes empty string', () => {
    const config = { api_key: 'secrets.missing' };
    const resolved = resolveSecrets(config, { other: 'x' });
    expect(resolved).toEqual({ api_key: '' });
  });

  it('returns config unchanged when no secrets provided', () => {
    const config = { api_key: 'secrets.openai_key' };
    const resolved = resolveSecrets(config, undefined);
    expect(resolved).toBe(config);
  });
});

describe('checkSecretCapability', () => {
  it('returns secrets:use when config references secrets but not granted', () => {
    const config = { api_key: 'secrets.openai_key' };
    expect(checkSecretCapability(config, ['messages:send'])).toEqual(['secrets:use']);
  });

  it('returns empty when secrets:use is granted', () => {
    const config = { api_key: 'secrets.openai_key' };
    expect(checkSecretCapability(config, ['messages:send', 'secrets:use'])).toEqual([]);
  });

  it('returns empty when config has no secret references', () => {
    const config = { api_key: 'sk-plain' };
    expect(checkSecretCapability(config, ['messages:send'])).toEqual([]);
  });
});

describe('deriveCapabilities with secrets', () => {
  const blueprint: Blueprint = {
    nodes: [
      { id: 't', type: 'trigger', name: 'T', config: {} },
      {
        id: 'llm',
        type: 'llm',
        name: 'LLM',
        config: { api_key: 'secrets.openai_key', api_url: 'https://api.openai.com', model: 'gpt-4' },
      },
      { id: 'r', type: 'reply', name: 'R', config: { template: '' } },
      { id: 'e', type: 'end', name: 'E', config: {} },
    ],
    connections: [],
    endConditions: [],
  };

  it('derives secrets:use when a node config references secrets', () => {
    const caps = deriveCapabilities(blueprint);
    expect(caps).toContain('secrets:use');
    expect(caps).toContain('network:external');
  });
});
