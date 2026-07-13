<template>
  <div class="flex flex-col h-full">
    <!-- 顶部工具栏 -->
    <div
      class="flex items-center gap-3 px-5 py-3 bg-bg-secondary border-b border-border-subtle flex-shrink-0"
    >
      <button
        class="p-1.5 rounded-lg hover:bg-hover-bg text-text-tertiary hover:text-text-primary transition-colors"
        aria-label="返回"
        title="返回"
        @click="$emit('back')"
      >
        <BsArrowLeft :size="18" />
      </button>
      <div class="flex-1 min-w-0">
        <h2 class="text-sm font-medium text-text-primary truncate">{{ form.name }}</h2>
        <p class="text-xs text-text-tertiary truncate">
          {{ form.description || '无描述' }}
        </p>
      </div>
      <span
        class="text-xs px-2 py-1 rounded-full"
        :class="
          form.status === 'active'
            ? 'bg-green-500/10 text-green-600'
            : 'bg-bg-quaternary text-text-tertiary'
        "
      >
        {{ form.status === 'active' ? '活跃' : '已禁用' }}
      </span>
      <!-- 导出/导入按钮（仅 owner） -->
      <button
        v-if="isOwned"
        class="p-1.5 rounded-lg hover:bg-hover-bg text-text-tertiary hover:text-text-primary transition-colors"
        aria-label="导入配置"
        title="导入配置"
        @click="handleImport"
      >
        <BsUpload :size="16" />
      </button>
      <button
        class="p-1.5 rounded-lg hover:bg-hover-bg text-text-tertiary hover:text-text-primary transition-colors"
        aria-label="导出配置"
        title="导出配置"
        @click="handleExport"
      >
        <BsDownload :size="16" />
      </button>
    </div>

    <!-- 非 owner：只读信息 + 操作 -->
    <div v-if="!isOwned" class="flex-1 overflow-y-auto">
      <div class="mx-auto max-w-2xl p-6 space-y-6">
        <!-- Bot 头像与名称 -->
        <div class="flex items-start gap-4">
          <div
            class="w-16 h-16 rounded-[var(--radius-lg,16px)] flex items-center justify-center flex-shrink-0 text-white font-bold"
            style="background: var(--theme-primary)"
          >
            <BsCpu v-if="!bot.avatar_url" :size="28" />
            <img
              v-else
              :src="bot.avatar_url"
              :alt="bot.name"
              class="w-full h-full rounded-[var(--radius-lg,16px)] object-cover"
              referrerpolicy="no-referrer"
            />
          </div>
          <div class="flex-1 min-w-0">
            <h2 class="text-lg font-semibold text-text-primary">{{ bot.name }}</h2>
            <p class="text-sm text-text-tertiary mt-1">{{ bot.description || '无描述' }}</p>
            <span
              class="inline-block mt-2 text-[10px] px-1.5 py-0.5 rounded-full bg-[var(--theme-primary)]/10 text-[var(--theme-primary)]"
            >
              公开 Bot
            </span>
          </div>
        </div>

        <!-- 操作按钮 -->
        <div class="flex gap-3">
          <button
            class="flex-1 flex items-center justify-center gap-2 px-4 py-2.5 text-sm font-medium rounded-[var(--radius-sm,8px)] text-white transition-colors"
            style="background: var(--theme-primary)"
            @click="$emit('create-conversation', bot.id)"
          >
            <BsChatDots :size="16" />
            开始对话
          </button>
          <button
            class="flex-1 flex items-center justify-center gap-2 px-4 py-2.5 text-sm font-medium rounded-[var(--radius-sm,8px)] bg-bg-quaternary text-text-secondary hover:bg-hover-bg transition-colors"
            @click="$emit('deploy', bot.id)"
          >
            <BsBoxArrowUpRight :size="16" />
            安装到群聊
          </button>
        </div>
      </div>
    </div>

    <!-- owner：完整编辑器 -->
    <div v-else class="flex-1 overflow-y-auto">
      <div class="mx-auto max-w-3xl p-6 space-y-8">
        <!-- 基本信息 -->
        <section>
          <h3 class="text-sm font-semibold text-text-primary mb-4 flex items-center gap-2">
            <BsInfoCircle :size="16" class="text-text-tertiary" />
            基本信息
          </h3>
          <div class="space-y-4">
            <div>
              <label class="block text-xs text-text-secondary mb-1.5">名称</label>
              <input
                v-model="form.name"
                type="text"
                maxlength="40"
                placeholder="Bot 名称"
                class="w-full px-3 py-2.5 text-sm rounded-[var(--radius-sm,8px)] bg-bg-quaternary text-text-primary placeholder:text-text-quaternary outline-none focus:ring-1 focus:ring-[var(--theme-primary)] transition-all"
              />
            </div>
            <div>
              <label class="block text-xs text-text-secondary mb-1.5">描述</label>
              <textarea
                v-model="form.description"
                maxlength="500"
                rows="2"
                placeholder="Bot 的简短描述..."
                class="w-full px-3 py-2.5 text-sm rounded-[var(--radius-sm,8px)] bg-bg-quaternary text-text-primary placeholder:text-text-quaternary outline-none focus:ring-1 focus:ring-[var(--theme-primary)] transition-all resize-none"
              />
            </div>
            <div>
              <label class="block text-xs text-text-secondary mb-1.5">可见性</label>
              <div class="flex gap-2">
                <button
                  v-for="v in visibilityOptions"
                  :key="v.value"
                  class="px-3 py-1.5 text-xs rounded-[var(--radius-sm,8px)] transition-colors"
                  :class="
                    form.visibility === v.value
                      ? 'text-white'
                      : 'bg-bg-quaternary text-text-secondary hover:bg-hover-bg'
                  "
                  :style="form.visibility === v.value ? { background: 'var(--theme-primary)' } : {}"
                  @click="form.visibility = v.value"
                >
                  {{ v.label }}
                </button>
              </div>
            </div>
            <div>
              <label class="block text-xs text-text-secondary mb-1.5">状态</label>
              <div class="flex gap-2">
                <button
                  class="px-3 py-1.5 text-xs rounded-[var(--radius-sm,8px)] transition-colors"
                  :class="
                    form.status === 'active'
                      ? 'text-white'
                      : 'bg-bg-quaternary text-text-secondary hover:bg-hover-bg'
                  "
                  :style="form.status === 'active' ? { background: 'var(--theme-primary)' } : {}"
                  @click="form.status = 'active'"
                >
                  启用
                </button>
                <button
                  class="px-3 py-1.5 text-xs rounded-[var(--radius-sm,8px)] transition-colors"
                  :class="
                    form.status === 'disabled'
                      ? 'text-white'
                      : 'bg-bg-quaternary text-text-secondary hover:bg-hover-bg'
                  "
                  :style="form.status === 'disabled' ? { background: 'var(--theme-primary)' } : {}"
                  @click="form.status = 'disabled'"
                >
                  禁用
                </button>
              </div>
            </div>
          </div>
        </section>

        <!-- 机制列表（仅 workflow bot） -->
        <section v-if="bot.bot_type !== 'external'">
          <h3 class="text-sm font-semibold text-text-primary mb-4 flex items-center gap-2">
            <BsGear :size="16" class="text-text-tertiary" />
            机制列表
            <span class="text-xs font-normal text-text-quaternary">
              消息从上到下依次匹配，首个匹配的机制将被执行
            </span>
          </h3>

          <!-- 机制卡片列表 -->
          <div class="space-y-3">
            <MechanismCard
              v-for="(mech, index) in form.mechanisms"
              :key="mech.id"
              :mechanism="mech"
              :can-move-up="index > 0"
              :can-move-down="index < form.mechanisms.length - 1"
              @update="updateMechanism(index, $event)"
              @delete="removeMechanism(index)"
              @move-up="moveMechanism(index, -1)"
              @move-down="moveMechanism(index, 1)"
              @open-workflow-editor="openWorkflowEditor"
            />
          </div>

          <!-- 添加机制 -->
          <div class="mt-3 flex gap-2">
            <button
              class="flex items-center gap-1.5 px-3 py-1.5 text-xs text-text-tertiary hover:text-text-primary rounded-[var(--radius-sm,8px)] hover:bg-hover-bg transition-colors"
              @click="addMechanism('rule')"
            >
              <BsPlus :size="12" />
              规则触发
            </button>
            <button
              class="flex items-center gap-1.5 px-3 py-1.5 text-xs text-text-tertiary hover:text-text-primary rounded-[var(--radius-sm,8px)] hover:bg-hover-bg transition-colors"
              :class="{ 'opacity-40 pointer-events-none': hasProbabilityMechanism }"
              @click="addMechanism('probability')"
            >
              <BsPlus :size="12" />
              概率触发
              <span v-if="hasProbabilityMechanism" class="text-text-quaternary">（已有）</span>
            </button>
          </div>
        </section>

        <!-- OneBot API 管理（仅 external bot） -->
        <template v-if="bot.bot_type === 'external'">
          <BotCredentialsPanel :bot-id="bot.id" />
          <BotApiGuide :bot-id="bot.id" />
        </template>

        <!-- 调用记录 -->
        <section class="mt-6">
          <h3 class="text-sm font-semibold text-text-primary mb-4 flex items-center gap-2">
            <BsClockHistory :size="16" class="text-text-tertiary" />
            调用记录
          </h3>
          <BotCallLogs :bot-id="bot.id" />
        </section>
      </div>
    </div>

    <!-- 浮动保存按钮（仅 owner） -->
    <div v-if="isOwned" class="fixed bottom-12 right-12 z-40 flex items-center gap-2">
      <Transition name="reset-btn">
        <button
          v-if="isDirty && !saving"
          class="flex items-center gap-1.5 px-4 py-2.5 text-sm font-medium rounded-[var(--radius-sm,8px)] bg-bg-quaternary text-text-secondary hover:bg-hover-bg transition-all duration-200 active:scale-[0.98]"
          @click="resetForm"
        >
          <BsArrowCounterclockwise :size="16" />
          重置
        </button>
      </Transition>
      <SaveButton :is-dirty="isDirty" :is-saving="saving" @save="handleSave" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, computed, watch } from 'vue';
