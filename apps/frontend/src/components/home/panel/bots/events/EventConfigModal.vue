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
          <!-- 控制流 -->
          <div class="type-section-label">控制流</div>
          <div class="type-selector type-selector--control">
            <button
              v-for="t in controlTypes"
              :key="t.value"
              class="type-card"
              :class="{ 'type-card--active': form.type === t.value }"
              @click="selectType(t.value)"
            >
              <span class="type-card__icon">{{ t.icon }}</span>
              <span class="type-card__label">{{ t.label }}</span>
            </button>
          </div>

          <!-- 处理 / 输出 -->
          <div class="type-section-label">处理 / 输出</div>
          <div class="type-selector type-selector--process">
            <button
              v-for="t in processTypes"
              :key="t.value"
              class="type-card"
              :class="{ 'type-card--active': form.type === t.value }"
              @click="selectType(t.value)"
            >
              <span class="type-card__icon">{{ t.icon }}</span>
              <span class="type-card__label">{{ t.label }}</span>
            </button>
          </div>
        </div>

        <!-- 配置表单 -->
        <div class="modal-body">
          <!-- 通用字段：事件名称 -->
          <div class="form-group">
            <label class="form-label">事件名称</label>
            <input v-model="form.name" type="text" class="form-input" placeholder="事件名称" />
            <p v-if="nameValidationError" class="validation-error">{{ nameValidationError }}</p>
          </div>

          <!-- trigger：仅名称，无额外配置 -->

          <!-- end：仅名称 -->

          <!-- wait 配置 -->
          <template v-if="form.type === 'wait'">
            <div class="form-group">
              <label class="form-label">等待类型</label>
              <select v-model="form.config.wait_type" class="form-input">
                <option value="user_message">用户消息</option>
                <option value="custom">自定义条件</option>
              </select>
            </div>
            <div v-if="form.config.wait_type === 'custom'" class="form-group">
              <label class="form-label">条件表达式</label>
              <textarea
                v-model="form.config.condition"
                class="form-textarea"
                rows="3"
                placeholder="输入等待条件表达式..."
              />
            </div>
          </template>

          <!-- if 配置 -->
          <template v-if="form.type === 'if'">
            <div class="form-group">
              <label class="form-label">运算符</label>
              <select v-model="form.config.operator" class="form-input">
                <option value="==">等于 ==</option>
                <option value="!=">不等于 !=</option>
                <option value="contains">包含</option>
                <option value=">">&gt; 大于</option>
                <option value="<">&lt; 小于</option>
              </select>
            </div>
            <p class="form-hint">
              条件由连接到"左操作数"和"右操作数"输入端口的值决定。
              断开输入时使用下方默认值。
            </p>
            <div class="form-row">
              <div class="form-group">
                <label class="form-label">左操作数（默认值）</label>
                <input
                  v-model="form.config.left_default"
                  type="text"
                  class="form-input"
                  placeholder="未连接时的默认值"
                />
              </div>
              <div class="form-group">
                <label class="form-label">右操作数（默认值）</label>
                <input
                  v-model="form.config.right_default"
                  type="text"
                  class="form-input"
                  placeholder="未连接时的默认值"
                />
              </div>
            </div>
          </template>

          <!-- loop 配置 -->
          <template v-if="form.type === 'loop'">
            <div class="form-group">
              <label class="form-label">条件表达式</label>
              <textarea
                v-model="form.config.condition"
                class="form-textarea"
                rows="3"
                placeholder="输入循环条件表达式..."
              />
            </div>
            <div class="form-group">
              <label class="form-label">最大迭代次数</label>
              <input
                v-model.number="form.config.max_iterations"
                type="number"
                min="1"
                class="form-input"
              />
              <p class="form-hint">设置合理的上限以避免无限循环</p>
            </div>
          </template>

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

          <!-- template 配置 -->
          <template v-if="form.type === 'template'">
            <div class="form-group">
              <label class="form-label">模板内容</label>
              <textarea
                v-model="form.config.template"
                class="form-textarea"
                rows="4"
                placeholder="使用 {variable} 引用变量..."
              />
              <p class="form-hint">
                可用变量：{'{input}'} 当前消息 · {'{username}'} 发送者 · {'{time}'} 时间 ·
                {'{args}'} 除首个词外的参数 · {'{args:N}'} 第 N 个词（0 起）
              </p>
            </div>
          </template>

          <!-- reply 配置 -->
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
                {'{args}'} 除首个词外的参数 · {'{args:N}'} 第 N 个词（0 起）。 用 $事件ID.output
                引用事件输出，用 || 分隔表示默认值
              </p>
            </div>
          </template>

          <!-- history 配置 -->
          <template v-if="form.type === 'history'">
            <div class="form-group">
              <label class="form-label">获取消息条数</label>
              <input
                v-model.number="form.config.count"
                type="number"
                min="1"
                max="100"
                class="form-input"
                placeholder="20"
              />
              <p class="form-hint">
                获取最近 N 条会话消息，格式化为 prompt 字符串。图片消息将显示为 [图片]。
              </p>
            </div>
          </template>

          <!-- 自定义端口 -->
          <div v-if="supportsCustomPorts" class="form-group">
            <label class="form-label">自定义端口</label>
            <div class="custom-ports">
              <div v-for="(port, idx) in customPorts" :key="idx" class="custom-port-row">
                <select v-model="port.direction" class="form-input" style="width: 70px">
                  <option value="input">输入</option>
                  <option value="output">输出</option>
                </select>
                <input v-model="port.name" placeholder="端口名称" class="form-input" />
                <select v-model="port.dataType" class="form-input" style="width: 80px">
                  <option value="string">字符串</option>
                  <option value="number">数值</option>
                  <option value="boolean">布尔</option>
                  <option value="any">任意</option>
                </select>
                <button class="port-remove-btn" @click="removeCustomPort(idx)">×</button>
              </div>
              <button class="add-port-btn" @click="addCustomPort">+ 添加端口</button>
            </div>
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
import { reactive, computed, watch, ref } from 'vue';
import { BsX } from 'vue-icons-plus/bs';
import { useAiStore } from '../../../../../stores/ai';
import { useAuthStore } from '../../../../../stores/auth';
import { getDefaultPorts, NODE_TYPE_META, type EventType } from '../../../../../utils/portTypes';
import type { EventPort, SpecialModeEvent } from '../../../../../models/types';

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

