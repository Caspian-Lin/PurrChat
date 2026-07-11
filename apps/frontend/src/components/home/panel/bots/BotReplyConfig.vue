<template>
  <div class="space-y-4">
    <!-- 类型选择 -->
    <div class="flex gap-2">
      <button
        v-for="type in types"
        :key="type.value"
        class="px-3 py-1.5 text-xs rounded-[var(--radius-sm,8px)] transition-colors"
        :class="
          localConfig.type === type.value
            ? 'text-white'
            : 'bg-bg-quaternary text-text-secondary hover:bg-hover-bg'
        "
        :style="localConfig.type === type.value ? { background: 'var(--theme-primary)' } : {}"
        @click="handleTypeSwitch(type.value)"
      >
        {{ type.label }}
      </button>
    </div>

    <!-- 预定义回复 -->
    <div v-if="localConfig.type === 'predefined'" class="space-y-4">
      <!-- 子模式选择 -->
      <div class="flex gap-2">
        <button
          v-for="mode in predefinedModes"
          :key="mode.value"
          class="px-3 py-1.5 text-xs rounded-[var(--radius-sm,8px)] transition-colors"
          :class="
            localConfig.predefined?.mode === mode.value
              ? 'text-white'
              : 'bg-bg-quaternary text-text-secondary hover:bg-hover-bg'
          "
          :style="
            localConfig.predefined?.mode === mode.value
              ? { background: 'var(--theme-primary)' }
              : {}
          "
          @click="setPredefinedMode(mode.value)"
        >
          {{ mode.label }}
        </button>
      </div>

      <!-- 模板模式 -->
      <div v-if="localConfig.predefined?.mode === 'template'" class="space-y-3">
        <div>
          <label class="block text-xs text-text-secondary mb-1.5">回复模板</label>
          <textarea
            v-model="localConfig.predefined.template"
            rows="3"
            placeholder="你好，{username}！现在是{time}"
            class="w-full px-3 py-2 text-xs font-mono rounded-[var(--radius-sm,8px)] bg-bg-quaternary text-text-primary placeholder:text-text-quaternary outline-none focus:ring-1 focus:ring-[var(--theme-primary)] resize-none"
            @input="emitUpdate()"
          />
          <div class="text-xs text-text-quaternary mt-1.5 space-y-1">
            <p class="font-medium text-text-tertiary">可用变量</p>
            <div class="grid grid-cols-2 gap-x-4 gap-y-0.5">
              <span><code class="text-text-secondary">{'{' + 'input}'}</code> 用户消息</span>
              <span><code class="text-text-secondary">{'{' + 'username}'}</code> 发送者名称</span>
              <span><code class="text-text-secondary">{'{' + 'time}'}</code> 当前时间</span>
              <span><code class="text-text-secondary">{'{' + 'args}'}</code> 除首个词外的参数</span>
              <span
                ><code class="text-text-secondary">{'{' + 'args:N}'}</code> 第 N 个词（0 起）</span
              >
            </div>
          </div>
        </div>
      </div>

      <!-- 回复列表（固定/随机模式） -->
      <div v-else class="space-y-2">
        <div
          v-for="(_reply, index) in localConfig.predefined?.replies"
          :key="index"
          class="flex items-center gap-2"
        >
          <input
            v-model="localConfig.predefined!.replies![index]"
            type="text"
            placeholder="回复内容"
            class="flex-1 px-3 py-2 text-xs rounded-[var(--radius-sm,8px)] bg-bg-quaternary text-text-primary placeholder:text-text-quaternary outline-none focus:ring-1 focus:ring-[var(--theme-primary)]"
            @input="emitUpdate()"
          />
          <button
            class="p-1.5 rounded-lg hover:bg-red-500/10 text-text-tertiary hover:text-red-500 transition-colors"
            aria-label="删除回复"
            @click="removeReply(index)"
          >
            <BsX :size="14" />
          </button>
        </div>
        <button
          class="flex items-center gap-1.5 px-3 py-1.5 text-xs text-text-tertiary hover:text-text-primary rounded-[var(--radius-sm,8px)] hover:bg-hover-bg transition-colors"
          @click="addReply"
        >
          <BsPlus :size="12" />
          添加回复
        </button>
      </div>
    </div>

    <!-- LLM 回复 -->
    <div v-if="localConfig.type === 'llm'" class="space-y-3">
      <!-- 从 AI 面板导入配置 -->
      <div v-if="aiStore.configs.length" class="flex items-center gap-2">
        <label class="text-xs text-text-secondary whitespace-nowrap">复用 AI 面板配置</label>
        <select
          class="flex-1 px-3 py-2 text-xs rounded-[var(--radius-sm,8px)] bg-bg-quaternary text-text-primary outline-none focus:ring-1 focus:ring-[var(--theme-primary)] cursor-pointer appearance-none"
          @change="importFromAiPanel(($event.target as HTMLSelectElement).value)"
        >
          <option value="" disabled selected>选择配置...</option>
          <option v-for="cfg in aiStore.configs" :key="cfg.id" :value="cfg.id">
            {{ cfg.name }}
          </option>
        </select>
      </div>
      <div>
        <label class="block text-xs text-text-secondary mb-1.5">API URL</label>
        <input
          v-model="localConfig.llm!.api_url"
          type="text"
          placeholder="https://api.openai.com/v1/chat/completions"
          class="w-full px-3 py-2 text-xs rounded-[var(--radius-sm,8px)] bg-bg-quaternary text-text-primary placeholder:text-text-quaternary outline-none focus:ring-1 focus:ring-[var(--theme-primary)]"
          @input="emitUpdate()"
        />
      </div>
      <div>
        <label class="block text-xs text-text-secondary mb-1.5">API Key</label>
        <input
          v-model="localConfig.llm!.api_key"
          type="password"
          placeholder="sk-xxx"
          class="w-full px-3 py-2 text-xs rounded-[var(--radius-sm,8px)] bg-bg-quaternary text-text-primary placeholder:text-text-quaternary outline-none focus:ring-1 focus:ring-[var(--theme-primary)]"
          @input="emitUpdate()"
        />
      </div>
      <div>
        <label class="block text-xs text-text-secondary mb-1.5">模型</label>
        <input
          v-model="localConfig.llm!.model"
          type="text"
          placeholder="gpt-4o"
          class="w-full px-3 py-2 text-xs rounded-[var(--radius-sm,8px)] bg-bg-quaternary text-text-primary placeholder:text-text-quaternary outline-none focus:ring-1 focus:ring-[var(--theme-primary)]"
          @input="emitUpdate()"
        />
      </div>
      <div>
        <label class="block text-xs text-text-secondary mb-1.5">System Prompt</label>
        <textarea
          v-model="localConfig.llm!.system_prompt"
          rows="4"
          placeholder="你是一个友好的助手..."
          class="w-full px-3 py-2 text-xs rounded-[var(--radius-sm,8px)] bg-bg-quaternary text-text-primary placeholder:text-text-quaternary outline-none focus:ring-1 focus:ring-[var(--theme-primary)] resize-none"
          @input="emitUpdate()"
        />
      </div>
      <div class="grid grid-cols-3 gap-3">
        <div>
          <label class="block text-xs text-text-secondary mb-1.5">Temperature</label>
          <input
            v-model.number="localConfig.llm!.temperature"
            type="number"
            min="0"
            max="2"
            step="0.1"
            class="w-full px-3 py-2 text-xs rounded-[var(--radius-sm,8px)] bg-bg-quaternary text-text-primary outline-none focus:ring-1 focus:ring-[var(--theme-primary)]"
            @input="emitUpdate()"
          />
        </div>
        <div>
          <label class="block text-xs text-text-secondary mb-1.5">Max Tokens</label>
          <input
            v-model.number="localConfig.llm!.max_tokens"
            type="number"
            min="1"
            max="4096"
            step="1"
            class="w-full px-3 py-2 text-xs rounded-[var(--radius-sm,8px)] bg-bg-quaternary text-text-primary outline-none focus:ring-1 focus:ring-[var(--theme-primary)]"
            @input="emitUpdate()"
          />
        </div>
        <div>
          <label class="block text-xs text-text-secondary mb-1.5">上下文窗口</label>
          <input
            v-model.number="localConfig.llm!.context_window"
            type="number"
            min="0"
            max="200"
            step="1"
            class="w-full px-3 py-2 text-xs rounded-[var(--radius-sm,8px)] bg-bg-quaternary text-text-primary outline-none focus:ring-1 focus:ring-[var(--theme-primary)]"
            @input="emitUpdate()"
          />
        </div>
      </div>
      <p class="text-xs text-text-quaternary">
        兼容 OpenAI API 格式的服务均可使用。上下文窗口 0 表示不限制。
      </p>
    </div>

    <!-- 工作流（Agent）回复 -->
    <div
      v-if="localConfig.type === 'workflow' || localConfig.type === 'special_mode'"
      class="space-y-3"
    >
      <!-- 事件链预览 -->
      <div
        v-if="(localConfig.workflow ?? localConfig.special_mode)?.events?.length"
        class="rounded-[var(--radius-sm,8px)] border border-border-subtle bg-bg-quaternary p-3"
      >
        <div class="flex items-center justify-between mb-2">
          <span class="text-xs text-text-secondary font-medium">
            事件链（{{ (localConfig.workflow ?? localConfig.special_mode)?.events?.length }}
            个事件）
          </span>
          <span class="text-xs text-text-quaternary">
            {{ (localConfig.workflow ?? localConfig.special_mode)?.end_conditions?.length || 0 }}
            个结束条件
          </span>
        </div>
        <div class="flex flex-wrap gap-1.5">
          <span
            v-for="event in (localConfig.workflow ?? localConfig.special_mode)?.events"
            :key="event.id"
            class="inline-flex items-center gap-1 px-2 py-0.5 text-xs rounded-[var(--radius-xs,4px)] bg-bg-tertiary text-text-secondary"
          >
            <span
              class="w-1.5 h-1.5 rounded-full"
              :style="{ background: getEventColor(event.type) }"
            />
            {{ event.name || event.id }}
          </span>
        </div>
      </div>

      <!-- 空状态 -->
      <div
        v-else
        class="rounded-[var(--radius-sm,8px)] border border-dashed border-border-subtle bg-bg-quaternary p-4 text-center"
      >
        <p class="text-xs text-text-quaternary">尚未配置事件链</p>
      </div>

      <!-- 打开全页面编辑按钮 -->
      <button
        class="flex items-center gap-2 w-full justify-center px-3 py-2.5 text-xs text-text-secondary rounded-[var(--radius-sm,8px)] border border-border-subtle bg-bg-quaternary hover:border-[var(--theme-primary)] hover:text-[var(--theme-primary)] transition-colors"
        @click="emit('openWorkflowEditor')"
      >
        <BsBoxArrowUpRight :size="14" />
        在新标签页中编辑事件链
      </button>

      <p class="text-xs text-text-quaternary">
        特殊模式（Agent）允许你构建多步骤事件链，适合 RPG、对话式任务等场景。
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue';
import { BsPlus, BsX, BsBoxArrowUpRight } from 'vue-icons-plus/bs';
import { useAiStore } from '../../../../stores/ai';
import type { ReplySpec, WorkflowSpec, TriggerSpec, WorkflowEvent } from '../../../../models/types';
import { getDefaultPorts } from '../../../../utils/portTypes';

