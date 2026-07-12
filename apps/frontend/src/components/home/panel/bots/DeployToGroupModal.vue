<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4" @click.self="$emit('close')">
    <div class="absolute inset-0 bg-black/30" aria-hidden="true" />

    <section
      class="relative w-full max-w-lg overflow-hidden rounded-[var(--radius-lg,16px)] bg-bg-primary shadow-lg"
      role="dialog"
      aria-modal="true"
      aria-labelledby="group-install-title"
    >
      <header class="flex items-center justify-between border-b border-border-subtle px-6 py-4">
        <div class="flex min-w-0 items-center gap-2">
          <button
            v-if="mode === 'authorize'"
            class="rounded-lg p-1.5 text-text-tertiary transition-colors hover:bg-hover-bg hover:text-text-primary"
            aria-label="返回群聊列表"
            :disabled="submitting"
            @click="backToList"
          >
            <BsArrowLeft :size="17" />
          </button>
          <div class="min-w-0">
            <h2 id="group-install-title" class="truncate text-base font-semibold text-text-primary">
              {{
                mode === 'list' ? '安装到群聊' : editingInstallation ? '管理 Bot 权限' : '授权安装'
              }}
            </h2>
            <p class="mt-0.5 truncate text-xs text-text-tertiary">
              {{ mode === 'list' ? bot.name : selectedTarget?.name }}
            </p>
          </div>
        </div>
        <button
          class="rounded-lg p-1.5 text-text-tertiary transition-colors hover:bg-hover-bg hover:text-text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--theme-primary)]"
          aria-label="关闭"
          :disabled="submitting"
          @click="$emit('close')"
        >
          <BsX :size="18" />
        </button>
      </header>

      <div class="max-h-[65vh] overflow-y-auto px-6 py-5">
        <div v-if="loading" class="space-y-3" aria-label="正在加载群聊">
          <div
            v-for="index in 3"
            :key="index"
            class="h-14 animate-pulse rounded-xl bg-bg-secondary"
          />
        </div>

        <template v-else-if="mode === 'list'">
          <section v-if="installedGroups.length" class="mb-5">
            <h3 class="mb-2 text-xs font-medium text-text-tertiary">已安装</h3>
            <div class="space-y-2">
              <div
                v-for="installation in installedGroups"
                :key="installation.id"
                class="flex items-center gap-3 rounded-[var(--radius-md,12px)] bg-bg-secondary px-3.5 py-3"
              >
                <div class="min-w-0 flex-1">
                  <p class="truncate text-sm font-medium text-text-primary">
                    {{ installation.target_name || installation.target_id }}
                  </p>
                  <p class="mt-0.5 text-xs text-text-tertiary">
                    已授权 {{ installation.granted_capabilities.length }} 项能力
                  </p>
                </div>
                <button
                  class="rounded-lg px-2.5 py-1.5 text-xs text-text-secondary transition-colors hover:bg-hover-bg hover:text-text-primary"
                  @click="manageInstallation(installation)"
                >
                  管理权限
                </button>
                <button
                  class="rounded-lg p-1.5 text-text-tertiary transition-colors hover:bg-red-500/10 hover:text-red-500 disabled:opacity-50"
                  aria-label="从群聊移除 Bot"
                  title="卸载"
                  :disabled="uninstalling === installation.id"
                  @click="handleUninstall(installation)"
                >
                  <span
                    v-if="uninstalling === installation.id"
                    class="block h-3.5 w-3.5 animate-spin rounded-full border-2 border-text-quaternary border-t-[var(--theme-primary)]"
                  />
                  <BsTrash v-else :size="14" />
                </button>
              </div>
            </div>
          </section>

          <section v-if="conversations.length">
            <h3 v-if="installedGroups.length" class="mb-2 text-xs font-medium text-text-tertiary">
              选择要安装的群聊
            </h3>
            <div class="space-y-1">
              <button
                v-for="conversation in conversations"
                :key="conversation.id"
                class="flex w-full items-center gap-3 rounded-[var(--radius-md,12px)] px-3 py-2.5 text-left transition-colors hover:bg-hover-bg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--theme-primary)]"
                @click="authorizeConversation(conversation)"
              >
                <div
                  class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-xl bg-[var(--theme-primary)] text-xs font-bold text-white"
                >
                  {{ conversation.name.charAt(0) }}
                </div>
                <span class="min-w-0 flex-1">
                  <span class="block truncate text-sm font-medium text-text-primary">
                    {{ conversation.name }}
                  </span>
                  <span class="block text-xs text-text-tertiary">
                    {{ conversation.member_count }} 位成员
                  </span>
                </span>
                <BsChevronRight :size="15" class="flex-shrink-0 text-text-quaternary" />
              </button>
            </div>
          </section>

          <div v-else-if="!installedGroups.length" class="py-8 text-center">
            <BsPeopleFill :size="32" class="mx-auto mb-3 text-text-quaternary" />
            <p class="text-sm text-text-secondary">没有可安装的群聊</p>
            <p class="mt-1 text-xs text-text-tertiary">只有群主或管理员可以安装 Bot</p>
          </div>
        </template>

        <BotPermissionReview
          v-else-if="selectedTarget"
          :key="editingInstallation?.id ?? selectedTarget.id"
          :bot-name="bot.name"
          :target-label="selectedTarget.name"
          :requested-capabilities="bot.requested_capabilities ?? []"
          :initial-capabilities="editingInstallation?.granted_capabilities"
          @change="handlePermissionChange"
        />
      </div>

      <footer class="flex items-center justify-end gap-3 border-t border-border-subtle px-6 py-4">
        <template v-if="mode === 'authorize'">
          <button
            class="rounded-[var(--radius-sm,8px)] px-4 py-2 text-sm text-text-secondary transition-colors hover:bg-hover-bg"
            :disabled="submitting"
            @click="backToList"
          >
            返回
          </button>
          <button
            class="inline-flex min-w-24 items-center justify-center gap-2 rounded-[var(--radius-sm,8px)] bg-[var(--theme-primary)] px-4 py-2 text-sm font-medium text-white transition-opacity disabled:cursor-not-allowed disabled:opacity-50"
            :disabled="submitting || !canConfirm"
            @click="confirmAuthorization"
          >
            <span
              v-if="submitting"
              class="h-4 w-4 animate-spin rounded-full border-2 border-white/40 border-t-white"
            />
            {{ submitting ? '保存中' : editingInstallation ? '保存权限' : '授权并安装' }}
          </button>
        </template>
        <button
          v-else
          class="rounded-[var(--radius-sm,8px)] bg-bg-quaternary px-4 py-2 text-sm text-text-secondary transition-colors hover:bg-hover-bg"
          @click="$emit('close')"
        >
          完成
        </button>
      </footer>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { BsArrowLeft, BsChevronRight, BsPeopleFill, BsTrash, BsX } from 'vue-icons-plus/bs';
