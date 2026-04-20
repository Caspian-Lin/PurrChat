<template>
  <Teleport to="body">
    <div v-if="visible" class="modal-overlay" @click.self="$emit('close')">
      <div class="modal-container">
        <!-- 头部 -->
        <div class="modal-header">
          <h3 class="modal-title">{{ isEditing ? '编辑事件' : '新建事件' }}</h3>
          <button class="modal-close" @click="$emit('close')">
            <BsX :size="18" />
          </button>
        </div>

        <!-- 类型选择（仅新建时） -->
        <div v-if="!isEditing" class="modal-body">
          <div class="type-selector">
            <button
              v-for="t in eventTypes"
              :key="t.value"
              class="type-card"
              :class="{ 'type-card--active': form.type === t.value }"
              @click="form.type = t.value"
            >
              <span class="type-card__icon">{{ t.icon }}</span>
              <span class="type-card__label">{{ t.label }}</span>
            </button>
          </div>
        </div>

        <!-- 配置表单 -->
        <div class="modal-body">
          <!-- 通用字段 -->
          <div class="form-group">
            <label class="form-label">事件名称</label>
            <input v-model="form.name" type="text" class="form-input" placeholder="事件名称" />
          </div>

          <!-- LLM 配置 -->
          <template v-if="form.type === 'llm'">
            <!-- 从 AI 面板导入配置 -->
            <div v-if="aiStore.configs.length" class="form-group">
              <label class="form-label">复用 AI 面板配置</label>
              <select
                class="form-input"
                style="cursor: pointer; appearance: none"
                @change="importFromAiPanel(($event.target as HTMLSelectElement).value)"
              >
                <option value="" disabled selected>选择配置...</option>
                <option v-for="cfg in aiStore.configs" :key="cfg.id" :value="cfg.id">
                  {{ cfg.name }}
                </option>
              </select>
            </div>
            <div class="form-group">
              <label class="form-label">API URL</label>
              <input
                v-model="form.config.api_url"
                type="text"
                class="form-input"
                placeholder="https://api.openai.com/v1/chat/completions"
              />
            </div>
            <div class="form-group">
              <label class="form-label">API Key</label>
              <input
                v-model="form.config.api_key"
                type="password"
                class="form-input"
                placeholder="sk-..."
              />
            </div>
            <div class="form-row">
              <div class="form-group">
                <label class="form-label">模型</label>
                <input
                  v-model="form.config.model"
                  type="text"
                  class="form-input"
                  placeholder="gpt-4o"
                />
              </div>
              <div class="form-group">
                <label class="form-label">Temperature</label>
                <input
                  v-model.number="form.config.temperature"
                  type="number"
                  step="0.1"
                  min="0"
                  max="2"
                  class="form-input"
                />
              </div>
            </div>
            <div class="form-group">
              <label class="form-label">System Prompt</label>
              <textarea
                v-model="form.config.system_prompt"
                class="form-textarea"
                rows="3"
                placeholder="你是一个..."
              />
            </div>
            <div class="form-row">
              <div class="form-group">
                <label class="form-label">Max Tokens</label>
                <input
                  v-model.number="form.config.max_tokens"
                  type="number"
                  min="1"
                  class="form-input"
                />
              </div>
              <div class="form-group">
                <label class="form-label">上下文窗口</label>
                <input
                  v-model.number="form.config.context_window"
                  type="number"
                  min="1"
                  class="form-input"
                />
              </div>
            </div>
          </template>

          <!-- 内置事件配置 -->
          <template v-if="form.type === 'builtin'">
            <div class="form-group">
              <label class="form-label">内置类型</label>
              <select v-model="form.config.builtin_type" class="form-input">
                <option v-for="bt in builtinTypes" :key="bt.value" :value="bt.value">
                  {{ bt.label }}
                </option>
              </select>
            </div>
            <div v-if="form.config.builtin_type === 'random_number'" class="form-row">
              <div class="form-group">
                <label class="form-label">最小值</label>
                <input v-model.number="form.config.min" type="number" class="form-input" />
              </div>
              <div class="form-group">
                <label class="form-label">最大值</label>
                <input v-model.number="form.config.max" type="number" class="form-input" />
              </div>
            </div>
            <div v-if="form.config.builtin_type === 'echo'" class="form-row">
              <div class="form-group">
                <label class="form-label">前缀</label>
                <input v-model="form.config.prefix" type="text" class="form-input" />
              </div>
              <div class="form-group">
                <label class="form-label">后缀</label>
                <input v-model="form.config.suffix" type="text" class="form-input" />
              </div>
            </div>
            <div v-if="form.config.builtin_type === 'template'">
              <div class="form-group">
                <label class="form-label">模板</label>
                <textarea
                  v-model="form.config.template"
                  class="form-textarea"
                  rows="3"
                  placeholder="你好，{input}！"
                />
                <p class="form-hint">
                  可用变量：{'{input}'} 当前消息 · {'{username}'} 发送者 · {'{time}'} 时间 ·
                  {'{args}'} 除首个词外的参数 · {'{args:N}'} 第 N 个词（0 起）·
                  以及事件链中其他事件设置的变量
                </p>
              </div>
            </div>
          </template>

          <!-- Python 事件配置 -->
          <template v-if="form.type === 'python'">
            <div class="form-group">
              <label class="form-label">Python 代码</label>
              <textarea
                v-model="form.config.code"
                class="form-textarea form-textarea--code"
                rows="8"
                placeholder="def run(context, input_data):&#10;    # 处理输入&#10;    result = input_data.get('input', '')&#10;    return {'result': result}"
                spellcheck="false"
              />
            </div>
            <div class="form-group">
              <label class="form-label">超时 (ms)</label>
              <input
                v-model.number="form.config.timeout_ms"
                type="number"
                min="1000"
                max="30000"
                step="1000"
                class="form-input"
              />
            </div>
          </template>

          <!-- 回复事件配置 -->
          <template v-if="form.type === 'reply'">
            <div class="form-group">
              <label class="form-label">回复模板</label>
              <textarea
                v-model="form.config.template"
                class="form-textarea"
                rows="3"
                placeholder="使用 $事件ID.output 引用其他事件的输出"
              />
              <p class="form-hint">
                可用变量：{'{input}'} 当前消息 · {'{username}'} 发送者 · {'{time}'} 时间 ·
                {'{args}'} 除首个词外的参数 · {'{args:N}'} 第 N 个词（0 起）。
                用 $事件ID.output 引用事件输出，用 || 分隔表示默认值
              </p>
            </div>
          </template>

          <!-- 下一个事件 -->
          <div class="form-group">
            <label class="form-label">下一步事件</label>
            <div v-if="availableNextEvents.length > 0" class="next-events">
              <label
                v-for="evt in availableNextEvents"
                :key="evt.id"
                class="next-event-item"
                :class="{ 'next-event-item--active': form.next.includes(evt.id) }"
              >
                <input
                  type="checkbox"
                  :checked="form.next.includes(evt.id)"
                  @change="toggleNext(evt.id)"
                />
                <span>{{ evt.name }}</span>
              </label>
            </div>
            <p v-else class="form-hint">没有可连接的事件</p>
          </div>
        </div>

        <!-- 底部 -->
        <div class="modal-footer">
          <button v-if="isEditing" class="btn btn--danger" @click="$emit('delete', form.id)">
            删除
          </button>
          <div class="modal-footer__right">
            <button class="btn btn--ghost" @click="$emit('close')">取消</button>
            <button class="btn btn--primary" @click="handleConfirm">确认</button>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { reactive, computed, watch } from 'vue';
