<template>
  <BasePanel
    panel-id="bots"
    :initial-sidebar-width="300"
    :min-sidebar-width="220"
    :max-sidebar-width="420"
  >
    <template #sidebar>
      <div class="flex flex-col h-full">
        <!-- 顶部操作栏 -->
        <div
          class="flex items-center gap-2 px-3 pt-5 pb-3 bg-bg-secondary border-b border-border-subtle flex-shrink-0"
        >
          <button
            class="relative p-2 flex items-center justify-center hover:bg-hover-bg transition-colors text-text-tertiary hover:text-text-primary"
            title="新建 Bot"
            @click="showCreateModal = true"
          >
            <BsPlusLg :size="20" />
          </button>
          <button
            class="relative p-2 flex items-center justify-center hover:bg-hover-bg transition-colors text-text-tertiary hover:text-text-primary"
            title="开发者 API"
            @click="router.push('/bot-studio/developer/api')"
          >
            API
          </button>
          <button
            class="relative p-2 flex items-center justify-center hover:bg-hover-bg transition-colors text-text-tertiary hover:text-text-primary"
            title="搜索公开 Bot"
            @click="showSearch = !showSearch"
          >
            <BsSearch :size="18" />
          </button>
        </div>

        <!-- 搜索栏 -->
        <div v-if="showSearch" class="px-3 py-2 border-b border-border-subtle flex-shrink-0">
          <input
            v-model="searchQuery"
            type="text"
            placeholder="搜索公开 Bot..."
            class="w-full px-3 py-2 text-sm rounded-[var(--radius-sm,8px)] bg-bg-quaternary text-text-primary placeholder:text-text-quaternary outline-none focus:ring-1 focus:ring-[var(--theme-primary)] transition-all"
            @input="handleSearch"
          />
        </div>

        <!-- Bot 列表 -->
        <div class="flex-1 min-h-0">
          <BotList
            :bots="displayBots"
            :active-bot-id="botStore.activeBotId"
            :loading="botStore.loading || botStore.searchLoading"
            :is-search="!!searchQuery"
            :has-more="botStore.searchHasMore"
            :current-user-id="currentUserId"
            @select="handleSelectBot"
            @delete="handleDeleteBot"
            @create-conversation="handleCreateConversation"
            @deploy="handleDeploy"
            @load-more="handleLoadMore"
          />
        </div>
      </div>
    </template>

    <!-- 编辑器 -->
    <BotEditor
      v-if="botStore.activeBot"
      :key="botStore.activeBotId ?? undefined"
      :bot="botStore.activeBot"
      :is-owned="botStore.activeBot.owner_id === currentUserId"
      @update="handleUpdateBot"
      @back="botStore.setActiveBot(null)"
      @create-conversation="handleCreateConversation"
      @deploy="handleDeploy"
    />

    <!-- 空状态 -->
    <div v-else class="flex-1 flex items-center justify-center">
      <div class="text-center space-y-3">
        <BsCpu :size="48" class="mx-auto text-text-quaternary" />
        <h3 class="text-lg font-medium text-text-secondary">Bot Studio</h3>
        <p class="text-sm text-text-tertiary max-w-[280px]">
          创建和管理你的 Bot，配置触发规则和回复方式，部署到对话中。
        </p>
        <button
          class="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-[var(--radius-sm,8px)] text-white transition-colors"
          style="background: var(--theme-primary)"
          @click="showCreateModal = true"
        >
          <BsPlusLg :size="16" />
          创建 Bot
        </button>
      </div>
    </div>
  </BasePanel>

  <!-- 创建 Bot 弹窗 -->
  <CreateBotModal
    v-if="showCreateModal"
    @create="handleCreateBot"
    @close="showCreateModal = false"
  />

  <!-- 安装到群聊弹窗 -->
  <DeployToGroupModal
    v-if="showDeployModal && deployBotTarget"
    :bot="deployBotTarget"
    @close="showDeployModal = false"
  />

  <InstallBotModal
    v-if="showDirectInstallModal && directInstallTarget"
    :bot="directInstallTarget"
    :installation="directInstallationTarget"
    @installed="handleDirectInstalled"
    @close="closeDirectInstall"
  />
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { BsPlusLg, BsSearch, BsCpu } from 'vue-icons-plus/bs';
import BasePanel from './BasePanel.vue';
import BotList from './bots/BotList.vue';
import BotEditor from './bots/BotEditor.vue';
import CreateBotModal from './bots/CreateBotModal.vue';
import DeployToGroupModal from './bots/DeployToGroupModal.vue';
import InstallBotModal from './bots/InstallBotModal.vue';
import { useBotStore } from '../../../stores/bot';
import { useBots } from '../../../composables/useBots';
import { useAuthController } from '../../../controllers/authController';
import { useRouter } from 'vue-router';
import type { Bot, BotDeployment } from '../../../models/types';
import { api } from '../../../models/api';

