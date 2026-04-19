<template>
  <div class="flex flex-col h-screen bg-bg-primary">
    <!-- 顶部工具栏 -->
    <div
      class="flex items-center gap-3 px-5 py-3 bg-bg-secondary border-b border-border-subtle flex-shrink-0"
    >
      <button
        class="p-1.5 rounded-lg hover:bg-hover-bg text-text-tertiary hover:text-text-primary transition-colors"
        title="返回"
        @click="goBack"
      >
        <BsArrowLeft :size="18" />
      </button>
      <div class="flex-1 min-w-0">
        <h2 class="text-sm font-medium text-text-primary truncate">
          {{ botName }}
          <span class="text-text-tertiary mx-1.5">/</span>
          {{ mechanismName }}
        </h2>
        <p class="text-xs text-text-tertiary truncate">特殊模式事件链编辑器</p>
      </div>
      <!-- 自动保存提示 -->
      <span v-if="saveState === 'saving'" class="text-xs text-text-tertiary">保存中...</span>
      <span v-else-if="saveState === 'saved'" class="text-xs text-green-600">已保存</span>
      <span v-else-if="saveState === 'error'" class="text-xs text-red-500">保存失败</span>
      <button
        class="px-4 py-1.5 text-xs rounded-[var(--radius-sm,8px)] text-white transition-colors"
        style="background: var(--theme-primary)"
        :disabled="saveState === 'saving'"
        @click="handleSave"
      >
        保存
      </button>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="flex-1 flex items-center justify-center">
      <p class="text-sm text-text-tertiary">加载中...</p>
    </div>

    <!-- 错误状态 -->
    <div v-else-if="error" class="flex-1 flex items-center justify-center">
      <div class="text-center">
        <p class="text-sm text-red-500 mb-2">{{ error }}</p>
        <button
          class="text-xs text-text-secondary hover:text-text-primary underline"
          @click="loadData"
        >
          重试
        </button>
      </div>
    </div>

    <!-- 编辑器主体 -->
    <template v-else>
      <!-- VueFlow DAG 编辑器 -->
      <div class="flex-1 relative overflow-hidden">
        <div class="editor-toolbar absolute top-3 left-3 z-10 flex gap-2">
          <button class="toolbar-btn" @click="showAddModal = true">
            <BsPlus :size="14" />
            添加事件
          </button>
        </div>

        <VueFlow
          v-model:nodes="flowNodes"
          v-model:edges="flowEdges"
          :node-types="customNodeTypes"
          :default-edge-options="defaultEdgeOptions"
          fit-view-on-init
          :min-zoom="0.3"
          :max-zoom="2"
          class="w-full h-full"
          @node-click="onNodeClick"
        >
          <Background :gap="20" :size="1" />
          <Controls />
        </VueFlow>
      </div>

      <!-- 底部面板：结束条件 + 调试 -->
      <div class="flex-shrink-0 border-t border-border-subtle bg-bg-secondary">
        <!-- Tab 切换 -->
        <div class="flex border-b border-border-subtle">
          <button
            class="bottom-tab"
            :class="{ 'bottom-tab--active': activeBottomTab === 'conditions' }"
            @click="activeBottomTab = 'conditions'"
          >
            结束条件
          </button>
          <button
            class="bottom-tab"
            :class="{ 'bottom-tab--active': activeBottomTab === 'debug' }"
            @click="activeBottomTab = 'debug'"
          >
            调试
          </button>
        </div>

        <!-- 结束条件面板 -->
        <div v-if="activeBottomTab === 'conditions'" class="p-4">
          <EndConditionConfig :conditions="endConditions" @update="handleEndConditionsUpdate" />
        </div>

        <!-- 调试面板 -->
        <div v-if="activeBottomTab === 'debug'" class="p-4">
          <BotDebugPanel
            v-if="localMechanism"
            :bot-id="botId"
            :mechanism="localMechanism"
            :bot-name="botName"
          />
          <p v-else class="text-xs text-text-quaternary">无法加载调试面板</p>
        </div>
      </div>
    </template>

    <!-- 事件配置弹窗 -->
    <EventConfigModal
      :visible="showAddModal || !!editingEvent"
      :editing-event="editingEvent"
      :existing-events="events"
      @close="closeModal"
      @confirm="handleEventConfirm"
      @delete="handleEventDelete"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, markRaw } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { VueFlow } from '@vue-flow/core';