import { BsX } from 'vue-icons-plus/bs';
import { useAiStore } from '../../../../../stores/ai';
import { useAuthStore } from '../../../../../stores/auth';
import type { SpecialModeEvent } from '../../../../../models/types';

interface Props {
  visible: boolean;
  editingEvent?: SpecialModeEvent | null;
  existingEvents?: SpecialModeEvent[];
}

const props = withDefaults(defineProps<Props>(), {
  editingEvent: null,
  existingEvents: () => [],
});

const emit = defineEmits<{
  close: [];
  confirm: [event: SpecialModeEvent];
  delete: [eventId: string];
}>();

const isEditing = computed(() => !!props.editingEvent);

const aiStore = useAiStore();
const authStore = useAuthStore();

// 初始化 AI store（特殊模式编辑器在新标签页打开，AiPanel 不会挂载）
aiStore.initStore(authStore.currentUser?.id);

function importFromAiPanel(configId: string) {
  const config = aiStore.configs.find((c) => c.id === configId);
  if (!config) return;
  const cfg = form.config as Record<string, any>;
  cfg.api_url = config.apiUrl;
  cfg.api_key = config.apiKey;
  cfg.model = config.model;
  cfg.temperature = config.temperature;
  if (config.maxTokens) cfg.max_tokens = config.maxTokens;
}

const eventTypes = [
  { value: 'llm' as const, label: 'LLM', icon: '🧠' },
  { value: 'builtin' as const, label: '内置', icon: '⚙' },
  { value: 'python' as const, label: 'Python', icon: '🐍' },
  { value: 'reply' as const, label: '回复', icon: '💬' },
];

const builtinTypes = [
  { value: 'random_number', label: '随机数' },
  { value: 'haiku', label: '俳句' },
  { value: 'echo', label: '回显' },
  { value: 'count', label: '计数器' },
  { value: 'template', label: '模板' },
];

const defaultForm = (): Omit<SpecialModeEvent, 'id'> => ({
  type: 'llm',
  name: '',
  config: {
    api_url: '',
    api_key: '',
    model: '',
    system_prompt: '',
    temperature: 0.7,
    max_tokens: 1000,
    context_window: 20,
  },
  next: [],
});

const form = reactive<Omit<SpecialModeEvent, 'id'> & { id: string }>({
  id: '',
  type: 'llm',
  name: '',
  config: {},
  next: [],
});

