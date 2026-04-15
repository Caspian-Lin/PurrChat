<template>
  <BaseModal
    :show="show"
    :title="editingConfig ? '编辑 AI 配置' : '添加 AI 配置'"
    class="max-w-lg"
    @update:show="emit('update:show', $event)"
  >
    <div class="flex flex-col gap-4">
      <!-- 配置名称 -->
      <div>
        <label class="block text-sm font-medium mb-2" style="color: var(--text-color)">
          配置名称
        </label>
        <input
          v-model="form.name"
          type="text"
          placeholder="如: GPT-4、Claude、DeepSeek"
          class="w-full px-4 py-2 rounded-lg border focus:outline-none focus:ring-2"
          style="
            background: var(--surface-color);
            border-color: var(--border-color);
            color: var(--text-color);
          "
        />
      </div>

      <!-- API Base URL -->
      <div>
        <label class="block text-sm font-medium mb-2" style="color: var(--text-color)">
          API Base URL
        </label>
        <input
          v-model="form.apiUrl"
          type="text"
          placeholder="https://api.openai.com/v1"
          class="w-full px-4 py-2 rounded-lg border focus:outline-none focus:ring-2"
          style="
            background: var(--surface-color);
            border-color: var(--border-color);
            color: var(--text-color);
          "
        />
        <p class="text-xs mt-1" style="color: var(--text-tertiary-color)">
          兼容 OpenAI API 格式，如 OpenAI、DeepSeek、Ollama 等
        </p>
      </div>

      <!-- API Key -->
      <div>
        <label class="block text-sm font-medium mb-2" style="color: var(--text-color)">
          API Key
        </label>
        <div class="relative">
          <input
            v-model="form.apiKey"
            :type="showApiKey ? 'text' : 'password'"
            placeholder="sk-..."
            class="w-full px-4 py-2 pr-10 rounded-lg border focus:outline-none focus:ring-2"
            style="
              background: var(--surface-color);
              border-color: var(--border-color);
              color: var(--text-color);
            "
          />
          <button
            class="absolute right-3 top-1/2 -translate-y-1/2 text-text-tertiary hover:text-text-primary transition-colors"
            @click="showApiKey = !showApiKey"
          >
            <BsEye v-if="!showApiKey" :size="16" />
            <BsEyeSlash v-else :size="16" />
          </button>
        </div>
        <p class="text-xs mt-1" style="color: var(--text-tertiary-color)">
          API Key 仅保存在本地浏览器中，不会上传到服务器
        </p>
      </div>

      <!-- 模型名称 -->
      <div>
        <label class="block text-sm font-medium mb-2" style="color: var(--text-color)">
          模型名称
        </label>
        <input
          v-model="form.model"
          type="text"
          placeholder="如: gpt-4o、claude-3-opus、deepseek-chat"
          class="w-full px-4 py-2 rounded-lg border focus:outline-none focus:ring-2"
          style="
            background: var(--surface-color);
            border-color: var(--border-color);
            color: var(--text-color);
          "
        />
      </div>

      <!-- 温度参数 -->
      <div>
        <label class="block text-sm font-medium mb-2" style="color: var(--text-color)">
          温度 (Temperature)
        </label>
        <div class="flex items-center gap-3">
          <input
            v-model.number="form.temperature"
            type="range"
            min="0"
            max="2"
            step="0.1"
            class="flex-1"
          />
          <span
            class="w-12 text-center px-2 py-1 rounded-md text-sm font-mono"
            style="
              background: var(--surface-color);
              border: 1px solid var(--border-color);
              color: var(--text-color);
            "
          >
            {{ form.temperature.toFixed(1) }}
          </span>
        </div>
        <p class="text-xs mt-1" style="color: var(--text-tertiary-color)">
          较低值使输出更确定，较高值使输出更随机
        </p>
      </div>

      <!-- 最大 Token 数 -->
      <div>
        <label class="block text-sm font-medium mb-2" style="color: var(--text-color)">
          最大 Token 数 <span class="text-text-tertiary">(可选)</span>
        </label>
        <input
          v-model.number="form.maxTokens"
          type="number"
          placeholder="2048"
          min="1"
          max="128000"
          class="w-full px-4 py-2 rounded-lg border focus:outline-none focus:ring-2"
          style="
            background: var(--surface-color);
            border-color: var(--border-color);
            color: var(--text-color);
          "
        />
      </div>

      <!-- 按钮 -->
      <div class="flex gap-3 mt-2">
        <button
          class="flex-1 px-4 py-2 bg-bg-secondary text-text-primary rounded-md hover:bg-hover-bg transition-colors"
          @click="emit('update:show', false)"
        >
          取消
        </button>
        <button
          class="flex-1 px-4 py-2 bg-[var(--theme-primary)] text-white rounded-md hover:opacity-80 transition-opacity disabled:opacity-50 disabled:cursor-not-allowed"
          :disabled="!isValid"
          @click="handleSave"
        >
          {{ editingConfig ? '保存修改' : '添加配置' }}
        </button>
      </div>
    </div>
  </BaseModal>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import BaseModal from '../common/BaseModal.vue';
import { BsEye, BsEyeSlash } from 'vue-icons-plus/bs';
import type { AiConfig } from '../../models/types';

interface Props {
  show: boolean;
  editingConfig?: AiConfig | null;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  'update:show': [value: boolean];
  'config-saved': [
    data: {
      name: string;
      apiUrl: string;
      apiKey: string;
      model: string;
      temperature: number;
      maxTokens?: number;
    },
  ];
}>();

const showApiKey = ref(false);

const form = ref({
  name: '',
  apiUrl: '',
  apiKey: '',
  model: '',
  temperature: 0.7,
  maxTokens: undefined as number | undefined,
});

const resetForm = () => {
  form.value = {
    name: '',
    apiUrl: '',
    apiKey: '',
    model: '',
    temperature: 0.7,
    maxTokens: undefined,
  };
  showApiKey.value = false;
};

const isValid = computed(() => {
  return (
    form.value.name.trim().length > 0 &&
    form.value.apiUrl.trim().length > 0 &&
    form.value.apiKey.trim().length > 0 &&
    form.value.model.trim().length > 0
  );
});

// 编辑模式时填充表单
watch(
  () => props.editingConfig,
  (config) => {
    if (config) {
      form.value = {
        name: config.name,
        apiUrl: config.apiUrl,
        apiKey: config.apiKey,
        model: config.model,
        temperature: config.temperature,
        maxTokens: config.maxTokens,
      };
    } else {
      resetForm();
    }
  },
  { immediate: true }
);

// 弹窗关闭时重置
watch(
  () => props.show,
  (newShow) => {
    if (newShow && !props.editingConfig) {
      resetForm();
    }
  }
);

const handleSave = () => {
  if (!isValid.value) return;
  emit('config-saved', { ...form.value });
};
</script>
