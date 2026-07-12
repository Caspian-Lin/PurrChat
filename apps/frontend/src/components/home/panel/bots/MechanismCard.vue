<template>
  <div class="mechanism-card" :class="{ 'mechanism-card--disabled': !localMechanism.enabled }">
    <!-- 折叠态头部 -->
    <div class="mechanism-card__header" @click="expanded = !expanded">
      <!-- 启用开关 -->
      <button
        class="mechanism-card__toggle"
        :class="{ 'mechanism-card__toggle--on': localMechanism.enabled }"
        @click.stop="toggleEnabled"
        :aria-label="localMechanism.enabled ? '禁用机制' : '启用机制'"
        :aria-pressed="localMechanism.enabled"
        :title="localMechanism.enabled ? '已启用' : '已禁用'"
      >
        <div class="mechanism-card__toggle-thumb" />
      </button>

      <!-- 摘要信息 -->
      <div class="mechanism-card__summary">
        <div class="mechanism-card__name-row">
          <input
            v-if="editingName"
            ref="nameInputRef"
            v-model="localMechanism.name"
            class="mechanism-card__name-input"
            @blur="editingName = false"
            @keydown.enter="editingName = false"
            @click.stop
          />
          <span v-else class="mechanism-card__name" @dblclick.stop="editingName = true">
            {{ localMechanism.name || '未命名机制' }}
          </span>
        </div>
        <div class="mechanism-card__types">
          <span class="mechanism-card__badge mechanism-card__badge--trigger">
            {{ triggerLabel }}
          </span>
          <span class="mechanism-card__badge mechanism-card__badge--reply">
            {{ replyLabel }}
          </span>
        </div>
      </div>

      <!-- 操作按钮 -->
      <div class="mechanism-card__actions" @click.stop>
        <!-- 上移 -->
        <button
          class="mechanism-card__action-btn"
          :disabled="!canMoveUp"
          aria-label="上移"
          @click="emit('moveUp')"
        >
          <BsChevronUp :size="14" />
        </button>
        <!-- 下移 -->
        <button
          class="mechanism-card__action-btn"
          :disabled="!canMoveDown"
          aria-label="下移"
          @click="emit('moveDown')"
        >
          <BsChevronDown :size="14" />
        </button>
        <!-- 展开/折叠 -->
        <button
          class="mechanism-card__action-btn"
          :aria-label="expanded ? '折叠' : '展开'"
          :title="expanded ? '折叠' : '展开'"
        >
          <BsChevronDown
            :size="14"
            class="transition-transform duration-200"
            :class="{ 'rotate-180': expanded }"
          />
        </button>
        <!-- 删除 -->
        <button
          class="mechanism-card__action-btn mechanism-card__action-btn--danger"
          aria-label="删除机制"
          @click="emit('delete')"
        >
          <BsTrash3 :size="13" />
        </button>
      </div>
    </div>

    <!-- 展开态编辑区 -->
    <div v-if="expanded" class="mechanism-card__body">
      <!-- 触发配置 -->
      <div class="mechanism-card__section">
        <h4 class="mechanism-card__section-title">触发规则</h4>
        <BotTriggerConfig :config="localMechanism.trigger" @update="handleTriggerUpdate" />
      </div>

      <!-- 回复配置 -->
      <div class="mechanism-card__section">
        <h4 class="mechanism-card__section-title">回复设置</h4>
        <BotReplyConfig
          :config="localMechanism.reply"
          :trigger="localMechanism.trigger"
          @update="handleReplyUpdate"
          @open-workflow-editor="emit('openWorkflowEditor', localMechanism.id)"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, watch, nextTick } from 'vue';
import { BsChevronUp, BsChevronDown, BsTrash3 } from 'vue-icons-plus/bs';
import type { Mechanism, TriggerSpec, ReplySpec } from '../../../../models/types';
import BotTriggerConfig from './BotTriggerConfig.vue';
import BotReplyConfig from './BotReplyConfig.vue';

interface Props {
  mechanism: Mechanism;
  canMoveUp?: boolean;
  canMoveDown?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  canMoveUp: false,
  canMoveDown: false,
});

const emit = defineEmits<{
  update: [mechanism: Mechanism];
  delete: [];
  moveUp: [];
  moveDown: [];
  openWorkflowEditor: [mechanismId: string];
}>();

const expanded = ref(false);
const editingName = ref(false);
const nameInputRef = ref<HTMLInputElement | null>(null);

const localMechanism = reactive<Mechanism>({
  ...props.mechanism,
  trigger: {
    ...props.mechanism.trigger,
    rules: props.mechanism.trigger?.rules?.map((r) => ({ ...r })) || [],
  },
  reply: deepCloneReply(props.mechanism.reply),
});

watch(
  () => props.mechanism,
  (newMech) => {
    localMechanism.id = newMech.id;
    localMechanism.name = newMech.name;
    localMechanism.enabled = newMech.enabled;
    localMechanism.trigger = {
      ...newMech.trigger,
      rules: newMech.trigger?.rules?.map((r) => ({ ...r })) || [],
    };
    localMechanism.reply = deepCloneReply(newMech.reply);
  },
  { deep: true }
);

watch(editingName, async (v) => {
  if (v) {
    await nextTick();
    nameInputRef.value?.select();
  } else {
    emitUpdate();
  }
});