import { Background } from '@vue-flow/background';
import { Controls } from '@vue-flow/controls';
import '@vue-flow/core/dist/style.css';
import '@vue-flow/core/dist/theme-default.css';
import '@vue-flow/controls/dist/style.css';
import { BsArrowLeft, BsPlus } from 'vue-icons-plus/bs';
import { api } from '../models/api';
import type { Bot, Mechanism, SpecialModeEvent, SpecialModeEndCondition } from '../models/types';
import type { Node, Edge } from '@vue-flow/core';
import { eventsToFlowNodes, eventsToFlowEdges } from '../utils/eventFlowUtils';
import EventNode from '../components/home/panel/bots/events/EventNode.vue';
import EventConfigModal from '../components/home/panel/bots/events/EventConfigModal.vue';
import EndConditionConfig from '../components/home/panel/bots/events/EndConditionConfig.vue';
import BotDebugPanel from '../components/home/panel/bots/BotDebugPanel.vue';

const route = useRoute();
const router = useRouter();

const botId = route.params.botId as string;
const mechanismId = route.params.mechanismId as string;

// 状态
const loading = ref(true);
const error = ref<string | null>(null);
const saveState = ref<'idle' | 'saving' | 'saved' | 'error'>('idle');
const activeBottomTab = ref<'conditions' | 'debug'>('conditions');
const showAddModal = ref(false);
const editingEvent = ref<SpecialModeEvent | null>(null);

// 数据
const bot = ref<Bot | null>(null);
const localMechanism = ref<Mechanism | null>(null);

const botName = computed(() => bot.value?.name || 'Bot');
const mechanismName = computed(() => localMechanism.value?.name || '机制');

const events = computed<SpecialModeEvent[]>(() => {
  return localMechanism.value?.reply?.special_mode?.events || [];
});

const endConditions = computed<SpecialModeEndCondition[]>(() => {
  return localMechanism.value?.reply?.special_mode?.end_conditions || [];
});

// VueFlow 注册
const customNodeTypes = {
  event: markRaw(EventNode),
};

const defaultEdgeOptions = {
  type: 'smoothstep',
  animated: true,
  style: { stroke: 'var(--theme-primary, #5a8f4e)', strokeWidth: 2 },
};

const nodePositions: Record<string, { x: number; y: number }> = {};

const flowNodes = computed<Node[]>({
  get() {
    return eventsToFlowNodes(events.value, nodePositions);
  },
  set() {},
});

const flowEdges = computed<Edge[]>({
  get() {
    return eventsToFlowEdges(events.value);
  },
  set() {},
});

// BroadcastChannel 用于主 Tab 同步
let channel: BroadcastChannel | null = null;
try {
  channel = new BroadcastChannel('purrchat-bot-editor');
} catch {
  // BroadcastChannel 不可用时忽略
}

onMounted(async () => {
  await loadData();

  // 监听来自其他 Tab 的数据变更通知
  channel?.addEventListener('message', (e) => {
    if (e.data.type === 'bot-updated' && e.data.botId === botId) {
      loadData();
    }
  });
});