import {
  BsArrowLeft,
  BsInfoCircle,
  BsGear,
  BsUpload,
  BsDownload,
  BsPlus,
  BsClockHistory,
  BsArrowCounterclockwise,
  BsCpu,
  BsChatDots,
  BsBoxArrowUpRight,
} from 'vue-icons-plus/bs';
import type {
  Bot,
  UpdateBotRequest,
  BotVisibility,
  Mechanism,
  MechanismConfig,
  TriggerSpec,
} from '../../../../models/types';
import MechanismCard from './MechanismCard.vue';
import BotCallLogs from './BotCallLogs.vue';
import BotCredentialsPanel from './BotCredentialsPanel.vue';
import BotApiGuide from './BotApiGuide.vue';
import SaveButton from '../settings/SaveButton.vue';

interface Props {
  bot: Bot;
  isOwned?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  isOwned: true,
});

const emit = defineEmits<{
  update: [botId: string, data: UpdateBotRequest];
  back: [];
  'create-conversation': [botId: string];
  deploy: [botId: string];
}>();

const saving = ref(false);

const visibilityOptions = [
  { value: 'private' as BotVisibility, label: '私有' },
  { value: 'public' as BotVisibility, label: '公开' },
  { value: 'global' as BotVisibility, label: '系统' },
];

