<template>
  <div>
    <div class="flex items-center justify-between mb-2">
      <label class="text-sm font-medium" style="color: var(--text-color)"> 已安装 Bot </label>
      <button
        v-if="canManage"
        class="px-3 py-1.5 text-sm bg-accent-color text-white rounded-md hover:opacity-80 transition-colors"
        @click="openBotPicker"
      >
        添加 Bot
      </button>
    </div>
    <div class="max-h-48 rounded-lg overflow-y-auto" style="background: var(--surface-color)">
      <div v-if="loading" class="px-4 py-6 text-center text-sm text-text-tertiary">加载中…</div>
      <div v-else-if="bots.length === 0" class="px-4 py-6 text-center text-sm text-text-tertiary">
        该会话还没有安装 Bot
      </div>
      <div v-else class="px-2 pt-2 pb-0.5">
        <BaseListItem v-for="dep in bots" :key="dep.id">
          <template #avatar>
            <div class="w-10 h-10 rounded-[var(--radius-md)] overflow-hidden">
              <img
                v-if="dep.app?.avatar_url"
                :src="dep.app.avatar_url"
                alt="avatar"
                class="w-full h-full object-cover"
              />
              <div
                v-else
                class="w-full h-full flex items-center justify-center text-white"
                style="background: var(--theme-primary)"
              >
                <BsCpu :size="20" />
              </div>
            </div>
          </template>
          <div class="flex flex-col gap-0.5 min-w-0">
            <div class="flex items-center gap-2">
              <span class="font-medium text-text-primary truncate">{{
                dep.app?.name || 'Bot'
              }}</span>
              <span
                v-if="dep.status === 'paused'"
                class="text-[11px] px-1.5 py-0.5 rounded-full bg-amber-500/10 text-amber-600"
              >
                已暂停
              </span>
            </div>
            <span class="text-xs text-text-tertiary"
              >{{ dep.granted_capabilities.length }} 项权限</span
            >
          </div>
          <template v-if="canManage" #actions>
            <div class="flex items-center gap-1.5">
              <button
                class="px-2.5 py-1 text-xs rounded-[var(--radius-sm)] bg-bg-quaternary text-text-secondary hover:bg-hover-bg transition-colors"
                @click.stop="handleTogglePause(dep)"
              >
                {{ dep.status === 'paused' ? '恢复' : '暂停' }}
              </button>
              <button
                class="px-2.5 py-1 text-xs rounded-[var(--radius-sm)] bg-[var(--theme-primary)]/10 text-[var(--theme-primary)] hover:bg-[var(--theme-primary)]/20 transition-colors"
                @click.stop="openAdjustPermissions(dep)"
              >
                权限
              </button>
              <button
                class="px-2.5 py-1 text-xs bg-red-500 text-white rounded-[var(--radius-sm)] hover:bg-red-600 transition-colors"
                @click.stop="handleUninstall(dep)"
              >
                移除
              </button>
            </div>
          </template>
        </BaseListItem>
      </div>
    </div>

    <Teleport to="body">
      <div
        v-if="showPicker"
        class="fixed inset-0 z-50 flex items-center justify-center p-4"
        @click.self="showPicker = false"
      >
        <div class="absolute inset-0 bg-black/30" aria-hidden="true" />
        <section
          class="relative w-full max-w-md overflow-hidden rounded-[var(--radius-lg)] bg-bg-primary shadow-lg"
          role="dialog"
          aria-modal="true"
        >
          <header class="flex items-center justify-between border-b border-border-subtle px-6 py-4">
            <h2 class="text-base font-semibold text-text-primary">选择 Bot 安装到群聊</h2>
            <button
              class="rounded-lg p-1.5 text-text-tertiary transition-colors hover:bg-hover-bg hover:text-text-primary"
              @click="showPicker = false"
            >
              <BsX :size="18" />
            </button>
          </header>
          <div class="max-h-[50vh] overflow-y-auto px-3 py-2">
            <div
              v-if="availableBots.length === 0"
              class="px-4 py-6 text-center text-sm text-text-tertiary"
            >
              没有可安装的 Bot
            </div>
            <BaseListItem v-for="bot in availableBots" :key="bot.id" @click="selectBot(bot)">
              <template #avatar>
                <div class="w-10 h-10 rounded-[var(--radius-md)] overflow-hidden">
                  <img
                    v-if="bot.avatar_url"
                    :src="bot.avatar_url"
                    alt="avatar"
                    class="w-full h-full object-cover"
                  />
                  <div
                    v-else
                    class="w-full h-full flex items-center justify-center text-white"
                    style="background: var(--theme-primary)"
                  >
                    <BsCpu :size="20" />
                  </div>
                </div>
              </template>
              <div class="flex flex-col gap-0.5 min-w-0">
                <span class="font-medium text-text-primary truncate">{{ bot.name }}</span>
                <span class="text-xs text-text-tertiary">
                  {{ (bot.requested_capabilities ?? []).length }} 项请求权限
                </span>
              </div>
            </BaseListItem>
          </div>
        </section>
      </div>

      <InstallBotModal
        v-if="installTarget"
        :bot="installTarget"
        :installation="adjustTarget"
        target-type="conversation"
        :target-id="conversationId"
        :target-label="conversationName"
        @installed="handleInstalled"
        @close="closeInstallModal"
      />
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue';
import { BsCpu, BsX } from 'vue-icons-plus/bs';
import BaseListItem from '../common/BaseListItem.vue';
import InstallBotModal from './panel/bots/InstallBotModal.vue';
import { api } from '../../models/api';
import { useBots } from '../../composables/useBots';
import { useBotStore } from '../../stores/bot';
import { websocketEventManager } from '../../services/websocketEventManager';
import type { Bot, BotDeployment } from '../../models/types';