// 名称验证错误
const nameValidationError = ref('');

// 自定义端口列表
const customPorts = reactive<{ name: string; dataType: string; direction: 'input' | 'output' }[]>([]);

// 类型分组
const controlTypes = (['trigger', 'end', 'wait', 'if', 'loop'] as EventType[]).map((value) => ({
  value,
  ...NODE_TYPE_META[value],
}));

const processTypes = (['llm', 'builtin', 'python', 'template', 'reply'] as EventType[]).map(
  (value) => ({ value, ...NODE_TYPE_META[value] })
);

// 是否为支持自定义端口的节点类型
const supportsCustomPorts = computed(() =>
  ['llm', 'builtin', 'python', 'template', 'if', 'wait', 'reply', 'history'].includes(form.type)
);

const builtinTypes = [
  { value: 'random_number', label: '随机数' },
  { value: 'haiku', label: '俳句' },
  { value: 'echo', label: '回显' },
  { value: 'count', label: '计数器' },
  { value: 'template', label: '模板' },
];

// 根据类型返回默认 config
function getDefaultConfig(type: EventType): Record<string, any> {
  switch (type) {
    case 'llm':
      return {
        api_url: '',
        api_key: '',
        model: '',
        system_prompt: '',
        temperature: 0.7,
        max_tokens: 1000,
        context_window: 20,
      };
    case 'builtin':
      return { builtin_type: 'random_number' };
    case 'python':
      return { code: '', timeout_ms: 5000 };
    case 'template':
      return { template: '' };
    case 'history':
      return { count: 20 };
    case 'reply':
      return { template: '' };
    case 'wait':
      return { wait_type: 'user_message', condition: '' };
    case 'if':
      return { operator: '==', left_default: '', right_default: '' };
    case 'loop':
      return { condition: '', max_iterations: 10 };
    case 'trigger':
    case 'end':
    default:
      return {};
  }
}