// 从 Bot 数据中提取机制列表
function extractMechanisms(bot: Bot): Mechanism[] {
  if (bot.bot_type === 'external') return [];
  if (bot.mechanism_config?.mechanisms?.length) {
    return bot.mechanism_config.mechanisms.map((m) => deepCloneMechanism(m));
  }
  // 无机制时创建一个默认的空规则机制
  return [
    {
      id: 'mech_default',
      name: '默认机制',
      enabled: true,
      trigger: { type: 'rule', rules: [] },
    },
  ];
}

function deepCloneMechanism(m: Mechanism): Mechanism {
  return {
    id: m.id,
    name: m.name,
    enabled: m.enabled,
    trigger: {
      ...m.trigger,
      rules: m.trigger?.rules?.map((r) => ({ ...r })) || [],
    },
  };
}

const form = reactive({
  name: props.bot.name,
  description: props.bot.description,
  visibility: props.bot.visibility,
  status: props.bot.status,
  mechanisms: extractMechanisms(props.bot),
});

// ─── Dirty 检测 ──────────────────────────────────────────────

function serializeForm(): string {
  return JSON.stringify({
    name: form.name,
    description: form.description,
    visibility: form.visibility,
    status: form.status,
    mechanisms: form.mechanisms,
  });
}

const baseline = ref(serializeForm());

const isDirty = computed(() => serializeForm() !== baseline.value);