interface Props {
  config: ReplySpec;
  trigger?: TriggerSpec;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  update: [config: ReplySpec];
  openWorkflowEditor: [];
}>();

const aiStore = useAiStore();

function importFromAiPanel(configId: string) {
  const config = aiStore.configs.find((c) => c.id === configId);
  if (!config || !localConfig.llm) return;
  localConfig.llm.api_url = config.apiUrl;
  localConfig.llm.api_key = config.apiKey;
  localConfig.llm.model = config.model;
  localConfig.llm.temperature = config.temperature;
  if (config.maxTokens) {
    localConfig.llm.max_tokens = config.maxTokens;
  }
  emitUpdate();
}

const types = [
  { value: 'predefined' as const, label: '预定义' },
  { value: 'llm' as const, label: 'LLM' },
  { value: 'workflow' as const, label: '工作流' },
  { value: 'special_mode' as const, label: '特殊模式（旧）' },
];

const predefinedModes = [
  { value: 'fixed' as const, label: '固定' },
  { value: 'random' as const, label: '随机' },
  { value: 'template' as const, label: '模板' },
];

const defaultPredefined = {
  mode: 'random' as const,
  replies: ['...'],
  template: '',
};

const defaultLLM = {
  api_url: '',
  api_key: '',
  model: '',
  system_prompt: '',
  temperature: 0.7,
  max_tokens: 1000,
  context_window: 20,
};