const triggerLabel = computed(() => {
  const t = localMechanism.trigger;
  if (!t) return '未配置';
  if (t.type === 'probability') return `概率 ${Math.round((t.probability ?? 0) * 100)}%`;
  if (!t.rules?.length) return '无规则（全部触发）';
  return `规则（${t.rules.length} 条）`;
});

const replyLabel = computed(() => {
  const r = localMechanism.reply;
  if (!r) return '未配置';
  const labels: Record<string, string> = {
    predefined: '预定义',
    llm: 'LLM',
    workflow: '工作流',
  };
  return labels[r.type] || r.type;
});

function toggleEnabled() {
  localMechanism.enabled = !localMechanism.enabled;
  emitUpdate();
}

function handleTriggerUpdate(trigger: TriggerSpec) {
  localMechanism.trigger = trigger;
  emitUpdate();
}

function handleReplyUpdate(reply: ReplySpec) {
  localMechanism.reply = reply;
  emitUpdate();
}

function emitUpdate() {
  emit('update', {
    id: localMechanism.id,
    name: localMechanism.name,
    enabled: localMechanism.enabled,
    trigger: { ...localMechanism.trigger, rules: [...(localMechanism.trigger?.rules || [])] },
    reply: deepCloneReply(localMechanism.reply),
  });
}

function deepCloneReply(reply?: ReplySpec): ReplySpec {
  if (!reply) return { type: 'predefined', predefined: { mode: 'random', replies: ['...'] } };
  return {
    ...reply,
    predefined: reply.predefined
      ? { ...reply.predefined, replies: [...(reply.predefined.replies || [])] }
      : undefined,
    llm: reply.llm ? { ...reply.llm } : undefined,
    workflow: (() => {
      const wf = reply.workflow;
      return wf
        ? {
            events: wf.events.map((e) => ({ ...e, config: { ...e.config } })),
            connections: wf.connections?.map((c) => ({ ...c })) || [],
            end_conditions: wf.end_conditions.map((c) => ({ ...c })),
          }
        : undefined;
    })(),
  };
}
</script>

<style scoped>
.mechanism-card {
  border: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.08));
  border-radius: var(--radius-md, 12px);
  background: var(--surface-secondary-color, #f4f1ec);
  transition:
    opacity 0.15s,
    border-color 0.15s;
}

.mechanism-card--disabled {
  opacity: 0.55;
}

.mechanism-card__header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  cursor: pointer;
  user-select: none;
}

/* 启用开关 */
.mechanism-card__toggle {
  flex-shrink: 0;
  width: 32px;
  height: 18px;
  border-radius: 9px;
  border: none;
  background: var(--border-subtle-color, rgba(0, 0, 0, 0.15));
  cursor: pointer;
  position: relative;
  transition: background 0.2s;
  padding: 0;
}

.mechanism-card__toggle--on {
  background: var(--theme-primary, #5a8f4e);
}

.mechanism-card__toggle-thumb {
  width: 14px;
  height: 14px;
  border-radius: 7px;
  background: white;
  position: absolute;
  top: 2px;
  left: 2px;
  transition: transform 0.2s;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.12);
}

.mechanism-card__toggle--on .mechanism-card__toggle-thumb {
  transform: translateX(14px);
}

/* 摘要 */
.mechanism-card__summary {
  flex: 1;
  min-width: 0;
}

.mechanism-card__name-row {
  margin-bottom: 2px;
}

.mechanism-card__name {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-color, #1c1917);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mechanism-card__name-input {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-color, #1c1917);
  background: var(--input-background, #fff);
  border: 1px solid var(--theme-primary, #5a8f4e);
  border-radius: var(--radius-xs, 4px);
  padding: 1px 6px;
  outline: none;
  width: 100%;
  box-sizing: border-box;
}

.mechanism-card__types {
  display: flex;
  gap: 6px;
}

.mechanism-card__badge {
  font-size: 11px;
  padding: 1px 6px;
  border-radius: 4px;
  line-height: 1.5;
}

.mechanism-card__badge--trigger {
  background: rgba(59, 130, 246, 0.08);
  color: #3b82f6;
}

.mechanism-card__badge--reply {
  background: rgba(90, 143, 78, 0.08);
  color: var(--theme-primary, #5a8f4e);
}

/* 操作按钮 */
.mechanism-card__actions {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
}

.mechanism-card__action-btn {
  padding: 4px;
  border-radius: var(--radius-xs, 4px);
  border: none;
  background: none;
  color: var(--text-tertiary-color, #a8a29e);
  cursor: pointer;
  transition: all 0.15s;
  display: flex;
  align-items: center;
  justify-content: center;
}

.mechanism-card__action-btn:hover:not(:disabled) {
  background: var(--hover-background, rgba(0, 0, 0, 0.04));
  color: var(--text-secondary-color, #57534e);
}

.mechanism-card__action-btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

.mechanism-card__action-btn--danger:hover:not(:disabled) {
  background: rgba(239, 68, 68, 0.08);
  color: #ef4444;
}

/* 展开编辑区 */
.mechanism-card__body {
  padding: 0 12px 12px;
  border-top: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.06));
  margin-top: 0;
  padding-top: 12px;
}

.mechanism-card__section {
  margin-bottom: 12px;
}

.mechanism-card__section:last-child {
  margin-bottom: 0;
}

.mechanism-card__section-title {
  font-size: 12px;
  font-weight: 500;
  color: var(--text-secondary-color, #57534e);
  margin-bottom: 8px;
}
</style>