const props = defineProps<{
  conversationId: string;
  conversationName: string;
  canManage: boolean;
}>();

const emit = defineEmits<{
  'bots-changed': [];
}>();

const { updateInstallation, uninstallInstallation } = useBots();
const botStore = useBotStore();

const bots = ref<BotDeployment[]>([]);
const loading = ref(false);
const showPicker = ref(false);
const installTarget = ref<Bot | null>(null);
const adjustTarget = ref<BotDeployment | null>(null);

const availableBots = computed(() => {
  const installedIds = new Set(bots.value.map((d) => d.app_id));
  return botStore.bots.filter((b) => !installedIds.has(b.id));
});

async function loadBots() {
  if (!props.conversationId) return;
  loading.value = true;
  try {
    const res = await api.getConversationBots(props.conversationId);
    bots.value = res.success && res.data ? res.data : [];
  } catch {
    bots.value = [];
  } finally {
    loading.value = false;
  }
}

function openBotPicker() {
  if (botStore.bots.length === 0) botStore.loadBots();
  showPicker.value = true;
}

function selectBot(bot: Bot) {
  showPicker.value = false;
  adjustTarget.value = null;
  installTarget.value = bot;
}

function openAdjustPermissions(dep: BotDeployment) {
  adjustTarget.value = dep;
  installTarget.value = dep.app ?? null;
}

function closeInstallModal() {
  installTarget.value = null;
  adjustTarget.value = null;
}

async function handleInstalled() {
  closeInstallModal();
  await loadBots();
  emit('bots-changed');
}

async function handleTogglePause(dep: BotDeployment) {
  const newStatus = dep.status === 'paused' ? 'active' : 'paused';
  const result = await updateInstallation(dep.id, { status: newStatus });
  if (result) await loadBots();
}

async function handleUninstall(dep: BotDeployment) {
  const name = dep.app?.name || '该 Bot';
  if (!confirm(`确定要从群聊移除 ${name} 吗？`)) return;
  const ok = await uninstallInstallation(dep.id);
  if (ok) {
    await loadBots();
    emit('bots-changed');
  }
}

let offBotDeployment: (() => void) | null = null;

onMounted(() => {
  loadBots();
  offBotDeployment = websocketEventManager.onBotDeploymentChange((_event, data) => {
    if (data.conversation_id === props.conversationId) loadBots();
  });
});

onUnmounted(() => {
  offBotDeployment?.();
});

watch(() => props.conversationId, loadBots);
</script>