interface FormData {
  id: string;
  type: EventType;
  name: string;
  config: Record<string, any>;
  ports: EventPort[];
  position?: { x: number; y: number };
}

const form = reactive<FormData>({
  id: '',
  type: 'llm',
  name: '',
  config: getDefaultConfig('llm'),
  ports: getDefaultPorts('llm'),
});

// 选择类型时重置 config、ports 和名称
function selectType(type: EventType) {
  form.type = type;
  form.config = getDefaultConfig(type);
  form.ports = getDefaultPorts(type);
  form.name = NODE_TYPE_META[type].label;
  customPorts.length = 0;
  nameValidationError.value = '';
}

function importFromAiPanel(configId: string) {
  const config = aiStore.configs.find((c) => c.id === configId);
  if (!config) return;
  form.config.api_url = config.apiUrl;
  form.config.api_key = config.apiKey;
  form.config.model = config.model;
  form.config.temperature = config.temperature;
  if (config.maxTokens) form.config.max_tokens = config.maxTokens;
}

// 自定义端口操作
function addCustomPort() {
  customPorts.push({ name: '', dataType: 'string', direction: 'input' });
}

function removeCustomPort(idx: number) {
  customPorts.splice(idx, 1);
}

// 从已有事件的 ports 中提取自定义端口
function extractCustomPorts(event: SpecialModeEvent): { name: string; dataType: string; direction: 'input' | 'output' }[] {
  if (!event.ports) return [];
  const defaultPorts = getDefaultPorts(event.type);
  const defaultIds = new Set(defaultPorts.map((p) => p.id));
  return event.ports
    .filter((p) => !defaultIds.has(p.id))
    .map((p) => ({ name: p.name, dataType: p.dataType, direction: p.direction }));
}

watch(
  () => props.visible,
  () => {
    if (props.visible && props.editingEvent) {
      Object.assign(form, {
        id: props.editingEvent.id,
        type: props.editingEvent.type,
        name: props.editingEvent.name,
        config: { ...props.editingEvent.config },
        ports: [...(props.editingEvent.ports || getDefaultPorts(props.editingEvent.type))],
        position: props.editingEvent.position ? { ...props.editingEvent.position } : undefined,
      });
      // 恢复自定义端口
      customPorts.length = 0;
      const extracted = extractCustomPorts(props.editingEvent);
      extracted.forEach((p) => customPorts.push(p));
    } else if (props.visible) {
      const type: EventType = 'llm';
      Object.assign(form, {
        id: `evt_${Date.now()}`,
        type,
        name: '',
        config: getDefaultConfig(type),
        ports: getDefaultPorts(type),
        position: undefined,
      });
      customPorts.length = 0;
    }
    nameValidationError.value = '';
  }
);

function handleConfirm() {
  // 验证名称
  if (!form.name.trim()) {
    nameValidationError.value = '事件名称不能为空';
    return;
  }
  nameValidationError.value = '';

  // 合并默认端口和自定义端口
  const defaultPorts = getDefaultPorts(form.type);
  const customEventPorts: EventPort[] = customPorts
    .filter((p) => p.name.trim())
    .map((p) => ({
      id: `${p.direction === 'output' ? 'out_custom' : 'in_custom'}_${p.name}`,
      name: p.name,
      dataType: p.dataType as EventPort['dataType'],
      direction: p.direction,
    }));

  const event: SpecialModeEvent = {
    id: form.id,
    type: form.type,
    name: form.name,
    config: { ...form.config },
    ports: [...defaultPorts, ...customEventPorts],
  };

  if (form.position) {
    event.position = { ...form.position };
  }

  emit('confirm', event);
  emit('close');
}
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: var(--modal-overlay-color, rgba(0, 0, 0, 0.4));
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
  backdrop-filter: blur(2px);
}