const botStore = useBotStore();
const { createBot, deleteBot, updateBot, createBotConversation } = useBots();
const auth = useAuthController();
const router = useRouter();

const showCreateModal = ref(false);
const showSearch = ref(false);
const searchQuery = ref('');
const showDeployModal = ref(false);
const deployBotTarget = ref<Bot | null>(null);
const showDirectInstallModal = ref(false);
const directInstallTarget = ref<Bot | null>(null);
const directInstallationTarget = ref<BotDeployment | null>(null);

const allKnownBots = computed(() => {
  const known = [...botStore.bots, ...botStore.searchResults];
  for (const deployment of botStore.deployments) {
    if (deployment.app) known.push(deployment.app);
  }
  return known;
});

const currentUserId = computed(() => auth.currentUser?.id ?? '');

// 合并：我的 Bot + 已安装的公开 Bot
const displayBots = computed<Bot[]>(() => {
  if (searchQuery.value) return botStore.searchResults;
  const myBots = botStore.bots;
  const seen = new Set(myBots.map((b) => b.id));
  const installed: Bot[] = [];
  for (const dep of botStore.deployments) {
    if (dep.app && !seen.has(dep.app.id) && dep.app.owner_id !== currentUserId.value) {
      installed.push(dep.app);
      seen.add(dep.app.id);
    }
  }
  return [...myBots, ...installed];
});

onMounted(async () => {
  await Promise.all([botStore.loadBots(), botStore.loadDeployments()]);
});

async function handleCreateBot(data: { name: string; description?: string; bot_type?: string }) {
  const bot = await createBot({
    name: data.name,
    description: data.description || '',
    bot_type: data.bot_type as any,
  });
  if (bot) {
    showCreateModal.value = false;
    botStore.setActiveBot(bot.id);
  }
}

async function handleDeleteBot(botId: string) {
  await deleteBot(botId);
}

async function handleUpdateBot(botId: string, data: any) {
  await updateBot(botId, data);
}

function handleSelectBot(botId: string) {
  botStore.setActiveBot(botId);
}

async function handleDeploy(botId: string) {
  const bot = await resolveBotForInstall(botId);
  if (!bot) return;
  deployBotTarget.value = bot;
  showDeployModal.value = true;
}

async function handleCreateConversation(botId: string) {
  const existingInstallation = botStore.deployments.find(
    (deployment) =>
      deployment.app_id === botId &&
      deployment.target_type === 'user' &&
      deployment.target_id === currentUserId.value
  );
  const bot = await resolveBotForInstall(botId);
  const requestedCapabilities = bot?.requested_capabilities ?? [];
  const needsAuthorization =
    !existingInstallation ||
    existingInstallation.status !== 'active' ||
    requestedCapabilities.some(
      (capability) => !existingInstallation.granted_capabilities.includes(capability)
    );
  if (needsAuthorization) {
    if (bot) {
      directInstallTarget.value = bot;
      directInstallationTarget.value = existingInstallation ?? null;
      showDirectInstallModal.value = true;
      return;
    }
  }
  await openBotConversation(botId);
}

async function resolveBotForInstall(botId: string): Promise<Bot | null> {
  try {
    const response = await api.getBot(botId);
    if (response.success && response.data) return response.data;
  } catch (error) {
    console.error('[BotStudioPanel] 获取 Bot 授权信息失败:', error);
  }
  return allKnownBots.value.find((item) => item.id === botId) ?? null;
}

async function openBotConversation(botId: string) {
  const conversation = await createBotConversation(botId);
  if (conversation) {
    router.push({ path: '/chat', query: { conversationId: conversation.id } });
  }
}

async function handleDirectInstalled() {
  const botId = directInstallTarget.value?.id;
  closeDirectInstall();
  if (botId) await openBotConversation(botId);
}

function closeDirectInstall() {
  showDirectInstallModal.value = false;
  directInstallTarget.value = null;
  directInstallationTarget.value = null;
}

async function handleSearch() {
  await botStore.searchPublicBots(searchQuery.value);
}

async function handleLoadMore() {
  await botStore.loadMoreSearchResults(searchQuery.value);
}
</script>
