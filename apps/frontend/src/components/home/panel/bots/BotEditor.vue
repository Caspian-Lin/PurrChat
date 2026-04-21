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
      <!-- 导出/导入按钮 -->
      <button
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

    <!-- 编辑器内容 -->
    <div class="flex-1 overflow-y-auto">
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

        <!-- 机制列表 -->
        <section>
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
              @open-special-mode-editor="openSpecialModeEditor"
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

        <!-- 调试面板（仅当有特殊模式机制时显示） -->
        <section v-if="specialModeMechanism">
          <h3 class="text-sm font-semibold text-text-primary mb-4 flex items-center gap-2">
            <BsBug :size="16" class="text-text-tertiary" />
            调试 — {{ specialModeMechanism.name }}
          </h3>
          <BotDebugPanel :bot-id="bot.id" :mechanism="specialModeMechanism" :bot-name="form.name" />
        </section>

        <!-- 保存按钮 -->
        <div class="flex justify-end gap-3 pb-6">
          <button
            class="px-4 py-2 text-sm rounded-[var(--radius-sm,8px)] bg-bg-quaternary text-text-secondary hover:bg-hover-bg transition-colors"
            @click="resetForm"
          >
            重置
          </button>
          <button
            class="px-4 py-2 text-sm rounded-[var(--radius-sm,8px)] text-white transition-colors disabled:opacity-50"
            style="background: var(--theme-primary)"
            :disabled="saving"
            @click="handleSave"
          >
            {{ saving ? '保存中...' : '保存更改' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, computed } from 'vue';
import {
  BsArrowLeft,
  BsInfoCircle,
  BsGear,
  BsBug,
  BsUpload,
  BsDownload,
  BsPlus,
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
import BotDebugPanel from './BotDebugPanel.vue';

interface Props {
  bot: Bot;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  update: [botId: string, data: UpdateBotRequest];
  back: [];
}>();

const saving = ref(false);

const visibilityOptions = [
  { value: 'private' as BotVisibility, label: '私有' },
  { value: 'public' as BotVisibility, label: '公开' },
  { value: 'global' as BotVisibility, label: '系统' },
];

// 从 Bot 数据中提取机制列表（兼容旧格式）
function extractMechanisms(bot: Bot): Mechanism[] {
  // 优先使用新的 mechanism_config
  if (bot.mechanism_config?.mechanisms?.length) {
    return bot.mechanism_config.mechanisms.map((m) => deepCloneMechanism(m));
  }

  // 兼容旧格式：合并 trigger_config + reply_config 为一个默认机制
  const mechanisms: Mechanism[] = [];

  if (bot.trigger_config || bot.reply_config) {
    mechanisms.push({
      id: 'mech_default',
      name: '默认机制',
      enabled: true,
      trigger: (bot.trigger_config as any)
        ? {
            type: (bot.trigger_config as any).mode === 'probability' ? 'probability' : 'rule',
            rules: (bot.trigger_config as any).rules,
            probability: (bot.trigger_config as any).probability,
          }
        : { type: 'rule', rules: [] },
      reply: (bot.reply_config as any) || {
        type: 'predefined',
        predefined: { mode: 'random', replies: ['...'] },
      },
    });
  }

  // 如果有 special_mode_config，创建第二个特殊模式机制
  if (bot.special_mode_config && (bot.special_mode_config as any).events?.length) {
    mechanisms.push({
      id: 'mech_special',
      name: '特殊模式',
      enabled: true,
      trigger: { type: 'rule', rules: [] },
      reply: {
        type: 'special_mode',
        special_mode: bot.special_mode_config as any,
      },
    });
  }

  // 如果没有任何机制，创建一个默认的空规则机制
  if (mechanisms.length === 0) {
    mechanisms.push({
      id: 'mech_default',
      name: '默认机制',
      enabled: true,
      trigger: { type: 'rule', rules: [] },
      reply: { type: 'predefined', predefined: { mode: 'random', replies: ['...'] } },
    });
  }

  return mechanisms;
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
    reply: {
      ...m.reply,
      predefined: m.reply.predefined
        ? { ...m.reply.predefined, replies: [...(m.reply.predefined.replies || [])] }
        : undefined,
      llm: m.reply.llm ? { ...m.reply.llm } : undefined,
      special_mode: m.reply.special_mode
        ? {
            events: m.reply.special_mode.events.map((e) => ({ ...e, config: { ...e.config } })),
            end_conditions: m.reply.special_mode.end_conditions.map((c) => ({ ...c })),
          }
        : undefined,
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

// 计算属性：是否已有概率机制
const hasProbabilityMechanism = computed(() => {
  return form.mechanisms.some((m) => m.trigger.type === 'probability');
});

// 计算属性：找到特殊模式机制（调试面板用）
const specialModeMechanism = computed<Mechanism | null>(() => {
  return form.mechanisms.find((m) => m.reply.type === 'special_mode') || null;
});

function resetForm() {
  form.name = props.bot.name;
  form.description = props.bot.description;
  form.visibility = props.bot.visibility;
  form.status = props.bot.status;
  form.mechanisms = extractMechanisms(props.bot);
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
    reply: { type: 'predefined', predefined: { mode: 'random', replies: ['...'] } },
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

function openSpecialModeEditor(mechanismId: string) {
  // 阶段 5 实现完整的路由跳转
  const url = `${window.location.origin}/bots/${props.bot.id}/mechanisms/${mechanismId}/special-mode`;
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

        // v2 格式（新机制列表）
        if (data.mechanism_config?.mechanisms) {
          form.mechanisms = data.mechanism_config.mechanisms.map((m: Mechanism) =>
            deepCloneMechanism(m)
          );
        }

        // 兼容 v1 格式（旧三字段）
        if (data.trigger_config && !data.mechanism_config) {
          form.mechanisms = extractMechanisms({
            ...props.bot,
            trigger_config: data.trigger_config,
            reply_config: data.reply_config,
            special_mode_config: data.special_mode_config,
          } as Bot);
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
    const mechanismConfig: MechanismConfig = {
      mechanisms: form.mechanisms.map((m) => deepCloneMechanism(m)),
    };

    emit('update', props.bot.id, {
      name: form.name,
      description: form.description,
      visibility: form.visibility,
      status: form.status,
      mechanism_config: mechanismConfig,
    });
  } finally {
    saving.value = false;
  }
}
</script>