.modal-container {
  background: var(--strong-background-color, #fff);
  border-radius: var(--radius-lg, 12px);
  width: 520px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  box-shadow: var(--shadow-lg, 0 8px 32px rgba(28, 25, 23, 0.1));
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.06));
}

.modal-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-color, #1c1917);
}

.modal-close {
  padding: 4px;
  border-radius: var(--radius-xs, 4px);
  color: var(--text-tertiary-color, #a8a29e);
  cursor: pointer;
  transition: color 0.15s;
}
.modal-close:hover {
  color: var(--text-color, #1c1917);
  background: var(--surface-tertiary-color, #e8e4de);
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
  border-top: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.06));
}

.modal-footer__right {
  display: flex;
  gap: 8px;
}

.type-section-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-tertiary-color, #a8a29e);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 8px;
}

.type-section-label:not(:first-child) {
  margin-top: 16px;
}

.type-selector {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
}

.type-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 12px 8px;
  border-radius: var(--radius-sm, 8px);
  border: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.06));
  cursor: pointer;
  transition: all 0.15s;
  background: none;
  color: var(--text-secondary-color, #57534e);
}
.type-card:hover {
  background: var(--surface-tertiary-color, #e8e4de);
}
.type-card--active {
  border-color: var(--theme-primary, #5a8f4e);
  background: color-mix(in srgb, var(--theme-primary, #5a8f4e) 6%, transparent);
}

.type-card__icon {
  font-size: 24px;
}

.type-card__label {
  font-size: 12px;
}

.form-group {
  margin-bottom: 12px;
}

.form-label {
  display: block;
  font-size: 12px;
  color: var(--text-secondary-color, #57534e);
  margin-bottom: 4px;
}

.form-input {
  width: 100%;
  padding: 8px 10px;
  font-size: 13px;
  border-radius: var(--radius-xs, 4px);
  border: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.1));
  background: var(--input-background, #fff);
  color: var(--text-color, #1c1917);
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
  border: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.1));
  background: var(--input-background, #fff);
  color: var(--text-color, #1c1917);
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
  color: var(--text-tertiary-color, #a8a29e);
  margin-top: 4px;
}

/* 自定义端口 */
.custom-ports {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.custom-port-row {
  display: grid;
  grid-template-columns: 70px 1fr 80px 32px;
  gap: 6px;
  align-items: center;
}

.add-port-btn {
  padding: 4px 10px;
  font-size: 11px;
  border-radius: var(--radius-xs, 4px);
  border: 1px dashed var(--border-subtle-color, rgba(0, 0, 0, 0.1));
  color: var(--text-tertiary-color, #a8a29e);
  cursor: pointer;
  background: none;
  transition: all 0.15s;
}
.add-port-btn:hover {
  border-color: var(--theme-primary, #5a8f4e);
  color: var(--theme-primary, #5a8f4e);
}

.port-remove-btn {
  width: 24px;
  height: 24px;
  border-radius: var(--radius-xs, 4px);
  border: none;
  background: none;
  color: var(--text-tertiary-color, #a8a29e);
  cursor: pointer;
  transition: all 0.15s;
}
.port-remove-btn:hover {
  color: var(--color-error, #dc2626);
  background: var(--color-error-bg, rgba(239, 68, 68, 0.06));
}

.validation-error {
  font-size: 11px;
  color: var(--color-error, #dc2626);
  margin-top: 4px;
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
  background: var(--surface-tertiary-color, #e8e4de);
  color: var(--text-secondary-color, #57534e);
}
.btn--ghost:hover {
  background: var(--surface-hover-color, #e2ddd7);
}

.btn--primary {
  background: var(--theme-primary, #5a8f4e);
  color: white;
}
.btn--primary:hover {
  opacity: 0.9;
}

.btn--danger {
  background: var(--color-error-bg, rgba(239, 68, 68, 0.06));
  color: var(--color-error, #dc2626);
}
.btn--danger:hover {
  background: var(--color-error-bg, rgba(239, 68, 68, 0.1));
}
</style>