const defaultWorkflow: WorkflowSpec = {
  events: [],
  connections: [],
  end_conditions: [],
};

function generateTriggerName(trigger?: TriggerSpec): string {
  if (!trigger) return '规则触发';
  if (trigger.type === 'probability') {
    return `概率触发（${Math.round((trigger.probability ?? 0) * 100)}%）`;
  }
  if (!trigger.rules?.length) return '规则触发';
  return `规则触发（${trigger.rules.length} 条）`;
}

function createDefaultTriggerEvent(trigger?: TriggerSpec): WorkflowEvent {
  return {
    id: 'evt_trigger_default',
    type: 'trigger',
    name: generateTriggerName(trigger),
    config: {},
    ports: getDefaultPorts('trigger'),
  };
}

function handleTypeSwitch(type: ReplySpec['type']) {
  localConfig.type = type;
  if (
    (type === 'workflow' || type === 'special_mode') &&
    (!localConfig.workflow || localConfig.workflow.events.length === 0)
  ) {
    localConfig.workflow = {
      events: [createDefaultTriggerEvent(props.trigger)],
      connections: [],
      end_conditions: [],
    };
  }
  emitUpdate();
}

function buildLocalConfig(config: ReplySpec): ReplySpec {
  return {
    type: config?.type || 'predefined',
    predefined: config?.predefined
      ? { ...config.predefined, replies: [...(config.predefined.replies || [])] }
      : { ...defaultPredefined },
    llm: config?.llm ? { ...config.llm } : { ...defaultLLM },
    workflow: config?.workflow
      ? deepCloneWorkflow(config.workflow)
      : config?.special_mode
        ? deepCloneWorkflow(config.special_mode)
        : { ...defaultWorkflow },
    special_mode: config?.special_mode ? deepCloneWorkflow(config.special_mode) : undefined,
  };
}