watch(
  () => props.visible,
  () => {
    if (props.visible && props.editingEvent) {
      Object.assign(form, {
        id: props.editingEvent.id,
        type: props.editingEvent.type,
        name: props.editingEvent.name,
        config: { ...props.editingEvent.config },
        next: [...(props.editingEvent.next || [])],
      });
    } else if (props.visible) {
      const defaults = defaultForm();
      Object.assign(form, { ...defaults, id: `evt_${Date.now()}` });
    }
  }
);

const availableNextEvents = computed(() => {
  return props.existingEvents.filter((evt) => evt.id !== form.id);
});

function toggleNext(eventId: string) {
  const idx = form.next.indexOf(eventId);
  if (idx >= 0) {
    form.next.splice(idx, 1);
  } else {
    form.next.push(eventId);
  }
}

function handleConfirm() {
  if (!form.name.trim()) return;
  emit('confirm', { ...form });
  emit('close');
}
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.4);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
  backdrop-filter: blur(2px);
}

.modal-container {
  background: var(--bg-primary, #fff);
  border-radius: var(--radius-lg, 12px);
  width: 520px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.12);
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-subtle, rgba(0, 0, 0, 0.06));
}

.modal-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary, #1a1a1a);
}

.modal-close {
  padding: 4px;
  border-radius: var(--radius-xs, 4px);
  color: var(--text-tertiary, #999);
  cursor: pointer;
  transition: color 0.15s;
}
.modal-close:hover {
  color: var(--text-primary, #1a1a1a);
  background: var(--bg-quaternary, #f0efed);
}

.modal-body {
  padding: 16px 20px;
  overflow-y: auto;
}

.modal-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 20px;
  border-top: 1px solid var(--border-subtle, rgba(0, 0, 0, 0.06));
}

.modal-footer__right {
  display: flex;
  gap: 8px;
}

.type-selector {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 8px;
}

.type-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 12px 8px;
  border-radius: var(--radius-sm, 8px);
  border: 1px solid var(--border-subtle, rgba(0, 0, 0, 0.06));
  cursor: pointer;
  transition: all 0.15s;
}
.type-card:hover {
  background: var(--bg-quaternary, #f0efed);
}
.type-card--active {
  border-color: var(--theme-primary, #5a8f4e);
  background: rgba(90, 143, 78, 0.06);
}

.type-card__icon {
  font-size: 24px;
}

.type-card__label {
  font-size: 12px;
  color: var(--text-secondary, #666);
}

.form-group {
  margin-bottom: 12px;
}

.form-label {
  display: block;
  font-size: 12px;
  color: var(--text-secondary, #666);
  margin-bottom: 4px;
}

.form-input {
  width: 100%;
  padding: 8px 10px;
  font-size: 13px;
  border-radius: var(--radius-xs, 4px);
  border: 1px solid var(--border-subtle, rgba(0, 0, 0, 0.1));
  background: var(--bg-quaternary, #f8f7f5);
  color: var(--text-primary, #1a1a1a);
  outline: none;
  transition: border-color 0.15s;
  box-sizing: border-box;
}
.form-input:focus {
  border-color: var(--theme-primary, #5a8f4e);
}

.form-textarea {
  width: 100%;
  padding: 8px 10px;
  font-size: 13px;
  border-radius: var(--radius-xs, 4px);
  border: 1px solid var(--border-subtle, rgba(0, 0, 0, 0.1));
  background: var(--bg-quaternary, #f8f7f5);
  color: var(--text-primary, #1a1a1a);
  outline: none;
  resize: vertical;
  font-family: inherit;
  transition: border-color 0.15s;
  box-sizing: border-box;
}
.form-textarea:focus {
  border-color: var(--theme-primary, #5a8f4e);
}

.form-textarea--code {
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 12px;
  line-height: 1.5;
  tab-size: 2;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.form-hint {
  font-size: 11px;
  color: var(--text-tertiary, #999);
  margin-top: 4px;
}

.next-events {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.next-event-item {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border-radius: var(--radius-xs, 4px);
  border: 1px solid var(--border-subtle, rgba(0, 0, 0, 0.06));
  font-size: 12px;
  color: var(--text-secondary, #666);
  cursor: pointer;
  transition: all 0.15s;
}
.next-event-item--active {
  border-color: var(--theme-primary, #5a8f4e);
  color: var(--theme-primary, #5a8f4e);
  background: rgba(90, 143, 78, 0.06);
}

.btn {
  padding: 6px 14px;
  font-size: 13px;
  border-radius: var(--radius-xs, 4px);
  cursor: pointer;
  transition: all 0.15s;
  border: none;
}

.btn--ghost {
  background: var(--bg-quaternary, #f0efed);
  color: var(--text-secondary, #666);
}
.btn--ghost:hover {
  background: var(--bg-tertiary, #e8e7e5);
}

.btn--primary {
  background: var(--theme-primary, #5a8f4e);
  color: white;
}
.btn--primary:hover {
  opacity: 0.9;
}

.btn--danger {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}
.btn--danger:hover {
  background: rgba(239, 68, 68, 0.15);
}
</style>