import { useBots } from '../../../../composables/useBots';
import { useBotStore } from '../../../../stores/bot';
import type { Bot, BotDeployment, DeployableConversation } from '../../../../models/types';
import BotPermissionReview from './BotPermissionReview.vue';

const props = defineProps<{ bot: Bot }>();
const emit = defineEmits<{
  deployed: [conversationId: string];
  close: [];
}>();

const { getDeployableConversations, installBot, updateInstallation, uninstallInstallation } =
  useBots();
const botStore = useBotStore();
const loading = ref(true);
const conversations = ref<DeployableConversation[]>([]);
const mode = ref<'list' | 'authorize'>('list');
const selectedTarget = ref<DeployableConversation | null>(null);
const editingInstallation = ref<BotDeployment | null>(null);
const selectedCapabilities = ref<string[]>([]);
const canConfirm = ref(false);
const submitting = ref(false);
const uninstalling = ref<string | null>(null);

const installedGroups = computed(() =>
  botStore.deployments.filter(
    (deployment) => deployment.app_id === props.bot.id && deployment.target_type === 'conversation'
  )
);

onMounted(loadData);

async function loadData() {
  loading.value = true;
  try {
    await botStore.loadDeployments();
    conversations.value = await getDeployableConversations(props.bot.id);
  } finally {
    loading.value = false;
  }
}

function authorizeConversation(conversation: DeployableConversation) {
  selectedTarget.value = conversation;
  editingInstallation.value = null;
  mode.value = 'authorize';
}

function manageInstallation(installation: BotDeployment) {
  selectedTarget.value = {
    id: installation.target_id,
    name: installation.target_name || installation.target_id,
    conversation_type: 'group',
    member_count: 0,
  };
  editingInstallation.value = installation;
  mode.value = 'authorize';
}

function backToList() {
  if (submitting.value) return;
  mode.value = 'list';
  selectedTarget.value = null;
  editingInstallation.value = null;
  selectedCapabilities.value = [];
  canConfirm.value = false;
}

function handlePermissionChange(capabilities: string[], allowed: boolean) {
  selectedCapabilities.value = capabilities;
  canConfirm.value = allowed;
}

async function confirmAuthorization() {
  if (!selectedTarget.value || !canConfirm.value) return;
  submitting.value = true;

  const diagnosticsConsent = selectedCapabilities.value.includes('network:external')
    ? 'granted'
    : 'denied';
  const installation = editingInstallation.value
    ? await updateInstallation(editingInstallation.value.id, {
        status: 'active',
        granted_capabilities: selectedCapabilities.value,
        diagnostics_consent: diagnosticsConsent,
      })
    : await installBot(props.bot.id, {
        target_type: 'conversation',
        target_id: selectedTarget.value.id,
        granted_capabilities: selectedCapabilities.value,
        diagnostics_consent: diagnosticsConsent,
      });

  submitting.value = false;
  if (!installation) return;

  if (!editingInstallation.value) {
    conversations.value = conversations.value.filter(
      (conversation) => conversation.id !== selectedTarget.value?.id
    );
    emit('deployed', installation.target_id);
  }
  backToList();
}

async function handleUninstall(installation: BotDeployment) {
  uninstalling.value = installation.id;
  const removed = await uninstallInstallation(installation.id);
  uninstalling.value = null;
  if (removed) conversations.value = await getDeployableConversations(props.bot.id);
}
</script>