const localConfig = reactive<ReplySpec>(buildLocalConfig(props.config));

watch(
  () => props.config,
  (newConfig) => {
    if (!newConfig) return;
    const rebuilt = buildLocalConfig(newConfig);
    localConfig.type = rebuilt.type;
    localConfig.predefined = rebuilt.predefined;
    localConfig.llm = rebuilt.llm;
    localConfig.workflow = rebuilt.workflow;
    localConfig.special_mode = rebuilt.special_mode;
  },
  { deep: true }
);

function emitUpdate() {
  emit('update', {
    ...localConfig,
    workflow: deepCloneWorkflow(localConfig.workflow),
    special_mode: deepCloneWorkflow(localConfig.special_mode),
  });
}

function deepCloneWorkflow(spec?: WorkflowSpec): WorkflowSpec | undefined {
  if (!spec) return undefined;
  return {
    events: spec.events.map((e) => ({ ...e, config: { ...e.config } })),
    connections: spec.connections?.map((c) => ({ ...c })) || [],
    end_conditions: spec.end_conditions.map((c) => ({ ...c })),
  };
}

function getEventColor(type: string): string {
  const colors: Record<string, string> = {
    llm: 'var(--theme-primary, #5A8F4E)',
    builtin: 'var(--color-info, #2563eb)',
    reply: 'var(--color-success, #16a34a)',
  };
  return colors[type] || 'var(--text-quaternary-color, #a8a29e)';
}

function setPredefinedMode(mode: 'fixed' | 'random' | 'template') {
  if (!localConfig.predefined) return;
  localConfig.predefined.mode = mode;
  if (mode === 'template') {
    localConfig.predefined.template = localConfig.predefined.template || '{input}';
  }
  if (!localConfig.predefined.replies?.length) {
    localConfig.predefined.replies = ['...'];
  }
  emitUpdate();
}

function addReply() {
  if (!localConfig.predefined) return;
  localConfig.predefined.replies!.push('');
  emitUpdate();
}

function removeReply(index: number) {
  localConfig.predefined!.replies!.splice(index, 1);
  emitUpdate();
}
</script>