async function loadData() {
  loading.value = true;
  error.value = null;

  try {
    const result = await api.getBot(botId);
    if (!result.success || !result.data) {
      error.value = 'Bot 不存在或加载失败';
      return;
    }

    bot.value = result.data;

    // 从 mechanism_config 中找到对应的机制
    const mechanisms = result.data.mechanism_config?.mechanisms || [];
    const found = mechanisms.find((m) => m.id === mechanismId);

    if (!found) {
      error.value = `未找到机制 ${mechanismId}`;
      return;
    }

    // 深拷贝一份用于本地编辑
    localMechanism.value = deepCloneMechanism(found);
  } catch (err: any) {
    error.value = err.response?.data?.message || '加载失败';
  } finally {
    loading.value = false;
  }
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

async function handleSave() {
  if (!localMechanism.value) return;
  saveState.value = 'saving';

  try {
    // 构建更新后的 mechanism_config
    const mechanisms = bot.value?.mechanism_config?.mechanisms?.map((m) =>
      m.id === mechanismId ? deepCloneMechanism(localMechanism.value!) : m
    ) || [deepCloneMechanism(localMechanism.value!)];

    const result = await api.updateBot(botId, {
      mechanism_config: { mechanisms },
    });

    if (result.success && result.data) {
      bot.value = result.data;
      // 重新从更新后的数据中找到机制
      const found = result.data.mechanism_config?.mechanisms?.find((m) => m.id === mechanismId);
      if (found) {
        localMechanism.value = deepCloneMechanism(found);
      }
      saveState.value = 'saved';
      // 通知主 Tab 数据已更新
      channel?.postMessage({ type: 'bot-updated', botId });
      setTimeout(() => {
        if (saveState.value === 'saved') saveState.value = 'idle';
      }, 2000);
    } else {
      saveState.value = 'error';
    }
  } catch {
    saveState.value = 'error';
  }
}

function goBack() {
  router.back();
}

function onNodeClick({ node }: { node: Node }) {
  const evt = events.value.find((e) => e.id === node.id);
  if (evt) {
    editingEvent.value = { ...evt };
  }
}

function closeModal() {
  showAddModal.value = false;
  editingEvent.value = null;
}

function handleEventConfirm(event: SpecialModeEvent) {
  if (!localMechanism.value?.reply?.special_mode) return;

  const currentEvents = [...events.value];
  const existingIndex = currentEvents.findIndex((e) => e.id === event.id);

  if (existingIndex >= 0) {
    currentEvents[existingIndex] = event;
  } else {
    currentEvents.push(event);
  }

  localMechanism.value.reply.special_mode.events = currentEvents;
  closeModal();
}

function handleEventDelete(eventId: string) {
  if (!localMechanism.value?.reply?.special_mode) return;

  const updatedEvents = events.value
    .filter((e) => e.id !== eventId)
    .map((e) => ({
      ...e,
      next: (e.next || []).filter((n) => n !== eventId),
    }));

  localMechanism.value.reply.special_mode.events = updatedEvents;
  closeModal();
}

function handleEndConditionsUpdate(conditions: SpecialModeEndCondition[]) {
  if (!localMechanism.value?.reply?.special_mode) return;
  localMechanism.value.reply.special_mode.end_conditions = conditions;
}

// 监听 node position 变化并缓存
watch(flowNodes, (nodes) => {
  for (const node of nodes) {
    nodePositions[node.id] = { ...node.position };
  }
});
</script>

<style scoped>
.editor-toolbar {
  display: flex;
  gap: 8px;
}

.toolbar-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  font-size: 12px;
  border-radius: var(--radius-xs, 4px);
  border: 1px solid var(--border-subtle, rgba(0, 0, 0, 0.1));
  background: var(--bg-quaternary, #f8f7f5);
  color: var(--text-secondary, #666);
  cursor: pointer;
  transition: all 0.15s;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
}

.toolbar-btn:hover {
  border-color: var(--theme-primary, #5a8f4e);
  color: var(--theme-primary, #5a8f4e);
}

.bottom-tab {
  padding: 8px 16px;
  font-size: 12px;
  border: none;
  border-bottom: 2px solid transparent;
  background: none;
  color: var(--text-tertiary, #999);
  cursor: pointer;
  transition: all 0.15s;
}

.bottom-tab:hover {
  color: var(--text-secondary, #666);
}

.bottom-tab--active {
  color: var(--text-primary, #1a1a1a);
  border-bottom-color: var(--theme-primary, #5a8f4e);
}
</style>