// Bot 切换时重置 baseline
watch(
  () => props.bot.id,
  () => {
    form.name = props.bot.name;
    form.description = props.bot.description;
    form.visibility = props.bot.visibility;
    form.status = props.bot.status;
    form.mechanisms = extractMechanisms(props.bot);
    baseline.value = serializeForm();
  }
);

// 计算属性：是否已有概率机制
const hasProbabilityMechanism = computed(() => {
  return form.mechanisms.some((m) => m.trigger.type === 'probability');
});

function resetForm() {
  form.name = props.bot.name;
  form.description = props.bot.description;
  form.visibility = props.bot.visibility;
  form.status = props.bot.status;
  form.mechanisms = extractMechanisms(props.bot);
  baseline.value = serializeForm();
}

function generateId(): string {
  return 'mech_' + Math.random().toString(36).slice(2, 10);
}

function addMechanism(triggerType: 'rule' | 'probability') {
  const trigger: TriggerSpec =
    triggerType === 'probability'
      ? { type: 'probability', probability: 0.1 }
      : { type: 'rule', rules: [] };

  const mechanism: Mechanism = {
    id: generateId(),
    name: triggerType === 'probability' ? '概率回复' : '新机制',
    enabled: true,
    trigger,
  };

  form.mechanisms.push(mechanism);
}

function removeMechanism(index: number) {
  form.mechanisms.splice(index, 1);
}

function updateMechanism(index: number, mechanism: Mechanism) {
  form.mechanisms[index] = deepCloneMechanism(mechanism);
}

function moveMechanism(index: number, direction: -1 | 1) {
  const newIndex = index + direction;
  if (newIndex < 0 || newIndex >= form.mechanisms.length) return;
  const temp = form.mechanisms[index]!;
  form.mechanisms.splice(index, 1);
  form.mechanisms.splice(newIndex, 0, temp);
}

function openWorkflowEditor(mechanismId: string) {
  const url = `${window.location.origin}/bots/${props.bot.id}/mechanisms/${mechanismId}/workflow`;
  window.open(url, '_blank');
}

function handleExport() {
  const mechanismConfig: MechanismConfig = {
    mechanisms: form.mechanisms.map((m) => deepCloneMechanism(m)),
  };

  const exportData = {
    version: 2,
    bot: {
      name: form.name,
      description: form.description,
      visibility: form.visibility,
    },
    mechanism_config: mechanismConfig,
  };
  const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: 'application/json' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `${form.name || 'bot'}-config.json`;
  a.click();
  URL.revokeObjectURL(url);
}

function handleImport() {
  const input = document.createElement('input');
  input.type = 'file';
  input.accept = '.json';
  input.onchange = (e) => {
    const file = (e.target as HTMLInputElement).files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = (ev) => {
      try {
        const data = JSON.parse(ev.target?.result as string);

        // v2 格式（机制列表）
        if (data.mechanism_config?.mechanisms) {
          form.mechanisms = data.mechanism_config.mechanisms.map((m: Mechanism) =>
            deepCloneMechanism(m)
          );
        }

        if (data.bot) {
          if (data.bot.name) form.name = data.bot.name;
          if (data.bot.description) form.description = data.bot.description;
        }
      } catch {
        // 无效 JSON，静默忽略
      }
    };
    reader.readAsText(file);
  };
  input.click();
}

async function handleSave() {
  saving.value = true;
  try {
    const updateData: UpdateBotRequest = {
      name: form.name,
      description: form.description,
      visibility: form.visibility,
      status: form.status,
    };

    if (props.bot.bot_type !== 'external') {
      updateData.mechanism_config = {
        mechanisms: form.mechanisms.map((m) => deepCloneMechanism(m)),
      };
    }

    emit('update', props.bot.id, updateData);

    baseline.value = serializeForm();
  } finally {
    saving.value = false;
  }
}
</script>

<style scoped>
.reset-btn-enter-active {
  transition: all 200ms cubic-bezier(0.25, 1, 0.5, 1);
}
.reset-btn-leave-active {
  transition: all 150ms cubic-bezier(0.16, 1, 0.3, 1);
}
.reset-btn-enter-from {
  opacity: 0;
  transform: translateY(16px);
}
.reset-btn-leave-to {
  opacity: 0;
  transform: translateY(8px);
}
</style>
