import {
  NODE_TYPE_META,
  getDefaultPorts,
  type EventPort,
  type EventType,
  type NodeTypeMeta,
} from './ports';

export interface NodeManifestEntry extends NodeTypeMeta {
  type: EventType;
  ports: EventPort[];
  defaultConfig: Readonly<Record<string, unknown>>;
  implemented: boolean;
  tested: boolean;
  productionReady: boolean;
}

type NodeManifestStatus = Omit<NodeManifestEntry, keyof NodeTypeMeta | 'ports'>;

const NODE_MANIFEST_STATUS = [
  {
    type: 'trigger',
    defaultConfig: {},
    implemented: true,
    tested: true,
    productionReady: true,
  },
  {
    type: 'end',
    defaultConfig: {},
    implemented: true,
    tested: true,
    productionReady: true,
  },
  {
    type: 'wait',
    defaultConfig: {},
    implemented: true,
    tested: true,
    productionReady: true,
  },
  {
    type: 'if',
    defaultConfig: {
      branches: [{ logic: 'and', conditions: [{ left: '', operator: '==', right: '' }] }],
    },
    implemented: true,
    tested: true,
    productionReady: true,
  },
  {
    type: 'loop',
    defaultConfig: { max_iterations: 10, condition: '' },
    implemented: true,
    tested: true,
    productionReady: true,
  },
  {
    type: 'switch',
    defaultConfig: {
      cases: [
        { value: '', label: '分支 1' },
        { value: '', label: '分支 2' },
      ],
    },
    implemented: true,
    tested: true,
    productionReady: true,
  },
  {
    type: 'merge',
    defaultConfig: { input_count: 2 },
    implemented: true,
    tested: true,
    productionReady: true,
  },
  {
    type: 'tool',
    defaultConfig: {
      method: 'GET',
      url: '',
      headers: {},
      timeout: 10000,
    },
    implemented: true,
    tested: false,
    productionReady: false,
  },
  {
    type: 'dify',
    defaultConfig: {
      api_base: '',
      api_key: '',
      app_type: 'workflow',
      response_mode: 'blocking',
      inputs_mapping: '',
    },
    implemented: true,
    tested: false,
    productionReady: false,
  },
  {
    type: 'n8n',
    defaultConfig: {
      webhook_url: '',
      method: 'POST',
      auth_type: 'none',
      auth_header_name: '',
      auth_header_value: '',
      auth_username: '',
      auth_password: '',
    },
    implemented: true,
    tested: false,
    productionReady: false,
  },
  {
    type: 'llm',
    defaultConfig: {
      api_url: '',
      api_key: '',
      model: '',
      system_prompt: '',
      temperature: 0.7,
      max_tokens: 2048,
      context_window: 20,
    },
    implemented: true,
    tested: false,
    productionReady: false,
  },
  {
    type: 'builtin',
    defaultConfig: { builtin_type: 'random_number' },
    implemented: true,
    tested: true,
    productionReady: true,
  },
  {
    type: 'python',
    defaultConfig: { code: '', timeout_ms: 5000 },
    implemented: false,
    tested: false,
    productionReady: false,
  },
  {
    type: 'reply',
    defaultConfig: { template: '' },
    implemented: true,
    tested: true,
    productionReady: true,
  },
  {
    type: 'template',
    defaultConfig: { template: '' },
    implemented: true,
    tested: true,
    productionReady: true,
  },
  {
    type: 'history',
    defaultConfig: { count: 20 },
    implemented: true,
    tested: false,
    productionReady: false,
  },
] as const satisfies readonly NodeManifestStatus[];

export const NODE_MANIFEST: readonly NodeManifestEntry[] = NODE_MANIFEST_STATUS.map((entry) => ({
  ...entry,
  ...NODE_TYPE_META[entry.type],
  ports: getDefaultPorts(entry.type),
}));
